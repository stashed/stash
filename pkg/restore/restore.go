/*
Copyright AppsCode Inc. and Contributors

Licensed under the AppsCode Community License 1.0.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    https://github.com/appscode/licenses/raw/1.0.0/AppsCode-Community-1.0.0.md

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package restore

import (
	"context"
	"fmt"
	"os"
	"time"

	"stash.appscode.dev/apimachinery/apis"
	api_v1beta1 "stash.appscode.dev/apimachinery/apis/stash/v1beta1"
	cs "stash.appscode.dev/apimachinery/client/clientset/versioned"
	stashHooks "stash.appscode.dev/apimachinery/pkg/hooks"
	"stash.appscode.dev/apimachinery/pkg/invoker"
	"stash.appscode.dev/apimachinery/pkg/metrics"
	"stash.appscode.dev/apimachinery/pkg/restic"
	"stash.appscode.dev/stash/pkg/eventer"
	"stash.appscode.dev/stash/pkg/status"
	"stash.appscode.dev/stash/pkg/util"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/errors"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/leaderelection"
	"k8s.io/client-go/tools/leaderelection/resourcelock"
	"k8s.io/klog/v2"
	kmapi "kmodules.xyz/client-go/api/v1"
	v1 "kmodules.xyz/offshoot-api/api/v1"
)

const (
	RestoreModelInitContainer = "init-container"
	RestoreModelJob           = "job"
)

type Options struct {
	Config         *rest.Config
	MasterURL      string
	KubeconfigPath string
	Namespace      string

	InvokerKind string
	InvokerName string
	TargetKind  string
	TargetName  string

	BackoffMaxWait time.Duration

	SetupOpt restic.SetupOptions
	Metrics  metrics.MetricsOptions
	Host     string

	KubeClient   kubernetes.Interface
	StashClient  cs.Interface
	RestoreModel string
}

func (opt *Options) Restore(inv invoker.RestoreInvoker, targetInfo invoker.RestoreTargetInfo) error {
	// if we are restoring using job then there no need to lock the repository
	if opt.RestoreModel == RestoreModelJob {
		return opt.restoreHost(inv, targetInfo)
	} else {
		// only one pod can acquire restic repository lock. so we need leader election to determine who will acquire the lock
		return opt.electRestoreLeader(inv, targetInfo)
	}
}

func (opt *Options) electRestoreLeader(inv invoker.RestoreInvoker, targetInfo invoker.RestoreTargetInfo) error {
	klog.Infoln("Attempting to elect restore leader")

	rlc := resourcelock.ResourceLockConfig{
		Identity:      os.Getenv(apis.KeyPodName),
		EventRecorder: eventer.NewEventRecorder(opt.KubeClient, eventer.EventSourceRestoreInitContainer),
	}

	resLock, err := resourcelock.New(
		resourcelock.ConfigMapsResourceLock,
		inv.GetObjectMeta().Namespace,
		util.GetRestoreConfigmapLockName(targetInfo.Target.Ref),
		opt.KubeClient.CoreV1(),
		opt.KubeClient.CoordinationV1(),
		rlc,
	)
	if err != nil {
		return fmt.Errorf("error during leader election: %s", err)
	}

	// use a Go context so we can tell the leader election code when we
	// want to step down
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// start the leader election code loop
	leaderelection.RunOrDie(ctx, leaderelection.LeaderElectionConfig{
		Lock:          resLock,
		LeaseDuration: 15 * time.Second,
		RenewDeadline: 10 * time.Second,
		RetryPeriod:   2 * time.Second,
		Callbacks: leaderelection.LeaderCallbacks{
			OnStartedLeading: func(ctx context.Context) {
				klog.Infoln("Got leadership, preparing for restore")
				// run restore process
				err := opt.restoreHost(inv, targetInfo)
				if err != nil {
					// step down from leadership so that other replicas try to restore
					cancel()
					// fail the container so that it restart and re-try to restore
					klog.Fatalf("failed to complete restore. Reason: %v", err)
				}
				// restore process is complete. now, step down from leadership so that other replicas can start
				cancel()
			},
			OnStoppedLeading: func() {
				klog.Infoln("Lost leadership")
			},
		},
	})
	return nil
}

func (opt *Options) restoreHost(inv invoker.RestoreInvoker, targetInfo invoker.RestoreTargetInfo) error {
	// execute at the end of restore. no matter if the restore succeed or fail.
	defer func() {
		if targetInfo.Hooks != nil && targetInfo.Hooks.PostRestore != nil {
			err := opt.executePostRestoreHook(inv, targetInfo)
			if err != nil {
				klog.Infoln("failed to execute postRestore hook. Reason: ", err)
			}
		}
	}()

	if targetInfo.Hooks != nil && targetInfo.Hooks.PreRestore != nil {
		err := opt.executePreRestoreHook(inv, targetInfo)
		if err != nil {
			return err
		}
	}

	restoreOutput, restoreErr := opt.runRestore(inv, targetInfo)
	if restoreErr != nil {
		restoreOutput = &restic.RestoreOutput{
			RestoreTargetStatus: api_v1beta1.RestoreMemberStatus{
				Ref: targetInfo.Target.Ref,
				Stats: []api_v1beta1.HostRestoreStats{
					{
						Hostname: opt.Host,
						Phase:    api_v1beta1.HostRestoreFailed,
						Error:    fmt.Sprintf("failed to complete restore process for host %s. Reason: %v", opt.Host, restoreErr),
					},
				},
			},
		}
	}
	statusErr := opt.updateHostRestoreStatus(restoreOutput, inv, targetInfo)
	if statusErr != nil {
		restoreErr = errors.NewAggregate([]error{restoreErr, statusErr})
	}

	return restoreErr
}

func (opt *Options) runRestore(inv invoker.RestoreInvoker, targetInfo invoker.RestoreTargetInfo) (*restic.RestoreOutput, error) {
	if targetInfo.Target == nil {
		return nil, fmt.Errorf("no restore target has specified")
	}

	repository, err := inv.GetRepository()
	if err != nil {
		return nil, err
	}

	secret, err := opt.KubeClient.CoreV1().Secrets(repository.Namespace).Get(context.TODO(), repository.Spec.Backend.StorageSecretName, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}

	extraOptions := &util.ExtraOptions{
		Host:          opt.Host,
		StorageSecret: secret,
		EnableCache:   opt.SetupOpt.EnableCache,
		ScratchDir:    opt.SetupOpt.ScratchDir,
	}
	setupOptions, err := util.SetupOptionsForRepository(*repository, *extraOptions)
	if err != nil {
		return nil, err
	}
	// apply nice, ionice settings from env
	setupOptions.Nice, err = v1.NiceSettingsFromEnv()
	if err != nil {
		return nil, err
	}
	setupOptions.IONice, err = v1.IONiceSettingsFromEnv()
	if err != nil {
		return nil, err
	}
	opt.SetupOpt = setupOptions

	// if already restored for this host then don't process further
	if opt.isRestoredForThisHost(inv, targetInfo, opt.Host) {
		klog.Infof("Skipping restore for %s %s/%s. Reason: restore already completed for host %q.",
			inv.GetTypeMeta().Kind,
			inv.GetObjectMeta().Namespace,
			inv.GetObjectMeta().Name,
			opt.Host,
		)
		return nil, nil
	}

	// setup restic wrapper
	w, err := restic.NewResticWrapper(opt.SetupOpt)
	if err != nil {
		return nil, err
	}
	restoreOptions := util.RestoreOptionsForHost(opt.Host, targetInfo.Target.Rules)
	restoreOptions.Args = targetInfo.Target.Args
	return w.RunRestore(restoreOptions, targetInfo.Target.Ref)
}

func (opt *Options) updateHostRestoreStatus(restoreOutput *restic.RestoreOutput, inv invoker.RestoreInvoker, targetInfo invoker.RestoreTargetInfo) error {
	statusOpt := status.UpdateStatusOptions{
		Config:      opt.Config,
		KubeClient:  opt.KubeClient,
		StashClient: opt.StashClient,
		Namespace:   opt.Namespace,
		Metrics:     opt.Metrics,
		SetupOpt:    opt.SetupOpt,
	}
	return statusOpt.UpdatePostRestoreStatus(restoreOutput, inv, targetInfo)
}

func (opt *Options) isRestoredForThisHost(inv invoker.RestoreInvoker, targetInfo invoker.RestoreTargetInfo, host string) bool {
	// if overall invoker Phase is "Succeeded" then restore has been complete for this host
	if inv.GetStatus().Phase == api_v1beta1.RestoreSucceeded {
		return true
	}
	for _, member := range inv.GetStatus().TargetStatus {
		if invoker.TargetMatched(member.Ref, targetInfo.Target.Ref) {
			// if restore invoker has entry for this host in status field and it is succeeded, then restore has been completed for this host
			for _, hostStat := range member.Stats {
				if hostStat.Hostname == host && hostStat.Phase == api_v1beta1.HostRestoreSucceeded {
					return true
				}
			}
		}
	}
	return false
}

func (opt *Options) executePreRestoreHook(inv invoker.RestoreInvoker, targetInfo invoker.RestoreTargetInfo) error {
	hookExecutor := stashHooks.RestoreHookExecutor{
		Config:  opt.Config,
		Invoker: inv,
		Target:  targetInfo.Target.Ref,
		ExecutorPod: kmapi.ObjectReference{
			Namespace: opt.Namespace,
			Name:      os.Getenv(apis.KeyPodName),
		},
		Hook:     targetInfo.Hooks.PreRestore,
		HookType: apis.PreRestoreHook,
	}
	return hookExecutor.Execute()
}

func (opt *Options) executePostRestoreHook(inv invoker.RestoreInvoker, targetInfo invoker.RestoreTargetInfo) error {
	hookExecutor := stashHooks.RestoreHookExecutor{
		Config:  opt.Config,
		Invoker: inv,
		Target:  targetInfo.Target.Ref,
		ExecutorPod: kmapi.ObjectReference{
			Namespace: opt.Namespace,
			Name:      os.Getenv(apis.KeyPodName),
		},
		Hook:     targetInfo.Hooks.PostRestore,
		HookType: apis.PostRestoreHook,
	}
	return hookExecutor.Execute()
}

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
	"stash.appscode.dev/apimachinery/pkg/restic"
	"stash.appscode.dev/stash/pkg/eventer"
	"stash.appscode.dev/stash/pkg/status"
	"stash.appscode.dev/stash/pkg/util"

	"github.com/appscode/go/log"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/errors"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/leaderelection"
	"k8s.io/client-go/tools/leaderelection/resourcelock"
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
	Metrics  restic.MetricsOptions
	Host     string

	KubeClient   kubernetes.Interface
	StashClient  cs.Interface
	RestoreModel string
}

func (opt *Options) Restore(invoker apis.RestoreInvoker, targetInfo apis.RestoreTargetInfo) (*restic.RestoreOutput, error) {

	if targetInfo.Target == nil {
		return nil, fmt.Errorf("no restore target has specified")
	}

	repository, err := opt.StashClient.StashV1alpha1().Repositories(opt.Namespace).Get(context.TODO(), invoker.Repository, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}

	extraOptions := util.ExtraOptions{
		Host:        opt.Host,
		SecretDir:   opt.SetupOpt.SecretDir,
		EnableCache: opt.SetupOpt.EnableCache,
		ScratchDir:  opt.SetupOpt.ScratchDir,
	}
	setupOptions, err := util.SetupOptionsForRepository(*repository, extraOptions)
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

	// if we are restoring using job then there no need to lock the repository
	if opt.RestoreModel == RestoreModelJob {
		return opt.runRestore(invoker, targetInfo)
	} else {
		// only one pod can acquire restic repository lock. so we need leader election to determine who will acquire the lock
		return nil, opt.electRestoreLeader(invoker, targetInfo)
	}
}

func (opt *Options) electRestoreLeader(invoker apis.RestoreInvoker, targetInfo apis.RestoreTargetInfo) error {

	log.Infoln("Attempting to elect restore leader")

	rlc := resourcelock.ResourceLockConfig{
		Identity:      os.Getenv(apis.KeyPodName),
		EventRecorder: eventer.NewEventRecorder(opt.KubeClient, eventer.EventSourceRestoreInitContainer),
	}

	resLock, err := resourcelock.New(
		resourcelock.ConfigMapsResourceLock,
		invoker.ObjectMeta.Namespace,
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
				log.Infoln("Got leadership, preparing for restore")
				// run restore process
				restoreOutput, restoreErr := opt.runRestore(invoker, targetInfo)
				if restoreErr != nil {
					e2 := opt.HandleRestoreFailure(invoker, targetInfo, restoreErr)
					if e2 != nil {
						restoreErr = errors.NewAggregate([]error{restoreErr, e2})
					}
					// step down from leadership so that other replicas try to restore
					cancel()
					// fail the container so that it restart and re-try to restore
					log.Fatalf("failed to complete restore. Reason: %v", restoreErr)
				}
				if restoreOutput != nil {
					err = opt.HandleRestoreSuccess(restoreOutput, invoker, targetInfo)
					if err != nil {
						cancel()
						log.Fatalf("failed to complete restore. Reason: %v", err)
					}
				}
				// restore process is complete. now, step down from leadership so that other replicas can start
				cancel()
			},
			OnStoppedLeading: func() {
				log.Infoln("Lost leadership")
			},
		},
	})
	return nil
}

func (opt *Options) runRestore(invoker apis.RestoreInvoker, targetInfo apis.RestoreTargetInfo) (*restic.RestoreOutput, error) {

	// if already restored for this host then don't process further
	if opt.isRestoredForThisHost(invoker, targetInfo, opt.Host) {
		log.Infof("Skipping restore for %s %s/%s. Reason: restore already completed for host %q.",
			invoker.TypeMeta.Kind,
			invoker.ObjectMeta.Namespace,
			invoker.ObjectMeta.Name,
			opt.Host,
		)
		return nil, nil
	}

	// If preRestore hook is specified, then execute those hooks first
	if targetInfo.Hooks != nil && targetInfo.Hooks.PreRestore != nil {
		err := util.ExecuteHook(opt.Config, targetInfo.Hooks, apis.PreRestoreHook, os.Getenv(apis.KeyPodName), opt.Namespace)
		if err != nil {
			return nil, err
		}
	}

	// setup restic wrapper
	w, err := restic.NewResticWrapper(opt.SetupOpt)
	if err != nil {
		return nil, err
	}

	// Run restore process
	// If there is an error during restore, don't return.
	// We will execute postRestore hook even if the restore failed.
	// Reason: https://github.com/stashed/stash/issues/986
	var restoreErr, hookErr error
	output, restoreErr := w.RunRestore(util.RestoreOptionsForHost(opt.Host, targetInfo.Target.Rules), targetInfo.Target.Ref)

	// If postRestore hook is specified, then execute those hooks
	if targetInfo.Hooks != nil && targetInfo.Hooks.PostRestore != nil {
		hookErr = util.ExecuteHook(opt.Config, targetInfo.Hooks, apis.PostRestoreHook, os.Getenv(apis.KeyPodName), opt.Namespace)
	}

	if restoreErr != nil || hookErr != nil {
		return nil, errors.NewAggregate([]error{restoreErr, hookErr})
	}
	return output, nil
}

func (c *Options) HandleRestoreSuccess(restoreOutput *restic.RestoreOutput, invoker apis.RestoreInvoker, targetInfo apis.RestoreTargetInfo) error {
	// write log
	log.Infof("Restore completed successfully for %s %s/%s",
		invoker.TypeMeta.Kind,
		invoker.ObjectMeta.Namespace,
		invoker.ObjectMeta.Name,
	)

	statusOpt := status.UpdateStatusOptions{
		Config:      c.Config,
		KubeClient:  c.KubeClient,
		StashClient: c.StashClient,
		Namespace:   c.Namespace,
		Repository:  invoker.Repository,
		Metrics:     c.Metrics,
	}
	return statusOpt.UpdatePostRestoreStatus(restoreOutput, invoker, targetInfo)
}

func (c *Options) HandleRestoreFailure(invoker apis.RestoreInvoker, targetInfo apis.RestoreTargetInfo, restoreErr error) error {
	// write log
	log.Errorf("Failed to complete restore process for %s %s/%s. Reason: %v",
		invoker.TypeMeta.Kind,
		invoker.ObjectMeta.Namespace,
		invoker.ObjectMeta.Name,
		restoreErr,
	)

	restoreOutput := &restic.RestoreOutput{
		RestoreTargetStatus: api_v1beta1.RestoreMemberStatus{
			Ref: targetInfo.Target.Ref,
			Stats: []api_v1beta1.HostRestoreStats{
				{
					Hostname: c.Host,
					Phase:    api_v1beta1.HostRestoreFailed,
					Error:    fmt.Sprintf("failed to complete restore process for host %s. Reason: %v", c.Host, restoreErr),
				},
			},
		},
	}

	statusOpt := status.UpdateStatusOptions{
		Config:      c.Config,
		KubeClient:  c.KubeClient,
		StashClient: c.StashClient,
		Namespace:   c.Namespace,
		Repository:  invoker.Repository,
		Metrics:     c.Metrics,
	}
	return statusOpt.UpdatePostRestoreStatus(restoreOutput, invoker, targetInfo)
}

func (opt *Options) isRestoredForThisHost(invoker apis.RestoreInvoker, targetInfo apis.RestoreTargetInfo, host string) bool {

	// if overall invoker Phase is "Succeeded" then restore has been complete for this host
	if invoker.Status.Phase == api_v1beta1.RestoreSucceeded {
		return true
	}
	for _, member := range invoker.Status.TargetStatus {
		if apis.TargetMatched(member.Ref, targetInfo.Target.Ref) {
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

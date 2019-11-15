/*
Copyright The Stash Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

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

	api_v1beta1 "stash.appscode.dev/stash/apis/stash/v1beta1"
	cs "stash.appscode.dev/stash/client/clientset/versioned"
	"stash.appscode.dev/stash/pkg/eventer"
	"stash.appscode.dev/stash/pkg/restic"
	"stash.appscode.dev/stash/pkg/status"
	"stash.appscode.dev/stash/pkg/util"

	"github.com/appscode/go/log"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/errors"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/leaderelection"
	"k8s.io/client-go/tools/leaderelection/resourcelock"
)

const (
	RestoreModelInitContainer = "init-container"
	RestoreModelJob           = "job"
)

type Options struct {
	Config             *rest.Config
	MasterURL          string
	KubeconfigPath     string
	Namespace          string
	RestoreSessionName string
	BackoffMaxWait     time.Duration

	SetupOpt restic.SetupOptions
	Metrics  restic.MetricsOptions
	Host     string

	KubeClient   kubernetes.Interface
	StashClient  cs.Interface
	RestoreModel string
}

func Restore(opt *Options) (*restic.RestoreOutput, error) {

	// get the RestoreSession crd
	restoreSession, err := opt.StashClient.StashV1beta1().RestoreSessions(opt.Namespace).Get(opt.RestoreSessionName, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}
	if restoreSession.Spec.Target == nil {
		return nil, fmt.Errorf("invalid RestoreSession. Target is nil")
	}

	repository, err := opt.StashClient.StashV1alpha1().Repositories(opt.Namespace).Get(restoreSession.Spec.Repository.Name, metav1.GetOptions{})
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
	setupOptions.Nice, err = util.NiceSettingsFromEnv()
	if err != nil {
		return nil, err
	}
	setupOptions.IONice, err = util.IONiceSettingsFromEnv()
	if err != nil {
		return nil, err
	}
	opt.SetupOpt = setupOptions

	// if we are restoring using job then there no need to lock the repository
	if opt.RestoreModel == RestoreModelJob {
		return opt.runRestore(restoreSession)
	} else {
		// only one pod can acquire restic repository lock. so we need leader election to determine who will acquire the lock
		return nil, opt.electRestoreLeader(restoreSession)
	}
}

func (opt *Options) electRestoreLeader(restoreSession *api_v1beta1.RestoreSession) error {

	log.Infoln("Attempting to elect restore leader")

	rlc := resourcelock.ResourceLockConfig{
		Identity:      os.Getenv(util.KeyPodName),
		EventRecorder: eventer.NewEventRecorder(opt.KubeClient, eventer.EventSourceRestoreInitContainer),
	}

	resLock, err := resourcelock.New(
		resourcelock.ConfigMapsResourceLock,
		restoreSession.Namespace,
		util.GetRestoreConfigmapLockName(restoreSession.Spec.Target.Ref),
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
				restoreOutput, restoreErr := opt.runRestore(restoreSession)
				if restoreErr != nil {
					e2 := opt.HandleRestoreFailure(restoreErr)
					if e2 != nil {
						restoreErr = errors.NewAggregate([]error{restoreErr, e2})
					}
					// step down from leadership so that other replicas try to restore
					cancel()
					// fail the container so that it restart and re-try to restore
					log.Fatalf("failed to complete restore. Reason: %v", restoreErr)
				}
				if restoreOutput != nil {
					err = opt.HandleRestoreSuccess(restoreOutput)
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

func (opt *Options) runRestore(restoreSession *api_v1beta1.RestoreSession) (*restic.RestoreOutput, error) {

	// if already restored for this host then don't process further
	if opt.isRestoredForThisHost(restoreSession, opt.Host) {
		log.Infof("Skipping restore for RestoreSession %s/%s. Reason: RestoreSession already processed for host %q.", restoreSession.Namespace, restoreSession.Name, opt.Host)
		return nil, nil
	}

	// setup restic wrapper
	w, err := restic.NewResticWrapper(opt.SetupOpt)
	if err != nil {
		return nil, err
	}

	// run restore process
	return w.RunRestore(util.RestoreOptionsForHost(opt.Host, restoreSession.Spec.Rules))
}

func (c *Options) HandleRestoreSuccess(restoreOutput *restic.RestoreOutput) error {
	// write log
	log.Infof("Restore completed successfully for RestoreSession %s", c.RestoreSessionName)

	// add/update entry into RestoreSession status for this host
	restoreSession, err := c.StashClient.StashV1beta1().RestoreSessions(c.Namespace).Get(c.RestoreSessionName, metav1.GetOptions{})
	if err != nil {
		return err
	}

	statusOpt := status.UpdateStatusOptions{
		Config:         c.Config,
		KubeClient:     c.KubeClient,
		StashClient:    c.StashClient,
		Namespace:      c.Namespace,
		Repository:     restoreSession.Spec.Repository.Name,
		RestoreSession: c.RestoreSessionName,
		Metrics:        c.Metrics,
	}
	return statusOpt.UpdatePostRestoreStatus(restoreOutput)
}

func (c *Options) HandleRestoreFailure(restoreErr error) error {
	// write log
	log.Warningf("Failed to complete restore process for RestoreSession %s. Reason: %v", c.RestoreSessionName, restoreErr)

	// add/update entry into RestoreSession status for this host
	restoreSession, err := c.StashClient.StashV1beta1().RestoreSessions(c.Namespace).Get(c.RestoreSessionName, metav1.GetOptions{})
	if err != nil {
		return err
	}

	restoreOutput := &restic.RestoreOutput{
		HostRestoreStats: []api_v1beta1.HostRestoreStats{
			{
				Hostname: c.Host,
				Phase:    api_v1beta1.HostRestoreFailed,
				Error:    fmt.Sprintf("failed to complete restore process for host %s. Reason: %v", c.Host, restoreErr),
			},
		},
	}

	statusOpt := status.UpdateStatusOptions{
		Config:         c.Config,
		KubeClient:     c.KubeClient,
		StashClient:    c.StashClient,
		Namespace:      c.Namespace,
		Repository:     restoreSession.Spec.Repository.Name,
		RestoreSession: c.RestoreSessionName,
		Metrics:        c.Metrics,
	}
	return statusOpt.UpdatePostRestoreStatus(restoreOutput)
}

func (opt *Options) isRestoredForThisHost(restoreSession *api_v1beta1.RestoreSession, host string) bool {

	// if overall restoreSession phase is "Succeeded" then restore has been complete for this host
	if restoreSession.Status.Phase == api_v1beta1.RestoreSessionSucceeded {
		return true
	}

	// if restoreSession has entry for this host in status field and it is succeeded, then restore has been completed for this host
	for _, hostStats := range restoreSession.Status.Stats {
		if hostStats.Hostname == host && hostStats.Phase == api_v1beta1.HostRestoreSucceeded {
			return true
		}
	}

	return false
}

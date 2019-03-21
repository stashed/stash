package restore

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/appscode/go/log"
	"github.com/appscode/stash/apis"
	api_v1beta1 "github.com/appscode/stash/apis/stash/v1beta1"
	cs "github.com/appscode/stash/client/clientset/versioned"
	stash_scheme "github.com/appscode/stash/client/clientset/versioned/scheme"
	stash_util_v1beta1 "github.com/appscode/stash/client/clientset/versioned/typed/stash/v1beta1/util"
	"github.com/appscode/stash/pkg/eventer"
	"github.com/appscode/stash/pkg/restic"
	"github.com/appscode/stash/pkg/util"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/leaderelection"
	"k8s.io/client-go/tools/leaderelection/resourcelock"
	"k8s.io/client-go/tools/reference"
	"k8s.io/kubernetes/pkg/apis/core"
)

const (
	RestoreSessionEventComponent = "stash-restore-session"
)

type Options struct {
	MasterURL          string
	KubeconfigPath     string
	Namespace          string
	RestoreSessionName string
	BackoffMaxWait     time.Duration

	SetupOpt restic.SetupOptions
	Metrics  restic.MetricsOptions

	KubeClient  kubernetes.Interface
	StashClient cs.Interface
}

func Restore(opt *Options) error {

	// get the RestoreSession crd
	restoreSession, err := opt.StashClient.StashV1beta1().RestoreSessions(opt.Namespace).Get(opt.RestoreSessionName, metav1.GetOptions{})
	if err != nil {
		return err
	}
	if restoreSession.Spec.Target == nil {
		return fmt.Errorf("invalid RestoreSession. Target is nil")
	}
	repository, err := opt.StashClient.StashV1alpha1().Repositories(opt.Namespace).Get(restoreSession.Spec.Repository.Name, metav1.GetOptions{})
	if err != nil {
		return err
	}
	host, err := util.GetHostName(restoreSession.Spec.Target)
	if err != nil {
		return err
	}

	extraOptions := util.ExtraOptions{
		Host:        host,
		SecretDir:   opt.SetupOpt.SecretDir,
		EnableCache: opt.SetupOpt.EnableCache,
		ScratchDir:  opt.SetupOpt.ScratchDir,
	}
	setupOptions, err := util.SetupOptionsForRepository(*repository, extraOptions)
	if err != nil {
		return err
	}
	opt.SetupOpt = setupOptions

	if restoreSession.Spec.Target == nil {
		return fmt.Errorf("restore target is nil")
	}

	// if target is a Deployment, ReplicaSet or ReplicationController then there might be multiple replica
	// using same volume. so only one replica should run restore process while others should not.
	// we can achieve that using leader election. only leader pod will restore. others will wait until restore complete.
	switch restoreSession.Spec.Target.Ref.Kind {
	case apis.KindDeployment, apis.KindReplicaSet, apis.KindReplicationController:
		return opt.electRestoreLeader(restoreSession)
	default:
		if err := opt.runRestore(restoreSession); err != nil {
			return err
		}
	}

	return nil
}

func (opt *Options) electRestoreLeader(restoreSession *api_v1beta1.RestoreSession) error {

	rlc := resourcelock.ResourceLockConfig{
		Identity:      os.Getenv(util.KeyPodName),
		EventRecorder: eventer.NewEventRecorder(opt.KubeClient, RestoreSessionEventComponent),
	}

	resLock, err := resourcelock.New(
		resourcelock.ConfigMapsResourceLock,
		restoreSession.Namespace,
		util.GetRestoreConfigmapLockName(restoreSession.Spec.Target.Ref),
		opt.KubeClient.CoreV1(),
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
				err := opt.runRestore(restoreSession)
				if err != nil {
					log.Errorln("failed to complete restore. Reason: ", err.Error())
					err = HandleRestoreFailure(opt, err)
					if err != nil {
						log.Errorln(err)
					}
				} else {
					// restore process is complete. now, step down from leadership
					cancel()
				}
			},
			OnStoppedLeading: func() {
				log.Infoln("Lost leadership")
			},
		},
	})
	return nil
}

func (opt *Options) runRestore(restoreSession *api_v1beta1.RestoreSession) error {
	// if RestoreSession completed don't proceed further
	if restoreSession.Status.Phase == api_v1beta1.RestoreSucceeded {
		log.Infoln("Skipping restore. Reason: Successfully restored on my previous session or by another replica.")
		return nil
	}

	// set RestoreSession phase "Running"
	_, err := stash_util_v1beta1.UpdateRestoreSessionStatus(opt.StashClient.StashV1beta1(), restoreSession, func(status *api_v1beta1.RestoreSessionStatus) *api_v1beta1.RestoreSessionStatus {
		status.Phase = api_v1beta1.RestoreRunning
		return status
	}, apis.EnableStatusSubresource)
	if err != nil {
		return err
	}

	// setup restic wrapper
	w, err := restic.NewResticWrapper(opt.SetupOpt)
	if err != nil {
		return err
	}

	hostname, err := util.GetHostName(restoreSession.Spec.Target)
	if err != nil {
		return err
	}

	// run restore process
	restoreOutput, err := w.RunRestore(util.RestoreOptionsForHost(hostname, restoreSession.Spec.Rules))
	if err != nil {
		return err
	}

	// if metrics are enabled then send metrics
	if opt.Metrics.Enabled {
		err := restoreOutput.HandleMetrics(&opt.Metrics, nil)
		if err != nil {
			return err
		}
	}

	// restore is complete. update RestoreSession status
	_, err = stash_util_v1beta1.UpdateRestoreSessionStatus(opt.StashClient.StashV1beta1(), restoreSession, func(status *api_v1beta1.RestoreSessionStatus) *api_v1beta1.RestoreSessionStatus {
		status.Phase = api_v1beta1.RestoreSucceeded
		status.Duration = restoreOutput.SessionDuration
		return status
	}, apis.EnableStatusSubresource)
	if err != nil {
		return err
	}

	// write success event
	ref, rerr := reference.GetReference(stash_scheme.Scheme, restoreSession)
	if rerr == nil {
		eventer.CreateEventWithLog(
			opt.KubeClient,
			RestoreSessionEventComponent,
			ref,
			core.EventTypeNormal,
			eventer.EventReasonRestoreSuccess,
			fmt.Sprintf("Successfully restored."),
		)
	}

	return nil
}

func HandleRestoreFailure(opt *Options, restoreErr error) error {
	restoreSession, err := opt.StashClient.StashV1beta1().RestoreSessions(opt.Namespace).Get(opt.RestoreSessionName, metav1.GetOptions{})
	if err != nil {
		return err
	}
	// set RestoreSession phase "Failed"
	_, err = stash_util_v1beta1.UpdateRestoreSessionStatus(opt.StashClient.StashV1beta1(), restoreSession, func(status *api_v1beta1.RestoreSessionStatus) *api_v1beta1.RestoreSessionStatus {
		status.Phase = api_v1beta1.RestoreFailed
		return status
	}, apis.EnableStatusSubresource)

	// write failure event
	opt.writeRestoreFailureEvent(restoreSession, restoreErr)

	// send prometheus metrics
	if opt.Metrics.Enabled {
		restoreOutput := &restic.RestoreOutput{}
		return restoreOutput.HandleMetrics(&opt.Metrics, restoreErr)
	}
	return nil
}

func (opt *Options) writeRestoreFailureEvent(restoreSession *api_v1beta1.RestoreSession, err error) {
	// write failure event
	ref, rerr := reference.GetReference(stash_scheme.Scheme, restoreSession)
	if rerr == nil {
		eventer.CreateEventWithLog(
			opt.KubeClient,
			RestoreSessionEventComponent,
			ref,
			core.EventTypeWarning,
			eventer.EventReasonRestoreFailed,
			fmt.Sprintf("Failed to restore. Reason: %v", err),
		)
	} else {
		log.Errorf("Failed to write failure event. Reason: %v", rerr)
	}
}

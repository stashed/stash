package backup

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/appscode/go/log"
	"github.com/appscode/stash/apis"
	api "github.com/appscode/stash/apis/stash/v1alpha1"
	v1beta1_api "github.com/appscode/stash/apis/stash/v1beta1"
	cs "github.com/appscode/stash/client/clientset/versioned"
	stash_scheme "github.com/appscode/stash/client/clientset/versioned/scheme"
	stash_v1beta1_util "github.com/appscode/stash/client/clientset/versioned/typed/stash/v1beta1/util"
	stashinformers "github.com/appscode/stash/client/informers/externalversions"
	"github.com/appscode/stash/client/listers/stash/v1beta1"
	"github.com/appscode/stash/pkg/controller"
	"github.com/appscode/stash/pkg/eventer"
	"github.com/appscode/stash/pkg/restic"
	"github.com/appscode/stash/pkg/status"
	"github.com/appscode/stash/pkg/util"
	"github.com/golang/glog"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/leaderelection"
	"k8s.io/client-go/tools/leaderelection/resourcelock"
	"k8s.io/client-go/tools/record"
	"k8s.io/client-go/tools/reference"
	"k8s.io/kubernetes/pkg/apis/core"
	"kmodules.xyz/client-go/tools/queue"
)

type Controllers struct {
	K8sClient            kubernetes.Interface
	StashClient          cs.Interface
	MasterURL            string
	KubeconfigPath       string
	StashInformerFactory stashinformers.SharedInformerFactory
	MaxNumRequeues       int
	NumThreads           int
	EnableRBAC           bool // rbac for check job
	ResyncPeriod         time.Duration
	//backupConfiguration
	BackupConfigurationName string
	Namespace               string
	//Restic
	SetupOpt restic.SetupOptions
	Metrics  restic.MetricsOptions
	//Backup Session
	bsQueue    *queue.Worker
	bsInformer cache.SharedIndexInformer
	bsLister   v1beta1.BackupSessionLister

	Status   status.UpdateStatusOptions
	Recorder record.EventRecorder
}

func (c *Controllers) RunBackup() error {
	stopCh := make(chan struct{})
	defer close(stopCh)

	//get BackupConfiguration
	backupConfiguration, err := c.StashClient.StashV1beta1().BackupConfigurations(c.Namespace).Get(c.BackupConfigurationName, metav1.GetOptions{})
	if err != nil {
		return err
	}
	if backupConfiguration.Spec.Target == nil {
		return fmt.Errorf("backupConfiguration target is nil")
	}

	// split code from here for leader election
	switch backupConfiguration.Spec.Target.Ref.Kind {
	case apis.KindDeployment, apis.KindReplicaSet, apis.KindReplicationController:
		if err := c.electBackupLeader(backupConfiguration, stopCh); err != nil {
			return err
		}
	default:
		if err := c.runBackupSessionController(backupConfiguration, stopCh); err != nil {
			return err
		}
	}
	glog.Info("Stopping Stash backup")
	return nil
}

func (c *Controllers) electBackupLeader(backupConfiguration *v1beta1_api.BackupConfiguration, stopCh <-chan struct{}) error {
	rlc := resourcelock.ResourceLockConfig{
		Identity:      os.Getenv(util.KeyPodName),
		EventRecorder: eventer.NewEventRecorder(c.K8sClient, BackupEventComponent),
	}
	resLock, err := resourcelock.New(
		resourcelock.ConfigMapsResourceLock,
		backupConfiguration.Namespace,
		util.GetConfigmapLockName(api.LocalTypedReference{
			Kind:       backupConfiguration.Spec.Target.Ref.Kind,
			Name:       backupConfiguration.Spec.Target.Ref.Name,
			APIVersion: backupConfiguration.Spec.Target.Ref.APIVersion,
		}),
		c.K8sClient.CoreV1(),
		rlc,
	)
	if err != nil {
		return fmt.Errorf("error during leader election: %s", err)
	}
	// use a Go context so we can tell the leader election code when we
	// want to step down
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go func() {
		// start the leader election code loop
		leaderelection.RunOrDie(ctx, leaderelection.LeaderElectionConfig{
			Lock:          resLock,
			LeaseDuration: 15 * time.Second,
			RenewDeadline: 10 * time.Second,
			RetryPeriod:   2 * time.Second,
			Callbacks: leaderelection.LeaderCallbacks{
				OnStartedLeading: func(ctx context.Context) {
					log.Infoln("Got leadership, preparing for backup")
					//run restore process
					err := c.runBackupSessionController(backupConfiguration, stopCh)
					if err != nil {
						log.Fatalln("failed to complete restore. Reason: ", err.Error())
						err = c.HandleBackupFailure(err)
						if err != nil {
							log.Errorln(err)
						}
					}
				},
				OnStoppedLeading: func() {
					log.Infoln("Lost leadership")
				},
			},
		})
	}()
	// wait until stop signal is sent.
	<-stopCh
	return nil
}

func (c *Controllers) initBackupSessionWatcher(backupConfiguration *v1beta1_api.BackupConfiguration) error {
	selector, err := metav1.LabelSelectorAsSelector(&metav1.LabelSelector{
		MatchLabels: map[string]string{
			util.LabelBackupConfiguration: backupConfiguration.Name,
		},
	})
	if err != nil {
		return err
	}
	c.bsInformer = c.StashInformerFactory.Stash().V1beta1().BackupSessions().Informer()
	c.bsQueue = queue.New(v1beta1_api.ResourceKindBackupSession, c.MaxNumRequeues, c.NumThreads, c.processBackupSession)
	c.bsInformer.AddEventHandler(queue.NewFilteredHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			if backupsession, ok := obj.(*v1beta1_api.BackupSession); ok {
				queue.Enqueue(c.bsQueue.GetQueue(), backupsession)
			}
		},
	}, selector))
	c.bsLister = c.StashInformerFactory.Stash().V1beta1().BackupSessions().Lister()
	return nil
}

// syncToStdout is the business logic of the controller. In this controller it simply prints
// information about the deployment to stdout. In case an error happened, it has to simply return the error.
// The retry logic should not be part of the business logic.
func (c *Controllers) processBackupSession(key string) error {
	obj, exists, err := c.bsInformer.GetIndexer().GetByKey(key)
	if err != nil {
		glog.Errorf("Fetching object with key %s from store failed with %v", key, err)
		return err
	}
	if !exists {
		glog.Warningf("Backup Session %s does not exist anymore\n", key)

	} else {
		backupSession := obj.(*v1beta1_api.BackupSession)
		glog.Infof("Sync/Add/Update for Backup Session %s", backupSession.GetName())
		if IsPending(backupSession) {
			err := c.backup(backupSession)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func (c *Controllers) backup(backupSession *v1beta1_api.BackupSession) error {
	// set backupSession phase "Running"
	_, err := stash_v1beta1_util.UpdateBackupSessionStatus(c.StashClient.StashV1beta1(), backupSession, func(status *v1beta1_api.BackupSessionStatus) *v1beta1_api.BackupSessionStatus {
		status.Phase = v1beta1_api.BackupSessionRunning
		return status
	}, apis.EnableStatusSubresource)
	if err != nil {
		return err
	}

	//get BackupConfiguration for BackupSession
	backupConfiguration, err := c.StashClient.StashV1beta1().BackupConfigurations(backupSession.Namespace).Get(
		backupSession.Spec.BackupConfiguration.Name,
		metav1.GetOptions{},
	)
	if err != nil {
		return fmt.Errorf("can't get BackupConfiguration for BackupSession %s/%s, reason: %s", backupSession.Namespace, backupSession.Name, err)
	}

	// get repository
	repository, err := c.StashClient.StashV1alpha1().Repositories(backupConfiguration.Namespace).Get(backupConfiguration.Spec.Repository.Name, metav1.GetOptions{})
	if err != nil {
		return err
	}

	//get host Name
	host, err := util.GetHostName(backupConfiguration.Spec.Target.Ref)
	if err != nil {
		return err
	}

	//Configure Host, SecretDirectory, EnableCache and ScratchDirectory
	extraOpt := util.ExtraOptions{
		Host:        host,
		SecretDir:   c.SetupOpt.SecretDir,
		EnableCache: c.SetupOpt.EnableCache,
		ScratchDir:  util.ScratchDir,
	}

	//Configure setupOption
	setupOpt, err := util.SetupOptionsForRepository(*repository, extraOpt)
	if err != nil {
		return fmt.Errorf("setup option for repository fail")
	}

	if backupConfiguration.Spec.Target != nil && util.BackupModel(backupConfiguration.Spec.Target.Ref.Kind) == util.ModelSidecar {
		//init restic wrapper
		resticWrapper, err := restic.NewResticWrapper(setupOpt)
		if err != nil {
			return err
		}
		//BackupOptions configuration
		backupOpt := util.BackupOptionsForBackupConfig(*backupConfiguration, extraOpt)
		backupOutput, err := resticWrapper.RunBackup(&backupOpt)
		if err != nil {
			return err
		}

		// If metrics are enabled then generate metrics
		if c.Metrics.Enabled {
			err := backupOutput.HandleMetrics(&c.Metrics, nil)
			if err != nil {
				return err
			}
		}
		//Update Backup Session and Repository status
		//_, err = stash_v1beta1_util.UpdateRestoreSessionStatus()
		o := status.UpdateStatusOptions{
			Namespace:               c.Namespace,
			BackupSession:           backupSession.Name,
			Repository:              backupConfiguration.Spec.Repository.Name,
			EnableStatusSubresource: apis.EnableStatusSubresource,
		}
		err = o.UpdateBackupStatus(backupOutput, c.StashClient.(*cs.Clientset))
		if err != nil {
			return err
		}
		glog.Info("Backup has been completed successfully")
	}
	return nil
}

func IsPending(backupsession *v1beta1_api.BackupSession) bool {
	if backupsession.Status.Phase == v1beta1_api.BackupSessionPending || backupsession.Status.Phase == "" {
		return true
	}
	return false
}

func (c *Controllers) runBackupSessionController(backupConfiguration *v1beta1_api.BackupConfiguration, stopCh <-chan struct{}) error {
	// start BackupSession watcher
	err := c.initBackupSessionWatcher(backupConfiguration)
	if err != nil {
		return err
	}

	// start BackupSession informer
	c.StashInformerFactory.Start(stopCh)
	for _, v := range c.StashInformerFactory.WaitForCacheSync(stopCh) {
		if !v {
			runtime.HandleError(fmt.Errorf("timed out waiting for caches to sync"))
			return nil
		}
	}
	c.bsQueue.Run(stopCh)
	return nil
}

func (c *Controllers) HandleBackupFailure(backupErr error) error {
	backupSession, err := c.StashClient.StashV1beta1().BackupSessions(c.Namespace).Get(c.BackupConfigurationName, metav1.GetOptions{})
	if err != nil {
		return err
	}
	// set backupSession phase "Failed"
	_, err = stash_v1beta1_util.UpdateBackupSessionStatus(c.StashClient.StashV1beta1(), backupSession, func(status *v1beta1_api.BackupSessionStatus) *v1beta1_api.BackupSessionStatus {
		status.Phase = v1beta1_api.BackupSessionFailed
		return status
	}, apis.EnableStatusSubresource)

	// write failure event
	c.writeBackupFailureEvent(backupSession, backupErr)

	// send prometheus metrics
	if c.Metrics.Enabled {
		restoreOutput := &restic.RestoreOutput{}
		return restoreOutput.HandleMetrics(&c.Metrics, backupErr)
	}
	return nil
}

func (c *Controllers) writeBackupFailureEvent(backupSession *v1beta1_api.BackupSession, err error) {
	// write failure event
	ref, rerr := reference.GetReference(stash_scheme.Scheme, backupSession)
	if rerr == nil {
		eventer.CreateEventWithLog(
			c.K8sClient,
			controller.BackupSessionEventComponent,
			ref,
			core.EventTypeWarning,
			eventer.EventReasonBackupSessionFailed,
			fmt.Sprintf("Failed to restore. Reason: %v", err),
		)
	} else {
		log.Errorf("Failed to restore. Reason: %v", rerr)
	}
}

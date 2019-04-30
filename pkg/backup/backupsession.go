package backup

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
	v1beta1_util "github.com/appscode/stash/client/clientset/versioned/typed/stash/v1beta1/util"
	stashinformers "github.com/appscode/stash/client/informers/externalversions"
	"github.com/appscode/stash/client/listers/stash/v1beta1"
	"github.com/appscode/stash/pkg/eventer"
	"github.com/appscode/stash/pkg/restic"
	"github.com/appscode/stash/pkg/status"
	"github.com/appscode/stash/pkg/util"
	"github.com/golang/glog"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/errors"
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

type BackupSessionController struct {
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

func (c *BackupSessionController) RunBackup() error {
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

	// for Deployment, ReplicaSet and ReplicationController run BackupSession watcher only in leader pod.
	// for others workload i.e. DaemonSet and StatefulSet run BackupSession watcher in all pods.
	switch backupConfiguration.Spec.Target.Ref.Kind {
	case apis.KindDeployment, apis.KindReplicaSet, apis.KindReplicationController, apis.KindDeploymentConfig:
		if err := c.electLeaderPod(backupConfiguration, stopCh); err != nil {
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

func (c *BackupSessionController) runBackupSessionController(backupConfiguration *api_v1beta1.BackupConfiguration, stopCh <-chan struct{}) error {
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

	// wait until stop signal is sent.
	<-stopCh
	return nil
}

func (c *BackupSessionController) initBackupSessionWatcher(backupConfiguration *api_v1beta1.BackupConfiguration) error {
	// only watch BackupSessions of this BackupConfiguration.
	// respective CronJob creates BackupSession with BackupConfiguration's name as label.
	// so we will watch only those BackupSessions that has this BackupConfiguration name in labels.
	selector, err := metav1.LabelSelectorAsSelector(&metav1.LabelSelector{
		MatchLabels: map[string]string{
			util.LabelBackupConfiguration: backupConfiguration.Name,
		},
	})
	if err != nil {
		return err
	}

	c.bsInformer = c.StashInformerFactory.Stash().V1beta1().BackupSessions().Informer()
	c.bsQueue = queue.New(api_v1beta1.ResourceKindBackupSession, c.MaxNumRequeues, c.NumThreads, c.processBackupSession)
	c.bsInformer.AddEventHandler(queue.NewFilteredHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			if backupsession, ok := obj.(*api_v1beta1.BackupSession); ok {
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
func (c *BackupSessionController) processBackupSession(key string) error {
	obj, exists, err := c.bsInformer.GetIndexer().GetByKey(key)
	if err != nil {
		glog.Errorf("Fetching object with key %s from store failed with %v", key, err)
		return err
	}
	if !exists {
		glog.Warningf("Backup Session %s does not exist anymore\n", key)

	} else {
		backupSession := obj.(*api_v1beta1.BackupSession)
		glog.Infof("Sync/Add/Update for Backup Session %s", backupSession.GetName())

		// get respective BackupConfiguration for BackupSession
		backupConfiguration, err := c.StashClient.StashV1beta1().BackupConfigurations(backupSession.Namespace).Get(
			backupSession.Spec.BackupConfiguration.Name,
			metav1.GetOptions{},
		)
		if err != nil {
			return fmt.Errorf("can't get BackupConfiguration for BackupSession %s/%s, reason: %s", backupSession.Namespace, backupSession.Name, err)
		}

		// skip if BackupConfiguration paused
		if backupConfiguration.Spec.Paused {
			log.Infof("Skipping processing BackupSession %s/%s. Reason: Backup Configuration is paused.", backupSession.Namespace, backupSession.Name)
			return nil
		}

		host, err := util.GetHostName(backupConfiguration.Spec.Target)
		if err != nil {
			return err
		}

		// if BackupSession already has been processed for this host then skip further processing
		if c.isBackupTakenForThisHost(backupSession, host) {
			log.Infof("Skip processing BackupSession %s/%s. Reason: BackupSession has been processed already for host %q\n", backupSession.Namespace, backupSession.Name, host)
			return nil
		}

		// For Deployment, ReplicaSet and ReplicationController only leader pod is running this controller so no problem with restic repo lock.
		// For StatefulSet and DaemonSet all pods are running this controller and all will try to backup simultaneously. But, restic repository can be
		// locked by only one pod. So, we need a leader election to determine who will take backup first. Once backup is complete, the leader pod will
		// step down from leadership so that another replica can acquire leadership and start taking backup.
		switch backupConfiguration.Spec.Target.Ref.Kind {
		case apis.KindDeployment, apis.KindReplicaSet, apis.KindReplicationController, apis.KindDeploymentConfig:
			return c.backup(backupSession, backupConfiguration)
		default:
			return c.electBackupLeader(backupSession, backupConfiguration)
		}
	}
	return nil
}

func (c *BackupSessionController) backup(backupSession *api_v1beta1.BackupSession, backupConfiguration *api_v1beta1.BackupConfiguration) error {

	// get repository
	repository, err := c.StashClient.StashV1alpha1().Repositories(backupConfiguration.Namespace).Get(backupConfiguration.Spec.Repository.Name, metav1.GetOptions{})
	if err != nil {
		return err
	}

	// get host name
	host, err := util.GetHostName(backupConfiguration.Spec.Target)
	if err != nil {
		return err
	}

	// configure SourceHost, SecretDirectory, EnableCache and ScratchDirectory
	extraOpt := util.ExtraOptions{
		Host:        host,
		SecretDir:   c.SetupOpt.SecretDir,
		EnableCache: c.SetupOpt.EnableCache,
		ScratchDir:  c.SetupOpt.ScratchDir,
	}

	// configure setupOption
	c.SetupOpt, err = util.SetupOptionsForRepository(*repository, extraOpt)
	if err != nil {
		return fmt.Errorf("setup option for repository fail")
	}

	// apply nice/ionice settings
	if backupConfiguration.Spec.RuntimeSettings.Container != nil {
		c.SetupOpt.Nice = backupConfiguration.Spec.RuntimeSettings.Container.Nice
		c.SetupOpt.IONice = backupConfiguration.Spec.RuntimeSettings.Container.IONice
	}

	// init restic wrapper
	resticWrapper, err := restic.NewResticWrapper(c.SetupOpt)
	if err != nil {
		return err
	}

	// BackupOptions configuration
	backupOpt := util.BackupOptionsForBackupConfig(*backupConfiguration, extraOpt)
	backupOutput, err := resticWrapper.RunBackup(backupOpt)
	if err != nil {
		return err
	}

	// if metrics are enabled then generate metrics
	if c.Metrics.Enabled {
		err := backupOutput.HandleMetrics(&c.Metrics, nil)
		if err != nil {
			return err
		}
	}

	// Update Backup Session and Repository status
	o := status.UpdateStatusOptions{
		KubeClient:    c.K8sClient,
		StashClient:   c.StashClient.(*cs.Clientset),
		Namespace:     c.Namespace,
		BackupSession: backupSession.Name,
		Repository:    backupConfiguration.Spec.Repository.Name,
	}

	err = o.UpdatePostBackupStatus(backupOutput)
	if err != nil {
		return err
	}

	glog.Info("Backup has been completed successfully")

	return nil
}

func (c *BackupSessionController) electLeaderPod(backupConfiguration *api_v1beta1.BackupConfiguration, stopCh <-chan struct{}) error {
	log.Infoln("Attempting to elect leader pod")

	rlc := resourcelock.ResourceLockConfig{
		Identity:      os.Getenv(util.KeyPodName),
		EventRecorder: eventer.NewEventRecorder(c.K8sClient, BackupEventComponent),
	}
	resLock, err := resourcelock.New(
		resourcelock.ConfigMapsResourceLock,
		backupConfiguration.Namespace,
		util.GetBackupConfigmapLockName(backupConfiguration.Spec.Target.Ref),
		c.K8sClient.CoreV1(),
		rlc,
	)
	if err != nil {
		return fmt.Errorf("failed to create resource lock. Reason: %s", err)
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
				log.Infoln("Got leadership, preparing starting BackupSession controller")
				// this pod is now leader. run BackupSession controller.
				err := c.runBackupSessionController(backupConfiguration, stopCh)
				if err != nil {
					e2 := c.HandleBackupFailure(err)
					if e2 != nil {
						err = errors.NewAggregate([]error{err, e2})
					}
					// step down from leadership so that other replicas can try to start BackupSession controller
					cancel()
					// fail the container so that it restart and re-try this process.
					log.Fatalln("failed to start BackupSession controller. Reason: ", err.Error())
				}
			},
			OnStoppedLeading: func() {
				log.Infoln("Lost leadership")
			},
		},
	})
	return nil
}

func (c *BackupSessionController) electBackupLeader(backupSession *api_v1beta1.BackupSession, backupConfiguration *api_v1beta1.BackupConfiguration) error {
	log.Infoln("Attempting to acquire leadership for backup")

	rlc := resourcelock.ResourceLockConfig{
		Identity:      os.Getenv(util.KeyPodName),
		EventRecorder: eventer.NewEventRecorder(c.K8sClient, BackupEventComponent),
	}

	resLock, err := resourcelock.New(
		resourcelock.ConfigMapsResourceLock,
		backupConfiguration.Namespace,
		util.GetBackupConfigmapLockName(backupConfiguration.Spec.Target.Ref),
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

	// start the leader election code loop
	leaderelection.RunOrDie(ctx, leaderelection.LeaderElectionConfig{
		Lock:          resLock,
		LeaseDuration: 15 * time.Second,
		RenewDeadline: 10 * time.Second,
		RetryPeriod:   2 * time.Second,
		Callbacks: leaderelection.LeaderCallbacks{
			OnStartedLeading: func(ctx context.Context) {
				log.Infoln("Got leadership, preparing for backup")
				// run backup process
				err := c.backup(backupSession, backupConfiguration)
				if err != nil {
					e2 := c.HandleBackupFailure(err)
					if e2 != nil {
						err = errors.NewAggregate([]error{err, e2})
					}
					// step down from leadership so that other replicas can start backup
					cancel()
					// fail the container so that it restart and re-try to backup
					log.Fatalln("failed to complete backup. Reason: ", err.Error())
				}
				// backup process is complete. now, step down from leadership so that other replicas can start
				cancel()
			},
			OnStoppedLeading: func() {
				log.Infoln("Lost leadership")
			},
		},
	})
	return nil
}

func (c *BackupSessionController) HandleBackupFailure(backupErr error) error {
	backupSession, err := c.StashClient.StashV1beta1().BackupSessions(c.Namespace).Get(c.BackupConfigurationName, metav1.GetOptions{})
	if err != nil {
		return err
	}

	backupConfiguration, err := c.StashClient.StashV1beta1().BackupConfigurations(backupSession.Namespace).Get(backupSession.Spec.BackupConfiguration.Name, metav1.GetOptions{})
	if err != nil {
		return err
	}

	host, err := util.GetHostName(backupConfiguration.Spec.Target)
	if err != nil {
		return err
	}

	hostStats := api_v1beta1.HostBackupStats{
		Hostname: host,
		Phase:    api_v1beta1.HostBackupFailed,
		Error:    backupErr.Error(),
	}

	// add or update entry for this host in BackupSession status
	_, err = v1beta1_util.UpdateBackupSessionStatusForHost(c.StashClient.StashV1beta1(), backupSession, hostStats)

	// write failure event
	c.writeBackupFailureEvent(backupSession, host, backupErr)

	// send prometheus metrics
	if c.Metrics.Enabled {
		backupOutput := &restic.BackupOutput{
			HostBackupStats: hostStats,
		}
		return backupOutput.HandleMetrics(&c.Metrics, backupErr)
	}
	return nil
}

func (c *BackupSessionController) writeBackupFailureEvent(backupSession *api_v1beta1.BackupSession, host string, err error) {
	// write failure event
	ref, rerr := reference.GetReference(stash_scheme.Scheme, backupSession)
	if rerr == nil {
		eventer.CreateEventWithLog(
			c.K8sClient,
			eventer.BackupSessionEventComponent,
			ref,
			core.EventTypeWarning,
			eventer.EventReasonHostBackupFailed,
			fmt.Sprintf("Failed to backup host %q. Reason: %v", host, err),
		)
	} else {
		log.Errorf("Failed to write backup failure event. Reason: %v", rerr)
	}
}

func (c *BackupSessionController) isBackupTakenForThisHost(backupSession *api_v1beta1.BackupSession, host string) bool {

	// if overall backupSession phase is "Succeeded" or "Failed" or "Skipped" then it has been processed already
	if backupSession.Status.Phase == api_v1beta1.BackupSessionSucceeded ||
		backupSession.Status.Phase == api_v1beta1.BackupSessionFailed ||
		backupSession.Status.Phase == api_v1beta1.BackupSessionSkipped {
		return true
	}

	// if backupSession has entry for this host in status field, then it has been already processed for this host
	for _, hostStats := range backupSession.Status.Stats {
		if hostStats.Hostname == host {
			return true
		}
	}
	return false
}

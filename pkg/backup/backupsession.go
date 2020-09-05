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

package backup

import (
	"context"
	"fmt"
	"os"
	"reflect"
	"time"

	"stash.appscode.dev/apimachinery/apis"
	api_v1beta1 "stash.appscode.dev/apimachinery/apis/stash/v1beta1"
	cs "stash.appscode.dev/apimachinery/client/clientset/versioned"
	stashinformers "stash.appscode.dev/apimachinery/client/informers/externalversions"
	"stash.appscode.dev/apimachinery/client/listers/stash/v1beta1"
	"stash.appscode.dev/apimachinery/pkg/restic"
	api_util "stash.appscode.dev/apimachinery/pkg/util"
	"stash.appscode.dev/stash/pkg/eventer"
	"stash.appscode.dev/stash/pkg/status"
	"stash.appscode.dev/stash/pkg/util"

	"github.com/appscode/go/log"
	"github.com/golang/glog"
	core "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/errors"
	"k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/leaderelection"
	"k8s.io/client-go/tools/leaderelection/resourcelock"
	"k8s.io/client-go/tools/record"
	"kmodules.xyz/client-go/meta"
	"kmodules.xyz/client-go/tools/queue"
	v1 "kmodules.xyz/offshoot-api/api/v1"
)

type BackupSessionController struct {
	Config               *rest.Config
	K8sClient            kubernetes.Interface
	StashClient          cs.Interface
	MasterURL            string
	KubeconfigPath       string
	StashInformerFactory stashinformers.SharedInformerFactory
	MaxNumRequeues       int
	NumThreads           int
	ResyncPeriod         time.Duration
	//backupConfiguration/BackupBatch
	InvokerKind      string
	InvokerName      string
	Namespace        string
	BackupTargetName string
	BackupTargetKind string
	//Backup Session
	bsQueue    *queue.Worker
	bsInformer cache.SharedIndexInformer
	bsLister   v1beta1.BackupSessionLister

	SetupOpt restic.SetupOptions
	Host     string
	Metrics  restic.MetricsOptions
	Recorder record.EventRecorder
}

func (c *BackupSessionController) RunBackup(targetInfo apis.TargetInfo, invokerRef *core.ObjectReference) error {
	stopCh := make(chan struct{})
	defer close(stopCh)

	// for Deployment, ReplicaSet and ReplicationController run BackupSession watcher only in leader pod.
	// for others workload i.e. DaemonSet and StatefulSet run BackupSession watcher in all pods.
	switch targetInfo.Target.Ref.Kind {
	case apis.KindDeployment, apis.KindReplicaSet, apis.KindReplicationController, apis.KindDeploymentConfig:
		if err := c.electLeaderPod(targetInfo, invokerRef, stopCh); err != nil {
			return err
		}
	default:
		if err := c.runBackupSessionController(invokerRef, stopCh); err != nil {
			return err
		}
	}
	glog.Info("Stopping Stash backup")
	return nil
}

func (c *BackupSessionController) runBackupSessionController(invokerRef *core.ObjectReference, stopCh <-chan struct{}) error {
	// start BackupSession watcher
	err := c.initBackupSessionWatcher()
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

	// BackupSession controller has started successfully. write event to the respective BackupConfiguration.
	err = c.handleBackupSetupSuccess(invokerRef)
	if err != nil {
		return err
	}

	// wait until stop signal is sent.
	<-stopCh
	return nil
}

func (c *BackupSessionController) initBackupSessionWatcher() error {
	// Only watches for BackupSession of the respective invoker.
	// Respective CronJob creates BackupSession with invoker's name and kind as label.
	selector, err := metav1.LabelSelectorAsSelector(&metav1.LabelSelector{
		MatchLabels: map[string]string{
			apis.LabelInvokerType: c.InvokerKind,
			apis.LabelInvokerName: c.InvokerName,
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
		UpdateFunc: func(oldObj, newObj interface{}) {
			oldBS, ok := oldObj.(*api_v1beta1.BackupSession)
			if !ok {
				glog.Errorf("Invalid BackupSession Object")
				return
			}
			newBS, ok := newObj.(*api_v1beta1.BackupSession)
			if !ok {
				glog.Errorf("Invalid BackupSession Object")
				return
			}
			if !reflect.DeepEqual(&oldBS.Status, &newBS.Status) {
				queue.Enqueue(c.bsQueue.GetQueue(), newObj)
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

		invoker, err := apis.ExtractBackupInvokerInfo(c.StashClient, backupSession.Spec.Invoker.Kind, backupSession.Spec.Invoker.Name, c.Namespace)
		if err != nil {
			return err
		}

		for _, targetInfo := range invoker.TargetsInfo {
			if targetInfo.Target != nil && c.targetMatched(targetInfo.Target.Ref) {

				// Ensure Execution Order
				if invoker.ExecutionOrder == api_v1beta1.Sequential &&
					!invoker.NextInOrder(targetInfo.Target.Ref, backupSession.Status.Targets) {
					// backup order is sequential and the current target is not yet to be executed.
					glog.Infof("Skipping backup. Reason: Backup order is sequential and some previous targets hasn't completed their backup process.")
					return nil
				}

				backupOutput, err := c.startBackupProcess(backupSession, invoker, targetInfo)
				if err != nil {
					return c.handleBackupFailure(backupSession.Name, invoker, targetInfo, err)
				}

				if backupOutput != nil {
					return c.handleBackupSuccess(backupSession.Name, invoker, targetInfo, backupOutput)
				}
			}
		}

	}
	return nil
}

func (c *BackupSessionController) startBackupProcess(backupSession *api_v1beta1.BackupSession, invoker apis.Invoker, targetInfo apis.TargetInfo) (*restic.BackupOutput, error) {
	// if BackupSession already has been processed for this host then skip further processing
	if c.isBackupTakenForThisHost(backupSession, targetInfo.Target) {
		log.Infof("Skip processing BackupSession %s/%s. Reason: BackupSession has been processed already for host %q", backupSession.Namespace, backupSession.Name, c.Host)
		return nil, nil
	}

	if !apis.TargetBackupInitiated(targetInfo.Target.Ref, backupSession.Status.Targets) {
		log.Infof("Skip processing BackupSession %s/%s. Reason: Backup process is not initiated by the operator", backupSession.Namespace, backupSession.Name)
		return nil, nil
	}
	// For Deployment, ReplicaSet and ReplicationController only leader pod is running this controller so no problem with restic repo lock.
	// For StatefulSet and DaemonSet all pods are running this controller and all will try to backup simultaneously. But, restic repository can be
	// locked by only one pod. So, we need a leader election to determine who will take backup first. Once backup is complete, the leader pod will
	// step down from leadership so that another replica can acquire leadership and start taking backup.
	switch targetInfo.Target.Ref.Kind {
	case apis.KindDeployment, apis.KindReplicaSet, apis.KindReplicationController, apis.KindDeploymentConfig:
		return c.backup(invoker, targetInfo, backupSession)
	default:
		return nil, c.electBackupLeader(backupSession, invoker, targetInfo)
	}
}

func (c *BackupSessionController) backup(invoker apis.Invoker, targetInfo apis.TargetInfo, backupSession *api_v1beta1.BackupSession) (*restic.BackupOutput, error) {
	_, err := c.setSetupOptions(invoker.Repository)
	if err != nil {
		return nil, err
	}
	// If there is any pre-backup actions assigned to this target, execute them first.
	err = api_util.ExecutePreBackupActions(api_util.ActionOptions{
		StashClient:       c.StashClient,
		TargetRef:         targetInfo.Target.Ref,
		SetupOptions:      c.SetupOpt,
		BackupSessionName: backupSession.Name,
		Namespace:         backupSession.Namespace,
	})
	if err != nil {
		return nil, err
	}

	repoInitialized, err := api_util.IsRepositoryInitialized(api_util.ActionOptions{
		StashClient:       c.StashClient,
		BackupSessionName: backupSession.Name,
		Namespace:         backupSession.Namespace,
	})
	if err != nil {
		return nil, err
	}
	// If the repository hasn't been initialized yet, it means some other process is responsible to initialize the repository.
	// So, retry after 5 seconds.
	if !repoInitialized {
		glog.Infof("Waiting for the backend repository.....")
		c.bsQueue.GetQueue().AddAfter(fmt.Sprintf("%s/%s", backupSession.Namespace, backupSession.Name), 5*time.Second)
		return nil, nil
	}

	// If preBackup hook is specified, then execute those hooks first
	if targetInfo.Hooks != nil && targetInfo.Hooks.PreBackup != nil {
		err := util.ExecuteHook(c.Config, targetInfo.Hooks, apis.PreBackupHook, os.Getenv(apis.KeyPodName), c.Namespace)
		if err != nil {
			return nil, err
		}
	}
	extraOpt, err := c.setSetupOptions(invoker.Repository)
	if err != nil {
		return nil, err
	}

	// init restic wrapper
	resticWrapper, err := restic.NewResticWrapper(c.SetupOpt)
	if err != nil {
		return nil, err
	}

	// BackupOptions backup target
	backupOpt := util.BackupOptionsForBackupTarget(targetInfo.Target, invoker.RetentionPolicy, *extraOpt)
	// Run Backup
	// If there is an error during backup, don't return.
	// We will execute postBackup hook even if the backup failed.
	// Reason: https://github.com/stashed/stash/issues/986
	var backupErr, hookErr error
	output, backupErr := resticWrapper.RunBackup(backupOpt, targetInfo.Target.Ref)

	// If postBackup hook is specified, then execute those hooks
	if targetInfo.Hooks != nil && targetInfo.Hooks.PostBackup != nil {
		hookErr = util.ExecuteHook(c.Config, targetInfo.Hooks, apis.PostBackupHook, os.Getenv(apis.KeyPodName), c.Namespace)
	}
	if backupErr != nil || hookErr != nil {
		return nil, errors.NewAggregate([]error{backupErr, hookErr})
	}
	return output, nil
}

func (c *BackupSessionController) electLeaderPod(targetInfo apis.TargetInfo, invokerRef *core.ObjectReference, stopCh <-chan struct{}) error {
	log.Infoln("Attempting to elect leader pod")

	rlc := resourcelock.ResourceLockConfig{
		Identity:      os.Getenv(apis.KeyPodName),
		EventRecorder: eventer.NewEventRecorder(c.K8sClient, BackupEventComponent),
	}
	resLock, err := resourcelock.New(
		resourcelock.ConfigMapsResourceLock,
		c.Namespace,
		util.GetBackupConfigmapLockName(targetInfo.Target.Ref),
		c.K8sClient.CoreV1(),
		c.K8sClient.CoordinationV1(),
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
				log.Infoln("Got leadership, starting BackupSession controller")
				// this pod is now leader. run BackupSession controller.
				setupErr := c.runBackupSessionController(invokerRef, stopCh)
				if setupErr != nil {
					// write event to the respective BackpConfiguration
					err := c.HandleBackupSetupFailure(invokerRef, err)
					// step down from leadership so that other replicas can try to start BackupSession controller
					cancel()
					// fail the container so that it restart and retry.
					// we should not fail container as it may interrupt user's service.
					// however, we are doing it here because it is happening for the first time when stash has injected
					// a sidecar. user's pod will restart automatically for sidecar injection. so, we can restart the pod
					// to ensure backup has been configured properly at this time as the user will be aware of service interruption.
					if err != nil {
						log.Fatal(err)
					}
				}
			},
			OnStoppedLeading: func() {
				log.Infoln("Lost leadership")
			},
		},
	})
	return nil
}

func (c *BackupSessionController) electBackupLeader(backupSession *api_v1beta1.BackupSession, invoker apis.Invoker, targetInfo apis.TargetInfo) error {
	log.Infoln("Attempting to acquire leadership for backup")

	rlc := resourcelock.ResourceLockConfig{
		Identity:      os.Getenv(apis.KeyPodName),
		EventRecorder: eventer.NewEventRecorder(c.K8sClient, BackupEventComponent),
	}

	resLock, err := resourcelock.New(
		resourcelock.ConfigMapsResourceLock,
		c.Namespace,
		util.GetBackupConfigmapLockName(targetInfo.Target.Ref),
		c.K8sClient.CoreV1(),
		c.K8sClient.CoordinationV1(),
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
				backupOutput, backupErr := c.backup(invoker, targetInfo, backupSession)
				if backupErr != nil {
					err := c.handleBackupFailure(backupSession.Name, invoker, targetInfo, backupErr)
					if err != nil {
						backupErr = errors.NewAggregate([]error{backupErr, err})
					}
					// step down from leadership so that other replicas can start backup
					cancel()
					// log failure. don't fail the container as it may interrupt user's service
					log.Warningf("failed to complete backup. Reason: %v", backupErr)
				}
				if backupOutput != nil {
					err := c.handleBackupSuccess(backupSession.Name, invoker, targetInfo, backupOutput)
					if err != nil {
						// log failure. don't fail the container as it may interrupt user's service
						log.Warningf("failed to complete backup. Reason: %v", err)
					}
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

func (c *BackupSessionController) HandleBackupSetupFailure(ref *core.ObjectReference, setupErr error) error {
	// write log
	log.Warningf("failed to start BackupSessionController. Reason: %v", setupErr)

	_, err := eventer.CreateEvent(
		c.K8sClient,
		eventer.EventSourceBackupSidecar,
		ref,
		core.EventTypeWarning,
		eventer.EventReasonFailedToStartBackupSessionController,
		fmt.Sprintf("failed to start BackupSession controller in pod %s/%s Reason: %v", meta.Namespace(), os.Getenv(apis.KeyPodName), setupErr),
	)
	return errors.NewAggregate([]error{setupErr, err})
}

func (c *BackupSessionController) handleBackupSetupSuccess(invokerRef *core.ObjectReference) error {
	// write log
	log.Infof("BackupSession controller started successfully.")

	// Write event to the invoker
	_, err := eventer.CreateEvent(
		c.K8sClient,
		eventer.EventSourceBackupSidecar,
		invokerRef,
		core.EventTypeNormal,
		eventer.EventReasonBackupSessionControllerStarted,
		fmt.Sprintf("BackupSession controller started successfully in pod %s/%s.", meta.Namespace(), os.Getenv(apis.KeyPodName)),
	)
	return err
}

func (c *BackupSessionController) handleBackupSuccess(backupSessionName string, invoker apis.Invoker, targetInfo apis.TargetInfo, backupOutput *restic.BackupOutput) error {
	// write log
	log.Infof("Backup completed successfully for BackupSession %s", backupSessionName)

	statusOpt := status.UpdateStatusOptions{
		Config:        c.Config,
		KubeClient:    c.K8sClient,
		StashClient:   c.StashClient,
		Namespace:     c.Namespace,
		Repository:    invoker.Repository,
		BackupSession: backupSessionName,
		Metrics:       c.Metrics,
		SetupOpt:      c.SetupOpt,
	}
	if targetInfo.Target != nil {
		statusOpt.TargetRef = targetInfo.Target.Ref
	}

	return statusOpt.UpdatePostBackupStatus(backupOutput, invoker, targetInfo)
}

func (c *BackupSessionController) handleBackupFailure(backupSessionName string, invoker apis.Invoker, targetInfo apis.TargetInfo, backupErr error) error {
	// write log
	log.Warningf("Failed to take backup for BackupSession %s. Reason: %v", backupSessionName, backupErr)
	backupOutput := &restic.BackupOutput{
		BackupTargetStatus: api_v1beta1.BackupTargetStatus{
			Ref: targetInfo.Target.Ref,
			Stats: []api_v1beta1.HostBackupStats{
				{
					Hostname: c.Host,
					Phase:    api_v1beta1.HostBackupFailed,
					Error:    fmt.Sprintf("failed to complete backup for host %s. Reason: %v", c.Host, backupErr),
				},
			},
		},
	}

	statusOpt := status.UpdateStatusOptions{
		Config:        c.Config,
		KubeClient:    c.K8sClient,
		StashClient:   c.StashClient,
		Namespace:     c.Namespace,
		Repository:    invoker.Repository,
		BackupSession: backupSessionName,
		Metrics:       c.Metrics,
		TargetRef:     targetInfo.Target.Ref,
		SetupOpt:      c.SetupOpt,
	}

	return statusOpt.UpdatePostBackupStatus(backupOutput, invoker, targetInfo)
}

func (c *BackupSessionController) isBackupTakenForThisHost(backupSession *api_v1beta1.BackupSession, backupTarget *api_v1beta1.BackupTarget) bool {

	// if overall backupSession phase is "Succeeded" or "Failed" or "Skipped" then it has been processed already
	if backupSession.Status.Phase == api_v1beta1.BackupSessionSucceeded ||
		backupSession.Status.Phase == api_v1beta1.BackupSessionFailed {
		return true
	}

	// if backupSession has entry for this host in status field, then it has been already processed for this host
	for _, target := range backupSession.Status.Targets {
		if backupTarget != nil &&
			backupTarget.Ref.Kind == c.BackupTargetKind &&
			backupTarget.Ref.Name == c.BackupTargetName {
			for _, hostStats := range target.Stats {
				if hostStats.Hostname == c.Host {
					return true
				}
			}
		}
	}
	return false
}

func (c *BackupSessionController) targetMatched(ref api_v1beta1.TargetRef) bool {
	if ref.Kind == c.BackupTargetKind &&
		ref.Name == c.BackupTargetName {
		return true
	}
	return false
}

func (c *BackupSessionController) setSetupOptions(repoName string) (*util.ExtraOptions, error) {
	// get repository
	repository, err := c.StashClient.StashV1alpha1().Repositories(c.Namespace).Get(context.TODO(), repoName, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}

	// configure SourceHost, SecretDirectory, EnableCache and ScratchDirectory
	extraOpt := &util.ExtraOptions{
		Host:        c.Host,
		SecretDir:   c.SetupOpt.SecretDir,
		EnableCache: c.SetupOpt.EnableCache,
		ScratchDir:  c.SetupOpt.ScratchDir,
	}

	// configure setupOption
	c.SetupOpt, err = util.SetupOptionsForRepository(*repository, *extraOpt)
	if err != nil {
		return nil, fmt.Errorf("setup option for repository fail")
	}

	// apply nice, ionice settings from env
	c.SetupOpt.Nice, err = v1.NiceSettingsFromEnv()
	if err != nil {
		return nil, err
	}
	c.SetupOpt.IONice, err = v1.IONiceSettingsFromEnv()
	if err != nil {
		return nil, err
	}
	return extraOpt, nil
}

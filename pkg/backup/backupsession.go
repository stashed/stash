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
	"reflect"
	"time"

	"stash.appscode.dev/apimachinery/apis"
	api_v1beta1 "stash.appscode.dev/apimachinery/apis/stash/v1beta1"
	cs "stash.appscode.dev/apimachinery/client/clientset/versioned"
	stashinformers "stash.appscode.dev/apimachinery/client/informers/externalversions"
	"stash.appscode.dev/apimachinery/client/listers/stash/v1beta1"
	stashHooks "stash.appscode.dev/apimachinery/pkg/hooks"
	"stash.appscode.dev/apimachinery/pkg/invoker"
	"stash.appscode.dev/apimachinery/pkg/metrics"
	"stash.appscode.dev/apimachinery/pkg/restic"
	api_util "stash.appscode.dev/apimachinery/pkg/util"
	"stash.appscode.dev/stash/pkg/eventer"
	"stash.appscode.dev/stash/pkg/status"
	"stash.appscode.dev/stash/pkg/util"

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
	"k8s.io/klog/v2"
	kmapi "kmodules.xyz/client-go/api/v1"
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
	// BackupConfiguration/BackupBatch
	InvokerKind string
	InvokerName string
	Namespace   string
	// Backup Session
	bsQueue    *queue.Worker
	bsInformer cache.SharedIndexInformer
	bsLister   v1beta1.BackupSessionLister

	TargetRef api_v1beta1.TargetRef

	SetupOpt restic.SetupOptions
	Host     string
	Metrics  metrics.MetricsOptions
	Recorder record.EventRecorder
}

const BackupEventComponent = "stash-backup"

func (c *BackupSessionController) RunBackup(targetInfo invoker.BackupTargetInfo, invokerRef *core.ObjectReference) error {
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
	klog.Info("Stopping Stash backup")
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
	// Only watches for BackupSession of the respective inv.
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
				klog.Errorf("Invalid BackupSession Object")
				return
			}
			newBS, ok := newObj.(*api_v1beta1.BackupSession)
			if !ok {
				klog.Errorf("Invalid BackupSession Object")
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
		klog.Errorf("Fetching object with key %s from store failed with %v", key, err)
		return err
	}
	if !exists {
		klog.Warningf("Backup Session %s does not exist anymore\n", key)
	} else {
		backupSession := obj.(*api_v1beta1.BackupSession)
		klog.Infof("Sync/Add/Update for Backup Session %s", backupSession.GetName())

		inv, err := invoker.NewBackupInvoker(c.StashClient, backupSession.Spec.Invoker.Kind, backupSession.Spec.Invoker.Name, c.Namespace)
		if err != nil {
			return err
		}

		for _, targetInfo := range inv.GetTargetInfo() {
			if targetInfo.Target != nil && c.targetMatched(targetInfo.Target.Ref) {

				// Ensure Execution Order
				if inv.GetExecutionOrder() == api_v1beta1.Sequential &&
					!inv.NextInOrder(targetInfo.Target.Ref, backupSession.Status.Targets) {
					// backup order is sequential and the current target is not yet to be executed.
					klog.Infof("Skipping backup. Reason: Backup order is sequential and some previous targets hasn't completed their backup process.")
					return nil
				}
				return c.startBackupProcess(backupSession, inv, targetInfo)
			}
		}
	}
	return nil
}

func (c *BackupSessionController) startBackupProcess(backupSession *api_v1beta1.BackupSession, inv invoker.BackupInvoker, targetInfo invoker.BackupTargetInfo) error {
	// if BackupSession already has been processed for this host then skip further processing
	if c.isBackupTakenForThisHost(backupSession, targetInfo.Target) {
		klog.Infof("Skip processing BackupSession %s/%s. Reason: BackupSession has been processed already for host %q", backupSession.Namespace, backupSession.Name, c.Host)
		return nil
	}

	if !invoker.TargetBackupInitiated(targetInfo.Target.Ref, backupSession.Status.Targets) {
		klog.Infof("Skip processing BackupSession %s/%s. Reason: Backup process is not initiated by the operator", backupSession.Namespace, backupSession.Name)
		return nil
	}

	// For Deployment, ReplicaSet and ReplicationController only leader pod is running this controller so no problem with restic repo lock.
	// For StatefulSet and DaemonSet all pods are running this controller and all will try to backup simultaneously. But, restic repository can be
	// locked by only one pod. So, we need a leader election to determine who will take backup first. Once backup is complete, the leader pod will
	// step down from leadership so that another replica can acquire leadership and start taking backup.
	switch targetInfo.Target.Ref.Kind {
	case apis.KindDeployment, apis.KindReplicaSet, apis.KindReplicationController, apis.KindDeploymentConfig:
		return c.backupHost(inv, targetInfo, backupSession)
	default:
		return c.electBackupLeader(backupSession, inv, targetInfo)
	}
}

func (c *BackupSessionController) backupHost(inv invoker.BackupInvoker, targetInfo invoker.BackupTargetInfo, backupSession *api_v1beta1.BackupSession) error {
	// If preBackup hook is specified, then execute those hooks first
	if targetInfo.Hooks != nil && targetInfo.Hooks.PreBackup != nil {
		err := c.executePreBackupHook(inv, targetInfo, backupSession)
		if err != nil {
			klog.Infof("failed to execute preBackup hook. Reason: ", err)
			return nil
		}
	}

	output, err := c.backup(inv, targetInfo, backupSession)
	if err != nil {
		return c.handleBackupFailure(backupSession, inv, targetInfo, err)
	}
	return c.handleBackupSuccess(backupSession, inv, targetInfo, output)
}

func (c *BackupSessionController) backup(inv invoker.BackupInvoker, targetInfo invoker.BackupTargetInfo, backupSession *api_v1beta1.BackupSession) (*restic.BackupOutput, error) {
	_, err := c.setSetupOptions(inv.GetRepoRef())
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
		klog.Infof("Waiting for the backend repository.....")
		c.bsQueue.GetQueue().AddAfter(fmt.Sprintf("%s/%s", backupSession.Namespace, backupSession.Name), 5*time.Second)
		return nil, nil
	}

	extraOpt, err := c.setSetupOptions(inv.GetRepoRef())
	if err != nil {
		return nil, err
	}

	// init restic wrapper
	resticWrapper, err := restic.NewResticWrapper(c.SetupOpt)
	if err != nil {
		return nil, err
	}
	backupOpt := util.BackupOptionsForBackupTarget(targetInfo.Target, inv.GetRetentionPolicy(), *extraOpt)
	return resticWrapper.RunBackup(backupOpt, targetInfo.Target.Ref)
}

func (c *BackupSessionController) electLeaderPod(targetInfo invoker.BackupTargetInfo, invokerRef *core.ObjectReference, stopCh <-chan struct{}) error {
	klog.Infoln("Attempting to elect leader pod")

	rlc := resourcelock.ResourceLockConfig{
		Identity:      meta.PodName(),
		EventRecorder: eventer.NewEventRecorder(c.K8sClient, BackupEventComponent),
	}
	resLock, err := resourcelock.New(
		resourcelock.ConfigMapsLeasesResourceLock,
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
				klog.Infoln("Got leadership, starting BackupSession controller")
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
						klog.Fatal(err)
					}
				}
			},
			OnStoppedLeading: func() {
				klog.Infoln("Lost leadership")
			},
		},
	})
	return nil
}

func (c *BackupSessionController) electBackupLeader(backupSession *api_v1beta1.BackupSession, inv invoker.BackupInvoker, targetInfo invoker.BackupTargetInfo) error {
	klog.Infoln("Attempting to acquire leadership for backup")

	rlc := resourcelock.ResourceLockConfig{
		Identity:      meta.PodName(),
		EventRecorder: eventer.NewEventRecorder(c.K8sClient, BackupEventComponent),
	}

	resLock, err := resourcelock.New(
		resourcelock.ConfigMapsLeasesResourceLock,
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
				klog.Infoln("Got leadership, preparing for backup")
				// run backup process
				_ = c.backupHost(inv, targetInfo, backupSession)

				// backup process is complete. now, step down from leadership so that other replicas can start
				cancel()
			},
			OnStoppedLeading: func() {
				klog.Infoln("Lost leadership")
			},
		},
	})
	return nil
}

func (c *BackupSessionController) HandleBackupSetupFailure(ref *core.ObjectReference, setupErr error) error {
	// write log
	klog.Warningf("failed to start BackupSessionController. Reason: %v", setupErr)

	_, err := eventer.CreateEvent(
		c.K8sClient,
		eventer.EventSourceBackupSidecar,
		ref,
		core.EventTypeWarning,
		eventer.EventReasonFailedToStartBackupSessionController,
		fmt.Sprintf("failed to start BackupSession controller in pod %s/%s Reason: %v", meta.PodNamespace(), meta.PodName(), setupErr),
	)
	return errors.NewAggregate([]error{setupErr, err})
}

func (c *BackupSessionController) handleBackupSetupSuccess(invokerRef *core.ObjectReference) error {
	// write log
	klog.Infof("BackupSession controller started successfully.")

	// Write event to the invoker
	_, err := eventer.CreateEvent(
		c.K8sClient,
		eventer.EventSourceBackupSidecar,
		invokerRef,
		core.EventTypeNormal,
		eventer.EventReasonBackupSessionControllerStarted,
		fmt.Sprintf("BackupSession controller started successfully in pod %s/%s.", meta.PodNamespace(), meta.PodName()),
	)
	return err
}

func (c *BackupSessionController) handleBackupSuccess(backupSession *api_v1beta1.BackupSession, inv invoker.BackupInvoker, targetInfo invoker.BackupTargetInfo, backupOutput *restic.BackupOutput) error {
	klog.Infof("Backup completed successfully for BackupSession %s", backupSession.Name)
	return c.handleBackupCompletion(inv, targetInfo, backupSession, backupOutput)
}

func (c *BackupSessionController) handleBackupFailure(backupSession *api_v1beta1.BackupSession, inv invoker.BackupInvoker, targetInfo invoker.BackupTargetInfo, backupErr error) error {
	klog.Warningf("Failed to take backup for BackupSession %s. Reason: %v", backupSession.Name, backupErr)
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

	return c.handleBackupCompletion(inv, targetInfo, backupSession, backupOutput)
}

func (c *BackupSessionController) handleBackupCompletion(inv invoker.BackupInvoker, targetInfo invoker.BackupTargetInfo, backupSession *api_v1beta1.BackupSession, backupOutput *restic.BackupOutput) error {
	// execute hooks at the end of backup completion. no matter if the backup succeed or fail.
	defer func() {
		if targetInfo.Hooks != nil && targetInfo.Hooks.PostBackup != nil {
			hookErr := c.executePostBackupHook(inv, targetInfo, backupSession)
			if hookErr != nil {
				klog.Infof("failed to execute postBackup hook. Reason: ", hookErr)
			}
		}
	}()

	statusOpt := status.UpdateStatusOptions{
		Config:        c.Config,
		KubeClient:    c.K8sClient,
		StashClient:   c.StashClient,
		Namespace:     c.Namespace,
		BackupSession: backupSession.Name,
		Metrics:       c.Metrics,
		SetupOpt:      c.SetupOpt,
	}
	if targetInfo.Target != nil {
		statusOpt.TargetRef = targetInfo.Target.Ref
	}

	return statusOpt.UpdatePostBackupStatus(backupOutput)
}

func (c *BackupSessionController) isBackupTakenForThisHost(backupSession *api_v1beta1.BackupSession, backupTarget *api_v1beta1.BackupTarget) bool {
	// if overall backupSession phase is "Succeeded" or "Failed" or "Skipped" then it has been processed already
	if backupSession.Status.Phase == api_v1beta1.BackupSessionSucceeded ||
		backupSession.Status.Phase == api_v1beta1.BackupSessionFailed ||
		backupSession.Status.Phase == api_v1beta1.BackupSessionSkipped {
		return true
	}

	// if backupSession has entry for this host in status field, then it has been already processed for this host
	for _, target := range backupSession.Status.Targets {
		if backupTarget != nil &&
			backupTarget.Ref.Kind == c.TargetRef.Kind &&
			backupTarget.Ref.Namespace == c.TargetRef.Namespace &&
			backupTarget.Ref.Name == c.TargetRef.Name {
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
	if ref.Kind == c.TargetRef.Kind &&
		ref.Namespace == c.TargetRef.Namespace &&
		ref.Name == c.TargetRef.Name {
		return true
	}
	return false
}

func (c *BackupSessionController) setSetupOptions(repo kmapi.ObjectReference) (*util.ExtraOptions, error) {
	// get repository
	repository, err := c.StashClient.StashV1alpha1().Repositories(repo.Namespace).Get(context.TODO(), repo.Name, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}

	secret, err := c.K8sClient.CoreV1().Secrets(repository.Namespace).Get(context.TODO(), repository.Spec.Backend.StorageSecretName, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}

	// configure SourceHost, SecretDirectory, EnableCache and ScratchDirectory
	extraOpt := &util.ExtraOptions{
		Host:          c.Host,
		StorageSecret: secret,
		EnableCache:   c.SetupOpt.EnableCache,
		ScratchDir:    c.SetupOpt.ScratchDir,
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

func (c *BackupSessionController) executePreBackupHook(inv invoker.BackupInvoker, targetInfo invoker.BackupTargetInfo, backupSession *api_v1beta1.BackupSession) error {
	hookExecutor := stashHooks.BackupHookExecutor{
		Config:        c.Config,
		StashClient:   c.StashClient,
		BackupSession: backupSession,
		Invoker:       inv,
		Target:        targetInfo.Target.Ref,
		ExecutorPod: kmapi.ObjectReference{
			Namespace: c.Namespace,
			Name:      meta.PodName(),
		},
		Hook:     targetInfo.Hooks.PreBackup,
		HookType: apis.PreBackupHook,
	}
	return hookExecutor.Execute()
}

func (c *BackupSessionController) executePostBackupHook(inv invoker.BackupInvoker, targetInfo invoker.BackupTargetInfo, backupSession *api_v1beta1.BackupSession) error {
	hookExecutor := stashHooks.BackupHookExecutor{
		Config:        c.Config,
		StashClient:   c.StashClient,
		BackupSession: backupSession,
		Invoker:       inv,
		Target:        targetInfo.Target.Ref,
		ExecutorPod: kmapi.ObjectReference{
			Namespace: c.Namespace,
			Name:      meta.PodName(),
		},
		Hook:     targetInfo.Hooks.PostBackup,
		HookType: apis.PostBackupHook,
	}
	return hookExecutor.Execute()
}

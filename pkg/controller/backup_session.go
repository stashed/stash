package controller

import (
	"fmt"

	"github.com/appscode/go/log"
	"github.com/appscode/stash/apis"
	"github.com/appscode/stash/apis/stash"
	api "github.com/appscode/stash/apis/stash/v1beta1"
	stash_util "github.com/appscode/stash/client/clientset/versioned/typed/stash/v1beta1/util"
	"github.com/appscode/stash/pkg/eventer"
	"github.com/appscode/stash/pkg/resolve"
	"github.com/appscode/stash/pkg/util"
	"github.com/golang/glog"
	batchv1 "k8s.io/api/batch/v1"
	core "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/tools/cache"
	batch_util "kmodules.xyz/client-go/batch/v1"
	core_util "kmodules.xyz/client-go/core/v1"
	"kmodules.xyz/client-go/meta"
	"kmodules.xyz/client-go/tools/queue"
	"kmodules.xyz/webhook-runtime/admission"
	hooks "kmodules.xyz/webhook-runtime/admission/v1beta1"
	webhook "kmodules.xyz/webhook-runtime/admission/v1beta1/generic"
)

const (
	BackupJobPrefix                    = "stash-backup-"
	BackupSessionEventComponent        = "stash-backup-session"
	EventReasonInvalidBackupSession    = "InvalidBackupSession"
	EventReasonBackupSessionFailed     = "BackupSessionFailedToExecute"
	EventReasonBackupSessionJobCreated = "BackupSessionJobCreated"
)

func (c *StashController) NewBackupSessionWebhook() hooks.AdmissionHook {
	return webhook.NewGenericWebhook(
		schema.GroupVersionResource{
			Group:    "admission.stash.appscode.com",
			Version:  "v1beta1",
			Resource: api.ResourcePluralBackupSession,
		},
		api.ResourceSingularBackupSession,
		[]string{stash.GroupName},
		api.SchemeGroupVersion.WithKind(api.ResourceKindBackupSession),
		nil,
		&admission.ResourceHandlerFuncs{
			CreateFunc: func(obj runtime.Object) (runtime.Object, error) {
				return nil, obj.(*api.BackupSession).IsValid()
			},
			UpdateFunc: func(oldObj, newObj runtime.Object) (runtime.Object, error) {
				// should not allow spec update
				if !meta.Equal(oldObj.(*api.BackupSession).Spec, newObj.(*api.BackupSession).Spec) {
					return nil, fmt.Errorf("BackupSession spec is immutable")
				}
				return nil, nil
			},
		},
	)
}

// process only add events
func (c *StashController) initBackupSessionWatcher() {
	c.backupSessionInformer = c.stashInformerFactory.Stash().V1beta1().BackupSessions().Informer()
	c.backupSessionQueue = queue.New("BackupSession", c.MaxNumRequeues, c.NumThreads, c.runBackupSessionInjector)
	c.backupSessionInformer.AddEventHandler(&cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			if r, ok := obj.(*api.BackupSession); ok {
				if err := r.IsValid(); err != nil {
					eventer.CreateEvent(c.kubeClient, BackupSessionEventComponent, r, core.EventTypeWarning, EventReasonInvalidBackupSession, err.Error())
					return
				}
				queue.Enqueue(c.backupSessionQueue.GetQueue(), obj)
			}
		},
	})
	c.backupSessionLister = c.stashInformerFactory.Stash().V1beta1().BackupSessions().Lister()
}

func (c *StashController) runBackupSessionInjector(key string) error {
	obj, exists, err := c.backupSessionInformer.GetIndexer().GetByKey(key)
	if err != nil {
		glog.Errorf("Fetching object with key %s from store failed with %v", key, err)
		return err
	}
	if !exists {
		glog.Warningf("BackupSession %s does not exist anymore\n", key)
		return nil
	}
	backupSession := obj.(*api.BackupSession)
	glog.Infof("Sync/Add/Update for BackupSession %s", backupSession.GetName())

	// execute backupSession, update status and write event
	job, err := c.executeBackupSession(backupSession)
	if err != nil {
		log.Errorln(err)
		eventer.CreateEvent(c.kubeClient, BackupSessionEventComponent, backupSession, core.EventTypeWarning, EventReasonBackupSessionFailed, err.Error())
		stash_util.UpdateBackupSessionStatus(c.stashClient.StashV1beta1(), backupSession, func(in *api.BackupSessionStatus) *api.BackupSessionStatus {
			in.Phase = api.BackupSessionFailed
			return in
		}, apis.EnableStatusSubresource)
		return err
	}
	if job != nil { // job successfully created
		eventer.CreateEvent(c.kubeClient, BackupSessionEventComponent, backupSession, core.EventTypeNormal, EventReasonBackupSessionJobCreated, fmt.Sprintf("backup job %s created", job.Name))
		stash_util.UpdateBackupSessionStatus(c.stashClient.StashV1beta1(), backupSession, func(in *api.BackupSessionStatus) *api.BackupSessionStatus {
			in.Phase = api.BackupSessionRunning
			return in
		}, apis.EnableStatusSubresource)
	} // else it was skipped
	return nil
}

func (c *StashController) executeBackupSession(backupSession *api.BackupSession) (*batchv1.Job, error) {
	if backupSession.Status.Phase == api.BackupSessionSucceeded || backupSession.Status.Phase == api.BackupSessionRunning {
		return nil, nil
	}
	// get BackupConfiguration for BackupSession
	backupConfig, err := c.stashClient.StashV1beta1().BackupConfigurations(backupSession.Namespace).Get(
		backupSession.Spec.BackupConfiguration.Name,
		metav1.GetOptions{},
	)
	if err != nil {
		return nil, fmt.Errorf("can't get BackupConfiguration for BackupSession %s/%s, reason: %s", backupSession.Namespace, backupSession.Name, err)
	}
	// skip if target is a workload (i.e. deployment/daemonset/replicaset/statefulset)
	// target is nil for cluster backup
	if backupConfig.Spec.Target != nil && backupConfig.Spec.Target.Ref.IsWorkload() {
		log.Infof("Skipping BackupSession %s/%s, reason: target is a workload", backupSession.Namespace, backupSession.Name)
		return nil, nil
	}

	explicitInputs := make(map[string]string)
	for _, param := range backupConfig.Spec.Task.Params {
		explicitInputs[param.Name] = param.Value
	}

	implicitInputs, err := c.inputsForBackupConfig(*backupConfig)
	if err != nil {
		return nil, fmt.Errorf("cannot resolve implicit inputs for BackupConfiguration %s/%s, reason: %s", backupConfig.Namespace, backupConfig.Name, err)
	}

	taskResolver := resolve.TaskResolver{
		StashClient:     c.stashClient,
		TaskName:        backupConfig.Spec.Task.Name,
		Inputs:          core_util.UpsertMap(explicitInputs, implicitInputs), // TODO: reverse priority ???
		RuntimeSettings: backupConfig.Spec.RuntimeSettings,
	}
	podSpec, err := taskResolver.GetPodSpec()
	if err != nil {
		return nil, fmt.Errorf("can't get PodSpec for BackupConfiguration %s/%s, reason: %s", backupConfig.Namespace, backupConfig.Name, err)
	}

	job := &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:      BackupJobPrefix + backupSession.Name,
			Namespace: backupSession.Namespace,
			OwnerReferences: []metav1.OwnerReference{
				{
					APIVersion: api.SchemeGroupVersion.String(),
					Kind:       api.ResourceKindBackupSession,
					Name:       backupSession.Name,
					UID:        backupSession.UID,
				},
			},
			Labels: map[string]string{
				// job controller should not delete this job on completion
				// use a different label than v1alpha1 job labels to skip deletion from job controller
				// TODO: Remove job controller, cleanup backup-session periodically
				"app": util.AppLabelStashV1Beta1,
			},
		},
		Spec: batchv1.JobSpec{
			Template: core.PodTemplateSpec{
				Spec: podSpec,
			},
		},
	}
	job, _, err = batch_util.CreateOrPatchJob(c.kubeClient, job.ObjectMeta, func(_ *batchv1.Job) *batchv1.Job {
		return job
	})
	return job, err
}

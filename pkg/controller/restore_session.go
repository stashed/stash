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
	RestoreJobPrefix                    = "stash-restore-"
	RestoreSessionEventComponent        = "stash-restore-session"
	EventReasonInvalidRestoreSession    = "InvalidRestoreSession"
	EventReasonRestoreSessionFailed     = "RestoreSessionFailedToExecute"
	EventReasonRestoreSessionJobCreated = "RestoreSessionJobCreated"
)

func (c *StashController) NewRestoreSessionWebhook() hooks.AdmissionHook {
	return webhook.NewGenericWebhook(
		schema.GroupVersionResource{
			Group:    "admission.stash.appscode.com",
			Version:  "v1beta1",
			Resource: api.ResourcePluralRestoreSession,
		},
		api.ResourceSingularRestoreSession,
		[]string{stash.GroupName},
		api.SchemeGroupVersion.WithKind(api.ResourceKindRestoreSession),
		nil,
		&admission.ResourceHandlerFuncs{
			CreateFunc: func(obj runtime.Object) (runtime.Object, error) {
				return nil, obj.(*api.RestoreSession).IsValid()
			},
			UpdateFunc: func(oldObj, newObj runtime.Object) (runtime.Object, error) {
				// TODO: should not allow spec update ???
				if !meta.Equal(oldObj.(*api.RestoreSession).Spec, newObj.(*api.RestoreSession).Spec) {
					return nil, fmt.Errorf("RestoreSession spec is immutable")
				}
				return nil, nil
			},
		},
	)
}

// process only add events
func (c *StashController) initRestoreSessionWatcher() {
	c.restoreSessionInformer = c.stashInformerFactory.Stash().V1beta1().RestoreSessions().Informer()
	c.restoreSessionQueue = queue.New("RestoreSession", c.MaxNumRequeues, c.NumThreads, c.runRestoreSessionInjector)
	c.restoreSessionInformer.AddEventHandler(&cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			if r, ok := obj.(*api.RestoreSession); ok {
				if err := r.IsValid(); err != nil {
					eventer.CreateEvent(c.kubeClient, RestoreSessionEventComponent, r, core.EventTypeWarning, EventReasonInvalidRestoreSession, err.Error())
					return
				}
				queue.Enqueue(c.restoreSessionQueue.GetQueue(), obj)
			}
		},
	})
	c.restoreSessionLister = c.stashInformerFactory.Stash().V1beta1().RestoreSessions().Lister()
}

func (c *StashController) runRestoreSessionInjector(key string) error {
	obj, exists, err := c.restoreSessionInformer.GetIndexer().GetByKey(key)
	if err != nil {
		glog.Errorf("Fetching object with key %s from store failed with %v", key, err)
		return err
	}
	if !exists {
		glog.Warningf("RestoreSession %s does not exist anymore\n", key)
		return nil
	}
	restoreSession := obj.(*api.RestoreSession)
	glog.Infof("Sync/Add/Update for RestoreSession %s", restoreSession.GetName())

	// execute restoreSession, update status and write event
	job, err := c.executeRestoreSession(restoreSession)
	if err != nil {
		log.Errorln(err)
		eventer.CreateEvent(c.kubeClient, RestoreSessionEventComponent, restoreSession, core.EventTypeWarning, EventReasonRestoreSessionFailed, err.Error())
		stash_util.UpdateRestoreSessionStatus(c.stashClient.StashV1beta1(), restoreSession, func(in *api.RestoreSessionStatus) *api.RestoreSessionStatus {
			in.Phase = api.RestoreFailed
			return in
		}, apis.EnableStatusSubresource)
		return err
	}
	if job != nil { // job successfully created
		eventer.CreateEvent(c.kubeClient, RestoreSessionEventComponent, restoreSession, core.EventTypeNormal, EventReasonRestoreSessionJobCreated, fmt.Sprintf("restore job %s created", job.Name))
		stash_util.UpdateRestoreSessionStatus(c.stashClient.StashV1beta1(), restoreSession, func(in *api.RestoreSessionStatus) *api.RestoreSessionStatus {
			in.Phase = api.RestoreRunning
			return in
		}, apis.EnableStatusSubresource)
	} // else it was skipped due to empty/workload target
	return nil
}

func (c *StashController) executeRestoreSession(restoreSession *api.RestoreSession) (*batchv1.Job, error) {
	if restoreSession.Status.Phase == api.RestoreSucceeded || restoreSession.Status.Phase == api.RestoreRunning {
		return nil, nil
	}
	// skip if target is a workload (i.e. deployment/daemonset/replicaset/statefulset)
	// target is nil for cluster backup
	if restoreSession.Spec.Target != nil && restoreSession.Spec.Target.Ref.IsWorkload() {
		log.Infof("Skipping RestoreSession %s/%s, reason: target is a workload", restoreSession.Namespace, restoreSession.Name)
		return nil, nil
	}

	explicitInputs := make(map[string]string)
	for _, param := range restoreSession.Spec.Task.Params {
		explicitInputs[param.Name] = param.Value
	}

	implicitInputs, err := c.inputsForRestoreSession(*restoreSession)
	if err != nil {
		return nil, fmt.Errorf("cannot resolve implicit inputs for RestoreSession %s/%s, reason: %s", restoreSession.Namespace, restoreSession.Name, err)
	}
	implicitInputs[apis.Namespace] = restoreSession.Namespace
	implicitInputs[apis.RestoreSession] = restoreSession.Name

	taskResolver := resolve.TaskResolver{
		StashClient:     c.stashClient,
		TaskName:        restoreSession.Spec.Task.Name,
		Inputs:          core_util.UpsertMap(explicitInputs, implicitInputs),
		RuntimeSettings: restoreSession.Spec.RuntimeSettings,
	}
	podSpec, err := taskResolver.GetPodSpec()
	if err != nil {
		return nil, fmt.Errorf("can't get PodSpec for RestoreSession %s/%s, reason: %s", restoreSession.Namespace, restoreSession.Name, err)
	}

	job := &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:      RestoreJobPrefix + restoreSession.Name,
			Namespace: restoreSession.Namespace,
			OwnerReferences: []metav1.OwnerReference{
				{
					APIVersion: api.SchemeGroupVersion.String(),
					Kind:       api.ResourceKindRestoreSession,
					Name:       restoreSession.Name,
					UID:        restoreSession.UID,
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

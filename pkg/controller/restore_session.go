package controller

import (
	"fmt"

	"github.com/appscode/go/log"
	"github.com/appscode/stash/apis"
	"github.com/appscode/stash/apis/stash"
	api_v1beta1 "github.com/appscode/stash/apis/stash/v1beta1"
	stash_scheme "github.com/appscode/stash/client/clientset/versioned/scheme"
	v1beta1_util "github.com/appscode/stash/client/clientset/versioned/typed/stash/v1beta1/util"
	"github.com/appscode/stash/pkg/eventer"
	"github.com/appscode/stash/pkg/resolve"
	"github.com/appscode/stash/pkg/util"
	"github.com/golang/glog"
	batchv1 "k8s.io/api/batch/v1"
	core "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/tools/reference"
	batch_util "kmodules.xyz/client-go/batch/v1"
	core_util "kmodules.xyz/client-go/core/v1"
	"kmodules.xyz/client-go/meta"
	"kmodules.xyz/client-go/tools/queue"
	"kmodules.xyz/webhook-runtime/admission"
	hooks "kmodules.xyz/webhook-runtime/admission/v1beta1"
	webhook "kmodules.xyz/webhook-runtime/admission/v1beta1/generic"
)

const (
	RestoreJobPrefix = "stash-restore-"
)

func (c *StashController) NewRestoreSessionWebhook() hooks.AdmissionHook {
	return webhook.NewGenericWebhook(
		schema.GroupVersionResource{
			Group:    "admission.stash.appscode.com",
			Version:  "v1beta1",
			Resource: api_v1beta1.ResourcePluralRestoreSession,
		},
		api_v1beta1.ResourceSingularRestoreSession,
		[]string{stash.GroupName},
		api_v1beta1.SchemeGroupVersion.WithKind(api_v1beta1.ResourceKindRestoreSession),
		nil,
		&admission.ResourceHandlerFuncs{
			CreateFunc: func(obj runtime.Object) (runtime.Object, error) {
				return nil, obj.(*api_v1beta1.RestoreSession).IsValid()
			},
			UpdateFunc: func(oldObj, newObj runtime.Object) (runtime.Object, error) {
				// TODO: should not allow spec update ???
				if !meta.Equal(oldObj.(*api_v1beta1.RestoreSession).Spec, newObj.(*api_v1beta1.RestoreSession).Spec) {
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
	c.restoreSessionQueue = queue.New(api_v1beta1.ResourceKindRestoreSession, c.MaxNumRequeues, c.NumThreads, c.runRestoreSessionProcessor)
	c.restoreSessionInformer.AddEventHandler(queue.NewObservableHandler(c.restoreSessionQueue.GetQueue(), apis.EnableStatusSubresource))
	c.restoreSessionLister = c.stashInformerFactory.Stash().V1beta1().RestoreSessions().Lister()
}

func (c *StashController) runRestoreSessionProcessor(key string) error {
	obj, exists, err := c.restoreSessionInformer.GetIndexer().GetByKey(key)
	if err != nil {
		glog.Errorf("Fetching object with key %s from store failed with %v", key, err)
		return err
	}
	if !exists {
		glog.Warningf("RestoreSession %s does not exist anymore\n", key)
		return nil
	} else {
		restoreSession := obj.(*api_v1beta1.RestoreSession)
		glog.Infof("Sync/Add/Update for RestoreSession %s", restoreSession.GetName())

		// if RestoreSession is being deleted then remove respective init-container
		if restoreSession.DeletionTimestamp != nil {

			// if RestoreSession has stash finalizer then respective init-container (for workloads) hasn't been removed
			// remove respective init-container and finally remove finalizer
			if core_util.HasFinalizer(restoreSession.ObjectMeta, api_v1beta1.StashKey) {
				if restoreSession.Spec.Target != nil && util.BackupModel(restoreSession.Spec.Target.Ref.Kind) == util.ModelSidecar {

					// send event to workload controller. workload controller will take care of removing restore init-container
					err := c.sendEventToWorkloadQueue(
						restoreSession.Spec.Target.Ref.Kind,
						restoreSession.Namespace,
						restoreSession.Spec.Target.Ref.Name,
					)
					if err != nil {
						log.Errorln(err)
						return err
					}
				}

				// remove finalizer
				_, _, err = v1beta1_util.PatchRestoreSession(c.stashClient.StashV1beta1(), restoreSession, func(in *api_v1beta1.RestoreSession) *api_v1beta1.RestoreSession {
					in.ObjectMeta = core_util.RemoveFinalizer(in.ObjectMeta, api_v1beta1.StashKey)
					return in
				})
				if err != nil {
					log.Errorln(err)
					return err
				}
			}
		} else {
			// add finalizer
			_, _, err = v1beta1_util.PatchRestoreSession(c.stashClient.StashV1beta1(), restoreSession, func(in *api_v1beta1.RestoreSession) *api_v1beta1.RestoreSession {
				in.ObjectMeta = core_util.AddFinalizer(in.ObjectMeta, api_v1beta1.StashKey)
				return in
			})
			if err != nil {
				return err
			}

			// don't process further if RestoreSession has been processed already
			if !util.RestorePending(restoreSession.Status.Phase) {
				log.Infoln("No pending RestoreSession. Skipping creating new restore job.")
				return nil
			}

			// if target is kubernetes workload i.e. Deployment, StatefulSet etc. then inject restore init-container
			if restoreSession.Spec.Target != nil && util.BackupModel(restoreSession.Spec.Target.Ref.Kind) == util.ModelSidecar {
				// send event to workload controller. workload controller will take care of injecting restore init-container
				err := c.sendEventToWorkloadQueue(
					restoreSession.Spec.Target.Ref.Kind,
					restoreSession.Namespace,
					restoreSession.Spec.Target.Ref.Name,
				)
				if err != nil {
					return err
				}
			} else {

				// target is not a workload. we have to restore by a job. create restore job.
				err := c.ensureRestoreJob(restoreSession)
				if err != nil {
					return c.setRestoreSessionFailed(restoreSession, err)
				}

				// restore job has been created successfully. set RestoreSession phase to "Running"
				err = c.setRestoreSessionRunning(restoreSession)
				if err != nil {
					return err
				}
			}
		}

	}
	return nil
}

func (c *StashController) ensureRestoreJob(restoreSession *api_v1beta1.RestoreSession) error {
	objectMeta := metav1.ObjectMeta{
		Name:      RestoreJobPrefix + restoreSession.Name,
		Namespace: restoreSession.Namespace,
	}

	ref, err := reference.GetReference(stash_scheme.Scheme, restoreSession)
	if err != nil {
		return err
	}

	// if RBAC is enabled then ensure respective ClusterRole,RoleBinding,ServiceAccount etc.
	serviceAccountName := "default"
	if c.EnableRBAC {
		if restoreSession.Spec.RuntimeSettings.Pod != nil &&
			restoreSession.Spec.RuntimeSettings.Pod.ServiceAccountName != "" {
			// ServiceAccount has been specified, so use it.
			serviceAccountName = restoreSession.Spec.RuntimeSettings.Pod.ServiceAccountName
		} else {
			// ServiceAccount hasn't been specified. so create new one with same name as RestoreSession object.
			serviceAccountName = objectMeta.Name

			_, _, err := core_util.CreateOrPatchServiceAccount(c.kubeClient, objectMeta, func(in *core.ServiceAccount) *core.ServiceAccount {
				core_util.EnsureOwnerReference(&in.ObjectMeta, ref)
				if in.Labels == nil {
					in.Labels = map[string]string{}
				}
				in.Labels[util.LabelApp] = util.AppLabelStash
				return in
			})
			if err != nil {
				return err
			}
		}

		err := c.ensureRestoreJobRBAC(ref, serviceAccountName)
		if err != nil {
			return err
		}
	}

	// resolve task template
	explicitInputs := make(map[string]string)
	for _, param := range restoreSession.Spec.Task.Params {
		explicitInputs[param.Name] = param.Value
	}

	implicitInputs, err := c.inputsForRestoreSession(*restoreSession)
	if err != nil {
		return err
	}
	implicitInputs[apis.Namespace] = restoreSession.Namespace
	implicitInputs[apis.RestoreSession] = restoreSession.Name
	implicitInputs[apis.StatusSubresourceEnabled] = fmt.Sprint(apis.EnableStatusSubresource)

	taskResolver := resolve.TaskResolver{
		StashClient:     c.stashClient,
		TaskName:        restoreSession.Spec.Task.Name,
		Inputs:          core_util.UpsertMap(explicitInputs, implicitInputs),
		RuntimeSettings: restoreSession.Spec.RuntimeSettings,
	}
	podSpec, err := taskResolver.GetPodSpec()
	if err != nil {
		return err
	}

	// create Restore Job
	_, _, err = batch_util.CreateOrPatchJob(c.kubeClient, objectMeta, func(in *batchv1.Job) *batchv1.Job {
		// set RestoreSession as owner of this Job
		core_util.EnsureOwnerReference(&in.ObjectMeta, ref)
		in.Labels = map[string]string{
			// job controller should not delete this job on completion
			// use a different label than v1alpha1 job labels to skip deletion from job controller
			// TODO: Remove job controller, cleanup backup-session periodically
			util.LabelApp: util.AppLabelStash,
		}
		in.Spec.Template.Spec = podSpec
		if c.EnableRBAC {
			in.Spec.Template.Spec.ServiceAccountName = serviceAccountName
		}
		return in
	})

	return err
}

func (c *StashController) setRestoreSessionFailed(restoreSession *api_v1beta1.RestoreSession, jobErr error) error {

	// set RestoreSession phase to "Failed"
	_, err := v1beta1_util.UpdateRestoreSessionStatus(c.stashClient.StashV1beta1(), restoreSession, func(in *api_v1beta1.RestoreSessionStatus) *api_v1beta1.RestoreSessionStatus {
		in.Phase = api_v1beta1.RestoreFailed
		return in
	}, apis.EnableStatusSubresource)
	if err != nil {
		return err
	}

	// write failure event
	_, err = eventer.CreateEvent(
		c.kubeClient,
		eventer.RestoreSessionEventComponent,
		restoreSession,
		core.EventTypeWarning,
		eventer.EventReasonRestoreSessionFailed,
		jobErr.Error(),
	)

	return err
}

func (c *StashController) setRestoreSessionRunning(restoreSession *api_v1beta1.RestoreSession) error {

	// set RestoreSession phase to "Running"
	_, err := v1beta1_util.UpdateRestoreSessionStatus(c.stashClient.StashV1beta1(), restoreSession, func(in *api_v1beta1.RestoreSessionStatus) *api_v1beta1.RestoreSessionStatus {
		in.Phase = api_v1beta1.RestoreRunning
		return in
	}, apis.EnableStatusSubresource)
	if err != nil {
		return err
	}

	// write job creation success event
	_, err = eventer.CreateEvent(
		c.kubeClient,
		eventer.RestoreSessionEventComponent,
		restoreSession,
		core.EventTypeNormal,
		eventer.EventReasonRestoreJobCreated,
		fmt.Sprintf("restore job has been created succesfully for RestoreSession %s/%s", restoreSession.Namespace, restoreSession.Name),
	)

	return err
}

package controller

import (
	"fmt"
	"time"

	"github.com/appscode/go/log"
	"github.com/appscode/stash/apis"
	"github.com/appscode/stash/apis/stash"
	api_v1beta1 "github.com/appscode/stash/apis/stash/v1beta1"
	stash_scheme "github.com/appscode/stash/client/clientset/versioned/scheme"
	stash_util "github.com/appscode/stash/client/clientset/versioned/typed/stash/v1beta1/util"
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
			Resource: "restoresessionvalidators",
		},
		"restoresessionvalidator",
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

			if restoreSession.Status.Phase == api_v1beta1.RestoreSessionFailed ||
				restoreSession.Status.Phase == api_v1beta1.RestoreSessionSucceeded {
				log.Infof("Skipping processing RestoreSession %s/%s. Reason: phase is %q.", restoreSession.Namespace, restoreSession.Name, restoreSession.Status.Phase)
				return nil
			}
			// check weather restore session is completed or running and set it's phase accordingly
			phase, err := c.getRestoreSessionPhase(restoreSession)

			if err != nil || phase == api_v1beta1.RestoreSessionFailed {
				return c.setRestoreSessionFailed(restoreSession, err)
			} else if phase == api_v1beta1.RestoreSessionSucceeded {
				return c.setRestoreSessionSucceeded(restoreSession)
			} else if phase == api_v1beta1.RestoreSessionRunning {
				log.Infof("Skipping processing RestoreSession %s/%s. Reason: phase is %q.", restoreSession.Namespace, restoreSession.Name, restoreSession.Status.Phase)
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
				return c.setRestoreSessionRunning(restoreSession)
			} else {

				// target is not a workload. we have to restore by a job. create restore job.
				err := c.ensureRestoreJob(restoreSession)
				if err != nil {
					return c.setRestoreSessionFailed(restoreSession, err)
				}

				// restore job has been created successfully. set RestoreSession phase to "Running"
				return c.setRestoreSessionRunning(restoreSession)
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

	// get repository for backupConfig
	repository, err := c.stashClient.StashV1alpha1().Repositories(restoreSession.Namespace).Get(
		restoreSession.Spec.Repository.Name,
		metav1.GetOptions{},
	)
	if err != nil {
		return err
	}

	// resolve task template

	explicitInputs := make(map[string]string)
	for _, param := range restoreSession.Spec.Task.Params {
		explicitInputs[param.Name] = param.Value
	}

	repoInputs, err := c.inputsForRepository(repository)
	if err != nil {
		return fmt.Errorf("cannot resolve implicit inputs for Repository %s/%s, reason: %s", repository.Namespace, repository.Name, err)
	}
	rsInputs, err := c.inputsForRestoreSession(*restoreSession)
	if err != nil {
		return fmt.Errorf("cannot resolve implicit inputs for RestoreSession %s/%s, reason: %s", restoreSession.Namespace, restoreSession.Name, err)
	}

	implicitInputs := core_util.UpsertMap(repoInputs, rsInputs)
	implicitInputs[apis.Namespace] = restoreSession.Namespace
	implicitInputs[apis.RestoreSession] = restoreSession.Name
	implicitInputs[apis.StatusSubresourceEnabled] = fmt.Sprint(apis.EnableStatusSubresource)

	taskResolver := resolve.TaskResolver{
		StashClient:     c.stashClient,
		TaskName:        restoreSession.Spec.Task.Name,
		Inputs:          core_util.UpsertMap(explicitInputs, implicitInputs),
		RuntimeSettings: restoreSession.Spec.RuntimeSettings,
		TempDir:         restoreSession.Spec.TempDir,
	}
	podSpec, err := taskResolver.GetPodSpec()
	if err != nil {
		return err
	}
	// for local backend, attach volume to all containers
	if repository.Spec.Backend.Local != nil {
		podSpec = util.AttachLocalBackend(podSpec, *repository.Spec.Backend.Local)
	}

	// create Restore Job
	_, _, err = batch_util.CreateOrPatchJob(c.kubeClient, objectMeta, func(in *batchv1.Job) *batchv1.Job {
		// set RestoreSession as owner of this Job
		core_util.EnsureOwnerReference(&in.ObjectMeta, ref)
		in.Labels = map[string]string{
			// job controller should not delete this job on completion
			// use a different label than v1alpha1 job labels to skip deletion from job controller
			// TODO: Remove job controller, cleanup restore-session periodically
			util.LabelApp: util.AppLabelStashV1Beta1,
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
		in.Phase = api_v1beta1.RestoreSessionFailed
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

	totalHosts, err := c.getTotalHosts(restoreSession.Spec.Target, restoreSession.Namespace)
	if err != nil {
		return err
	}

	// set RestoreSession phase to "Running"
	_, err = v1beta1_util.UpdateRestoreSessionStatus(c.stashClient.StashV1beta1(), restoreSession, func(in *api_v1beta1.RestoreSessionStatus) *api_v1beta1.RestoreSessionStatus {
		in.Phase = api_v1beta1.RestoreSessionRunning
		in.TotalHosts = totalHosts
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

func (c *StashController) setRestoreSessionSucceeded(restoreSession *api_v1beta1.RestoreSession) error {

	// total restore session duration is sum of individual host restore duration
	var sessionDuration time.Duration
	for _, hostStats := range restoreSession.Status.Stats {
		hostRestoreDuration, err := time.ParseDuration(hostStats.Duration)
		if err != nil {
			return err
		}
		sessionDuration = sessionDuration + hostRestoreDuration
	}

	// update RestoreSession status
	_, err := stash_util.UpdateRestoreSessionStatus(c.stashClient.StashV1beta1(), restoreSession, func(in *api_v1beta1.RestoreSessionStatus) *api_v1beta1.RestoreSessionStatus {
		in.Phase = api_v1beta1.RestoreSessionSucceeded
		in.SessionDuration = sessionDuration.String()
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
		eventer.EventReasonRestoreSessionSucceeded,
		fmt.Sprintf("restore has been completed succesfully for RestoreSession %s/%s", restoreSession.Namespace, restoreSession.Name),
	)

	return err
}

func (c *StashController) getRestoreSessionPhase(restoreSession *api_v1beta1.RestoreSession) (api_v1beta1.RestoreSessionPhase, error) {
	// RestoreSession phase is empty or "Pending" then return it. controller will process accordingly
	if restoreSession.Status.TotalHosts == nil ||
		restoreSession.Status.Phase == "" ||
		restoreSession.Status.Phase == api_v1beta1.RestoreSessionPending {
		return api_v1beta1.RestoreSessionPending, nil
	}

	// all hosts hasn't completed it's restore process. RestoreSession phase must be "Running".
	if *restoreSession.Status.TotalHosts != int32(len(restoreSession.Status.Stats)) {
		return api_v1beta1.RestoreSessionRunning, nil
	}

	// check if any of the host has failed to restore. if any of them has failed, then consider entire restore session as a failure.
	for _, host := range restoreSession.Status.Stats {
		if host.Phase == api_v1beta1.HostRestoreFailed {
			return api_v1beta1.RestoreSessionFailed, fmt.Errorf("restore failed for host: %s. Reason: %s", host.Hostname, host.Error)
		}
	}

	// restore has been completed successfully
	return api_v1beta1.RestoreSessionSucceeded, nil
}

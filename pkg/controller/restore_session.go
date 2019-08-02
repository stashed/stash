package controller

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/appscode/go/log"
	"github.com/appscode/go/types"
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
	ofst "kmodules.xyz/offshoot-api/api/v1"
	"kmodules.xyz/webhook-runtime/admission"
	hooks "kmodules.xyz/webhook-runtime/admission/v1beta1"
	webhook "kmodules.xyz/webhook-runtime/admission/v1beta1/generic"
	"stash.appscode.dev/stash/apis"
	"stash.appscode.dev/stash/apis/stash"
	api_v1alpha1 "stash.appscode.dev/stash/apis/stash/v1alpha1"
	api_v1beta1 "stash.appscode.dev/stash/apis/stash/v1beta1"
	stash_scheme "stash.appscode.dev/stash/client/clientset/versioned/scheme"
	stash_util "stash.appscode.dev/stash/client/clientset/versioned/typed/stash/v1beta1/util"
	v1beta1_util "stash.appscode.dev/stash/client/clientset/versioned/typed/stash/v1beta1/util"
	"stash.appscode.dev/stash/pkg/docker"
	"stash.appscode.dev/stash/pkg/eventer"
	stash_rbac "stash.appscode.dev/stash/pkg/rbac"
	"stash.appscode.dev/stash/pkg/resolve"
	"stash.appscode.dev/stash/pkg/util"
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
				restoreSession.Status.Phase == api_v1beta1.RestoreSessionSucceeded ||
				restoreSession.Status.Phase == api_v1beta1.RestoreSessionUnknown {
				log.Infof("Skipping processing RestoreSession %s/%s. Reason: phase is %q.", restoreSession.Namespace, restoreSession.Name, restoreSession.Status.Phase)
				return nil
			}
			// check whether restore session is completed or running and set it's phase accordingly
			phase, err := c.getRestoreSessionPhase(restoreSession)

			if phase == api_v1beta1.RestoreSessionFailed {
				return c.setRestoreSessionFailed(restoreSession, err)
			} else if phase == api_v1beta1.RestoreSessionUnknown {
				return c.setRestoreSessionUnknown(restoreSession, err)
			} else if phase == api_v1beta1.RestoreSessionSucceeded {
				return c.setRestoreSessionSucceeded(restoreSession)
			} else if phase == api_v1beta1.RestoreSessionRunning {
				log.Infof("Skipping processing RestoreSession %s/%s. Reason: phase is %q.", restoreSession.Namespace, restoreSession.Name, restoreSession.Status.Phase)
				return nil
			}

			if restoreSession.Spec.Target != nil && restoreSession.Spec.Driver == api_v1beta1.VolumeSnapshotter {
				err := c.setRestoreSessionRunning(restoreSession)
				if err != nil {
					return err
				}
				return c.ensureVolumeRestorerJob(restoreSession)
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

				// target is not a workload. we have to restore by a job.
				err := c.ensureRestoreJob(restoreSession)
				if err != nil {
					log.Warningln("failed to ensure restore job. Reason: ", err)
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
	image := docker.Docker{
		Registry: c.DockerRegistry,
		Image:    docker.ImageStash,
		Tag:      c.StashImageTag,
	}

	offshootLabels := restoreSession.OffshootLabels()

	jobMeta := metav1.ObjectMeta{
		Name:      getRestoreJobName(restoreSession),
		Namespace: restoreSession.Namespace,
		Labels:    offshootLabels,
	}

	ref, err := reference.GetReference(stash_scheme.Scheme, restoreSession)
	if err != nil {
		return err
	}

	// Ensure respective RBAC and PSP stuff.
	serviceAccountName := "default"
	if restoreSession.Spec.RuntimeSettings.Pod != nil &&
		restoreSession.Spec.RuntimeSettings.Pod.ServiceAccountName != "" {
		// ServiceAccount has been specified, so use it.
		serviceAccountName = restoreSession.Spec.RuntimeSettings.Pod.ServiceAccountName
	} else {
		// ServiceAccount hasn't been specified. so create new one with same name as RestoreSession object.
		serviceAccountName = jobMeta.Name

		_, _, err = core_util.CreateOrPatchServiceAccount(c.kubeClient, jobMeta, func(in *core.ServiceAccount) *core.ServiceAccount {
			core_util.EnsureOwnerReference(&in.ObjectMeta, ref)
			in.Labels = offshootLabels
			return in
		})
		if err != nil {
			return err
		}
	}

	psps, err := c.getRestoreJobPSPNames(restoreSession)
	if err != nil {
		return err
	}

	err = stash_rbac.EnsureRestoreJobRBAC(c.kubeClient, ref, serviceAccountName, psps, offshootLabels)
	if err != nil {
		return err
	}

	// get repository for RestoreSession
	repository, err := c.stashClient.StashV1alpha1().Repositories(restoreSession.Namespace).Get(
		restoreSession.Spec.Repository.Name,
		metav1.GetOptions{},
	)
	if err != nil {
		return err
	}

	// Now, there could be two restore scenario for restoring through job.
	// 1. Restore process follows Function-Task model. In this case, we have to resolve respective Functions and Task to get desired job definition.
	// 2. Restore process does not follow Function-Task model. In this case, we have to generate simple volume restorer job definition.

	var jobTemplate *core.PodTemplateSpec

	if restoreSession.Spec.Task.Name != "" {
		// Restore process follows Function-Task model. So, resolve Function and Task to get desired job definition.
		jobTemplate, err = c.resolveRestoreTask(restoreSession, repository, ref, serviceAccountName, "host-0")
		if err != nil {
			return err
		}
	} else {
		// Restore process does not follow Function-Task model. So, generate simple volume restorer job definition.
		jobTemplate, err = util.NewPVCRestorerJob(restoreSession, repository, image)
		if err != nil {
			return err
		}
	}

	// If volumeClaimTemplate is not specified then we don't need any further processing. Just, create the job
	if restoreSession.Spec.Target == nil ||
		(restoreSession.Spec.Target != nil && len(restoreSession.Spec.Target.VolumeClaimTemplates) == 0) {
		return c.createRestoreJob(jobTemplate, jobMeta, ref, serviceAccountName)
	}

	// volumeClaimTemplate has been specified. Now, we have to do the following for each replica:
	// 1. Create PVCs according to the template.
	// 2. Mount the PVCs to the restore job.
	// 3. Create the restore job to restore the target.

	replicas := int32(1) // set default replicas to 1
	if restoreSession.Spec.Target.Replicas != nil {
		replicas = *restoreSession.Spec.Target.Replicas
	}

	for ordinal := int32(0); ordinal < replicas; ordinal++ {
		// resolve template part of the volumeClaimTemplate and generate PVC definition according to the template
		pvcList, err := getPVCFromVolumeClaimTemplates(ordinal, restoreSession.Spec.Target.VolumeClaimTemplates)
		if err != nil {
			return err
		}

		// create the PVCs
		err = util.CreateBatchPVC(c.kubeClient, restoreSession.Namespace, pvcList)
		if err != nil {
			return err
		}

		// add ordinal suffix to the job name so that multiple restore job can run concurrently
		restoreJobMeta := jobMeta.DeepCopy()
		restoreJobMeta.Name = fmt.Sprintf("%s-%d", jobMeta.Name, ordinal)

		var restoreJobTemplate *core.PodTemplateSpec

		// if restore process follows Function-Task model, then resolve the Functions and Task  for this host
		if restoreSession.Spec.Task.Name != "" {
			restoreJobTemplate, err = c.resolveRestoreTask(restoreSession, repository, ref,
				serviceAccountName, fmt.Sprintf("host-%d", ordinal))

			if err != nil {
				return err
			}
		} else {
			// use copy of the original job template. otherwise, each iteration will append volumes in the same template
			restoreJobTemplate = jobTemplate.DeepCopy()
		}

		// mount the newly created PVCs into the job
		volumes := util.PVCListToVolumes(pvcList, ordinal)
		restoreJobTemplate.Spec = util.AttachPVC(restoreJobTemplate.Spec, volumes, restoreSession.Spec.Target.VolumeMounts)

		ordinalEnv := core.EnvVar{
			Name:  util.KeyPodOrdinal,
			Value: fmt.Sprintf("%d", ordinal),
		}

		// insert POD_ORDINAL env in all init-containers.
		for i, c := range restoreJobTemplate.Spec.InitContainers {
			restoreJobTemplate.Spec.InitContainers[i].Env = core_util.UpsertEnvVars(c.Env, ordinalEnv)
		}

		// insert POD_ORDINAL env in all containers.
		for i, c := range restoreJobTemplate.Spec.Containers {
			restoreJobTemplate.Spec.Containers[i].Env = core_util.UpsertEnvVars(c.Env, ordinalEnv)
		}

		// create restore job
		err = c.createRestoreJob(restoreJobTemplate, *restoreJobMeta, ref, serviceAccountName)
		if err != nil {
			return err
		}
	}
	return nil
}

func (c *StashController) createRestoreJob(jobTemplate *core.PodTemplateSpec, meta metav1.ObjectMeta, ref *core.ObjectReference, serviceAccountName string) error {
	_, _, err := batch_util.CreateOrPatchJob(c.kubeClient, meta, func(in *batchv1.Job) *batchv1.Job {
		// set BackupSession as owner of this Job
		core_util.EnsureOwnerReference(&in.ObjectMeta, ref)

		if in.Labels == nil {
			in.Labels = make(map[string]string, 0)
		}
		// ensure that job gets deleted when complete
		in.Labels[apis.KeyDeleteJobOnCompletion] = "false"

		in.Spec.Template = *jobTemplate
		in.Spec.Template.Spec.ServiceAccountName = serviceAccountName
		return in
	})
	return err
}

// resolveRestoreTask resolves Functions and Tasks then returns a job definition to restore the target.
func (c *StashController) resolveRestoreTask(restoreSession *api_v1beta1.RestoreSession,
	repository *api_v1alpha1.Repository, ref *core.ObjectReference, serviceAccountName, hostname string) (*core.PodTemplateSpec, error) {

	// resolve task template
	explicitInputs := make(map[string]string)
	for _, param := range restoreSession.Spec.Task.Params {
		explicitInputs[param.Name] = param.Value
	}

	repoInputs, err := c.inputsForRepository(repository)
	if err != nil {
		return nil, fmt.Errorf("cannot resolve implicit inputs for Repository %s/%s, reason: %s", repository.Namespace, repository.Name, err)
	}
	rsInputs, err := c.inputsForRestoreSession(*restoreSession, hostname)
	if err != nil {
		return nil, fmt.Errorf("cannot resolve implicit inputs for RestoreSession %s/%s, reason: %s", restoreSession.Namespace, restoreSession.Name, err)
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

	// In order to preserve file ownership, restore process need to be run as root user.
	// Stash image uses non-root user "stash"(1005). We have to use securityContext to run stash as root user.
	// If a user specify securityContext either in pod level or container level in RuntimeSetting,
	// don't overwrite that. In this case, user must take the responsibility of possible file ownership modification.
	defaultSecurityContext := &core.PodSecurityContext{
		RunAsUser:  types.Int64P(0),
		RunAsGroup: types.Int64P(0),
	}

	if taskResolver.RuntimeSettings.Pod == nil {
		taskResolver.RuntimeSettings.Pod = &ofst.PodRuntimeSettings{}
	}
	taskResolver.RuntimeSettings.Pod.SecurityContext = util.UpsertPodSecurityContext(defaultSecurityContext, taskResolver.RuntimeSettings.Pod.SecurityContext)

	podSpec, err := taskResolver.GetPodSpec()
	if err != nil {
		return nil, err
	}

	// for local backend, attach volume to all containers
	if repository.Spec.Backend.Local != nil {
		podSpec = util.AttachLocalBackend(podSpec, *repository.Spec.Backend.Local)
	}

	podTemplate := &core.PodTemplateSpec{
		Spec: podSpec,
	}
	return podTemplate, nil
}

func (c *StashController) ensureVolumeRestorerJob(restoreSession *api_v1beta1.RestoreSession) error {
	offshootLabels := restoreSession.OffshootLabels()

	jobMeta := metav1.ObjectMeta{
		Name:      getVolumeRestorerJobName(restoreSession),
		Namespace: restoreSession.Namespace,
		Labels:    offshootLabels,
	}

	ref, err := reference.GetReference(stash_scheme.Scheme, restoreSession)
	if err != nil {
		return err
	}

	serviceAccountName := "default"
	//ensure respective RBAC stuffs
	//Create new ServiceAccount
	serviceAccountName = restoreSession.Name
	saMeta := metav1.ObjectMeta{
		Name:      serviceAccountName,
		Namespace: restoreSession.Namespace,
	}
	_, _, err = core_util.CreateOrPatchServiceAccount(c.kubeClient, saMeta, func(in *core.ServiceAccount) *core.ServiceAccount {
		core_util.EnsureOwnerReference(&in.ObjectMeta, ref)
		in.Labels = offshootLabels
		return in
	})
	if err != nil {
		return err
	}

	err = stash_rbac.EnsureVolumeSnapshotRestorerJobRBAC(c.kubeClient, ref, serviceAccountName, offshootLabels)
	if err != nil {
		return err
	}

	image := docker.Docker{
		Registry: c.DockerRegistry,
		Image:    docker.ImageStash,
		Tag:      c.StashImageTag,
	}

	jobTemplate, err := util.NewVolumeRestorerJob(restoreSession, image)
	if err != nil {
		return err
	}

	// Create Volume restorer Job
	_, _, err = batch_util.CreateOrPatchJob(c.kubeClient, jobMeta, func(in *batchv1.Job) *batchv1.Job {
		// set BackupSession as owner of this Job
		core_util.EnsureOwnerReference(&in.ObjectMeta, ref)

		in.Labels = offshootLabels
		// ensure that job gets deleted when complete
		in.Labels[apis.KeyDeleteJobOnCompletion] = "true"

		in.Spec.Template = *jobTemplate
		in.Spec.Template.Spec.ServiceAccountName = serviceAccountName
		return in
	})
	return nil
}

func (c *StashController) setRestoreSessionRunning(restoreSession *api_v1beta1.RestoreSession) error {

	totalHosts, err := c.getTotalHosts(restoreSession.Spec.Target, restoreSession.Namespace, restoreSession.Spec.Driver)
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
		eventer.EventSourceRestoreSessionController,
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
		eventer.EventSourceRestoreSessionController,
		restoreSession,
		core.EventTypeNormal,
		eventer.EventReasonRestoreSessionSucceeded,
		fmt.Sprintf("restore has been completed succesfully for RestoreSession %s/%s", restoreSession.Namespace, restoreSession.Name),
	)

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
		eventer.EventSourceRestoreSessionController,
		restoreSession,
		core.EventTypeWarning,
		eventer.EventReasonRestoreSessionFailed,
		jobErr.Error(),
	)

	return err
}

func (c *StashController) setRestoreSessionUnknown(restoreSession *api_v1beta1.RestoreSession, jobErr error) error {

	// set RestoreSession phase to "Unknown"
	_, err := v1beta1_util.UpdateRestoreSessionStatus(c.stashClient.StashV1beta1(), restoreSession, func(in *api_v1beta1.RestoreSessionStatus) *api_v1beta1.RestoreSessionStatus {
		in.Phase = api_v1beta1.RestoreSessionUnknown
		return in
	}, apis.EnableStatusSubresource)
	if err != nil {
		return err
	}

	// write failure event
	_, err = eventer.CreateEvent(
		c.kubeClient,
		eventer.EventSourceRestoreSessionController,
		restoreSession,
		core.EventTypeWarning,
		eventer.EventReasonRestorePhaseUnknown,
		jobErr.Error(),
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

	// check if any of the host phase is "Unknown". if any of their phase is "Unknown", then consider entire restore session phase is unknown.
	for _, host := range restoreSession.Status.Stats {
		if host.Phase == api_v1beta1.HostRestoreUnknown {
			return api_v1beta1.RestoreSessionUnknown, fmt.Errorf("restore phase is 'Unknown' for host: %s. Reason: %s", host.Hostname, host.Error)
		}
	}

	// restore has been completed successfully
	return api_v1beta1.RestoreSessionSucceeded, nil
}

// getPVCFromVolumeClaimTemplates returns list of PVCs generated according to the VolumeClaimTemplate
func getPVCFromVolumeClaimTemplates(ordinal int32, claimTemplates []core.PersistentVolumeClaim) ([]core.PersistentVolumeClaim, error) {
	pvcList := make([]core.PersistentVolumeClaim, 0)
	for _, claim := range claimTemplates {
		inputs := make(map[string]string)
		inputs[util.KeyPodOrdinal] = strconv.Itoa(int(ordinal))
		err := resolve.ResolvePVCSpec(&claim, inputs)
		if err != nil {
			return pvcList, err
		}
		pvcList = append(pvcList, claim)
	}
	return pvcList, nil
}

func getRestoreJobName(restoreSession *api_v1beta1.RestoreSession) string {
	return RestoreJobPrefix + strings.ReplaceAll(restoreSession.Name, ".", "-")
}

func getVolumeRestorerJobName(restoreSession *api_v1beta1.RestoreSession) string {
	return VolumeSnapshotPrefix + strings.ReplaceAll(restoreSession.Name, ".", "-")
}

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

package controller

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"stash.appscode.dev/apimachinery/apis"
	"stash.appscode.dev/apimachinery/apis/stash"
	api_v1alpha1 "stash.appscode.dev/apimachinery/apis/stash/v1alpha1"
	api_v1beta1 "stash.appscode.dev/apimachinery/apis/stash/v1beta1"
	"stash.appscode.dev/apimachinery/pkg/conditions"
	"stash.appscode.dev/apimachinery/pkg/docker"
	stashHooks "stash.appscode.dev/apimachinery/pkg/hooks"
	"stash.appscode.dev/apimachinery/pkg/invoker"
	"stash.appscode.dev/apimachinery/pkg/metrics"
	api_util "stash.appscode.dev/apimachinery/pkg/util"
	stash_rbac "stash.appscode.dev/stash/pkg/rbac"
	"stash.appscode.dev/stash/pkg/resolve"
	"stash.appscode.dev/stash/pkg/util"

	"gomodules.xyz/pointer"
	batch "k8s.io/api/batch/v1"
	core "k8s.io/api/core/v1"
	kerr "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/klog/v2"
	kmapi "kmodules.xyz/client-go/api/v1"
	batch_util "kmodules.xyz/client-go/batch/v1"
	core_util "kmodules.xyz/client-go/core/v1"
	"kmodules.xyz/client-go/meta"
	meta_util "kmodules.xyz/client-go/meta"
	"kmodules.xyz/client-go/tools/queue"
	appcat "kmodules.xyz/custom-resources/apis/appcatalog/v1alpha1"
	ofst "kmodules.xyz/offshoot-api/api/v1"
	"kmodules.xyz/webhook-runtime/admission"
	hooks "kmodules.xyz/webhook-runtime/admission/v1beta1"
	webhook "kmodules.xyz/webhook-runtime/admission/v1beta1/generic"
)

const (
	RestorerInitContainer = "init-container"
	RestorerCSIDriver     = "csi-driver"
	RestorerJob           = "restore-job"
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
				rs := obj.(*api_v1beta1.RestoreSession)
				return nil, c.validateRestoreSession(rs)
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

func (c *StashController) validateRestoreSession(rs *api_v1beta1.RestoreSession) error {
	err := rs.IsValid()
	if err != nil {
		return err
	}

	if rs.Spec.Target != nil {
		err := verifyCrossNamespacePermission(rs.ObjectMeta, rs.Spec.Target.Ref, rs.Spec.Task.Name)
		if err != nil {
			return err
		}
	}
	return c.validateAgainstUsagePolicy(rs.Spec.Repository, rs.Namespace)
}

func (c *StashController) NewRestoreSessionMutator() hooks.AdmissionHook {
	return webhook.NewGenericWebhook(
		schema.GroupVersionResource{
			Group:    "admission.stash.appscode.com",
			Version:  "v1beta1",
			Resource: "restoresessionmutators",
		},
		"restoresessionmutator",
		[]string{stash.GroupName},
		api_v1beta1.SchemeGroupVersion.WithKind(api_v1beta1.ResourceKindRestoreSession),
		nil,
		&admission.ResourceHandlerFuncs{
			CreateFunc: func(obj runtime.Object) (runtime.Object, error) {
				restoreSession := obj.(*api_v1beta1.RestoreSession)
				// if any deprecated field is used, migrate it to appropriate field
				restoreSession.Migrate()
				return restoreSession, nil
			},
			UpdateFunc: func(oldObj, newObj runtime.Object) (runtime.Object, error) {
				restoreSession := newObj.(*api_v1beta1.RestoreSession)
				// if any deprecated field is used, migrate it to appropriate field
				restoreSession.Migrate()
				return restoreSession, nil
			},
		},
	)
}

// process only add events
func (c *StashController) initRestoreSessionWatcher() {
	c.restoreSessionInformer = c.stashInformerFactory.Stash().V1beta1().RestoreSessions().Informer()
	c.restoreSessionQueue = queue.New(api_v1beta1.ResourceKindRestoreSession, c.MaxNumRequeues, c.NumThreads, c.runRestoreSessionProcessor)
	if c.auditor != nil {
		c.restoreSessionInformer.AddEventHandler(c.auditor.ForGVK(api_v1beta1.SchemeGroupVersion.WithKind(api_v1beta1.ResourceKindRestoreSession)))
	}
	c.restoreSessionInformer.AddEventHandler(queue.DefaultEventHandler(c.restoreSessionQueue.GetQueue(), core.NamespaceAll))
	c.restoreSessionLister = c.stashInformerFactory.Stash().V1beta1().RestoreSessions().Lister()
}

func (c *StashController) runRestoreSessionProcessor(key string) error {
	obj, exists, err := c.restoreSessionInformer.GetIndexer().GetByKey(key)
	if err != nil {
		klog.Errorf("Fetching object with key %s from store failed with %v", key, err)
		return err
	}
	if !exists {
		klog.Warningf("RestoreSession %s does not exist anymore\n", key)
		return nil
	}

	restoreSession := obj.(*api_v1beta1.RestoreSession)
	klog.Infof("Sync/Add/Update for RestoreSession %s", restoreSession.GetName())

	// process sync/add/update event
	inv := invoker.NewRestoreSessionInvoker(c.kubeClient, c.stashClient, restoreSession)

	// Apply any modification requires for smooth KubeDB integration
	err = inv.EnsureKubeDBIntegration(c.appCatalogClient)
	if err != nil {
		return err
	}

	return c.applyRestoreInvokerReconciliationLogic(inv, key)
}

func (c *StashController) applyRestoreInvokerReconciliationLogic(inv invoker.RestoreInvoker, key string) error {
	if isRestoreRunning(inv) && isRestoreDeadlineSet(inv) {
		if isRestoreTimeLimitExceeded(inv) {
			klog.Infof("Time Limit exceeded for %s  %s/%s.",
				inv.GetTypeMeta().Kind,
				inv.GetObjectMeta().Namespace,
				inv.GetObjectMeta().Name,
			)
			return conditions.SetRestoreDeadlineExceededConditionToTrue(inv, inv.GetTimeOut())
		}
	}

	// if the restore invoker is being deleted then remove respective init-container
	invMeta := inv.GetObjectMeta()
	invokerRef, err := inv.GetObjectRef()
	if err != nil {
		return err
	}

	if invMeta.DeletionTimestamp != nil {
		if core_util.HasFinalizer(invMeta, api_v1beta1.StashKey) {
			err := c.cleanupRestoreInvokerOffshoots(inv, invokerRef)
			if err != nil {
				return err
			}

			// remove finalizer
			return inv.RemoveFinalizer()
		}
		return nil
	}

	err = inv.AddFinalizer()
	if err != nil {
		return err
	}

	status := inv.GetStatus()
	if invoker.RestoreCompletedForAllTargets(status.TargetStatus, len(inv.GetTargetInfo())) {
		if !postRestoreHooksExecuted(inv) {
			klog.Infof("Waiting for postRestore hook to be executed for %s %s/%s.....",
				inv.GetTypeMeta().Kind,
				invMeta.Namespace,
				invMeta.Name,
			)
			return nil
		}

		if !globalPostRestoreHookExecuted(inv) {
			err = c.executeGlobalPostRestoreHook(inv)
			if err != nil {
				condErr := conditions.SetGlobalPostRestoreHookSucceededConditionToFalse(inv, err)
				if condErr != nil {
					return condErr
				}
			}
		}

		if !restoreMetricsPushed(status.Conditions) {
			err = c.sendRestoreMetrics(inv)
			if err != nil {
				condErr := conditions.SetRestoreMetricsPushedConditionToFalse(inv, err)
				if condErr != nil {
					return condErr
				}
			}
		}

		klog.Infof("Skipping processing %s %s/%s. Reason: phase is %q.",
			inv.GetTypeMeta().Kind,
			invMeta.Namespace,
			invMeta.Name,
			status.Phase,
		)
		return nil
	}

	if inv.GetDriver() == api_v1beta1.ResticSnapshotter {
		shouldRequeue, err := c.checkForResticSnapshotterRequirements(inv, inv)
		if err != nil {
			return err
		}
		if shouldRequeue {
			return c.requeueRestoreInvoker(inv, key, requeueTimeInterval)
		}
	}

	if !globalPreRestoreHookExecuted(inv) {
		err := c.executeGlobalPreRestoreHook(inv)
		if err != nil {
			return conditions.SetGlobalPreRestoreHookSucceededConditionToFalse(inv, err)
		}
	}

	// ===================== Run Restore for the Individual Targets ============================
	for i, targetInfo := range inv.GetTargetInfo() {
		if targetInfo.Target != nil {
			// Skip processing if the restore process has been already initiated before for this target
			if targetRestoreInitiated(inv, targetInfo.Target.Ref) {
				klog.Infof("Skipping initiating restore for %s %s/%s. Reason: Restore has been already initiated for this target.", targetInfo.Target.Ref.Kind, targetInfo.Target.Ref.Namespace, targetInfo.Target.Ref.Name)
				continue
			}

			// Skip processing if the target is not in next in order
			if !nextInOrder(inv, targetInfo) {
				klog.Infof("Skipping restoring for target %s %s/%s. Reason: Previous targets hasn't been executed.",
					targetInfo.Target.Ref.Kind,
					targetInfo.Target.Ref.Namespace,
					targetInfo.Target.Ref.Name,
				)
				err = c.setTargetRestorePending(inv, targetInfo.Target.Ref)
				if err != nil {
					return err
				}
				continue
			}

			tref := targetInfo.Target.Ref
			shouldRequeue, err := c.checkForTargetExistence(inv, tref, i)
			if err != nil {
				klog.Errorf("Failed to check whether APIVersion: %s Kind: %s Namespace: %s Name: %s exist or not. Reason: %v.",
					tref.APIVersion,
					tref.Kind,
					tref.Namespace,
					tref.Name,
					err.Error(),
				)
				return conditions.SetRestoreTargetFoundConditionToUnknown(inv, i, err)
			}
			if shouldRequeue {
				klog.Infof("Requeueing invoker %s %s/%s after 5 seconds....",
					inv.GetTypeMeta().Kind,
					inv.GetObjectMeta().Namespace,
					inv.GetObjectMeta().Name,
				)
				return c.requeueRestoreInvoker(inv, key, requeueTimeInterval)
			}

			err = c.ensureRestoreExecutor(inv, targetInfo, i)
			if err != nil {
				msg := fmt.Sprintf("failed to ensure restore executor. Reason: %v", err)
				klog.Warning(msg)
				return conditions.SetRestoreExecutorEnsuredToFalse(inv, &tref, msg)
			}
			return c.initiateTargetRestore(inv, i)
		}
	}

	if inv.GetTimeOut() != "" {
		if err := c.requeueRestoreAfterTimeOut(inv, key, inv.GetTimeOut()); err != nil {
			return err
		}
	}
	return nil
}

func isRestoreDeadlineSet(inv invoker.RestoreInvoker) bool {
	deadline := inv.GetStatus().SessionDeadline
	return !deadline.IsZero()
}

func isRestoreRunning(inv invoker.RestoreInvoker) bool {
	return inv.GetStatus().Phase == api_v1beta1.RestoreRunning
}

func isRestoreTimeLimitExceeded(inv invoker.RestoreInvoker) bool {
	return metav1.Now().After(inv.GetStatus().SessionDeadline.Time)
}

func (c *StashController) requeueRestoreAfterTimeOut(inv invoker.RestoreInvoker, key string, timeOut string) error {
	if !isRestoreDeadlineSet(inv) {
		duration, err := time.ParseDuration(timeOut)
		if err != nil {
			return err
		}
		if err := c.requeueRestoreInvoker(inv, key, duration); err != nil {
			return err
		}
		klog.Infof("Timeout is set for %s: %s/%s.\nRequeueing after %v .....",
			inv.GetTypeMeta().Kind,
			inv.GetObjectMeta().Namespace,
			inv.GetObjectMeta().Name,
			timeOut,
		)
		return c.setRestoreDeadline(inv, duration)
	}
	return nil
}

func (c *StashController) setRestoreDeadline(inv invoker.RestoreInvoker, timeOut time.Duration) error {
	return inv.UpdateStatus(invoker.RestoreInvokerStatus{
		SessionDeadline: metav1.NewTime(inv.GetObjectMeta().CreationTimestamp.Add(timeOut)),
	})
}

func (c *StashController) ensureRestoreJob(inv invoker.RestoreInvoker, index int) error {
	invMeta := inv.GetObjectMeta()

	image := docker.Docker{
		Registry: c.DockerRegistry,
		Image:    c.StashImage,
		Tag:      c.StashImageTag,
	}

	jobMeta := metav1.ObjectMeta{
		Name:      getRestoreJobName(invMeta, strconv.Itoa(index)),
		Namespace: invMeta.Namespace,
		Labels:    inv.GetLabels(),
	}

	targetInfo := inv.GetTargetInfo()[index]

	psps, err := c.getRestoreJobPSPNames(targetInfo.Task)
	if err != nil {
		return err
	}

	rbacOptions, err := c.getRestoreRBACOptions(inv, &index)
	if err != nil {
		return err
	}

	runtimeSettings := targetInfo.RuntimeSettings
	if runtimeSettings.Pod != nil && runtimeSettings.Pod.ServiceAccountAnnotations != nil {
		rbacOptions.ServiceAccount.Annotations = runtimeSettings.Pod.ServiceAccountAnnotations
	}

	rbacOptions.PodSecurityPolicyNames = psps

	if targetInfo.RuntimeSettings.Pod != nil && targetInfo.RuntimeSettings.Pod.ServiceAccountName != "" {
		rbacOptions.ServiceAccount.Name = targetInfo.RuntimeSettings.Pod.ServiceAccountName
	}

	err = rbacOptions.EnsureRestoreJobRBAC()
	if err != nil {
		return err
	}

	// if the Stash is using a private registry, then ensure the image pull secrets
	var imagePullSecrets []core.LocalObjectReference
	if c.ImagePullSecrets != nil {
		imagePullSecrets, err = c.ensureImagePullSecrets(invMeta, inv.GetOwnerRef())
		if err != nil {
			return err
		}
	}

	// get repository for RestoreSession
	repository, err := inv.GetRepository()
	if err != nil {
		return err
	}
	addon, err := api_util.ExtractAddonInfo(c.appCatalogClient, targetInfo.Task, targetInfo.Target.Ref)
	if err != nil {
		return err
	}

	// Now, there could be two restore scenario for restoring through job.
	// 1. Restore process follows Function-Task model. In this case, we have to resolve respective Functions and Task to get desired job definition.
	// 2. Restore process does not follow Function-Task model. In this case, we have to generate simple volume restorer job definition.

	var jobTemplate *core.PodTemplateSpec

	if addon.RestoreTask.Name != "" {
		// Restore process follows Function-Task model. So, resolve Function and Task to get desired job definition.
		jobTemplate, err = c.resolveRestoreTask(inv, repository, index, addon)
		if err != nil {
			return err
		}
	} else {
		// Restore process does not follow Function-Task model. So, generate simple volume restorer job definition.
		jobTemplate, err = util.NewPVCRestorerJob(inv, index, repository, image)
		if err != nil {
			return err
		}
	}

	// If volumeClaimTemplate is not specified then we don't need any further processing. Just, create the job
	if targetInfo.Target == nil ||
		(targetInfo.Target != nil && len(targetInfo.Target.VolumeClaimTemplates) == 0) {
		// upsert InterimVolume to hold the backup/restored data temporarily
		jobTemplate.Spec, err = util.UpsertInterimVolume(
			c.kubeClient,
			jobTemplate.Spec,
			targetInfo.InterimVolumeTemplate.ToCorePVC(),
			invMeta.Namespace,
			inv.GetOwnerRef(),
		)
		if err != nil {
			return err
		}
		runtimeSettings := targetInfo.RuntimeSettings
		// pass offshoot labels to job's pod
		jobTemplate.Labels = meta_util.OverwriteKeys(jobTemplate.Labels, inv.GetLabels())
		jobTemplate.Spec.ImagePullSecrets = core_util.MergeLocalObjectReferences(jobTemplate.Spec.ImagePullSecrets, imagePullSecrets)
		jobTemplate.Spec.ServiceAccountName = rbacOptions.ServiceAccount.Name
		if runtimeSettings.Pod != nil && runtimeSettings.Pod.PodAnnotations != nil {
			jobTemplate.Annotations = runtimeSettings.Pod.PodAnnotations
		}
		return c.createRestoreJob(jobTemplate, jobMeta, inv.GetOwnerRef())
	}

	// volumeClaimTemplate has been specified. Now, we have to do the following for each replica:
	// 1. Create PVCs according to the template.
	// 2. Mount the PVCs to the restore job.
	// 3. Create the restore job to restore the target.

	replicas := int32(1) // set default replicas to 1
	if targetInfo.Target.Replicas != nil {
		replicas = *targetInfo.Target.Replicas
	}

	for ordinal := int32(0); ordinal < replicas; ordinal++ {
		// resolve template part of the volumeClaimTemplate and generate PVC definition according to the template
		pvcList, err := resolve.GetPVCFromVolumeClaimTemplates(ordinal, targetInfo.Target.VolumeClaimTemplates)
		if err != nil {
			return err
		}

		// create the PVCs
		err = util.CreateBatchPVC(c.kubeClient, invMeta.Namespace, pvcList)
		if err != nil {
			return err
		}

		// add ordinal suffix to the job name so that multiple restore job can run concurrently
		restoreJobMeta := jobMeta.DeepCopy()
		restoreJobMeta.Name = fmt.Sprintf("%s-%d", jobMeta.Name, ordinal)

		var restoreJobTemplate *core.PodTemplateSpec

		// if restore process follows Function-Task model, then resolve the Functions and Task  for this host
		if addon.RestoreTask.Name != "" {
			restoreJobTemplate, err = c.resolveRestoreTask(inv, repository, index, addon)

			if err != nil {
				return err
			}
		} else {
			// use copy of the original job template. otherwise, each iteration will append volumes in the same template
			restoreJobTemplate = jobTemplate.DeepCopy()
		}

		// mount the newly created PVCs into the job
		volumes := util.PVCListToVolumes(pvcList, ordinal)
		restoreJobTemplate.Spec = util.AttachPVC(restoreJobTemplate.Spec, volumes, targetInfo.Target.VolumeMounts)

		ordinalEnv := core.EnvVar{
			Name:  apis.KeyPodOrdinal,
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

		restoreJobTemplate.Spec.ImagePullSecrets = core_util.MergeLocalObjectReferences(restoreJobTemplate.Spec.ImagePullSecrets, imagePullSecrets)
		restoreJobTemplate.Spec.ServiceAccountName = rbacOptions.ServiceAccount.Name

		// create restore job
		err = c.createRestoreJob(restoreJobTemplate, *restoreJobMeta, inv.GetOwnerRef())
		if err != nil {
			return err
		}
	}
	return nil
}

func (c *StashController) createRestoreJob(jobTemplate *core.PodTemplateSpec, meta metav1.ObjectMeta, owner *metav1.OwnerReference) error {
	_, _, err := batch_util.CreateOrPatchJob(
		context.TODO(),
		c.kubeClient,
		meta,
		func(in *batch.Job) *batch.Job {
			// set RestoreSession as owner of this Job
			core_util.EnsureOwnerReference(&in.ObjectMeta, owner)

			in.Spec.Template.Spec = jobTemplate.Spec
			in.Spec.Template.Labels = meta_util.OverwriteKeys(in.Spec.Template.Labels, jobTemplate.Labels)
			in.Spec.BackoffLimit = pointer.Int32P(0)
			return in
		},
		metav1.PatchOptions{},
	)
	return err
}

// resolveRestoreTask resolves Functions and Tasks then returns a job definition to restore the target.
func (c *StashController) resolveRestoreTask(inv invoker.RestoreInvoker, repository *api_v1alpha1.Repository, index int, addon *appcat.StashTaskSpec) (*core.PodTemplateSpec, error) {
	invMeta := inv.GetObjectMeta()
	targetInfo := inv.GetTargetInfo()[index]

	// resolve task template
	repoInputs, err := c.inputsForRepository(repository)
	if err != nil {
		return nil, fmt.Errorf("cannot resolve implicit inputs for Repository %s/%s, reason: %s", repository.Namespace, repository.Name, err)
	}
	rsInputs := c.inputsForRestoreInvoker(inv, index)

	implicitInputs := meta_util.OverwriteKeys(repoInputs, rsInputs)
	implicitInputs[apis.Namespace] = invMeta.Namespace
	implicitInputs[apis.RestoreSession] = invMeta.Name

	// add docker image specific input
	implicitInputs[apis.StashDockerRegistry] = c.DockerRegistry
	implicitInputs[apis.StashDockerImage] = c.StashImage
	implicitInputs[apis.StashImageTag] = c.StashImageTag
	// license related inputs
	implicitInputs[apis.LicenseApiService] = c.LicenseApiService

	taskResolver := resolve.TaskResolver{
		StashClient:     c.stashClient,
		TaskName:        addon.RestoreTask.Name,
		Inputs:          meta_util.OverwriteKeys(explicitInputs(addon.RestoreTask.Params), implicitInputs),
		RuntimeSettings: targetInfo.RuntimeSettings,
		TempDir:         targetInfo.TempDir,
	}

	// if preRestore or postRestore Hook is specified, add their specific inputs
	if targetInfo.Hooks != nil && targetInfo.Hooks.PreRestore != nil {
		taskResolver.PreTaskHookInput = make(map[string]string)
		taskResolver.PreTaskHookInput[apis.HookType] = apis.PreRestoreHook
	}
	if targetInfo.Hooks != nil &&
		targetInfo.Hooks.PostRestore != nil &&
		targetInfo.Hooks.PostRestore.Handler != nil {
		taskResolver.PostTaskHookInput = make(map[string]string)
		taskResolver.PostTaskHookInput[apis.HookType] = apis.PostRestoreHook
	}

	// In order to preserve file ownership, restore process need to be run as root user.
	// Stash image uses non-root user 65535. We have to use securityContext to run stash as root user.
	// If a user specify securityContext either in pod level or container level in RuntimeSetting,
	// don't overwrite that. In this case, user must take the responsibility of possible file ownership modification.
	defaultSecurityContext := &core.PodSecurityContext{
		RunAsUser:  pointer.Int64P(0),
		RunAsGroup: pointer.Int64P(0),
	}

	if taskResolver.RuntimeSettings.Pod == nil {
		taskResolver.RuntimeSettings.Pod = &ofst.PodRuntimeSettings{}
	}
	taskResolver.RuntimeSettings.Pod.SecurityContext = util.UpsertPodSecurityContext(defaultSecurityContext, taskResolver.RuntimeSettings.Pod.SecurityContext)

	podSpec, err := taskResolver.GetPodSpec(inv.GetTypeMeta().Kind, invMeta.Name, targetInfo.Target.Ref)
	if err != nil {
		return nil, err
	}

	podTemplate := &core.PodTemplateSpec{
		Spec: podSpec,
	}
	return podTemplate, nil
}

func (c *StashController) ensureVolumeRestorerJob(inv invoker.RestoreInvoker, index int) error {
	invMeta := inv.GetObjectMeta()
	jobMeta := metav1.ObjectMeta{
		Name:      getVolumeRestorerJobName(invMeta, strconv.Itoa(index)),
		Namespace: invMeta.Namespace,
		Labels:    inv.GetLabels(),
	}

	targetInfo := inv.GetTargetInfo()[index]

	rbacOptions, err := c.getRestoreRBACOptions(inv, &index)
	if err != nil {
		return err
	}

	runtimeSettings := targetInfo.RuntimeSettings
	if runtimeSettings.Pod != nil && runtimeSettings.Pod.ServiceAccountAnnotations != nil {
		rbacOptions.ServiceAccount.Annotations = runtimeSettings.Pod.ServiceAccountAnnotations
	}

	err = rbacOptions.EnsureVolumeSnapshotRestorerJobRBAC()
	if err != nil {
		return err
	}

	// if the Stash is using a private registry, then ensure the image pull secrets
	var imagePullSecrets []core.LocalObjectReference
	if c.ImagePullSecrets != nil {
		imagePullSecrets, err = c.ensureImagePullSecrets(invMeta, inv.GetOwnerRef())
		if err != nil {
			return err
		}
	}

	image := docker.Docker{
		Registry: c.DockerRegistry,
		Image:    c.StashImage,
		Tag:      c.StashImageTag,
	}

	jobTemplate, err := util.NewVolumeRestorerJob(inv, index, image)
	if err != nil {
		return err
	}

	// Create Volume restorer Job
	_, _, err = batch_util.CreateOrPatchJob(
		context.TODO(),
		c.kubeClient,
		jobMeta,
		func(in *batch.Job) *batch.Job {
			// set restore invoker as owner of this Job
			core_util.EnsureOwnerReference(&in.ObjectMeta, inv.GetOwnerRef())

			in.Spec.Template.Spec = jobTemplate.Spec
			in.Spec.Template.Labels = meta_util.OverwriteKeys(in.Spec.Template.Labels, inv.GetLabels())
			in.Spec.Template.Spec.ImagePullSecrets = core_util.MergeLocalObjectReferences(in.Spec.Template.Spec.ImagePullSecrets, imagePullSecrets)
			in.Spec.Template.Spec.ServiceAccountName = rbacOptions.ServiceAccount.Name
			in.Spec.BackoffLimit = pointer.Int32P(0)
			if runtimeSettings.Pod != nil && runtimeSettings.Pod.PodAnnotations != nil {
				jobTemplate.Annotations = runtimeSettings.Pod.PodAnnotations
			}
			return in
		},
		metav1.PatchOptions{},
	)
	return err
}

func getRestoreJobName(invokerMeta metav1.ObjectMeta, suffix string) string {
	return meta.ValidNameWithPrefixNSuffix(apis.PrefixStashRestore, strings.ReplaceAll(invokerMeta.Name, ".", "-"), suffix)
}

func getVolumeRestorerJobName(invokerMeta metav1.ObjectMeta, index string) string {
	return meta.ValidNameWithPrefixNSuffix(apis.PrefixStashVolumeSnapshot, strings.ReplaceAll(invokerMeta.Name, ".", "-"), index)
}

func (c *StashController) restorerEntity(targetInfo invoker.RestoreTargetInfo, driver api_v1beta1.Snapshotter) string {
	if util.RestoreModel(targetInfo.Target.Ref.Kind, targetInfo.Task.Name) == apis.ModelSidecar {
		return RestorerInitContainer
	} else if driver == api_v1beta1.VolumeSnapshotter {
		return RestorerCSIDriver
	} else {
		return RestorerJob
	}
}

func (c *StashController) requeueRestoreInvoker(inv invoker.RestoreInvoker, key string, requeueTimeInterval time.Duration) error {
	invTypeMeta := inv.GetTypeMeta()
	switch invTypeMeta.Kind {
	case api_v1beta1.ResourceKindRestoreSession:
		c.restoreSessionQueue.GetQueue().AddAfter(key, requeueTimeInterval)
	default:
		return fmt.Errorf("unable to requeue. Reason: Restore invoker %s %s is not supported",
			invTypeMeta.APIVersion,
			invTypeMeta.Kind,
		)
	}
	return nil
}

func postRestoreHooksExecuted(inv invoker.RestoreInvoker) bool {
	for _, targetInfo := range inv.GetTargetInfo() {
		if targetInfo.Hooks != nil && targetInfo.Hooks.PostRestore != nil {
			if !postRestoreHookExecutedForTarget(inv, targetInfo) {
				return false
			}
		}
	}
	return true
}

func postRestoreHookExecutedForTarget(inv invoker.RestoreInvoker, targetInfo invoker.RestoreTargetInfo) bool {
	if targetInfo.Target == nil {
		return true
	}
	status := inv.GetStatus()

	for _, s := range status.TargetStatus {
		if invoker.TargetMatched(s.Ref, targetInfo.Target.Ref) {
			if kmapi.HasCondition(s.Conditions, api_v1beta1.PostRestoreHookExecutionSucceeded) {
				return true
			}
		}
	}
	return false
}

func globalPostRestoreHookExecuted(inv invoker.RestoreInvoker) bool {
	if inv.GetGlobalHooks() == nil ||
		inv.GetGlobalHooks().PostRestore == nil ||
		inv.GetGlobalHooks().PostRestore.Handler == nil {
		return true
	}
	return kmapi.HasCondition(inv.GetStatus().Conditions, api_v1beta1.GlobalPostRestoreHookSucceeded) &&
		kmapi.IsConditionTrue(inv.GetStatus().Conditions, api_v1beta1.GlobalPostRestoreHookSucceeded)
}

func (c *StashController) executeGlobalPostRestoreHook(inv invoker.RestoreInvoker) error {
	hookExecutor := stashHooks.HookExecutor{
		Config: c.clientConfig,
		Hook:   inv.GetGlobalHooks().PostRestore.Handler,
		ExecutorPod: kmapi.ObjectReference{
			Namespace: meta.PodNamespace(),
			Name:      meta.PodName(),
		},
		Summary: inv.GetSummary(api_v1beta1.TargetRef{}, kmapi.ObjectReference{
			Namespace: inv.GetObjectMeta().Namespace,
			Name:      inv.GetObjectMeta().Name,
		}),
	}

	executionPolicy := inv.GetGlobalHooks().PostRestore.ExecutionPolicy
	if executionPolicy == "" {
		executionPolicy = api_v1beta1.ExecuteAlways
	}

	if !stashHooks.IsAllowedByExecutionPolicy(executionPolicy, hookExecutor.Summary) {
		reason := fmt.Sprintf("Skipping executing %s. Reason: executionPolicy is %q but phase is %q.",
			apis.PostRestoreHook,
			executionPolicy,
			hookExecutor.Summary.Status.Phase,
		)
		return conditions.SetGlobalPostRestoreHookSucceededConditionToTrueWithMsg(inv, reason)
	}

	if err := hookExecutor.Execute(); err != nil {
		return err
	}
	return conditions.SetGlobalPostRestoreHookSucceededConditionToTrue(inv)
}

func globalPreRestoreHookExecuted(inv invoker.RestoreInvoker) bool {
	if inv.GetGlobalHooks() == nil || inv.GetGlobalHooks().PreRestore == nil {
		return true
	}
	return kmapi.HasCondition(inv.GetStatus().Conditions, api_v1beta1.GlobalPreRestoreHookSucceeded) &&
		kmapi.IsConditionTrue(inv.GetStatus().Conditions, api_v1beta1.GlobalPreRestoreHookSucceeded)
}

func (c *StashController) executeGlobalPreRestoreHook(inv invoker.RestoreInvoker) error {
	hookExecutor := stashHooks.HookExecutor{
		Config: c.clientConfig,
		Hook:   inv.GetGlobalHooks().PreRestore,
		ExecutorPod: kmapi.ObjectReference{
			Namespace: meta.PodNamespace(),
			Name:      meta.PodName(),
		},
		Summary: inv.GetSummary(api_v1beta1.TargetRef{}, kmapi.ObjectReference{
			Namespace: inv.GetObjectMeta().Namespace,
			Name:      inv.GetObjectMeta().Name,
		}),
	}
	if err := hookExecutor.Execute(); err != nil {
		return err
	}
	return conditions.SetGlobalPreRestoreHookSucceededConditionToTrue(inv)
}

func targetRestoreInitiated(inv invoker.RestoreInvoker, targetRef api_v1beta1.TargetRef) bool {
	status := inv.GetStatus()
	if invoker.TargetRestoreCompleted(targetRef, status.TargetStatus) {
		return true
	}
	for _, target := range status.TargetStatus {
		if invoker.TargetMatched(target.Ref, targetRef) {
			return target.Phase == api_v1beta1.TargetRestoreRunning
		}
	}
	return false
}

func (c *StashController) getRestoreRBACOptions(inv invoker.RestoreInvoker, index *int) (stash_rbac.RBACOptions, error) {
	invMeta := inv.GetObjectMeta()
	repo := inv.GetRepoRef()

	rbacOptions := stash_rbac.RBACOptions{
		KubeClient: c.kubeClient,
		Invoker: stash_rbac.InvokerOptions{
			ObjectMeta: invMeta,
			TypeMeta:   inv.GetTypeMeta(),
		},
		Owner:          inv.GetOwnerRef(),
		OffshootLabels: inv.GetLabels(),
		ServiceAccount: metav1.ObjectMeta{
			Namespace: invMeta.Namespace,
		},
	}

	if repo.Namespace != invMeta.Namespace {
		repository, err := c.repoLister.Repositories(repo.Namespace).Get(repo.Name)
		if err != nil {
			if kerr.IsNotFound(err) {
				return rbacOptions, nil
			}
			return rbacOptions, err
		}

		rbacOptions.CrossNamespaceResources = &stash_rbac.CrossNamespaceResources{
			Repository: repo.Name,
			Namespace:  repo.Namespace,
			Secret:     repository.Spec.Backend.StorageSecretName,
		}
	}
	rbacOptions.Suffix = "0"
	if index != nil {
		rbacOptions.Suffix = fmt.Sprintf("%d", *index)
	}
	return rbacOptions, nil
}

func (c *StashController) initiateTargetRestore(inv invoker.RestoreInvoker, index int) error {
	targetInfo := inv.GetTargetInfo()[index]
	totalHosts, err := c.getTotalHosts(targetInfo.Target, inv.GetDriver())
	if err != nil {
		return err
	}
	return inv.UpdateStatus(invoker.RestoreInvokerStatus{
		TargetStatus: []api_v1beta1.RestoreMemberStatus{
			{
				Ref:        targetInfo.Target.Ref,
				TotalHosts: totalHosts,
			},
		},
	})
}

func (c *StashController) setTargetRestorePending(inv invoker.RestoreInvoker, targetRef api_v1beta1.TargetRef) error {
	return inv.UpdateStatus(invoker.RestoreInvokerStatus{
		TargetStatus: []api_v1beta1.RestoreMemberStatus{
			{
				Ref: targetRef,
			},
		},
	})
}

func restoreMetricsPushed(conditions []kmapi.Condition) bool {
	return kmapi.HasCondition(conditions, api_v1beta1.MetricsPushed)
}

func (c *StashController) sendRestoreMetrics(inv invoker.RestoreInvoker) error {
	// send restore metrics
	metricsOpt := &metrics.MetricsOptions{
		Enabled:        true,
		PushgatewayURL: metrics.GetPushgatewayURL(),
		JobName:        fmt.Sprintf("%s-%s-%s", strings.ToLower(inv.GetTypeMeta().Kind), inv.GetObjectMeta().Namespace, inv.GetObjectMeta().Name),
	}
	// send target specific metrics
	for _, target := range inv.GetStatus().TargetStatus {
		err := metricsOpt.SendRestoreTargetMetrics(c.clientConfig, inv, target.Ref)
		if err != nil {
			return err
		}
	}
	// send restore session metrics
	err := metricsOpt.SendRestoreSessionMetrics(inv)
	if err != nil {
		return err
	}
	return conditions.SetRestoreMetricsPushedConditionToTrue(inv)
}

func (c *StashController) checkForTargetExistence(inv invoker.RestoreInvoker, tref api_v1beta1.TargetRef, idx int) (bool, error) {
	// if target hasn't been specified, we don't need to check for its existence
	if tref.Name == "" {
		return false, nil
	}

	targetExist, err := util.IsTargetExist(c.clientConfig, tref)
	if err != nil {
		return false, err
	}

	if !targetExist {
		// Target does not exist. Log the information.
		klog.Infof("Restore target %s %s %s/%s does not exist.",
			tref.APIVersion,
			tref.Kind,
			tref.Namespace,
			tref.Name,
		)
		return true, conditions.SetRestoreTargetFoundConditionToFalse(inv, idx)
	}

	return false, conditions.SetRestoreTargetFoundConditionToTrue(inv, idx)
}

func nextInOrder(inv invoker.RestoreInvoker, targetInfo invoker.RestoreTargetInfo) bool {
	if inv.GetExecutionOrder() == api_v1beta1.Sequential &&
		!inv.NextInOrder(targetInfo.Target.Ref, inv.GetStatus().TargetStatus) {
		return false
	}
	return true
}

func (c *StashController) ensureRestoreExecutor(inv invoker.RestoreInvoker, targetInfo invoker.RestoreTargetInfo, idx int) error {
	tref := targetInfo.Target.Ref
	switch c.restorerEntity(targetInfo, inv.GetDriver()) {
	case RestorerInitContainer:
		// The target is kubernetes workload i.e. Deployment, StatefulSet etc.
		// Send event to the respective workload controller. The workload controller will take care of injecting restore init-container.
		err := c.sendEventToWorkloadQueue(
			tref.Kind,
			tref.Namespace,
			tref.Name,
		)
		if err != nil {
			return fmt.Errorf("failed to trigger workload controller for %s %s/%s. Error: %v", tref.Kind, tref.Namespace, tref.Name, err)
		}
	case RestorerCSIDriver:
		// VolumeSnapshotter driver has been used. So, ensure VolumeRestorer job
		err := c.ensureVolumeRestorerJob(inv, idx)
		if err != nil {
			return fmt.Errorf("failed to ensure volume snapshotter job for %s %s/%s. Error: %v", tref.Kind, tref.Namespace, tref.Name, err)
		}
	case RestorerJob:
		// Restic driver has been used. Ensure restore job.
		err := c.ensureRestoreJob(inv, idx)
		if err != nil {
			return fmt.Errorf("failed to ensure restore job for %s %s/%s. Error: %v", tref.Kind, tref.Namespace, tref.Name, err)
		}
	default:
		return fmt.Errorf("unable to identify restorer entity for target %s %s/%s", tref.Kind, tref.Namespace, tref.Name)
	}
	msg := fmt.Sprintf("Restorer job/init-container has been ensured successfully for %s %s/%s.", tref.Kind, tref.Namespace, tref.Name)
	return conditions.SetRestoreExecutorEnsuredToTrue(inv, &tref, msg)
}

func (c *StashController) cleanupRestoreInvokerOffshoots(inv invoker.RestoreInvoker, invokerRef *core.ObjectReference) error {
	for _, targetInfo := range inv.GetTargetInfo() {
		target := targetInfo.Target
		if target != nil && util.RestoreModel(target.Ref.Kind, targetInfo.Task.Name) == apis.ModelSidecar {
			// send event to workload controller. workload controller will take care of removing restore init-container
			err := c.sendEventToWorkloadQueue(
				target.Ref.Kind,
				target.Ref.Namespace,
				target.Ref.Name,
			)
			if err != nil {
				return c.handleWorkloadControllerTriggerFailure(invokerRef, err)
			}
		}
	}

	rbacOptions, err := c.getRestoreRBACOptions(inv, nil)
	if err != nil {
		return err
	}

	if err := rbacOptions.EnsureRBACResourcesDeleted(); err != nil {
		return err
	}

	return c.deleteRepositoryReferences(inv)
}

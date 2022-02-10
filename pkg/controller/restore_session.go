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
	"os"
	"strconv"
	"strings"
	"time"

	"stash.appscode.dev/apimachinery/apis"
	"stash.appscode.dev/apimachinery/apis/stash"
	api_v1alpha1 "stash.appscode.dev/apimachinery/apis/stash/v1alpha1"
	api_v1beta1 "stash.appscode.dev/apimachinery/apis/stash/v1beta1"
	"stash.appscode.dev/apimachinery/pkg/conditions"
	"stash.appscode.dev/apimachinery/pkg/docker"
	"stash.appscode.dev/apimachinery/pkg/invoker"
	"stash.appscode.dev/apimachinery/pkg/metrics"
	api_util "stash.appscode.dev/apimachinery/pkg/util"
	stash_rbac "stash.appscode.dev/stash/pkg/rbac"
	"stash.appscode.dev/stash/pkg/resolve"
	"stash.appscode.dev/stash/pkg/util"

	"gomodules.xyz/pointer"
	batchv1 "k8s.io/api/batch/v1"
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
				err := rs.IsValid()
				if err != nil {
					return nil, err
				}
				return nil, c.validateAgainstUsagePolicy(rs.Spec.Repository, rs.Namespace)
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
	// if the restore invoker is being deleted then remove respective init-container
	invMeta := inv.GetObjectMeta()
	invokerRef, err := inv.GetObjectRef()
	if err != nil {
		return err
	}

	if invMeta.DeletionTimestamp != nil {
		// if RestoreSession has stash finalizer then respective init-container (for workloads) hasn't been removed
		// remove respective init-container and finally remove finalizer
		if core_util.HasFinalizer(invMeta, api_v1beta1.StashKey) {
			for _, targetInfo := range inv.GetTargetInfo() {
				target := targetInfo.Target
				if target != nil && util.RestoreModel(target.Ref.Kind) == apis.ModelSidecar {
					// send event to workload controller. workload controller will take care of removing restore init-container
					err := c.sendEventToWorkloadQueue(
						target.Ref.Kind,
						invMeta.Namespace,
						target.Ref.Name,
					)
					if err != nil {
						return c.handleWorkloadControllerTriggerFailure(invokerRef, err)
					}
				}
			}

			rbacOptions, err := c.getRestoreRBACOptions(inv)
			if err != nil {
				return err
			}

			if err := rbacOptions.EnsureRBACResourcesDeleted(); err != nil {
				return err
			}

			err = c.deleteRepositoryReferences(inv)
			if err != nil {
				return err
			}

			// remove finalizer
			return inv.RemoveFinalizer()
		}
		return nil
	}

	// add finalizer
	err = inv.AddFinalizer()
	if err != nil {
		return err
	}

	// ================= Don't Process Completed Invoker ===========================
	status := inv.GetStatus()
	if invoker.IsRestoreCompleted(status.Phase) {
		klog.Infof("Skipping processing %s %s/%s. Reason: phase is %q.",
			inv.GetTypeMeta().Kind,
			invMeta.Namespace,
			invMeta.Name,
			status.Phase,
		)
		err = c.ensureMetricsPushed(inv)
		if err != nil {
			return conditions.SetMetricsPushedConditionToFalse(inv, nil, err)
		}
		return nil
	}

	// ======================== Set Global Conditions ============================
	if inv.GetDriver() == api_v1beta1.ResticSnapshotter {
		// Check whether Repository exist or not
		repository, err := inv.GetRepository()

		if err != nil {
			if kerr.IsNotFound(err) {
				klog.Infof("Repository %s/%s does not exist."+
					"\nRequeueing after 5 seconds......",
					inv.GetRepoRef().Namespace,
					inv.GetRepoRef().Name)

				err = conditions.SetRepositoryFoundConditionToFalse(inv)
				if err != nil {
					return err
				}

				return c.requeueRestoreInvoker(inv, key)
			}
			return conditions.SetRepositoryFoundConditionToUnknown(inv, err)
		}

		if repository.ObjectMeta.DeletionTimestamp != nil {
			klog.Infof("Repository %s/%s has been deleted."+
				"\nRequeueing after 5 seconds......",
				inv.GetRepoRef().Namespace,
				inv.GetRepoRef().Name)

			err = conditions.SetRepositoryFoundConditionToFalse(inv)
			if err != nil {
				return err
			}

			return c.requeueRestoreInvoker(inv, key)
		}

		err = conditions.SetRepositoryFoundConditionToTrue(inv)
		if err != nil {
			return err
		}

		err = c.upsertRepositoryReferences(inv)
		if err != nil {
			return err
		}

		// Check whether the backend Secret exist or not
		secret, err := c.kubeClient.CoreV1().Secrets(repository.Namespace).Get(context.TODO(), repository.Spec.Backend.StorageSecretName, metav1.GetOptions{})
		if err != nil {
			if kerr.IsNotFound(err) {
				klog.Infof("Backend Secret %s/%s does not exist for Repository %s/%s.\nRequeueing after 5 seconds......",
					secret.Namespace,
					secret.Name,
					repository.Namespace,
					repository.Name,
				)
				err2 := conditions.SetBackendSecretFoundConditionToFalse(inv, secret.Name)
				if err2 != nil {
					return err2
				}
				return c.requeueRestoreInvoker(inv, key)
			}
			return conditions.SetBackendSecretFoundConditionToUnknown(inv, secret.Name, err)
		}
		err = conditions.SetBackendSecretFoundConditionToTrue(inv, secret.Name)
		if err != nil {
			return err
		}

		err = c.validateAgainstUsagePolicy(inv.GetRepoRef(), inv.GetObjectMeta().Namespace)
		if err != nil {
			condErr := conditions.SetValidationPassedToFalse(inv, err)
			if condErr != nil {
				return condErr
			}
		} else {
			condErr := conditions.SetValidationPassedToTrue(inv)
			if condErr != nil {
				return condErr
			}
		}
	}
	phase := inv.GetStatus().Phase

	// ==================== Execute Global PostRestore Hooks ===========================
	// if the restore process has completed(Failed or Succeeded or Unknown), then execute global postRestore hook if not yet executed
	if invoker.IsRestoreCompleted(phase) && !globalPostRestoreHookExecuted(inv) {
		err = util.ExecuteHook(c.clientConfig, inv.GetGlobalHooks(), apis.PostRestoreHook, os.Getenv("MY_POD_NAME"), os.Getenv("MY_POD_NAMESPACE"))
		if err != nil {
			return conditions.SetGlobalPostRestoreHookSucceededConditionToFalse(inv, err)
		}
		err = conditions.SetGlobalPostRestoreHookSucceededConditionToTrue(inv)
		if err != nil {
			return err
		}
	}

	// ==================== Execute Global PreRestore Hook =====================
	// if global preRestore hook exist and not executed yet, then execute the preRestoreHook
	if !globalPreRestoreHookExecuted(inv) {
		err = util.ExecuteHook(c.clientConfig, inv.GetGlobalHooks(), apis.PreBackupHook, os.Getenv("MY_POD_NAME"), os.Getenv("MY_POD_NAMESPACE"))
		if err != nil {
			return conditions.SetGlobalPreRestoreHookSucceededConditionToFalse(inv, err)
		}
		err = conditions.SetGlobalPreRestoreHookSucceededConditionToTrue(inv)
		if err != nil {
			return err
		}
	}

	// ===================== Run Restore for the Individual Targets ============================
	for i, targetInfo := range inv.GetTargetInfo() {
		if targetInfo.Target != nil {
			// Skip processing if the restore process has been already initiated before for this target
			if targetRestoreInitiated(inv, targetInfo.Target.Ref) {
				continue
			}
			// ----------------- Ensure Execution Order -------------------
			if inv.GetExecutionOrder() == api_v1beta1.Sequential &&
				!inv.NextInOrder(targetInfo.Target.Ref, inv.GetStatus().TargetStatus) {
				continue
			}

			// ------------- Set Target Specific Conditions --------------
			tref := targetInfo.Target.Ref
			if tref.Name != "" {
				wc := util.WorkloadClients{
					KubeClient:       c.kubeClient,
					OcClient:         c.ocClient,
					StashClient:      c.stashClient,
					CRDClient:        c.crdClient,
					AppCatalogClient: c.appCatalogClient,
				}
				targetExist, err := wc.IsTargetExist(tref, invMeta.Namespace)
				if err != nil {
					klog.Errorf("Failed to check whether %s %s %s/%s exist or not. Reason: %v.",
						tref.APIVersion,
						tref.Kind,
						invMeta.Namespace,
						tref.Name,
						err.Error(),
					)
					return conditions.SetRestoreTargetFoundConditionToUnknown(inv, i, err)
				}

				if !targetExist {
					// Target does not exist. Log the information.
					klog.Infof("Restore target %s %s %s/%s does not exist.",
						tref.APIVersion,
						tref.Kind,
						invMeta.Namespace,
						tref.Name)
					// Set the "RestoreTargetFound" condition to "False"
					err = conditions.SetRestoreTargetFoundConditionToFalse(inv, i)
					if err != nil {
						return err
					}
					// Now retry after 5 seconds
					klog.Infof("Requeueing Restore Invoker %s %s/%s after 5 seconds....",
						inv.GetTypeMeta().Kind,
						invMeta.Namespace,
						invMeta.Name,
					)
					c.restoreSessionQueue.GetQueue().AddAfter(key, 5*time.Second)
					return nil
				}

				// Restore target exist. So, set "RestoreTargetFound" condition to "True"
				err = conditions.SetRestoreTargetFoundConditionToTrue(inv, i)
				if err != nil {
					return err
				}
			}

			// -------------- Ensure Restore Process for the Target ------------------
			// Take appropriate step to restore based on restore model
			switch c.restorerEntity(tref, inv.GetDriver()) {
			case RestorerInitContainer:
				// The target is kubernetes workload i.e. Deployment, StatefulSet etc.
				// Send event to the respective workload controller. The workload controller will take care of injecting restore init-container.
				err := c.sendEventToWorkloadQueue(
					tref.Kind,
					invMeta.Namespace,
					tref.Name,
				)
				if err != nil {
					msg := fmt.Sprintf("failed to trigger workload controller for %s %s/%s. Reason: %v", tref.Kind, invMeta.Namespace, tref.Name, err)
					klog.Warning(msg)
					return conditions.SetRestorerEnsuredToFalse(inv, &tref, msg)
				}
			case RestorerCSIDriver:
				// VolumeSnapshotter driver has been used. So, ensure VolumeRestorer job
				err := c.ensureVolumeRestorerJob(inv, i)
				if err != nil {
					msg := fmt.Sprintf("failed to ensure volume snapshotter job for %s %s/%s. Reason: %v", tref.Kind, invMeta.Namespace, tref.Name, err)
					klog.Warning(msg)
					return conditions.SetRestorerEnsuredToFalse(inv, &tref, msg)
				}
			case RestorerJob:
				// Restic driver has been used. Ensure restore job.
				err = c.ensureRestoreJob(inv, i)
				if err != nil {
					msg := fmt.Sprintf("failed to ensure restore job for %s %s/%s. Reason: %v", tref.Kind, invMeta.Namespace, tref.Name, err)
					klog.Warning(msg)
					return conditions.SetRestorerEnsuredToFalse(inv, &tref, msg)
				}
			default:
				msg := fmt.Sprintf("unable to identify restorer entity for target %s %s/%s", tref.Kind, invMeta.Namespace, tref.Name)
				klog.Warning(msg)
				return conditions.SetRestorerEnsuredToFalse(inv, &tref, msg)
			}
			msg := fmt.Sprintf("Restorer job/init-container has been ensured successfully for %s %s/%s.", tref.Kind, invMeta.Namespace, tref.Name)
			err = conditions.SetRestorerEnsuredToTrue(inv, &tref, msg)
			if err != nil {
				return err
			}
			return c.initiateTargetRestore(inv, i)
		}
	}
	return nil
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

	rbacOptions, err := c.getRestoreRBACOptions(inv)
	if err != nil {
		return err
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
	addon, err := api_util.ExtractAddonInfo(c.appCatalogClient, targetInfo.Task, targetInfo.Target.Ref, invMeta.Namespace)
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
		// pass offshoot labels to job's pod
		jobTemplate.Labels = meta_util.OverwriteKeys(jobTemplate.Labels, inv.GetLabels())
		jobTemplate.Spec.ImagePullSecrets = imagePullSecrets
		jobTemplate.Spec.ServiceAccountName = rbacOptions.ServiceAccount.Name

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

		restoreJobTemplate.Spec.ImagePullSecrets = imagePullSecrets
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
		func(in *batchv1.Job) *batchv1.Job {
			// set RestoreSession as owner of this Job
			core_util.EnsureOwnerReference(&in.ObjectMeta, owner)

			in.Spec.Template = *jobTemplate
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
	if targetInfo.Hooks != nil && targetInfo.Hooks.PostRestore != nil {
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

	var targetKind, targetName string
	if targetInfo.Target != nil {
		targetKind = targetInfo.Target.Ref.Kind
		targetName = targetInfo.Target.Ref.Name
	}
	podSpec, err := taskResolver.GetPodSpec(inv.GetTypeMeta().Kind, invMeta.Name, targetKind, targetName)
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

	// ensure respective RBAC stuffs
	var serviceAccountName string
	if targetInfo.RuntimeSettings.Pod != nil &&
		targetInfo.RuntimeSettings.Pod.ServiceAccountName != "" {
		// ServiceAccount has been specified, so use it.
		serviceAccountName = targetInfo.RuntimeSettings.Pod.ServiceAccountName
	} else {
		serviceAccountName = getVolumeRestorerServiceAccountName(invMeta.Name, strconv.Itoa(index))
		saMeta := metav1.ObjectMeta{
			Name:      serviceAccountName,
			Namespace: invMeta.Namespace,
			Labels:    inv.GetLabels(),
		}

		_, _, err := core_util.CreateOrPatchServiceAccount(
			context.TODO(),
			c.kubeClient,
			saMeta,
			func(in *core.ServiceAccount) *core.ServiceAccount {
				core_util.EnsureOwnerReference(&in.ObjectMeta, inv.GetOwnerRef())
				in.Labels = inv.GetLabels()
				return in
			},
			metav1.PatchOptions{},
		)
		if err != nil {
			return err
		}
	}

	err := stash_rbac.EnsureVolumeSnapshotRestorerJobRBAC(
		c.kubeClient,
		inv.GetOwnerRef(),
		invMeta.Namespace,
		serviceAccountName,
		inv.GetLabels(),
	)
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
		func(in *batchv1.Job) *batchv1.Job {
			// set restore invoker as owner of this Job
			core_util.EnsureOwnerReference(&in.ObjectMeta, inv.GetOwnerRef())

			in.Labels = inv.GetLabels()
			// pass offshoot labels to job's pod
			in.Spec.Template.Labels = meta_util.OverwriteKeys(in.Spec.Template.Labels, inv.GetLabels())
			in.Spec.Template = *jobTemplate
			in.Spec.Template.Spec.ImagePullSecrets = imagePullSecrets
			in.Spec.Template.Spec.ServiceAccountName = serviceAccountName
			in.Spec.BackoffLimit = pointer.Int32P(0)
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

func getVolumeRestorerServiceAccountName(name, index string) string {
	return meta.ValidNameWithPrefixNSuffix(apis.PrefixStashVolumeSnapshot, strings.ReplaceAll(name, ".", "-"), index)
}

func (c *StashController) restorerEntity(ref api_v1beta1.TargetRef, driver api_v1beta1.Snapshotter) string {
	if util.RestoreModel(ref.Kind) == apis.ModelSidecar {
		return RestorerInitContainer
	} else if driver == api_v1beta1.VolumeSnapshotter {
		return RestorerCSIDriver
	} else {
		return RestorerJob
	}
}

func (c *StashController) requeueRestoreInvoker(inv invoker.RestoreInvoker, key string) error {
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

func globalPostRestoreHookExecuted(inv invoker.RestoreInvoker) bool {
	if inv.GetGlobalHooks() == nil || inv.GetGlobalHooks().PostRestore == nil {
		return true
	}
	return kmapi.HasCondition(inv.GetStatus().Conditions, apis.GlobalPostRestoreHookSucceeded) &&
		kmapi.IsConditionTrue(inv.GetStatus().Conditions, apis.GlobalPostRestoreHookSucceeded)
}

func globalPreRestoreHookExecuted(inv invoker.RestoreInvoker) bool {
	if inv.GetGlobalHooks() == nil || inv.GetGlobalHooks().PreRestore == nil {
		return true
	}
	return kmapi.HasCondition(inv.GetStatus().Conditions, apis.GlobalPreRestoreHookSucceeded) &&
		kmapi.IsConditionTrue(inv.GetStatus().Conditions, apis.GlobalPreRestoreHookSucceeded)
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

func (c *StashController) getRestoreRBACOptions(inv invoker.RestoreInvoker) (stash_rbac.RBACOptions, error) {
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
		ServiceAccount: kmapi.ObjectReference{
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
	return rbacOptions, nil
}

func (c *StashController) initiateTargetRestore(inv invoker.RestoreInvoker, index int) error {
	targetInfo := inv.GetTargetInfo()[index]
	totalHosts, err := c.getTotalHosts(targetInfo.Target, inv.GetObjectMeta().Namespace, inv.GetDriver())
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

func (c *StashController) ensureMetricsPushed(inv invoker.RestoreInvoker) error {
	metricsPushed, err := inv.IsConditionTrue(nil, apis.MetricsPushed)
	if err != nil {
		return err
	}
	if metricsPushed {
		return nil
	}

	// send restore metrics
	metricsOpt := &metrics.MetricsOptions{
		Enabled:        true,
		PushgatewayURL: metrics.GetPushgatewayURL(),
		JobName:        fmt.Sprintf("%s-%s-%s", strings.ToLower(inv.GetTypeMeta().Kind), inv.GetObjectMeta().Namespace, inv.GetObjectMeta().Name),
	}
	// send target specific metrics
	for _, target := range inv.GetStatus().TargetStatus {
		err = metricsOpt.SendRestoreTargetMetrics(c.clientConfig, inv, target.Ref)
		if err != nil {
			return err
		}
	}
	// send restore session metrics
	err = metricsOpt.SendRestoreSessionMetrics(inv)
	if err != nil {
		return err
	}
	return conditions.SetMetricsPushedConditionToTrue(inv, nil)
}

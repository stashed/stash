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
	"stash.appscode.dev/apimachinery/pkg/restic"
	api_util "stash.appscode.dev/apimachinery/pkg/util"
	"stash.appscode.dev/stash/pkg/eventer"
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
	"k8s.io/apimachinery/pkg/util/errors"
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
			// Ensure that the ClusterRoleBindings for this restore invoker has been deleted
			if err := stash_rbac.EnsureClusterRoleBindingDeleted(c.kubeClient, invMeta, inv.GetLabels()); err != nil {
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

	// ======================== Set Global Conditions ============================
	if inv.GetDriver() == api_v1beta1.ResticSnapshotter {
		// Check whether Repository exist or not
		repository, err := inv.GetRepository()
		if err != nil {
			if kerr.IsNotFound(err) {
				klog.Infof("Repository %s/%s for invoker: %s %s/%s does not exist.\nRequeueing after 5 seconds......",
					inv.GetRepoRef().Namespace,
					inv.GetRepoRef().Name,
					inv.GetTypeMeta().Kind,
					invMeta.Namespace,
					invMeta.Name,
				)
				err2 := conditions.SetRepositoryFoundConditionToFalse(inv)
				if err2 != nil {
					return err2
				}
				return c.requeueRestoreInvoker(inv, key, 5*time.Second)
			}
			err2 := conditions.SetRepositoryFoundConditionToUnknown(inv, err)
			return errors.NewAggregate([]error{err, err2})
		}
		err = conditions.SetRepositoryFoundConditionToTrue(inv)
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
				return c.requeueRestoreInvoker(inv, key, 5*time.Second)
			}
			err2 := conditions.SetBackendSecretFoundConditionToUnknown(inv, secret.Name, err)
			return errors.NewAggregate([]error{err, err2})
		}
		err = conditions.SetBackendSecretFoundConditionToTrue(inv, secret.Name)
		if err != nil {
			return err
		}
	}

	// ================= Don't Process Completed Invoker ===========================
	status := inv.GetStatus()
	if status.Phase == api_v1beta1.RestoreFailed ||
		status.Phase == api_v1beta1.RestoreSucceeded ||
		status.Phase == api_v1beta1.RestorePhaseUnknown {
		klog.Infof("Skipping processing %s %s/%s. Reason: phase is %q.",
			inv.GetTypeMeta().Kind,
			invMeta.Namespace,
			invMeta.Name,
			status.Phase,
		)
		return nil
	}

	// ensure that target phases are up-to-date
	err = c.ensureRestoreTargetPhases(inv)
	if err != nil {
		return err
	}
	// check whether restore process has completed or running and set it's phase accordingly
	phase, err := c.getRestorePhase(inv.GetStatus())

	// ==================== Execute Global PostRestore Hooks ===========================
	// if the restore process has completed(Failed or Succeeded or Unknown), then execute global postRestore hook if not yet executed
	if restoreCompleted(phase) && !globalPostRestoreHookExecuted(inv) {
		hookErr := util.ExecuteHook(c.clientConfig, inv.GetGlobalHooks(), apis.PostRestoreHook, os.Getenv("MY_POD_NAME"), os.Getenv("MY_POD_NAMESPACE"))
		if hookErr != nil {
			condErr := conditions.SetGlobalPostRestoreHookSucceededConditionToFalse(inv, hookErr)
			// set restore phase failed
			return c.setRestorePhaseFailed(inv, errors.NewAggregate([]error{err, hookErr, condErr}))
		}
		condErr := conditions.SetGlobalPostRestoreHookSucceededConditionToTrue(inv)
		if condErr != nil {
			return condErr
		}
	}

	// ==================== Set Restore Invoker Phase ======================================
	if phase == api_v1beta1.RestoreFailed {
		// one or more target has failed to complete their restore process.
		// mark entire restore process as failure.
		// now, set restore phase "Failed", create an event. and send respective metrics.
		return c.setRestorePhaseFailed(inv, err)
	} else if phase == api_v1beta1.RestorePhaseUnknown {
		return c.setRestorePhaseUnknown(inv, err)
	} else if phase == api_v1beta1.RestoreSucceeded {
		// all targets has completed their restore process successfully.
		// now, set restore phase "Succeeded", create an event, and send respective metrics .
		return c.setRestorePhaseSucceeded(inv)
	}

	// ==================== Execute Global PreRestore Hook =====================
	// if global preRestore hook exist and not executed yet, then execute the preRestoreHook
	if !globalPreRestoreHookExecuted(inv) {
		hookErr := util.ExecuteHook(c.clientConfig, inv.GetGlobalHooks(), apis.PreBackupHook, os.Getenv("MY_POD_NAME"), os.Getenv("MY_POD_NAMESPACE"))
		if hookErr != nil {
			condErr := conditions.SetGlobalPreRestoreHookSucceededConditionToFalse(inv, hookErr)
			return c.setRestorePhaseFailed(inv, errors.NewAggregate([]error{hookErr, condErr}))
		}
		condErr := conditions.SetGlobalPreRestoreHookSucceededConditionToTrue(inv)
		if condErr != nil {
			return condErr
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
				// restore order is sequential and the current target is not yet to be executed.
				// so, set its phase to "Pending".
				err = c.setRestoreTargetPhasePending(inv, i)
				if err != nil {
					return err
				}
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
					// Set the "RestoreTargetFound" condition to "Unknown"
					cerr := conditions.SetRestoreTargetFoundConditionToUnknown(inv, i, err)
					return errors.NewAggregate([]error{err, cerr})
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
					return c.handleWorkloadControllerTriggerFailure(invokerRef, err)
				}
			case RestorerCSIDriver:
				// VolumeSnapshotter driver has been used. So, ensure VolumeRestorer job
				err := c.ensureVolumeRestorerJob(inv, i)
				if err != nil {
					// Failed to ensure VolumeRestorer job. So, set "RestoreJobCreated" condition to "False"
					cerr := conditions.SetRestoreJobCreatedConditionToFalse(inv, &tref, err)
					return c.handleRestoreJobCreationFailure(inv, errors.NewAggregate([]error{err, cerr}))
				}
				// Successfully created VolumeRestorer job. So, set "RestoreJobCreated" condition to "True"
				cerr := conditions.SetRestoreJobCreatedConditionToTrue(inv, &tref)
				if cerr != nil {
					return cerr
				}
			case RestorerJob:
				// Restic driver has been used. Ensure restore job.
				err = c.ensureRestoreJob(inv, i)
				if err != nil {
					// Failed to ensure restorer job. So, set "RestoreJobCreated" condition to "False"
					cerr := conditions.SetRestoreJobCreatedConditionToFalse(inv, &tref, err)
					// Set RestoreSession phase "Failed" and send prometheus metrics.
					return c.handleRestoreJobCreationFailure(inv, errors.NewAggregate([]error{err, cerr}))
				}
				err = conditions.SetRestoreJobCreatedConditionToTrue(inv, &tref)
				if err != nil {
					return err
				}
			default:
				return fmt.Errorf("unable to idenitfy restorer entity")
			}
			// Set target phase "Running"
			err = c.setRestoreTargetPhaseRunning(inv, i)
			if err != nil {
				return err
			}
		}
	}

	// Restorer entity has been ensured. Set RestoreSession phase to "Running".
	return c.setRestorePhaseRunning(inv)
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

	// Ensure respective RBAC and PSP stuff.
	var serviceAccountName string
	if targetInfo.RuntimeSettings.Pod != nil &&
		targetInfo.RuntimeSettings.Pod.ServiceAccountName != "" {
		// ServiceAccount has been specified, so use it.
		serviceAccountName = targetInfo.RuntimeSettings.Pod.ServiceAccountName
	} else {
		// ServiceAccount hasn't been specified. so create new one with name generated from the invoker name.
		serviceAccountName = getRestoreJobServiceAccountName(invMeta.Name, strconv.Itoa(index))
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

	psps, err := c.getRestoreJobPSPNames(targetInfo.Task)
	if err != nil {
		return err
	}

	err = stash_rbac.EnsureRestoreJobRBAC(
		c.kubeClient,
		inv.GetOwnerRef(),
		invMeta.Namespace,
		serviceAccountName,
		psps,
		inv.GetLabels(),
	)
	if err != nil {
		return err
	}
	// Give the ServiceAccount permission to send request to the license handler
	err = stash_rbac.EnsureLicenseReaderClusterRoleBinding(c.kubeClient, inv.GetOwnerRef(), invMeta.Namespace, serviceAccountName, inv.GetLabels())
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
		jobTemplate.Spec.ServiceAccountName = serviceAccountName

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
		restoreJobTemplate.Spec.ServiceAccountName = serviceAccountName

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

func (c *StashController) setRestorePhaseRunning(inv invoker.RestoreInvoker) error {
	// update restore invoker status
	err := inv.UpdateStatus(invoker.RestoreInvokerStatus{
		Phase: api_v1beta1.RestoreRunning,
	})
	if err != nil {
		return err
	}

	// crate event against the restore invoker
	err = inv.CreateEvent(
		core.EventTypeNormal,
		"",
		eventer.EventReasonRestoreRunning,
		fmt.Sprintf("restore has been started for %s %s/%s",
			inv.GetTypeMeta().Kind,
			inv.GetObjectMeta().Namespace,
			inv.GetObjectMeta().Name,
		),
	)
	return err
}

func (c *StashController) setRestorePhaseSucceeded(inv invoker.RestoreInvoker) error {
	var err error
	invMeta := inv.GetObjectMeta()
	// total restore session duration is the difference between the time when restore invoker was created and when it completed
	sessionDuration := time.Since(invMeta.CreationTimestamp.Time)
	// update restore invoker status
	err = inv.UpdateStatus(invoker.RestoreInvokerStatus{
		Phase:           api_v1beta1.RestoreSucceeded,
		SessionDuration: sessionDuration.Round(time.Second).String(),
	})
	if err != nil {
		return err
	}

	// crate event against the restore invoker
	err = inv.CreateEvent(
		core.EventTypeNormal,
		"",
		eventer.EventReasonRestoreSucceeded,
		"Restore has been completed successfully",
	)

	// if there is any error during writing the event, log i. we have to send metrics even if we fail to write the event.
	if err != nil {
		klog.Errorf("failed to write event in %s %s/%s. Reason: %v",
			inv.GetTypeMeta().Kind,
			invMeta.Namespace,
			invMeta.Name,
			err,
		)
	}
	// send restore metrics
	metricsOpt := &restic.MetricsOptions{
		Enabled:        true,
		PushgatewayURL: apis.PushgatewayLocalURL,
		JobName:        fmt.Sprintf("%s-%s-%s", strings.ToLower(inv.GetTypeMeta().Kind), invMeta.Namespace, invMeta.Name),
	}
	// send target specific metrics
	for _, target := range inv.GetStatus().TargetStatus {
		metricErr := metricsOpt.SendRestoreTargetMetrics(c.clientConfig, inv, target.Ref)
		if err != nil {
			return metricErr
		}
	}
	// send restore session metrics
	return metricsOpt.SendRestoreSessionMetrics(inv)
}

func (c *StashController) setRestorePhaseFailed(inv invoker.RestoreInvoker, restoreErr error) error {
	var err error
	invMeta := inv.GetObjectMeta()
	// total restore session duration is the difference between the time when restore invoker was created and when it completed
	sessionDuration := time.Since(invMeta.CreationTimestamp.Time)

	// update restore invoker status
	err = inv.UpdateStatus(invoker.RestoreInvokerStatus{
		Phase:           api_v1beta1.RestoreFailed,
		SessionDuration: sessionDuration.Round(time.Second).String(),
	})
	if err != nil {
		return err
	}

	// crate event against the restore invoker
	err = inv.CreateEvent(
		core.EventTypeWarning,
		"",
		eventer.EventReasonRestoreFailed,
		fmt.Sprintf("Restore has failed to complete. Reason: %v", restoreErr),
	)

	// if there is any error during writing the event, log i. we have to send metrics even if we fail to write the event.
	if err != nil {
		klog.Errorf("failed to write event in %s %s/%s. Reason: %v",
			inv.GetTypeMeta().Kind,
			invMeta.Namespace,
			invMeta.Name,
			err,
		)
	}
	// send restore metrics
	metricsOpt := &restic.MetricsOptions{
		Enabled:        true,
		PushgatewayURL: apis.PushgatewayLocalURL,
		JobName:        fmt.Sprintf("%s-%s-%s", strings.ToLower(inv.GetTypeMeta().Kind), invMeta.Namespace, invMeta.Name),
	}
	// send target specific metrics
	for _, target := range inv.GetStatus().TargetStatus {
		metricErr := metricsOpt.SendRestoreTargetMetrics(c.clientConfig, inv, target.Ref)
		if err != nil {
			return errors.NewAggregate([]error{restoreErr, metricErr})
		}
	}
	// send restore session metrics
	err = metricsOpt.SendRestoreSessionMetrics(inv)
	return errors.NewAggregate([]error{restoreErr, err})
}

func (c *StashController) setRestorePhaseUnknown(inv invoker.RestoreInvoker, restoreErr error) error {
	var err error
	invMeta := inv.GetObjectMeta()
	// total restore session duration is the difference between the time when restore invoker was created and when it completed
	sessionDuration := time.Since(invMeta.CreationTimestamp.Time)

	// update restore invoker status
	err = inv.UpdateStatus(invoker.RestoreInvokerStatus{
		Phase:           api_v1beta1.RestorePhaseUnknown,
		SessionDuration: sessionDuration.Round(time.Second).String(),
	})
	if err != nil {
		return err
	}

	// crate event against the restore invoker
	err = inv.CreateEvent(
		core.EventTypeWarning,
		"",
		eventer.EventReasonRestorePhaseUnknown,
		fmt.Sprintf("Unable to ensure whether restore has completed or not. Reason: %v", restoreErr),
	)

	// if there is any error during writing the event, log i. we have to send metrics even if we fail to write the event.
	if err != nil {
		klog.Errorf("failed to write event in %s %s/%s. Reason: %v",
			inv.GetTypeMeta().Kind,
			invMeta.Namespace,
			invMeta.Name,
			err,
		)
	}
	// send restore metrics
	metricsOpt := &restic.MetricsOptions{
		Enabled:        true,
		PushgatewayURL: apis.PushgatewayLocalURL,
		JobName:        fmt.Sprintf("%s-%s-%s", strings.ToLower(inv.GetTypeMeta().Kind), invMeta.Namespace, invMeta.Name),
	}
	// send target specific metrics
	for _, target := range inv.GetStatus().TargetStatus {
		metricErr := metricsOpt.SendRestoreTargetMetrics(c.clientConfig, inv, target.Ref)
		if err != nil {
			return errors.NewAggregate([]error{restoreErr, metricErr})
		}
	}
	// send restore session metrics
	err = metricsOpt.SendRestoreSessionMetrics(inv)
	return errors.NewAggregate([]error{restoreErr, err})
}

func (c *StashController) setRestoreTargetPhasePending(inv invoker.RestoreInvoker, index int) error {
	targetInfo := inv.GetTargetInfo()[index]
	err := inv.UpdateStatus(invoker.RestoreInvokerStatus{
		TargetStatus: []api_v1beta1.RestoreMemberStatus{
			{
				Ref:   targetInfo.Target.Ref,
				Phase: api_v1beta1.TargetRestorePending,
			},
		},
	})
	return err
}

func (c *StashController) setRestoreTargetPhaseRunning(inv invoker.RestoreInvoker, index int) error {
	targetInfo := inv.GetTargetInfo()[index]
	totalHosts, err := c.getTotalHosts(targetInfo.Target, inv.GetObjectMeta().Namespace, inv.GetDriver())
	if err != nil {
		return err
	}
	err = inv.UpdateStatus(invoker.RestoreInvokerStatus{
		TargetStatus: []api_v1beta1.RestoreMemberStatus{
			{
				Ref:        targetInfo.Target.Ref,
				TotalHosts: totalHosts,
				Phase:      api_v1beta1.TargetRestoreRunning,
			},
		},
	})
	return err
}

func (c *StashController) getRestorePhase(status invoker.RestoreInvokerStatus) (api_v1beta1.RestorePhase, error) {
	// If the Phase is empty or "Pending" then return it. controller will process accordingly
	if status.Phase == "" || status.Phase == api_v1beta1.RestorePending {
		return api_v1beta1.RestorePending, nil
	}

	// If any of the target fail, then mark the entire restore process as "Failed".
	// Mark the entire restore process "Succeeded" only and if only the restore of all targets has succeeded.
	// Otherwise, mark the restore process as "Running".
	completedTargets := 0
	var errList []error
	for _, target := range status.TargetStatus {
		if target.Phase == api_v1beta1.TargetRestoreSucceeded ||
			target.Phase == api_v1beta1.TargetRestoreFailed {
			completedTargets = completedTargets + 1
		}
		if target.Phase == api_v1beta1.TargetRestoreFailed {
			errList = append(errList, fmt.Errorf("restore failed for target: %s/%s", target.Ref.Kind, target.Ref.Name))
		}
		if target.Phase == api_v1beta1.TargetRestoreRunning {
			return api_v1beta1.RestoreRunning, nil
		}
	}

	if errList != nil {
		return api_v1beta1.RestoreFailed, errors.NewAggregate(errList)
	}

	// check if any of the target phase is "Unknown". if any of their phase is "Unknown", then consider entire restore process phase is unknown.
	for _, target := range status.TargetStatus {
		if target.Phase == api_v1beta1.TargetRestorePhaseUnknown {
			return api_v1beta1.RestorePhaseUnknown, fmt.Errorf("restore phase is 'Unknown' for target: %s/%s", target.Ref.Kind, target.Ref.Name)
		}
	}

	if completedTargets != len(status.TargetStatus) {
		return api_v1beta1.RestoreRunning, nil
	}

	// Restore has been completed successfully for all targets.
	return api_v1beta1.RestoreSucceeded, nil
}

func (c *StashController) handleRestoreJobCreationFailure(inv invoker.RestoreInvoker, restoreErr error) error {
	invMeta := inv.GetObjectMeta()
	klog.Errorf("failed to ensure restore job for %s %s/%s. Reason: %v",
		inv.GetTypeMeta().Kind,
		invMeta.Namespace,
		invMeta.Name,
		restoreErr,
	)

	// write event to RestoreSession
	err := inv.CreateEvent(
		core.EventTypeWarning,
		"",
		eventer.EventReasonRestoreJobCreationFailed,
		fmt.Sprintf("failed to create restore job. Reason: %v", restoreErr),
	)
	if err != nil {
		klog.Errorf("failed to write event for %s %s/%s. Reason: ",
			inv.GetTypeMeta().Kind,
			invMeta.Namespace,
			invMeta.Name,
		)
	}

	// set RestoreSession phase failed
	return c.setRestorePhaseFailed(inv, restoreErr)
}

func getRestoreJobName(invokerMeta metav1.ObjectMeta, suffix string) string {
	return meta.ValidNameWithPrefixNSuffix(apis.PrefixStashRestore, strings.ReplaceAll(invokerMeta.Name, ".", "-"), suffix)
}

func getRestoreJobServiceAccountName(name, suffix string) string {
	return meta.ValidNameWithPrefixNSuffix(apis.PrefixStashRestore, strings.ReplaceAll(name, ".", "-"), suffix)
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

func restoreCompleted(phase api_v1beta1.RestorePhase) bool {
	return phase == api_v1beta1.RestoreFailed ||
		phase == api_v1beta1.RestoreSucceeded ||
		phase == api_v1beta1.RestorePhaseUnknown
}

func (c *StashController) requeueRestoreInvoker(inv invoker.RestoreInvoker, key string, delay time.Duration) error {
	invTypeMeta := inv.GetTypeMeta()
	switch invTypeMeta.Kind {
	case api_v1beta1.ResourceKindRestoreSession:
		c.rsQueue.GetQueue().AddAfter(key, delay)
	default:
		return fmt.Errorf("unable to requeue. Reason: Restore invoker %s %s is not supported",
			invTypeMeta.APIVersion,
			invTypeMeta.Kind,
		)
	}
	return nil
}

func (c *StashController) ensureRestoreTargetPhases(inv invoker.RestoreInvoker) error {
	status := inv.GetStatus()
	targetStats := status.TargetStatus
	for i, target := range status.TargetStatus {
		if target.TotalHosts == nil {
			targetStats[i].Phase = api_v1beta1.TargetRestorePending
			continue
		}
		// check if any host failed to restore, running, or it's phase 'Unknown'
		anyHostFailed := false
		anyHostPhaseUnknown := false
		anyHostRunning := false
		for _, hostStats := range target.Stats {
			if hostStats.Phase == api_v1beta1.HostRestoreFailed {
				anyHostFailed = true
				break
			}
			if hostStats.Phase == api_v1beta1.HostRestoreUnknown {
				anyHostPhaseUnknown = true
				break
			}
			if hostStats.Phase == api_v1beta1.HostRestoreRunning {
				anyHostRunning = true
				break
			}
		}
		// if any host fail to restore, the overall target phase should be "Failed"
		if anyHostFailed {
			targetStats[i].Phase = api_v1beta1.TargetRestoreFailed
			continue
		}
		// if any host's restore phase is 'Unknown', the overall target phase should be "Unknown"
		if anyHostPhaseUnknown {
			targetStats[i].Phase = api_v1beta1.TargetRestorePhaseUnknown
			continue
		}
		// Restore should be succeeded only if all the conditions of this restore target are true
		allConditionTrue := true
		for _, c := range target.Conditions {
			if c.Status == core.ConditionFalse {
				allConditionTrue = false
				break
			}
		}
		// if some host hasn't completed their restore yet, phase should be "Running"
		if !anyHostRunning && *target.TotalHosts <= int32(len(target.Stats)) && allConditionTrue {
			targetStats[i].Phase = api_v1beta1.TargetRestoreSucceeded
			continue
		}
		// all host completed their restore process and none of them failed. so, phase should be "Succeeded".
		targetStats[i].Phase = api_v1beta1.TargetRestoreRunning
	}
	return inv.UpdateStatus(invoker.RestoreInvokerStatus{TargetStatus: targetStats})
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
	for _, target := range inv.GetStatus().TargetStatus {
		if invoker.TargetMatched(target.Ref, targetRef) {
			return target.Phase == api_v1beta1.TargetRestoreRunning ||
				target.Phase == api_v1beta1.TargetRestoreSucceeded ||
				target.Phase == api_v1beta1.TargetRestoreFailed ||
				target.Phase == api_v1beta1.TargetRestorePhaseUnknown
		}
	}
	return false
}

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
	"reflect"
	"strings"
	"time"

	"stash.appscode.dev/apimachinery/apis"
	"stash.appscode.dev/apimachinery/apis/stash"
	api_v1alpha1 "stash.appscode.dev/apimachinery/apis/stash/v1alpha1"
	api_v1beta1 "stash.appscode.dev/apimachinery/apis/stash/v1beta1"
	v1beta1_util "stash.appscode.dev/apimachinery/client/clientset/versioned/typed/stash/v1beta1/util"
	"stash.appscode.dev/apimachinery/pkg/conditions"
	"stash.appscode.dev/apimachinery/pkg/docker"
	"stash.appscode.dev/apimachinery/pkg/invoker"
	"stash.appscode.dev/stash/pkg/eventer"
	"stash.appscode.dev/stash/pkg/util"

	"gomodules.xyz/pointer"
	batchv1beta1 "k8s.io/api/batch/v1beta1"
	core "k8s.io/api/core/v1"
	kerr "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/errors"
	"k8s.io/client-go/tools/cache"
	"k8s.io/klog/v2"
	batch_v1beta1_util "kmodules.xyz/client-go/batch/v1beta1"
	core_util "kmodules.xyz/client-go/core/v1"
	meta2 "kmodules.xyz/client-go/meta"
	meta_util "kmodules.xyz/client-go/meta"
	"kmodules.xyz/client-go/tools/queue"
	ofst_util "kmodules.xyz/offshoot-api/util"
	"kmodules.xyz/webhook-runtime/admission"
	hooks "kmodules.xyz/webhook-runtime/admission/v1beta1"
	webhook "kmodules.xyz/webhook-runtime/admission/v1beta1/generic"
	workload_api "kmodules.xyz/webhook-runtime/apis/workload/v1"
)

const requeueTimeInterval = 5 * time.Second

func (c *StashController) NewBackupConfigurationWebhook() hooks.AdmissionHook {
	return webhook.NewGenericWebhook(
		schema.GroupVersionResource{
			Group:    "admission.stash.appscode.com",
			Version:  "v1beta1",
			Resource: "backupconfigurationvalidators",
		},
		"backupconfigurationvalidator",
		[]string{stash.GroupName},
		api_v1beta1.SchemeGroupVersion.WithKind(api_v1beta1.ResourceKindBackupConfiguration),
		nil,
		&admission.ResourceHandlerFuncs{
			CreateFunc: func(obj runtime.Object) (runtime.Object, error) {
				bc := obj.(*api_v1beta1.BackupConfiguration)
				return nil, c.validateBackupConfiguration(bc)
			},
			UpdateFunc: func(oldObj, newObj runtime.Object) (runtime.Object, error) {
				newBc := newObj.(*api_v1beta1.BackupConfiguration)

				if newBc.ObjectMeta.DeletionTimestamp != nil {
					return nil, nil
				}

				oldBc := oldObj.(*api_v1beta1.BackupConfiguration)

				if oldBc.Status.Phase == api_v1beta1.BackupInvokerReady && isTargetUpdated(*oldBc, *newBc) {
					return nil, fmt.Errorf("Updating target is forbidden when BackupConfiguration is in READY state")
				}

				return nil, c.validateBackupConfiguration(newBc)
			},
		},
	)
}

func isTargetUpdated(oldBc api_v1beta1.BackupConfiguration, newBc api_v1beta1.BackupConfiguration) bool {
	return !reflect.DeepEqual(oldBc.Spec.Target, newBc.Spec.Target)
}

func (c *StashController) validateBackupConfiguration(bc *api_v1beta1.BackupConfiguration) error {
	if bc.Spec.Target != nil {
		err := verifyCrossNamespacePermission(bc.ObjectMeta, bc.Spec.Target.Ref, bc.Spec.Task.Name)
		if err != nil {
			return err
		}
	}
	return c.validateAgainstUsagePolicy(bc.Spec.Repository, bc.Namespace)
}

func (c *StashController) initBackupConfigurationWatcher() {
	c.bcInformer = c.stashInformerFactory.Stash().V1beta1().BackupConfigurations().Informer()
	c.bcQueue = queue.New(api_v1beta1.ResourceKindBackupConfiguration, c.MaxNumRequeues, c.NumThreads, c.runBackupConfigurationProcessor)
	if c.auditor != nil {
		c.bcInformer.AddEventHandler(c.auditor.ForGVK(api_v1beta1.SchemeGroupVersion.WithKind(api_v1beta1.ResourceKindBackupConfiguration)))
	}
	c.bcInformer.AddEventHandler(queue.NewEventHandler(c.bcQueue.GetQueue(), func(oldObj, newObj interface{}) bool {
		bc := newObj.(*api_v1beta1.BackupConfiguration)
		desiredPhase := invoker.CalculateBackupInvokerPhase(bc.Spec.Driver, bc.Status.Conditions)
		return bc.GetDeletionTimestamp() != nil ||
			!meta_util.MustAlreadyReconciled(bc) ||
			bc.Status.Phase != desiredPhase ||
			bc.Status.Phase != api_v1beta1.BackupInvokerReady
	}, core.NamespaceAll))
	c.bcLister = c.stashInformerFactory.Stash().V1beta1().BackupConfigurations().Lister()
}

// syncToStdout is the business logic of the controller. In this controller it simply prints
// information about the deployment to stdout. In case an error happened, it has to simply return the error.
// The retry logic should not be part of the business logic.
func (c *StashController) runBackupConfigurationProcessor(key string) error {
	obj, exists, err := c.bcInformer.GetIndexer().GetByKey(key)
	if err != nil {
		klog.Errorf("Fetching object with key %s from store failed with %v", key, err)
		return err
	}
	if !exists {
		klog.Warningf("BackupConfiguration %s does not exit anymore\n", key)
		return nil
	}

	backupConfiguration := obj.(*api_v1beta1.BackupConfiguration)
	klog.Infof("Sync/Add/Update for BackupConfiguration %s", backupConfiguration.GetName())
	// process syc/add/update event
	inv := invoker.NewBackupConfigurationInvoker(c.stashClient, backupConfiguration)
	err = c.applyBackupInvokerReconciliationLogic(inv, key)
	if err != nil {
		return err
	}

	// We have successfully completed respective stuffs for the current state of this resource.
	// Hence, let's set observed generation as same as the current generation.
	_, err = v1beta1_util.UpdateBackupConfigurationStatus(
		context.TODO(),
		c.stashClient.StashV1beta1(),
		backupConfiguration.ObjectMeta,
		func(in *api_v1beta1.BackupConfigurationStatus) (types.UID, *api_v1beta1.BackupConfigurationStatus) {
			in.ObservedGeneration = backupConfiguration.Generation
			return backupConfiguration.UID, in
		},
		metav1.UpdateOptions{},
	)
	return err
}

func (c *StashController) applyBackupInvokerReconciliationLogic(inv invoker.BackupInvoker, key string) error {
	// check if backup invoker is being deleted. if it is being deleted then delete respective resources.
	invMeta := inv.GetObjectMeta()
	invokerRef, err := inv.GetObjectRef()
	if err != nil {
		return err
	}
	if invMeta.DeletionTimestamp != nil {
		if core_util.HasFinalizer(invMeta, api_v1beta1.StashKey) {
			err := c.cleanupBackupInvokerOffshoots(inv, invokerRef)
			if err != nil {
				return err
			}

			// Remove finalizer
			return inv.RemoveFinalizer()
		}
		return nil
	}

	if err := inv.AddFinalizer(); err != nil {
		return err
	}

	shouldRequeue, err := c.validateDriverRequirements(inv)
	if err != nil {
		return err
	}
	if shouldRequeue {
		return c.requeueBackupInvoker(inv, key)
	}

	someTargetMissing := false
	for _, targetInfo := range inv.GetTargetInfo() {
		if targetInfo.Target != nil {
			tref := targetInfo.Target.Ref
			targetExist, err := util.IsTargetExist(c.clientConfig, tref)
			if err != nil {
				klog.Errorf("Failed to check whether Kind: %s Namespace: %s Name: %s exist or not. Reason: %v.",
					tref.Kind,
					tref.Namespace,
					tref.Name,
					err.Error(),
				)
				// Set the "BackupTargetFound" condition to "Unknown"
				cerr := conditions.SetBackupTargetFoundConditionToUnknown(inv, tref, err)
				return errors.NewAggregate([]error{err, cerr})
			}
			if !targetExist {
				// Target does not exist. Log the information.
				someTargetMissing = true
				klog.Infof("Backup target %s %s/%s does not exist.",
					tref.Kind,
					tref.Namespace,
					tref.Name)
				err = conditions.SetBackupTargetFoundConditionToFalse(inv, tref)
				if err != nil {
					return err
				}
				// Process next target.
				continue
			}
			// Backup target exist. So, set "BackupTargetFound" condition to "True"
			err = conditions.SetBackupTargetFoundConditionToTrue(inv, tref)
			if err != nil {
				return err
			}
			// For sidecar model, ensure the stash sidecar
			if inv.GetDriver() == api_v1beta1.ResticSnapshotter && util.BackupModel(tref.Kind, targetInfo.Task.Name) == apis.ModelSidecar {
				err := c.EnsureV1beta1Sidecar(tref)
				if err != nil {
					return c.handleWorkloadControllerTriggerFailure(invokerRef, err)
				}
			}
		}
	}
	// If some backup targets are missing, then retry after some time.
	if someTargetMissing {
		klog.Infof("Some targets are missing for backup invoker: %s %s/%s.\nRequeueing after 5 seconds......",
			inv.GetTypeMeta().Kind,
			invMeta.Namespace,
			invMeta.Name,
		)
		return c.requeueBackupInvoker(inv, key)
	}
	// create a CronJob that will create BackupSession on each schedule
	err = c.EnsureBackupTriggeringCronJob(inv)
	if err != nil {
		// Failed to ensure the backup triggering CronJob. So, set "CronJobCreated" condition to "False"
		cerr := conditions.SetCronJobCreatedConditionToFalse(inv, err)
		return c.handleCronJobCreationFailure(invokerRef, errors.NewAggregate([]error{err, cerr}))
	}

	// Successfully ensured the backup triggering CronJob. So, set "CronJobCreated" condition to "True"
	return conditions.SetCronJobCreatedConditionToTrue(inv)
}

// EnsureV1beta1SidecarDeleted send an event to workload respective controller
// the workload controller will take care of removing respective sidecar
func (c *StashController) EnsureV1beta1SidecarDeleted(targetRef api_v1beta1.TargetRef) error {
	return c.sendEventToWorkloadQueue(
		targetRef.Kind,
		targetRef.Namespace,
		targetRef.Name,
	)
}

// EnsureV1beta1Sidecar send an event to workload respective controller
// the workload controller will take care of injecting backup sidecar
func (c *StashController) EnsureV1beta1Sidecar(targetRef api_v1beta1.TargetRef) error {
	return c.sendEventToWorkloadQueue(
		targetRef.Kind,
		targetRef.Namespace,
		targetRef.Name,
	)
}

func (c *StashController) sendEventToWorkloadQueue(kind, namespace, resourceName string) error {
	switch kind {
	case workload_api.KindDeployment:
		if resource, err := c.dpLister.Deployments(namespace).Get(resourceName); err == nil {
			key, err := cache.MetaNamespaceKeyFunc(resource)
			if err == nil {
				c.dpQueue.GetQueue().Add(key)
			}
			return err
		}
	case workload_api.KindDaemonSet:
		if resource, err := c.dsLister.DaemonSets(namespace).Get(resourceName); err == nil {
			key, err := cache.MetaNamespaceKeyFunc(resource)
			if err == nil {
				c.dsQueue.GetQueue().Add(key)
			}
			return err
		}
	case workload_api.KindStatefulSet:
		if resource, err := c.ssLister.StatefulSets(namespace).Get(resourceName); err == nil {
			key, err := cache.MetaNamespaceKeyFunc(resource)
			if err == nil {
				c.ssQueue.GetQueue().Add(key)
			}
		}
	case workload_api.KindReplicationController:
		if resource, err := c.rcLister.ReplicationControllers(namespace).Get(resourceName); err == nil {
			key, err := cache.MetaNamespaceKeyFunc(resource)
			if err == nil {
				c.rcQueue.GetQueue().Add(key)
			}
			return err
		}
	case workload_api.KindReplicaSet:
		if resource, err := c.rsLister.ReplicaSets(namespace).Get(resourceName); err == nil {
			key, err := cache.MetaNamespaceKeyFunc(resource)
			if err == nil {
				c.rsQueue.GetQueue().Add(key)
			}
			return err
		}
	case workload_api.KindDeploymentConfig:
		if c.ocClient != nil && c.dcLister != nil {
			if resource, err := c.dcLister.DeploymentConfigs(namespace).Get(resourceName); err == nil {
				key, err := cache.MetaNamespaceKeyFunc(resource)
				if err == nil {
					c.dcQueue.GetQueue().Add(key)
				}
				return err
			}
		}
	}
	return nil
}

// EnsureBackupTriggeringCronJob creates a Kubernetes CronJob for the respective backup invoker
// the CornJob will create a BackupSession object in each schedule
// respective BackupSession controller will watch this BackupSession object and take backup instantly
func (c *StashController) EnsureBackupTriggeringCronJob(inv invoker.BackupInvoker) error {
	invMeta := inv.GetObjectMeta()
	runtimeSettings := inv.GetRuntimeSettings()
	ownerRef := inv.GetOwnerRef()

	image := docker.Docker{
		Registry: c.DockerRegistry,
		Image:    c.StashImage,
		Tag:      c.StashImageTag,
	}

	meta := metav1.ObjectMeta{
		Name:      getBackupCronJobName(inv),
		Namespace: invMeta.Namespace,
		Labels:    inv.GetLabels(),
	}

	rbacOptions, err := c.getBackupRBACOptions(inv, nil)
	if err != nil {
		return err
	}
	rbacOptions.PodSecurityPolicyNames = c.getBackupSessionCronJobPSPNames()

	if runtimeSettings.Pod != nil && runtimeSettings.Pod.ServiceAccountName != "" {
		rbacOptions.ServiceAccount.Name = runtimeSettings.Pod.ServiceAccountName
	}

	err = rbacOptions.EnsureCronJobRBAC(meta.Name)
	if err != nil {
		return err
	}

	// if the Stash is using a private registry, then ensure the image pull secrets
	var imagePullSecrets []core.LocalObjectReference
	if c.ImagePullSecrets != nil {
		imagePullSecrets, err = c.ensureImagePullSecrets(invMeta, ownerRef)
		if err != nil {
			return err
		}
	}
	_, _, err = batch_v1beta1_util.CreateOrPatchCronJob(
		context.TODO(),
		c.kubeClient,
		meta,
		func(in *batchv1beta1.CronJob) *batchv1beta1.CronJob {
			// set backup invoker object as cron-job owner
			core_util.EnsureOwnerReference(&in.ObjectMeta, ownerRef)

			in.Spec.Schedule = inv.GetSchedule()
			in.Spec.Suspend = pointer.BoolP(inv.IsPaused()) // this ensure that the CronJob is suspended when the backup invoker is paused.
			in.Spec.JobTemplate.Labels = meta_util.OverwriteKeys(in.Spec.JobTemplate.Labels, inv.GetLabels())
			// ensure that job gets deleted on completion
			in.Spec.JobTemplate.Labels[apis.KeyDeleteJobOnCompletion] = apis.AllowDeletingJobOnCompletion
			// pass offshoot labels to the CronJob's pod
			in.Spec.JobTemplate.Spec.Template.Labels = meta_util.OverwriteKeys(in.Spec.JobTemplate.Spec.Template.Labels, inv.GetLabels())

			container := core.Container{
				Name:            apis.StashCronJobContainer,
				ImagePullPolicy: core.PullIfNotPresent,
				Image:           image.ToContainerImage(),
				Args: []string{
					"create-backupsession",
					fmt.Sprintf("--invoker-name=%s", ownerRef.Name),
					fmt.Sprintf("--invoker-kind=%s", ownerRef.Kind),
				},
			}
			// only apply the container level runtime settings that make sense for the CronJob
			if runtimeSettings.Container != nil {
				container.Resources = runtimeSettings.Container.Resources
				container.Env = runtimeSettings.Container.Env
				container.EnvFrom = runtimeSettings.Container.EnvFrom
				container.SecurityContext = runtimeSettings.Container.SecurityContext
			}

			in.Spec.JobTemplate.Spec.Template.Spec.Containers = core_util.UpsertContainer(
				in.Spec.JobTemplate.Spec.Template.Spec.Containers, container)
			in.Spec.JobTemplate.Spec.Template.Spec.RestartPolicy = core.RestartPolicyNever
			in.Spec.JobTemplate.Spec.Template.Spec.ServiceAccountName = rbacOptions.ServiceAccount.Name
			in.Spec.JobTemplate.Spec.Template.Spec.ImagePullSecrets = imagePullSecrets

			// apply the pod level runtime settings to the CronJob
			if runtimeSettings.Pod != nil {
				in.Spec.JobTemplate.Spec.Template.Spec = ofst_util.ApplyPodRuntimeSettings(in.Spec.JobTemplate.Spec.Template.Spec, *runtimeSettings.Pod)
			}

			return in
		},
		metav1.PatchOptions{},
	)

	return err
}

// EnsureBackupTriggeringCronJobDeleted ensure that the CronJob of the respective backup invoker has it as owner.
// Kuebernetes garbage collector will take care of removing the CronJob
func (c *StashController) EnsureBackupTriggeringCronJobDeleted(inv invoker.BackupInvoker) error {
	invMeta := inv.GetObjectMeta()
	cur, err := c.kubeClient.BatchV1beta1().CronJobs(invMeta.Namespace).Get(context.TODO(), getBackupCronJobName(inv), metav1.GetOptions{})
	if err != nil {
		if kerr.IsNotFound(err) {
			return nil
		}
		return err
	}
	_, _, err = batch_v1beta1_util.PatchCronJob(
		context.TODO(),
		c.kubeClient,
		cur,
		func(in *batchv1beta1.CronJob) *batchv1beta1.CronJob {
			core_util.EnsureOwnerReference(&in.ObjectMeta, inv.GetOwnerRef())
			return in
		},
		metav1.PatchOptions{},
	)
	return err
}

func getBackupCronJobName(inv invoker.BackupInvoker) string {
	invMeta := inv.GetObjectMeta()
	if getTargetNamespace(inv) != invMeta.Namespace {
		return meta2.ValidCronJobNameWithPrefixNSuffix(apis.PrefixStashTrigger, invMeta.Namespace, strings.ReplaceAll(invMeta.Name, ".", "-"))
	}
	return meta2.ValidCronJobNameWithPrefixNSuffix(apis.PrefixStashTrigger, "", strings.ReplaceAll(invMeta.Name, ".", "-"))
}

func getTargetNamespace(inv invoker.BackupInvoker) string {
	for _, t := range inv.GetTargetInfo() {
		if t.Target != nil && t.Target.Ref.Namespace != "" {
			return t.Target.Ref.Namespace
		}
	}
	return inv.GetObjectMeta().Namespace
}

func (c *StashController) handleCronJobCreationFailure(ref *core.ObjectReference, err error) error {
	if ref == nil {
		return errors.NewAggregate([]error{err, fmt.Errorf("failed to write cronjob creation failure event. Reason: provided ObjectReference is nil")})
	}

	var eventSource string
	switch ref.Kind {
	case api_v1beta1.ResourceKindBackupConfiguration:
		eventSource = eventer.EventSourceBackupConfigurationController
	default:
		return errors.NewAggregate([]error{err, fmt.Errorf("failed to write cronjob creation failure event. Reason: Stash does not create cron job for %s", ref.Kind)})
	}
	// write log
	klog.Warningf("failed to create CronJob for %s %s/%s. Reason: %v", ref.Kind, ref.Namespace, ref.Name, err)

	// write event to Backup invoker
	_, err2 := eventer.CreateEvent(
		c.kubeClient,
		eventSource,
		ref,
		core.EventTypeWarning,
		eventer.EventReasonCronJobCreationFailed,
		fmt.Sprintf("failed to ensure CronJob for %s  %s/%s. Reason: %v", ref.Kind, ref.Namespace, ref.Name, err))
	return errors.NewAggregate([]error{err, err2})
}

func (c *StashController) handleWorkloadControllerTriggerFailure(ref *core.ObjectReference, err error) error {
	if ref == nil {
		return errors.NewAggregate([]error{err, fmt.Errorf("failed to write workload controller triggering failure event. Reason: provided ObjectReference is nil")})
	}
	var eventSource string
	switch ref.Kind {
	case api_v1beta1.ResourceKindBackupConfiguration:
		eventSource = eventer.EventSourceBackupConfigurationController
	case api_v1beta1.ResourceKindRestoreSession:
		eventSource = eventer.EventSourceRestoreSessionController
	}

	klog.Warningf("failed to trigger workload controller for %s %s/%s. Reason: %v", ref.Kind, ref.Namespace, ref.Name, err)

	// write event to backup invoker/RestoreSession
	_, err2 := eventer.CreateEvent(
		c.kubeClient,
		eventSource,
		ref,
		core.EventTypeWarning,
		eventer.EventReasonWorkloadControllerTriggeringFailed,
		fmt.Sprintf("failed to trigger workload controller for %s %s/%s. Reason: %v", ref.Kind, ref.Namespace, ref.Name, err),
	)
	return errors.NewAggregate([]error{err, err2})
}

func (c *StashController) requeueBackupInvoker(inv invoker.BackupInvoker, key string) error {
	switch inv.GetTypeMeta().Kind {
	case api_v1beta1.ResourceKindBackupConfiguration:
		c.bcQueue.GetQueue().AddAfter(key, requeueTimeInterval)
	default:
		return fmt.Errorf("unable to requeue. Reason: Backup invoker %s  %s is not supported",
			inv.GetTypeMeta().APIVersion,
			inv.GetTypeMeta().Kind,
		)
	}
	return nil
}

func (c *StashController) cleanupBackupInvokerOffshoots(inv invoker.BackupInvoker, invokerRef *core.ObjectReference) error {
	for _, targetInfo := range inv.GetTargetInfo() {
		if targetInfo.Target != nil {
			err := c.EnsureV1beta1SidecarDeleted(targetInfo.Target.Ref)
			if err != nil {
				return c.handleWorkloadControllerTriggerFailure(invokerRef, err)
			}
		}
	}

	if err := c.EnsureBackupTriggeringCronJobDeleted(inv); err != nil {
		return err
	}

	rbacOptions, err := c.getBackupRBACOptions(inv, nil)
	if err != nil {
		return err
	}

	if err := rbacOptions.EnsureRBACResourcesDeleted(); err != nil {
		return err
	}

	return c.deleteRepositoryReferences(inv)
}

func (c *StashController) validateDriverRequirements(inv invoker.BackupInvoker) (bool, error) {
	if inv.GetDriver() == api_v1beta1.ResticSnapshotter {
		return c.checkForResticSnapshotterRequirements(inv, inv)
	} else if inv.GetDriver() == api_v1beta1.VolumeSnapshotter {
		return false, c.checkVolumeSnapshotterRequirements(inv)
	}
	return false, nil
}

func (c *StashController) checkForResticSnapshotterRequirements(inv interface{}, r repoReferenceHandler) (bool, error) {
	repository, err := c.checkForRepositoryExistence(inv, r)
	if err != nil {
		return false, err
	}

	if repository == nil {
		klog.Infof("Repository %s/%s does not exist.\nRequeueing after 5 seconds......",
			r.GetRepoRef().Namespace,
			r.GetRepoRef().Name)
		return true, nil
	}

	secret, err := c.checkForBackendSecretExistence(inv, repository)
	if err != nil {
		return false, err
	}
	if secret == nil {
		klog.Infof("Backend Secret %s/%s does not exist for Repository %s/%s.\nRequeueing after 5 seconds......",
			repository.Namespace,
			repository.Spec.Backend.StorageSecretName,
			repository.Namespace,
			repository.Name,
		)
		return true, nil
	}

	err = c.validateAgainstUsagePolicy(r.GetRepoRef(), r.GetObjectMeta().Namespace)
	if err != nil {
		return false, conditions.SetValidationPassedToFalse(inv, err)
	}
	return false, conditions.SetValidationPassedToTrue(inv)
}

func (c *StashController) checkVolumeSnapshotterRequirements(inv interface{}) error {
	// nothing to do
	return conditions.SetValidationPassedToTrue(inv)
}

func (c *StashController) checkForRepositoryExistence(inv interface{}, r repoReferenceHandler) (*api_v1alpha1.Repository, error) {
	repository, err := r.GetRepository()
	if err != nil {
		if kerr.IsNotFound(err) {
			return nil, conditions.SetRepositoryFoundConditionToFalse(inv)
		}
		return nil, conditions.SetRepositoryFoundConditionToUnknown(inv, err)
	}

	if repository.ObjectMeta.DeletionTimestamp != nil {
		return nil, conditions.SetRepositoryFoundConditionToFalse(inv)
	}

	err = conditions.SetRepositoryFoundConditionToTrue(inv)
	if err != nil {
		return repository, err
	}

	return repository, c.upsertRepositoryReferences(r)
}

func (c *StashController) checkForBackendSecretExistence(inv interface{}, repository *api_v1alpha1.Repository) (*core.Secret, error) {
	secret, err := c.kubeClient.CoreV1().Secrets(repository.Namespace).Get(context.TODO(), repository.Spec.Backend.StorageSecretName, metav1.GetOptions{})
	if err != nil {
		if kerr.IsNotFound(err) {
			return nil, conditions.SetBackendSecretFoundConditionToFalse(inv, secret.Name)
		}
		return nil, conditions.SetBackendSecretFoundConditionToUnknown(inv, secret.Name, err)
	}
	return secret, conditions.SetBackendSecretFoundConditionToTrue(inv, secret.Name)
}

func verifyCrossNamespacePermission(inv metav1.ObjectMeta, target api_v1beta1.TargetRef, taskName string) error {
	if target.Namespace == "" || target.Namespace == inv.Namespace {
		return nil
	}
	if strings.HasPrefix(taskName, "kubedump") {
		return nil
	}
	if target.Kind == apis.KindPersistentVolumeClaim ||
		util.BackupModel(target.Kind, taskName) == apis.ModelSidecar {
		return fmt.Errorf("cross-namespace target reference is not allowed for %q", target.Kind)
	}
	return nil
}

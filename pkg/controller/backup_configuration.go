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
	"stash.appscode.dev/apimachinery/pkg/conditions"
	"stash.appscode.dev/apimachinery/pkg/docker"
	"stash.appscode.dev/apimachinery/pkg/invoker"
	"stash.appscode.dev/stash/pkg/eventer"
	"stash.appscode.dev/stash/pkg/executor"
	"stash.appscode.dev/stash/pkg/rbac"
	"stash.appscode.dev/stash/pkg/scheduler"
	"stash.appscode.dev/stash/pkg/util"

	core "k8s.io/api/core/v1"
	kerr "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/errors"
	"k8s.io/klog/v2"
	core_util "kmodules.xyz/client-go/core/v1"
	meta_util "kmodules.xyz/client-go/meta"
	"kmodules.xyz/client-go/tools/queue"
	ofst "kmodules.xyz/offshoot-api/api/v1"
	"kmodules.xyz/webhook-runtime/admission"
	hooks "kmodules.xyz/webhook-runtime/admission/v1beta1"
	webhook "kmodules.xyz/webhook-runtime/admission/v1beta1/generic"
)

const requeueTimeInterval = 5 * time.Second

type backupInvokerReconciler struct {
	ctrl    *StashController
	logger  klog.Logger
	invoker invoker.BackupInvoker
	key     string
}

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
					return nil, fmt.Errorf("updating target is forbidden when BackupConfiguration is in READY state")
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
		c.auditor.ForGVK(c.bcInformer, api_v1beta1.SchemeGroupVersion.WithKind(api_v1beta1.ResourceKindBackupConfiguration))
	}
	_, _ = c.bcInformer.AddEventHandler(queue.NewEventHandler(c.bcQueue.GetQueue(), func(oldObj, newObj interface{}) bool {
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
		klog.ErrorS(err, "Failed to fetch object from indexer",
			apis.ObjectKind, api_v1beta1.ResourceKindBackupConfiguration,
			apis.ObjectKey, key,
		)
		return err
	}
	if !exists {
		klog.V(4).InfoS("Object does not exit anymore",
			apis.ObjectKind, api_v1beta1.ResourceKindBackupConfiguration,
			apis.ObjectKey, key,
		)
		return nil
	}
	backupConfig := obj.(*api_v1beta1.BackupConfiguration)

	logger := klog.NewKlogr().WithValues(
		apis.ObjectKind, api_v1beta1.ResourceKindBackupConfiguration,
		apis.ObjectName, backupConfig.Name,
		apis.ObjectNamespace, backupConfig.Namespace,
	)
	logger.V(4).Info("Received Sync/Add/Update event")

	r := backupInvokerReconciler{
		ctrl:    c,
		logger:  logger,
		invoker: invoker.NewBackupConfigurationInvoker(c.stashClient, backupConfig),
		key:     key,
	}

	if err := r.reconcile(); err != nil {
		r.logger.Error(err, "Failed to reconcile")
		return nil
	}
	return r.invoker.UpdateObservedGeneration()
}

func (r *backupInvokerReconciler) reconcile() error {
	// check if backup invoker is being deleted. if it is being deleted then delete respective resources.
	invMeta := r.invoker.GetObjectMeta()
	invokerRef, err := r.invoker.GetObjectRef()
	if err != nil {
		return err
	}
	if invMeta.DeletionTimestamp != nil {
		if core_util.HasFinalizer(invMeta, api_v1beta1.StashKey) {
			if err := r.cleanupBackupInvokerOffshoots(invokerRef); err != nil {
				return err
			}
			return r.invoker.RemoveFinalizer()
		}
		return nil
	}

	if err := r.invoker.AddFinalizer(); err != nil {
		return err
	}

	shouldRequeue, err := r.validateDriverRequirements()
	if err != nil {
		return err
	}
	if shouldRequeue {
		return r.requeue()
	}

	someTargetMissing := false
	for _, targetInfo := range r.invoker.GetTargetInfo() {
		if targetInfo.Target != nil {
			tref := targetInfo.Target.Ref
			targetExist, err := util.IsTargetExist(r.ctrl.clientConfig, tref)
			if err != nil {
				r.logger.Error(err, "Failed to check target existence",
					apis.KeyTargetKind, tref.Kind,
					apis.KeyTargetName, tref.Name,
					apis.KeyTargetNamespace, tref.Namespace,
				)
				// Set the "BackupTargetFound" condition to "Unknown"
				return conditions.SetBackupTargetFoundConditionToUnknown(r.invoker, tref, err)
			}
			if !targetExist {
				// Target does not exist. Log the information.
				someTargetMissing = true
				r.logger.Info("Backup target does not exist.",
					apis.KeyTargetKind, tref.Kind,
					apis.KeyTargetName, tref.Name,
					apis.KeyTargetNamespace, tref.Namespace,
				)
				if err := conditions.SetBackupTargetFoundConditionToFalse(r.invoker, tref); err != nil {
					return err
				}
				// Process next target.
				continue
			}
			// Backup target exist. So, set "BackupTargetFound" condition to "True"
			if err := conditions.SetBackupTargetFoundConditionToTrue(r.invoker, tref); err != nil {
				return err
			}

			// For sidecar model, send event to the respective workload queue. The workload controller will ensure the stash sidecar.
			if r.invoker.GetDriver() == api_v1beta1.ResticSnapshotter && util.BackupModel(tref.Kind, targetInfo.Task.Name) == apis.ModelSidecar {
				err := r.ctrl.sendEventToWorkloadQueue(
					tref.Kind,
					tref.Namespace,
					tref.Name,
				)
				if err != nil {
					return r.ctrl.handleWorkloadControllerTriggerFailure(r.logger, invokerRef, tref, err)
				}
			}
		}
	}

	// If some backup targets are missing, then retry after some time.
	if someTargetMissing {
		r.logger.Info("One or more targets are missing")
		return r.requeue()
	}
	return r.ensurePeriodicScheduler(invokerRef, nil)
}

func (r *backupInvokerReconciler) ensurePeriodicScheduler(invokerRef *core.ObjectReference, index *int) error {
	rbacOptions, err := r.ctrl.getRBACOptions(
		r.invoker,
		r.invoker,
		r.invoker.GetRuntimeSettings(),
		index,
	)
	if err != nil {
		return err
	}

	s := &scheduler.PeriodicScheduler{
		KubeClient: r.ctrl.kubeClient,
		Image: docker.Docker{
			Registry: r.ctrl.DockerRegistry,
			Image:    r.ctrl.StashImage,
			Tag:      r.ctrl.StashImageTag,
		},
		Invoker:     r.invoker,
		RBACOptions: rbacOptions,
	}
	s.RBACOptions.SetPSPNames(r.ctrl.getBackupSchedulerPSPNames())

	if r.ctrl.ImagePullSecrets != nil {
		s.ImagePullSecrets, err = r.ctrl.ensureImagePullSecrets(r.invoker.GetObjectMeta(), r.invoker.GetOwnerRef())
		if err != nil {
			return err
		}
	}

	if err := s.Ensure(); err != nil {
		cerr := conditions.SetCronJobCreatedConditionToFalse(r.invoker, err)
		return r.handleCronJobCreationFailure(invokerRef, errors.NewAggregate([]error{err, cerr}))
	}
	return conditions.SetCronJobCreatedConditionToTrue(r.invoker)
}

func (c *StashController) getRBACOptions(
	inv invoker.MetadataHandler,
	repo invoker.RepositoryGetter,
	runtimeSettings ofst.RuntimeSettings,
	index *int,
) (*rbac.Options, error) {
	repository, err := c.repoLister.Repositories(repo.GetRepoRef().Namespace).Get(repo.GetRepoRef().Name)
	if err != nil && !kerr.IsNotFound(err) {
		return nil, err
	}

	rbacOptions, err := rbac.NewRBACOptions(
		c.kubeClient,
		inv,
		repository,
		index,
	)
	if err != nil {
		return nil, err
	}
	rbacOptions.SetOptionsFromRuntimeSettings(runtimeSettings)

	return rbacOptions, nil
}

func (r *backupInvokerReconciler) handleCronJobCreationFailure(invRef *core.ObjectReference, err error) error {
	if invRef == nil {
		return errors.NewAggregate([]error{err, fmt.Errorf("failed to write cronjob creation failure event. Reason: provided ObjectReference is nil")})
	}

	var eventSource string
	switch invRef.Kind {
	case api_v1beta1.ResourceKindBackupConfiguration:
		eventSource = eventer.EventSourceBackupConfigurationController
	default:
		return errors.NewAggregate([]error{err, fmt.Errorf("failed to write cronjob creation failure event. Reason: Stash does not create cron job for %s", invRef.Kind)})
	}
	r.logger.Error(err, "Failed to create backup triggering CronJob")

	// write event to Backup invoker
	_, err2 := eventer.CreateEvent(
		r.ctrl.kubeClient,
		eventSource,
		invRef,
		core.EventTypeWarning,
		eventer.EventReasonCronJobCreationFailed,
		fmt.Sprintf("failed to ensure CronJob for %s  %s/%s. Reason: %v", invRef.Kind, invRef.Namespace, invRef.Name, err))
	return errors.NewAggregate([]error{err, err2})
}

func (c *StashController) handleWorkloadControllerTriggerFailure(logger klog.Logger, invRef *core.ObjectReference, tref api_v1beta1.TargetRef, err error) error {
	if invRef == nil {
		return errors.NewAggregate([]error{err, fmt.Errorf("failed to write workload controller triggering failure event. Reason: provided ObjectReference is nil")})
	}
	var eventSource string
	switch invRef.Kind {
	case api_v1beta1.ResourceKindBackupConfiguration:
		eventSource = eventer.EventSourceBackupConfigurationController
	case api_v1beta1.ResourceKindRestoreSession:
		eventSource = eventer.EventSourceRestoreSessionController
	}

	logger.Error(err, "Failed to trigger workload controller",
		apis.KeyTargetKind, tref.Kind,
		apis.KeyTargetName, tref.Name,
		apis.KeyTargetNamespace, tref.Namespace,
	)

	// write event to backup invoker/RestoreSession
	_, err2 := eventer.CreateEvent(
		c.kubeClient,
		eventSource,
		invRef,
		core.EventTypeWarning,
		eventer.EventReasonWorkloadControllerTriggeringFailed,
		fmt.Sprintf("failed to trigger workload controller for target %s %s/%s. Reason: %v", tref.Kind, tref.Namespace, tref.Name, err),
	)
	return errors.NewAggregate([]error{err, err2})
}

func (r *backupInvokerReconciler) requeue() error {
	r.logger.Info("Requeueing after 5 seconds......")
	switch r.invoker.GetTypeMeta().Kind {
	case api_v1beta1.ResourceKindBackupConfiguration:
		r.ctrl.bcQueue.GetQueue().AddAfter(r.key, requeueTimeInterval)
	default:
		return fmt.Errorf("unable to requeue. Reason: Backup invoker %s  %s is not supported",
			r.invoker.GetTypeMeta().APIVersion,
			r.invoker.GetTypeMeta().Kind,
		)
	}
	return nil
}

func (r *backupInvokerReconciler) cleanupBackupInvokerOffshoots(invokerRef *core.ObjectReference) error {
	for _, targetInfo := range r.invoker.GetTargetInfo() {
		if targetInfo.Target != nil && backupExecutorType(r.invoker, targetInfo) == executor.TypeSidecar {
			err := r.ctrl.sendEventToWorkloadQueue(
				targetInfo.Target.Ref.Kind,
				targetInfo.Target.Ref.Namespace,
				targetInfo.Target.Ref.Name,
			)
			if err != nil {
				return r.ctrl.handleWorkloadControllerTriggerFailure(r.logger, invokerRef, targetInfo.Target.Ref, err)
			}
		}
	}

	if err := r.cleanupScheduler(); err != nil {
		return err
	}
	return r.ctrl.deleteRepositoryReferences(r.invoker)
}

func (r *backupInvokerReconciler) cleanupScheduler() error {
	rbacOptions, err := r.ctrl.getRBACOptions(r.invoker, r.invoker, r.invoker.GetRuntimeSettings(), nil)
	if err != nil {
		return err
	}
	s := &scheduler.PeriodicScheduler{
		Invoker:     r.invoker,
		KubeClient:  r.ctrl.kubeClient,
		RBACOptions: rbacOptions,
	}
	return s.Cleanup()
}

func (r *backupInvokerReconciler) validateDriverRequirements() (bool, error) {
	if r.invoker.GetDriver() == api_v1beta1.ResticSnapshotter {
		return r.ctrl.checkForResticSnapshotterRequirements(r.logger, r.invoker, r.invoker)
	}

	if r.invoker.GetDriver() == api_v1beta1.VolumeSnapshotter {
		return false, r.ctrl.checkVolumeSnapshotterRequirements(r.invoker)
	}
	return false, nil
}

func (c *StashController) checkForResticSnapshotterRequirements(logger klog.Logger, inv interface{}, r repoReferenceHandler) (bool, error) {
	repository, err := c.checkForRepositoryExistence(inv, r)
	if err != nil {
		return false, err
	}

	if repository == nil {
		logger.Info("Repository does not exist",
			apis.KeyRepositoryName, r.GetRepoRef().Name,
			apis.KeyRepositoryNamespace, r.GetRepoRef().Namespace,
		)
		return true, nil
	}

	secret, err := c.checkForBackendSecretExistence(inv, repository)
	if err != nil {
		return false, err
	}
	if secret == nil {
		klog.InfoS("Secret does not exist",
			"secret_name", repository.Spec.Backend.StorageSecretName,
			"secret_namespace", repository.Namespace,
			apis.KeyRepositoryName, repository.Name,
			apis.KeyRepositoryNamespace, repository.Namespace,
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

	if err := conditions.SetRepositoryFoundConditionToTrue(inv); err != nil {
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

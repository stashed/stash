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
	"fmt"
	"strings"
	"time"

	"stash.appscode.dev/apimachinery/apis"
	"stash.appscode.dev/apimachinery/apis/stash"
	api_v1beta1 "stash.appscode.dev/apimachinery/apis/stash/v1beta1"
	"stash.appscode.dev/apimachinery/pkg/conditions"
	stashHooks "stash.appscode.dev/apimachinery/pkg/hooks"
	"stash.appscode.dev/apimachinery/pkg/invoker"
	"stash.appscode.dev/apimachinery/pkg/metrics"
	"stash.appscode.dev/stash/pkg/executor"
	"stash.appscode.dev/stash/pkg/util"

	"gomodules.xyz/pointer"
	core "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/klog/v2"
	kutil "kmodules.xyz/client-go"
	kmapi "kmodules.xyz/client-go/api/v1"
	condutil "kmodules.xyz/client-go/conditions"
	core_util "kmodules.xyz/client-go/core/v1"
	"kmodules.xyz/client-go/meta"
	"kmodules.xyz/client-go/tools/queue"
	"kmodules.xyz/webhook-runtime/admission"
	hooks "kmodules.xyz/webhook-runtime/admission/v1beta1"
	webhook "kmodules.xyz/webhook-runtime/admission/v1beta1/generic"
	wapi "kmodules.xyz/webhook-runtime/apis/workload/v1"
	wcs "kmodules.xyz/webhook-runtime/client/workload/v1"
)

type restoreInvokerReconciler struct {
	ctrl    *StashController
	logger  klog.Logger
	invoker invoker.RestoreInvoker
	key     string
}

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
	if err := rs.IsValid(); err != nil {
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
	c.restoreSessionQueue = queue.New(api_v1beta1.ResourceKindRestoreSession, c.MaxNumRequeues, c.NumThreads, c.processRestoreSessionEvent)
	if c.auditor != nil {
		c.auditor.ForGVK(c.restoreSessionInformer, api_v1beta1.SchemeGroupVersion.WithKind(api_v1beta1.ResourceKindRestoreSession))
	}
	_, _ = c.restoreSessionInformer.AddEventHandler(queue.DefaultEventHandler(c.restoreSessionQueue.GetQueue(), core.NamespaceAll))
	c.restoreSessionLister = c.stashInformerFactory.Stash().V1beta1().RestoreSessions().Lister()
}

func (c *StashController) processRestoreSessionEvent(key string) error {
	obj, exists, err := c.restoreSessionInformer.GetIndexer().GetByKey(key)
	if err != nil {
		klog.ErrorS(err, "Failed to fetch object from indexer",
			apis.ObjectKind, api_v1beta1.ResourceKindRestoreSession,
			apis.ObjectKey, key,
		)
		return err
	}
	if !exists {
		klog.V(4).InfoS("Object does not exit anymore",
			apis.ObjectKind, api_v1beta1.ResourceKindRestoreSession,
			apis.ObjectKey, key,
		)
		return nil
	}

	restoreSession := obj.(*api_v1beta1.RestoreSession)
	logger := klog.NewKlogr().WithValues(
		apis.ObjectKind, api_v1beta1.ResourceKindRestoreSession,
		apis.ObjectName, restoreSession.Name,
		apis.ObjectNamespace, restoreSession.Namespace,
	)
	logger.V(4).Info("Received Sync/Add/Update event")

	r := restoreInvokerReconciler{
		ctrl:    c,
		logger:  logger,
		invoker: invoker.NewRestoreSessionInvoker(c.kubeClient, c.stashClient, restoreSession),
		key:     key,
	}

	// Apply any modification requires for smooth KubeDB integration
	err = r.invoker.EnsureKubeDBIntegration(c.appCatalogClient)
	if err != nil {
		return err
	}

	err = r.reconcile()
	if err != nil {
		r.logger.Error(err, "Failed to reconcile")
	}
	return nil
}

func (r *restoreInvokerReconciler) reconcile() error {
	// if the restore invoker is being deleted then remove respective init-container
	invMeta := r.invoker.GetObjectMeta()
	invokerRef, err := r.invoker.GetObjectRef()
	if err != nil {
		return err
	}

	if invMeta.DeletionTimestamp != nil {
		if core_util.HasFinalizer(invMeta, api_v1beta1.StashKey) {
			if err := r.cleanupRestoreInvokerOffshoots(invokerRef); err != nil {
				return err
			}
			return r.invoker.RemoveFinalizer()
		}
		return nil
	}

	if err := r.invoker.AddFinalizer(); err != nil {
		return err
	}

	if r.isAlreadyInFinalPhase() {
		r.logger.V(4).Info("Skipping processing event",
			apis.KeyReason, fmt.Sprintf("Restore has been completed already with phase %q", r.invoker.GetStatus().Phase),
		)
		return nil
	}

	if r.isSessionCompleted() {
		if r.shouldWaitForTargetPostRestoreHookExecution() {
			r.logger.Info("Waiting for target specific postRestore hook to be executed")
			return nil
		}

		if r.shouldExecuteGlobalPostRestoreHook() {
			if err := r.executeGlobalPostRestoreHook(); err != nil {
				condErr := conditions.SetGlobalPostRestoreHookSucceededConditionToFalse(r.invoker, err)
				if condErr != nil {
					return condErr
				}
			}
		}

		if !restoreMetricsPushed(r.invoker.GetStatus().Conditions) {
			if err := r.sendRestoreMetrics(); err != nil {
				condErr := conditions.SetRestoreMetricsPushedConditionToFalse(r.invoker, err)
				if condErr != nil {
					return condErr
				}
			}
			// this was the last step of restore. so, log indicating the completion.
			r.logRestoreCompletion()
		}
		return nil
	}

	if r.isDeadlineExceeded() {
		r.logger.Info("Cancelling restore process",
			apis.KeyReason, "Time limit exceeded",
		)
		return conditions.SetRestoreDeadlineExceededConditionToTrue(r.invoker, *r.invoker.GetTimeOut())
	}

	if r.invoker.GetDriver() == api_v1beta1.ResticSnapshotter {
		shouldRequeue, err := r.ctrl.checkForResticSnapshotterRequirements(r.logger, r.invoker, r.invoker)
		if err != nil {
			return err
		}
		if shouldRequeue {
			return r.requeue(requeueTimeInterval)
		}
	}

	if r.shouldExecuteGlobalPreRestoreHook() {
		if err := r.executeGlobalPreRestoreHook(); err != nil {
			return conditions.SetGlobalPreRestoreHookSucceededConditionToFalse(r.invoker, err)
		}
	}

	// ===================== Run Restore for the Individual Targets ============================
	for i, targetInfo := range r.invoker.GetTargetInfo() {
		if targetInfo.Target != nil {
			// Skip processing if the restore process has been already initiated before for this target
			if r.targetRestoreInitiated(targetInfo.Target.Ref) {
				continue
			}

			// Skip processing if the target is not in next in order
			if !r.nextInOrder(targetInfo) {
				r.logger.Info("Skipping restoring for target",
					apis.KeyReason, "Previous targets hasn't been executed",
					apis.KeyTargetKind, targetInfo.Target.Ref.Kind,
					apis.KeyTargetName, targetInfo.Target.Ref.Name,
					apis.KeyTargetNamespace, targetInfo.Target.Ref.Namespace,
				)
				if err := r.setTargetRestorePending(targetInfo.Target.Ref); err != nil {
					return err
				}
				continue
			}

			tref := targetInfo.Target.Ref
			shouldRequeue, err := r.checkForTargetExistence(tref, i)
			if err != nil {
				r.logger.Error(err, "Failed to check target existence",
					apis.KeyTargetKind, tref.Kind,
					apis.KeyTargetName, tref.Name,
					apis.KeyTargetNamespace, tref.Namespace,
				)
				return conditions.SetRestoreTargetFoundConditionToUnknown(r.invoker, i, err)
			}
			if shouldRequeue {
				return r.requeue(requeueTimeInterval)
			}

			if err := r.ensureRestoreExecutor(targetInfo, i); err != nil {
				r.logger.Error(err, "Failed to ensure restore executor",
					apis.KeyTargetKind, tref.Kind,
					apis.KeyTargetName, tref.Name,
					apis.KeyTargetNamespace, tref.Namespace,
				)
				msg := fmt.Sprintf("failed to ensure restore executor. Reason: %v", err)
				return conditions.SetRestoreExecutorEnsuredToFalse(r.invoker, &tref, msg)
			}
			return r.initiateTargetRestore(i)
		}
	}

	if r.invoker.GetTimeOut() != nil {
		if err := r.requeueAfterTimeOut(); err != nil {
			return err
		}
	}
	return nil
}

func (r *restoreInvokerReconciler) isAlreadyInFinalPhase() bool {
	phase := r.invoker.GetStatus().Phase
	return phase == api_v1beta1.RestoreSucceeded ||
		phase == api_v1beta1.RestoreFailed ||
		phase == api_v1beta1.RestorePhaseUnknown
}

func (r *restoreInvokerReconciler) isDeadlineExceeded() bool {
	if r.isRestoreRunning() &&
		r.isDeadlineSet() &&
		metav1.Now().After(r.invoker.GetStatus().SessionDeadline.Time) {
		return true
	}
	return false
}

func (r *restoreInvokerReconciler) isDeadlineSet() bool {
	deadline := r.invoker.GetStatus().SessionDeadline
	return !deadline.IsZero()
}

func (r *restoreInvokerReconciler) isRestoreRunning() bool {
	return r.invoker.GetStatus().Phase == api_v1beta1.RestoreRunning
}

func (r *restoreInvokerReconciler) requeueAfterTimeOut() error {
	if !r.isDeadlineSet() {
		if err := r.requeue(r.invoker.GetTimeOut().Duration); err != nil {
			return err
		}
		return r.setDeadline(r.invoker.GetTimeOut().Duration)
	}
	return nil
}

func (r *restoreInvokerReconciler) setDeadline(timeOut time.Duration) error {
	r.logger.Info("Deadline has been set")
	deadline := metav1.NewTime(r.invoker.GetObjectMeta().CreationTimestamp.Add(timeOut))
	return r.invoker.UpdateStatus(invoker.RestoreInvokerStatus{
		SessionDeadline: &deadline,
	})
}

func restorerExecutorType(targetInfo invoker.RestoreTargetInfo, driver api_v1beta1.Snapshotter) executor.Type {
	if util.RestoreModel(targetInfo.Target.Ref.Kind, targetInfo.Task.Name) == apis.ModelSidecar {
		return executor.TypeInitContainer
	} else if driver == api_v1beta1.VolumeSnapshotter {
		return executor.TypeCSISnapshotRestorer
	} else {
		return executor.TypeRestoreJob
	}
}

func (r *restoreInvokerReconciler) requeue(requeueTimeInterval time.Duration) error {
	r.logger.Info(fmt.Sprintf("Requeueing after %s....", requeueTimeInterval.String()))

	invTypeMeta := r.invoker.GetTypeMeta()
	switch invTypeMeta.Kind {
	case api_v1beta1.ResourceKindRestoreSession:
		r.ctrl.restoreSessionQueue.GetQueue().AddAfter(r.key, requeueTimeInterval)
	default:
		return fmt.Errorf("unable to requeue. Reason: Restore invoker %s %s is not supported",
			invTypeMeta.APIVersion,
			invTypeMeta.Kind,
		)
	}
	return nil
}

func (r *restoreInvokerReconciler) isSessionCompleted() bool {
	if r.globalPreRestoreHookFailed() {
		return true
	}

	if condutil.IsConditionTrue(r.invoker.GetStatus().Conditions, api_v1beta1.DeadlineExceeded) {
		return true
	}

	if invoker.RestoreCompletedForAllTargets(r.invoker.GetStatus().TargetStatus, len(r.invoker.GetTargetInfo())) {
		return true
	}
	return false
}

func (r *restoreInvokerReconciler) globalPreRestoreHookFailed() bool {
	return condutil.IsConditionFalse(r.invoker.GetStatus().Conditions, api_v1beta1.GlobalPreRestoreHookSucceeded)
}

func (r *restoreInvokerReconciler) shouldWaitForTargetPostRestoreHookExecution() bool {
	for _, targetInfo := range r.invoker.GetTargetInfo() {
		if targetInfo.Hooks != nil && targetInfo.Hooks.PostRestore != nil {
			if r.targetPreRestoreHookFailed(targetInfo.Target.Ref) {
				continue
			}
			if !r.postRestoreHookExecutedForTarget(targetInfo) {
				return true
			}
		}
	}
	return false
}

func (r *restoreInvokerReconciler) targetPreRestoreHookFailed(targetRef api_v1beta1.TargetRef) bool {
	for _, s := range r.invoker.GetStatus().TargetStatus {
		if invoker.TargetMatched(s.Ref, targetRef) {
			return condutil.IsConditionFalse(s.Conditions, api_v1beta1.PreRestoreHookExecutionSucceeded)
		}
	}
	return false
}

func (r *restoreInvokerReconciler) postRestoreHookExecutedForTarget(targetInfo invoker.RestoreTargetInfo) bool {
	if targetInfo.Target == nil {
		return true
	}
	status := r.invoker.GetStatus()

	for _, s := range status.TargetStatus {
		if invoker.TargetMatched(s.Ref, targetInfo.Target.Ref) {
			if condutil.HasCondition(s.Conditions, api_v1beta1.PostRestoreHookExecutionSucceeded) {
				return true
			}
		}
	}
	return false
}

func (r *restoreInvokerReconciler) shouldExecuteGlobalPostRestoreHook() bool {
	hook := r.invoker.GetGlobalHooks()
	if hook != nil && hook.PostRestore != nil && hook.PostRestore.Handler != nil {
		if r.globalPreRestoreHookFailed() {
			return false
		}
		return !condutil.HasCondition(r.invoker.GetStatus().Conditions, api_v1beta1.GlobalPostRestoreHookSucceeded)
	}
	return false
}

func (r *restoreInvokerReconciler) executeGlobalPostRestoreHook() error {
	hookExecutor := stashHooks.HookExecutor{
		Config: r.ctrl.clientConfig,
		Hook:   r.invoker.GetGlobalHooks().PostRestore.Handler,
		ExecutorPod: kmapi.ObjectReference{
			Namespace: meta.PodNamespace(),
			Name:      meta.PodName(),
		},
		Summary: r.invoker.GetSummary(api_v1beta1.TargetRef{}, kmapi.ObjectReference{
			Namespace: r.invoker.GetObjectMeta().Namespace,
			Name:      r.invoker.GetObjectMeta().Name,
		}),
	}

	executionPolicy := r.invoker.GetGlobalHooks().PostRestore.ExecutionPolicy
	if executionPolicy == "" {
		executionPolicy = api_v1beta1.ExecuteAlways
	}

	if !stashHooks.IsAllowedByExecutionPolicy(executionPolicy, hookExecutor.Summary) {
		reason := fmt.Sprintf("Skipping executing %s. Reason: executionPolicy is %q but phase is %q.",
			apis.PostRestoreHook,
			executionPolicy,
			hookExecutor.Summary.Status.Phase,
		)
		return conditions.SetGlobalPostRestoreHookSucceededConditionToTrueWithMsg(r.invoker, reason)
	}

	if err := hookExecutor.Execute(); err != nil {
		return err
	}
	return conditions.SetGlobalPostRestoreHookSucceededConditionToTrue(r.invoker)
}

func (r *restoreInvokerReconciler) shouldExecuteGlobalPreRestoreHook() bool {
	hook := r.invoker.GetGlobalHooks()
	if hook != nil && hook.PreRestore != nil {
		return !condutil.HasCondition(r.invoker.GetStatus().Conditions, api_v1beta1.GlobalPreRestoreHookSucceeded)
	}
	return false
}

func (r *restoreInvokerReconciler) executeGlobalPreRestoreHook() error {
	hookExecutor := stashHooks.HookExecutor{
		Config: r.ctrl.clientConfig,
		Hook:   r.invoker.GetGlobalHooks().PreRestore,
		ExecutorPod: kmapi.ObjectReference{
			Namespace: meta.PodNamespace(),
			Name:      meta.PodName(),
		},
		Summary: r.invoker.GetSummary(api_v1beta1.TargetRef{}, kmapi.ObjectReference{
			Namespace: r.invoker.GetObjectMeta().Namespace,
			Name:      r.invoker.GetObjectMeta().Name,
		}),
	}
	if err := hookExecutor.Execute(); err != nil {
		return err
	}
	return conditions.SetGlobalPreRestoreHookSucceededConditionToTrue(r.invoker)
}

func (r *restoreInvokerReconciler) targetRestoreInitiated(targetRef api_v1beta1.TargetRef) bool {
	status := r.invoker.GetStatus()
	if invoker.TargetRestoreCompleted(targetRef, status.TargetStatus) {
		return true
	}
	for _, target := range status.TargetStatus {
		if invoker.TargetMatched(target.Ref, targetRef) {
			return condutil.HasCondition(target.Conditions, api_v1beta1.RestoreExecutorEnsured) || target.Phase == api_v1beta1.TargetRestoreRunning
		}
	}
	return false
}

func (r *restoreInvokerReconciler) initiateTargetRestore(index int) error {
	targetInfo := r.invoker.GetTargetInfo()[index]
	totalHosts, err := r.ctrl.getTotalHostForRestore(targetInfo.Target)
	if err != nil {
		return err
	}
	return r.invoker.UpdateStatus(invoker.RestoreInvokerStatus{
		TargetStatus: []api_v1beta1.RestoreMemberStatus{
			{
				Ref:        targetInfo.Target.Ref,
				TotalHosts: totalHosts,
			},
		},
	})
}

func (r *restoreInvokerReconciler) setTargetRestorePending(targetRef api_v1beta1.TargetRef) error {
	return r.invoker.UpdateStatus(invoker.RestoreInvokerStatus{
		TargetStatus: []api_v1beta1.RestoreMemberStatus{
			{
				Ref: targetRef,
			},
		},
	})
}

func restoreMetricsPushed(conditions []kmapi.Condition) bool {
	return condutil.IsConditionTrue(conditions, api_v1beta1.MetricsPushed)
}

func (r *restoreInvokerReconciler) sendRestoreMetrics() error {
	// send restore metrics
	metricsOpt := &metrics.MetricsOptions{
		Enabled:        true,
		PushgatewayURL: metrics.GetPushgatewayURL(),
		JobName: fmt.Sprintf("%s-%s-%s",
			strings.ToLower(
				r.invoker.GetTypeMeta().Kind),
			r.invoker.GetObjectMeta().Namespace,
			r.invoker.GetObjectMeta().Name,
		),
	}
	// send target specific metrics
	for _, target := range r.invoker.GetStatus().TargetStatus {
		if err := metricsOpt.SendRestoreTargetMetrics(r.ctrl.clientConfig, r.invoker, target.Ref); err != nil {
			return err
		}
	}
	// send restore session metrics
	if err := metricsOpt.SendRestoreSessionMetrics(r.invoker); err != nil {
		return err
	}
	return conditions.SetRestoreMetricsPushedConditionToTrue(r.invoker)
}

func (r *restoreInvokerReconciler) checkForTargetExistence(tref api_v1beta1.TargetRef, idx int) (bool, error) {
	// if target hasn't been specified, we don't need to check for its existence
	if tref.Name == "" {
		return false, nil
	}

	targetExist, err := util.IsTargetExist(r.ctrl.clientConfig, tref)
	if err != nil {
		return false, err
	}

	if !targetExist {
		r.logger.Info("Restore target does not exist",
			apis.KeyTargetKind, tref.Kind,
			apis.KeyTargetName, tref.Name,
			apis.KeyTargetNamespace, tref.Namespace,
		)
		return true, conditions.SetRestoreTargetFoundConditionToFalse(r.invoker, idx)
	}

	return false, conditions.SetRestoreTargetFoundConditionToTrue(r.invoker, idx)
}

func (r *restoreInvokerReconciler) nextInOrder(targetInfo invoker.RestoreTargetInfo) bool {
	if r.invoker.GetExecutionOrder() == api_v1beta1.Sequential &&
		!r.invoker.NextInOrder(targetInfo.Target.Ref, r.invoker.GetStatus().TargetStatus) {
		return false
	}
	return true
}

func (r *restoreInvokerReconciler) ensureRestoreExecutor(targetInfo invoker.RestoreTargetInfo, idx int) error {
	var restoreExecutor executor.Executor
	var err error

	tref := targetInfo.Target.Ref
	switch restorerExecutorType(targetInfo, r.invoker.GetDriver()) {
	case executor.TypeInitContainer:
		obj, err := r.ctrl.getTargetWorkload(targetInfo.Target.Ref)
		if err != nil {
			return err
		}
		w, err := wcs.ConvertToWorkload(obj.DeepCopyObject())
		if err != nil {
			return err
		}
		restoreExecutor, err = r.ctrl.newInitContainerExecutor(r.invoker, w, idx, apis.CallerController)
		if err != nil {
			return err
		}
	case executor.TypeCSISnapshotRestorer:
		restoreExecutor, err = r.ctrl.newVolumeSnapshotRestorer(r.invoker, idx)
		if err != nil {
			return err
		}
	case executor.TypeRestoreJob:
		restoreExecutor, err = r.ctrl.newRestoreJobExecutor(r.invoker, idx)
		if err != nil {
			return err
		}
	default:
		return fmt.Errorf("unable to identify restorer entity for target %s %s/%s", tref.Kind, tref.Namespace, tref.Name)
	}

	_, verb, err := restoreExecutor.Ensure()
	if err != nil {
		return err
	}

	if verb != kutil.VerbUnchanged {
		r.logger.Info("Successfully ensured restore executor",
			apis.KeyTargetKind, tref.Kind,
			apis.KeyTargetName, tref.Name,
			apis.KeyTargetNamespace, tref.Namespace,
		)
		msg := fmt.Sprintf("Restorer job/init-container has been ensured successfully for %s %s/%s.", tref.Kind, tref.Namespace, tref.Name)
		return conditions.SetRestoreExecutorEnsuredToTrue(r.invoker, &tref, msg)
	}
	return nil
}

func (c *StashController) newInitContainerExecutor(inv invoker.RestoreInvoker, w *wapi.Workload, index int, caller string) (*executor.InitContainer, error) {
	targetInfo := inv.GetTargetInfo()[index]

	rbacOptions, err := c.getRBACOptions(inv, inv, targetInfo.RuntimeSettings, &index)
	if err != nil {
		return nil, err
	}

	e := &executor.InitContainer{
		KubeClient:        c.kubeClient,
		OpenshiftClient:   c.ocClient,
		StashClient:       c.stashClient,
		RBACOptions:       rbacOptions,
		Invoker:           inv,
		Index:             index,
		Image:             c.getDockerImage(),
		LicenseApiService: c.LicenseApiService,
		Caller:            caller,
		Workload:          w,
	}

	e.Repository, err = c.repoLister.Repositories(inv.GetRepoRef().Namespace).Get(inv.GetRepoRef().Name)
	if err != nil {
		return nil, err
	}

	if c.ImagePullSecrets != nil {
		e.ImagePullSecrets, err = c.ensureImagePullSecrets(inv.GetObjectMeta(), inv.GetOwnerRef())
		if err != nil {
			return nil, err
		}
	}
	return e, nil
}

func (c *StashController) newVolumeSnapshotRestorer(inv invoker.RestoreInvoker, index int) (*executor.CSISnapshotRestorer, error) {
	targetInfo := inv.GetTargetInfo()[index]

	rbacOptions, err := c.getRBACOptions(inv, inv, targetInfo.RuntimeSettings, &index)
	if err != nil {
		return nil, err
	}

	re := &executor.CSISnapshotRestorer{
		KubeClient:  c.kubeClient,
		RBACOptions: rbacOptions,
		Invoker:     inv,
		Index:       index,
		Image:       c.getDockerImage(),
	}

	if c.ImagePullSecrets != nil {
		re.ImagePullSecrets, err = c.ensureImagePullSecrets(inv.GetObjectMeta(), inv.GetOwnerRef())
		if err != nil {
			return nil, err
		}
	}

	return re, nil
}

func (c *StashController) newRestoreJobExecutor(inv invoker.RestoreInvoker, index int) (*executor.RestoreJob, error) {
	targetInfo := inv.GetTargetInfo()[index]

	rbacOptions, err := c.getRBACOptions(inv, inv, targetInfo.RuntimeSettings, &index)
	if err != nil {
		return nil, err
	}

	e := &executor.RestoreJob{
		KubeClient:        c.kubeClient,
		StashClient:       c.stashClient,
		CatalogClient:     c.appCatalogClient,
		RBACOptions:       rbacOptions,
		Invoker:           inv,
		Index:             index,
		Image:             c.getDockerImage(),
		LicenseApiService: c.LicenseApiService,
	}

	e.Repository, err = c.repoLister.Repositories(inv.GetRepoRef().Namespace).Get(inv.GetRepoRef().Name)
	if err != nil {
		return nil, err
	}

	psps, err := c.getRestoreJobPSPNames(targetInfo.Task)
	if err != nil {
		return nil, nil
	}
	e.RBACOptions.SetPSPNames(psps)

	if c.ImagePullSecrets != nil {
		e.ImagePullSecrets, err = c.ensureImagePullSecrets(inv.GetObjectMeta(), inv.GetOwnerRef())
		if err != nil {
			return nil, err
		}
	}

	return e, nil
}

func (r *restoreInvokerReconciler) cleanupRestoreInvokerOffshoots(invokerRef *core.ObjectReference) error {
	for _, targetInfo := range r.invoker.GetTargetInfo() {
		target := targetInfo.Target
		if target != nil && util.RestoreModel(target.Ref.Kind, targetInfo.Task.Name) == apis.ModelSidecar {
			// send event to workload controller. workload controller will take care of removing restore init-container
			err := r.ctrl.sendEventToWorkloadQueue(
				target.Ref.Kind,
				target.Ref.Namespace,
				target.Ref.Name,
			)
			if err != nil {
				return r.ctrl.handleWorkloadControllerTriggerFailure(r.logger, invokerRef, target.Ref, err)
			}
		}
		rbacOptions, err := r.ctrl.getRBACOptions(r.invoker, r.invoker, targetInfo.RuntimeSettings, nil)
		if err != nil {
			return err
		}

		if err := rbacOptions.EnsureRBACResourcesDeleted(); err != nil {
			return err
		}
	}

	return r.ctrl.deleteRepositoryReferences(r.invoker)
}

func (c *StashController) getTotalHostForRestore(t *api_v1beta1.RestoreTarget) (*int32, error) {
	if t == nil {
		return pointer.Int32P(1), nil
	}

	// if volumeClaimTemplates is specified, Stash creates the PVCs and restore into it.
	if len(t.VolumeClaimTemplates) != 0 {
		return getNumberOfPVCToCreate(t), nil
	}
	return c.getTotalHostForRestic(t.Ref)
}

func getNumberOfPVCToCreate(t *api_v1beta1.RestoreTarget) *int32 {
	replica := int32(1)
	if t.Replicas != nil {
		replica = pointer.Int32(t.Replicas)
	}
	return pointer.Int32P(replica * int32(len(t.VolumeClaimTemplates)))
}

func (r *restoreInvokerReconciler) logRestoreCompletion() {
	if r.invoker.GetStatus().Phase == api_v1beta1.RestoreSucceeded {
		r.logger.Info("Successfully completed restore")
		return
	}
	if r.invoker.GetStatus().Phase == api_v1beta1.RestoreFailed {
		r.logger.Info("Restore has failed")
	}
}

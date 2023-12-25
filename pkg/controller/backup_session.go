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
	"sort"
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
	"stash.appscode.dev/stash/pkg/scheduler"
	"stash.appscode.dev/stash/pkg/util"

	"gomodules.xyz/pointer"
	core "k8s.io/api/core/v1"
	kerr "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/tools/clientcmd/api"
	"k8s.io/klog/v2"
	kutil "kmodules.xyz/client-go"
	kmapi "kmodules.xyz/client-go/api/v1"
	condutil "kmodules.xyz/client-go/conditions"
	"kmodules.xyz/client-go/meta"
	"kmodules.xyz/client-go/tools/queue"
	"kmodules.xyz/webhook-runtime/admission"
	hooks "kmodules.xyz/webhook-runtime/admission/v1beta1"
	webhook "kmodules.xyz/webhook-runtime/admission/v1beta1/generic"
	wapi "kmodules.xyz/webhook-runtime/apis/workload/v1"
	wcs "kmodules.xyz/webhook-runtime/client/workload/v1"
)

type backupSessionReconciler struct {
	ctrl    *StashController
	logger  klog.Logger
	session *invoker.BackupSessionHandler
	invoker invoker.BackupInvoker
	key     string
}

func (c *StashController) NewBackupSessionWebhook() hooks.AdmissionHook {
	return webhook.NewGenericWebhook(
		schema.GroupVersionResource{
			Group:    "admission.stash.appscode.com",
			Version:  "v1beta1",
			Resource: "backupsessionvalidators",
		},
		"backupsessionvalidator",
		[]string{stash.GroupName},
		api.SchemeGroupVersion.WithKind(api_v1beta1.ResourceKindBackupSession),
		nil,
		&admission.ResourceHandlerFuncs{
			CreateFunc: func(obj runtime.Object) (runtime.Object, error) {
				return nil, obj.(*api_v1beta1.BackupSession).IsValid()
			},
			UpdateFunc: func(oldObj, newObj runtime.Object) (runtime.Object, error) {
				// should not allow spec update
				if !meta.Equal(oldObj.(*api_v1beta1.BackupSession).Spec, newObj.(*api_v1beta1.BackupSession).Spec) {
					return nil, fmt.Errorf("BackupSession spec is immutable")
				}
				return nil, nil
			},
		},
	)
}

func (c *StashController) initBackupSessionWatcher() {
	c.backupSessionInformer = c.stashInformerFactory.Stash().V1beta1().BackupSessions().Informer()
	c.backupSessionQueue = queue.New(api_v1beta1.ResourceKindBackupSession, c.MaxNumRequeues, c.NumThreads, c.processBackupSessionEvent)
	if c.auditor != nil {
		c.auditor.ForGVK(c.backupSessionInformer, api_v1beta1.SchemeGroupVersion.WithKind(api_v1beta1.ResourceKindBackupSession))
	}
	_, _ = c.backupSessionInformer.AddEventHandler(queue.DefaultEventHandler(c.backupSessionQueue.GetQueue(), core.NamespaceAll))
	c.backupSessionLister = c.stashInformerFactory.Stash().V1beta1().BackupSessions().Lister()
}

func (c *StashController) processBackupSessionEvent(key string) error {
	obj, exists, err := c.backupSessionInformer.GetIndexer().GetByKey(key)
	if err != nil {
		klog.ErrorS(err, "Failed to fetch object from indexer",
			apis.ObjectKind, api_v1beta1.ResourceKindBackupSession,
			apis.ObjectKey, key,
		)
		return err
	}
	if !exists {
		klog.V(4).InfoS("Object does not exist anymore",
			apis.ObjectKind, api_v1beta1.ResourceKindBackupSession,
			apis.ObjectKey, key,
		)
		return nil
	}
	backupSession := obj.(*api_v1beta1.BackupSession)

	logger := klog.NewKlogr().WithValues(
		apis.ObjectKind, api_v1beta1.ResourceKindBackupSession,
		apis.ObjectName, backupSession.Name,
		apis.ObjectNamespace, backupSession.Namespace,
	)
	logger.V(4).Info("Received Sync/Add/Update event")

	r := backupSessionReconciler{
		ctrl:    c,
		logger:  logger,
		session: invoker.NewBackupSessionHandler(c.stashClient, backupSession),
		key:     key,
	}
	err = r.reconcile()
	if err != nil {
		r.logger.Error(err, "Failed to reconcile")
	}
	return nil
}

func (r *backupSessionReconciler) reconcile() error {
	var err error
	r.invoker, err = r.session.GetInvoker()
	if err != nil {
		return err
	}

	if r.isAlreadyInFinalPhase() {
		if r.isBackupFailed() && r.shouldRetry() {
			if r.retryDelayPassed() {
				return r.retryNow()
			}
			return r.requeueAfterRetryDelay()
		}

		r.logger.V(4).Info("Skipping processing event",
			apis.KeyReason, fmt.Sprintf("Backup has been completed already with phase %q", r.session.GetStatus().Phase),
		)
		return nil
	}

	if r.isSessionCompleted() {
		if r.shouldWaitForTargetPostBackupHookExecution() {
			r.logger.Info("Waiting for target specific postBackup hook to be executed",
				apis.KeyInvokerKind, r.invoker.GetTypeMeta().Kind,
				apis.KeyInvokerName, r.invoker.GetObjectMeta().Name,
				apis.KeyInvokerNamespace, r.invoker.GetObjectMeta().Namespace,
			)
			return nil
		}

		if r.shouldExecuteGlobalPostBackupHook() {
			if err := r.executeGlobalPostBackupHook(); err != nil {
				condErr := conditions.SetGlobalPostBackupHookSucceededConditionToFalse(r.session, err)
				if condErr != nil {
					return condErr
				}
			}
		}

		// cleanup old BackupSession according to backupHistoryLimit
		if !r.isBackupHistoryCleaned() {
			if err := r.cleanupBackupHistory(); err != nil {
				condErr := conditions.SetBackupHistoryCleanedConditionToFalse(r.session, err)
				if condErr != nil {
					return condErr
				}
			}
		}

		if !r.backupMetricPushed() {
			if err := r.sendBackupMetrics(); err != nil {
				condErr := conditions.SetBackupMetricsPushedConditionToFalse(r.session, err)
				if condErr != nil {
					return condErr
				}
			}
			// this was the last step of backup. so, log indicating the completion.
			r.logBackupCompletion()
		}
		return nil
	}

	if r.isDeadlineExceeded() {
		r.logger.Info("Cancelling backup", apis.KeyReason, "Time Limit exceeded")
		return conditions.SetBackupDeadlineExceededConditionToTrue(r.session, *r.invoker.GetTimeOut())
	}

	skippingReason, err := r.checkIfBackupShouldBeSkipped()
	if err != nil {
		return err
	}

	if skippingReason != "" {
		r.logger.Info(skippingReason)
		if err := conditions.SetBackupSkippedConditionToTrue(r.session, skippingReason); err != nil {
			return err
		}
		// cleanup old BackupSession according to backupHistoryLimit
		if !r.isBackupHistoryCleaned() {
			if err := r.cleanupBackupHistory(); err != nil {
				return conditions.SetBackupHistoryCleanedConditionToFalse(r.session, err)
			}
		}
		return nil
	}

	if r.shouldExecuteGlobalPreBackupHook() {
		if err := r.executeGlobalPreBackupHook(); err != nil {
			return conditions.SetGlobalPreBackupHookSucceededConditionToFalse(r.session, err)
		}
	}

	// ===================== Run Backup for the Individual Targets ============================
	shouldRequeue := false
	for i, targetInfo := range r.invoker.GetTargetInfo() {
		if targetInfo.Target != nil {
			// Skip processing if the backup has been already initiated before for this target
			if invoker.TargetBackupInitiated(targetInfo.Target.Ref, r.session.GetTargetStatus()) {
				continue
			}

			pendingReason, err := r.checkIfBackupShouldBePending(targetInfo.Target.Ref)
			if err != nil {
				return err
			}
			if pendingReason != "" {
				r.logger.Info("Keeping backup pending",
					apis.KeyReason, pendingReason,
					apis.KeyTargetKind, targetInfo.Target.Ref.Kind,
					apis.KeyTargetName, targetInfo.Target.Ref.Name,
					apis.KeyTargetNamespace, targetInfo.Target.Ref.Namespace,
				)
				if err := r.setTargetBackupPending(targetInfo.Target.Ref); err != nil {
					return err
				}
				shouldRequeue = true
				continue
			}

			if err := r.ensureBackupExecutor(targetInfo, i); err != nil {
				r.logger.Error(err, "Failed to ensure backup executor",
					apis.KeyTargetKind, targetInfo.Target.Ref.Kind,
					apis.KeyTargetName, targetInfo.Target.Ref.Name,
					apis.KeyTargetNamespace, targetInfo.Target.Ref.Namespace,
				)
				return conditions.SetBackupExecutorEnsuredToFalse(r.session, targetInfo.Target.Ref, err)
			}

			// Set target backup phase to "Running"
			if err = r.initiateTargetBackup(i); err != nil {
				return err
			}
		}
	}
	if shouldRequeue {
		r.requeue(requeueTimeInterval)
		return nil
	}

	if r.invoker.GetTimeOut() != nil {
		if err := r.requeueAfterTimeOut(); err != nil {
			return err
		}
	}

	return nil
}

func (r *backupSessionReconciler) isDeadlineSet() bool {
	deadline := r.session.GetStatus().SessionDeadline
	return !deadline.IsZero()
}

func (r *backupSessionReconciler) setDeadline(timeOut time.Duration) error {
	r.logger.Info("Deadline has been set")
	deadline := metav1.NewTime(r.session.GetObjectMeta().CreationTimestamp.Add(timeOut))
	return r.session.UpdateStatus(&api_v1beta1.BackupSessionStatus{
		SessionDeadline: &deadline,
	})
}

func (r *backupSessionReconciler) requeueAfterTimeOut() error {
	if !r.isDeadlineSet() {
		r.requeue(r.invoker.GetTimeOut().Duration)
		return r.setDeadline(r.invoker.GetTimeOut().Duration)
	}
	return nil
}

func (r *backupSessionReconciler) isDeadlineExceeded() bool {
	if r.isBackupRunning() &&
		r.isDeadlineSet() &&
		metav1.Now().After(r.session.GetStatus().SessionDeadline.Time) {
		return true
	}
	return false
}

func (r *backupSessionReconciler) requeue(timeOut time.Duration) {
	r.logger.Info(fmt.Sprintf("Requeueing after %s.....", timeOut.String()))
	r.ctrl.backupSessionQueue.GetQueue().AddAfter(r.key, timeOut)
}

func (r *backupSessionReconciler) ensureBackupExecutor(targetInfo invoker.BackupTargetInfo, idx int) error {
	var backupExecutor executor.Executor
	var err error

	switch backupExecutorType(r.invoker, targetInfo) {
	case executor.TypeSidecar:
		obj, err := r.ctrl.getTargetWorkload(targetInfo.Target.Ref)
		if err != nil {
			return err
		}
		w, err := wcs.ConvertToWorkload(obj.DeepCopyObject())
		if err != nil {
			return err
		}
		inv := r.invoker
		latestInvoker, err := util.FindLatestBackupInvoker(r.ctrl.bcLister, targetInfo.Target.Ref)
		if err != nil {
			return err
		}
		if latestInvoker.Object != nil && inv.GetObjectMeta().UID != latestInvoker.GetUID() {
			inv, err = invoker.NewBackupInvoker(
				r.ctrl.stashClient,
				latestInvoker.GetKind(),
				latestInvoker.GetName(),
				latestInvoker.GetNamespace(),
			)
			if err != nil {
				return err
			}
			for i, t := range inv.GetTargetInfo() {
				if invoker.TargetMatched(t.Target.Ref, targetInfo.Target.Ref) {
					idx = i
					targetInfo = t
					break
				}
			}
		}

		backupExecutor, err = r.ctrl.newSidecarExecutor(inv, w, idx, apis.CallerController)
		if err != nil {
			return err
		}
	case executor.TypeCSISnapshooter:
		backupExecutor, err = r.ctrl.newVolumeSnapshooter(r.invoker, r.session, idx)
		if err != nil {
			return err
		}
	case executor.TypeBackupJob:
		backupExecutor, err = r.ctrl.newBackupJob(r.invoker, r.session, idx)
		if err != nil {
			return err
		}
	default:
		return fmt.Errorf("unable to identify backup executor entity")
	}

	_, verb, err := backupExecutor.Ensure()
	if err != nil {
		return err
	}
	if verb != kutil.VerbUnchanged {
		r.logger.Info("Successfully ensured backup executor",
			apis.KeyTargetKind, targetInfo.Target.Ref.Kind,
			apis.KeyTargetName, targetInfo.Target.Ref.Name,
			apis.KeyTargetNamespace, targetInfo.Target.Ref.Namespace,
		)
		return conditions.SetBackupExecutorEnsuredToTrue(r.session, targetInfo.Target.Ref)
	}
	return nil
}

func (c *StashController) newSidecarExecutor(inv invoker.BackupInvoker, w *wapi.Workload, index int, caller string) (*executor.Sidecar, error) {
	targetInfo := inv.GetTargetInfo()[index]
	rbacOptions, err := c.getRBACOptions(inv, inv, targetInfo.RuntimeSettings, &index)
	if err != nil {
		return nil, err
	}

	e := &executor.Sidecar{
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

func (c *StashController) newVolumeSnapshooter(inv invoker.BackupInvoker, session *invoker.BackupSessionHandler, index int) (*executor.CSISnapshooter, error) {
	targetInfo := inv.GetTargetInfo()[index]
	rbacOptions, err := c.getRBACOptions(inv, inv, targetInfo.RuntimeSettings, &index)
	if err != nil {
		return nil, err
	}

	e := &executor.CSISnapshooter{
		KubeClient:  c.kubeClient,
		RBACOptions: rbacOptions,
		Invoker:     inv,
		Session:     session,
		Index:       index,
		Image:       c.getDockerImage(),
	}

	if c.ImagePullSecrets != nil {
		e.ImagePullSecrets, err = c.ensureImagePullSecrets(inv.GetObjectMeta(), inv.GetOwnerRef())
		if err != nil {
			return nil, err
		}
	}

	return e, nil
}

func (c *StashController) newBackupJob(inv invoker.BackupInvoker, session *invoker.BackupSessionHandler, index int) (*executor.BackupJob, error) {
	targetInfo := inv.GetTargetInfo()[index]
	rbacOptions, err := c.getRBACOptions(inv, inv, targetInfo.RuntimeSettings, &index)
	if err != nil {
		return nil, err
	}

	e := &executor.BackupJob{
		KubeClient:        c.kubeClient,
		StashClient:       c.stashClient,
		CatalogClient:     c.appCatalogClient,
		RBACOptions:       rbacOptions,
		Invoker:           inv,
		Session:           session,
		Index:             index,
		Image:             c.getDockerImage(),
		LicenseApiService: c.LicenseApiService,
	}

	e.Repository, err = c.repoLister.Repositories(inv.GetRepoRef().Namespace).Get(inv.GetRepoRef().Name)
	if err != nil {
		return nil, err
	}
	psps, err := c.getBackupJobPSPNames(targetInfo.Task)
	if err != nil {
		return nil, err
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

func (r *backupSessionReconciler) setTargetBackupPending(targetRef api_v1beta1.TargetRef) error {
	return r.session.UpdateStatus(&api_v1beta1.BackupSessionStatus{
		Targets: []api_v1beta1.BackupTargetStatus{
			{
				Ref: targetRef,
			},
		},
	})
}

func (r *backupSessionReconciler) initiateTargetBackup(index int) error {
	targetsInfo := r.invoker.GetTargetInfo()
	target := targetsInfo[index].Target
	// find out the total number of hosts in target that will be backed up in this backup session
	totalHosts, err := r.ctrl.getTotalHostForBackup(target, r.invoker.GetDriver())
	if err != nil {
		return err
	}
	// For Restic driver, set preBackupAction and postBackupAction
	var preBackupActions, postBackupActions []string
	if r.invoker.GetDriver() == api_v1beta1.ResticSnapshotter {
		// assign preBackupAction to the first target
		if index == 0 {
			preBackupActions = []string{api_v1beta1.InitializeBackendRepository}
		}
		// assign postBackupAction to the last target
		if index == len(targetsInfo)-1 {
			postBackupActions = []string{api_v1beta1.ApplyRetentionPolicy, api_v1beta1.VerifyRepositoryIntegrity, api_v1beta1.SendRepositoryMetrics}
		}
	}
	if r.session.GetStatus().Phase != api_v1beta1.BackupSessionRunning {
		r.logger.Info("Initiating target backup",
			apis.KeyTargetKind, target.Ref.Kind,
			apis.KeyTargetName, target.Ref.Name,
			apis.KeyTargetNamespace, target.Ref.Namespace,
		)
	}
	return r.session.UpdateStatus(&api_v1beta1.BackupSessionStatus{
		Targets: []api_v1beta1.BackupTargetStatus{
			{
				TotalHosts:        totalHosts,
				Ref:               target.Ref,
				PreBackupActions:  preBackupActions,
				PostBackupActions: postBackupActions,
			},
		},
	})
}

func (r *backupSessionReconciler) isBackupHistoryCleaned() bool {
	return condutil.HasCondition(r.session.GetConditions(), api_v1beta1.BackupHistoryCleaned)
}

// cleanupBackupHistory deletes old BackupSessions and theirs associate resources according to BackupHistoryLimit
func (r *backupSessionReconciler) cleanupBackupHistory() error {
	// default history limit is 1
	historyLimit := int32(1)
	if r.invoker.GetBackupHistoryLimit() != nil {
		historyLimit = *r.invoker.GetBackupHistoryLimit()
	}

	// BackupSession use BackupConfiguration name as label. We can use this label as selector to list only the BackupSession
	// of this particular BackupConfiguration.
	label := metav1.LabelSelector{
		MatchLabels: map[string]string{
			apis.LabelInvokerType: r.session.GetInvokerRef().Kind,
			apis.LabelInvokerName: r.session.GetInvokerRef().Name,
		},
	}
	selector, err := metav1.LabelSelectorAsSelector(&label)
	if err != nil {
		return err
	}

	// list all the BackupSessions of this particular BackupConfiguration
	bsList, err := r.ctrl.backupSessionLister.BackupSessions(r.session.GetObjectMeta().Namespace).List(selector)
	if err != nil {
		return err
	}

	// sort BackupSession according to creation timestamp. keep latest BackupSession first.
	sort.Slice(bsList, func(i, j int) bool {
		return bsList[i].CreationTimestamp.After(bsList[j].CreationTimestamp.Time)
	})

	var lastCompletedSession string
	for i := range bsList {
		if bsList[i].Status.Phase == api_v1beta1.BackupSessionSucceeded || bsList[i].Status.Phase == api_v1beta1.BackupSessionFailed {
			lastCompletedSession = bsList[i].Name
			break
		}
	}
	// delete the BackupSession that does not fit within the history limit
	for i := int(historyLimit); i < len(bsList); i++ {
		if invoker.IsBackupCompleted(bsList[i].Status.Phase) && !(bsList[i].Name == lastCompletedSession && historyLimit > 0) {
			err = r.ctrl.stashClient.StashV1beta1().BackupSessions(r.session.GetObjectMeta().Namespace).Delete(context.TODO(), bsList[i].Name, meta.DeleteInBackground())
			if err != nil && !(kerr.IsNotFound(err) || kerr.IsGone(err)) {
				return err
			}
		}
	}
	return conditions.SetBackupHistoryCleanedConditionToTrue(r.session)
}

func backupExecutorType(inv invoker.BackupInvoker, targetInfo invoker.BackupTargetInfo) executor.Type {
	if inv.GetDriver() == api_v1beta1.ResticSnapshotter &&
		util.BackupModel(targetInfo.Target.Ref.Kind, targetInfo.Task.Name) == apis.ModelSidecar {
		return executor.TypeSidecar
	}
	if inv.GetDriver() == api_v1beta1.VolumeSnapshotter {
		return executor.TypeCSISnapshooter
	}
	return executor.TypeBackupJob
}

func (r *backupSessionReconciler) shouldWaitForTargetPostBackupHookExecution() bool {
	for _, targetInfo := range r.invoker.GetTargetInfo() {
		if targetInfo.Hooks != nil && targetInfo.Hooks.PostBackup != nil {
			// don't execute postBackup hook if preBackup hook has failed
			if r.targetPreBackupHookFailed(targetInfo.Target.Ref) {
				continue
			}
			if !r.postBackupHookExecutedForTarget(targetInfo) {
				return true
			}
		}
	}
	return false
}

func (r *backupSessionReconciler) targetPreBackupHookFailed(targetRef api_v1beta1.TargetRef) bool {
	for _, s := range r.session.GetTargetStatus() {
		if invoker.TargetMatched(s.Ref, targetRef) {
			return condutil.IsConditionFalse(s.Conditions, api_v1beta1.PreBackupHookExecutionSucceeded)
		}
	}
	return false
}

func (r *backupSessionReconciler) postBackupHookExecutedForTarget(targetInfo invoker.BackupTargetInfo) bool {
	if targetInfo.Target == nil {
		return true
	}

	for _, s := range r.session.GetTargetStatus() {
		if invoker.TargetMatched(s.Ref, targetInfo.Target.Ref) {
			if condutil.HasCondition(s.Conditions, api_v1beta1.PostBackupHookExecutionSucceeded) {
				return true
			}
		}
	}
	return false
}

func (r *backupSessionReconciler) shouldExecuteGlobalPostBackupHook() bool {
	hook := r.invoker.GetGlobalHooks()
	if hook != nil && hook.PostBackup != nil && hook.PostBackup.Handler != nil {
		if r.globalPreBackupHookFailed() {
			return false
		}
		return !condutil.HasCondition(r.session.GetConditions(), api_v1beta1.GlobalPostBackupHookSucceeded)
	}
	return false
}

func (r *backupSessionReconciler) executeGlobalPostBackupHook() error {
	summary := r.invoker.GetSummary(api_v1beta1.TargetRef{}, kmapi.ObjectReference{
		Namespace: r.session.GetObjectMeta().Namespace,
		Name:      r.session.GetObjectMeta().Name,
	})
	summary.Status.Phase = string(r.session.GetStatus().Phase)
	summary.Status.Duration = r.session.GetStatus().SessionDuration

	hookExecutor := stashHooks.HookExecutor{
		Config: r.ctrl.clientConfig,
		Hook:   r.invoker.GetGlobalHooks().PostBackup.Handler,
		ExecutorPod: kmapi.ObjectReference{
			Namespace: meta.PodNamespace(),
			Name:      meta.PodName(),
		},
		Summary: summary,
	}

	executionPolicy := r.invoker.GetGlobalHooks().PostBackup.ExecutionPolicy
	if executionPolicy == "" {
		executionPolicy = api_v1beta1.ExecuteAlways
	}

	if !stashHooks.IsAllowedByExecutionPolicy(executionPolicy, summary) {
		reason := fmt.Sprintf("Skipping executing %s. Reason: executionPolicy is %q but phase is %q.",
			apis.PostBackupHook,
			executionPolicy,
			summary.Status.Phase,
		)
		return conditions.SetGlobalPostBackupHookSucceededConditionToTrueWithMsg(r.session, reason)
	}
	if err := hookExecutor.Execute(); err != nil {
		return err
	}
	return conditions.SetGlobalPostBackupHookSucceededConditionToTrue(r.session)
}

func (r *backupSessionReconciler) shouldExecuteGlobalPreBackupHook() bool {
	hook := r.invoker.GetGlobalHooks()
	if hook != nil && hook.PreBackup != nil {
		return !condutil.HasCondition(r.session.GetConditions(), api_v1beta1.GlobalPreBackupHookSucceeded)
	}
	return false
}

func (r *backupSessionReconciler) executeGlobalPreBackupHook() error {
	hookExecutor := stashHooks.HookExecutor{
		Config: r.ctrl.clientConfig,
		Hook:   r.invoker.GetGlobalHooks().PreBackup,
		ExecutorPod: kmapi.ObjectReference{
			Namespace: meta.PodNamespace(),
			Name:      meta.PodName(),
		},
		Summary: r.invoker.GetSummary(api_v1beta1.TargetRef{}, kmapi.ObjectReference{
			Namespace: r.session.GetObjectMeta().Namespace,
			Name:      r.session.GetObjectMeta().Name,
		}),
	}
	if err := hookExecutor.Execute(); err != nil {
		return err
	}
	return conditions.SetGlobalPreBackupHookSucceededConditionToTrue(r.session)
}

func (r *backupSessionReconciler) checkIfBackupShouldBeSkipped() (string, error) {
	// Skip if the respective backup invoker is not in ready state
	if r.invoker.GetPhase() != api_v1beta1.BackupInvokerReady {
		return fmt.Sprintf("Skipped taking backup. Reason: %s %s/%s is not ready.",
			r.invoker.GetTypeMeta().Kind,
			r.invoker.GetObjectMeta().Namespace,
			r.invoker.GetObjectMeta().Name,
		), nil
	}

	// Skip taking backup if there is another running BackupSession
	runningBS, err := r.checkForAnotherRunningBackupSessionWithSameInvoker()
	if err != nil {
		return "", err
	}
	if r.isBackupPending() && runningBS != nil {
		return fmt.Sprintf("Skipped taking new backup. Reason: Previous BackupSession: %s is %q.",
			runningBS.Name,
			runningBS.Status.Phase,
		), nil
	}
	return "", nil
}

func (r *backupSessionReconciler) checkIfBackupShouldBePending(targetRef api_v1beta1.TargetRef) (string, error) {
	// Keep backup pending if the target is not in next in order
	if r.invoker.GetExecutionOrder() == api_v1beta1.Sequential &&
		!r.invoker.NextInOrder(targetRef, r.session.GetTargetStatus()) {
		return "Backup order is sequential and some previous targets hasn't completed their backup process.", nil
	}

	yes, err := r.hasMultipleBackupInvokers(targetRef)
	if err != nil || !yes {
		return "", err
	}

	otherSession, err := r.checkForAnotherIncompleteBackupSessionWithDifferentInvoker(targetRef)
	if err != nil {
		return "", err
	}
	if otherSession == nil {
		if time.Since(r.session.GetObjectMeta().CreationTimestamp.Time) < 2*time.Second {
			return "Multiple backup invoker found. Will be processed in the next requeue to handle concurrent session.", nil
		}
		return "", nil
	}

	if shouldKeepCurrentSessionPending(r.session.GetBackupSession(), otherSession) {
		return fmt.Sprintf("Found another incomplete BackupSession %s/%s invoked by %s/%s",
			otherSession.Namespace,
			otherSession.Name,
			otherSession.Spec.Invoker.Kind,
			otherSession.Spec.Invoker.Name,
		), nil
	}
	return "", nil
}

func shouldKeepCurrentSessionPending(cur, other *api_v1beta1.BackupSession) bool {
	if other.Status.Phase == api_v1beta1.BackupSessionRunning {
		return true
	}

	if cur.CreationTimestamp.Equal(other.CreationTimestamp.DeepCopy()) {
		return other.Name < cur.Name
	}
	return cur.CreationTimestamp.After(other.CreationTimestamp.Time)
}

func (r *backupSessionReconciler) hasMultipleBackupInvokers(targetRef api_v1beta1.TargetRef) (bool, error) {
	invokers, err := util.FindBackupInvokers(r.ctrl.bcLister, targetRef)
	if err != nil {
		return false, err
	}

	return len(invokers) > 1, nil
}

func (r *backupSessionReconciler) isAlreadyInFinalPhase() bool {
	phase := r.session.GetStatus().Phase
	return phase == api_v1beta1.BackupSessionSucceeded ||
		phase == api_v1beta1.BackupSessionFailed ||
		phase == api_v1beta1.BackupSessionSkipped
}

func (r *backupSessionReconciler) isSessionCompleted() bool {
	if r.globalPreBackupHookFailed() {
		return true
	}

	if condutil.IsConditionTrue(r.session.GetConditions(), api_v1beta1.DeadlineExceeded) {
		return true
	}

	if invoker.BackupCompletedForAllTargets(r.session.GetTargetStatus(), len(r.invoker.GetTargetInfo())) {
		return true
	}
	return false
}

func (r *backupSessionReconciler) globalPreBackupHookFailed() bool {
	return condutil.IsConditionFalse(r.session.GetConditions(), api_v1beta1.GlobalPreBackupHookSucceeded)
}

func (r *backupSessionReconciler) isBackupRunning() bool {
	return r.session.GetStatus().Phase == api_v1beta1.BackupSessionRunning
}

func (r *backupSessionReconciler) isBackupPending() bool {
	return r.session.GetStatus().Phase == "" || r.session.GetStatus().Phase == api_v1beta1.BackupSessionPending
}

func (r *backupSessionReconciler) checkForAnotherRunningBackupSessionWithSameInvoker() (*api_v1beta1.BackupSession, error) {
	runningBS, err := r.getRunningBackupSessionForInvoker()
	if err != nil {
		return nil, err
	}
	if runningBS != nil && runningBS.Name != r.session.GetObjectMeta().Name {
		return runningBS, nil
	}
	return nil, nil
}

func (r *backupSessionReconciler) getRunningBackupSessionForInvoker() (*api_v1beta1.BackupSession, error) {
	backupSessions, err := r.ctrl.backupSessionLister.BackupSessions(r.invoker.GetObjectMeta().Namespace).List(labels.SelectorFromSet(map[string]string{
		apis.LabelInvokerName: r.invoker.GetObjectMeta().Name,
		apis.LabelInvokerType: r.invoker.GetTypeMeta().Kind,
	}))
	if err != nil {
		return nil, err
	}
	for i := range backupSessions {
		if backupSessions[i].Status.Phase == api_v1beta1.BackupSessionRunning {
			return backupSessions[i], nil
		}
	}
	return nil, nil
}

func (r *backupSessionReconciler) checkForAnotherIncompleteBackupSessionWithDifferentInvoker(targetRef api_v1beta1.TargetRef) (*api_v1beta1.BackupSession, error) {
	sessions, err := r.getIncompleteBackupSessionForTarget(targetRef)
	if err != nil {
		return nil, err
	}

	if len(sessions) > 1 {
		sort.Slice(sessions, func(i, j int) bool {
			if sessions[i].Status.Phase == sessions[j].Status.Phase {
				return sessions[i].Name < sessions[j].Name
			}
			if sessions[i].Status.Phase == api_v1beta1.BackupSessionRunning && sessions[j].Status.Phase != api_v1beta1.BackupSessionRunning {
				return true
			}
			return false
		})
	}

	for i, s := range sessions {
		if r.invokedByDifferentInvoker(s.Spec.Invoker) {
			s := sessions[i]
			return &s, nil
		}
	}
	return nil, nil
}

func (r *backupSessionReconciler) invokedByDifferentInvoker(invRef api_v1beta1.BackupInvokerRef) bool {
	return invRef.Kind != r.invoker.GetTypeMeta().Kind ||
		invRef.Name != r.invoker.GetObjectMeta().Name
}

func (r *backupSessionReconciler) getIncompleteBackupSessionForTarget(targetRef api_v1beta1.TargetRef) ([]api_v1beta1.BackupSession, error) {
	selector := labels.SelectorFromSet(map[string]string{
		apis.LabelTargetKind:      targetRef.Kind,
		apis.LabelTargetName:      targetRef.Name,
		apis.LabelTargetNamespace: targetRef.Namespace,
	})
	bsList, err := r.ctrl.stashClient.StashV1beta1().BackupSessions(r.invoker.GetObjectMeta().Namespace).List(context.TODO(), metav1.ListOptions{LabelSelector: selector.String()})
	if err != nil {
		return nil, err
	}

	sessions := make([]api_v1beta1.BackupSession, 0)
	for i := range bsList.Items {
		if bsList.Items[i].Status.Phase == api_v1beta1.BackupSessionRunning ||
			bsList.Items[i].Status.Phase == api_v1beta1.BackupSessionPending ||
			bsList.Items[i].Status.Phase == "" {
			sessions = append(sessions, bsList.Items[i])
		}
	}
	return sessions, nil
}

func (r *backupSessionReconciler) backupMetricPushed() bool {
	return condutil.IsConditionTrue(r.session.GetConditions(), api_v1beta1.MetricsPushed)
}

func (r *backupSessionReconciler) sendBackupMetrics() error {
	metricsOpt := &metrics.MetricsOptions{
		Enabled:        true,
		PushgatewayURL: metrics.GetPushgatewayURL(),
		JobName:        fmt.Sprintf("%s-%s-%s", strings.ToLower(r.invoker.GetTypeMeta().Kind), r.invoker.GetObjectMeta().Namespace, r.invoker.GetObjectMeta().Name),
	}

	status := r.session.GetStatus()
	if status.SessionDuration == "" {
		status.SessionDuration = time.Since(r.session.GetObjectMeta().CreationTimestamp.Time).Round(time.Second).String()
	}

	// send backup session related metrics
	err := metricsOpt.SendBackupSessionMetrics(r.invoker, status)
	if err != nil {
		return err
	}
	// send target related metrics
	for _, target := range r.session.GetTargetStatus() {
		err = metricsOpt.SendBackupTargetMetrics(r.ctrl.clientConfig, r.invoker, target.Ref, r.session.GetStatus())
		if err != nil {
			return err
		}
	}

	return conditions.SetBackupMetricsPushedConditionToTrue(r.session)
}

func (c *StashController) getTotalHostForBackup(t *api_v1beta1.BackupTarget, driver api_v1beta1.Snapshotter) (*int32, error) {
	if t == nil {
		return pointer.Int32P(1), nil
	}
	if driver == api_v1beta1.VolumeSnapshotter {
		return c.getNumberOfVolumeToSnapshot(t)
	}
	return c.getTotalHostForRestic(t.Ref)
}

func (c *StashController) getNumberOfVolumeToSnapshot(t *api_v1beta1.BackupTarget) (*int32, error) {
	switch t.Ref.Kind {
	case apis.KindStatefulSet:
		ss, err := c.kubeClient.AppsV1().StatefulSets(t.Ref.Namespace).Get(context.TODO(), t.Ref.Name, metav1.GetOptions{})
		if err != nil {
			return nil, err
		}
		claimedVolumes := int32(0)
		if t.Replicas != nil {
			claimedVolumes = *t.Replicas * int32(len(ss.Spec.VolumeClaimTemplates))
		} else {
			claimedVolumes = pointer.Int32(ss.Spec.Replicas) * int32(len(ss.Spec.VolumeClaimTemplates))
		}
		nonClaimedVolumes := countPVC(ss.Spec.Template.Spec.Volumes)

		return pointer.Int32P(claimedVolumes + *nonClaimedVolumes), nil

	case apis.KindDeployment:
		deployment, err := c.kubeClient.AppsV1().Deployments(t.Ref.Namespace).Get(context.TODO(), t.Ref.Name, metav1.GetOptions{})
		if err != nil {
			return nil, err
		}
		return countPVC(deployment.Spec.Template.Spec.Volumes), err

	case apis.KindDaemonSet:
		daemon, err := c.kubeClient.AppsV1().DaemonSets(t.Ref.Namespace).Get(context.TODO(), t.Ref.Name, metav1.GetOptions{})
		if err != nil {
			return nil, err
		}
		return countPVC(daemon.Spec.Template.Spec.Volumes), err

	default:
		return pointer.Int32P(1), nil
	}
}

func countPVC(volList []core.Volume) *int32 {
	var count int32
	for _, vol := range volList {
		if vol.PersistentVolumeClaim != nil {
			count++
		}
	}
	return &count
}

func (c *StashController) getTotalHostForRestic(targetRef api_v1beta1.TargetRef) (*int32, error) {
	switch targetRef.Kind {
	// all replicas of StatefulSet will take backup/restore. so total number of hosts will be number of replicas.
	case apis.KindStatefulSet:
		ss, err := c.kubeClient.AppsV1().StatefulSets(targetRef.Namespace).Get(context.TODO(), targetRef.Name, metav1.GetOptions{})
		if err != nil {
			return nil, err
		}
		return ss.Spec.Replicas, nil
	// all Daemon pod will take backup/restore. so total number of hosts will be number of ready replicas
	case apis.KindDaemonSet:
		dmn, err := c.kubeClient.AppsV1().DaemonSets(targetRef.Namespace).Get(context.TODO(), targetRef.Name, metav1.GetOptions{})
		if err != nil {
			return nil, err
		}
		return &dmn.Status.DesiredNumberScheduled, nil
	// for all other workloads, only one replica will take backup/restore. so number of total host will be 1
	default:
		return pointer.Int32P(1), nil
	}
}

func (r *backupSessionReconciler) logBackupCompletion() {
	if r.session.GetStatus().Phase == api_v1beta1.BackupSessionSucceeded {
		r.logger.Info("Successfully completed backup")
		return
	}
	if r.session.GetStatus().Phase == api_v1beta1.BackupSessionFailed {
		r.logger.Info("Backup has failed")
	}
}

func (r *backupSessionReconciler) isBackupFailed() bool {
	return r.session.GetStatus().Phase == api_v1beta1.BackupSessionFailed
}

func (r *backupSessionReconciler) shouldRetry() bool {
	bs := r.session.GetBackupSession()
	if bs.Spec.RetryLeft > 0 && !alreadyRetried(bs) {
		return true
	}
	return false
}

func alreadyRetried(bs *api_v1beta1.BackupSession) bool {
	if bs.Status.Retried != nil && *bs.Status.Retried {
		return true
	}
	return false
}

func (r *backupSessionReconciler) retryDelayPassed() bool {
	status := r.session.GetStatus()
	if status.NextRetry != nil && time.Now().After(status.NextRetry.Time) {
		return true
	}
	return false
}

func (r *backupSessionReconciler) retryNow() error {
	r.logger.Info("Retrying the failed backup")
	s := scheduler.InstantScheduler{
		StashClient: r.ctrl.stashClient,
		Invoker:     r.invoker,
		RetryLeft:   r.session.GetBackupSession().Spec.RetryLeft - 1,
	}
	err := s.Ensure()
	if err != nil {
		return err
	}

	return r.session.UpdateStatus(&api_v1beta1.BackupSessionStatus{
		Retried: pointer.BoolP(true),
	})
}

func (r *backupSessionReconciler) requeueAfterRetryDelay() error {
	retryConfig := r.invoker.GetRetryConfig()
	if retryConfig == nil {
		return nil
	}

	if r.session.GetStatus().NextRetry == nil {
		err := r.setNextRetryTimestamp(retryConfig)
		if err != nil {
			return err
		}
	}
	r.requeue(time.Until(r.session.GetStatus().NextRetry.Time))

	return nil
}

func (r *backupSessionReconciler) setNextRetryTimestamp(retryConfig *api_v1beta1.RetryConfig) error {
	nextRetry := metav1.NewTime(time.Now().Add(retryConfig.Delay.Duration))

	return r.session.UpdateStatus(&api_v1beta1.BackupSessionStatus{
		NextRetry: &nextRetry,
	})
}

/*
Copyright AppsCode Inc. and Contributors

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package invoker

import (
	"context"
	"fmt"
	"time"

	"stash.appscode.dev/apimachinery/apis/stash/v1alpha1"
	"stash.appscode.dev/apimachinery/apis/stash/v1beta1"
	cs "stash.appscode.dev/apimachinery/client/clientset/versioned"
	stash_scheme "stash.appscode.dev/apimachinery/client/clientset/versioned/scheme"
	v1beta1_util "stash.appscode.dev/apimachinery/client/clientset/versioned/typed/stash/v1beta1/util"

	"gomodules.xyz/pointer"
	core "k8s.io/api/core/v1"
	kerr "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/reference"
	kmapi "kmodules.xyz/client-go/api/v1"
	core_util "kmodules.xyz/client-go/core/v1"
	"kmodules.xyz/client-go/meta"
	appcat "kmodules.xyz/custom-resources/apis/appcatalog/v1alpha1"
	appcatalog_cs "kmodules.xyz/custom-resources/client/clientset/versioned"
)

type RestoreSessionInvoker struct {
	kubeClient     kubernetes.Interface
	stashClient    cs.Interface
	restoreSession *v1beta1.RestoreSession
}

func NewRestoreSessionInvoker(kubeClient kubernetes.Interface, stashClient cs.Interface, restoreSession *v1beta1.RestoreSession) RestoreInvoker {
	return &RestoreSessionInvoker{
		kubeClient:     kubeClient,
		stashClient:    stashClient,
		restoreSession: restoreSession,
	}
}

func (inv *RestoreSessionInvoker) GetObjectMeta() metav1.ObjectMeta {
	return inv.restoreSession.ObjectMeta
}

func (inv *RestoreSessionInvoker) GetTypeMeta() metav1.TypeMeta {
	return metav1.TypeMeta{
		Kind:       v1beta1.ResourceKindRestoreSession,
		APIVersion: v1beta1.SchemeGroupVersion.String(),
	}
}

func (inv *RestoreSessionInvoker) GetObjectRef() (*core.ObjectReference, error) {
	return reference.GetReference(stash_scheme.Scheme, inv.restoreSession)
}

func (inv *RestoreSessionInvoker) GetOwnerRef() *metav1.OwnerReference {
	return metav1.NewControllerRef(inv.restoreSession, v1beta1.SchemeGroupVersion.WithKind(v1beta1.ResourceKindRestoreSession))
}

func (inv *RestoreSessionInvoker) GetLabels() map[string]string {
	return inv.restoreSession.OffshootLabels()
}

func (inv *RestoreSessionInvoker) AddFinalizer() error {
	updatedRestoreSession, _, err := v1beta1_util.PatchRestoreSession(context.TODO(), inv.stashClient.StashV1beta1(), inv.restoreSession, func(in *v1beta1.RestoreSession) *v1beta1.RestoreSession {
		in.ObjectMeta = core_util.AddFinalizer(in.ObjectMeta, v1beta1.StashKey)
		return in
	}, metav1.PatchOptions{})
	if err != nil {
		return err
	}
	inv.restoreSession = updatedRestoreSession
	return nil
}

func (inv *RestoreSessionInvoker) RemoveFinalizer() error {
	updatedRestoreSession, _, err := v1beta1_util.PatchRestoreSession(context.TODO(), inv.stashClient.StashV1beta1(), inv.restoreSession, func(in *v1beta1.RestoreSession) *v1beta1.RestoreSession {
		in.ObjectMeta = core_util.RemoveFinalizer(in.ObjectMeta, v1beta1.StashKey)
		return in
	}, metav1.PatchOptions{})
	if err != nil {
		return err
	}
	inv.restoreSession = updatedRestoreSession
	return nil
}

func (inv *RestoreSessionInvoker) HasCondition(target *v1beta1.TargetRef, conditionType string) (bool, error) {
	restoreSession, err := inv.stashClient.StashV1beta1().RestoreSessions(inv.restoreSession.Namespace).Get(context.TODO(), inv.restoreSession.Name, metav1.GetOptions{})
	if err != nil {
		return false, err
	}
	return kmapi.HasCondition(restoreSession.Status.Conditions, conditionType), nil
}

func (inv *RestoreSessionInvoker) GetCondition(target *v1beta1.TargetRef, conditionType string) (int, *kmapi.Condition, error) {
	restoreSession, err := inv.stashClient.StashV1beta1().RestoreSessions(inv.restoreSession.Namespace).Get(context.TODO(), inv.restoreSession.Name, metav1.GetOptions{})
	if err != nil {
		return -1, nil, err
	}
	idx, cond := kmapi.GetCondition(restoreSession.Status.Conditions, conditionType)
	return idx, cond, nil
}

func (inv *RestoreSessionInvoker) SetCondition(target *v1beta1.TargetRef, newCondition kmapi.Condition) error {
	status := inv.GetStatus()
	status.Conditions = kmapi.SetCondition(status.Conditions, newCondition)
	status.TargetStatus[0].Conditions = status.Conditions
	return inv.UpdateStatus(status)
}

func (inv *RestoreSessionInvoker) IsConditionTrue(target *v1beta1.TargetRef, conditionType string) (bool, error) {
	restoreSession, err := inv.stashClient.StashV1beta1().RestoreSessions(inv.restoreSession.Namespace).Get(context.TODO(), inv.restoreSession.Name, metav1.GetOptions{})
	if err != nil {
		return false, err
	}
	return kmapi.IsConditionTrue(restoreSession.Status.Conditions, conditionType), nil
}

func (inv *RestoreSessionInvoker) GetTargetInfo() []RestoreTargetInfo {
	return []RestoreTargetInfo{
		{
			Task:                  inv.restoreSession.Spec.Task,
			Target:                inv.restoreSession.Spec.Target,
			RuntimeSettings:       inv.restoreSession.Spec.RuntimeSettings,
			TempDir:               inv.restoreSession.Spec.TempDir,
			InterimVolumeTemplate: inv.restoreSession.Spec.InterimVolumeTemplate,
			Hooks:                 inv.restoreSession.Spec.Hooks,
		},
	}
}

func (inv *RestoreSessionInvoker) GetDriver() v1beta1.Snapshotter {
	driver := inv.restoreSession.Spec.Driver
	if driver == "" {
		driver = v1beta1.ResticSnapshotter
	}
	return driver
}

func (inv *RestoreSessionInvoker) GetRepoRef() kmapi.ObjectReference {
	var repo kmapi.ObjectReference
	repo.Name = inv.restoreSession.Spec.Repository.Name
	repo.Namespace = inv.restoreSession.Spec.Repository.Namespace
	if repo.Namespace == "" {
		repo.Namespace = inv.restoreSession.Namespace
	}
	return repo
}

func (inv *RestoreSessionInvoker) GetRepository() (*v1alpha1.Repository, error) {
	repo := inv.GetRepoRef()
	return inv.stashClient.StashV1alpha1().Repositories(repo.Namespace).Get(context.TODO(), repo.Name, metav1.GetOptions{})
}

func (inv *RestoreSessionInvoker) GetGlobalHooks() *v1beta1.RestoreHooks {
	return nil
}

func (inv *RestoreSessionInvoker) GetExecutionOrder() v1beta1.ExecutionOrder {
	return v1beta1.Sequential
}

func (inv *RestoreSessionInvoker) NextInOrder(curTarget v1beta1.TargetRef, targetStatus []v1beta1.RestoreMemberStatus) bool {
	for _, t := range inv.GetTargetInfo() {
		if t.Target != nil {
			if TargetMatched(t.Target.Ref, curTarget) {
				return true
			}
			if !TargetRestoreCompleted(t.Target.Ref, targetStatus) {
				return false
			}
		}
	}
	// By default, return true so that nil target(i.e. cluster restore) does not get stuck here.
	return true
}

func (inv *RestoreSessionInvoker) GetHash() string {
	return inv.restoreSession.GetSpecHash()
}

func (inv *RestoreSessionInvoker) GetObjectJSON() (string, error) {
	jsonObj, err := meta.MarshalToJson(inv.restoreSession, v1beta1.SchemeGroupVersion)
	if err != nil {
		return "", err
	}
	return string(jsonObj), nil
}

func (inv *RestoreSessionInvoker) GetRuntimeObject() runtime.Object {
	return inv.restoreSession
}

func (inv *RestoreSessionInvoker) CreateEvent(eventType, source, reason, message string) error {
	objRef, err := inv.GetObjectRef()
	if err != nil {
		return err
	}

	t := metav1.Time{Time: time.Now()}
	if source == "" {
		source = EventSourceRestoreSessionController
	}
	_, err = inv.kubeClient.CoreV1().Events(inv.restoreSession.Namespace).Create(context.TODO(), &core.Event{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%v.%x", inv.restoreSession.Name, t.UnixNano()),
			Namespace: inv.restoreSession.Namespace,
		},
		InvolvedObject: *objRef,
		Reason:         reason,
		Message:        message,
		FirstTimestamp: t,
		LastTimestamp:  t,
		Count:          1,
		Type:           eventType,
		Source:         core.EventSource{Component: source},
	}, metav1.CreateOptions{})
	return err
}

func (inv *RestoreSessionInvoker) EnsureKubeDBIntegration(appClient appcatalog_cs.Interface) error {
	target := inv.restoreSession.Spec.Target
	// Don't do anything if the target is not an AppBinding
	if target == nil || !TargetOfGroupKind(target.Ref, appcat.SchemeGroupVersion.Group, appcat.ResourceKindApp) {
		return nil
	}

	appBinding, err := appClient.AppcatalogV1alpha1().AppBindings(inv.restoreSession.Namespace).Get(context.TODO(), target.Ref.Name, metav1.GetOptions{})
	if err != nil {
		// If the AppBinding does not exist, then don't do anything.
		if kerr.IsNotFound(err) {
			return nil
		}
		return err
	}
	// If the AppBinding is not managed by KubeDB, then don't do anything
	if manager, err := meta.GetStringValue(appBinding.Labels, meta.ManagedByLabelKey); err != nil || manager != "kubedb.com" {
		return nil
	}
	// Extract the name, instance, and managed-by labels.
	appLabels, err := extractLabels(appBinding.Labels, meta.InstanceLabelKey, meta.ManagedByLabelKey, meta.NameLabelKey)
	if err != nil {
		return err
	}

	// Add the labels to the invoker
	updatedRestoreSession, _, err := v1beta1_util.PatchRestoreSession(context.TODO(), inv.stashClient.StashV1beta1(), inv.restoreSession, func(in *v1beta1.RestoreSession) *v1beta1.RestoreSession {
		in.Labels = meta.OverwriteKeys(in.Labels, appLabels)
		return in
	}, metav1.PatchOptions{})
	if err != nil {
		return err
	}
	inv.restoreSession = updatedRestoreSession
	return nil
}

func (inv *RestoreSessionInvoker) GetStatus() RestoreInvokerStatus {
	return getInvokerStatusFromRestoreSession(inv.restoreSession)
}

func (inv *RestoreSessionInvoker) UpdateStatus(status RestoreInvokerStatus) error {
	startTime := inv.GetObjectMeta().CreationTimestamp.Time
	updatedRestoreSession, err := v1beta1_util.UpdateRestoreSessionStatus(
		context.TODO(),
		inv.stashClient.StashV1beta1(),
		inv.restoreSession.ObjectMeta,
		func(in *v1beta1.RestoreSessionStatus) (types.UID, *v1beta1.RestoreSessionStatus) {
			curStatus := v1beta1.RestoreMemberStatus{
				Conditions: in.Conditions,
				TotalHosts: in.TotalHosts,
				Stats:      in.Stats,
			}
			newStatus := v1beta1.RestoreMemberStatus{
				Conditions: status.TargetStatus[0].Conditions,
				TotalHosts: status.TargetStatus[0].TotalHosts,
				Stats:      status.TargetStatus[0].Stats,
			}
			updatedStatus := upsertRestoreTargetStatus(curStatus, newStatus)

			in.Conditions = updatedStatus.Conditions
			in.Stats = updatedStatus.Stats
			in.TotalHosts = updatedStatus.TotalHosts
			in.Phase = calculateRestoreSessionPhase(updatedStatus)

			if IsRestoreCompleted(in.Phase) && in.SessionDuration == "" {
				in.SessionDuration = time.Since(startTime).Round(time.Second).String()
			}
			return inv.restoreSession.ObjectMeta.UID, in
		},
		metav1.UpdateOptions{},
	)
	if err != nil {
		return err
	}
	inv.restoreSession = updatedRestoreSession
	return nil
}

func (inv *RestoreSessionInvoker) GetSummary(target v1beta1.TargetRef, session kmapi.ObjectReference) *v1beta1.Summary {
	summary := &v1beta1.Summary{
		Name:      session.Name,
		Namespace: session.Namespace,
		Target:    target,
		Invoker: core.TypedLocalObjectReference{
			APIGroup: pointer.StringP(v1beta1.SchemeGroupVersion.Group),
			Kind:     v1beta1.ResourceKindRestoreSession,
			Name:     inv.restoreSession.Name,
		},
	}
	restoreSession, err := inv.stashClient.StashV1beta1().RestoreSessions(session.Namespace).Get(context.TODO(), session.Name, metav1.GetOptions{})
	if err != nil {
		summary.Status.Phase = string(v1beta1.RestorePhaseUnknown)
		summary.Status.Error = fmt.Sprintf("Unable to summarize target restore state. Reason: %s", err.Error())
		return summary
	}
	summary.Status.Duration = time.Since(restoreSession.CreationTimestamp.Time).Round(time.Second).String()

	failureFound, reason := checkRestoreFailureInHostStatus(restoreSession.Status.Stats)
	if failureFound {
		summary.Status.Phase = string(v1beta1.RestoreFailed)
		summary.Status.Error = reason
		return summary
	}

	failureFound, reason = checkFailureInConditions(restoreSession.Status.Conditions)
	if failureFound {
		summary.Status.Phase = string(v1beta1.RestoreFailed)
		summary.Status.Error = reason
		return summary
	}

	summary.Status.Phase = string(v1beta1.RestoreSucceeded)

	return summary
}

func checkRestoreFailureInHostStatus(status []v1beta1.HostRestoreStats) (bool, string) {
	for _, host := range status {
		if hostRestoreCompleted(host.Phase) && host.Phase != v1beta1.HostRestoreSucceeded {
			return true, host.Error
		}
	}
	return false, ""
}

func checkFailureInConditions(conditions []kmapi.Condition) (bool, string) {
	for _, c := range conditions {
		if c.Status == core.ConditionFalse {
			return true, c.Message
		}
	}
	return false, ""
}

func calculateRestoreSessionPhase(status v1beta1.RestoreMemberStatus) v1beta1.RestorePhase {
	if kmapi.IsConditionFalse(status.Conditions, v1beta1.RestoreExecutorEnsured) ||
		kmapi.IsConditionFalse(status.Conditions, v1beta1.PreRestoreHookExecutionSucceeded) ||
		kmapi.IsConditionFalse(status.Conditions, v1beta1.PostRestoreHookExecutionSucceeded) {
		return v1beta1.RestoreFailed
	}

	if len(status.Conditions) == 0 || isAllTargetRestorePending([]v1beta1.RestoreMemberStatus{status}) {
		return v1beta1.RestorePending
	}

	if RestoreCompletedForAllTargets([]v1beta1.RestoreMemberStatus{status}) {
		if status.Phase == v1beta1.TargetRestorePhaseUnknown {
			return v1beta1.RestorePhaseUnknown
		}

		if status.Phase == v1beta1.TargetRestoreFailed ||
			kmapi.IsConditionFalse(status.Conditions, v1beta1.MetricsPushed) {
			return v1beta1.RestoreFailed
		}

		if kmapi.IsConditionTrue(status.Conditions, v1beta1.MetricsPushed) {
			return v1beta1.RestoreSucceeded
		}
	}

	if status.Phase == v1beta1.TargetRestorePending ||
		kmapi.IsConditionFalse(status.Conditions, v1beta1.RepositoryFound) ||
		kmapi.IsConditionFalse(status.Conditions, v1beta1.BackendSecretFound) ||
		kmapi.IsConditionFalse(status.Conditions, v1beta1.RestoreTargetFound) {
		return v1beta1.RestorePending
	}

	if kmapi.IsConditionFalse(status.Conditions, v1beta1.ValidationPassed) {
		return v1beta1.RestorePhaseInvalid
	}

	return v1beta1.RestoreRunning
}

func RestoreCompletedForAllTargets(status []v1beta1.RestoreMemberStatus) bool {
	for _, t := range status {
		if t.TotalHosts == nil || !restoreCompletedForAllHosts(t.Stats, *t.TotalHosts) {
			return false
		}
	}
	return len(status) > 0
}

func restoreCompletedForAllHosts(status []v1beta1.HostRestoreStats, totalHosts int32) bool {
	for _, h := range status {
		if !hostRestoreCompleted(h.Phase) {
			return false
		}
	}
	return len(status) == int(totalHosts)
}

func hostRestoreCompleted(phase v1beta1.HostRestorePhase) bool {
	return phase == v1beta1.HostRestoreSucceeded ||
		phase == v1beta1.HostRestoreFailed ||
		phase == v1beta1.HostRestoreUnknown
}

func isAllTargetRestorePending(status []v1beta1.RestoreMemberStatus) bool {
	for _, m := range status {
		if m.Phase != v1beta1.TargetRestorePending {
			return false
		}
	}
	return true
}

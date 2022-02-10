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
	"strings"

	"stash.appscode.dev/apimachinery/apis"
	"stash.appscode.dev/apimachinery/apis/stash/v1beta1"
	cs "stash.appscode.dev/apimachinery/client/clientset/versioned"

	core "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	kmapi "kmodules.xyz/client-go/api/v1"
	"kmodules.xyz/client-go/meta"
	ofst "kmodules.xyz/offshoot-api/api/v1"
)

const (
	EventSourceRestoreBatchController   = "RestoreBatch Controller"
	EventSourceRestoreSessionController = "RestoreSession Controller"
)

type RestoreInvoker interface {
	MetadataHandler
	ConditionHandler
	RestoreExecutionOrderHandler
	RestoreTargetHandler
	RepositoryGetter
	DriverHandler
	Eventer
	KubeDBIntegrator
	ObjectFormatter
	RestoreStatusHandler
}

type RestoreExecutionOrderHandler interface {
	GetExecutionOrder() v1beta1.ExecutionOrder
	NextInOrder(curTarget v1beta1.TargetRef, targetStatus []v1beta1.RestoreMemberStatus) bool
}

type RestoreTargetHandler interface {
	GetTargetInfo() []RestoreTargetInfo
	GetGlobalHooks() *v1beta1.RestoreHooks
}

type RestoreStatusHandler interface {
	GetStatus() RestoreInvokerStatus
	UpdateStatus(newStatus RestoreInvokerStatus) error
}

type RestoreInvokerStatus struct {
	Phase           v1beta1.RestorePhase
	SessionDuration string
	Conditions      []kmapi.Condition
	TargetStatus    []v1beta1.RestoreMemberStatus
}

type RestoreTargetInfo struct {
	Task                  v1beta1.TaskRef
	Target                *v1beta1.RestoreTarget
	RuntimeSettings       ofst.RuntimeSettings
	TempDir               v1beta1.EmptyDirSettings
	InterimVolumeTemplate *ofst.PersistentVolumeClaim
	Hooks                 *v1beta1.RestoreHooks
}

func NewRestoreInvoker(kubeClient kubernetes.Interface, stashClient cs.Interface, kind, name, namespace string) (RestoreInvoker, error) {
	switch kind {
	case v1beta1.ResourceKindRestoreSession:
		restoreSession, err := stashClient.StashV1beta1().RestoreSessions(namespace).Get(context.TODO(), name, metav1.GetOptions{})
		if err != nil {
			return nil, err
		}
		return &RestoreSessionInvoker{kubeClient: kubeClient, stashClient: stashClient, restoreSession: restoreSession}, nil
	case v1beta1.ResourceKindRestoreBatch:
		restoreBatch, err := stashClient.StashV1beta1().RestoreBatches(namespace).Get(context.TODO(), name, metav1.GetOptions{})
		if err != nil {
			return nil, err
		}
		return &RestoreBatchInvoker{kubeClient: kubeClient, stashClient: stashClient, restoreBatch: restoreBatch}, nil
	default:
		return nil, fmt.Errorf("unknown backup invoker kind: %s", kind)
	}
}

func hasRestoreMemberCondition(status []v1beta1.RestoreMemberStatus, target v1beta1.TargetRef, condType string) bool {
	// If the target is present in the list, then return the respective value
	for i := range status {
		if TargetMatched(status[i].Ref, target) {
			return kmapi.HasCondition(status[i].Conditions, condType)
		}
	}
	// Member is not present in the list, so the condition is not there too
	return false
}

func getRestoreMemberCondition(status []v1beta1.RestoreMemberStatus, target v1beta1.TargetRef, condType string) (int, *kmapi.Condition) {
	// If the target is present in the list, then return the respective condition
	for i := range status {
		if TargetMatched(status[i].Ref, target) {
			return kmapi.GetCondition(status[i].Conditions, condType)
		}
	}
	// Member is not present in the list
	return -1, nil
}

func setRestoreMemberCondition(status []v1beta1.RestoreMemberStatus, target v1beta1.TargetRef, newCondition kmapi.Condition) []v1beta1.RestoreMemberStatus {
	// If the target is already exist in the list, update its condition
	for i := range status {
		if TargetMatched(status[i].Ref, target) {
			status[i].Conditions = kmapi.SetCondition(status[i].Conditions, newCondition)
			return status
		}
	}
	// The target does not exist in the list. So, add a new entry.
	memberStatus := v1beta1.RestoreMemberStatus{
		Ref:        target,
		Conditions: kmapi.SetCondition(nil, newCondition),
	}
	return upsertRestoreMemberStatus(status, memberStatus)
}

func isRestoreMemberConditionTrue(status []v1beta1.RestoreMemberStatus, target v1beta1.TargetRef, condType string) bool {
	// If the target is present in the list, then return the respective value
	for i := range status {
		if TargetMatched(status[i].Ref, target) {
			return kmapi.IsConditionTrue(status[i].Conditions, condType)
		}
	}
	// Member is not present in the list, so the condition is false
	return false
}

func upsertConditions(cur []kmapi.Condition, new []kmapi.Condition) []kmapi.Condition {
	for i := range new {
		cur = kmapi.SetCondition(cur, new[i])
	}
	return cur
}

func upsertRestoreHostStatus(cur []v1beta1.HostRestoreStats, new []v1beta1.HostRestoreStats) []v1beta1.HostRestoreStats {
	for i := range new {
		index, hostEntryExist := hostEntryIndex(cur, new[i])
		if hostEntryExist {
			cur[index] = new[i]
		} else {
			cur = append(cur, new[i])
		}
	}
	return cur
}

func hostEntryIndex(entries []v1beta1.HostRestoreStats, target v1beta1.HostRestoreStats) (int, bool) {
	for i := range entries {
		if entries[i].Hostname == target.Hostname {
			return i, true
		}
	}
	return -1, false
}

func getInvokerStatusFromRestoreBatch(restoreBatch *v1beta1.RestoreBatch) RestoreInvokerStatus {
	return RestoreInvokerStatus{
		Phase:           restoreBatch.Status.Phase,
		SessionDuration: restoreBatch.Status.SessionDuration,
		Conditions:      restoreBatch.Status.Conditions,
		TargetStatus:    restoreBatch.Status.Members,
	}
}

func getInvokerStatusFromRestoreSession(restoreSession *v1beta1.RestoreSession) RestoreInvokerStatus {
	invokerStatus := RestoreInvokerStatus{
		Phase:           restoreSession.Status.Phase,
		SessionDuration: restoreSession.Status.SessionDuration,
		Conditions:      restoreSession.Status.Conditions,
	}
	if restoreSession.Spec.Target != nil {
		invokerStatus.TargetStatus = append(invokerStatus.TargetStatus, v1beta1.RestoreMemberStatus{
			Ref:        restoreSession.Spec.Target.Ref,
			Conditions: restoreSession.Status.Conditions,
			TotalHosts: restoreSession.Status.TotalHosts,
			Phase:      v1beta1.RestoreTargetPhase(restoreSession.Status.Phase),
			Stats:      restoreSession.Status.Stats,
		})
	}
	return invokerStatus
}

func TargetRestoreCompleted(ref v1beta1.TargetRef, targetStatus []v1beta1.RestoreMemberStatus) bool {
	for i := range targetStatus {
		if TargetMatched(ref, targetStatus[i].Ref) {
			return targetStatus[i].Phase == v1beta1.TargetRestoreSucceeded ||
				targetStatus[i].Phase == v1beta1.TargetRestoreFailed ||
				targetStatus[i].Phase == v1beta1.TargetRestorePhaseUnknown
		}
	}
	return false
}

func extractLabels(in map[string]string, keys ...string) (map[string]string, error) {
	out := make(map[string]string, len(keys))
	for _, k := range keys {
		val, err := meta.GetStringValue(in, k)
		if err != nil {
			return nil, err
		}
		out[k] = val
	}
	return out, nil
}

func TargetOfGroupKind(targetRef v1beta1.TargetRef, group, kind string) bool {
	gv := strings.Split(targetRef.APIVersion, "/")
	if len(gv) > 0 && gv[0] == group && targetRef.Kind == kind {
		return true
	}
	return false
}

func upsertRestoreTargetStatus(cur, new v1beta1.RestoreMemberStatus) v1beta1.RestoreMemberStatus {
	if len(new.Conditions) > 0 {
		cur.Conditions = upsertConditions(cur.Conditions, new.Conditions)
	}

	if new.TotalHosts != nil {
		cur.TotalHosts = new.TotalHosts
	}

	if len(new.Stats) > 0 {
		cur.Stats = upsertRestoreHostStatus(cur.Stats, new.Stats)
	}

	cur.Phase = calculateRestoreTargetPhase(cur)
	return cur
}

func calculateRestoreTargetPhase(status v1beta1.RestoreMemberStatus) v1beta1.RestoreTargetPhase {
	if kmapi.IsConditionFalse(status.Conditions, apis.RestorerEnsured) ||
		kmapi.IsConditionFalse(status.Conditions, apis.MetricsPushed) {
		return v1beta1.TargetRestoreFailed
	}

	allConditionTrue := true
	for _, c := range status.Conditions {
		if c.Status != core.ConditionTrue {
			allConditionTrue = false
		}
	}
	if !allConditionTrue || status.TotalHosts == nil {
		return v1beta1.TargetRestorePending
	}

	failedHostCount := int32(0)
	unknownHostCount := int32(0)
	successfulHostCount := int32(0)
	for _, hostStats := range status.Stats {
		switch hostStats.Phase {
		case v1beta1.HostRestoreFailed:
			failedHostCount++
		case v1beta1.HostRestoreUnknown:
			unknownHostCount++
		case v1beta1.HostRestoreSucceeded:
			successfulHostCount++
		}
	}

	completedHosts := successfulHostCount + failedHostCount + unknownHostCount
	if completedHosts < *status.TotalHosts {
		return v1beta1.TargetRestoreRunning
	}

	if failedHostCount > 0 {
		return v1beta1.TargetRestoreFailed
	}
	if unknownHostCount > 0 {
		return v1beta1.TargetRestorePhaseUnknown
	}

	return v1beta1.TargetRestoreSucceeded
}

func IsRestoreCompleted(phase v1beta1.RestorePhase) bool {
	return phase == v1beta1.RestoreSucceeded ||
		phase == v1beta1.RestoreFailed ||
		phase == v1beta1.RestorePhaseUnknown
}

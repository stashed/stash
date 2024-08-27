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
	"time"

	"stash.appscode.dev/apimachinery/apis/stash/v1beta1"
	cs "stash.appscode.dev/apimachinery/client/clientset/versioned"
	stash_util "stash.appscode.dev/apimachinery/client/clientset/versioned/typed/stash/v1beta1/util"

	"gomodules.xyz/x/arrays"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	kmapi "kmodules.xyz/client-go/api/v1"
	cutil "kmodules.xyz/client-go/conditions"
)

type BackupSessionHandler struct {
	stashClient   cs.Interface
	backupSession *v1beta1.BackupSession
}

func NewBackupSessionHandler(stashClient cs.Interface, backupSession *v1beta1.BackupSession) *BackupSessionHandler {
	return &BackupSessionHandler{
		stashClient:   stashClient,
		backupSession: backupSession,
	}
}

func (h *BackupSessionHandler) UpdateStatus(status *v1beta1.BackupSessionStatus) error {
	updatedBackupSession, err := stash_util.UpdateBackupSessionStatus(
		context.TODO(),
		h.stashClient.StashV1beta1(),
		h.backupSession.ObjectMeta,
		func(in *v1beta1.BackupSessionStatus) (types.UID, *v1beta1.BackupSessionStatus) {
			in.Conditions = upsertConditions(in.Conditions, status.Conditions)

			if len(status.Targets) > 0 {
				for i := range status.Targets {
					in.Targets = upsertBackupMembersStatus(in.Targets, status.Targets[i])
				}
			}

			in.Phase = calculateBackupSessionPhase(in)
			if IsBackupCompleted(in.Phase) && in.SessionDuration == "" {
				in.SessionDuration = time.Since(h.backupSession.ObjectMeta.CreationTimestamp.Time).Round(time.Second).String()
			}
			if in.SessionDeadline.IsZero() {
				in.SessionDeadline = status.SessionDeadline
			}

			if in.Retried == nil {
				in.Retried = status.Retried
			}

			if in.NextRetry == nil {
				in.NextRetry = status.NextRetry
			}

			return h.backupSession.ObjectMeta.UID, in
		},
		metav1.UpdateOptions{},
	)
	if err != nil {
		return err
	}
	h.backupSession = updatedBackupSession
	return nil
}

func (h *BackupSessionHandler) GetObjectMeta() metav1.ObjectMeta {
	return h.backupSession.ObjectMeta
}

func (h *BackupSessionHandler) GetStatus() v1beta1.BackupSessionStatus {
	return h.backupSession.Status
}

func (h *BackupSessionHandler) GetTargetStatus() []v1beta1.BackupTargetStatus {
	return h.backupSession.Status.Targets
}

func (h *BackupSessionHandler) GetConditions() []kmapi.Condition {
	return h.backupSession.Status.Conditions
}

func (h *BackupSessionHandler) GetTargetConditions(target v1beta1.TargetRef) []kmapi.Condition {
	for _, t := range h.backupSession.Status.Targets {
		if TargetMatched(t.Ref, target) {
			return t.Conditions
		}
	}
	return nil
}

func (h *BackupSessionHandler) GetInvoker() (BackupInvoker, error) {
	return NewBackupInvoker(h.stashClient, h.backupSession.Spec.Invoker.Kind, h.backupSession.Spec.Invoker.Name, h.backupSession.Namespace)
}

func (h *BackupSessionHandler) GetInvokerRef() v1beta1.BackupInvokerRef {
	return h.backupSession.Spec.Invoker
}

func (h *BackupSessionHandler) GetBackupSession() *v1beta1.BackupSession {
	b := h.backupSession.DeepCopy()
	return b
}

func IsBackupCompleted(phase v1beta1.BackupSessionPhase) bool {
	return phase == v1beta1.BackupSessionSucceeded ||
		phase == v1beta1.BackupSessionFailed ||
		phase == v1beta1.BackupSessionSkipped ||
		phase == v1beta1.BackupSessionUnknown
}

func BackupCompletedForAllTargets(status []v1beta1.BackupTargetStatus, totalTargets int) bool {
	for _, t := range status {
		if t.Phase == v1beta1.TargetBackupFailed || t.Phase == v1beta1.TargetBackupSucceeded {
			continue
		}
		if t.TotalHosts == nil || !backupCompletedForAllHosts(t.Stats, *t.TotalHosts) {
			return false
		}
	}
	return len(status) == totalTargets
}

func backupCompletedForAllHosts(status []v1beta1.HostBackupStats, totalHosts int32) bool {
	for _, h := range status {
		if !hostBackupCompleted(h.Phase) {
			return false
		}
	}
	return len(status) == int(totalHosts)
}

func hostBackupCompleted(phase v1beta1.HostBackupPhase) bool {
	return phase == v1beta1.HostBackupSucceeded ||
		phase == v1beta1.HostBackupFailed
}

func upsertBackupMembersStatus(cur []v1beta1.BackupTargetStatus, new v1beta1.BackupTargetStatus) []v1beta1.BackupTargetStatus {
	// if the member status already exist, then update it
	for i := range cur {
		if TargetMatched(cur[i].Ref, new.Ref) {
			cur[i] = upsertBackupTargetStatus(cur[i], new)
			return cur
		}
	}

	// the member status does not exist. so, add new entry.
	new.Phase = calculateBackupTargetPhase(new)
	cur = append(cur, new)
	return cur
}

func upsertBackupTargetStatus(cur, new v1beta1.BackupTargetStatus) v1beta1.BackupTargetStatus {
	if len(new.Conditions) > 0 {
		cur.Conditions = upsertConditions(cur.Conditions, new.Conditions)
	}

	if new.TotalHosts != nil {
		cur.TotalHosts = new.TotalHosts
	}

	if len(new.Stats) > 0 {
		cur.Stats = upsertBackupHostStatus(cur.Stats, new.Stats)
	}

	if len(new.PreBackupActions) > 0 {
		cur.PreBackupActions = upsertArray(cur.PreBackupActions, new.PreBackupActions)
	}

	if len(new.PostBackupActions) > 0 {
		cur.PostBackupActions = upsertArray(cur.PostBackupActions, new.PostBackupActions)
	}

	cur.Phase = calculateBackupTargetPhase(cur)
	return cur
}

func upsertBackupHostStatus(cur, new []v1beta1.HostBackupStats) []v1beta1.HostBackupStats {
	for i := range new {
		index, hostEntryExist := backupHostEntryIndex(cur, new[i])
		if hostEntryExist {
			cur[index] = new[i]
		} else {
			cur = append(cur, new[i])
		}
	}
	return cur
}

func calculateBackupTargetPhase(status v1beta1.BackupTargetStatus) v1beta1.TargetPhase {
	if cutil.IsConditionFalse(status.Conditions, v1beta1.BackupExecutorEnsured) ||
		cutil.IsConditionFalse(status.Conditions, v1beta1.PreBackupHookExecutionSucceeded) ||
		cutil.IsConditionTrue(status.Conditions, v1beta1.BackupDisrupted) ||
		cutil.IsConditionFalse(status.Conditions, v1beta1.PostBackupHookExecutionSucceeded) {
		return v1beta1.TargetBackupFailed
	}

	if status.TotalHosts == nil {
		return v1beta1.TargetBackupPending
	}

	failedHostCount := int32(0)
	successfulHostCount := int32(0)
	for _, hostStats := range status.Stats {
		switch hostStats.Phase {
		case v1beta1.HostBackupFailed:
			failedHostCount++
		case v1beta1.HostBackupSucceeded:
			successfulHostCount++
		}
	}
	completedHosts := successfulHostCount + failedHostCount

	if completedHosts == *status.TotalHosts {
		if failedHostCount > 0 {
			return v1beta1.TargetBackupFailed
		}
		return v1beta1.TargetBackupSucceeded
	}
	return v1beta1.TargetBackupRunning
}

func calculateBackupSessionPhase(status *v1beta1.BackupSessionStatus) v1beta1.BackupSessionPhase {
	if cutil.IsConditionFalse(status.Conditions, v1beta1.MetricsPushed) {
		return v1beta1.BackupSessionFailed
	}

	if cutil.IsConditionTrue(status.Conditions, v1beta1.BackupSkipped) {
		return v1beta1.BackupSessionSkipped
	}

	if cutil.IsConditionTrue(status.Conditions, v1beta1.MetricsPushed) &&
		(cutil.IsConditionTrue(status.Conditions, v1beta1.DeadlineExceeded) ||
			cutil.IsConditionFalse(status.Conditions, v1beta1.BackupHistoryCleaned) ||
			cutil.IsConditionFalse(status.Conditions, v1beta1.GlobalPreBackupHookSucceeded) ||
			cutil.IsConditionFalse(status.Conditions, v1beta1.GlobalPostBackupHookSucceeded)) {
		return v1beta1.BackupSessionFailed
	}

	if len(status.Targets) == 0 || isAllTargetBackupPending(status.Targets) {
		return v1beta1.BackupSessionPending
	}

	failedTargetCount := 0
	successfulTargetCount := 0

	for _, t := range status.Targets {
		switch t.Phase {
		case v1beta1.TargetBackupFailed:
			failedTargetCount++
		case v1beta1.TargetBackupSucceeded:
			successfulTargetCount++
		}
	}
	completedTargets := successfulTargetCount + failedTargetCount

	if completedTargets == len(status.Targets) && cutil.IsConditionTrue(status.Conditions, v1beta1.MetricsPushed) { // Pushing metrics is the last step.
		if failedTargetCount > 0 ||
			cutil.IsConditionFalse(status.Conditions, v1beta1.RetentionPolicyApplied) ||
			cutil.IsConditionFalse(status.Conditions, v1beta1.RepositoryMetricsPushed) ||
			cutil.IsConditionFalse(status.Conditions, v1beta1.RepositoryIntegrityVerified) {
			return v1beta1.BackupSessionFailed
		}
		return v1beta1.BackupSessionSucceeded
	}

	return v1beta1.BackupSessionRunning
}

func backupHostEntryIndex(entries []v1beta1.HostBackupStats, target v1beta1.HostBackupStats) (int, bool) {
	for i := range entries {
		if entries[i].Hostname == target.Hostname {
			return i, true
		}
	}
	return -1, false
}

func upsertArray(cur, new []string) []string {
	for i := range new {
		if exist, idx := arrays.Contains(cur, new[i]); exist {
			cur[idx] = new[i]
			continue
		}
		cur = append(cur, new[i])
	}
	return cur
}

func isAllTargetBackupPending(status []v1beta1.BackupTargetStatus) bool {
	for _, t := range status {
		if t.Phase != v1beta1.TargetBackupPending && t.Phase != "" {
			return false
		}
	}
	return true
}

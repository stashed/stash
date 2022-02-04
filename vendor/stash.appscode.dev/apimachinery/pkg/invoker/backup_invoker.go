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

	"stash.appscode.dev/apimachinery/apis"
	"stash.appscode.dev/apimachinery/apis/stash/v1alpha1"
	"stash.appscode.dev/apimachinery/apis/stash/v1beta1"
	cs "stash.appscode.dev/apimachinery/client/clientset/versioned"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	kmapi "kmodules.xyz/client-go/api/v1"
	ofst "kmodules.xyz/offshoot-api/api/v1"
)

type BackupInvoker interface {
	MetadataHandler
	ConditionHandler
	BackupExecutionOrderHandler
	BackupTargetHandler
	RepositoryGetter
	DriverHandler
	ObjectFormatter
	BackupInvokerStatusHandler
}

type BackupExecutionOrderHandler interface {
	GetExecutionOrder() v1beta1.ExecutionOrder
	NextInOrder(curTarget v1beta1.TargetRef, targetStatus []v1beta1.BackupTargetStatus) bool
}

type BackupTargetHandler interface {
	GetTargetInfo() []BackupTargetInfo
	GetRuntimeSettings() ofst.RuntimeSettings
	GetSchedule() string
	GetRetentionPolicy() v1alpha1.RetentionPolicy
	IsPaused() bool
	GetBackupHistoryLimit() *int32
	GetGlobalHooks() *v1beta1.BackupHooks
}

type BackupInvokerStatusHandler interface {
	GetPhase() v1beta1.BackupInvokerPhase
}

type BackupTargetInfo struct {
	Task                  v1beta1.TaskRef
	Target                *v1beta1.BackupTarget
	RuntimeSettings       ofst.RuntimeSettings
	TempDir               v1beta1.EmptyDirSettings
	InterimVolumeTemplate *ofst.PersistentVolumeClaim
	Hooks                 *v1beta1.BackupHooks
}

func NewBackupInvoker(stashClient cs.Interface, kind, name, namespace string) (BackupInvoker, error) {
	switch kind {
	case v1beta1.ResourceKindBackupConfiguration:
		backupConfig, err := stashClient.StashV1beta1().BackupConfigurations(namespace).Get(context.TODO(), name, metav1.GetOptions{})
		if err != nil {
			return nil, err
		}
		return &BackupConfigurationInvoker{stashClient: stashClient, backupConfig: backupConfig}, nil
	case v1beta1.ResourceKindBackupBatch:
		backupBatch, err := stashClient.StashV1beta1().BackupBatches(namespace).Get(context.TODO(), name, metav1.GetOptions{})
		if err != nil {
			return nil, err
		}
		return &BackupBatchInvoker{stashClient: stashClient, backupBatch: backupBatch}, nil
	default:
		return nil, fmt.Errorf("unknown backup invoker kind: %s", kind)
	}
}

func hasMemberCondition(conditions []v1beta1.MemberConditions, target v1beta1.TargetRef, condType string) bool {
	// If the target is present in the list, then return the respective value
	for i := range conditions {
		if TargetMatched(conditions[i].Target, target) {
			return kmapi.HasCondition(conditions[i].Conditions, condType)
		}
	}
	// Member is not present in the list, so the condition is not there too
	return false
}

func getMemberCondition(conditions []v1beta1.MemberConditions, target v1beta1.TargetRef, condType string) (int, *kmapi.Condition) {
	// If the target is present in the list, then return the respective condition
	for i := range conditions {
		if TargetMatched(conditions[i].Target, target) {
			return kmapi.GetCondition(conditions[i].Conditions, condType)
		}
	}
	// Member is not present in the list
	return -1, nil
}

func TargetMatched(t1, t2 v1beta1.TargetRef) bool {
	return t1.APIVersion == t2.APIVersion && t1.Kind == t2.Kind && t1.Name == t2.Name
}

func setMemberCondition(conditions []v1beta1.MemberConditions, target v1beta1.TargetRef, newCondition kmapi.Condition) []v1beta1.MemberConditions {
	// If the target is already exist in the list, update its condition
	for i := range conditions {
		if TargetMatched(conditions[i].Target, target) {
			conditions[i].Conditions = kmapi.SetCondition(conditions[i].Conditions, newCondition)
			return conditions
		}
	}
	// The target does not exist in the list. So, add a new entry.
	memberConditions := v1beta1.MemberConditions{
		Target:     target,
		Conditions: kmapi.SetCondition(nil, newCondition),
	}
	return append(conditions, memberConditions)
}

func isMemberConditionTrue(conditions []v1beta1.MemberConditions, target v1beta1.TargetRef, condType string) bool {
	// If the target is present in the list, then return the respective value
	for i := range conditions {
		if TargetMatched(conditions[i].Target, target) {
			return kmapi.IsConditionTrue(conditions[i].Conditions, condType)
		}
	}
	// Member is not present in the list, so the condition is false
	return false
}

func TargetBackupInitiated(ref v1beta1.TargetRef, targetStatus []v1beta1.BackupTargetStatus) bool {
	for i := range targetStatus {
		if TargetMatched(ref, targetStatus[i].Ref) {
			return targetStatus[i].Phase == v1beta1.TargetBackupRunning ||
				targetStatus[i].Phase == v1beta1.TargetBackupSucceeded ||
				targetStatus[i].Phase == v1beta1.TargetBackupFailed
		}
	}
	return false
}

func TargetBackupCompleted(ref v1beta1.TargetRef, targetStatus []v1beta1.BackupTargetStatus) bool {
	for i := range targetStatus {
		if TargetMatched(ref, targetStatus[i].Ref) {
			return targetStatus[i].Phase == v1beta1.TargetBackupSucceeded ||
				targetStatus[i].Phase == v1beta1.TargetBackupFailed
		}
	}
	return false
}

func isConditionSatisfied(conditions []kmapi.Condition, condType string) bool {
	if kmapi.IsConditionFalse(conditions, condType) || kmapi.IsConditionUnknown(conditions, condType) {
		return false
	}

	return true
}

func calculateBackupInvokerPhase(driver v1beta1.Snapshotter, conditions []kmapi.Condition) v1beta1.BackupInvokerPhase {
	if !isConditionSatisfied(conditions, apis.RepositoryFound) ||
		!isConditionSatisfied(conditions, apis.BackendSecretFound) {
		return v1beta1.BackupInvokerNotReady
	}

	if kmapi.IsConditionFalse(conditions, apis.ValidationPassed) {
		return v1beta1.BackupInvokerInvalid
	}

	if kmapi.IsConditionTrue(conditions, apis.ValidationPassed) &&
		kmapi.IsConditionTrue(conditions, apis.CronJobCreated) &&
		backendRequirementsSatisfied(driver, conditions) {
		return v1beta1.BackupInvokerReady
	}

	return v1beta1.BackupInvokerNotReady
}

func backendRequirementsSatisfied(driver v1beta1.Snapshotter, conditions []kmapi.Condition) bool {
	if driver == v1beta1.ResticSnapshotter {
		return kmapi.IsConditionTrue(conditions, apis.RepositoryFound) && kmapi.IsConditionTrue(conditions, apis.BackendSecretFound)
	}
	return true
}

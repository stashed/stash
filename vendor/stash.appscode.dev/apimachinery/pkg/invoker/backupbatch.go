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

	"stash.appscode.dev/apimachinery/apis/stash/v1alpha1"
	"stash.appscode.dev/apimachinery/apis/stash/v1beta1"
	cs "stash.appscode.dev/apimachinery/client/clientset/versioned"
	stash_scheme "stash.appscode.dev/apimachinery/client/clientset/versioned/scheme"
	v1beta1_util "stash.appscode.dev/apimachinery/client/clientset/versioned/typed/stash/v1beta1/util"

	"gomodules.xyz/pointer"
	core "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/reference"
	kmapi "kmodules.xyz/client-go/api/v1"
	core_util "kmodules.xyz/client-go/core/v1"
	"kmodules.xyz/client-go/meta"
	ofst "kmodules.xyz/offshoot-api/api/v1"
)

type BackupBatchInvoker struct {
	backupBatch *v1beta1.BackupBatch
	stashClient cs.Interface
}

func NewBackupBatchInvoker(stashClient cs.Interface, backupBatch *v1beta1.BackupBatch) BackupInvoker {
	return &BackupBatchInvoker{
		stashClient: stashClient,
		backupBatch: backupBatch,
	}
}

func (inv *BackupBatchInvoker) GetObjectMeta() metav1.ObjectMeta {
	return inv.backupBatch.ObjectMeta
}

func (inv *BackupBatchInvoker) GetTypeMeta() metav1.TypeMeta {
	return metav1.TypeMeta{
		Kind:       v1beta1.ResourceKindBackupBatch,
		APIVersion: v1beta1.SchemeGroupVersion.String(),
	}
}

func (inv *BackupBatchInvoker) GetObjectRef() (*core.ObjectReference, error) {
	return reference.GetReference(stash_scheme.Scheme, inv.backupBatch)
}

func (inv *BackupBatchInvoker) GetOwnerRef() *metav1.OwnerReference {
	return metav1.NewControllerRef(inv.backupBatch, v1beta1.SchemeGroupVersion.WithKind(v1beta1.ResourceKindBackupBatch))
}

func (inv *BackupBatchInvoker) GetLabels() map[string]string {
	return inv.backupBatch.OffshootLabels()
}

func (inv *BackupBatchInvoker) AddFinalizer() error {
	updatedBackupBatch, _, err := v1beta1_util.PatchBackupBatch(context.TODO(), inv.stashClient.StashV1beta1(), inv.backupBatch, func(in *v1beta1.BackupBatch) *v1beta1.BackupBatch {
		in.ObjectMeta = core_util.AddFinalizer(in.ObjectMeta, v1beta1.StashKey)
		return in
	}, metav1.PatchOptions{})
	if err != nil {
		return err
	}
	inv.backupBatch = updatedBackupBatch
	return nil
}

func (inv *BackupBatchInvoker) RemoveFinalizer() error {
	updatedBackupBatch, _, err := v1beta1_util.PatchBackupBatch(context.TODO(), inv.stashClient.StashV1beta1(), inv.backupBatch, func(in *v1beta1.BackupBatch) *v1beta1.BackupBatch {
		in.ObjectMeta = core_util.RemoveFinalizer(in.ObjectMeta, v1beta1.StashKey)
		return in
	}, metav1.PatchOptions{})
	if err != nil {
		return err
	}
	inv.backupBatch = updatedBackupBatch
	return nil
}

func (inv *BackupBatchInvoker) HasCondition(target *v1beta1.TargetRef, conditionType string) (bool, error) {
	backupBatch, err := inv.stashClient.StashV1beta1().BackupBatches(inv.backupBatch.Namespace).Get(context.TODO(), inv.backupBatch.Name, metav1.GetOptions{})
	if err != nil {
		return false, err
	}
	if target != nil {
		return hasMemberCondition(backupBatch.Status.MemberConditions, *target, conditionType), nil
	}
	return kmapi.HasCondition(backupBatch.Status.Conditions, conditionType), nil
}

func (inv *BackupBatchInvoker) GetCondition(target *v1beta1.TargetRef, conditionType string) (int, *kmapi.Condition, error) {
	backupBatch, err := inv.stashClient.StashV1beta1().BackupBatches(inv.backupBatch.Namespace).Get(context.TODO(), inv.backupBatch.Name, metav1.GetOptions{})
	if err != nil {
		return -1, nil, err
	}
	if target != nil {
		idx, cond := getMemberCondition(backupBatch.Status.MemberConditions, *target, conditionType)
		return idx, cond, nil
	}
	idx, cond := kmapi.GetCondition(backupBatch.Status.Conditions, conditionType)
	return idx, cond, nil
}

func (inv *BackupBatchInvoker) SetCondition(target *v1beta1.TargetRef, newCondition kmapi.Condition) error {
	updatedBackupBatch, err := v1beta1_util.UpdateBackupBatchStatus(context.TODO(), inv.stashClient.StashV1beta1(), inv.backupBatch.ObjectMeta, func(in *v1beta1.BackupBatchStatus) (types.UID, *v1beta1.BackupBatchStatus) {
		if target != nil {
			in.MemberConditions = setMemberCondition(in.MemberConditions, *target, newCondition)
		} else {
			in.Conditions = kmapi.SetCondition(in.Conditions, newCondition)
		}
		in.Phase = CalculateBackupInvokerPhase(inv.GetDriver(), in.Conditions)
		return inv.backupBatch.UID, in
	}, metav1.UpdateOptions{})
	if err != nil {
		return err
	}
	inv.backupBatch = updatedBackupBatch
	return nil
}

func (inv *BackupBatchInvoker) IsConditionTrue(target *v1beta1.TargetRef, conditionType string) (bool, error) {
	backupBatch, err := inv.stashClient.StashV1beta1().BackupBatches(inv.backupBatch.Namespace).Get(context.TODO(), inv.backupBatch.Name, metav1.GetOptions{})
	if err != nil {
		return false, err
	}
	if target != nil {
		return isMemberConditionTrue(backupBatch.Status.MemberConditions, *target, conditionType), nil
	}
	return kmapi.IsConditionTrue(backupBatch.Status.Conditions, conditionType), nil
}

func (inv *BackupBatchInvoker) GetTargetInfo() []BackupTargetInfo {
	var targetInfo []BackupTargetInfo
	for _, member := range inv.backupBatch.Spec.Members {
		targetInfo = append(targetInfo, BackupTargetInfo{
			Task:                  member.Task,
			Target:                getBackupTarget(member.Target, inv.backupBatch.Namespace),
			RuntimeSettings:       member.RuntimeSettings,
			TempDir:               member.TempDir,
			InterimVolumeTemplate: member.InterimVolumeTemplate,
			Hooks:                 member.Hooks,
		})
	}
	return targetInfo
}

func getBackupTarget(target *v1beta1.BackupTarget, invNamespace string) *v1beta1.BackupTarget {
	if target == nil {
		return &v1beta1.BackupTarget{
			Ref: v1beta1.EmptyTargetRef(),
		}
	}
	if target.Ref.Namespace == "" {
		target.Ref.Namespace = invNamespace
	}
	return target
}

func (inv *BackupBatchInvoker) GetDriver() v1beta1.Snapshotter {
	driver := inv.backupBatch.Spec.Driver
	if driver == "" {
		driver = v1beta1.ResticSnapshotter
	}
	return driver
}

func (inv *BackupBatchInvoker) GetTimeOut() string {
	return inv.backupBatch.Spec.TimeOut
}

func (inv *BackupBatchInvoker) GetRepoRef() kmapi.ObjectReference {
	var repo kmapi.ObjectReference
	repo.Name = inv.backupBatch.Spec.Repository.Name
	repo.Namespace = inv.backupBatch.Spec.Repository.Namespace
	if repo.Namespace == "" {
		repo.Namespace = inv.backupBatch.Namespace
	}
	return repo
}

func (inv *BackupBatchInvoker) GetRepository() (*v1alpha1.Repository, error) {
	repo := inv.GetRepoRef()
	return inv.stashClient.StashV1alpha1().Repositories(repo.Namespace).Get(context.TODO(), repo.Name, metav1.GetOptions{})
}

func (inv *BackupBatchInvoker) GetRuntimeSettings() ofst.RuntimeSettings {
	return inv.backupBatch.Spec.RuntimeSettings
}

func (inv *BackupBatchInvoker) GetSchedule() string {
	return inv.backupBatch.Spec.Schedule
}

func (inv *BackupBatchInvoker) IsPaused() bool {
	return inv.backupBatch.Spec.Paused
}

func (inv *BackupBatchInvoker) GetBackupHistoryLimit() *int32 {
	return inv.backupBatch.Spec.BackupHistoryLimit
}

func (inv *BackupBatchInvoker) GetGlobalHooks() *v1beta1.BackupHooks {
	return inv.backupBatch.Spec.Hooks
}

func (inv *BackupBatchInvoker) GetExecutionOrder() v1beta1.ExecutionOrder {
	return inv.backupBatch.Spec.ExecutionOrder
}

func (inv *BackupBatchInvoker) NextInOrder(curTarget v1beta1.TargetRef, targetStatus []v1beta1.BackupTargetStatus) bool {
	for _, t := range inv.GetTargetInfo() {
		if t.Target != nil {
			if TargetMatched(t.Target.Ref, curTarget) {
				return true
			}
			if !TargetBackupCompleted(t.Target.Ref, targetStatus) {
				return false
			}
		}
	}
	// By default, return true so that nil target(i.e. cluster backup) does not get stuck here.
	return true
}

func (inv *BackupBatchInvoker) GetHash() string {
	return inv.backupBatch.GetSpecHash()
}

func (inv *BackupBatchInvoker) GetObjectJSON() (string, error) {
	jsonObj, err := meta.MarshalToJson(inv.backupBatch, v1beta1.SchemeGroupVersion)
	if err != nil {
		return "", err
	}
	return string(jsonObj), nil
}

func (inv *BackupBatchInvoker) GetRuntimeObject() runtime.Object {
	return inv.backupBatch
}

func (inv *BackupBatchInvoker) GetRetentionPolicy() v1alpha1.RetentionPolicy {
	return inv.backupBatch.Spec.RetentionPolicy
}

func (inv *BackupBatchInvoker) GetPhase() v1beta1.BackupInvokerPhase {
	return inv.backupBatch.Status.Phase
}

func (inv *BackupBatchInvoker) GetSummary(target v1beta1.TargetRef, session kmapi.ObjectReference) *v1beta1.Summary {
	summary := getTargetBackupSummary(inv.stashClient, target, session)
	summary.Invoker = core.TypedLocalObjectReference{
		APIGroup: pointer.StringP(v1beta1.SchemeGroupVersion.Group),
		Kind:     v1beta1.ResourceKindBackupBatch,
		Name:     inv.backupBatch.Name,
	}
	return summary
}

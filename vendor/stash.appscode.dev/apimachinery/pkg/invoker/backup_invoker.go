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

	"stash.appscode.dev/apimachinery/apis/stash/v1alpha1"
	"stash.appscode.dev/apimachinery/apis/stash/v1beta1"
	cs "stash.appscode.dev/apimachinery/client/clientset/versioned"
	stash_scheme "stash.appscode.dev/apimachinery/client/clientset/versioned/scheme"
	v1beta1_util "stash.appscode.dev/apimachinery/client/clientset/versioned/typed/stash/v1beta1/util"

	core "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/reference"
	kmapi "kmodules.xyz/client-go/api/v1"
	core_util "kmodules.xyz/client-go/core/v1"
	"kmodules.xyz/client-go/meta"
	ofst "kmodules.xyz/offshoot-api/api/v1"
)

type BackupTargetInfo struct {
	Task                  v1beta1.TaskRef
	Target                *v1beta1.BackupTarget
	RuntimeSettings       ofst.RuntimeSettings
	TempDir               v1beta1.EmptyDirSettings
	InterimVolumeTemplate *ofst.PersistentVolumeClaim
	Hooks                 *v1beta1.BackupHooks
}

type BackupInvoker struct {
	TypeMeta           metav1.TypeMeta
	ObjectMeta         metav1.ObjectMeta
	Labels             map[string]string
	Hash               string
	Driver             v1beta1.Snapshotter
	Schedule           string
	Paused             bool
	Repository         string
	RetentionPolicy    v1alpha1.RetentionPolicy
	RuntimeSettings    ofst.RuntimeSettings
	BackupHistoryLimit *int32
	TargetsInfo        []BackupTargetInfo
	ExecutionOrder     v1beta1.ExecutionOrder
	Hooks              *v1beta1.BackupHooks
	ObjectRef          *core.ObjectReference
	OwnerRef           *metav1.OwnerReference
	ObjectJson         []byte
	AddFinalizer       func() error
	RemoveFinalizer    func() error
	HasCondition       func(*v1beta1.TargetRef, string) (bool, error)
	GetCondition       func(*v1beta1.TargetRef, string) (int, *kmapi.Condition, error)
	SetCondition       func(*v1beta1.TargetRef, kmapi.Condition) error
	IsConditionTrue    func(*v1beta1.TargetRef, string) (bool, error)
	NextInOrder        func(v1beta1.TargetRef, []v1beta1.BackupTargetStatus) bool
}

func ExtractBackupInvokerInfo(stashClient cs.Interface, invokerType, invokerName, namespace string) (BackupInvoker, error) {
	var invoker BackupInvoker
	switch invokerType {
	case v1beta1.ResourceKindBackupBatch:
		// get BackupBatch
		backupBatch, err := stashClient.StashV1beta1().BackupBatches(namespace).Get(context.TODO(), invokerName, metav1.GetOptions{})
		if err != nil {
			return invoker, err
		}
		invoker.TypeMeta = metav1.TypeMeta{
			Kind:       v1beta1.ResourceKindBackupBatch,
			APIVersion: v1beta1.SchemeGroupVersion.String(),
		}
		invoker.ObjectMeta = backupBatch.ObjectMeta
		invoker.Labels = backupBatch.OffshootLabels()
		invoker.Hash = backupBatch.GetSpecHash()
		invoker.Driver = backupBatch.Spec.Driver
		invoker.Schedule = backupBatch.Spec.Schedule
		invoker.Paused = backupBatch.Spec.Paused
		invoker.Repository = backupBatch.Spec.Repository.Name
		invoker.RetentionPolicy = backupBatch.Spec.RetentionPolicy
		invoker.RuntimeSettings = backupBatch.Spec.RuntimeSettings
		invoker.BackupHistoryLimit = backupBatch.Spec.BackupHistoryLimit
		invoker.Hooks = backupBatch.Spec.Hooks
		invoker.ExecutionOrder = backupBatch.Spec.ExecutionOrder
		invoker.OwnerRef = metav1.NewControllerRef(backupBatch, v1beta1.SchemeGroupVersion.WithKind(v1beta1.ResourceKindBackupBatch))
		invoker.ObjectRef, err = reference.GetReference(stash_scheme.Scheme, backupBatch)
		if err != nil {
			return invoker, err
		}

		invoker.ObjectJson, err = meta.MarshalToJson(backupBatch, v1beta1.SchemeGroupVersion)
		if err != nil {
			return invoker, err
		}

		for _, member := range backupBatch.Spec.Members {
			invoker.TargetsInfo = append(invoker.TargetsInfo, BackupTargetInfo{
				Task:                  member.Task,
				Target:                member.Target,
				RuntimeSettings:       member.RuntimeSettings,
				TempDir:               member.TempDir,
				InterimVolumeTemplate: member.InterimVolumeTemplate,
				Hooks:                 member.Hooks,
			})
		}
		invoker.AddFinalizer = func() error {
			_, _, err := v1beta1_util.PatchBackupBatch(context.TODO(), stashClient.StashV1beta1(), backupBatch, func(in *v1beta1.BackupBatch) *v1beta1.BackupBatch {
				in.ObjectMeta = core_util.AddFinalizer(in.ObjectMeta, v1beta1.StashKey)
				return in
			}, metav1.PatchOptions{})
			return err
		}
		invoker.RemoveFinalizer = func() error {
			_, _, err := v1beta1_util.PatchBackupBatch(context.TODO(), stashClient.StashV1beta1(), backupBatch, func(in *v1beta1.BackupBatch) *v1beta1.BackupBatch {
				in.ObjectMeta = core_util.RemoveFinalizer(in.ObjectMeta, v1beta1.StashKey)
				return in
			}, metav1.PatchOptions{})
			return err
		}
		invoker.HasCondition = func(target *v1beta1.TargetRef, condType string) (bool, error) {
			backupBatch, err := stashClient.StashV1beta1().BackupBatches(namespace).Get(context.TODO(), invokerName, metav1.GetOptions{})
			if err != nil {
				return false, err
			}
			if target != nil {
				return hasMemberCondition(backupBatch.Status.MemberConditions, *target, condType), nil
			}
			return kmapi.HasCondition(backupBatch.Status.Conditions, condType), nil
		}
		invoker.GetCondition = func(target *v1beta1.TargetRef, condType string) (int, *kmapi.Condition, error) {
			backupBatch, err := stashClient.StashV1beta1().BackupBatches(namespace).Get(context.TODO(), invokerName, metav1.GetOptions{})
			if err != nil {
				return -1, nil, err
			}
			if target != nil {
				idx, cond := getMemberCondition(backupBatch.Status.MemberConditions, *target, condType)
				return idx, cond, nil
			}
			idx, cond := kmapi.GetCondition(backupBatch.Status.Conditions, condType)
			return idx, cond, nil

		}
		invoker.SetCondition = func(target *v1beta1.TargetRef, condition kmapi.Condition) error {
			_, err = v1beta1_util.UpdateBackupBatchStatus(context.TODO(), stashClient.StashV1beta1(), backupBatch.ObjectMeta, func(in *v1beta1.BackupBatchStatus) (types.UID, *v1beta1.BackupBatchStatus) {
				if target != nil {
					in.MemberConditions = setMemberCondition(in.MemberConditions, *target, condition)
				} else {
					in.Conditions = kmapi.SetCondition(in.Conditions, condition)
				}
				return backupBatch.UID, in
			}, metav1.UpdateOptions{})
			return err
		}
		invoker.IsConditionTrue = func(target *v1beta1.TargetRef, condType string) (bool, error) {
			backupBatch, err := stashClient.StashV1beta1().BackupBatches(namespace).Get(context.TODO(), invokerName, metav1.GetOptions{})
			if err != nil {
				return false, err
			}
			if target != nil {
				return isMemberConditionTrue(backupBatch.Status.MemberConditions, *target, condType), nil
			}
			return kmapi.IsConditionTrue(backupBatch.Status.Conditions, condType), nil
		}
		invoker.NextInOrder = func(ref v1beta1.TargetRef, targetStatus []v1beta1.BackupTargetStatus) bool {
			for _, t := range invoker.TargetsInfo {
				if t.Target != nil {
					if TargetMatched(t.Target.Ref, ref) {
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
	case v1beta1.ResourceKindBackupConfiguration:
		// get BackupConfiguration
		backupConfig, err := stashClient.StashV1beta1().BackupConfigurations(namespace).Get(context.TODO(), invokerName, metav1.GetOptions{})
		if err != nil {
			return invoker, err
		}
		invoker.TypeMeta = metav1.TypeMeta{
			Kind:       v1beta1.ResourceKindBackupConfiguration,
			APIVersion: v1beta1.SchemeGroupVersion.String(),
		}
		invoker.ObjectMeta = backupConfig.ObjectMeta
		invoker.Labels = backupConfig.OffshootLabels()
		invoker.Hash = backupConfig.GetSpecHash()
		invoker.Driver = backupConfig.Spec.Driver
		invoker.Schedule = backupConfig.Spec.Schedule
		invoker.Paused = backupConfig.Spec.Paused
		invoker.Repository = backupConfig.Spec.Repository.Name
		invoker.RetentionPolicy = backupConfig.Spec.RetentionPolicy
		invoker.RuntimeSettings = backupConfig.Spec.RuntimeSettings
		invoker.BackupHistoryLimit = backupConfig.Spec.BackupHistoryLimit
		invoker.OwnerRef = metav1.NewControllerRef(backupConfig, v1beta1.SchemeGroupVersion.WithKind(v1beta1.ResourceKindBackupConfiguration))
		invoker.ObjectRef, err = reference.GetReference(stash_scheme.Scheme, backupConfig)
		if err != nil {
			return invoker, err
		}

		invoker.ObjectJson, err = meta.MarshalToJson(backupConfig, v1beta1.SchemeGroupVersion)
		if err != nil {
			return invoker, err
		}

		invoker.TargetsInfo = append(invoker.TargetsInfo, BackupTargetInfo{
			Task:                  backupConfig.Spec.Task,
			Target:                backupConfig.Spec.Target,
			RuntimeSettings:       backupConfig.Spec.RuntimeSettings,
			TempDir:               backupConfig.Spec.TempDir,
			InterimVolumeTemplate: backupConfig.Spec.InterimVolumeTemplate,
			Hooks:                 backupConfig.Spec.Hooks,
		})
		invoker.AddFinalizer = func() error {
			_, _, err := v1beta1_util.PatchBackupConfiguration(context.TODO(), stashClient.StashV1beta1(), backupConfig, func(in *v1beta1.BackupConfiguration) *v1beta1.BackupConfiguration {
				in.ObjectMeta = core_util.AddFinalizer(in.ObjectMeta, v1beta1.StashKey)
				return in
			}, metav1.PatchOptions{})
			return err
		}
		invoker.RemoveFinalizer = func() error {
			_, _, err := v1beta1_util.PatchBackupConfiguration(context.TODO(), stashClient.StashV1beta1(), backupConfig, func(in *v1beta1.BackupConfiguration) *v1beta1.BackupConfiguration {
				in.ObjectMeta = core_util.RemoveFinalizer(in.ObjectMeta, v1beta1.StashKey)
				return in
			}, metav1.PatchOptions{})
			return err
		}
		invoker.HasCondition = func(target *v1beta1.TargetRef, condType string) (bool, error) {
			backupConfig, err := stashClient.StashV1beta1().BackupConfigurations(namespace).Get(context.TODO(), invokerName, metav1.GetOptions{})
			if err != nil {
				return false, err
			}
			return kmapi.HasCondition(backupConfig.Status.Conditions, condType), nil
		}
		invoker.GetCondition = func(target *v1beta1.TargetRef, condType string) (int, *kmapi.Condition, error) {
			backupConfig, err := stashClient.StashV1beta1().BackupConfigurations(namespace).Get(context.TODO(), invokerName, metav1.GetOptions{})
			if err != nil {
				return -1, nil, err
			}
			idx, cond := kmapi.GetCondition(backupConfig.Status.Conditions, condType)
			return idx, cond, nil
		}
		invoker.SetCondition = func(target *v1beta1.TargetRef, condition kmapi.Condition) error {
			_, err = v1beta1_util.UpdateBackupConfigurationStatus(context.TODO(), stashClient.StashV1beta1(), backupConfig.ObjectMeta, func(in *v1beta1.BackupConfigurationStatus) (types.UID, *v1beta1.BackupConfigurationStatus) {
				in.Conditions = kmapi.SetCondition(in.Conditions, condition)
				return backupConfig.UID, in
			}, metav1.UpdateOptions{})
			return err
		}
		invoker.IsConditionTrue = func(target *v1beta1.TargetRef, condType string) (bool, error) {
			backupConfig, err := stashClient.StashV1beta1().BackupConfigurations(namespace).Get(context.TODO(), invokerName, metav1.GetOptions{})
			if err != nil {
				return false, err
			}
			return kmapi.IsConditionTrue(backupConfig.Status.Conditions, condType), nil
		}
		invoker.NextInOrder = func(ref v1beta1.TargetRef, targetStatus []v1beta1.BackupTargetStatus) bool {
			for _, t := range invoker.TargetsInfo {
				if t.Target != nil {
					if TargetMatched(t.Target.Ref, ref) {
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
	default:
		return invoker, fmt.Errorf("failed to extract invoker info. Reason: unknown invoker")
	}
	return invoker, nil
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

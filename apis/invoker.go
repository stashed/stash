/*
Copyright The Stash Authors.

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

package apis

import (
	"fmt"

	"stash.appscode.dev/stash/apis/stash/v1alpha1"
	"stash.appscode.dev/stash/apis/stash/v1beta1"
	cs "stash.appscode.dev/stash/client/clientset/versioned"
	stash_scheme "stash.appscode.dev/stash/client/clientset/versioned/scheme"
	v1beta1_util "stash.appscode.dev/stash/client/clientset/versioned/typed/stash/v1beta1/util"

	core "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/reference"
	core_util "kmodules.xyz/client-go/core/v1"
	"kmodules.xyz/client-go/meta"
	ofst "kmodules.xyz/offshoot-api/api/v1"
)

type TargetInfo struct {
	Task                  v1beta1.TaskRef
	Target                *v1beta1.BackupTarget
	RuntimeSettings       ofst.RuntimeSettings
	TempDir               v1beta1.EmptyDirSettings
	InterimVolumeTemplate *ofst.PersistentVolumeClaim
	Hooks                 *v1beta1.BackupHooks
}

type Invoker struct {
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
	TargetsInfo        []TargetInfo
	Hooks              *v1beta1.BackupHooks
	ObjectRef          *core.ObjectReference
	OwnerRef           *metav1.OwnerReference
	ObjectJson         []byte
	AddFinalizer       func() error
	RemoveFinalizer    func() error
}

func ExtractBackupInvokerInfo(stashClient cs.Interface, invokerType, invokerName, namespace string) (Invoker, error) {
	var invoker Invoker
	switch invokerType {
	case v1beta1.ResourceKindBackupBatch:
		// get BackupBatch
		backupBatch, err := stashClient.StashV1beta1().BackupBatches(namespace).Get(invokerName, metav1.GetOptions{})
		if err != nil {
			return invoker, err
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
			invoker.TargetsInfo = append(invoker.TargetsInfo, TargetInfo{
				Task:                  member.Task,
				Target:                member.Target,
				RuntimeSettings:       member.RuntimeSettings,
				TempDir:               member.TempDir,
				InterimVolumeTemplate: member.InterimVolumeTemplate,
				Hooks:                 member.Hooks,
			})
		}
		invoker.AddFinalizer = func() error {
			_, _, err := v1beta1_util.PatchBackupBatch(stashClient.StashV1beta1(), backupBatch, func(in *v1beta1.BackupBatch) *v1beta1.BackupBatch {
				in.ObjectMeta = core_util.AddFinalizer(in.ObjectMeta, v1beta1.StashKey)
				return in

			})
			return err
		}
		invoker.RemoveFinalizer = func() error {
			_, _, err := v1beta1_util.PatchBackupBatch(stashClient.StashV1beta1(), backupBatch, func(in *v1beta1.BackupBatch) *v1beta1.BackupBatch {
				in.ObjectMeta = core_util.RemoveFinalizer(in.ObjectMeta, v1beta1.StashKey)
				return in

			})
			return err
		}
	case v1beta1.ResourceKindBackupConfiguration:
		// get BackupConfiguration
		backupConfig, err := stashClient.StashV1beta1().BackupConfigurations(namespace).Get(invokerName, metav1.GetOptions{})
		if err != nil {
			return invoker, err
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

		invoker.TargetsInfo = append(invoker.TargetsInfo, TargetInfo{
			Task:                  backupConfig.Spec.Task,
			Target:                backupConfig.Spec.Target,
			RuntimeSettings:       backupConfig.Spec.RuntimeSettings,
			TempDir:               backupConfig.Spec.TempDir,
			InterimVolumeTemplate: backupConfig.Spec.InterimVolumeTemplate,
			Hooks:                 backupConfig.Spec.Hooks,
		})
		invoker.AddFinalizer = func() error {
			_, _, err := v1beta1_util.PatchBackupConfiguration(stashClient.StashV1beta1(), backupConfig, func(in *v1beta1.BackupConfiguration) *v1beta1.BackupConfiguration {
				in.ObjectMeta = core_util.AddFinalizer(in.ObjectMeta, v1beta1.StashKey)
				return in

			})
			return err
		}
		invoker.RemoveFinalizer = func() error {
			_, _, err := v1beta1_util.PatchBackupConfiguration(stashClient.StashV1beta1(), backupConfig, func(in *v1beta1.BackupConfiguration) *v1beta1.BackupConfiguration {
				in.ObjectMeta = core_util.RemoveFinalizer(in.ObjectMeta, v1beta1.StashKey)
				return in

			})
			return err
		}

	default:
		return invoker, fmt.Errorf("failed to extract invoker info. Reason: unknown invoker")
	}
	return invoker, nil
}

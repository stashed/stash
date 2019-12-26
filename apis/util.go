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

	core "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/reference"
	v1 "kmodules.xyz/offshoot-api/api/v1"
)

const (
	StashKey   = "stash.appscode.com"
	VersionTag = StashKey + "/tag"

	KeyDeleteJobOnCompletion     = StashKey + "/delete-job-on-completion"
	AllowDeletingJobOnCompletion = "true"
)

const (
	KindDeployment            = "Deployment"
	KindReplicaSet            = "ReplicaSet"
	KindReplicationController = "ReplicationController"
	KindStatefulSet           = "StatefulSet"
	KindDaemonSet             = "DaemonSet"
	KindPod                   = "Pod"
	KindPersistentVolumeClaim = "PersistentVolumeClaim"
	KindAppBinding            = "AppBinding"
	KindDeploymentConfig      = "DeploymentConfig"
	KindSecret                = "Secret"
	KindService               = "Service"
	KindJob                   = "Job"
	KindCronJob               = "CronJob"
)

const (
	ResourcePluralDeployment            = "deployments"
	ResourcePluralReplicaSet            = "replicasets"
	ResourcePluralReplicationController = "replicationcontrollers"
	ResourcePluralStatefulSet           = "statefulsets"
	ResourcePluralDaemonSet             = "daemonsets"
	ResourcePluralPod                   = "pods"
	ResourcePluralPersistentVolumeClaim = "persistentvolumeclaims"
	ResourcePluralAppBinding            = "appbindings"
	ResourcePluralDeploymentConfig      = "deploymentconfigs"
	ResourcePluralSecret                = "secrets"
	ResourcePluralService               = "services"
)

type TargetInfo struct {
	Target                *v1beta1.BackupTarget
	RuntimeSettings       v1.RuntimeSettings
	InterimVolumeTemplate *core.PersistentVolumeClaim
	TempDir               v1beta1.EmptyDirSettings
	Task                  v1beta1.TaskRef
}

type InvokerInfo struct {
	ObjMeta            metav1.ObjectMeta
	InvokerKind        string
	RepoName           string
	RetentionPolicy    v1alpha1.RetentionPolicy
	OffShootLabels     map[string]string
	InvokerHash        string
	Hooks              *v1beta1.Hooks
	Schedule           string
	Driver             v1beta1.Snapshotter
	Paused             bool
	BackupHistoryLimit *int32
	InvokerRef         *core.ObjectReference
	TargetsInfo        []TargetInfo
}

func BackupInfoForInvoker(invokerType, invokerName, namespace string, stashClient cs.Interface) (InvokerInfo, error) {
	var invokerInfo InvokerInfo
	if invokerType == v1beta1.ResourceKindBackupBatch {
		// get BackupBatch
		backupBatch, err := stashClient.StashV1beta1().BackupBatches(namespace).Get(invokerName, metav1.GetOptions{})
		if err != nil {
			return invokerInfo, err
		}

		invokerInfo.ObjMeta = backupBatch.ObjectMeta
		invokerInfo.InvokerKind = v1beta1.ResourceKindBackupBatch
		invokerInfo.Paused = backupBatch.Spec.Paused
		invokerInfo.RetentionPolicy = backupBatch.Spec.RetentionPolicy
		invokerInfo.RepoName = backupBatch.Spec.Repository.Name
		invokerInfo.Driver = backupBatch.Spec.Driver
		invokerInfo.InvokerHash = backupBatch.GetSpecHash()
		invokerInfo.OffShootLabels = backupBatch.OffshootLabels()
		invokerInfo.Hooks = backupBatch.Spec.Hooks
		invokerInfo.Schedule = backupBatch.Spec.Schedule

		// get BackupBatch object reference to use writing event
		invokerInfo.InvokerRef, err = reference.GetReference(stash_scheme.Scheme, backupBatch)
		if err != nil {
			return invokerInfo, err
		}

		for i, backupConfigTemp := range backupBatch.Spec.BackupConfigurationTemplates {
			if backupConfigTemp.Spec.Target != nil {
				invokerInfo.TargetsInfo = append(invokerInfo.TargetsInfo, TargetInfo{
					Task:            backupConfigTemp.Spec.Task,
					RuntimeSettings: backupConfigTemp.Spec.RuntimeSettings,
					TempDir:         backupConfigTemp.Spec.TempDir,
					Target:          backupConfigTemp.Spec.Target,
				})
				if backupConfigTemp.Spec.InterimVolumeTemplate != nil {
					invokerInfo.TargetsInfo[i].InterimVolumeTemplate = backupConfigTemp.Spec.InterimVolumeTemplate
				}
			}
			if backupConfigTemp.Spec.Target == nil {
				return invokerInfo, fmt.Errorf("in backupBatch, backupConfigurtionTemplate target is nil")
			}
		}
		return invokerInfo, nil
	}
	// get BackupConfiguration
	backupConfig, err := stashClient.StashV1beta1().BackupConfigurations(namespace).Get(invokerName, metav1.GetOptions{})
	if err != nil {
		return invokerInfo, err
	}

	invokerInfo.ObjMeta = backupConfig.ObjectMeta
	invokerInfo.InvokerKind = v1beta1.ResourceKindBackupConfiguration
	invokerInfo.Paused = backupConfig.Spec.Paused
	invokerInfo.RetentionPolicy = backupConfig.Spec.RetentionPolicy
	invokerInfo.RepoName = backupConfig.Spec.Repository.Name
	invokerInfo.Driver = backupConfig.Spec.Driver
	invokerInfo.InvokerHash = backupConfig.GetSpecHash()
	invokerInfo.OffShootLabels = backupConfig.OffshootLabels()
	invokerInfo.Hooks = backupConfig.Spec.Hooks
	invokerInfo.Schedule = backupConfig.Spec.Schedule
	// get BackupConfiguration object reference to use writing event
	invokerInfo.InvokerRef, err = reference.GetReference(stash_scheme.Scheme, backupConfig)
	if err != nil {
		return invokerInfo, err
	}

	invokerInfo.TargetsInfo = append(invokerInfo.TargetsInfo, TargetInfo{
		Task:            backupConfig.Spec.Task,
		RuntimeSettings: backupConfig.Spec.RuntimeSettings,
		TempDir:         backupConfig.Spec.TempDir,
	})
	if backupConfig.Spec.InterimVolumeTemplate != nil {
		invokerInfo.TargetsInfo[0].InterimVolumeTemplate = backupConfig.Spec.InterimVolumeTemplate
	}
	if backupConfig.Spec.Target != nil {
		invokerInfo.TargetsInfo[0].Target = backupConfig.Spec.Target
	}
	if backupConfig.Spec.Target == nil {
		return invokerInfo, fmt.Errorf("backupConfiguration target is nil")
	}

	return invokerInfo, nil
}

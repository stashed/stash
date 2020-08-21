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

package conditions

import (
	"context"
	"fmt"

	"stash.appscode.dev/apimachinery/apis"
	api_v1beta1 "stash.appscode.dev/apimachinery/apis/stash/v1beta1"
	cs "stash.appscode.dev/apimachinery/client/clientset/versioned"
	stash_util "stash.appscode.dev/apimachinery/client/clientset/versioned/typed/stash/v1beta1/util"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	kmapi "kmodules.xyz/client-go/api/v1"
)

func SetGlobalPreBackupHookSucceededConditionToFalse(stashClient cs.Interface, backupSession *api_v1beta1.BackupSession, hookErr error) (*api_v1beta1.BackupSession, error) {
	return stash_util.UpdateBackupSessionStatus(
		context.TODO(),
		stashClient.StashV1beta1(),
		backupSession.ObjectMeta,
		func(in *api_v1beta1.BackupSessionStatus) *api_v1beta1.BackupSessionStatus {
			in.Conditions = kmapi.SetCondition(in.Conditions, kmapi.Condition{
				Type:    apis.GlobalPreBackupHookSucceeded,
				Status:  kmapi.ConditionFalse,
				Reason:  apis.GlobalPreBackupHookExecutionFailed,
				Message: fmt.Sprintf("Failed to execute global PreBackup Hook. Reason: %v.", hookErr),
			},
			)
			return in
		},
		metav1.UpdateOptions{},
	)
}

func SetGlobalPreBackupHookSucceededConditionToTrue(stashClient cs.Interface, backupSession *api_v1beta1.BackupSession) (*api_v1beta1.BackupSession, error) {
	return stash_util.UpdateBackupSessionStatus(
		context.TODO(),
		stashClient.StashV1beta1(),
		backupSession.ObjectMeta,
		func(in *api_v1beta1.BackupSessionStatus) *api_v1beta1.BackupSessionStatus {
			in.Conditions = kmapi.SetCondition(in.Conditions, kmapi.Condition{
				Type:    apis.GlobalPreBackupHookSucceeded,
				Status:  kmapi.ConditionTrue,
				Reason:  apis.GlobalPreBackupHookExecutedSuccessfully,
				Message: "Global PreBackup hook has been executed successfully",
			},
			)
			return in
		},
		metav1.UpdateOptions{},
	)
}

func SetGlobalPostBackupHookSucceededConditionToFalse(stashClient cs.Interface, backupSession *api_v1beta1.BackupSession, hookErr error) (*api_v1beta1.BackupSession, error) {
	return stash_util.UpdateBackupSessionStatus(
		context.TODO(),
		stashClient.StashV1beta1(),
		backupSession.ObjectMeta,
		func(in *api_v1beta1.BackupSessionStatus) *api_v1beta1.BackupSessionStatus {
			in.Conditions = kmapi.SetCondition(in.Conditions, kmapi.Condition{
				Type:    apis.GlobalPostBackupHookSucceeded,
				Status:  kmapi.ConditionFalse,
				Reason:  apis.GlobalPostBackupHookExecutionFailed,
				Message: fmt.Sprintf("Failed to execute global PostBackup Hook. Reason: %v.", hookErr),
			},
			)
			return in
		},
		metav1.UpdateOptions{},
	)
}

func SetGlobalPostBackupHookSucceededConditionToTrue(stashClient cs.Interface, backupSession *api_v1beta1.BackupSession) (*api_v1beta1.BackupSession, error) {
	return stash_util.UpdateBackupSessionStatus(
		context.TODO(),
		stashClient.StashV1beta1(),
		backupSession.ObjectMeta,
		func(in *api_v1beta1.BackupSessionStatus) *api_v1beta1.BackupSessionStatus {
			in.Conditions = kmapi.SetCondition(in.Conditions, kmapi.Condition{
				Type:    apis.GlobalPostBackupHookSucceeded,
				Status:  kmapi.ConditionTrue,
				Reason:  apis.GlobalPostBackupHookExecutedSuccessfully,
				Message: "Global PostBackup hook has been executed successfully",
			},
			)
			return in
		},
		metav1.UpdateOptions{},
	)
}

func SetGlobalPreRestoreHookSucceededConditionToFalse(invoker apis.RestoreInvoker, hookErr error) error {
	return invoker.SetCondition(nil, kmapi.Condition{
		Type:    apis.GlobalPreRestoreHookSucceeded,
		Status:  kmapi.ConditionFalse,
		Reason:  apis.GlobalPreRestoreHookExecutionFailed,
		Message: fmt.Sprintf("Failed to execute global PreRestore Hook. Reason: %v.", hookErr),
	})
}

func SetGlobalPreRestoreHookSucceededConditionToTrue(invoker apis.RestoreInvoker) error {
	return invoker.SetCondition(nil, kmapi.Condition{
		Type:    apis.GlobalPreRestoreHookSucceeded,
		Status:  kmapi.ConditionTrue,
		Reason:  apis.GlobalPreRestoreHookExecutedSuccessfully,
		Message: "Global PreRestore hook has been executed successfully",
	})
}

func SetGlobalPostRestoreHookSucceededConditionToFalse(invoker apis.RestoreInvoker, hookErr error) error {
	return invoker.SetCondition(nil, kmapi.Condition{
		Type:    apis.GlobalPostRestoreHookSucceeded,
		Status:  kmapi.ConditionFalse,
		Reason:  apis.GlobalPostRestoreHookExecutionFailed,
		Message: fmt.Sprintf("Failed to execute global PostRestore Hook. Reason: %v.", hookErr),
	})
}

func SetGlobalPostRestoreHookSucceededConditionToTrue(invoker apis.RestoreInvoker) error {
	return invoker.SetCondition(nil, kmapi.Condition{
		Type:    apis.GlobalPostRestoreHookSucceeded,
		Status:  kmapi.ConditionTrue,
		Reason:  apis.GlobalPostRestoreHookExecutedSuccessfully,
		Message: "Global PostRestore hook has been executed successfully",
	})
}

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
	"stash.appscode.dev/apimachinery/apis/stash/v1beta1"
	api_v1beta1 "stash.appscode.dev/apimachinery/apis/stash/v1beta1"
	cs "stash.appscode.dev/apimachinery/client/clientset/versioned"
	stash_util "stash.appscode.dev/apimachinery/client/clientset/versioned/typed/stash/v1beta1/util"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	kmapi "kmodules.xyz/client-go/api/v1"
)

func SetBackendRepositoryInitializedConditionToFalse(stashClient cs.Interface, backupSession *v1beta1.BackupSession, err error) (*v1beta1.BackupSession, error) {
	return stash_util.UpdateBackupSessionStatus(
		context.TODO(),
		stashClient.StashV1beta1(),
		backupSession.ObjectMeta,
		func(in *api_v1beta1.BackupSessionStatus) *api_v1beta1.BackupSessionStatus {
			in.Conditions = kmapi.SetCondition(in.Conditions, kmapi.Condition{
				Type:    apis.BackendRepositoryInitialized,
				Status:  kmapi.ConditionFalse,
				Reason:  apis.FailedToInitializeBackendRepository,
				Message: fmt.Sprintf("Failed to initialize backend repository. Reason: %v", err.Error()),
			},
			)
			return in
		},
		metav1.UpdateOptions{},
	)
}

func SetBackendRepositoryInitializedConditionToTrue(stashClient cs.Interface, backupSession *v1beta1.BackupSession) (*v1beta1.BackupSession, error) {
	return stash_util.UpdateBackupSessionStatus(
		context.TODO(),
		stashClient.StashV1beta1(),
		backupSession.ObjectMeta,
		func(in *api_v1beta1.BackupSessionStatus) *api_v1beta1.BackupSessionStatus {
			in.Conditions = kmapi.SetCondition(in.Conditions, kmapi.Condition{
				Type:    apis.BackendRepositoryInitialized,
				Status:  kmapi.ConditionTrue,
				Reason:  apis.BackendRepositoryFound,
				Message: "Repository exist in the backend.",
			},
			)
			return in
		},
		metav1.UpdateOptions{},
	)
}

func SetRepositoryFoundConditionToUnknown(invoker interface{}, err error) error {
	switch in := invoker.(type) {
	case apis.Invoker:
		return in.SetCondition(nil, kmapi.Condition{
			Type:   apis.RepositoryFound,
			Status: kmapi.ConditionUnknown,
			Reason: apis.UnableToCheckRepositoryAvailability,
			Message: fmt.Sprintf("Failed to check whether the Repository %s/%s exist or not. Reason: %v",
				in.ObjectMeta.Namespace,
				in.Repository,
				err.Error(),
			),
		})
	case apis.RestoreInvoker:
		return in.SetCondition(nil, kmapi.Condition{
			Type:   apis.RepositoryFound,
			Status: kmapi.ConditionUnknown,
			Reason: apis.UnableToCheckRepositoryAvailability,
			Message: fmt.Sprintf("Failed to check whether the Repository %s/%s exist or not. Reason: %v",
				in.ObjectMeta.Namespace,
				in.Repository,
				err.Error(),
			),
		})
	default:
		return fmt.Errorf("unable to set %s condition. Reason: invoker type unknown", apis.RepositoryFound)
	}
}

func SetRepositoryFoundConditionToFalse(invoker interface{}) error {
	switch in := invoker.(type) {
	case apis.Invoker:
		return in.SetCondition(nil, kmapi.Condition{
			Type:   apis.RepositoryFound,
			Status: kmapi.ConditionFalse,
			Reason: apis.RepositoryNotAvailable,
			Message: fmt.Sprintf("Repository %s/%s does not exist.",
				in.ObjectMeta.Namespace,
				in.Repository,
			),
		})
	case apis.RestoreInvoker:
		return in.SetCondition(nil, kmapi.Condition{
			Type:   apis.RepositoryFound,
			Status: kmapi.ConditionFalse,
			Reason: apis.RepositoryNotAvailable,
			Message: fmt.Sprintf("Repository %s/%s does not exist.",
				in.ObjectMeta.Namespace,
				in.Repository,
			),
		})
	default:
		return fmt.Errorf("unable to set %s condition. Reason: invoker type unknown", apis.RepositoryFound)
	}
}

func SetRepositoryFoundConditionToTrue(invoker interface{}) error {
	switch in := invoker.(type) {
	case apis.Invoker:
		return in.SetCondition(nil, kmapi.Condition{
			Type:   apis.RepositoryFound,
			Status: kmapi.ConditionTrue,
			Reason: apis.RepositoryAvailable,
			Message: fmt.Sprintf("Repository %s/%s exist.",
				in.ObjectMeta.Namespace,
				in.Repository,
			),
		})
	case apis.RestoreInvoker:
		return in.SetCondition(nil, kmapi.Condition{
			Type:   apis.RepositoryFound,
			Status: kmapi.ConditionTrue,
			Reason: apis.RepositoryAvailable,
			Message: fmt.Sprintf("Repository %s/%s exist.",
				in.ObjectMeta.Namespace,
				in.Repository,
			),
		})
	default:
		return fmt.Errorf("unable to set %s condition. Reason: invoker type unknown", apis.RepositoryFound)
	}
}

func SetBackendSecretFoundConditionToUnknown(invoker interface{}, secretName string, err error) error {
	switch in := invoker.(type) {
	case apis.Invoker:
		return in.SetCondition(nil, kmapi.Condition{
			Type:   apis.BackendSecretFound,
			Status: kmapi.ConditionUnknown,
			Reason: apis.UnableToCheckBackendSecretAvailability,
			Message: fmt.Sprintf("Failed to check whether the backend Secret %s/%s exist or not. Reason: %v",
				in.ObjectMeta.Namespace,
				secretName,
				err.Error(),
			),
		})
	case apis.RestoreInvoker:
		return in.SetCondition(nil, kmapi.Condition{
			Type:   apis.BackendSecretFound,
			Status: kmapi.ConditionUnknown,
			Reason: apis.UnableToCheckBackendSecretAvailability,
			Message: fmt.Sprintf("Failed to check whether the backend Secret %s/%s exist or not. Reason: %v",
				in.ObjectMeta.Namespace,
				secretName,
				err.Error(),
			),
		})
	default:
		return fmt.Errorf("unable to set %s condition. Reason: invoker type unknown", apis.BackendSecretFound)
	}
}

func SetBackendSecretFoundConditionToFalse(invoker interface{}, secretName string) error {
	switch in := invoker.(type) {
	case apis.Invoker:
		return in.SetCondition(nil, kmapi.Condition{
			Type:   apis.BackendSecretFound,
			Status: kmapi.ConditionFalse,
			Reason: apis.BackendSecretNotAvailable,
			Message: fmt.Sprintf("Backend Secret %s/%s does not exist.",
				in.ObjectMeta.Namespace,
				secretName,
			),
		})
	case apis.RestoreInvoker:
		return in.SetCondition(nil, kmapi.Condition{
			Type:   apis.BackendSecretFound,
			Status: kmapi.ConditionFalse,
			Reason: apis.BackendSecretNotAvailable,
			Message: fmt.Sprintf("Backend Secret %s/%s does not exist.",
				in.ObjectMeta.Namespace,
				secretName,
			),
		})
	default:
		return fmt.Errorf("unable to set %s condition. Reason: invoker type unknown", apis.BackendSecretFound)
	}
}

func SetBackendSecretFoundConditionToTrue(invoker interface{}, secretName string) error {
	switch in := invoker.(type) {
	case apis.Invoker:
		return in.SetCondition(nil, kmapi.Condition{
			Type:   apis.BackendSecretFound,
			Status: kmapi.ConditionTrue,
			Reason: apis.BackendSecretAvailable,
			Message: fmt.Sprintf("Backend Secret %s/%s exist.",
				in.ObjectMeta.Namespace,
				secretName,
			),
		})
	case apis.RestoreInvoker:
		return in.SetCondition(nil, kmapi.Condition{
			Type:   apis.BackendSecretFound,
			Status: kmapi.ConditionTrue,
			Reason: apis.BackendSecretAvailable,
			Message: fmt.Sprintf("Backend Secret %s/%s exist.",
				in.ObjectMeta.Namespace,
				secretName,
			),
		})
	default:
		return fmt.Errorf("unable to set %s condition. Reason: invoker type unknown", apis.BackendSecretFound)
	}
}

func SetRetentionPolicyAppliedConditionToFalse(stashClient cs.Interface, backupSession *v1beta1.BackupSession, err error) (*v1beta1.BackupSession, error) {
	return stash_util.UpdateBackupSessionStatus(
		context.TODO(),
		stashClient.StashV1beta1(),
		backupSession.ObjectMeta,
		func(in *api_v1beta1.BackupSessionStatus) *api_v1beta1.BackupSessionStatus {
			in.Conditions = kmapi.SetCondition(in.Conditions, kmapi.Condition{
				Type:    apis.RetentionPolicyApplied,
				Status:  kmapi.ConditionFalse,
				Reason:  apis.FailedToApplyRetentionPolicy,
				Message: fmt.Sprintf("Failed to apply retention policy. Reason: %v", err.Error()),
			},
			)
			return in
		},
		metav1.UpdateOptions{},
	)
}

func SetRetentionPolicyAppliedConditionToTrue(stashClient cs.Interface, backupSession *v1beta1.BackupSession) (*v1beta1.BackupSession, error) {
	return stash_util.UpdateBackupSessionStatus(
		context.TODO(),
		stashClient.StashV1beta1(),
		backupSession.ObjectMeta,
		func(in *api_v1beta1.BackupSessionStatus) *api_v1beta1.BackupSessionStatus {
			in.Conditions = kmapi.SetCondition(in.Conditions, kmapi.Condition{
				Type:    apis.RetentionPolicyApplied,
				Status:  kmapi.ConditionTrue,
				Reason:  apis.SuccessfullyAppliedRetentionPolicy,
				Message: "Successfully applied retention policy.",
			},
			)
			return in
		},
		metav1.UpdateOptions{},
	)
}

func SetRepositoryIntegrityVerifiedConditionToFalse(stashClient cs.Interface, backupSession *v1beta1.BackupSession, err error) (*v1beta1.BackupSession, error) {
	return stash_util.UpdateBackupSessionStatus(
		context.TODO(),
		stashClient.StashV1beta1(),
		backupSession.ObjectMeta,
		func(in *api_v1beta1.BackupSessionStatus) *api_v1beta1.BackupSessionStatus {
			in.Conditions = kmapi.SetCondition(in.Conditions, kmapi.Condition{
				Type:    apis.RepositoryIntegrityVerified,
				Status:  kmapi.ConditionFalse,
				Reason:  apis.FailedToVerifyRepositoryIntegrity,
				Message: fmt.Sprintf("Repository integrity verification failed. Reason: %v", err.Error()),
			},
			)
			return in
		},
		metav1.UpdateOptions{},
	)
}

func SetRepositoryIntegrityVerifiedConditionToTrue(stashClient cs.Interface, backupSession *v1beta1.BackupSession) (*v1beta1.BackupSession, error) {
	return stash_util.UpdateBackupSessionStatus(
		context.TODO(),
		stashClient.StashV1beta1(),
		backupSession.ObjectMeta,
		func(in *api_v1beta1.BackupSessionStatus) *api_v1beta1.BackupSessionStatus {
			in.Conditions = kmapi.SetCondition(in.Conditions, kmapi.Condition{
				Type:    apis.RepositoryIntegrityVerified,
				Status:  kmapi.ConditionTrue,
				Reason:  apis.SuccessfullyVerifiedRepositoryIntegrity,
				Message: "Repository integrity verification succeeded.",
			},
			)
			return in
		},
		metav1.UpdateOptions{},
	)
}

func SetRepositoryMetricsPushedConditionToFalse(stashClient cs.Interface, backupSession *v1beta1.BackupSession, err error) (*v1beta1.BackupSession, error) {
	return stash_util.UpdateBackupSessionStatus(
		context.TODO(),
		stashClient.StashV1beta1(),
		backupSession.ObjectMeta,
		func(in *api_v1beta1.BackupSessionStatus) *api_v1beta1.BackupSessionStatus {
			in.Conditions = kmapi.SetCondition(in.Conditions, kmapi.Condition{
				Type:    apis.RepositoryMetricsPushed,
				Status:  kmapi.ConditionFalse,
				Reason:  apis.FailedToPushRepositoryMetrics,
				Message: fmt.Sprintf("Failed to push repository metrics. Reason: %v", err.Error()),
			},
			)
			return in
		},
		metav1.UpdateOptions{},
	)
}

func SetRepositoryMetricsPushedConditionToTrue(stashClient cs.Interface, backupSession *v1beta1.BackupSession) (*v1beta1.BackupSession, error) {
	return stash_util.UpdateBackupSessionStatus(
		context.TODO(),
		stashClient.StashV1beta1(),
		backupSession.ObjectMeta,
		func(in *api_v1beta1.BackupSessionStatus) *api_v1beta1.BackupSessionStatus {
			in.Conditions = kmapi.SetCondition(in.Conditions, kmapi.Condition{
				Type:    apis.RepositoryMetricsPushed,
				Status:  kmapi.ConditionTrue,
				Reason:  apis.SuccessfullyPushedRepositoryMetrics,
				Message: "Successfully pushed repository metrics.",
			},
			)
			return in
		},
		metav1.UpdateOptions{},
	)
}

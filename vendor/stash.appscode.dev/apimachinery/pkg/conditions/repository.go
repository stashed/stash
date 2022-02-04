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
	"stash.appscode.dev/apimachinery/pkg/invoker"

	core "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	kmapi "kmodules.xyz/client-go/api/v1"
)

func SetBackendRepositoryInitializedConditionToFalse(stashClient cs.Interface, backupSession *v1beta1.BackupSession, err error) (*v1beta1.BackupSession, error) {
	return stash_util.UpdateBackupSessionStatus(
		context.TODO(),
		stashClient.StashV1beta1(),
		backupSession.ObjectMeta,
		func(in *api_v1beta1.BackupSessionStatus) (types.UID, *api_v1beta1.BackupSessionStatus) {
			in.Conditions = kmapi.SetCondition(in.Conditions, kmapi.Condition{
				Type:    apis.BackendRepositoryInitialized,
				Status:  core.ConditionFalse,
				Reason:  apis.FailedToInitializeBackendRepository,
				Message: fmt.Sprintf("Failed to initialize backend repository. Reason: %v", err.Error()),
			},
			)
			return backupSession.UID, in
		},
		metav1.UpdateOptions{},
	)
}

func SetBackendRepositoryInitializedConditionToTrue(stashClient cs.Interface, backupSession *v1beta1.BackupSession) (*v1beta1.BackupSession, error) {
	return stash_util.UpdateBackupSessionStatus(
		context.TODO(),
		stashClient.StashV1beta1(),
		backupSession.ObjectMeta,
		func(in *api_v1beta1.BackupSessionStatus) (types.UID, *api_v1beta1.BackupSessionStatus) {
			in.Conditions = kmapi.SetCondition(in.Conditions, kmapi.Condition{
				Type:    apis.BackendRepositoryInitialized,
				Status:  core.ConditionTrue,
				Reason:  apis.BackendRepositoryFound,
				Message: "Repository exist in the backend.",
			})
			return backupSession.UID, in
		},
		metav1.UpdateOptions{},
	)
}

func SetRepositoryFoundConditionToUnknown(i interface{}, err error) error {
	switch in := i.(type) {
	case invoker.BackupInvoker:
		return in.SetCondition(nil, kmapi.Condition{
			Type:   apis.RepositoryFound,
			Status: core.ConditionUnknown,
			Reason: apis.UnableToCheckRepositoryAvailability,
			Message: fmt.Sprintf("Failed to check whether the Repository %s/%s exist or not. Reason: %v",
				in.GetRepoRef().Namespace,
				in.GetRepoRef().Name,
				err.Error(),
			),
		})
	case invoker.RestoreInvoker:
		return in.SetCondition(nil, kmapi.Condition{
			Type:   apis.RepositoryFound,
			Status: core.ConditionUnknown,
			Reason: apis.UnableToCheckRepositoryAvailability,
			Message: fmt.Sprintf("Failed to check whether the Repository %s/%s exist or not. Reason: %v",
				in.GetRepoRef().Namespace,
				in.GetRepoRef().Name,
				err.Error(),
			),
		})
	default:
		return fmt.Errorf("unable to set %s condition. Reason: invoker type unknown", apis.RepositoryFound)
	}
}

func SetRepositoryFoundConditionToFalse(i interface{}) error {
	switch in := i.(type) {
	case invoker.BackupInvoker:
		return in.SetCondition(nil, kmapi.Condition{
			Type:   apis.RepositoryFound,
			Status: core.ConditionFalse,
			Reason: apis.RepositoryNotAvailable,
			Message: fmt.Sprintf("Repository %s/%s does not exist.",
				in.GetRepoRef().Namespace,
				in.GetRepoRef().Name,
			),
		})
	case invoker.RestoreInvoker:
		return in.SetCondition(nil, kmapi.Condition{
			Type:   apis.RepositoryFound,
			Status: core.ConditionFalse,
			Reason: apis.RepositoryNotAvailable,
			Message: fmt.Sprintf("Repository %s/%s does not exist.",
				in.GetRepoRef().Namespace,
				in.GetRepoRef().Name,
			),
		})
	default:
		return fmt.Errorf("unable to set %s condition. Reason: invoker type unknown", apis.RepositoryFound)
	}
}

func SetRepositoryFoundConditionToTrue(i interface{}) error {
	switch in := i.(type) {
	case invoker.BackupInvoker:
		return in.SetCondition(nil, kmapi.Condition{
			Type:   apis.RepositoryFound,
			Status: core.ConditionTrue,
			Reason: apis.RepositoryAvailable,
			Message: fmt.Sprintf("Repository %s/%s exist.",
				in.GetRepoRef().Namespace,
				in.GetRepoRef().Name,
			),
		})
	case invoker.RestoreInvoker:
		return in.SetCondition(nil, kmapi.Condition{
			Type:   apis.RepositoryFound,
			Status: core.ConditionTrue,
			Reason: apis.RepositoryAvailable,
			Message: fmt.Sprintf("Repository %s/%s exist.",
				in.GetRepoRef().Namespace,
				in.GetRepoRef().Name,
			),
		})
	default:
		return fmt.Errorf("unable to set %s condition. Reason: invoker type unknown", apis.RepositoryFound)
	}
}

func SetValidationPassedToTrue(i interface{}) error {
	switch in := i.(type) {
	case invoker.BackupInvoker:
		return in.SetCondition(nil, kmapi.Condition{
			Type:    apis.ValidationPassed,
			Status:  core.ConditionTrue,
			Reason:  apis.ResourceValidationPassed,
			Message: "Successfully validated.",
		})
	case invoker.RestoreInvoker:
		return in.SetCondition(nil, kmapi.Condition{
			Type:    apis.ValidationPassed,
			Status:  core.ConditionTrue,
			Reason:  apis.ResourceValidationPassed,
			Message: "Successfully validated.",
		})
	default:
		return fmt.Errorf("unable to set %s condition. Reason: invoker type unknown", apis.ValidationPassed)
	}
}

func SetValidationPassedToFalse(i interface{}, err error) error {
	switch in := i.(type) {
	case invoker.BackupInvoker:
		return in.SetCondition(nil, kmapi.Condition{
			Type:    apis.ValidationPassed,
			Status:  core.ConditionFalse,
			Reason:  apis.ResourceValidationFailed,
			Message: err.Error(),
		})
	case invoker.RestoreInvoker:
		return in.SetCondition(nil, kmapi.Condition{
			Type:    apis.ValidationPassed,
			Status:  core.ConditionFalse,
			Reason:  apis.ResourceValidationFailed,
			Message: err.Error(),
		})
	default:
		return fmt.Errorf("unable to set %s condition. Reason: invoker type unknown", apis.ValidationPassed)
	}
}

func SetBackendSecretFoundConditionToUnknown(i interface{}, secretName string, err error) error {
	switch in := i.(type) {
	case invoker.BackupInvoker:
		return in.SetCondition(nil, kmapi.Condition{
			Type:   apis.BackendSecretFound,
			Status: core.ConditionUnknown,
			Reason: apis.UnableToCheckBackendSecretAvailability,
			Message: fmt.Sprintf("Failed to check whether the backend Secret %s/%s exist or not. Reason: %v",
				in.GetRepoRef().Namespace,
				secretName,
				err.Error(),
			),
		})
	case invoker.RestoreInvoker:
		return in.SetCondition(nil, kmapi.Condition{
			Type:   apis.BackendSecretFound,
			Status: core.ConditionUnknown,
			Reason: apis.UnableToCheckBackendSecretAvailability,
			Message: fmt.Sprintf("Failed to check whether the backend Secret %s/%s exist or not. Reason: %v",
				in.GetRepoRef().Namespace,
				secretName,
				err.Error(),
			),
		})
	default:
		return fmt.Errorf("unable to set %s condition. Reason: invoker type unknown", apis.BackendSecretFound)
	}
}

func SetBackendSecretFoundConditionToFalse(i interface{}, secretName string) error {
	switch in := i.(type) {
	case invoker.BackupInvoker:
		return in.SetCondition(nil, kmapi.Condition{
			Type:   apis.BackendSecretFound,
			Status: core.ConditionFalse,
			Reason: apis.BackendSecretNotAvailable,
			Message: fmt.Sprintf("Backend Secret %s/%s does not exist.",
				in.GetRepoRef().Namespace,
				secretName,
			),
		})
	case invoker.RestoreInvoker:
		return in.SetCondition(nil, kmapi.Condition{
			Type:   apis.BackendSecretFound,
			Status: core.ConditionFalse,
			Reason: apis.BackendSecretNotAvailable,
			Message: fmt.Sprintf("Backend Secret %s/%s does not exist.",
				in.GetRepoRef().Namespace,
				secretName,
			),
		})
	default:
		return fmt.Errorf("unable to set %s condition. Reason: invoker type unknown", apis.BackendSecretFound)
	}
}

func SetBackendSecretFoundConditionToTrue(i interface{}, secretName string) error {
	switch in := i.(type) {
	case invoker.BackupInvoker:
		return in.SetCondition(nil, kmapi.Condition{
			Type:   apis.BackendSecretFound,
			Status: core.ConditionTrue,
			Reason: apis.BackendSecretAvailable,
			Message: fmt.Sprintf("Backend Secret %s/%s exist.",
				in.GetRepoRef().Namespace,
				secretName,
			),
		})
	case invoker.RestoreInvoker:
		return in.SetCondition(nil, kmapi.Condition{
			Type:   apis.BackendSecretFound,
			Status: core.ConditionTrue,
			Reason: apis.BackendSecretAvailable,
			Message: fmt.Sprintf("Backend Secret %s/%s exist.",
				in.GetRepoRef().Namespace,
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
		func(in *api_v1beta1.BackupSessionStatus) (types.UID, *api_v1beta1.BackupSessionStatus) {
			in.Conditions = kmapi.SetCondition(in.Conditions, kmapi.Condition{
				Type:    apis.RetentionPolicyApplied,
				Status:  core.ConditionFalse,
				Reason:  apis.FailedToApplyRetentionPolicy,
				Message: fmt.Sprintf("Failed to apply retention policy. Reason: %v", err.Error()),
			},
			)
			return backupSession.UID, in
		},
		metav1.UpdateOptions{},
	)
}

func SetRetentionPolicyAppliedConditionToTrue(stashClient cs.Interface, backupSession *v1beta1.BackupSession) (*v1beta1.BackupSession, error) {
	return stash_util.UpdateBackupSessionStatus(
		context.TODO(),
		stashClient.StashV1beta1(),
		backupSession.ObjectMeta,
		func(in *api_v1beta1.BackupSessionStatus) (types.UID, *api_v1beta1.BackupSessionStatus) {
			in.Conditions = kmapi.SetCondition(in.Conditions, kmapi.Condition{
				Type:    apis.RetentionPolicyApplied,
				Status:  core.ConditionTrue,
				Reason:  apis.SuccessfullyAppliedRetentionPolicy,
				Message: "Successfully applied retention policy.",
			},
			)
			return backupSession.UID, in
		},
		metav1.UpdateOptions{},
	)
}

func SetRepositoryIntegrityVerifiedConditionToFalse(stashClient cs.Interface, backupSession *v1beta1.BackupSession, err error) (*v1beta1.BackupSession, error) {
	return stash_util.UpdateBackupSessionStatus(
		context.TODO(),
		stashClient.StashV1beta1(),
		backupSession.ObjectMeta,
		func(in *api_v1beta1.BackupSessionStatus) (types.UID, *api_v1beta1.BackupSessionStatus) {
			in.Conditions = kmapi.SetCondition(in.Conditions, kmapi.Condition{
				Type:    apis.RepositoryIntegrityVerified,
				Status:  core.ConditionFalse,
				Reason:  apis.FailedToVerifyRepositoryIntegrity,
				Message: fmt.Sprintf("Repository integrity verification failed. Reason: %v", err.Error()),
			},
			)
			return backupSession.UID, in
		},
		metav1.UpdateOptions{},
	)
}

func SetRepositoryIntegrityVerifiedConditionToTrue(stashClient cs.Interface, backupSession *v1beta1.BackupSession) (*v1beta1.BackupSession, error) {
	return stash_util.UpdateBackupSessionStatus(
		context.TODO(),
		stashClient.StashV1beta1(),
		backupSession.ObjectMeta,
		func(in *api_v1beta1.BackupSessionStatus) (types.UID, *api_v1beta1.BackupSessionStatus) {
			in.Conditions = kmapi.SetCondition(in.Conditions, kmapi.Condition{
				Type:    apis.RepositoryIntegrityVerified,
				Status:  core.ConditionTrue,
				Reason:  apis.SuccessfullyVerifiedRepositoryIntegrity,
				Message: "Repository integrity verification succeeded.",
			},
			)
			return backupSession.UID, in
		},
		metav1.UpdateOptions{},
	)
}

func SetRepositoryMetricsPushedConditionToFalse(stashClient cs.Interface, backupSession *v1beta1.BackupSession, err error) (*v1beta1.BackupSession, error) {
	return stash_util.UpdateBackupSessionStatus(
		context.TODO(),
		stashClient.StashV1beta1(),
		backupSession.ObjectMeta,
		func(in *api_v1beta1.BackupSessionStatus) (types.UID, *api_v1beta1.BackupSessionStatus) {
			in.Conditions = kmapi.SetCondition(in.Conditions, kmapi.Condition{
				Type:    apis.RepositoryMetricsPushed,
				Status:  core.ConditionFalse,
				Reason:  apis.FailedToPushRepositoryMetrics,
				Message: fmt.Sprintf("Failed to push repository metrics. Reason: %v", err.Error()),
			},
			)
			return backupSession.UID, in
		},
		metav1.UpdateOptions{},
	)
}

func SetRepositoryMetricsPushedConditionToTrue(stashClient cs.Interface, backupSession *v1beta1.BackupSession) (*v1beta1.BackupSession, error) {
	return stash_util.UpdateBackupSessionStatus(
		context.TODO(),
		stashClient.StashV1beta1(),
		backupSession.ObjectMeta,
		func(in *api_v1beta1.BackupSessionStatus) (types.UID, *api_v1beta1.BackupSessionStatus) {
			in.Conditions = kmapi.SetCondition(in.Conditions, kmapi.Condition{
				Type:    apis.RepositoryMetricsPushed,
				Status:  core.ConditionTrue,
				Reason:  apis.SuccessfullyPushedRepositoryMetrics,
				Message: "Successfully pushed repository metrics.",
			},
			)
			return backupSession.UID, in
		},
		metav1.UpdateOptions{},
	)
}

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
	"fmt"

	"stash.appscode.dev/apimachinery/apis/stash/v1beta1"
	"stash.appscode.dev/apimachinery/pkg/invoker"

	core "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	kmapi "kmodules.xyz/client-go/api/v1"
)

func SetBackendRepositoryInitializedConditionToFalse(session *invoker.BackupSessionHandler, err error) error {
	return session.UpdateStatus(&v1beta1.BackupSessionStatus{
		Conditions: []kmapi.Condition{
			{
				Type:               v1beta1.BackendRepositoryInitialized,
				Status:             core.ConditionFalse,
				Reason:             v1beta1.FailedToInitializeBackendRepository,
				Message:            fmt.Sprintf("Failed to initialize backend repository. Reason: %v", err.Error()),
				LastTransitionTime: metav1.Now(),
			},
		},
	})
}

func SetBackendRepositoryInitializedConditionToTrue(session *invoker.BackupSessionHandler) error {
	return session.UpdateStatus(&v1beta1.BackupSessionStatus{
		Conditions: []kmapi.Condition{
			{
				Type:               v1beta1.BackendRepositoryInitialized,
				Status:             core.ConditionTrue,
				Reason:             v1beta1.BackendRepositoryFound,
				Message:            "Repository exist in the backend.",
				LastTransitionTime: metav1.Now(),
			},
		},
	})
}

func SetBackupExecutorEnsuredToFalse(session *invoker.BackupSessionHandler, target v1beta1.TargetRef, err error) error {
	return session.UpdateStatus(&v1beta1.BackupSessionStatus{
		Targets: []v1beta1.BackupTargetStatus{
			{
				Ref: target,
				Conditions: []kmapi.Condition{
					{
						Type:               v1beta1.BackupExecutorEnsured,
						Status:             core.ConditionFalse,
						Reason:             v1beta1.FailedToEnsureBackupExecutor,
						Message:            fmt.Sprintf("Failed to ensure backup executor. Reason: %v", err.Error()),
						LastTransitionTime: metav1.Now(),
					},
				},
			},
		},
	})
}

func SetBackupExecutorEnsuredToTrue(session *invoker.BackupSessionHandler, target v1beta1.TargetRef) error {
	return session.UpdateStatus(&v1beta1.BackupSessionStatus{
		Targets: []v1beta1.BackupTargetStatus{
			{
				Ref: target,
				Conditions: []kmapi.Condition{
					{
						Type:               v1beta1.BackupExecutorEnsured,
						Status:             core.ConditionTrue,
						Reason:             v1beta1.SuccessfullyEnsuredBackupExecutor,
						Message:            "Successfully ensured backup executor.",
						LastTransitionTime: metav1.Now(),
					},
				},
			},
		},
	})
}

func SetPreBackupHookExecutionSucceededToFalse(session *invoker.BackupSessionHandler, target v1beta1.TargetRef, err error) error {
	return session.UpdateStatus(&v1beta1.BackupSessionStatus{
		Targets: []v1beta1.BackupTargetStatus{
			{
				Ref: target,
				Conditions: []kmapi.Condition{
					{
						Type:               v1beta1.PreBackupHookExecutionSucceeded,
						Status:             core.ConditionFalse,
						Reason:             v1beta1.FailedToExecutePreBackupHook,
						Message:            fmt.Sprintf("Failed to execute preBackup hook. Reason: %v", err.Error()),
						LastTransitionTime: metav1.Now(),
					},
				},
			},
		},
	})
}

func SetPreBackupHookExecutionSucceededToTrue(session *invoker.BackupSessionHandler, target v1beta1.TargetRef) error {
	return session.UpdateStatus(&v1beta1.BackupSessionStatus{
		Targets: []v1beta1.BackupTargetStatus{
			{
				Ref: target,
				Conditions: []kmapi.Condition{
					{
						Type:               v1beta1.PreBackupHookExecutionSucceeded,
						Status:             core.ConditionTrue,
						Reason:             v1beta1.SuccessfullyExecutedPreBackupHook,
						Message:            "Successfully executed preBackup hook.",
						LastTransitionTime: metav1.Now(),
					},
				},
			},
		},
	})
}

func SetPostBackupHookExecutionSucceededToFalse(session *invoker.BackupSessionHandler, target v1beta1.TargetRef, err error) error {
	return session.UpdateStatus(&v1beta1.BackupSessionStatus{
		Targets: []v1beta1.BackupTargetStatus{
			{
				Ref: target,
				Conditions: []kmapi.Condition{
					{
						Type:               v1beta1.PostBackupHookExecutionSucceeded,
						Status:             core.ConditionFalse,
						Reason:             v1beta1.FailedToExecutePostBackupHook,
						Message:            fmt.Sprintf("Failed to execute postBackup hook. Reason: %v", err.Error()),
						LastTransitionTime: metav1.Now(),
					},
				},
			},
		},
	})
}

func SetPostBackupHookExecutionSucceededToTrue(session *invoker.BackupSessionHandler, target v1beta1.TargetRef) error {
	return SetPostBackupHookExecutionSucceededToTrueWithMsg(session, target, "Successfully executed postBackup hook.")
}

func SetPostBackupHookExecutionSucceededToTrueWithMsg(session *invoker.BackupSessionHandler, target v1beta1.TargetRef, msg string) error {
	return session.UpdateStatus(&v1beta1.BackupSessionStatus{
		Targets: []v1beta1.BackupTargetStatus{
			{
				Ref: target,
				Conditions: []kmapi.Condition{
					{
						Type:               v1beta1.PostBackupHookExecutionSucceeded,
						Status:             core.ConditionTrue,
						Reason:             v1beta1.SuccessfullyExecutedPostBackupHook,
						Message:            msg,
						LastTransitionTime: metav1.Now(),
					},
				},
			},
		},
	})
}

func SetGlobalPreBackupHookSucceededConditionToFalse(session *invoker.BackupSessionHandler, hookErr error) error {
	return session.UpdateStatus(&v1beta1.BackupSessionStatus{
		Conditions: []kmapi.Condition{
			{
				Type:               v1beta1.GlobalPreBackupHookSucceeded,
				Status:             core.ConditionFalse,
				Reason:             v1beta1.GlobalPreBackupHookExecutionFailed,
				Message:            fmt.Sprintf("Failed to execute global PreBackup Hook. Reason: %v.", hookErr),
				LastTransitionTime: metav1.Now(),
			},
		},
	})
}

func SetGlobalPreBackupHookSucceededConditionToTrue(session *invoker.BackupSessionHandler) error {
	return session.UpdateStatus(&v1beta1.BackupSessionStatus{
		Conditions: []kmapi.Condition{
			{
				Type:               v1beta1.GlobalPreBackupHookSucceeded,
				Status:             core.ConditionTrue,
				Reason:             v1beta1.GlobalPreBackupHookExecutedSuccessfully,
				Message:            "Global PreBackup hook has been executed successfully",
				LastTransitionTime: metav1.Now(),
			},
		},
	})
}

func SetGlobalPostBackupHookSucceededConditionToFalse(session *invoker.BackupSessionHandler, hookErr error) error {
	return session.UpdateStatus(&v1beta1.BackupSessionStatus{
		Conditions: []kmapi.Condition{
			{
				Type:               v1beta1.GlobalPostBackupHookSucceeded,
				Status:             core.ConditionFalse,
				Reason:             v1beta1.GlobalPostBackupHookExecutionFailed,
				Message:            fmt.Sprintf("Failed to execute global PostBackup Hook. Reason: %v.", hookErr),
				LastTransitionTime: metav1.Now(),
			},
		},
	})
}

func SetGlobalPostBackupHookSucceededConditionToTrue(session *invoker.BackupSessionHandler) error {
	return SetGlobalPostBackupHookSucceededConditionToTrueWithMsg(session, "Global PostBackup hook has been executed successfully")
}

func SetGlobalPostBackupHookSucceededConditionToTrueWithMsg(session *invoker.BackupSessionHandler, msg string) error {
	return session.UpdateStatus(&v1beta1.BackupSessionStatus{
		Conditions: []kmapi.Condition{
			{
				Type:               v1beta1.GlobalPostBackupHookSucceeded,
				Status:             core.ConditionTrue,
				Reason:             v1beta1.GlobalPostBackupHookExecutedSuccessfully,
				Message:            msg,
				LastTransitionTime: metav1.Now(),
			},
		},
	})
}

func SetRetentionPolicyAppliedConditionToFalse(session *invoker.BackupSessionHandler, err error) error {
	return session.UpdateStatus(&v1beta1.BackupSessionStatus{
		Conditions: []kmapi.Condition{
			{
				Type:               v1beta1.RetentionPolicyApplied,
				Status:             core.ConditionFalse,
				Reason:             v1beta1.FailedToApplyRetentionPolicy,
				Message:            fmt.Sprintf("Failed to apply retention policy. Reason: %v", err.Error()),
				LastTransitionTime: metav1.Now(),
			},
		},
	})
}

func SetRetentionPolicyAppliedConditionToTrue(session *invoker.BackupSessionHandler) error {
	return session.UpdateStatus(&v1beta1.BackupSessionStatus{
		Conditions: []kmapi.Condition{
			{
				Type:               v1beta1.RetentionPolicyApplied,
				Status:             core.ConditionTrue,
				Reason:             v1beta1.SuccessfullyAppliedRetentionPolicy,
				Message:            "Successfully applied retention policy.",
				LastTransitionTime: metav1.Now(),
			},
		},
	})
}

func SetRepositoryIntegrityVerifiedConditionToFalse(session *invoker.BackupSessionHandler, err error) error {
	return session.UpdateStatus(&v1beta1.BackupSessionStatus{
		Conditions: []kmapi.Condition{
			{
				Type:               v1beta1.RepositoryIntegrityVerified,
				Status:             core.ConditionFalse,
				Reason:             v1beta1.FailedToVerifyRepositoryIntegrity,
				Message:            fmt.Sprintf("Repository integrity verification failed. Reason: %v", err.Error()),
				LastTransitionTime: metav1.Now(),
			},
		},
	})
}

func SetRepositoryIntegrityVerifiedConditionToTrue(session *invoker.BackupSessionHandler) error {
	return session.UpdateStatus(&v1beta1.BackupSessionStatus{
		Conditions: []kmapi.Condition{
			{
				Type:               v1beta1.RepositoryIntegrityVerified,
				Status:             core.ConditionTrue,
				Reason:             v1beta1.SuccessfullyVerifiedRepositoryIntegrity,
				Message:            "Repository integrity verification succeeded.",
				LastTransitionTime: metav1.Now(),
			},
		},
	})
}

func SetRepositoryMetricsPushedConditionToFalse(session *invoker.BackupSessionHandler, err error) error {
	return session.UpdateStatus(&v1beta1.BackupSessionStatus{
		Conditions: []kmapi.Condition{
			{
				Type:               v1beta1.RepositoryMetricsPushed,
				Status:             core.ConditionFalse,
				Reason:             v1beta1.FailedToPushRepositoryMetrics,
				Message:            fmt.Sprintf("Failed to push repository metrics. Reason: %v", err.Error()),
				LastTransitionTime: metav1.Now(),
			},
		},
	})
}

func SetRepositoryMetricsPushedConditionToTrue(session *invoker.BackupSessionHandler) error {
	return session.UpdateStatus(&v1beta1.BackupSessionStatus{
		Conditions: []kmapi.Condition{
			{
				Type:               v1beta1.RepositoryMetricsPushed,
				Status:             core.ConditionTrue,
				Reason:             v1beta1.SuccessfullyPushedRepositoryMetrics,
				Message:            "Successfully pushed repository metrics.",
				LastTransitionTime: metav1.Now(),
			},
		},
	})
}

func SetBackupSkippedConditionToTrue(session *invoker.BackupSessionHandler, msg string) error {
	return session.UpdateStatus(&v1beta1.BackupSessionStatus{
		Conditions: []kmapi.Condition{
			{
				Type:               v1beta1.BackupSkipped,
				Status:             core.ConditionTrue,
				Reason:             v1beta1.SkippedTakingNewBackup,
				Message:            msg,
				LastTransitionTime: metav1.Now(),
			},
		},
	})
}

func SetBackupMetricsPushedConditionToFalse(session *invoker.BackupSessionHandler, err error) error {
	return session.UpdateStatus(&v1beta1.BackupSessionStatus{
		Conditions: []kmapi.Condition{
			{
				Type:               v1beta1.MetricsPushed,
				Status:             core.ConditionFalse,
				Reason:             v1beta1.FailedToPushMetrics,
				Message:            fmt.Sprintf("Failed to push metrics. Reason: %v", err.Error()),
				LastTransitionTime: metav1.Now(),
			},
		},
	})
}

func SetBackupMetricsPushedConditionToTrue(session *invoker.BackupSessionHandler) error {
	return session.UpdateStatus(&v1beta1.BackupSessionStatus{
		Conditions: []kmapi.Condition{
			{
				Type:               v1beta1.MetricsPushed,
				Status:             core.ConditionTrue,
				Reason:             v1beta1.SuccessfullyPushedMetrics,
				Message:            "Successfully pushed metrics.",
				LastTransitionTime: metav1.Now(),
			},
		},
	})
}

func SetBackupHistoryCleanedConditionToFalse(session *invoker.BackupSessionHandler, err error) error {
	return session.UpdateStatus(&v1beta1.BackupSessionStatus{
		Conditions: []kmapi.Condition{
			{
				Type:               v1beta1.BackupHistoryCleaned,
				Status:             core.ConditionFalse,
				Reason:             v1beta1.FailedToCleanBackupHistory,
				Message:            fmt.Sprintf("Failed to cleanup old BackupSessions. Reason: %v", err.Error()),
				LastTransitionTime: metav1.Now(),
			},
		},
	})
}

func SetBackupHistoryCleanedConditionToTrue(session *invoker.BackupSessionHandler) error {
	return session.UpdateStatus(&v1beta1.BackupSessionStatus{
		Conditions: []kmapi.Condition{
			{
				Type:               v1beta1.BackupHistoryCleaned,
				Status:             core.ConditionTrue,
				Reason:             v1beta1.SuccessfullyCleanedBackupHistory,
				Message:            "Successfully cleaned up backup history according to backupHistoryLimit.",
				LastTransitionTime: metav1.Now(),
			},
		},
	})
}

func SetBackupDeadlineExceededConditionToTrue(session *invoker.BackupSessionHandler, timeOut string) error {
	return session.UpdateStatus(&v1beta1.BackupSessionStatus{
		Conditions: []kmapi.Condition{
			{
				Type:               v1beta1.DeadlineExceeded,
				Status:             core.ConditionTrue,
				Reason:             v1beta1.FailedToCompleteWithinDeadline,
				Message:            fmt.Sprintf("Failed to complete backup within %s.", timeOut),
				LastTransitionTime: metav1.Now(),
			},
		},
	})
}

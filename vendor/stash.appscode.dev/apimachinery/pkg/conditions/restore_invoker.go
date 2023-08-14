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

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	kmapi "kmodules.xyz/client-go/api/v1"
)

func SetRestoreTargetFoundConditionToTrue(inv invoker.RestoreInvoker, index int) error {
	target := inv.GetTargetInfo()[index].Target
	return inv.SetCondition(&target.Ref, kmapi.Condition{
		Type:   v1beta1.RestoreTargetFound,
		Status: metav1.ConditionTrue,
		Reason: v1beta1.TargetAvailable,
		Message: fmt.Sprintf("Restore target %s %s/%s found.",
			target.Ref.Kind,
			target.Ref.Namespace,
			target.Ref.Name,
		),
		LastTransitionTime: metav1.Now(),
	})
}

func SetRestoreTargetFoundConditionToFalse(inv invoker.RestoreInvoker, index int) error {
	target := inv.GetTargetInfo()[index].Target
	return inv.SetCondition(&target.Ref, kmapi.Condition{
		Type:   v1beta1.RestoreTargetFound,
		Status: metav1.ConditionFalse,
		Reason: v1beta1.TargetNotAvailable,
		Message: fmt.Sprintf("Restore target %s %s/%s does not exist.",
			target.Ref.Kind,
			target.Ref.Namespace,
			target.Ref.Name,
		),
		LastTransitionTime: metav1.Now(),
	})
}

func SetRestoreTargetFoundConditionToUnknown(inv invoker.RestoreInvoker, index int, err error) error {
	target := inv.GetTargetInfo()[index].Target
	return inv.SetCondition(&target.Ref, kmapi.Condition{
		Type:   v1beta1.RestoreTargetFound,
		Status: metav1.ConditionUnknown,
		Reason: v1beta1.UnableToCheckTargetAvailability,
		Message: fmt.Sprintf("Failed to check whether restore target %s %s/%s exist or not. Reason: %v",
			target.Ref.Kind,
			target.Ref.Namespace,
			target.Ref.Name,
			err,
		),
		LastTransitionTime: metav1.Now(),
	})
}

func SetRestoreJobCreatedConditionToTrue(inv invoker.RestoreInvoker, tref *v1beta1.TargetRef) error {
	return inv.SetCondition(tref, kmapi.Condition{
		Type:               v1beta1.RestoreJobCreated,
		Status:             metav1.ConditionTrue,
		Reason:             v1beta1.RestoreJobCreationSucceeded,
		Message:            "Successfully created restore job.",
		LastTransitionTime: metav1.Now(),
	})
}

func SetRestoreJobCreatedConditionToFalse(inv invoker.RestoreInvoker, tref *v1beta1.TargetRef, err error) error {
	return inv.SetCondition(tref, kmapi.Condition{
		Type:               v1beta1.RestoreJobCreated,
		Status:             metav1.ConditionFalse,
		Reason:             v1beta1.RestoreJobCreationFailed,
		Message:            fmt.Sprintf("Failed to create restore job. Reason: %v", err.Error()),
		LastTransitionTime: metav1.Now(),
	})
}

func SetInitContainerInjectedConditionToTrue(inv invoker.RestoreInvoker, tref *v1beta1.TargetRef) error {
	return inv.SetCondition(tref, kmapi.Condition{
		Type:               v1beta1.StashInitContainerInjected,
		Status:             metav1.ConditionTrue,
		Reason:             v1beta1.InitContainerInjectionSucceeded,
		Message:            "Successfully injected stash init-container.",
		LastTransitionTime: metav1.Now(),
	})
}

func SetInitContainerInjectedConditionToFalse(inv invoker.RestoreInvoker, tref *v1beta1.TargetRef, err error) error {
	return inv.SetCondition(tref, kmapi.Condition{
		Type:               v1beta1.StashInitContainerInjected,
		Status:             metav1.ConditionFalse,
		Reason:             v1beta1.InitContainerInjectionFailed,
		Message:            fmt.Sprintf("Failed to inject Stash init-container. Reason: %v", err.Error()),
		LastTransitionTime: metav1.Now(),
	})
}

func SetRestoreCompletedConditionToTrue(inv invoker.RestoreInvoker, tref *v1beta1.TargetRef, msg string) error {
	return inv.SetCondition(tref, kmapi.Condition{
		Type:               v1beta1.RestoreCompleted,
		Status:             metav1.ConditionTrue,
		Reason:             v1beta1.PostRestoreTasksExecuted,
		Message:            msg,
		LastTransitionTime: metav1.Now(),
	})
}

func SetRestoreCompletedConditionToFalse(inv invoker.RestoreInvoker, tref *v1beta1.TargetRef, msg string) error {
	return inv.SetCondition(tref, kmapi.Condition{
		Type:               v1beta1.RestoreCompleted,
		Status:             metav1.ConditionFalse,
		Reason:             v1beta1.PostRestoreTasksNotExecuted,
		Message:            msg,
		LastTransitionTime: metav1.Now(),
	})
}

func SetRestoreExecutorEnsuredToTrue(inv invoker.RestoreInvoker, tref *v1beta1.TargetRef, msg string) error {
	return inv.SetCondition(tref, kmapi.Condition{
		Type:               v1beta1.RestoreExecutorEnsured,
		Status:             metav1.ConditionTrue,
		Reason:             v1beta1.SuccessfullyEnsuredRestoreExecutor,
		Message:            msg,
		LastTransitionTime: metav1.Now(),
	})
}

func SetRestoreExecutorEnsuredToFalse(inv invoker.RestoreInvoker, tref *v1beta1.TargetRef, msg string) error {
	return inv.SetCondition(tref, kmapi.Condition{
		Type:               v1beta1.RestoreExecutorEnsured,
		Status:             metav1.ConditionFalse,
		Reason:             v1beta1.FailedToEnsureRestoreExecutor,
		Message:            msg,
		LastTransitionTime: metav1.Now(),
	})
}

func SetRestoreMetricsPushedConditionToFalse(inv invoker.RestoreInvoker, err error) error {
	return inv.SetCondition(nil, kmapi.Condition{
		Type:               v1beta1.MetricsPushed,
		Status:             metav1.ConditionFalse,
		Reason:             v1beta1.FailedToPushMetrics,
		Message:            fmt.Sprintf("Failed to push metrics. Reason: %v", err.Error()),
		LastTransitionTime: metav1.Now(),
	})
}

func SetRestoreMetricsPushedConditionToTrue(inv invoker.RestoreInvoker) error {
	return inv.SetCondition(nil, kmapi.Condition{
		Type:               v1beta1.MetricsPushed,
		Status:             metav1.ConditionTrue,
		Reason:             v1beta1.SuccessfullyPushedMetrics,
		Message:            "Successfully pushed metrics.",
		LastTransitionTime: metav1.Now(),
	})
}

func SetPreRestoreHookExecutionSucceededToFalse(inv invoker.RestoreInvoker, err error) error {
	return inv.SetCondition(nil, kmapi.Condition{
		Type:               v1beta1.PreRestoreHookExecutionSucceeded,
		Status:             metav1.ConditionFalse,
		Reason:             v1beta1.FailedToExecutePreRestoreHook,
		Message:            fmt.Sprintf("Failed to execute preRestore hook. Reason: %v", err.Error()),
		LastTransitionTime: metav1.Now(),
	})
}

func SetPreRestoreHookExecutionSucceededToTrue(inv invoker.RestoreInvoker) error {
	return inv.SetCondition(nil, kmapi.Condition{
		Type:               v1beta1.PreRestoreHookExecutionSucceeded,
		Status:             metav1.ConditionTrue,
		Reason:             v1beta1.SuccessfullyExecutedPreRestoreHook,
		Message:            "Successfully executed preRestore hook.",
		LastTransitionTime: metav1.Now(),
	})
}

func SetPostRestoreHookExecutionSucceededToFalse(inv invoker.RestoreInvoker, err error) error {
	return inv.SetCondition(nil, kmapi.Condition{
		Type:               v1beta1.PostRestoreHookExecutionSucceeded,
		Status:             metav1.ConditionFalse,
		Reason:             v1beta1.FailedToExecutePostRestoreHook,
		Message:            fmt.Sprintf("Failed to execute postRestore hook. Reason: %v", err.Error()),
		LastTransitionTime: metav1.Now(),
	})
}

func SetPostRestoreHookExecutionSucceededToTrue(inv invoker.RestoreInvoker) error {
	return SetPostRestoreHookExecutionSucceededToTrueWithMsg(inv, "Successfully executed postRestore hook.")
}

func SetPostRestoreHookExecutionSucceededToTrueWithMsg(inv invoker.RestoreInvoker, msg string) error {
	return inv.SetCondition(nil, kmapi.Condition{
		Type:               v1beta1.PostRestoreHookExecutionSucceeded,
		Status:             metav1.ConditionTrue,
		Reason:             v1beta1.SuccessfullyExecutedPostRestoreHook,
		Message:            msg,
		LastTransitionTime: metav1.Now(),
	})
}

func SetGlobalPreRestoreHookSucceededConditionToFalse(invoker invoker.RestoreInvoker, hookErr error) error {
	return invoker.SetCondition(nil, kmapi.Condition{
		Type:               v1beta1.GlobalPreRestoreHookSucceeded,
		Status:             metav1.ConditionFalse,
		Reason:             v1beta1.GlobalPreRestoreHookExecutionFailed,
		Message:            fmt.Sprintf("Failed to execute global PreRestore Hook. Reason: %v.", hookErr),
		LastTransitionTime: metav1.Now(),
	})
}

func SetGlobalPreRestoreHookSucceededConditionToTrue(invoker invoker.RestoreInvoker) error {
	return invoker.SetCondition(nil, kmapi.Condition{
		Type:               v1beta1.GlobalPreRestoreHookSucceeded,
		Status:             metav1.ConditionTrue,
		Reason:             v1beta1.GlobalPreRestoreHookExecutedSuccessfully,
		Message:            "Global PreRestore hook has been executed successfully",
		LastTransitionTime: metav1.Now(),
	})
}

func SetGlobalPostRestoreHookSucceededConditionToFalse(invoker invoker.RestoreInvoker, hookErr error) error {
	return invoker.SetCondition(nil, kmapi.Condition{
		Type:               v1beta1.GlobalPostRestoreHookSucceeded,
		Status:             metav1.ConditionFalse,
		Reason:             v1beta1.GlobalPostRestoreHookExecutionFailed,
		Message:            fmt.Sprintf("Failed to execute global PostRestore Hook. Reason: %v.", hookErr),
		LastTransitionTime: metav1.Now(),
	})
}

func SetGlobalPostRestoreHookSucceededConditionToTrue(invoker invoker.RestoreInvoker) error {
	return SetGlobalPostRestoreHookSucceededConditionToTrueWithMsg(invoker, "Global PostRestore hook has been executed successfully")
}

func SetGlobalPostRestoreHookSucceededConditionToTrueWithMsg(invoker invoker.RestoreInvoker, msg string) error {
	return invoker.SetCondition(nil, kmapi.Condition{
		Type:               v1beta1.GlobalPostRestoreHookSucceeded,
		Status:             metav1.ConditionTrue,
		Reason:             v1beta1.GlobalPostRestoreHookExecutedSuccessfully,
		Message:            msg,
		LastTransitionTime: metav1.Now(),
	})
}

func SetRestoreDeadlineExceededConditionToTrue(invoker invoker.RestoreInvoker, timeOut metav1.Duration) error {
	return invoker.SetCondition(nil, kmapi.Condition{
		Type:               v1beta1.DeadlineExceeded,
		Status:             metav1.ConditionTrue,
		Reason:             v1beta1.FailedToCompleteWithinDeadline,
		Message:            fmt.Sprintf("Failed to complete restore within %s.", timeOut),
		LastTransitionTime: metav1.Now(),
	})
}

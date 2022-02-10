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
	"strings"

	"stash.appscode.dev/apimachinery/apis"
	api_v1beta1 "stash.appscode.dev/apimachinery/apis/stash/v1beta1"
	"stash.appscode.dev/apimachinery/pkg/invoker"

	core "k8s.io/api/core/v1"
	kmapi "kmodules.xyz/client-go/api/v1"
)

func SetRestoreTargetFoundConditionToTrue(inv invoker.RestoreInvoker, index int) error {
	target := inv.GetTargetInfo()[index].Target
	return inv.SetCondition(&target.Ref, kmapi.Condition{
		Type:   apis.RestoreTargetFound,
		Status: core.ConditionTrue,
		Reason: apis.TargetAvailable,
		Message: fmt.Sprintf("Restore target %s %s/%s found.",
			target.Ref.APIVersion,
			strings.ToLower(target.Ref.Kind),
			target.Ref.Name,
		),
	})
}

func SetRestoreTargetFoundConditionToFalse(inv invoker.RestoreInvoker, index int) error {
	target := inv.GetTargetInfo()[index].Target
	return inv.SetCondition(&target.Ref, kmapi.Condition{
		Type:   apis.RestoreTargetFound,
		Status: core.ConditionFalse,
		Reason: apis.TargetNotAvailable,
		Message: fmt.Sprintf("Restore target %s %s/%s does not exist.",
			target.Ref.APIVersion,
			strings.ToLower(target.Ref.Kind),
			target.Ref.Name,
		),
	})
}

func SetRestoreTargetFoundConditionToUnknown(inv invoker.RestoreInvoker, index int, err error) error {
	target := inv.GetTargetInfo()[index].Target
	return inv.SetCondition(&target.Ref, kmapi.Condition{
		Type:   apis.RestoreTargetFound,
		Status: core.ConditionUnknown,
		Reason: apis.UnableToCheckTargetAvailability,
		Message: fmt.Sprintf("Failed to check whether restore target %s %s/%s exist or not. Reason: %v",
			target.Ref.APIVersion,
			strings.ToLower(target.Ref.Kind),
			target.Ref.Name,
			err,
		),
	})
}

func SetRestoreJobCreatedConditionToTrue(inv invoker.RestoreInvoker, tref *api_v1beta1.TargetRef) error {
	return inv.SetCondition(tref, kmapi.Condition{
		Type:    apis.RestoreJobCreated,
		Status:  core.ConditionTrue,
		Reason:  apis.RestoreJobCreationSucceeded,
		Message: "Successfully created restore job.",
	})
}

func SetRestoreJobCreatedConditionToFalse(inv invoker.RestoreInvoker, tref *api_v1beta1.TargetRef, err error) error {
	return inv.SetCondition(tref, kmapi.Condition{
		Type:    apis.RestoreJobCreated,
		Status:  core.ConditionFalse,
		Reason:  apis.RestoreJobCreationFailed,
		Message: fmt.Sprintf("Failed to create restore job. Reason: %v", err.Error()),
	})
}

func SetInitContainerInjectedConditionToTrue(inv invoker.RestoreInvoker, tref *api_v1beta1.TargetRef) error {
	return inv.SetCondition(tref, kmapi.Condition{
		Type:    apis.StashInitContainerInjected,
		Status:  core.ConditionTrue,
		Reason:  apis.InitContainerInjectionSucceeded,
		Message: "Successfully injected stash init-container.",
	})
}

func SetInitContainerInjectedConditionToFalse(inv invoker.RestoreInvoker, tref *api_v1beta1.TargetRef, err error) error {
	return inv.SetCondition(tref, kmapi.Condition{
		Type:    apis.StashInitContainerInjected,
		Status:  core.ConditionFalse,
		Reason:  apis.InitContainerInjectionFailed,
		Message: fmt.Sprintf("Failed to inject Stash init-container. Reason: %v", err.Error()),
	})
}

func SetRestoreCompletedConditionToTrue(inv invoker.RestoreInvoker, tref *api_v1beta1.TargetRef, msg string) error {
	return inv.SetCondition(tref, kmapi.Condition{
		Type:    apis.RestoreCompleted,
		Status:  core.ConditionTrue,
		Reason:  "PostRestoreTasksExecuted",
		Message: msg,
	})
}

func SetRestoreCompletedConditionToFalse(inv invoker.RestoreInvoker, tref *api_v1beta1.TargetRef, msg string) error {
	return inv.SetCondition(tref, kmapi.Condition{
		Type:    apis.RestoreCompleted,
		Status:  core.ConditionFalse,
		Reason:  "PostRestoreTasksNotExecuted",
		Message: msg,
	})
}

func SetRestorerEnsuredToTrue(inv invoker.RestoreInvoker, tref *api_v1beta1.TargetRef, msg string) error {
	return inv.SetCondition(tref, kmapi.Condition{
		Type:    apis.RestorerEnsured,
		Status:  core.ConditionTrue,
		Reason:  "SuccessfullyEnsuredRestorerEntity",
		Message: msg,
	})
}

func SetRestorerEnsuredToFalse(inv invoker.RestoreInvoker, tref *api_v1beta1.TargetRef, msg string) error {
	return inv.SetCondition(tref, kmapi.Condition{
		Type:    apis.RestorerEnsured,
		Status:  core.ConditionFalse,
		Reason:  "FailedToEnsureRestorerEntity",
		Message: msg,
	})
}

func SetMetricsPushedConditionToFalse(inv invoker.RestoreInvoker, tref *api_v1beta1.TargetRef, err error) error {
	return inv.SetCondition(tref, kmapi.Condition{
		Type:    apis.MetricsPushed,
		Status:  core.ConditionFalse,
		Reason:  apis.FailedToPushMetrics,
		Message: fmt.Sprintf("Failed to push metrics. Reason: %v", err.Error()),
	})
}

func SetMetricsPushedConditionToTrue(inv invoker.RestoreInvoker, tref *api_v1beta1.TargetRef) error {
	return inv.SetCondition(tref, kmapi.Condition{
		Type:    apis.MetricsPushed,
		Status:  core.ConditionTrue,
		Reason:  apis.SuccessfullyPushedMetrics,
		Message: "Successfully pushed metrics.",
	})
}

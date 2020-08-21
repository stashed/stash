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

	kmapi "kmodules.xyz/client-go/api/v1"
)

func SetRestoreTargetFoundConditionToTrue(invoker apis.RestoreInvoker, index int) error {
	target := invoker.TargetsInfo[index].Target
	return invoker.SetCondition(&target.Ref, kmapi.Condition{
		Type:   apis.RestoreTargetFound,
		Status: kmapi.ConditionTrue,
		Reason: apis.TargetAvailable,
		Message: fmt.Sprintf("Restore target %s %s/%s found.",
			target.Ref.APIVersion,
			strings.ToLower(target.Ref.Kind),
			target.Ref.Name,
		),
	})
}

func SetRestoreTargetFoundConditionToFalse(invoker apis.RestoreInvoker, index int) error {
	target := invoker.TargetsInfo[index].Target
	return invoker.SetCondition(&target.Ref, kmapi.Condition{
		Type:   apis.RestoreTargetFound,
		Status: kmapi.ConditionFalse,
		Reason: apis.TargetNotAvailable,
		Message: fmt.Sprintf("Restore target %s %s/%s does not exist.",
			target.Ref.APIVersion,
			strings.ToLower(target.Ref.Kind),
			target.Ref.Name,
		),
	})
}

func SetRestoreTargetFoundConditionToUnknown(invoker apis.RestoreInvoker, index int, err error) error {
	target := invoker.TargetsInfo[index].Target
	return invoker.SetCondition(&target.Ref, kmapi.Condition{
		Type:   apis.RestoreTargetFound,
		Status: kmapi.ConditionUnknown,
		Reason: apis.UnableToCheckTargetAvailability,
		Message: fmt.Sprintf("Failed to check whether restore target %s %s/%s exist or not. Reason: %v",
			target.Ref.APIVersion,
			strings.ToLower(target.Ref.Kind),
			target.Ref.Name,
			err,
		),
	})
}

func SetRestoreJobCreatedConditionToTrue(invoker apis.RestoreInvoker, tref *api_v1beta1.TargetRef) error {
	return invoker.SetCondition(tref, kmapi.Condition{
		Type:    apis.RestoreJobCreated,
		Status:  kmapi.ConditionTrue,
		Reason:  apis.RestoreJobCreationSucceeded,
		Message: "Successfully created restore job.",
	})
}

func SetRestoreJobCreatedConditionToFalse(invoker apis.RestoreInvoker, tref *api_v1beta1.TargetRef, err error) error {
	return invoker.SetCondition(tref, kmapi.Condition{
		Type:    apis.RestoreJobCreated,
		Status:  kmapi.ConditionFalse,
		Reason:  apis.RestoreJobCreationFailed,
		Message: fmt.Sprintf("Failed to create restore job. Reason: %v", err.Error()),
	})
}

func SetInitContainerInjectedConditionToTrue(invoker apis.RestoreInvoker, tref api_v1beta1.TargetRef) error {
	return invoker.SetCondition(&tref, kmapi.Condition{
		Type:    apis.StashInitContainerInjected,
		Status:  kmapi.ConditionTrue,
		Reason:  apis.InitContainerInjectionSucceeded,
		Message: "Successfully injected stash init-container.",
	})
}

func SetInitContainerInjectedConditionToFalse(invoker apis.RestoreInvoker, tref api_v1beta1.TargetRef, err error) error {
	return invoker.SetCondition(&tref, kmapi.Condition{
		Type:    apis.StashInitContainerInjected,
		Status:  kmapi.ConditionFalse,
		Reason:  apis.InitContainerInjectionFailed,
		Message: fmt.Sprintf("Failed to inject Stash init-container. Reason: %v", err.Error()),
	})
}

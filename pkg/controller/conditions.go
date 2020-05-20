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

package controller

import (
	"fmt"
	"strings"

	"stash.appscode.dev/apimachinery/apis"
	api_v1beta1 "stash.appscode.dev/apimachinery/apis/stash/v1beta1"
	v1beta1_util "stash.appscode.dev/apimachinery/client/clientset/versioned/typed/stash/v1beta1/util"

	kmapi "kmodules.xyz/client-go/api/v1"
)

func (c *StashController) setBackupTargetFoundConditionToUnknown(invoker apis.Invoker, tref api_v1beta1.TargetRef, err error) error {
	return invoker.SetCondition(&tref, kmapi.Condition{
		Type:   string(api_v1beta1.BackupTargetFound),
		Status: kmapi.ConditionUnknown,
		Reason: apis.UnableToCheckTargetAvailability,
		Message: fmt.Sprintf("Failed to check whether backup target %s %s/%s exist or not. Reason: %v",
			tref.APIVersion,
			strings.ToLower(tref.Kind),
			tref.Name,
			err.Error(),
		),
	})
}

func (c *StashController) setBackupTargetFoundConditionToFalse(invoker apis.Invoker, tref api_v1beta1.TargetRef) error {
	return invoker.SetCondition(&tref, kmapi.Condition{
		// Set the "BackupTargetFound" condition to "False"
		Type:   string(api_v1beta1.BackupTargetFound),
		Status: kmapi.ConditionFalse,
		Reason: apis.TargetNotAvailable,
		Message: fmt.Sprintf("Backup target %s %s/%s does not exist.",
			tref.APIVersion,
			strings.ToLower(tref.Kind),
			tref.Name,
		),
	})
}

func (c *StashController) setBackupTargetFoundConditionToTrue(invoker apis.Invoker, tref api_v1beta1.TargetRef) error {
	return invoker.SetCondition(&tref, kmapi.Condition{
		Type:   string(api_v1beta1.BackupTargetFound),
		Status: kmapi.ConditionTrue,
		Reason: apis.TargetAvailable,
		Message: fmt.Sprintf("Backup target %s %s/%s found.",
			tref.APIVersion,
			strings.ToLower(tref.Kind),
			tref.Name,
		),
	})
}

func (c *StashController) setCronJobCreatedConditionToFalse(invoker apis.Invoker, err error) error {
	return invoker.SetCondition(nil, kmapi.Condition{
		Type:    string(api_v1beta1.CronJobCreated),
		Status:  kmapi.ConditionFalse,
		Reason:  apis.CronJobCreationFailed,
		Message: fmt.Sprintf("Failed to create backup triggering CronJob. Reason: %v", err.Error()),
	})
}

func (c *StashController) setCronJobCreatedConditionToTrue(invoker apis.Invoker) error {
	return invoker.SetCondition(nil, kmapi.Condition{
		Type:    string(api_v1beta1.CronJobCreated),
		Status:  kmapi.ConditionTrue,
		Reason:  apis.CronJobCreationSucceeded,
		Message: "Successfully created backup triggering CronJob.",
	})
}

func (c *StashController) setRestoreTargetFoundConditionToUnknown(rs *api_v1beta1.RestoreSession, err error) error {
	return c.setRestoreSessionCondition(rs, kmapi.Condition{
		Type:   string(api_v1beta1.RestoreTargetFound),
		Status: kmapi.ConditionUnknown,
		Reason: apis.UnableToCheckTargetAvailability,
		Message: fmt.Sprintf("Failed to check whether restore target %s %s/%s exist or not. Reason: %v",
			rs.Spec.Target.Ref.APIVersion,
			strings.ToLower(rs.Spec.Target.Ref.Kind),
			rs.Spec.Target.Ref.Name,
			err.Error(),
		),
	})
}
func (c *StashController) setRestoreTargetFoundConditionToFalse(rs *api_v1beta1.RestoreSession) error {
	return c.setRestoreSessionCondition(rs, kmapi.Condition{
		Type:   string(api_v1beta1.RestoreTargetFound),
		Status: kmapi.ConditionFalse,
		Reason: apis.TargetNotAvailable,
		Message: fmt.Sprintf("Restore target %s %s/%s does not exist.",
			rs.Spec.Target.Ref.APIVersion,
			strings.ToLower(rs.Spec.Target.Ref.Kind),
			rs.Spec.Target.Ref.Name,
		),
	})
}

func (c *StashController) setRestoreTargetFoundConditionToTrue(rs *api_v1beta1.RestoreSession) error {
	return c.setRestoreSessionCondition(rs, kmapi.Condition{
		Type:   string(api_v1beta1.RestoreTargetFound),
		Status: kmapi.ConditionTrue,
		Reason: apis.TargetAvailable,
		Message: fmt.Sprintf("Restore target %s %s/%s found.",
			rs.Spec.Target.Ref.APIVersion,
			strings.ToLower(rs.Spec.Target.Ref.Kind),
			rs.Spec.Target.Ref.Name,
		),
	})
}

func (c *StashController) setRestoreJobCreatedConditionToTrue(rs *api_v1beta1.RestoreSession) error {
	return c.setRestoreSessionCondition(rs, kmapi.Condition{
		Type:    string(api_v1beta1.RestoreJobCreated),
		Status:  kmapi.ConditionTrue,
		Reason:  apis.RestoreJobCreationSucceeded,
		Message: "Successfully created restore job.",
	})
}

func (c *StashController) setRestoreJobCreatedConditionToFalse(rs *api_v1beta1.RestoreSession, err error) error {
	return c.setRestoreSessionCondition(rs, kmapi.Condition{
		Type:    string(api_v1beta1.RestoreJobCreated),
		Status:  kmapi.ConditionFalse,
		Reason:  apis.RestoreJobCreationFailed,
		Message: fmt.Sprintf("Failed to create restore job. Reason: %v", err.Error()),
	})
}

func (c *StashController) setRestoreSessionCondition(rs *api_v1beta1.RestoreSession, condition kmapi.Condition) error {
	_, err := v1beta1_util.UpdateRestoreSessionStatus(c.stashClient.StashV1beta1(), rs.ObjectMeta, func(in *api_v1beta1.RestoreSessionStatus) *api_v1beta1.RestoreSessionStatus {
		in.Conditions = kmapi.SetCondition(in.Conditions, condition)
		return in
	})
	return err
}

func (c *StashController) setSidecarInjectedConditionToTrue(invoker apis.Invoker, tref api_v1beta1.TargetRef) error {
	return invoker.SetCondition(&tref, kmapi.Condition{
		Type:   string(api_v1beta1.StashSidecarInjected),
		Status: kmapi.ConditionTrue,
		Reason: apis.SidecarInjectionSucceeded,
		Message: fmt.Sprintf("Successfully injected stash sidecar into %s %s/%s",
			tref.APIVersion,
			strings.ToLower(tref.Kind),
			tref.Name,
		),
	})
}

func (c *StashController) setSidecarInjectedConditionToFalse(invoker apis.Invoker, tref api_v1beta1.TargetRef, err error) error {
	return invoker.SetCondition(&tref, kmapi.Condition{
		Type:   string(api_v1beta1.StashSidecarInjected),
		Status: kmapi.ConditionFalse,
		Reason: apis.SidecarInjectionFailed,
		Message: fmt.Sprintf("Failed to inject stash sidecar into %s %s/%s. Reason: %v",
			tref.APIVersion,
			strings.ToLower(tref.Kind),
			tref.Name,
			err.Error(),
		),
	})
}

func (c *StashController) setInitContainerInjectedConditionToTrue(rs *api_v1beta1.RestoreSession) error {
	return c.setRestoreSessionCondition(rs, kmapi.Condition{
		Type:    string(api_v1beta1.StashInitContainerInjected),
		Status:  kmapi.ConditionTrue,
		Reason:  apis.InitContainerInjectionSucceeded,
		Message: "Successfully injected stash init-container.",
	})
}
func (c *StashController) setInitContainerInjectedConditionToFalse(rs *api_v1beta1.RestoreSession, err error) error {
	return c.setRestoreSessionCondition(rs, kmapi.Condition{
		Type:    string(api_v1beta1.StashInitContainerInjected),
		Status:  kmapi.ConditionFalse,
		Reason:  apis.InitContainerInjectionFailed,
		Message: fmt.Sprintf("Failed to inject Stash init-container. Reason: %v", err.Error()),
	})
}

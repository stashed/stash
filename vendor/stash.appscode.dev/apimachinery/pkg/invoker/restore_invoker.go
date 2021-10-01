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

package invoker

import (
	"context"
	"fmt"
	"strings"
	"time"

	"stash.appscode.dev/apimachinery/apis/stash/v1beta1"
	cs "stash.appscode.dev/apimachinery/client/clientset/versioned"
	stash_scheme "stash.appscode.dev/apimachinery/client/clientset/versioned/scheme"
	v1beta1_util "stash.appscode.dev/apimachinery/client/clientset/versioned/typed/stash/v1beta1/util"

	core "k8s.io/api/core/v1"
	kerr "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/reference"
	kmapi "kmodules.xyz/client-go/api/v1"
	core_util "kmodules.xyz/client-go/core/v1"
	"kmodules.xyz/client-go/meta"
	appcat "kmodules.xyz/custom-resources/apis/appcatalog/v1alpha1"
	appcatalog_cs "kmodules.xyz/custom-resources/client/clientset/versioned"
	ofst "kmodules.xyz/offshoot-api/api/v1"
)

const (
	EventSourceRestoreBatchController   = "RestoreBatch Controller"
	EventSourceRestoreSessionController = "RestoreSession Controller"
)

type RestoreTargetInfo struct {
	Task                  v1beta1.TaskRef
	Target                *v1beta1.RestoreTarget
	RuntimeSettings       ofst.RuntimeSettings
	TempDir               v1beta1.EmptyDirSettings
	InterimVolumeTemplate *ofst.PersistentVolumeClaim
	Hooks                 *v1beta1.RestoreHooks
}

type RestoreInvokerStatus struct {
	Phase           v1beta1.RestorePhase
	SessionDuration string
	Conditions      []kmapi.Condition
	TargetStatus    []v1beta1.RestoreMemberStatus
}

type RestoreInvoker struct {
	TypeMeta                metav1.TypeMeta
	ObjectMeta              metav1.ObjectMeta
	Labels                  map[string]string
	Hash                    string
	Driver                  v1beta1.Snapshotter
	Repository              string
	TargetsInfo             []RestoreTargetInfo
	ExecutionOrder          v1beta1.ExecutionOrder
	Hooks                   *v1beta1.RestoreHooks
	ObjectRef               *core.ObjectReference
	OwnerRef                *metav1.OwnerReference
	Status                  RestoreInvokerStatus
	ObjectJson              []byte
	AddFinalizer            func() error
	RemoveFinalizer         func() error
	HasCondition            func(*v1beta1.TargetRef, string) (bool, error)
	GetCondition            func(*v1beta1.TargetRef, string) (int, *kmapi.Condition, error)
	SetCondition            func(*v1beta1.TargetRef, kmapi.Condition) error
	IsConditionTrue         func(*v1beta1.TargetRef, string) (bool, error)
	NextInOrder             func(v1beta1.TargetRef, []v1beta1.RestoreMemberStatus) bool
	EnsureKubeDBIntegration func(appClient appcatalog_cs.Interface) (map[string]string, error)

	UpdateRestoreInvokerStatus func(status RestoreInvokerStatus) (RestoreInvokerStatus, error)
	CreateEvent                func(eventType, source, reason, message string) error
}

func ExtractRestoreInvokerInfo(kubeClient kubernetes.Interface, stashClient cs.Interface, invokerType, invokerName, namespace string) (RestoreInvoker, error) {
	var invoker RestoreInvoker
	switch invokerType {
	case v1beta1.ResourceKindRestoreBatch:
		// get RestoreBatch
		restoreBatch, err := stashClient.StashV1beta1().RestoreBatches(namespace).Get(context.TODO(), invokerName, metav1.GetOptions{})
		if err != nil {
			return invoker, err
		}
		invoker.TypeMeta = metav1.TypeMeta{
			Kind:       v1beta1.ResourceKindRestoreBatch,
			APIVersion: v1beta1.SchemeGroupVersion.String(),
		}
		invoker.ObjectMeta = restoreBatch.ObjectMeta
		invoker.Labels = restoreBatch.OffshootLabels()
		invoker.Hash = restoreBatch.GetSpecHash()
		invoker.Driver = restoreBatch.Spec.Driver
		invoker.Repository = restoreBatch.Spec.Repository.Name
		invoker.Hooks = restoreBatch.Spec.Hooks
		invoker.ExecutionOrder = restoreBatch.Spec.ExecutionOrder
		invoker.OwnerRef = metav1.NewControllerRef(restoreBatch, v1beta1.SchemeGroupVersion.WithKind(v1beta1.ResourceKindRestoreBatch))
		invoker.ObjectRef, err = reference.GetReference(stash_scheme.Scheme, restoreBatch)
		if err != nil {
			return invoker, err
		}

		invoker.ObjectJson, err = meta.MarshalToJson(restoreBatch, v1beta1.SchemeGroupVersion)
		if err != nil {
			return invoker, err
		}

		for _, member := range restoreBatch.Spec.Members {
			invoker.TargetsInfo = append(invoker.TargetsInfo, RestoreTargetInfo{
				Task:                  member.Task,
				Target:                member.Target,
				RuntimeSettings:       member.RuntimeSettings,
				TempDir:               member.TempDir,
				InterimVolumeTemplate: member.InterimVolumeTemplate,
				Hooks:                 member.Hooks,
			})
		}

		invoker.Status = getInvokerStatusFromRestoreBatch(restoreBatch)

		invoker.AddFinalizer = func() error {
			_, _, err := v1beta1_util.PatchRestoreBatch(context.TODO(), stashClient.StashV1beta1(), restoreBatch, func(in *v1beta1.RestoreBatch) *v1beta1.RestoreBatch {
				in.ObjectMeta = core_util.AddFinalizer(in.ObjectMeta, v1beta1.StashKey)
				return in
			}, metav1.PatchOptions{})
			return err
		}
		invoker.RemoveFinalizer = func() error {
			_, _, err := v1beta1_util.PatchRestoreBatch(context.TODO(), stashClient.StashV1beta1(), restoreBatch, func(in *v1beta1.RestoreBatch) *v1beta1.RestoreBatch {
				in.ObjectMeta = core_util.RemoveFinalizer(in.ObjectMeta, v1beta1.StashKey)
				return in
			}, metav1.PatchOptions{})
			return err
		}
		invoker.HasCondition = func(target *v1beta1.TargetRef, condType string) (bool, error) {
			restoreBatch, err := stashClient.StashV1beta1().RestoreBatches(namespace).Get(context.TODO(), invokerName, metav1.GetOptions{})
			if err != nil {
				return false, err
			}
			if target != nil {
				return hasRestoreMemberCondition(restoreBatch.Status.Members, *target, condType), nil
			}
			return kmapi.HasCondition(restoreBatch.Status.Conditions, condType), nil
		}
		invoker.GetCondition = func(target *v1beta1.TargetRef, condType string) (int, *kmapi.Condition, error) {
			restoreBatch, err := stashClient.StashV1beta1().RestoreBatches(namespace).Get(context.TODO(), invokerName, metav1.GetOptions{})
			if err != nil {
				return -1, nil, err
			}
			if target != nil {
				idx, cond := getRestoreMemberCondition(restoreBatch.Status.Members, *target, condType)
				return idx, cond, nil
			}
			idx, cond := kmapi.GetCondition(restoreBatch.Status.Conditions, condType)
			return idx, cond, nil

		}
		invoker.SetCondition = func(target *v1beta1.TargetRef, condition kmapi.Condition) error {
			_, err = v1beta1_util.UpdateRestoreBatchStatus(context.TODO(), stashClient.StashV1beta1(), restoreBatch.ObjectMeta, func(in *v1beta1.RestoreBatchStatus) (types.UID, *v1beta1.RestoreBatchStatus) {
				if target != nil {
					in.Members = setRestoreMemberCondition(in.Members, *target, condition)
				} else {
					in.Conditions = kmapi.SetCondition(in.Conditions, condition)
				}
				return restoreBatch.UID, in
			}, metav1.UpdateOptions{})
			return err
		}
		invoker.IsConditionTrue = func(target *v1beta1.TargetRef, condType string) (bool, error) {
			restoreBatch, err := stashClient.StashV1beta1().RestoreBatches(namespace).Get(context.TODO(), invokerName, metav1.GetOptions{})
			if err != nil {
				return false, err
			}
			if target != nil {
				return isRestoreMemberConditionTrue(restoreBatch.Status.Members, *target, condType), nil
			}
			return kmapi.IsConditionTrue(restoreBatch.Status.Conditions, condType), nil
		}

		invoker.NextInOrder = func(ref v1beta1.TargetRef, targetStatus []v1beta1.RestoreMemberStatus) bool {
			for _, t := range invoker.TargetsInfo {
				if t.Target != nil {
					if TargetMatched(t.Target.Ref, ref) {
						return true
					}
					if !TargetRestoreCompleted(t.Target.Ref, targetStatus) {
						return false
					}
				}
			}
			// By default, return true so that nil target(i.e. cluster backup) does not get stuck here.
			return true
		}

		invoker.UpdateRestoreInvokerStatus = func(status RestoreInvokerStatus) (RestoreInvokerStatus, error) {
			updatedRestoreBatch, err := v1beta1_util.UpdateRestoreBatchStatus(
				context.TODO(),
				stashClient.StashV1beta1(),
				invoker.ObjectMeta,
				func(in *v1beta1.RestoreBatchStatus) (types.UID, *v1beta1.RestoreBatchStatus) {
					if status.Phase != "" {
						in.Phase = status.Phase
					}
					if status.SessionDuration != "" {
						in.SessionDuration = status.SessionDuration
					}
					if len(status.Conditions) > 0 {
						in.Conditions = upsertConditions(in.Conditions, status.Conditions)
					}
					if len(status.TargetStatus) > 0 {
						for i := range status.TargetStatus {
							in.Members = upsertRestoreMemberStatus(in.Members, status.TargetStatus[i])
						}
					}
					return invoker.ObjectMeta.UID, in
				},
				metav1.UpdateOptions{},
			)
			if err != nil {
				return RestoreInvokerStatus{}, err
			}
			return getInvokerStatusFromRestoreBatch(updatedRestoreBatch), nil
		}
		invoker.CreateEvent = func(eventType, source, reason, message string) error {
			t := metav1.Time{Time: time.Now()}

			if source == "" {
				source = EventSourceRestoreBatchController
			}
			_, err := kubeClient.CoreV1().Events(invoker.ObjectMeta.Namespace).Create(context.TODO(), &core.Event{
				ObjectMeta: metav1.ObjectMeta{
					Name:      fmt.Sprintf("%v.%x", invoker.ObjectRef.Name, t.UnixNano()),
					Namespace: invoker.ObjectRef.Namespace,
				},
				InvolvedObject: *invoker.ObjectRef,
				Reason:         reason,
				Message:        message,
				FirstTimestamp: t,
				LastTimestamp:  t,
				Count:          1,
				Type:           eventType,
				Source:         core.EventSource{Component: source},
			}, metav1.CreateOptions{})
			return err
		}
		invoker.EnsureKubeDBIntegration = func(appClient appcatalog_cs.Interface) (map[string]string, error) {
			for i := range restoreBatch.Spec.Members {
				target := restoreBatch.Spec.Members[i].Target
				// Don't do anything if the target is not an AppBinding
				if target == nil || !TargetOfGroupKind(target.Ref, appcat.SchemeGroupVersion.Group, appcat.ResourceKindApp) {
					continue
				}
				// Get the respective AppBinding
				appBinding, err := appClient.AppcatalogV1alpha1().AppBindings(restoreBatch.Namespace).Get(context.TODO(), target.Ref.Name, metav1.GetOptions{})
				if err != nil {
					// If the AppBinding does not exist, then don't do anything.
					if kerr.IsNotFound(err) {
						continue
					}
					return nil, err
				}
				// If the AppBinding is not managed by KubeDB, then don't do anything
				if manager, err := meta.GetStringValue(appBinding.Labels, meta.ManagedByLabelKey); err != nil || manager != "kubedb.com" {
					continue
				}
				// Extract the name, and managed-by labels. We are not passing "instance" label because there could be multiple AppBindings.
				appLabels, err := extractLabels(appBinding.Labels, meta.ManagedByLabelKey, meta.NameLabelKey)
				if err != nil {
					return nil, err
				}

				// Add the labels to the invoker
				restoreBatch, _, err := v1beta1_util.PatchRestoreBatch(context.TODO(), stashClient.StashV1beta1(), restoreBatch, func(in *v1beta1.RestoreBatch) *v1beta1.RestoreBatch {
					in.Labels = meta.OverwriteKeys(in.Labels, appLabels)
					return in
				}, metav1.PatchOptions{})
				if err != nil {
					return nil, err
				}
				return restoreBatch.Labels, nil
			}
			return nil, nil
		}
	case v1beta1.ResourceKindRestoreSession:
		// get RestoreSession
		restoreSession, err := stashClient.StashV1beta1().RestoreSessions(namespace).Get(context.TODO(), invokerName, metav1.GetOptions{})
		if err != nil {
			return invoker, err
		}
		invoker.TypeMeta = metav1.TypeMeta{
			Kind:       v1beta1.ResourceKindRestoreSession,
			APIVersion: v1beta1.SchemeGroupVersion.String(),
		}
		invoker.ObjectMeta = restoreSession.ObjectMeta
		invoker.Labels = restoreSession.OffshootLabels()
		invoker.Hash = restoreSession.GetSpecHash()
		invoker.Driver = restoreSession.Spec.Driver
		invoker.Repository = restoreSession.Spec.Repository.Name
		invoker.OwnerRef = metav1.NewControllerRef(restoreSession, v1beta1.SchemeGroupVersion.WithKind(v1beta1.ResourceKindRestoreSession))
		invoker.ObjectRef, err = reference.GetReference(stash_scheme.Scheme, restoreSession)
		if err != nil {
			return invoker, err
		}

		invoker.ObjectJson, err = meta.MarshalToJson(restoreSession, v1beta1.SchemeGroupVersion)
		if err != nil {
			return invoker, err
		}

		invoker.TargetsInfo = append(invoker.TargetsInfo, RestoreTargetInfo{
			Task:                  restoreSession.Spec.Task,
			Target:                restoreSession.Spec.Target,
			RuntimeSettings:       restoreSession.Spec.RuntimeSettings,
			TempDir:               restoreSession.Spec.TempDir,
			InterimVolumeTemplate: restoreSession.Spec.InterimVolumeTemplate,
			Hooks:                 restoreSession.Spec.Hooks,
		})

		invoker.Status = getInvokerStatusFromRestoreSession(restoreSession)

		invoker.AddFinalizer = func() error {
			_, _, err := v1beta1_util.PatchRestoreSession(context.TODO(), stashClient.StashV1beta1(), restoreSession, func(in *v1beta1.RestoreSession) *v1beta1.RestoreSession {
				in.ObjectMeta = core_util.AddFinalizer(in.ObjectMeta, v1beta1.StashKey)
				return in
			}, metav1.PatchOptions{})
			return err
		}
		invoker.RemoveFinalizer = func() error {
			_, _, err := v1beta1_util.PatchRestoreSession(context.TODO(), stashClient.StashV1beta1(), restoreSession, func(in *v1beta1.RestoreSession) *v1beta1.RestoreSession {
				in.ObjectMeta = core_util.RemoveFinalizer(in.ObjectMeta, v1beta1.StashKey)
				return in
			}, metav1.PatchOptions{})
			return err
		}
		invoker.HasCondition = func(target *v1beta1.TargetRef, condType string) (bool, error) {
			restoreSession, err := stashClient.StashV1beta1().RestoreSessions(namespace).Get(context.TODO(), invokerName, metav1.GetOptions{})
			if err != nil {
				return false, err
			}
			return kmapi.HasCondition(restoreSession.Status.Conditions, condType), nil
		}
		invoker.GetCondition = func(target *v1beta1.TargetRef, condType string) (int, *kmapi.Condition, error) {
			restoreSession, err := stashClient.StashV1beta1().RestoreSessions(namespace).Get(context.TODO(), invokerName, metav1.GetOptions{})
			if err != nil {
				return -1, nil, err
			}
			idx, cond := kmapi.GetCondition(restoreSession.Status.Conditions, condType)
			return idx, cond, nil
		}
		invoker.SetCondition = func(target *v1beta1.TargetRef, condition kmapi.Condition) error {
			_, err = v1beta1_util.UpdateRestoreSessionStatus(context.TODO(), stashClient.StashV1beta1(), restoreSession.ObjectMeta, func(in *v1beta1.RestoreSessionStatus) (types.UID, *v1beta1.RestoreSessionStatus) {
				in.Conditions = kmapi.SetCondition(in.Conditions, condition)
				return restoreSession.UID, in
			}, metav1.UpdateOptions{})
			return err
		}
		invoker.IsConditionTrue = func(target *v1beta1.TargetRef, condType string) (bool, error) {
			restoreSession, err := stashClient.StashV1beta1().RestoreSessions(namespace).Get(context.TODO(), invokerName, metav1.GetOptions{})
			if err != nil {
				return false, err
			}
			return kmapi.IsConditionTrue(restoreSession.Status.Conditions, condType), nil
		}

		invoker.NextInOrder = func(ref v1beta1.TargetRef, targetStatus []v1beta1.RestoreMemberStatus) bool {
			for _, t := range invoker.TargetsInfo {
				if t.Target != nil {
					if TargetMatched(t.Target.Ref, ref) {
						return true
					}
					if !TargetRestoreCompleted(t.Target.Ref, targetStatus) {
						return false
					}
				}
			}
			// By default, return true so that nil target(i.e. cluster backup) does not get stuck here.
			return true
		}

		invoker.UpdateRestoreInvokerStatus = func(status RestoreInvokerStatus) (RestoreInvokerStatus, error) {
			updatedRestoreSession, err := v1beta1_util.UpdateRestoreSessionStatus(
				context.TODO(),
				stashClient.StashV1beta1(),
				invoker.ObjectMeta,
				func(in *v1beta1.RestoreSessionStatus) (types.UID, *v1beta1.RestoreSessionStatus) {
					if status.Phase != "" {
						in.Phase = status.Phase
					}
					if status.SessionDuration != "" {
						in.SessionDuration = status.SessionDuration
					}
					if status.Conditions != nil {
						in.Conditions = upsertConditions(in.Conditions, status.Conditions)
					}
					if status.TargetStatus != nil {
						targetStatus := status.TargetStatus[0]
						if targetStatus.TotalHosts != nil {
							in.TotalHosts = targetStatus.TotalHosts
						}
						if targetStatus.Conditions != nil {
							in.Conditions = upsertConditions(in.Conditions, targetStatus.Conditions)
						}
						if targetStatus.Stats != nil {
							in.Stats = upsertRestoreHostStatus(in.Stats, targetStatus.Stats)
						}
						if targetStatus.Phase != "" {
							in.Phase = v1beta1.RestorePhase(targetStatus.Phase)
						}
					}
					return invoker.ObjectMeta.UID, in
				},
				metav1.UpdateOptions{},
			)
			if err != nil {
				return RestoreInvokerStatus{}, err
			}
			return getInvokerStatusFromRestoreSession(updatedRestoreSession), nil
		}
		invoker.CreateEvent = func(eventType, source, reason, message string) error {
			t := metav1.Time{Time: time.Now()}
			if source == "" {
				source = EventSourceRestoreSessionController
			}
			_, err := kubeClient.CoreV1().Events(invoker.ObjectMeta.Namespace).Create(context.TODO(), &core.Event{
				ObjectMeta: metav1.ObjectMeta{
					Name:      fmt.Sprintf("%v.%x", invoker.ObjectRef.Name, t.UnixNano()),
					Namespace: invoker.ObjectRef.Namespace,
				},
				InvolvedObject: *invoker.ObjectRef,
				Reason:         reason,
				Message:        message,
				FirstTimestamp: t,
				LastTimestamp:  t,
				Count:          1,
				Type:           eventType,
				Source:         core.EventSource{Component: source},
			}, metav1.CreateOptions{})
			return err
		}
		invoker.EnsureKubeDBIntegration = func(appClient appcatalog_cs.Interface) (map[string]string, error) {
			// Don't do anything if the target is not an AppBinding
			if restoreSession.Spec.Target == nil || !TargetOfGroupKind(restoreSession.Spec.Target.Ref, appcat.SchemeGroupVersion.Group, appcat.ResourceKindApp) {
				return nil, nil
			}
			// Get the AppBinding
			appBinding, err := appClient.AppcatalogV1alpha1().AppBindings(restoreSession.Namespace).Get(context.TODO(), restoreSession.Spec.Target.Ref.Name, metav1.GetOptions{})
			if err != nil {
				// If the AppBinding does not exist, then don't do anything.
				if kerr.IsNotFound(err) {
					return nil, nil
				}
				return nil, err
			}
			// If the AppBinding is not managed by KubeDB, then don't do anything
			if manager, err := meta.GetStringValue(appBinding.Labels, meta.ManagedByLabelKey); err != nil || manager != "kubedb.com" {
				return nil, nil
			}
			// Extract the name, instance, and managed-by labels.
			appLabels, err := extractLabels(appBinding.Labels, meta.InstanceLabelKey, meta.ManagedByLabelKey, meta.NameLabelKey)
			if err != nil {
				return nil, err
			}

			// Add the labels to the invoker
			restoreSession, _, err = v1beta1_util.PatchRestoreSession(context.TODO(), stashClient.StashV1beta1(), restoreSession, func(in *v1beta1.RestoreSession) *v1beta1.RestoreSession {
				in.Labels = meta.OverwriteKeys(in.Labels, appLabels)
				return in
			}, metav1.PatchOptions{})
			if err != nil {
				return nil, err
			}
			return restoreSession.Labels, nil
		}
	default:
		return invoker, fmt.Errorf("failed to extract invoker info. Reason: unknown invoker")
	}
	return invoker, nil
}

func hasRestoreMemberCondition(status []v1beta1.RestoreMemberStatus, target v1beta1.TargetRef, condType string) bool {
	// If the target is present in the list, then return the respective value
	for i := range status {
		if TargetMatched(status[i].Ref, target) {
			return kmapi.HasCondition(status[i].Conditions, condType)
		}
	}
	// Member is not present in the list, so the condition is not there too
	return false
}

func getRestoreMemberCondition(status []v1beta1.RestoreMemberStatus, target v1beta1.TargetRef, condType string) (int, *kmapi.Condition) {
	// If the target is present in the list, then return the respective condition
	for i := range status {
		if TargetMatched(status[i].Ref, target) {
			return kmapi.GetCondition(status[i].Conditions, condType)
		}
	}
	// Member is not present in the list
	return -1, nil
}

func setRestoreMemberCondition(status []v1beta1.RestoreMemberStatus, target v1beta1.TargetRef, newCondition kmapi.Condition) []v1beta1.RestoreMemberStatus {
	// If the target is already exist in the list, update its condition
	for i := range status {
		if TargetMatched(status[i].Ref, target) {
			status[i].Conditions = kmapi.SetCondition(status[i].Conditions, newCondition)
			return status
		}
	}
	// The target does not exist in the list. So, add a new entry.
	memberStatus := v1beta1.RestoreMemberStatus{
		Ref:        target,
		Conditions: kmapi.SetCondition(nil, newCondition),
	}
	return upsertRestoreMemberStatus(status, memberStatus)
}

func isRestoreMemberConditionTrue(status []v1beta1.RestoreMemberStatus, target v1beta1.TargetRef, condType string) bool {
	// If the target is present in the list, then return the respective value
	for i := range status {
		if TargetMatched(status[i].Ref, target) {
			return kmapi.IsConditionTrue(status[i].Conditions, condType)
		}
	}
	// Member is not present in the list, so the condition is false
	return false
}

func upsertRestoreMemberStatus(cur []v1beta1.RestoreMemberStatus, new v1beta1.RestoreMemberStatus) []v1beta1.RestoreMemberStatus {
	// if the member status already exist, then update it
	for i := range cur {
		if TargetMatched(cur[i].Ref, new.Ref) {
			if new.Phase != "" {
				cur[i].Phase = new.Phase
			}
			if len(new.Conditions) > 0 {
				cur[i].Conditions = upsertConditions(cur[i].Conditions, new.Conditions)
			}
			if new.TotalHosts != nil {
				cur[i].TotalHosts = new.TotalHosts
			}
			if len(new.Stats) > 0 {
				cur[i].Stats = upsertRestoreHostStatus(cur[i].Stats, new.Stats)
			}
			return cur
		}
	}
	// the member status does not exist. so, add new entry.
	cur = append(cur, new)
	return cur
}

func upsertConditions(cur []kmapi.Condition, new []kmapi.Condition) []kmapi.Condition {
	for i := range new {
		cur = kmapi.SetCondition(cur, new[i])
	}
	return cur
}

func upsertRestoreHostStatus(cur []v1beta1.HostRestoreStats, new []v1beta1.HostRestoreStats) []v1beta1.HostRestoreStats {
	for i := range new {
		index, hostEntryExist := hostEntryIndex(cur, new[i])
		if hostEntryExist {
			cur[index] = new[i]
		} else {
			cur = append(cur, new[i])
		}
	}
	return cur
}

func hostEntryIndex(entries []v1beta1.HostRestoreStats, target v1beta1.HostRestoreStats) (int, bool) {
	for i := range entries {
		if entries[i].Hostname == target.Hostname {
			return i, true
		}
	}
	return -1, false
}

func getInvokerStatusFromRestoreBatch(restoreBatch *v1beta1.RestoreBatch) RestoreInvokerStatus {
	return RestoreInvokerStatus{
		Phase:           restoreBatch.Status.Phase,
		SessionDuration: restoreBatch.Status.SessionDuration,
		Conditions:      restoreBatch.Status.Conditions,
		TargetStatus:    restoreBatch.Status.Members,
	}
}

func getInvokerStatusFromRestoreSession(restoreSession *v1beta1.RestoreSession) RestoreInvokerStatus {
	invokerStatus := RestoreInvokerStatus{
		Phase:           restoreSession.Status.Phase,
		SessionDuration: restoreSession.Status.SessionDuration,
		Conditions:      restoreSession.Status.Conditions,
	}
	if restoreSession.Spec.Target != nil {
		invokerStatus.TargetStatus = append(invokerStatus.TargetStatus, v1beta1.RestoreMemberStatus{
			Ref:        restoreSession.Spec.Target.Ref,
			Conditions: restoreSession.Status.Conditions,
			TotalHosts: restoreSession.Status.TotalHosts,
			Phase:      v1beta1.RestoreTargetPhase(restoreSession.Status.Phase),
			Stats:      restoreSession.Status.Stats,
		})
	}
	return invokerStatus
}

func TargetRestoreCompleted(ref v1beta1.TargetRef, targetStatus []v1beta1.RestoreMemberStatus) bool {
	for i := range targetStatus {
		if TargetMatched(ref, targetStatus[i].Ref) {
			return targetStatus[i].Phase == v1beta1.TargetRestoreSucceeded ||
				targetStatus[i].Phase == v1beta1.TargetRestoreFailed ||
				targetStatus[i].Phase == v1beta1.TargetRestorePhaseUnknown
		}
	}
	return false
}

func extractLabels(in map[string]string, keys ...string) (map[string]string, error) {
	out := make(map[string]string, len(keys))
	for _, k := range keys {
		val, err := meta.GetStringValue(in, k)
		if err != nil {
			return nil, err
		}
		out[k] = val
	}
	return out, nil
}

func TargetOfGroupKind(targetRef v1beta1.TargetRef, group, kind string) bool {
	gv := strings.Split(targetRef.APIVersion, "/")
	if len(gv) > 0 && gv[0] == group && targetRef.Kind == kind {
		return true
	}
	return false
}

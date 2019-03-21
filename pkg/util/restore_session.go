package util

import (
	"fmt"
	"reflect"

	v1beta1_api "github.com/appscode/stash/apis/stash/v1beta1"
	v1beta1_listers "github.com/appscode/stash/client/listers/stash/v1beta1"
	kerr "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/labels"
	"kmodules.xyz/client-go/meta"
	wapi "kmodules.xyz/webhook-runtime/apis/workload/v1"
)

// GetAppliedRestoreSession check weather RestoreSession was applied as annotation and returns the object definition if exist.
func GetAppliedRestoreSession(m map[string]string) (*v1beta1_api.RestoreSession, error) {
	data := GetString(m, v1beta1_api.KeyLastAppliedRestoreSession)
	if data == "" {
		return nil, nil
	}

	obj, err := meta.UnmarshalFromJSON([]byte(data), v1beta1_api.SchemeGroupVersion)
	if err != nil {
		return nil, err
	}
	restoreSession, ok := obj.(*v1beta1_api.RestoreSession)
	if !ok {
		return nil, fmt.Errorf("%s annotations has invalid RestoreSession object", v1beta1_api.KeyLastAppliedRestoreSession)
	}
	return restoreSession, nil
}

// FindRestoreSession check if there is any pending RestoreSession exist for a particular workload.
// If multiple pending RestoreSessions are found for the workload, it returns error.
func FindRestoreSession(lister v1beta1_listers.RestoreSessionLister, w *wapi.Workload) (*v1beta1_api.RestoreSession, error) {
	// list all RestoreSessions from the lister
	restoreSessions, err := lister.RestoreSessions(w.Namespace).List(labels.Everything())
	if kerr.IsNotFound(err) {
		return nil, nil
	} else if err != nil {
		return nil, err
	}

	result := make([]*v1beta1_api.RestoreSession, 0)
	// keep only those RestoreSession that has this workload as target and restore is not completed
	for _, restoreSession := range restoreSessions {
		if restoreSession.DeletionTimestamp == nil && IsTarget(restoreSession.Spec.Target, w) {
			result = append(result, restoreSession)
		}
	}

	if len(result) == 0 {
		return nil, nil
	}

	// return currently running RestoreSession
	for _, r := range result {
		if r.Status.Phase == v1beta1_api.RestoreRunning {
			return r, nil
		}
	}
	// no running RestoreSession. so return pending one
	for _, r := range result {
		if r.Status.Phase == v1beta1_api.RestorePending {
			return r, nil
		}
	}
	// no pending or running restore session so return failed one
	for _, r := range result {
		if r.Status.Phase == v1beta1_api.RestoreFailed {
			return r, nil
		}
	}
	// by default return first RestoreSession
	return result[0], nil
}

// RestoreSessionEqual check weather two RestoreSessions has same specification.
func RestoreSessionEqual(old, new *v1beta1_api.RestoreSession) bool {

	var oldSpec, newSpec *v1beta1_api.RestoreSessionSpec
	if old != nil {
		oldSpec = &old.Spec
	}
	if new != nil {
		newSpec = &new.Spec
	}
	return reflect.DeepEqual(oldSpec, newSpec)
}

func RestorePending(phase v1beta1_api.RestorePhase) bool {
	if phase == "" || phase == v1beta1_api.RestorePending {
		return true
	}
	return false
}

func RestoreRunning(phase v1beta1_api.RestorePhase) bool {
	if phase == v1beta1_api.RestoreRunning {
		return true
	}
	return false
}

func RestoreCompleted(phase v1beta1_api.RestorePhase) bool {
	return !(RestorePending(phase) || RestoreRunning(phase))
}

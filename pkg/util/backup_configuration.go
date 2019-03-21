package util

import (
	"bytes"
	"fmt"
	"reflect"

	v1beta1_api "github.com/appscode/stash/apis/stash/v1beta1"
	v1beta1_listers "github.com/appscode/stash/client/listers/stash/v1beta1"
	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/labels"
	"kmodules.xyz/client-go/meta"
	wapi "kmodules.xyz/webhook-runtime/apis/workload/v1"
)

// GetAppliedBackupConfiguration check weather BackupConfiguration was applied as annotation and returns the object definition if exist.
func GetAppliedBackupConfiguration(m map[string]string) (*v1beta1_api.BackupConfiguration, error) {
	data := GetString(m, v1beta1_api.KeyLastAppliedBackupConfiguration)

	if data == "" {
		return nil, nil
	}
	obj, err := meta.UnmarshalFromJSON([]byte(data), v1beta1_api.SchemeGroupVersion)
	if err != nil {
		return nil, err
	}
	backupConfiguration, ok := obj.(*v1beta1_api.BackupConfiguration)
	if !ok {
		return nil, fmt.Errorf("%s annotations has invalid BackupConfiguration object", v1beta1_api.KeyLastAppliedBackupConfiguration)
	}
	return backupConfiguration, nil
}

// FindBackupConfiguration check if there is any BackupConfiguration exist for a particular workload.
// If multiple BackupConfigurations are found for the workload, it returns error.
func FindBackupConfiguration(lister v1beta1_listers.BackupConfigurationLister, w *wapi.Workload) (*v1beta1_api.BackupConfiguration, error) {
	// list all BackupConfigurations from the lister
	backupConfigurations, err := lister.BackupConfigurations(w.Namespace).List(labels.Everything())
	if err != nil {
		return nil, err
	}
	result := make([]*v1beta1_api.BackupConfiguration, 0)
	// keep only those BackupConfiguration that has this workload as target
	for _, bc := range backupConfigurations {
		if bc.DeletionTimestamp == nil && IsTarget(bc.Spec.Target, w) {
			result = append(result, bc)
		}
	}

	// if there is more than one BackupConfiguration then return error
	if len(result) > 1 {
		var msg bytes.Buffer
		msg.WriteString(fmt.Sprintf("Workload %s/%s matches multiple BackupConfigurations:", w.Namespace, w.Name))
		for i, bc := range result {
			if i > 0 {
				msg.WriteString(", ")
			}
			msg.WriteString(bc.Name)
		}
		return nil, errors.New(msg.String())
	} else if len(result) == 1 {
		// only one BackupConfiguration is found for this workload. So, return it.
		return result[0], nil
	}
	return nil, nil
}

// BackupConfigurationEqual check weather two BackupConfigurations has same specification.
func BackupConfigurationEqual(old, new *v1beta1_api.BackupConfiguration) bool {
	var oldSpec, newSpec *v1beta1_api.BackupConfigurationSpec
	if old != nil {
		oldSpec = &old.Spec
	}
	if new != nil {
		newSpec = &new.Spec
	}
	return reflect.DeepEqual(oldSpec, newSpec)
}

func BackupPending(phase v1beta1_api.BackupSessionPhase) bool {
	if phase == "" || phase == v1beta1_api.BackupSessionPending {
		return true
	}
	return false
}

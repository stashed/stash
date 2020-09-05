/*
Copyright AppsCode Inc. and Contributors

Licensed under the AppsCode Community License 1.0.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    https://github.com/appscode/licenses/raw/1.0.0/AppsCode-Community-1.0.0.md

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controller

import (
	"fmt"

	"stash.appscode.dev/apimachinery/apis"
	api_v1beta1 "stash.appscode.dev/apimachinery/apis/stash/v1beta1"
	"stash.appscode.dev/stash/pkg/util"

	appsv1 "k8s.io/api/apps/v1"
	core "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ocapps "kmodules.xyz/openshift/apis/apps/v1"
	wapi "kmodules.xyz/webhook-runtime/apis/workload/v1"
)

// applyBackupLogic check if Backup annotations or BackupConfiguration is configured for this workload
// and perform operation accordingly
func (c *StashController) applyBackupLogic(w *wapi.Workload, caller string) (bool, error) {
	// check if any BackupConfiguration exist for this workload.
	// if exist then inject sidecar container
	modified, err := c.applyBackupConfigurationLogic(w, caller)
	if err != nil {
		return false, err
	}

	// if no BackupConfiguration is configured then check if Restic is configured (backward compatibility)
	if !modified {
		return c.applyResticLogic(w, caller)
	}
	return modified, nil
}

func (c *StashController) applyBackupConfigurationLogic(w *wapi.Workload, caller string) (bool, error) {
	// detect old BackupConfiguration from annotations if it does exist.
	oldbc, err := util.GetAppliedBackupConfiguration(w.Annotations)
	if err != nil {
		return false, err
	}
	// find existing BackupConfiguration for this workload
	newbc, err := util.FindBackupConfiguration(c.bcLister, w)
	if err != nil {
		return false, err
	}
	// if BackupConfiguration currently exist for this workload but it is not same as old one,
	// this means BackupConfiguration has been newly created/updated.
	// in this case, we have to add/update sidecar container accordingly.
	if newbc != nil && !util.BackupConfigurationEqual(oldbc, newbc) {
		invoker, err := apis.ExtractBackupInvokerInfo(c.stashClient, api_v1beta1.ResourceKindBackupConfiguration, newbc.Name, newbc.Namespace)
		if err != nil {
			return true, err
		}
		for _, targetInfo := range invoker.TargetsInfo {
			if targetInfo.Target != nil &&
				targetInfo.Target.Ref.Kind == w.Kind &&
				targetInfo.Target.Ref.Name == w.Name {
				err = c.ensureBackupSidecar(w, invoker, targetInfo, caller)
				if err != nil {
					return false, c.handleSidecarInjectionFailure(w, invoker, targetInfo.Target.Ref, err)
				}
				return true, c.handleSidecarInjectionSuccess(w, invoker, targetInfo.Target.Ref)
			}
		}

	} else if oldbc != nil && newbc == nil {
		// there was BackupConfiguration before but it does not exist now.
		// this means BackupConfiguration has been removed.
		// in this case, we have to delete the backup sidecar container
		// and remove respective annotations from the workload.
		c.ensureBackupSidecarDeleted(w)
		// write sidecar deletion failure/success event
		return true, c.handleSidecarDeletionSuccess(w)
	}
	return false, nil
}

func (c *StashController) applyResticLogic(w *wapi.Workload, caller string) (bool, error) {
	// detect old Restic from annotations if it does exist
	oldRestic, err := util.GetAppliedRestic(w.Annotations)
	if err != nil {
		return false, err
	}

	// find existing Restic for this workload
	newRestic, err := util.FindRestic(c.rstLister, w.ObjectMeta)
	if err != nil {
		return false, err
	}

	// if Restic currently exist for this workload but it is not same as old one,
	// this means Restic has been newly created/updated.
	// in this case, we have to add/update the sidecar container accordingly.
	if newRestic != nil && !util.ResticEqual(oldRestic, newRestic) {
		err := c.ensureWorkloadSidecar(w, newRestic, caller)
		if err != nil {
			return false, err
		}
		return true, nil
	} else if oldRestic != nil && newRestic == nil {
		// there was Restic before but currently does not exist.
		// this means Restic has been removed.
		// in this case, we have to delete the backup sidecar container
		// and remove respective annotations from the workload.
		c.ensureWorkloadSidecarDeleted(w, oldRestic)
		return true, nil
	}

	return false, nil
}

func ownerWorkload(w *wapi.Workload) (*metav1.OwnerReference, error) {
	switch w.Kind {
	case apis.KindDeployment, apis.KindStatefulSet, apis.KindDaemonSet, apis.KindReplicaSet:
		return metav1.NewControllerRef(w, appsv1.SchemeGroupVersion.WithKind(w.Kind)), nil
	case apis.KindReplicationController:
		return metav1.NewControllerRef(w, core.SchemeGroupVersion.WithKind(w.Kind)), nil
	case apis.KindDeploymentConfig:
		return metav1.NewControllerRef(w, ocapps.GroupVersion.WithKind(w.Kind)), nil
	default:
		return nil, fmt.Errorf("failed to set workload as owner. Reason: unknown workload kind")
	}
}

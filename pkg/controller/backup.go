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
	"stash.appscode.dev/apimachinery/pkg/invoker"
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
	return c.applyBackupConfigurationLogic(w, caller)
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
		inv := invoker.NewBackupConfigurationInvoker(c.stashClient, newbc)
		for _, targetInfo := range inv.GetTargetInfo() {
			if util.IsBackupTarget(targetInfo.Target, w, inv.GetObjectMeta().Namespace) {
				err = c.ensureBackupSidecar(w, inv, targetInfo, caller)
				if err != nil {
					return false, c.handleSidecarInjectionFailure(w, inv, targetInfo.Target.Ref, err)
				}
				return true, c.handleSidecarInjectionSuccess(w, inv, targetInfo.Target.Ref)
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

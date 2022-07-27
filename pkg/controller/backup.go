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
	"stash.appscode.dev/apimachinery/pkg/invoker"
	"stash.appscode.dev/stash/pkg/util"

	appsv1 "k8s.io/api/apps/v1"
	core "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	ocapps "kmodules.xyz/openshift/apis/apps/v1"
	wapi "kmodules.xyz/webhook-runtime/apis/workload/v1"
)

// applyBackupLogic check if Backup annotations or BackupConfiguration is configured for this workload
// and perform operation accordingly
func (c *StashController) applyBackupLogic(w *wapi.Workload, caller string) (bool, error) {
	return c.applyBackupInvokerLogic(w, caller)
}

func (c *StashController) applyBackupInvokerLogic(w *wapi.Workload, caller string) (bool, error) {
	oldInvoker, err := util.ExtractAppliedBackupInvokerFromAnnotation(w.Annotations)
	if err != nil {
		return false, err
	}
	targetRef := api_v1beta1.TargetRef{
		APIVersion: w.APIVersion,
		Kind:       w.Kind,
		Name:       w.Name,
		Namespace:  w.Namespace,
	}
	newInvoker, err := util.FindLatestBackupInvoker(c.bcLister, targetRef)
	if err != nil {
		return false, err
	}
	yes, err := backupInvokerCreatedOrUpdated(oldInvoker, newInvoker)
	if err != nil {
		return false, err
	}
	if yes {
		inv, err := invoker.NewBackupInvoker(
			c.stashClient,
			newInvoker.GetKind(),
			newInvoker.GetName(),
			newInvoker.GetNamespace(),
		)
		if err != nil {
			return false, err
		}

		for _, targetInfo := range inv.GetTargetInfo() {
			if util.IsBackupTarget(targetInfo.Target, targetRef, inv.GetObjectMeta().Namespace) {
				err = c.ensureBackupSidecar(w, inv, targetInfo, caller)
				if err != nil {
					return false, c.handleSidecarInjectionFailure(w, inv, targetInfo.Target.Ref, err)
				}
				return true, c.handleSidecarInjectionSuccess(w, inv, targetInfo.Target.Ref)
			}
		}

	} else if backupInvokerDeleted(oldInvoker, newInvoker) {
		c.ensureBackupSidecarDeleted(w)
		return true, c.handleSidecarDeletionSuccess(w)
	}
	return false, nil
}

func backupInvokerCreatedOrUpdated(old, new unstructured.Unstructured) (bool, error) {
	if new.Object == nil {
		return false, nil
	}
	equal, err := util.InvokerEqual(old, new)
	if err != nil {
		return false, err
	}
	return !equal, nil
}

func backupInvokerDeleted(old, new unstructured.Unstructured) bool {
	return old.Object != nil && new.Object == nil
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

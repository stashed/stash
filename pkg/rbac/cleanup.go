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

package rbac

import (
	"context"
	"fmt"

	"stash.appscode.dev/stash/pkg/util"

	kerr "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/kubernetes"
	"k8s.io/klog/v2"
	core_util "kmodules.xyz/client-go/core/v1"
	wapi "kmodules.xyz/webhook-runtime/apis/workload/v1"
)

func (opt *Options) EnsureRBACResourcesDeleted() error {
	err := opt.ensureClusterRoleBindingDeleted()
	if err != nil {
		return err
	}

	return opt.ensureCrossNamespaceRBACResourcesDeleted()
}

func (opt *Options) ensureClusterRoleBindingDeleted() error {
	// List all the ClusterRoleBinding with the provided labels
	resources, err := opt.kubeClient.RbacV1().ClusterRoleBindings().List(context.TODO(), metav1.ListOptions{LabelSelector: labels.SelectorFromSet(opt.offshootLabels).String()})
	if err != nil {
		return err
	}
	// Delete the ClusterRoleBindings that are controlled by the provided owner
	for i := range resources.Items {
		if owned, _ := core_util.IsOwnedBy(&resources.Items[i], &opt.invOpts); owned {
			err = opt.kubeClient.RbacV1().ClusterRoleBindings().Delete(context.TODO(), resources.Items[i].Name, metav1.DeleteOptions{})
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func (opt *Options) ensureCrossNamespaceRBACResourcesDeleted() error {
	if opt.crossNamespaceResources == nil {
		return nil
	}

	rolebindings, err := opt.kubeClient.RbacV1().RoleBindings(opt.crossNamespaceResources.Namespace).List(context.TODO(), metav1.ListOptions{LabelSelector: labels.SelectorFromSet(opt.offshootLabels).String()})
	if err != nil {
		return err
	}

	for i := range rolebindings.Items {
		err = opt.kubeClient.RbacV1().RoleBindings(opt.crossNamespaceResources.Namespace).Delete(context.TODO(), rolebindings.Items[i].Name, metav1.DeleteOptions{})
		if err != nil {
			return err
		}
	}

	roles, err := opt.kubeClient.RbacV1().Roles(opt.crossNamespaceResources.Namespace).List(context.TODO(), metav1.ListOptions{LabelSelector: labels.SelectorFromSet(opt.offshootLabels).String()})
	if err != nil {
		return err
	}

	for i := range roles.Items {
		err = opt.kubeClient.RbacV1().Roles(opt.crossNamespaceResources.Namespace).Delete(context.TODO(), roles.Items[i].Name, metav1.DeleteOptions{})
		if err != nil {
			return err
		}
	}

	return nil
}

func ensureSidecarRoleBindingDeleted(kubeClient kubernetes.Interface, logger klog.Logger, w *wapi.Workload) error {
	err := kubeClient.RbacV1().RoleBindings(w.Namespace).Delete(
		context.TODO(),
		getSidecarRoleBindingName(w.Name, w.Kind),
		metav1.DeleteOptions{})
	if err != nil {
		if kerr.IsNotFound(err) {
			return nil
		}
		return err
	}
	logger.Info(fmt.Sprintf("RoleBinding %s/%s has been deleted",
		w.Namespace,
		getSidecarRoleBindingName(w.Name, w.Kind),
	))
	return nil
}

func EnsureUnnecessaryWorkloadRBACDeleted(kubeClient kubernetes.Interface, logger klog.Logger, w *wapi.Workload) error {
	// delete backup sidecar RoleBinding if workload does not have stash sidecar
	if !util.HasStashSidecar(w.Spec.Template.Spec.Containers) {
		err := ensureSidecarRoleBindingDeleted(kubeClient, logger, w)
		if err != nil && !kerr.IsNotFound(err) {
			return err
		}
	}

	// delete restore init-container RoleBinding if workload does not have sash init-container
	if !util.HasStashInitContainer(w.Spec.Template.Spec.InitContainers) {
		err := ensureRestoreInitContainerRoleBindingDeleted(kubeClient, logger, w)
		if err != nil && !kerr.IsNotFound(err) {
			return err
		}
	}

	return nil
}

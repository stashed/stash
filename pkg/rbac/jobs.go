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

package rbac

import (
	"context"
	"fmt"

	"stash.appscode.dev/apimachinery/apis"

	rbac "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/kubernetes"
	core_util "kmodules.xyz/client-go/core/v1"
	meta_util "kmodules.xyz/client-go/meta"
	rbac_util "kmodules.xyz/client-go/rbac/v1"
)

func EnsureLicenseReaderClusterRoleBinding(kc kubernetes.Interface, owner *metav1.OwnerReference, namespace, sa string, labels map[string]string) error {
	meta := metav1.ObjectMeta{
		Name:   meta_util.NameWithSuffix(apis.LicenseReader, fmt.Sprintf("%s-%s", namespace, sa)),
		Labels: labels,
	}
	_, _, err := rbac_util.CreateOrPatchClusterRoleBinding(context.TODO(), kc, meta, func(in *rbac.ClusterRoleBinding) *rbac.ClusterRoleBinding {
		core_util.EnsureOwnerReference(&in.ObjectMeta, owner)

		in.RoleRef = rbac.RoleRef{
			APIGroup: rbac.GroupName,
			Kind:     apis.KindClusterRole,
			Name:     apis.LicenseReader,
		}
		in.Subjects = []rbac.Subject{
			{
				Kind:      rbac.ServiceAccountKind,
				Name:      sa,
				Namespace: namespace,
			},
		}
		return in
	}, metav1.PatchOptions{})
	return err
}

func EnsureClusterRoleBindingDeleted(kubeClient kubernetes.Interface, owner metav1.ObjectMeta, selector map[string]string) error {
	// List all the ClusterRoleBinding with the provided labels
	resources, err := kubeClient.RbacV1().ClusterRoleBindings().List(context.TODO(), metav1.ListOptions{LabelSelector: labels.SelectorFromSet(selector).String()})
	if err != nil {
		return err
	}
	// Delete the ClusterRoleBindings that are controlled by the provided owner
	for i := range resources.Items {
		if metav1.IsControlledBy(&resources.Items[i], &owner) {
			err = kubeClient.RbacV1().ClusterRoleBindings().Delete(context.TODO(), resources.Items[i].Name, metav1.DeleteOptions{})
			if err != nil {
				return err
			}
		}
	}
	return nil
}

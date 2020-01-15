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
	"fmt"
	"strings"

	"stash.appscode.dev/stash/apis"
	api_v1alpha1 "stash.appscode.dev/stash/apis/stash/v1alpha1"
	api_v1beta1 "stash.appscode.dev/stash/apis/stash/v1beta1"

	core "k8s.io/api/core/v1"
	policy "k8s.io/api/policy/v1beta1"
	rbac "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	core_util "kmodules.xyz/client-go/core/v1"
	rbac_util "kmodules.xyz/client-go/rbac/v1"
	appCatalog "kmodules.xyz/custom-resources/apis/appcatalog/v1alpha1"
)

func EnsureBackupJobRBAC(kubeClient kubernetes.Interface, owner *metav1.OwnerReference, namespace, sa string, psps []string, labels map[string]string) error {
	// ensure ClusterRole for restore job
	err := ensureBackupJobClusterRole(kubeClient, psps, labels)
	if err != nil {
		return err
	}

	// ensure RoleBinding for restore job
	err = ensureBackupJobRoleBinding(kubeClient, owner, namespace, sa, labels)
	if err != nil {
		return err
	}

	return nil
}

func ensureBackupJobClusterRole(kubeClient kubernetes.Interface, psps []string, labels map[string]string) error {

	meta := metav1.ObjectMeta{
		Name:   apis.StashBackupJobClusterRole,
		Labels: labels,
	}
	_, _, err := rbac_util.CreateOrPatchClusterRole(kubeClient, meta, func(in *rbac.ClusterRole) *rbac.ClusterRole {

		in.Rules = []rbac.PolicyRule{
			{
				APIGroups: []string{api_v1beta1.SchemeGroupVersion.Group},
				Resources: []string{"*"},
				Verbs:     []string{"*"},
			},
			{
				APIGroups: []string{api_v1alpha1.SchemeGroupVersion.Group},
				Resources: []string{"*"},
				Verbs:     []string{"*"},
			},
			{
				APIGroups: []string{appCatalog.SchemeGroupVersion.Group},
				Resources: []string{appCatalog.ResourceApps},
				Verbs:     []string{"get"},
			},
			{
				APIGroups: []string{core.SchemeGroupVersion.Group},
				Resources: []string{"secrets", "endpoints", "pods"},
				Verbs:     []string{"get"},
			},
			{
				APIGroups: []string{core.SchemeGroupVersion.Group},
				Resources: []string{"pods/exec"},
				Verbs:     []string{"get", "create"},
			},
			{
				APIGroups: []string{core.GroupName},
				Resources: []string{"events"},
				Verbs:     []string{"create"},
			},
			{
				APIGroups:     []string{policy.GroupName},
				Resources:     []string{"podsecuritypolicies"},
				Verbs:         []string{"use"},
				ResourceNames: psps,
			},
		}
		return in
	})
	return err
}

func ensureBackupJobRoleBinding(kubeClient kubernetes.Interface, owner *metav1.OwnerReference, namespace, sa string, labels map[string]string) error {

	meta := metav1.ObjectMeta{
		Name:      getBackupJobRoleBindingName(owner.Name),
		Namespace: namespace,
		Labels:    labels,
	}
	_, _, err := rbac_util.CreateOrPatchRoleBinding(kubeClient, meta, func(in *rbac.RoleBinding) *rbac.RoleBinding {
		core_util.EnsureOwnerReference(&in.ObjectMeta, owner)

		in.RoleRef = rbac.RoleRef{
			APIGroup: rbac.GroupName,
			Kind:     apis.KindClusterRole,
			Name:     apis.StashBackupJobClusterRole,
		}
		in.Subjects = []rbac.Subject{
			{
				Kind:      rbac.ServiceAccountKind,
				Name:      sa,
				Namespace: namespace,
			},
		}
		return in
	})
	return err
}

func getBackupJobRoleBindingName(name string) string {
	return fmt.Sprintf("%s-%s", apis.StashBackupJobClusterRole, strings.ReplaceAll(name, ".", "-"))
}

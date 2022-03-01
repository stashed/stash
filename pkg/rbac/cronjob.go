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
	"strings"

	"stash.appscode.dev/apimachinery/apis"
	api_v1beta1 "stash.appscode.dev/apimachinery/apis/stash/v1beta1"

	apps "k8s.io/api/apps/v1"
	core "k8s.io/api/core/v1"
	policy "k8s.io/api/policy/v1beta1"
	rbac "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	core_util "kmodules.xyz/client-go/core/v1"
	rbac_util "kmodules.xyz/client-go/rbac/v1"
)

func EnsureCronJobRBAC(kubeClient kubernetes.Interface, owner *metav1.OwnerReference, namespace, sa string, psps []string, labels map[string]string) error {
	// ensure CronJob cluster role
	err := ensureCronJobClusterRole(kubeClient, psps, labels)
	if err != nil {
		return err
	}

	// ensure RoleBinding
	return ensureCronJobRoleBinding(kubeClient, owner, namespace, sa, labels)
}

func ensureCronJobClusterRole(kc kubernetes.Interface, psps []string, labels map[string]string) error {
	meta := metav1.ObjectMeta{
		Name:   apis.StashCronJobClusterRole,
		Labels: labels,
	}
	_, _, err := rbac_util.CreateOrPatchClusterRole(context.TODO(), kc, meta, func(in *rbac.ClusterRole) *rbac.ClusterRole {
		in.Rules = []rbac.PolicyRule{
			{
				APIGroups: []string{api_v1beta1.SchemeGroupVersion.Group},
				Resources: []string{"*"},
				Verbs:     []string{"*"},
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
			{
				APIGroups: []string{apps.GroupName},
				Resources: []string{"deployments", "statefulsets", "replicasets", "daemonsets"},
				Verbs:     []string{"get"},
			},
			{
				APIGroups: []string{core.GroupName},
				Resources: []string{"replicationcontrollers", "persistentvolumeclaims"},
				Verbs:     []string{"get"},
			},
			{
				APIGroups: []string{"apps.openshift.io"},
				Resources: []string{"deploymentconfigs"},
				Verbs:     []string{"get"},
			},
			{
				APIGroups: []string{"appcatalog.appscode.com"},
				Resources: []string{"*"},
				Verbs:     []string{"get"},
			},
		}
		return in
	}, metav1.PatchOptions{})
	return err
}

func ensureCronJobRoleBinding(kc kubernetes.Interface, owner *metav1.OwnerReference, namespace, sa string, labels map[string]string) error {
	meta := metav1.ObjectMeta{
		Name:      getCronJobRoleBindingName(sa),
		Namespace: namespace,
		Labels:    labels,
	}

	// ensure role binding
	_, _, err := rbac_util.CreateOrPatchRoleBinding(context.TODO(), kc, meta, func(in *rbac.RoleBinding) *rbac.RoleBinding {
		core_util.EnsureOwnerReference(&in.ObjectMeta, owner)

		in.RoleRef = rbac.RoleRef{
			APIGroup: rbac.GroupName,
			Kind:     apis.KindClusterRole,
			Name:     apis.StashCronJobClusterRole,
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
	if err != nil {
		return err
	}
	return nil
}

func getCronJobRoleBindingName(name string) string {
	// Create RoleBinding with name same as the ServiceAccount name.
	// The ServiceAccount already has Stash specific prefix in it's name.
	return strings.ReplaceAll(name, ".", "-")
}

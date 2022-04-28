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
	api_v1alpha1 "stash.appscode.dev/apimachinery/apis/stash/v1alpha1"
	api_v1beta1 "stash.appscode.dev/apimachinery/apis/stash/v1beta1"

	apps "k8s.io/api/apps/v1"
	core "k8s.io/api/core/v1"
	policy "k8s.io/api/policy/v1beta1"
	rbac "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	core_util "kmodules.xyz/client-go/core/v1"
	"kmodules.xyz/client-go/meta"
	rbac_util "kmodules.xyz/client-go/rbac/v1"
	appCatalog "kmodules.xyz/custom-resources/apis/appcatalog/v1alpha1"
)

func (opt *RBACOptions) EnsureRestoreJobRBAC() error {
	if opt.ServiceAccount.Name == "" {
		opt.ServiceAccount.Name = meta.ValidNameWithPrefixNSuffix(strings.ToLower(opt.Invoker.Kind), opt.Invoker.Name, opt.Suffix)
		err := opt.ensureServiceAccount()
		if err != nil {
			return err
		}
	}

	// ensure ClusterRole for restore job
	err := opt.ensureRestoreJobClusterRole()
	if err != nil {
		return err
	}

	// ensure RoleBinding for restore job
	err = opt.ensureRestoreJobRoleBinding()
	if err != nil {
		return err
	}

	err = opt.ensureCrossNamespaceRBAC()
	if err != nil {
		return err
	}

	return opt.ensureLicenseReaderClusterRoleBinding()
}

func (opt *RBACOptions) ensureRestoreJobClusterRole() error {
	meta := metav1.ObjectMeta{
		Name:   apis.StashRestoreJobClusterRole,
		Labels: opt.OffshootLabels,
	}
	_, _, err := rbac_util.CreateOrPatchClusterRole(context.TODO(), opt.KubeClient, meta, func(in *rbac.ClusterRole) *rbac.ClusterRole {
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
				Resources: []string{"secrets", "endpoints", "persistentvolumeclaims"},
				Verbs:     []string{"get"},
			},
			{
				APIGroups: []string{core.SchemeGroupVersion.Group},
				Resources: []string{"pods", "pods/exec"},
				Verbs:     []string{"get", "create", "list"},
			},
			{
				APIGroups: []string{core.SchemeGroupVersion.Group},
				Resources: []string{"serviceaccounts"},
				Verbs:     []string{"get", "create", "patch"},
			},
			{
				APIGroups: []string{apps.SchemeGroupVersion.Group},
				Resources: []string{"statefulsets"},
				Verbs:     []string{"get", "patch"},
			},
			{
				APIGroups: []string{rbac.SchemeGroupVersion.Group},
				Resources: []string{"roles", "rolebindings"},
				Verbs:     []string{"get", "create", "patch"},
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
				ResourceNames: opt.PodSecurityPolicyNames,
			},
		}
		return in
	}, metav1.PatchOptions{})
	return err
}

func (opt *RBACOptions) ensureRestoreJobRoleBinding() error {
	meta := metav1.ObjectMeta{
		Namespace: opt.Invoker.Namespace,
		Name:      opt.getRoleBindingName(),
		Labels:    opt.OffshootLabels,
	}
	_, _, err := rbac_util.CreateOrPatchRoleBinding(context.TODO(), opt.KubeClient, meta, func(in *rbac.RoleBinding) *rbac.RoleBinding {
		core_util.EnsureOwnerReference(&in.ObjectMeta, opt.Owner)

		in.RoleRef = rbac.RoleRef{
			APIGroup: rbac.GroupName,
			Kind:     apis.KindClusterRole,
			Name:     apis.StashRestoreJobClusterRole,
		}
		in.Subjects = []rbac.Subject{
			{
				Kind:      rbac.ServiceAccountKind,
				Name:      opt.ServiceAccount.Name,
				Namespace: opt.ServiceAccount.Namespace,
			},
		}
		return in
	}, metav1.PatchOptions{})
	return err
}

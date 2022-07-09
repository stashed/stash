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
	api "stash.appscode.dev/apimachinery/apis/stash/v1alpha1"
	api_v1beta1 "stash.appscode.dev/apimachinery/apis/stash/v1beta1"

	apps "k8s.io/api/apps/v1"
	coordination "k8s.io/api/coordination/v1"
	core "k8s.io/api/core/v1"
	rbac "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/kubernetes/pkg/apis/batch"
	core_util "kmodules.xyz/client-go/core/v1"
	meta_util "kmodules.xyz/client-go/meta"
	rbac_util "kmodules.xyz/client-go/rbac/v1"
)

func getSidecarRoleBindingName(name string, kind string) string {
	return meta_util.ValidNameWithPrefixNSuffix(apis.StashSidecarClusterRole, strings.ToLower(kind), name)
}

func (opt *RBACOptions) EnsureSideCarRBAC() error {
	err := opt.ensureSidecarClusterRole()
	if err != nil {
		return err
	}

	err = opt.ensureSidecarRoleBinding()
	if err != nil {
		return err
	}

	return opt.ensureCrossNamespaceRBAC()
}

func (opt *RBACOptions) ensureSidecarClusterRole() error {
	meta := metav1.ObjectMeta{Name: apis.StashSidecarClusterRole}
	_, _, err := rbac_util.CreateOrPatchClusterRole(context.TODO(), opt.KubeClient, meta, func(in *rbac.ClusterRole) *rbac.ClusterRole {
		if in.Labels == nil {
			in.Labels = map[string]string{}
		}
		in.Labels[apis.LabelApp] = apis.AppLabelStash

		in.Rules = []rbac.PolicyRule{
			{
				APIGroups: []string{api.SchemeGroupVersion.Group},
				Resources: []string{"*"},
				Verbs:     []string{"*"},
			},
			{
				APIGroups: []string{api_v1beta1.SchemeGroupVersion.Group},
				Resources: []string{"*"},
				Verbs:     []string{"*"},
			},
			{
				APIGroups: []string{apps.GroupName},
				Resources: []string{"deployments", "statefulsets", "daemonsets", "replicasets"},
				Verbs:     []string{"get", "list", "patch"},
			},
			{
				APIGroups: []string{apps.GroupName},
				Resources: []string{
					"deployments/finalizers",
					"statefulsets/finalizers",
					"daemonsets/finalizers",
					"replicasets/finalizers",
				},
				Verbs: []string{"update"},
			},
			{
				APIGroups: []string{core.GroupName},
				Resources: []string{"replicationcontrollers"},
				Verbs:     []string{"get", "list", "patch"},
			},
			{
				APIGroups: []string{core.GroupName},
				Resources: []string{"replicationcontrollers/finalizers"},
				Verbs:     []string{"update"},
			},
			{
				APIGroups: []string{core.GroupName},
				Resources: []string{"secrets", "pods"},
				Verbs:     []string{"get"},
			},
			{
				APIGroups: []string{core.GroupName},
				Resources: []string{"pods/exec"},
				Verbs:     []string{"get", "create"},
			},
			{
				APIGroups: []string{core.GroupName},
				Resources: []string{"configmaps"},
				Verbs:     []string{"create", "update", "get"},
			},
			{
				APIGroups: []string{core.GroupName},
				Resources: []string{"events"},
				Verbs:     []string{"create"},
			},
			{
				APIGroups: []string{batch.GroupName},
				Resources: []string{"jobs"},
				Verbs:     []string{"create", "get"},
			},
			{
				APIGroups: []string{rbac.GroupName},
				Resources: []string{"clusterroles", "roles", "rolebindings"},
				Verbs:     []string{"get", "create"},
			},
			{
				APIGroups: []string{core.GroupName},
				Resources: []string{"serviceaccounts"},
				Verbs:     []string{"get", "create"},
			},
			{
				APIGroups: []string{coordination.GroupName},
				Resources: []string{"leases"},
				Verbs:     []string{"*"},
			},
			{
				APIGroups: []string{"apps.openshift.io"},
				Resources: []string{"deploymentconfigs"},
				Verbs:     []string{"get", "list", "patch"},
			},
			{
				APIGroups: []string{"apps.openshift.io"},
				Resources: []string{"deploymentconfigs/finalizers"},
				Verbs:     []string{"update"},
			},
		}
		return in
	}, metav1.PatchOptions{})
	return err
}

func (opt *RBACOptions) ensureSidecarRoleBinding() error {
	meta := metav1.ObjectMeta{
		Namespace: opt.Invoker.Namespace,
		Name:      getSidecarRoleBindingName(opt.Owner.Name, opt.Owner.Kind),
		Labels:    opt.OffshootLabels,
	}
	_, _, err := rbac_util.CreateOrPatchRoleBinding(context.TODO(), opt.KubeClient, meta, func(in *rbac.RoleBinding) *rbac.RoleBinding {
		core_util.EnsureOwnerReference(&in.ObjectMeta, opt.Owner)

		if in.Annotations == nil {
			in.Annotations = map[string]string{}
		}

		in.RoleRef = rbac.RoleRef{
			APIGroup: rbac.GroupName,
			Kind:     apis.KindClusterRole,
			Name:     apis.StashSidecarClusterRole,
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

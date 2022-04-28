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

	"stash.appscode.dev/apimachinery/apis"
	api_v1beta1 "stash.appscode.dev/apimachinery/apis/stash/v1beta1"

	apps "k8s.io/api/apps/v1"
	core "k8s.io/api/core/v1"
	policy "k8s.io/api/policy/v1beta1"
	rbac "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	core_util "kmodules.xyz/client-go/core/v1"
	rbac_util "kmodules.xyz/client-go/rbac/v1"
)

func (opt *RBACOptions) EnsureCronJobRBAC(cronJobName string) error {
	if opt.ServiceAccount.Name == "" {
		opt.ServiceAccount.Name = cronJobName
		err := opt.ensureServiceAccount()
		if err != nil {
			return err
		}
	}

	err := opt.ensureCronJobClusterRole()
	if err != nil {
		return err
	}
	return opt.ensureCronJobRoleBinding(cronJobName)
}

func (opt *RBACOptions) ensureCronJobClusterRole() error {
	meta := metav1.ObjectMeta{
		Name:   apis.StashCronJobClusterRole,
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

func (opt *RBACOptions) ensureCronJobRoleBinding(cronJobName string) error {
	meta := metav1.ObjectMeta{
		Name:      cronJobName,
		Namespace: opt.Invoker.Namespace,
		Labels:    opt.OffshootLabels,
	}

	// ensure role binding
	_, _, err := rbac_util.CreateOrPatchRoleBinding(context.TODO(), opt.KubeClient, meta, func(in *rbac.RoleBinding) *rbac.RoleBinding {
		core_util.EnsureOwnerReference(&in.ObjectMeta, opt.Owner)

		in.RoleRef = rbac.RoleRef{
			APIGroup: rbac.GroupName,
			Kind:     apis.KindClusterRole,
			Name:     apis.StashCronJobClusterRole,
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
	if err != nil {
		return err
	}
	return nil
}

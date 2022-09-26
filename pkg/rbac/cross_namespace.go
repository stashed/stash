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
	"strings"

	"stash.appscode.dev/apimachinery/apis"
	api_v1alpha1 "stash.appscode.dev/apimachinery/apis/stash/v1alpha1"

	core "k8s.io/api/core/v1"
	rbac "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	meta_util "kmodules.xyz/client-go/meta"
	rbac_util "kmodules.xyz/client-go/rbac/v1"
)

func (opt *Options) ensureCrossNamespaceRBAC() error {
	if opt.crossNamespaceResources == nil {
		return nil
	}

	err := opt.ensureCrossNamespaceRole()
	if err != nil {
		return err
	}

	return opt.ensureCrossNamespaceRoleBinding()
}

func (opt *Options) ensureCrossNamespaceRole() error {
	meta := metav1.ObjectMeta{
		Name:      opt.getCrossNamespaceRoleName(),
		Namespace: opt.crossNamespaceResources.Namespace,
		Labels:    opt.offshootLabels,
	}
	_, _, err := rbac_util.CreateOrPatchRole(context.TODO(), opt.kubeClient, meta, func(in *rbac.Role) *rbac.Role {
		in.Rules = []rbac.PolicyRule{
			{
				APIGroups:     []string{api_v1alpha1.SchemeGroupVersion.Group},
				Resources:     []string{"repositories", "repositories/status"},
				Verbs:         []string{"get", "list", "patch", "update"},
				ResourceNames: []string{opt.crossNamespaceResources.Repository},
			},
			{
				APIGroups:     []string{core.SchemeGroupVersion.Group},
				Resources:     []string{"secrets", "pods", "endpoints"},
				Verbs:         []string{"get"},
				ResourceNames: []string{opt.crossNamespaceResources.Secret},
			},
		}
		return in
	}, metav1.PatchOptions{})
	return err
}

func (opt *Options) ensureCrossNamespaceRoleBinding() error {
	meta := metav1.ObjectMeta{
		Name:      opt.getCrossNamespaceRoleName(),
		Namespace: opt.crossNamespaceResources.Namespace,
		Labels:    opt.offshootLabels,
	}

	_, _, err := rbac_util.CreateOrPatchRoleBinding(context.TODO(), opt.kubeClient, meta, func(in *rbac.RoleBinding) *rbac.RoleBinding {
		in.RoleRef = rbac.RoleRef{
			APIGroup: rbac.GroupName,
			Kind:     apis.KindRole,
			Name:     opt.getCrossNamespaceRoleName(),
		}
		in.Subjects = []rbac.Subject{
			{
				Kind:      rbac.ServiceAccountKind,
				Name:      opt.serviceAccount.Name,
				Namespace: opt.serviceAccount.Namespace,
			},
		}
		return in
	}, metav1.PatchOptions{})
	return err
}

func (opt *Options) getRoleBindingName() string {
	return meta_util.ValidNameWithPrefixNSuffix(strings.ToLower(opt.invOpts.Kind), opt.invOpts.Name, opt.suffix)
}

func (opt *Options) getCrossNamespaceRoleName() string {
	return meta_util.ValidNameWithPrefixNSuffix(
		opt.invOpts.Namespace,
		strings.Join([]string{strings.ToLower(opt.invOpts.Kind), opt.invOpts.Name}, "-"),
		opt.suffix,
	)
}

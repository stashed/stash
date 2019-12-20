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

	api_v1alpha1 "stash.appscode.dev/stash/apis/stash/v1alpha1"
	stash_cs "stash.appscode.dev/stash/client/clientset/versioned"
	"stash.appscode.dev/stash/pkg/util"

	"github.com/golang/glog"
	apps "k8s.io/api/apps/v1"
	core "k8s.io/api/core/v1"
	rbac "k8s.io/api/rbac/v1"
	kerr "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd/api"
	core_util "kmodules.xyz/client-go/core/v1"
	meta_util "kmodules.xyz/client-go/meta"
	rbac_util "kmodules.xyz/client-go/rbac/v1"
)

const (
	ScaledownJobRole          = "stash-scaledownjob"
	StashRestoreInitContainer = "stash-restore-init-container"
	KindRole                  = "Role"
	KindClusterRole           = "ClusterRole"
)

// use scaledownjob-role, service-account and role-binding name same as job name
// set job as owner of role, service-account and role-binding
func EnsureScaledownJobRBAC(kubeClient kubernetes.Interface, owner *metav1.OwnerReference, namespace string) error {
	// ensure roles
	meta := metav1.ObjectMeta{
		Name:      ScaledownJobRole,
		Namespace: namespace,
	}
	_, _, err := rbac_util.CreateOrPatchRole(kubeClient, meta, func(in *rbac.Role) *rbac.Role {
		core_util.EnsureOwnerReference(&in.ObjectMeta, owner)

		if in.Labels == nil {
			in.Labels = map[string]string{}
		}
		in.Labels["app"] = util.AppLabelStash

		in.Rules = []rbac.PolicyRule{
			{
				APIGroups: []string{core.GroupName},
				Resources: []string{"pods"},
				Verbs:     []string{"get", "list", "delete", "deletecollection"},
			},
			{
				APIGroups: []string{apps.GroupName},
				Resources: []string{"deployments", "statefulsets"},
				Verbs:     []string{"get", "list", "patch"},
			},
			{
				APIGroups: []string{apps.GroupName},
				Resources: []string{"daemonsets", "replicasets"},
				Verbs:     []string{"get", "list", "patch"},
			},
			{
				APIGroups: []string{core.GroupName},
				Resources: []string{"replicationcontrollers"},
				Verbs:     []string{"get", "list", "patch"},
			},
		}
		return in
	})
	if err != nil {
		return err
	}

	// ensure service account
	meta = metav1.ObjectMeta{
		Name:      owner.Name,
		Namespace: namespace,
	}
	_, _, err = core_util.CreateOrPatchServiceAccount(kubeClient, meta, func(in *core.ServiceAccount) *core.ServiceAccount {
		core_util.EnsureOwnerReference(&in.ObjectMeta, owner)
		if in.Labels == nil {
			in.Labels = map[string]string{}
		}
		in.Labels["app"] = util.AppLabelStash
		return in
	})
	if err != nil {
		return err
	}

	// ensure role binding
	_, _, err = rbac_util.CreateOrPatchRoleBinding(kubeClient, meta, func(in *rbac.RoleBinding) *rbac.RoleBinding {
		core_util.EnsureOwnerReference(&in.ObjectMeta, owner)

		if in.Labels == nil {
			in.Labels = map[string]string{}
		}
		in.Labels["app"] = util.AppLabelStash

		in.RoleRef = rbac.RoleRef{
			APIGroup: rbac.GroupName,
			Kind:     "Role",
			Name:     ScaledownJobRole,
		}
		in.Subjects = []rbac.Subject{
			{
				Kind:      "ServiceAccount",
				Name:      meta.Name,
				Namespace: meta.Namespace,
			},
		}
		return in
	})
	return err
}

// use sidecar-cluster-role, service-account and role-binding name same as job name
// set job as owner of service-account and role-binding
func EnsureRecoveryRBAC(kubeClient kubernetes.Interface, owner *metav1.OwnerReference, namespace string) error {
	// ensure service account
	meta := metav1.ObjectMeta{
		Name:      owner.Name,
		Namespace: namespace,
	}
	_, _, err := core_util.CreateOrPatchServiceAccount(kubeClient, meta, func(in *core.ServiceAccount) *core.ServiceAccount {
		core_util.EnsureOwnerReference(&in.ObjectMeta, owner)
		if in.Labels == nil {
			in.Labels = map[string]string{}
		}
		return in
	})
	if err != nil {
		return err
	}

	// ensure role binding
	_, _, err = rbac_util.CreateOrPatchRoleBinding(kubeClient, meta, func(in *rbac.RoleBinding) *rbac.RoleBinding {
		core_util.EnsureOwnerReference(&in.ObjectMeta, owner)

		if in.Labels == nil {
			in.Labels = map[string]string{}
		}
		in.Labels["app"] = util.AppLabelStash

		in.RoleRef = rbac.RoleRef{
			APIGroup: rbac.GroupName,
			Kind:     "ClusterRole",
			Name:     StashSidecar,
		}
		in.Subjects = []rbac.Subject{
			{
				Kind:      "ServiceAccount",
				Name:      meta.Name,
				Namespace: meta.Namespace,
			},
		}
		return in
	})
	return err
}

func EnsureRepoReaderRBAC(kubeClient kubernetes.Interface, stashClient stash_cs.Interface, owner *metav1.OwnerReference, rec *api_v1alpha1.Recovery) error {
	meta := metav1.ObjectMeta{
		Name:      GetRepoReaderRoleBindingName(owner.Name, rec.Namespace),
		Namespace: rec.Spec.Repository.Namespace,
	}

	repo, err := stashClient.StashV1alpha1().Repositories(rec.Spec.Repository.Namespace).Get(rec.Spec.Repository.Name, metav1.GetOptions{})
	if err != nil {
		return err
	}

	// ensure repo-reader role
	err = ensureRepoReaderRole(kubeClient, repo)
	if err != nil {
		return err
	}

	// ensure repo-reader role binding
	_, _, err = rbac_util.CreateOrPatchRoleBinding(kubeClient, meta, func(in *rbac.RoleBinding) *rbac.RoleBinding {

		if in.Labels == nil {
			in.Labels = map[string]string{}
		}
		in.Labels["app"] = util.AppLabelStash

		in.RoleRef = rbac.RoleRef{
			APIGroup: rbac.GroupName,
			Kind:     "Role",
			Name:     getRepoReaderRoleName(rec.Spec.Repository.Name),
		}

		in.Subjects = []rbac.Subject{
			{
				Kind:      "ServiceAccount",
				Name:      owner.Name,
				Namespace: rec.Namespace,
			},
		}
		return in
	})
	return err
}

func ensureRepoReaderRole(kubeClient kubernetes.Interface, repo *api_v1alpha1.Repository) error {
	meta := metav1.ObjectMeta{
		Name:      getRepoReaderRoleName(repo.Name),
		Namespace: repo.Namespace,
	}

	owner := metav1.NewControllerRef(repo, api_v1alpha1.SchemeGroupVersion.WithKind(api_v1alpha1.ResourceKindRepository))
	_, _, err := rbac_util.CreateOrPatchRole(kubeClient, meta, func(in *rbac.Role) *rbac.Role {
		core_util.EnsureOwnerReference(&in.ObjectMeta, owner)

		if in.Labels == nil {
			in.Labels = map[string]string{}
		}
		in.Labels["app"] = util.AppLabelStash

		in.Rules = []rbac.PolicyRule{
			{
				APIGroups:     []string{api.SchemeGroupVersion.Group},
				Resources:     []string{"repositories"},
				ResourceNames: []string{repo.Name},
				Verbs:         []string{"get"},
			},
			{
				APIGroups:     []string{core.GroupName},
				Resources:     []string{"secrets"},
				ResourceNames: []string{repo.Spec.Backend.StorageSecretName},
				Verbs:         []string{"get"},
			},
		}

		return in
	})
	return err
}

func getRepoReaderRoleName(repoName string) string {
	return "appscode:stash:repo-reader:" + repoName
}

func GetRepoReaderRoleBindingName(name, namespace string) string {
	return name + ":" + namespace + ":repo-reader"
}

func EnsureRepoReaderRolebindingDeleted(kubeClient kubernetes.Interface, stashClient stash_cs.Interface, meta *metav1.ObjectMeta) error {
	// if the job is not recovery job then don't do anything
	if !strings.HasPrefix(meta.Name, util.RecoveryJobPrefix) {
		return nil
	}

	// read recovery name from label
	if !meta_util.HasKey(meta.Labels, util.AnnotationRecovery) {
		return fmt.Errorf("missing recovery name in job's label")
	}

	recoveryName, err := meta_util.GetStringValue(meta.Labels, util.AnnotationRecovery)
	if err != nil {
		return err
	}

	// read recovery object
	recovery, err := stashClient.StashV1alpha1().Recoveries(meta.Namespace).Get(recoveryName, metav1.GetOptions{})
	if err != nil {
		return err
	}

	// delete role binding
	err = kubeClient.RbacV1().RoleBindings(recovery.Spec.Repository.Namespace).Delete(GetRepoReaderRoleBindingName(meta.Name, meta.Namespace), meta_util.DeleteInBackground())
	if err != nil && !kerr.IsNotFound(err) {
		return err
	}
	glog.Infof("Deleted repo-reader rolebinding: " + GetRepoReaderRoleBindingName(meta.Name, meta.Namespace))
	return nil
}

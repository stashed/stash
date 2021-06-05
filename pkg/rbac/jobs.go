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
	"strings"

	"stash.appscode.dev/apimachinery/apis"
	api_v1alpha1 "stash.appscode.dev/apimachinery/apis/stash/v1alpha1"
	stash_cs "stash.appscode.dev/apimachinery/client/clientset/versioned"

	apps "k8s.io/api/apps/v1"
	core "k8s.io/api/core/v1"
	rbac "k8s.io/api/rbac/v1"
	kerr "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd/api"
	"k8s.io/klog/v2"
	core_util "kmodules.xyz/client-go/core/v1"
	meta_util "kmodules.xyz/client-go/meta"
	rbac_util "kmodules.xyz/client-go/rbac/v1"
)

const (
	ScaledownJobRole = "stash-scaledownjob"
)

// use scaledownjob-role, service-account and role-binding name same as job name
// set job as owner of role, service-account and role-binding
func EnsureScaledownJobRBAC(kc kubernetes.Interface, owner *metav1.OwnerReference, namespace string) error {
	// ensure roles
	meta := metav1.ObjectMeta{
		Name:      ScaledownJobRole,
		Namespace: namespace,
	}
	_, _, err := rbac_util.CreateOrPatchRole(context.TODO(), kc, meta, func(in *rbac.Role) *rbac.Role {
		core_util.EnsureOwnerReference(&in.ObjectMeta, owner)

		if in.Labels == nil {
			in.Labels = map[string]string{}
		}
		in.Labels[apis.LabelApp] = apis.AppLabelStash

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
	}, metav1.PatchOptions{})
	if err != nil {
		return err
	}

	// ensure service account
	meta = metav1.ObjectMeta{
		Name:      owner.Name,
		Namespace: namespace,
	}
	_, _, err = core_util.CreateOrPatchServiceAccount(context.TODO(), kc, meta, func(in *core.ServiceAccount) *core.ServiceAccount {
		core_util.EnsureOwnerReference(&in.ObjectMeta, owner)
		if in.Labels == nil {
			in.Labels = map[string]string{}
		}
		in.Labels[apis.LabelApp] = apis.AppLabelStash
		return in
	}, metav1.PatchOptions{})
	if err != nil {
		return err
	}

	// ensure role binding
	_, _, err = rbac_util.CreateOrPatchRoleBinding(context.TODO(), kc, meta, func(in *rbac.RoleBinding) *rbac.RoleBinding {
		core_util.EnsureOwnerReference(&in.ObjectMeta, owner)

		if in.Labels == nil {
			in.Labels = map[string]string{}
		}
		in.Labels[apis.LabelApp] = apis.AppLabelStash

		in.RoleRef = rbac.RoleRef{
			APIGroup: rbac.GroupName,
			Kind:     apis.KindRole,
			Name:     ScaledownJobRole,
		}
		in.Subjects = []rbac.Subject{
			{
				Kind:      rbac.ServiceAccountKind,
				Name:      meta.Name,
				Namespace: meta.Namespace,
			},
		}
		return in
	}, metav1.PatchOptions{})
	return err
}

// use sidecar-cluster-role, service-account and role-binding name same as job name
// set job as owner of service-account and role-binding
func EnsureRecoveryRBAC(kc kubernetes.Interface, owner *metav1.OwnerReference, namespace string) error {
	// ensure service account
	meta := metav1.ObjectMeta{
		Name:      owner.Name,
		Namespace: namespace,
	}
	_, _, err := core_util.CreateOrPatchServiceAccount(context.TODO(), kc, meta, func(in *core.ServiceAccount) *core.ServiceAccount {
		core_util.EnsureOwnerReference(&in.ObjectMeta, owner)
		if in.Labels == nil {
			in.Labels = map[string]string{}
		}
		return in
	}, metav1.PatchOptions{})
	if err != nil {
		return err
	}

	// ensure role binding
	_, _, err = rbac_util.CreateOrPatchRoleBinding(context.TODO(), kc, meta, func(in *rbac.RoleBinding) *rbac.RoleBinding {
		core_util.EnsureOwnerReference(&in.ObjectMeta, owner)

		if in.Labels == nil {
			in.Labels = map[string]string{}
		}
		in.Labels[apis.LabelApp] = apis.AppLabelStash

		in.RoleRef = rbac.RoleRef{
			APIGroup: rbac.GroupName,
			Kind:     apis.KindClusterRole,
			Name:     apis.StashSidecarClusterRole,
		}
		in.Subjects = []rbac.Subject{
			{
				Kind:      rbac.ServiceAccountKind,
				Name:      meta.Name,
				Namespace: meta.Namespace,
			},
		}
		return in
	}, metav1.PatchOptions{})
	return err
}

func EnsureRepoReaderRBAC(kc kubernetes.Interface, stashClient stash_cs.Interface, owner *metav1.OwnerReference, rec *api_v1alpha1.Recovery) error {
	meta := metav1.ObjectMeta{
		Name:      GetRepoReaderRoleBindingName(owner.Name, rec.Namespace),
		Namespace: rec.Spec.Repository.Namespace,
	}

	repo, err := stashClient.StashV1alpha1().Repositories(rec.Spec.Repository.Namespace).Get(context.TODO(), rec.Spec.Repository.Name, metav1.GetOptions{})
	if err != nil {
		return err
	}

	// ensure repo-reader role
	err = ensureRepoReaderRole(kc, repo)
	if err != nil {
		return err
	}

	// ensure repo-reader role binding
	_, _, err = rbac_util.CreateOrPatchRoleBinding(context.TODO(), kc, meta, func(in *rbac.RoleBinding) *rbac.RoleBinding {

		if in.Labels == nil {
			in.Labels = map[string]string{}
		}
		in.Labels[apis.LabelApp] = apis.AppLabelStash

		in.RoleRef = rbac.RoleRef{
			APIGroup: rbac.GroupName,
			Kind:     apis.KindRole,
			Name:     getRepoReaderRoleName(rec.Spec.Repository.Name),
		}

		in.Subjects = []rbac.Subject{
			{
				Kind:      rbac.ServiceAccountKind,
				Name:      owner.Name,
				Namespace: rec.Namespace,
			},
		}
		return in
	}, metav1.PatchOptions{})
	return err
}

func ensureRepoReaderRole(kc kubernetes.Interface, repo *api_v1alpha1.Repository) error {
	meta := metav1.ObjectMeta{
		Name:      getRepoReaderRoleName(repo.Name),
		Namespace: repo.Namespace,
	}

	owner := metav1.NewControllerRef(repo, api_v1alpha1.SchemeGroupVersion.WithKind(api_v1alpha1.ResourceKindRepository))
	_, _, err := rbac_util.CreateOrPatchRole(context.TODO(), kc, meta, func(in *rbac.Role) *rbac.Role {
		core_util.EnsureOwnerReference(&in.ObjectMeta, owner)

		if in.Labels == nil {
			in.Labels = map[string]string{}
		}
		in.Labels[apis.LabelApp] = apis.AppLabelStash

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
	}, metav1.PatchOptions{})
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
	if !strings.HasPrefix(meta.Name, apis.RecoveryJobPrefix) {
		return nil
	}

	// read recovery name from label
	if !meta_util.HasKey(meta.Labels, apis.AnnotationRecovery) {
		return fmt.Errorf("missing recovery name in job's label")
	}

	recoveryName, err := meta_util.GetStringValue(meta.Labels, apis.AnnotationRecovery)
	if err != nil {
		return err
	}

	// read recovery object
	recovery, err := stashClient.StashV1alpha1().Recoveries(meta.Namespace).Get(context.TODO(), recoveryName, metav1.GetOptions{})
	if err != nil {
		return err
	}

	// delete role binding
	err = kubeClient.RbacV1().RoleBindings(recovery.Spec.Repository.Namespace).Delete(context.TODO(), GetRepoReaderRoleBindingName(meta.Name, meta.Namespace), meta_util.DeleteInBackground())
	if err != nil && !kerr.IsNotFound(err) {
		return err
	}
	klog.Infof("Deleted repo-reader rolebinding: " + GetRepoReaderRoleBindingName(meta.Name, meta.Namespace))
	return nil
}

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

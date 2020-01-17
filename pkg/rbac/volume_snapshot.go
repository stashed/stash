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
	"strings"

	"stash.appscode.dev/stash/apis"
	api_v1alpha1 "stash.appscode.dev/stash/apis/stash/v1alpha1"
	api_v1beta1 "stash.appscode.dev/stash/apis/stash/v1beta1"

	crdv1 "github.com/kubernetes-csi/external-snapshotter/pkg/apis/volumesnapshot/v1beta1"
	apps "k8s.io/api/apps/v1"
	core "k8s.io/api/core/v1"
	rbac "k8s.io/api/rbac/v1"
	storage_api_v1 "k8s.io/api/storage/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	core_util "kmodules.xyz/client-go/core/v1"
	meta_util "kmodules.xyz/client-go/meta"
	rbac_util "kmodules.xyz/client-go/rbac/v1"
)

func EnsureVolumeSnapshotterJobRBAC(kubeClient kubernetes.Interface, owner *metav1.OwnerReference, namespace, sa string, labels map[string]string) error {
	// ensure ClusterRole for VolumeSnapshot job
	err := ensureVolumeSnapshotterJobClusterRole(kubeClient, labels)
	if err != nil {
		return err
	}

	// ensure RoleBinding for VolumeSnapshot job
	err = ensureVolumeSnapshotterJobRoleBinding(kubeClient, owner, namespace, sa, labels)
	if err != nil {
		return err
	}

	return nil
}

func ensureVolumeSnapshotterJobClusterRole(kubeClient kubernetes.Interface, labels map[string]string) error {

	meta := metav1.ObjectMeta{
		Name:   apis.StashVolumeSnapshotterClusterRole,
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
				APIGroups: []string{core.GroupName},
				Resources: []string{"events"},
				Verbs:     []string{"create"},
			},
			{
				APIGroups: []string{core.GroupName},
				Resources: []string{"pods"},
				Verbs:     []string{"get"},
			},
			{
				APIGroups: []string{core.GroupName},
				Resources: []string{"pods/exec"},
				Verbs:     []string{"get", "create"},
			},
			{
				APIGroups: []string{apps.GroupName},
				Resources: []string{"deployments", "statefulsets"},
				Verbs:     []string{"get", "list"},
			},
			{
				APIGroups: []string{apps.GroupName},
				Resources: []string{"daemonsets", "replicasets"},
				Verbs:     []string{"get", "list"},
			},
			{
				APIGroups: []string{core.GroupName},
				Resources: []string{"replicationcontrollers"},
				Verbs:     []string{"get", "list"},
			},
			{
				APIGroups: []string{crdv1.GroupName},
				Resources: []string{"volumesnapshots", "volumesnapshotcontents", "volumesnapshotclasses"},
				Verbs:     []string{"create", "get", "list", "watch", "patch", "delete"},
			},
		}
		return in
	})
	return err
}

func ensureVolumeSnapshotterJobRoleBinding(kubeClient kubernetes.Interface, resource *metav1.OwnerReference, namespace, sa string, labels map[string]string) error {

	meta := metav1.ObjectMeta{
		Namespace: namespace,
		Name:      getVolumesnapshotterJobRoleBindingName(sa),
		Labels:    labels,
	}
	_, _, err := rbac_util.CreateOrPatchRoleBinding(kubeClient, meta, func(in *rbac.RoleBinding) *rbac.RoleBinding {
		core_util.EnsureOwnerReference(&in.ObjectMeta, resource)

		in.RoleRef = rbac.RoleRef{
			APIGroup: rbac.GroupName,
			Kind:     apis.KindClusterRole,
			Name:     apis.StashVolumeSnapshotterClusterRole,
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

func getVolumesnapshotterJobRoleBindingName(name string) string {
	// Create RoleBinding with name same as the ServiceAccount name.
	// The ServiceAccount already has Stash specific prefix in it's name.
	return strings.ReplaceAll(name, ".", "-")
}

func EnsureVolumeSnapshotRestorerJobRBAC(kubeClient kubernetes.Interface, owner *metav1.OwnerReference, namespace, sa string, labels map[string]string) error {
	// ensure ClusterRole for restore job
	err := ensureVolumeSnapshotRestorerJobClusterRole(kubeClient, labels)
	if err != nil {
		return err
	}

	// ensure RoleBinding for restore job
	err = ensureVolumeSnapshotRestorerJobRoleBinding(kubeClient, owner, namespace, sa, labels)
	if err != nil {
		return err
	}

	//ensure storageClass ClusterRole for restore job
	err = ensureStorageReaderClassClusterRole(kubeClient, labels)
	if err != nil {
		return err
	}

	//ensure storageClass ClusterRoleBinding for restore job
	err = ensureStorageClassReaderClusterRoleBinding(kubeClient, owner, namespace, sa, labels)
	if err != nil {
		return err
	}

	return nil
}

func ensureVolumeSnapshotRestorerJobClusterRole(kubeClient kubernetes.Interface, labels map[string]string) error {

	meta := metav1.ObjectMeta{
		Name:   apis.StashVolumeSnapshotRestorerClusterRole,
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
				APIGroups: []string{core.GroupName},
				Resources: []string{"events"},
				Verbs:     []string{"create"},
			},
			{
				APIGroups: []string{core.GroupName},
				Resources: []string{"persistentvolumeclaims"},
				Verbs:     []string{"get", "list", "watch", "create", "patch"},
			},
			{
				APIGroups: []string{storage_api_v1.GroupName},
				Resources: []string{"storageclasses"},
				Verbs:     []string{"get"},
			},
			{
				APIGroups: []string{crdv1.GroupName},
				Resources: []string{"volumesnapshots"},
				Verbs:     []string{"get"},
			},
		}
		return in

	})
	return err
}

func ensureVolumeSnapshotRestorerJobRoleBinding(kubeClient kubernetes.Interface, resource *metav1.OwnerReference, namespace, sa string, labels map[string]string) error {

	meta := metav1.ObjectMeta{
		Namespace: namespace,
		Name:      getVolumeSnapshotRestorerJobRoleBindingName(sa),
		Labels:    labels,
	}
	_, _, err := rbac_util.CreateOrPatchRoleBinding(kubeClient, meta, func(in *rbac.RoleBinding) *rbac.RoleBinding {
		core_util.EnsureOwnerReference(&in.ObjectMeta, resource)

		in.RoleRef = rbac.RoleRef{
			APIGroup: rbac.GroupName,
			Kind:     apis.KindClusterRole,
			Name:     apis.StashVolumeSnapshotRestorerClusterRole,
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

func getVolumeSnapshotRestorerJobRoleBindingName(name string) string {
	// Create RoleBinding with name same as the ServiceAccount name.
	// The ServiceAccount already has Stash specific prefix in it's name.
	return strings.ReplaceAll(name, ".", "-")
}

func ensureStorageReaderClassClusterRole(kubeClient kubernetes.Interface, labels map[string]string) error {

	meta := metav1.ObjectMeta{
		Name:   apis.StashStorageClassReaderClusterRole,
		Labels: labels,
	}
	_, _, err := rbac_util.CreateOrPatchClusterRole(kubeClient, meta, func(in *rbac.ClusterRole) *rbac.ClusterRole {

		in.Rules = []rbac.PolicyRule{
			{
				APIGroups: []string{storage_api_v1.GroupName},
				Resources: []string{"storageclasses"},
				Verbs:     []string{"get"},
			},
			{
				APIGroups: []string{crdv1.GroupName},
				Resources: []string{"volumesnapshots"},
				Verbs:     []string{"get"},
			},
		}
		return in

	})
	return err
}

func ensureStorageClassReaderClusterRoleBinding(kubeClient kubernetes.Interface, resource *metav1.OwnerReference, namespace, sa string, labels map[string]string) error {

	meta := metav1.ObjectMeta{
		Name:      getStorageClassReaderClusterRoleBindingName(sa),
		Namespace: namespace,
		Labels:    labels,
	}
	_, _, err := rbac_util.CreateOrPatchClusterRoleBinding(kubeClient, meta, func(in *rbac.ClusterRoleBinding) *rbac.ClusterRoleBinding {
		core_util.EnsureOwnerReference(&in.ObjectMeta, resource)

		in.RoleRef = rbac.RoleRef{
			APIGroup: rbac.GroupName,
			Kind:     apis.KindClusterRole,
			Name:     apis.StashStorageClassReaderClusterRole,
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

func getStorageClassReaderClusterRoleBindingName(name string) string {
	return meta_util.ValidNameWithPrefix(apis.StashStorageClassReaderClusterRole, strings.ReplaceAll(name, ".", "-"))
}

package rbac

import (
	crdv1 "github.com/kubernetes-csi/external-snapshotter/pkg/apis/volumesnapshot/v1alpha1"
	apps "k8s.io/api/apps/v1"
	core "k8s.io/api/core/v1"
	rbac "k8s.io/api/rbac/v1"
	storage_api_v1 "k8s.io/api/storage/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	core_util "kmodules.xyz/client-go/core/v1"
	rbac_util "kmodules.xyz/client-go/rbac/v1"
	api_v1alpha1 "stash.appscode.dev/stash/apis/stash/v1alpha1"
	api_v1beta1 "stash.appscode.dev/stash/apis/stash/v1beta1"
)

const (
	VolumeSnapshotterClusterRole      = "stash-volumesnapshot-job"
	VolumeSnapshotRestorerClusterRole = "stash-volumesnapshot-restore-job"
	StorageClassReaderClusterRole     = "stash-storageclass"
)

func EnsureVolumeSnapshotterJobRBAC(kubeClient kubernetes.Interface, ref *core.ObjectReference, sa string, labels map[string]string) error {
	// ensure ClusterRole for VolumeSnapshot job
	err := ensureVolumeSnapshotterJobClusterRole(kubeClient, labels)
	if err != nil {
		return err
	}

	// ensure RoleBinding for VolumeSnapshot job
	err = ensureVolumeSnapshotterJobRoleBinding(kubeClient, ref, sa, labels)
	if err != nil {
		return err
	}

	return nil
}

func ensureVolumeSnapshotterJobClusterRole(kubeClient kubernetes.Interface, labels map[string]string) error {

	meta := metav1.ObjectMeta{
		Name:   VolumeSnapshotterClusterRole,
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
				Verbs:     []string{"create", "get", "list", "watch", "patch"},
			},
		}
		return in
	})
	return err
}

func ensureVolumeSnapshotterJobRoleBinding(kubeClient kubernetes.Interface, resource *core.ObjectReference, sa string, labels map[string]string) error {

	meta := metav1.ObjectMeta{
		Namespace: resource.Namespace,
		Name:      getVolumesnapshotterJobRoleBindingName(resource.Name),
		Labels:    labels,
	}
	_, _, err := rbac_util.CreateOrPatchRoleBinding(kubeClient, meta, func(in *rbac.RoleBinding) *rbac.RoleBinding {
		core_util.EnsureOwnerReference(&in.ObjectMeta, resource)

		in.RoleRef = rbac.RoleRef{
			APIGroup: rbac.GroupName,
			Kind:     KindClusterRole,
			Name:     VolumeSnapshotterClusterRole,
		}
		in.Subjects = []rbac.Subject{
			{
				Kind:      rbac.ServiceAccountKind,
				Name:      sa,
				Namespace: resource.Namespace,
			},
		}
		return in
	})
	return err
}

func getVolumesnapshotterJobRoleBindingName(name string) string {
	return name + "-" + VolumeSnapshotterClusterRole
}

func EnsureVolumeSnapshotRestorerJobRBAC(kubeClient kubernetes.Interface, ref *core.ObjectReference, sa string, labels map[string]string) error {
	// ensure ClusterRole for restore job
	err := ensureVolumeSnapshotRestorerJobClusterRole(kubeClient, labels)
	if err != nil {
		return err
	}

	// ensure RoleBinding for restore job
	err = ensureVolumeSnapshotRestorerJobRoleBinding(kubeClient, ref, sa, labels)
	if err != nil {
		return err
	}

	//ensure storageClass ClusterRole for restore job
	err = ensureStorageReaderClassClusterRole(kubeClient, labels)
	if err != nil {
		return err
	}

	//ensure storageClass ClusterRoleBinding for restore job
	err = ensureStorageClassReaderClusterRoleBinding(kubeClient, ref, sa, labels)
	if err != nil {
		return err
	}

	return nil
}

func ensureVolumeSnapshotRestorerJobClusterRole(kubeClient kubernetes.Interface, labels map[string]string) error {

	meta := metav1.ObjectMeta{
		Name:   VolumeSnapshotRestorerClusterRole,
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
		}
		return in

	})
	return err
}

func ensureVolumeSnapshotRestorerJobRoleBinding(kubeClient kubernetes.Interface, resource *core.ObjectReference, sa string, labels map[string]string) error {

	meta := metav1.ObjectMeta{
		Namespace: resource.Namespace,
		Name:      getVolumeSnapshotRestorerJobRoleBindingName(resource.Name),
		Labels:    labels,
	}
	_, _, err := rbac_util.CreateOrPatchRoleBinding(kubeClient, meta, func(in *rbac.RoleBinding) *rbac.RoleBinding {
		core_util.EnsureOwnerReference(&in.ObjectMeta, resource)

		in.RoleRef = rbac.RoleRef{
			APIGroup: rbac.GroupName,
			Kind:     "ClusterRole",
			Name:     VolumeSnapshotRestorerClusterRole,
		}
		in.Subjects = []rbac.Subject{
			{
				Kind:      rbac.ServiceAccountKind,
				Name:      sa,
				Namespace: resource.Namespace,
			},
		}
		return in
	})
	return err
}

func getVolumeSnapshotRestorerJobRoleBindingName(name string) string {
	return name + "-" + VolumeSnapshotRestorerClusterRole
}

func ensureStorageReaderClassClusterRole(kubeClient kubernetes.Interface, labels map[string]string) error {

	meta := metav1.ObjectMeta{
		Name:   StorageClassReaderClusterRole,
		Labels: labels,
	}
	_, _, err := rbac_util.CreateOrPatchClusterRole(kubeClient, meta, func(in *rbac.ClusterRole) *rbac.ClusterRole {

		in.Rules = []rbac.PolicyRule{
			{
				APIGroups: []string{storage_api_v1.GroupName},
				Resources: []string{"storageclasses"},
				Verbs:     []string{"get"},
			},
		}
		return in

	})
	return err
}

func ensureStorageClassReaderClusterRoleBinding(kubeClient kubernetes.Interface, resource *core.ObjectReference, sa string, labels map[string]string) error {

	meta := metav1.ObjectMeta{
		Name:      getStorageClassReaderClusterRoleBindingName(resource.Name),
		Namespace: resource.Namespace,
		Labels:    labels,
	}
	_, _, err := rbac_util.CreateOrPatchClusterRoleBinding(kubeClient, meta, func(in *rbac.ClusterRoleBinding) *rbac.ClusterRoleBinding {
		core_util.EnsureOwnerReference(&in.ObjectMeta, resource)

		in.RoleRef = rbac.RoleRef{
			APIGroup: rbac.GroupName,
			Kind:     "ClusterRole",
			Name:     StorageClassReaderClusterRole,
		}
		in.Subjects = []rbac.Subject{
			{
				Kind:      rbac.ServiceAccountKind,
				Name:      sa,
				Namespace: resource.Namespace,
			},
		}
		return in
	})
	return err
}

func getStorageClassReaderClusterRoleBindingName(name string) string {
	return name + "-" + StorageClassReaderClusterRole
}

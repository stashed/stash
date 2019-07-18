package rbac

import (
	"strings"

	"github.com/appscode/go/log"
	core "k8s.io/api/core/v1"
	rbac "k8s.io/api/rbac/v1"
	kerr "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	core_util "kmodules.xyz/client-go/core/v1"
	rbac_util "kmodules.xyz/client-go/rbac/v1"
	wapi "kmodules.xyz/webhook-runtime/apis/workload/v1"
	api "stash.appscode.dev/stash/apis/stash/v1alpha1"
	api_v1beta1 "stash.appscode.dev/stash/apis/stash/v1beta1"
)

func EnsureRestoreInitContainerRBAC(kubeClient kubernetes.Interface, ref *core.ObjectReference, sa string, labels map[string]string) error {
	// ensure ClusterRole for restore init container
	err := ensureRestoreInitContainerClusterRole(kubeClient, labels)
	if err != nil {
		return err
	}

	// ensure RoleBinding for restore init container
	err = ensureRestoreInitContainerRoleBinding(kubeClient, ref, sa, labels)
	if err != nil {
		return err
	}

	return nil
}

func ensureRestoreInitContainerClusterRole(kubeClient kubernetes.Interface, labels map[string]string) error {
	meta := metav1.ObjectMeta{
		Name:   RestoreInitContainerClusterRole,
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
				APIGroups: []string{api.SchemeGroupVersion.Group},
				Resources: []string{"*"},
				Verbs:     []string{"*"},
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
		}
		return in
	})
	return err
}

func ensureRestoreInitContainerRoleBinding(kubeClient kubernetes.Interface, resource *core.ObjectReference, sa string, labels map[string]string) error {
	meta := metav1.ObjectMeta{
		Namespace: resource.Namespace,
		Name:      getRestoreInitContainerRoleBindingName(resource.Name, resource.Kind),
		Labels:    labels,
	}
	_, _, err := rbac_util.CreateOrPatchRoleBinding(kubeClient, meta, func(in *rbac.RoleBinding) *rbac.RoleBinding {
		core_util.EnsureOwnerReference(&in.ObjectMeta, resource)

		if in.Annotations == nil {
			in.Annotations = map[string]string{}
		}

		in.RoleRef = rbac.RoleRef{
			APIGroup: rbac.GroupName,
			Kind:     "ClusterRole",
			Name:     RestoreInitContainerClusterRole,
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

func getRestoreInitContainerRoleBindingName(name string, kind string) string {
	return strings.ToLower(kind) + "-" + name + "-" + RestoreInitContainerClusterRole
}

func ensureRestoreInitContainerRoleBindingDeleted(kubeClient kubernetes.Interface, w *wapi.Workload) error {
	err := kubeClient.RbacV1().RoleBindings(w.Namespace).Delete(
		getRestoreInitContainerRoleBindingName(w.Name, w.Kind),
		&metav1.DeleteOptions{},
	)
	if err != nil && !kerr.IsNotFound(err) {
		return err
	}
	if err == nil {
		log.Infof("RoleBinding %s/%s has been deleted", w.Namespace, getRestoreInitContainerRoleBindingName(w.Name, w.Kind))
	}
	return nil
}

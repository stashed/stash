package rbac

import (
	"fmt"

	core "k8s.io/api/core/v1"
	policy "k8s.io/api/policy/v1beta1"
	rbac "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	core_util "kmodules.xyz/client-go/core/v1"
	rbac_util "kmodules.xyz/client-go/rbac/v1"
	appCatalog "kmodules.xyz/custom-resources/apis/appcatalog/v1alpha1"
	api_v1alpha1 "stash.appscode.dev/stash/apis/stash/v1alpha1"
	api_v1beta1 "stash.appscode.dev/stash/apis/stash/v1beta1"
)

const (
	RestoreJobClusterRole = "stash-restore-job"
)

func EnsureRestoreJobRBAC(kubeClient kubernetes.Interface, ref *core.ObjectReference, sa string, psps []string, labels map[string]string) error {
	// ensure ClusterRole for restore job
	err := ensureRestoreJobClusterRole(kubeClient, psps, labels)
	if err != nil {
		return err
	}

	// ensure RoleBinding for restore job
	err = ensureRestoreJobRoleBinding(kubeClient, ref, sa, labels)
	if err != nil {
		return err
	}

	return nil
}

func ensureRestoreJobClusterRole(kubeClient kubernetes.Interface, psps []string, labels map[string]string) error {

	meta := metav1.ObjectMeta{
		Name:   RestoreJobClusterRole,
		Labels: labels,
	}
	_, _, err := rbac_util.CreateOrPatchClusterRole(kubeClient, meta, func(in *rbac.ClusterRole) *rbac.ClusterRole {

		in.Rules = []rbac.PolicyRule{
			{
				APIGroups: []string{api_v1beta1.SchemeGroupVersion.Group},
				Resources: []string{
					api_v1beta1.ResourcePluralRestoreSession,
					fmt.Sprintf("%s/status", api_v1beta1.ResourcePluralRestoreSession)},
				Verbs: []string{"*"},
			},
			{
				APIGroups: []string{api_v1alpha1.SchemeGroupVersion.Group},
				Resources: []string{api_v1alpha1.ResourcePluralRepository},
				Verbs:     []string{"*"},
			},
			{
				APIGroups: []string{appCatalog.SchemeGroupVersion.Group},
				Resources: []string{appCatalog.ResourceApps},
				Verbs:     []string{"get"},
			},
			{
				APIGroups: []string{core.SchemeGroupVersion.Group},
				Resources: []string{"secrets"},
				Verbs:     []string{"get"},
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
		}
		return in
	})
	return err
}

func ensureRestoreJobRoleBinding(kubeClient kubernetes.Interface, resource *core.ObjectReference, sa string, labels map[string]string) error {

	meta := metav1.ObjectMeta{
		Namespace: resource.Namespace,
		Name:      getRestoreJobRoleBindingName(resource.Name),
		Labels:    labels,
	}
	_, _, err := rbac_util.CreateOrPatchRoleBinding(kubeClient, meta, func(in *rbac.RoleBinding) *rbac.RoleBinding {
		core_util.EnsureOwnerReference(&in.ObjectMeta, resource)

		in.RoleRef = rbac.RoleRef{
			APIGroup: rbac.GroupName,
			Kind:     "ClusterRole",
			Name:     RestoreJobClusterRole,
		}
		in.Subjects = []rbac.Subject{
			{
				Kind:      "ServiceAccount",
				Name:      sa,
				Namespace: resource.Namespace,
			},
		}
		return in
	})
	return err
}

func getRestoreJobRoleBindingName(name string) string {
	return name + "-" + RestoreJobClusterRole
}

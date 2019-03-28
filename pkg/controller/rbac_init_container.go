package controller

import (
	"github.com/appscode/go/log"
	api "github.com/appscode/stash/apis/stash/v1alpha1"
	api_v1beta1 "github.com/appscode/stash/apis/stash/v1beta1"
	"github.com/appscode/stash/pkg/util"
	core "k8s.io/api/core/v1"
	rbac "k8s.io/api/rbac/v1"
	kerr "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	core_util "kmodules.xyz/client-go/core/v1"
	rbac_util "kmodules.xyz/client-go/rbac/v1"
	wapi "kmodules.xyz/webhook-runtime/apis/workload/v1"
)

func (c *StashController) getRestoreInitContainerRoleBindingName(name string) string {
	return name + "-" + RestoreInitContainerClusterRole
}

func (c *StashController) ensureRestoreInitContainerRBAC(ref *core.ObjectReference, sa string) error {
	// ensure ClusterRole for restore init container
	err := c.ensureRestoreInitContainerClusterRole()
	if err != nil {
		return err
	}

	// ensure RoleBinding for restore init container
	err = c.ensureRestoreInitContainerRoleBinding(ref, sa)
	if err != nil {
		return err
	}

	return nil
}

func (c *StashController) ensureRestoreInitContainerClusterRole() error {
	meta := metav1.ObjectMeta{Name: RestoreInitContainerClusterRole}
	_, _, err := rbac_util.CreateOrPatchClusterRole(c.kubeClient, meta, func(in *rbac.ClusterRole) *rbac.ClusterRole {
		if in.Labels == nil {
			in.Labels = map[string]string{}
		}
		in.Labels[util.LabelApp] = util.AppLabelStash

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

func (c *StashController) ensureRestoreInitContainerRoleBinding(resource *core.ObjectReference, sa string) error {
	meta := metav1.ObjectMeta{
		Namespace: resource.Namespace,
		Name:      c.getRestoreInitContainerRoleBindingName(resource.Name),
	}
	_, _, err := rbac_util.CreateOrPatchRoleBinding(c.kubeClient, meta, func(in *rbac.RoleBinding) *rbac.RoleBinding {
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

func (c *StashController) ensureRestoreInitContainerRoleBindingDeleted(w *wapi.Workload) error {
	err := c.kubeClient.RbacV1().RoleBindings(w.Namespace).Delete(
		c.getRestoreInitContainerRoleBindingName(w.Name),
		&metav1.DeleteOptions{},
	)
	if err != nil && !kerr.IsNotFound(err) {
		return err
	}
	if err == nil {
		log.Infof("RoleBinding %s/%s has been deleted", w.Namespace, c.getRestoreInitContainerRoleBindingName(w.Name))
	}
	return nil
}

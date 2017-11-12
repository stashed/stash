package controller

import (
	"github.com/appscode/go/log"
	"github.com/appscode/go/types"
	rbac_util "github.com/appscode/kutil/rbac/v1beta1"
	api "github.com/appscode/stash/apis/stash/v1alpha1"
	apps "k8s.io/api/apps/v1beta1"
	core "k8s.io/api/core/v1"
	extensions "k8s.io/api/extensions/v1beta1"
	rbac "k8s.io/api/rbac/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	sidecarClusterRole = "stash-sidecar"
)

func (c *StashController) getRoleBindingName(name string) string {
	return name + "-" + sidecarClusterRole
}

func (c *StashController) ensureOwnerReference(rb metav1.ObjectMeta, resource *core.ObjectReference) metav1.ObjectMeta {
	fi := -1
	for i, ref := range rb.OwnerReferences {
		if ref.Kind == ref.Kind && ref.Name == ref.Name {
			fi = i
			break
		}
	}
	if fi == -1 {
		rb.OwnerReferences = append(rb.OwnerReferences, metav1.OwnerReference{})
		fi = len(rb.OwnerReferences) - 1
	}
	rb.OwnerReferences[fi].APIVersion = resource.APIVersion
	rb.OwnerReferences[fi].Kind = resource.Kind
	rb.OwnerReferences[fi].Name = resource.Name
	rb.OwnerReferences[fi].UID = resource.UID
	rb.OwnerReferences[fi].BlockOwnerDeletion = types.TrueP()
	return rb
}

func (c *StashController) ensureRoleBinding(resource *core.ObjectReference, sa string) error {
	meta := metav1.ObjectMeta{
		Namespace: resource.Namespace,
		Name:      c.getRoleBindingName(resource.Name),
	}
	_, err := rbac_util.CreateOrPatchRoleBinding(c.k8sClient, meta, func(in *rbac.RoleBinding) *rbac.RoleBinding {
		in.ObjectMeta = c.ensureOwnerReference(in.ObjectMeta, resource)

		if in.Annotations == nil {
			in.Annotations = map[string]string{}
		}

		in.RoleRef = rbac.RoleRef{
			APIGroup: rbac.GroupName,
			Kind:     "ClusterRole",
			Name:     sidecarClusterRole,
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

func (c *StashController) ensureRoleBindingDeleted(resource metav1.ObjectMeta) error {
	log.Infof("Deleting RoleBinding %s/%s", resource.Namespace, c.getRoleBindingName(resource.Name))
	return c.k8sClient.RbacV1beta1().
		RoleBindings(resource.Namespace).
		Delete(c.getRoleBindingName(resource.Name), &metav1.DeleteOptions{})
}

func (c *StashController) ensureSidecarClusterRole() error {
	meta := metav1.ObjectMeta{Name: sidecarClusterRole}
	_, err := rbac_util.CreateOrPatchClusterRole(c.k8sClient, meta, func(in *rbac.ClusterRole) *rbac.ClusterRole {
		if in.Labels == nil {
			in.Labels = map[string]string{}
		}
		in.Labels["app"] = "stash"

		in.Rules = []rbac.PolicyRule{
			{
				APIGroups: []string{api.SchemeGroupVersion.Group},
				Resources: []string{"*"},
				Verbs:     []string{"*"},
			},
			{
				APIGroups: []string{apps.GroupName},
				Resources: []string{"deployments"},
				Verbs:     []string{"get"},
			},
			{
				APIGroups: []string{extensions.GroupName},
				Resources: []string{"daemonsets", "replicasets"},
				Verbs:     []string{"get"},
			},
			{
				APIGroups: []string{core.GroupName},
				Resources: []string{"replicationcontrollers", "secrets"},
				Verbs:     []string{"get"},
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

package plugin

import (
	"github.com/appscode/go/log"
	rbac_util "github.com/appscode/kutil/rbac/v1"
	core "k8s.io/api/core/v1"
	rbac "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	SidecarClusterRole = "stash-sidecar"
)

func (c *MutatorOptions) getSidecarRoleBindingName(name string) string {
	return name + "-" + SidecarClusterRole
}

func (c MutatorOptions) ensureSidecarRoleBinding(resource *core.ObjectReference, sa string) error {
	meta := metav1.ObjectMeta{
		Namespace: resource.Namespace,
		Name:      c.getSidecarRoleBindingName(resource.Name),
	}
	_, _, err := rbac_util.CreateOrPatchRoleBinding(c.KubeClient, meta, func(in *rbac.RoleBinding) *rbac.RoleBinding {
		//in.ObjectMeta = core_util.EnsureOwnerReference(in.ObjectMeta, resource)

		if in.Annotations == nil {
			in.Annotations = map[string]string{}
		}

		in.RoleRef = rbac.RoleRef{
			APIGroup: rbac.GroupName,
			Kind:     "ClusterRole",
			Name:     SidecarClusterRole,
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

func (c *MutatorOptions) ensureSidecarRoleBindingDeleted(resource metav1.ObjectMeta) error {
	log.Infof("Deleting RoleBinding %s/%s", resource.Namespace, c.getSidecarRoleBindingName(resource.Name))
	return c.KubeClient.RbacV1().
		RoleBindings(resource.Namespace).
		Delete(c.getSidecarRoleBindingName(resource.Name), &metav1.DeleteOptions{})
}

package controller

import (
	"github.com/appscode/go/log"
	core_util "github.com/appscode/kutil/core/v1"
	rbac_util "github.com/appscode/kutil/rbac/v1"
	api "github.com/appscode/stash/apis/stash/v1alpha1"
	"github.com/appscode/stash/client/clientset/versioned/scheme"
	apps "k8s.io/api/apps/v1beta1"
	batch "k8s.io/api/batch/v1"
	core "k8s.io/api/core/v1"
	extensions "k8s.io/api/extensions/v1beta1"
	rbac "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/reference"
)

const (
	SidecarClusterRole = "stash-sidecar"
	ScaledownJobRole   = "stash-scaledownjob"
)

func (c *StashController) getSidecarRoleBindingName(name string) string {
	return name + "-" + SidecarClusterRole
}

func (c *StashController) ensureSidecarRoleBinding(resource *core.ObjectReference, sa string) error {
	meta := metav1.ObjectMeta{
		Namespace: resource.Namespace,
		Name:      c.getSidecarRoleBindingName(resource.Name),
	}
	_, _, err := rbac_util.CreateOrPatchRoleBinding(c.kubeClient, meta, func(in *rbac.RoleBinding) *rbac.RoleBinding {
		in.ObjectMeta = core_util.EnsureOwnerReference(in.ObjectMeta, resource)

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

func (c *StashController) ensureSidecarRoleBindingDeleted(resource metav1.ObjectMeta) error {
	log.Infof("Deleting RoleBinding %s/%s", resource.Namespace, c.getSidecarRoleBindingName(resource.Name))
	return c.kubeClient.RbacV1().
		RoleBindings(resource.Namespace).
		Delete(c.getSidecarRoleBindingName(resource.Name), &metav1.DeleteOptions{})
}

func (c *StashController) ensureSidecarClusterRole() error {
	meta := metav1.ObjectMeta{Name: SidecarClusterRole}
	_, _, err := rbac_util.CreateOrPatchClusterRole(c.kubeClient, meta, func(in *rbac.ClusterRole) *rbac.ClusterRole {
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
				Resources: []string{"deployments", "statefulsets"},
				Verbs:     []string{"get", "list", "patch"},
			},
			{
				APIGroups: []string{extensions.GroupName},
				Resources: []string{"daemonsets", "replicasets"},
				Verbs:     []string{"get", "list", "patch"},
			},
			{
				APIGroups: []string{core.GroupName},
				Resources: []string{"replicationcontrollers"},
				Verbs:     []string{"get", "list", "patch"},
			},
			{
				APIGroups: []string{core.GroupName},
				Resources: []string{"secrets"},
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
			{
				APIGroups: []string{batch.GroupName},
				Resources: []string{"jobs"},
				Verbs:     []string{"create", "get"},
			},
			{
				APIGroups: []string{rbac.GroupName},
				Resources: []string{"clusterroles", "roles", "rolebindings"},
				Verbs:     []string{"get", "create"},
			},
			{
				APIGroups: []string{core.GroupName},
				Resources: []string{"serviceaccounts"},
				Verbs:     []string{"get", "create"},
			},
		}
		return in
	})
	return err
}

// use scaledownjob-role, service-account and role-binding name same as job name
// set job as owner of role, service-account and role-binding
func (c *StashController) ensureScaledownJobRBAC(resource *core.ObjectReference) error {
	// ensure roles
	meta := metav1.ObjectMeta{
		Name:      ScaledownJobRole,
		Namespace: resource.Namespace,
	}
	_, _, err := rbac_util.CreateOrPatchRole(c.kubeClient, meta, func(in *rbac.Role) *rbac.Role {
		in.ObjectMeta = core_util.EnsureOwnerReference(in.ObjectMeta, resource)

		if in.Labels == nil {
			in.Labels = map[string]string{}
		}
		in.Labels["app"] = "stash"

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
				APIGroups: []string{extensions.GroupName},
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
		Name:      resource.Name,
		Namespace: resource.Namespace,
	}
	_, _, err = core_util.CreateOrPatchServiceAccount(c.kubeClient, meta, func(in *core.ServiceAccount) *core.ServiceAccount {
		in.ObjectMeta = core_util.EnsureOwnerReference(in.ObjectMeta, resource)
		if in.Labels == nil {
			in.Labels = map[string]string{}
		}
		in.Labels["app"] = "stash"
		return in
	})
	if err != nil {
		return err
	}

	// ensure role binding
	_, _, err = rbac_util.CreateOrPatchRoleBinding(c.kubeClient, meta, func(in *rbac.RoleBinding) *rbac.RoleBinding {
		in.ObjectMeta = core_util.EnsureOwnerReference(in.ObjectMeta, resource)

		if in.Labels == nil {
			in.Labels = map[string]string{}
		}
		in.Labels["app"] = "stash"

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
func (c *StashController) ensureRecoveryRBAC(resource *core.ObjectReference) error {
	// ensure service account
	meta := metav1.ObjectMeta{
		Name:      resource.Name,
		Namespace: resource.Namespace,
	}
	_, _, err := core_util.CreateOrPatchServiceAccount(c.kubeClient, meta, func(in *core.ServiceAccount) *core.ServiceAccount {
		in.ObjectMeta = core_util.EnsureOwnerReference(in.ObjectMeta, resource)
		if in.Labels == nil {
			in.Labels = map[string]string{}
		}
		in.Labels["app"] = "stash"
		return in
	})
	if err != nil {
		return err
	}

	// ensure role binding
	_, _, err = rbac_util.CreateOrPatchRoleBinding(c.kubeClient, meta, func(in *rbac.RoleBinding) *rbac.RoleBinding {
		in.ObjectMeta = core_util.EnsureOwnerReference(in.ObjectMeta, resource)

		if in.Labels == nil {
			in.Labels = map[string]string{}
		}
		in.Labels["app"] = "stash"

		in.RoleRef = rbac.RoleRef{
			APIGroup: rbac.GroupName,
			Kind:     "ClusterRole",
			Name:     SidecarClusterRole,
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

func getRepoReaderRoleName(repoName string) string {
	return "appscode:stash:repo-reader:" + repoName
}

func getRepoReaderRole(repo *api.Repository) (*rbac.Role, error) {
	role := &rbac.Role{
		ObjectMeta: metav1.ObjectMeta{
			Name:      getRepoReaderRoleName(repo.Name),
			Namespace: repo.Namespace,
		},

		Rules: []rbac.PolicyRule{
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
		},
	}

	ref, err := reference.GetReference(scheme.Scheme, repo)
	if err != nil {
		return nil, err
	}
	role.ObjectMeta = core_util.EnsureOwnerReference(role.ObjectMeta, ref)

	return role, nil
}

func (c *StashController) ensureRepoReaderRole(repo *api.Repository) error {
	repoReaderRole, err := getRepoReaderRole(repo)
	if err != nil {
		return err
	}
	_, _, err = rbac_util.CreateOrPatchRole(c.kubeClient, repoReaderRole.ObjectMeta, func(in *rbac.Role) *rbac.Role {
		in = repoReaderRole
		return in
	})
	return err
}

func (c *StashController) ensureRepoReaderRBAC(resource *core.ObjectReference, rec *api.Recovery) error {
	meta := metav1.ObjectMeta{
		Name:      resource.Name + ":repo-reader",
		Namespace: rec.Spec.Repository.Namespace,
	}

	repo, err := c.stashClient.StashV1alpha1().Repositories(rec.Spec.Repository.Namespace).Get(rec.Spec.Repository.Name, metav1.GetOptions{})
	if err != nil {
		return err
	}

	// ensure repo-reader role
	err = c.ensureRepoReaderRole(repo)
	if err != nil {
		return err
	}

	// ensure repo-reader role binding
	_, _, err = rbac_util.CreateOrPatchRoleBinding(c.kubeClient, meta, func(in *rbac.RoleBinding) *rbac.RoleBinding {
		in.ObjectMeta = core_util.EnsureOwnerReference(in.ObjectMeta, resource)

		if in.Labels == nil {
			in.Labels = map[string]string{}
		}
		in.Labels["app"] = "stash"

		in.RoleRef = rbac.RoleRef{
			APIGroup: rbac.GroupName,
			Kind:     "Role",
			Name:     getRepoReaderRoleName(rec.Spec.Repository.Name),
		}

		in.Subjects = []rbac.Subject{
			{
				Kind:      "ServiceAccount",
				Name:      resource.Name,
				Namespace: resource.Namespace,
			},
		}
		return in
	})
	return err
}

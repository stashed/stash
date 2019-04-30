package controller

import (
	"fmt"
	"strings"

	"github.com/appscode/stash/apis"
	api_v1alpha1 "github.com/appscode/stash/apis/stash/v1alpha1"
	api_v1beta1 "github.com/appscode/stash/apis/stash/v1beta1"
	stash_scheme "github.com/appscode/stash/client/clientset/versioned/scheme"
	"github.com/appscode/stash/pkg/util"
	"github.com/golang/glog"
	apps "k8s.io/api/apps/v1"
	core "k8s.io/api/core/v1"
	rbac "k8s.io/api/rbac/v1"
	kerr "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/clientcmd/api"
	"k8s.io/client-go/tools/reference"
	core_util "kmodules.xyz/client-go/core/v1"
	meta_util "kmodules.xyz/client-go/meta"
	rbac_util "kmodules.xyz/client-go/rbac/v1"
	appCatalog "kmodules.xyz/custom-resources/apis/appcatalog/v1alpha1"
)

const (
	SidecarClusterRole              = "stash-sidecar"
	ScaledownJobRole                = "stash-scaledownjob"
	RestoreInitContainerClusterRole = "stash-restore-init-container"
	RestoreJobClusterRole           = "stash-restore-job"
	BackupJobClusterRole            = "stash-backup-job"
	CronJobClusterRole              = "stash-cron-job"
	KindRole                        = "Role"
	KindClusterRole                 = "ClusterRole"
)

func (c *StashController) getBackupJobRoleBindingName(name string) string {
	return name + "-" + BackupJobClusterRole
}

func (c *StashController) getRestoreJobRoleBindingName(name string) string {
	return name + "-" + RestoreJobClusterRole
}

func (c *StashController) ensureCronJobRBAC(resource *core.ObjectReference, sa string) error {
	// ensure CronJob cluster role
	err := c.ensureCronJobClusterRole()
	if err != nil {
		return err
	}

	// ensure RoleBinding
	err = c.ensureCronJobRoleBinding(resource, sa)
	return nil
}

func (c *StashController) ensureCronJobClusterRole() error {
	meta := metav1.ObjectMeta{
		Name: CronJobClusterRole,
	}
	_, _, err := rbac_util.CreateOrPatchClusterRole(c.kubeClient, meta, func(in *rbac.ClusterRole) *rbac.ClusterRole {
		if in.Labels == nil {
			in.Labels = map[string]string{}
		}
		in.Labels[util.LabelApp] = util.AppLabelStash
		in.Rules = []rbac.PolicyRule{
			{
				APIGroups: []string{api_v1beta1.SchemeGroupVersion.Group},
				Resources: []string{api_v1beta1.ResourcePluralBackupSession},
				Verbs:     []string{"*"},
			},
			{
				APIGroups: []string{api_v1beta1.SchemeGroupVersion.Group},
				Resources: []string{api_v1beta1.ResourcePluralBackupConfiguration},
				Verbs:     []string{"*"},
			},
		}
		return in

	})
	return err
}

func (c *StashController) ensureCronJobRoleBinding(resource *core.ObjectReference, sa string) error {
	meta := metav1.ObjectMeta{
		Name:      resource.Name,
		Namespace: resource.Namespace,
	}

	bcObj, err := c.stashClient.StashV1beta1().BackupConfigurations(resource.Namespace).Get(resource.Name, metav1.GetOptions{})
	if err != nil {
		return err
	}

	exists, objRef := c.checkAndGetWorrkloadReference(bcObj)

	// ensure role binding
	_, _, err = rbac_util.CreateOrPatchRoleBinding(c.kubeClient, meta, func(in *rbac.RoleBinding) *rbac.RoleBinding {
		if exists == true {
			core_util.EnsureOwnerReference(&in.ObjectMeta, objRef)
		}
		if in.Labels == nil {
			in.Labels = map[string]string{}
		}
		in.Labels[util.LabelApp] = util.AppLabelStash

		in.RoleRef = rbac.RoleRef{
			APIGroup: rbac.GroupName,
			Kind:     KindClusterRole,
			Name:     CronJobClusterRole,
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
	if err != nil {
		return err
	}
	return nil
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
		core_util.EnsureOwnerReference(&in.ObjectMeta, resource)

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
		Name:      resource.Name,
		Namespace: resource.Namespace,
	}
	_, _, err = core_util.CreateOrPatchServiceAccount(c.kubeClient, meta, func(in *core.ServiceAccount) *core.ServiceAccount {
		core_util.EnsureOwnerReference(&in.ObjectMeta, resource)
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
		core_util.EnsureOwnerReference(&in.ObjectMeta, resource)

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
		core_util.EnsureOwnerReference(&in.ObjectMeta, resource)
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
		core_util.EnsureOwnerReference(&in.ObjectMeta, resource)

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

func GetRepoReaderRoleBindingName(name, namespace string) string {
	return name + ":" + namespace + ":repo-reader"
}

func (c *StashController) ensureRepoReaderRole(repo *api_v1alpha1.Repository) error {
	meta := metav1.ObjectMeta{
		Name:      getRepoReaderRoleName(repo.Name),
		Namespace: repo.Namespace,
	}

	ref, err := reference.GetReference(stash_scheme.Scheme, repo)
	if err != nil {
		return err
	}
	_, _, err = rbac_util.CreateOrPatchRole(c.kubeClient, meta, func(in *rbac.Role) *rbac.Role {
		core_util.EnsureOwnerReference(&in.ObjectMeta, ref)

		if in.Labels == nil {
			in.Labels = map[string]string{}
		}
		in.Labels["app"] = "stash"

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

func (c *StashController) ensureRepoReaderRBAC(resource *core.ObjectReference, rec *api_v1alpha1.Recovery) error {
	meta := metav1.ObjectMeta{
		Name:      GetRepoReaderRoleBindingName(resource.Name, resource.Namespace),
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

func (c *StashController) ensureRepoReaderRolebindingDeleted(meta *metav1.ObjectMeta) error {
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
	recovery, err := c.stashClient.StashV1alpha1().Recoveries(meta.Namespace).Get(recoveryName, metav1.GetOptions{})
	if err != nil {
		return err
	}

	// delete role binding
	err = c.kubeClient.RbacV1().RoleBindings(recovery.Spec.Repository.Namespace).Delete(GetRepoReaderRoleBindingName(meta.Name, meta.Namespace), meta_util.DeleteInBackground())
	if err != nil && !kerr.IsNotFound(err) {
		return err
	}
	glog.Infof("Deleted repo-reader rolebinding: " + GetRepoReaderRoleBindingName(meta.Name, meta.Namespace))
	return nil
}

func (c *StashController) ensureRestoreJobRBAC(ref *core.ObjectReference, sa string) error {
	// ensure ClusterRole for restore job
	err := c.ensureRestoreJobClusterRole()
	if err != nil {
		return err
	}

	// ensure RoleBinding for restore job
	err = c.ensureRestoreJobRoleBinding(ref, sa)
	if err != nil {
		return err
	}

	return nil
}

func (c *StashController) ensureRestoreJobClusterRole() error {

	meta := metav1.ObjectMeta{Name: RestoreJobClusterRole}
	_, _, err := rbac_util.CreateOrPatchClusterRole(c.kubeClient, meta, func(in *rbac.ClusterRole) *rbac.ClusterRole {
		if in.Labels == nil {
			in.Labels = map[string]string{}
		}
		in.Labels["app"] = "stash"

		in.Rules = []rbac.PolicyRule{
			{
				APIGroups: []string{api_v1beta1.SchemeGroupVersion.Group},
				Resources: []string{
					api_v1beta1.ResourcePluralRestoreSession,
					fmt.Sprintf("%s/status", api_v1beta1.ResourcePluralRestoreSession)},
				Verbs: []string{"*"},
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

func (c *StashController) ensureRestoreJobRoleBinding(resource *core.ObjectReference, sa string) error {

	meta := metav1.ObjectMeta{
		Namespace: resource.Namespace,
		Name:      c.getRestoreJobRoleBindingName(resource.Name),
	}
	_, _, err := rbac_util.CreateOrPatchRoleBinding(c.kubeClient, meta, func(in *rbac.RoleBinding) *rbac.RoleBinding {
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

func (c *StashController) ensureBackupJobRBAC(ref *core.ObjectReference, sa string) error {
	// ensure ClusterRole for restore job
	err := c.ensureBackupJobClusterRole()
	if err != nil {
		return err
	}

	// ensure RoleBinding for restore job
	err = c.ensureBackupJobRoleBinding(ref, sa)
	if err != nil {
		return err
	}

	return nil
}

func (c *StashController) ensureBackupJobClusterRole() error {

	meta := metav1.ObjectMeta{Name: BackupJobClusterRole}
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
				APIGroups: []string{api_v1alpha1.SchemeGroupVersion.Group},
				Resources: []string{"*"},
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
		}
		return in
	})
	return err
}

func (c *StashController) ensureBackupJobRoleBinding(resource *core.ObjectReference, sa string) error {

	meta := metav1.ObjectMeta{
		Namespace: resource.Namespace,
		Name:      c.getBackupJobRoleBindingName(resource.Name),
	}
	_, _, err := rbac_util.CreateOrPatchRoleBinding(c.kubeClient, meta, func(in *rbac.RoleBinding) *rbac.RoleBinding {
		core_util.EnsureOwnerReference(&in.ObjectMeta, resource)

		in.RoleRef = rbac.RoleRef{
			APIGroup: rbac.GroupName,
			Kind:     "ClusterRole",
			Name:     BackupJobClusterRole,
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

func (c *StashController) checkAndGetWorrkloadReference(bcObj *api_v1beta1.BackupConfiguration) (bool bool, ref *core.ObjectReference) {
	kind := bcObj.Spec.Target.Ref.Kind
	name := bcObj.Spec.Target.Ref.Name
	switch kind {
	case apis.KindDeployment:
		w, err := c.kubeClient.AppsV1().Deployments(bcObj.Namespace).Get(name, metav1.GetOptions{})
		if err == nil {
			objRef, err := reference.GetReference(stash_scheme.Scheme, w)
			if err == nil {
				return true, objRef
			}
		}
	case apis.KindReplicaSet:
		w, err := c.kubeClient.AppsV1().ReplicaSets(bcObj.Namespace).Get(name, metav1.GetOptions{})
		if err == nil {
			objRef, err := reference.GetReference(stash_scheme.Scheme, w)
			if err == nil {
				return true, objRef
			}
		}
	case apis.KindReplicationController:
		w, err := c.kubeClient.CoreV1().ReplicationControllers(bcObj.Namespace).Get(name, metav1.GetOptions{})
		if err == nil {
			objRef, err := reference.GetReference(stash_scheme.Scheme, w)
			if err == nil {
				return true, objRef
			}
		}
	case apis.KindDaemonSet:
		w, err := c.kubeClient.AppsV1().DaemonSets(bcObj.Namespace).Get(name, metav1.GetOptions{})
		if err == nil {
			objRef, err := reference.GetReference(stash_scheme.Scheme, w)
			if err == nil {
				return true, objRef
			}
		}
	case apis.KindStatefulSet:
		w, err := c.kubeClient.AppsV1().StatefulSets(bcObj.Namespace).Get(name, metav1.GetOptions{})
		if err == nil {
			objRef, err := reference.GetReference(stash_scheme.Scheme, w)
			if err == nil {
				return true, objRef
			}
		}
	}
	return false, nil
}

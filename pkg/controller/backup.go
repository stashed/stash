package controller

import (
	"fmt"
	"strings"

	core "k8s.io/api/core/v1"
	kerr "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/tools/reference"
	core_util "kmodules.xyz/client-go/core/v1"
	meta_util "kmodules.xyz/client-go/meta"
	wapi "kmodules.xyz/webhook-runtime/apis/workload/v1"
	"stash.appscode.dev/stash/apis"
	api_v1alpha1 "stash.appscode.dev/stash/apis/stash/v1alpha1"
	api_v1beta1 "stash.appscode.dev/stash/apis/stash/v1beta1"
	v1alpha1_util "stash.appscode.dev/stash/client/clientset/versioned/typed/stash/v1alpha1/util"
	v1beta1_util "stash.appscode.dev/stash/client/clientset/versioned/typed/stash/v1beta1/util"
	"stash.appscode.dev/stash/pkg/resolve"
	"stash.appscode.dev/stash/pkg/util"
)

// applyBackupAnnotationLogic check if the workload has backup annotations then ensure respective Repository and BackupConfiguration.
func (c *StashController) applyBackupAnnotationLogic(w *wapi.Workload) error {
	targetRef, err := reference.GetReference(scheme.Scheme, w)
	if err != nil {
		return fmt.Errorf("failed to create object reference of %s %s/%s. Reason: %v", w.Kind, w.Namespace, w.Namespace, err)
	}
	// if workload has backup annotations then ensure respective Repository and BackupConfiguration
	if meta_util.HasKey(w.Annotations, api_v1beta1.KeyBackupConfigurationTemplate) &&
		meta_util.HasKey(w.Annotations, api_v1beta1.KeyTargetDirectories) &&
		meta_util.HasKey(w.Annotations, api_v1beta1.KeyVolumeMounts) {
		// backup annotations found. so, we have to ensure Repository and BackupConfiguration from BackupConfigurationTemplate
		backupTemplateName, err := meta_util.GetStringValue(w.Annotations, api_v1beta1.KeyBackupConfigurationTemplate)
		if err != nil {
			return err
		}
		directories, err := meta_util.GetStringValue(w.Annotations, api_v1beta1.KeyTargetDirectories)
		if err != nil {
			return err
		}

		v, err := meta_util.GetStringValue(w.Annotations, api_v1beta1.KeyVolumeMounts)
		if err != nil {
			return err
		}
		// extract volume and mount information from volumeMount annotation
		mounts := strings.Split(v, ",")
		volumeMounts := []core.VolumeMount{}
		for _, m := range mounts {
			vol := strings.Split(m, ":")
			if len(vol) == 2 {
				volumeMounts = append(volumeMounts, core.VolumeMount{Name: vol[0], MountPath: vol[1]})
			} else {
				return fmt.Errorf("invalid volume-mounts annotations. use 'vol1Name:mountPath,vol2Name:mountPath' format")
			}
		}
		// read respective BackupConfigurationTemplate crd
		backupTemplate, err := c.stashClient.StashV1beta1().BackupConfigurationTemplates().Get(backupTemplateName, metav1.GetOptions{})
		if err != nil {
			return err
		}

		// resolve BackupConfigurationTemplate's variables
		inputs := make(map[string]string, 0)
		inputs[apis.TargetAPIVersion] = w.APIVersion
		inputs[apis.TargetKind] = strings.ToLower(w.Kind)
		inputs[apis.TargetName] = w.Name
		inputs[apis.TargetNamespace] = w.Namespace

		err = resolve.ResolveBackend(&backupTemplate.Spec.Backend, inputs)
		if err != nil {
			return err
		}

		// ensure Repository crd
		err = c.ensureRepository(backupTemplate, targetRef)
		if err != nil {
			return err
		}

		// ensure BackupConfiguration crd
		err = c.ensureBackupConfiguration(backupTemplate, strings.Split(directories, ","), volumeMounts, targetRef)
		if err != nil {
			return err
		}

	} else {
		// workload does not have backup annotations. it might be removed or was never added.
		// if respective BackupConfiguration exist then backup annotations has been removed.
		// in this case, we have to remove the BackupConfiguration too.
		// however, we will keep Repository crd as it is required for restore.
		_, err := c.stashClient.StashV1beta1().BackupConfigurations(w.Namespace).Get(getBackupConfigurationName(targetRef), metav1.GetOptions{})
		if err != nil && !kerr.IsNotFound(err) {
			return err
		}
		// BackupConfiguration exist. so, we have to remove it.
		err = c.stashClient.StashV1beta1().BackupConfigurations(w.Namespace).Delete(getBackupConfigurationName(targetRef), meta_util.DeleteInBackground())
		if err != nil && !kerr.IsNotFound(err) {
			return err
		}
	}
	return nil
}

func (c *StashController) applyBackupConfigurationLogic(w *wapi.Workload, caller string) (bool, error) {
	// detect old BackupConfiguration from annotations if it does exist.
	oldbc, err := util.GetAppliedBackupConfiguration(w.Annotations)
	if err != nil {
		return false, err
	}
	// find existing BackupConfiguration for this workload
	newbc, err := util.FindBackupConfiguration(c.bcLister, w)
	if err != nil {
		return false, err
	}
	// if BackupConfiguration currently exist for this workload but it is not same as old one,
	// this means BackupConfiguration has been newly created/updated.
	// in this case, we have to add/update sidecar container accordingly.
	if newbc != nil && !util.BackupConfigurationEqual(oldbc, newbc) {
		err := c.ensureBackupSidecar(w, newbc, caller)
		if err != nil {
			return false, err
		}
		return true, nil

	} else if oldbc != nil && newbc == nil {
		// there was BackupConfiguration before but it does not exist now.
		// this means BackupConfiguration has been removed.
		// in this case, we have to delete the backup sidecar container
		// and remove respective annotations from the workload.
		err := c.ensureBackupSidecarDeleted(w, oldbc)
		if err != nil {
			return false, err
		}
		return true, nil
	}
	return false, nil
}

func (c *StashController) applyResticLogic(w *wapi.Workload, caller string) (bool, error) {
	// detect old Restic from annotations if it does exist
	oldRestic, err := util.GetAppliedRestic(w.Annotations)
	if err != nil {
		return false, err
	}

	// find existing Restic for this workload
	newRestic, err := util.FindRestic(c.rstLister, w.ObjectMeta)
	if err != nil {
		return false, err
	}

	// if Restic currently exist for this workload but it is not same as old one,
	// this means Restic has been newly created/updated.
	// in this case, we have to add/update the sidecar container accordingly.
	if newRestic != nil && !util.ResticEqual(oldRestic, newRestic) {
		err := c.ensureWorkloadSidecar(w, newRestic, caller)
		if err != nil {
			return false, err
		}
		return true, nil
	} else if oldRestic != nil && newRestic == nil {
		// there was Restic before but currently does not exist.
		// this means Restic has been removed.
		// in this case, we have to delete the backup sidecar container
		// and remove respective annotations from the workload.
		err := c.ensureWorkloadSidecarDeleted(w, oldRestic)
		if err != nil {
			return false, nil
		}
		return true, nil
	}

	return false, nil
}

func (c *StashController) ensureRepository(backupTemplate *api_v1beta1.BackupConfigurationTemplate, target *core.ObjectReference) error {
	meta := metav1.ObjectMeta{
		Name:      getRepositoryName(target),
		Namespace: target.Namespace,
	}
	_, _, err := v1alpha1_util.CreateOrPatchRepository(c.stashClient.StashV1alpha1(), meta, func(in *api_v1alpha1.Repository) *api_v1alpha1.Repository {
		in.Spec.Backend = backupTemplate.Spec.Backend
		return in
	})
	return err
}

func (c *StashController) ensureBackupConfiguration(backupTemplate *api_v1beta1.BackupConfigurationTemplate, directories []string, volumeMounts []core.VolumeMount, target *core.ObjectReference) error {
	meta := metav1.ObjectMeta{
		Name:      getBackupConfigurationName(target),
		Namespace: target.Namespace,
	}
	_, _, err := v1beta1_util.CreateOrPatchBackupConfiguration(c.stashClient.StashV1beta1(), meta, func(in *api_v1beta1.BackupConfiguration) *api_v1beta1.BackupConfiguration {
		// set workload as owner of this backupConfiguration object
		core_util.EnsureOwnerReference(&in.ObjectMeta, target)
		in.Spec.Repository.Name = getRepositoryName(target)
		in.Spec.Target = &api_v1beta1.BackupTarget{
			Ref: api_v1beta1.TargetRef{
				APIVersion: target.APIVersion,
				Kind:       target.Kind,
				Name:       target.Name,
			},
			Directories:  directories,
			VolumeMounts: volumeMounts,
		}

		in.Spec.Schedule = backupTemplate.Spec.Schedule
		in.Spec.Task = backupTemplate.Spec.Task
		in.Spec.RetentionPolicy = backupTemplate.Spec.RetentionPolicy
		in.Spec.RuntimeSettings = backupTemplate.Spec.RuntimeSettings
		in.Spec.TempDir = backupTemplate.Spec.TempDir
		return in
	})

	return err
}

func getRepositoryName(target *core.ObjectReference) string {
	return strings.ToLower(target.Kind) + "-" + target.Name
}

func getBackupConfigurationName(target *core.ObjectReference) string {
	return strings.ToLower(target.Kind) + "-" + target.Name
}

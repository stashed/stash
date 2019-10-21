package controller

import (
	"fmt"
	"strings"

	"github.com/appscode/go/log"
	core "k8s.io/api/core/v1"
	kerr "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/tools/reference"
	kutil "kmodules.xyz/client-go"
	core_util "kmodules.xyz/client-go/core/v1"
	discovery "kmodules.xyz/client-go/discovery"
	meta_util "kmodules.xyz/client-go/meta"
	wapi "kmodules.xyz/webhook-runtime/apis/workload/v1"
	"stash.appscode.dev/stash/apis"
	api_v1alpha1 "stash.appscode.dev/stash/apis/stash/v1alpha1"
	api_v1beta1 "stash.appscode.dev/stash/apis/stash/v1beta1"
	v1alpha1_util "stash.appscode.dev/stash/client/clientset/versioned/typed/stash/v1alpha1/util"
	v1beta1_util "stash.appscode.dev/stash/client/clientset/versioned/typed/stash/v1beta1/util"
	"stash.appscode.dev/stash/pkg/eventer"
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
	if meta_util.HasKey(w.Annotations, api_v1beta1.KeyBackupBlueprint) &&
		meta_util.HasKey(w.Annotations, api_v1beta1.KeyTargetPaths) &&
		meta_util.HasKey(w.Annotations, api_v1beta1.KeyVolumeMounts) {
		// backup annotations found. so, we have to ensure Repository and BackupConfiguration from BackupBlueprint
		verb, err := c.ensureAutoBackupResourcesForWorkload(w, targetRef)
		if err != nil {
			return c.handleAutoBackupResourcesCreationFailure(targetRef, err)
		}
		if verb != kutil.VerbUnchanged {
			return c.handleAutoBackupResourcesCreationSuccess(targetRef)
		}
	} else {
		// workload does not have backup annotations. it might be removed or was never added.
		// if respective BackupConfiguration exist then backup annotations has been removed.
		// in this case, we have to remove the BackupConfiguration too.
		// however, we will keep Repository crd as it is required for restore.
		verb, err := c.ensureAutoBackupResourcesDeleted(targetRef, w.Namespace, targetRef.Kind)
		if err != nil {
			return c.handleAutoBackupResourcesDeletionFailure(targetRef, err)
		}
		if verb != kutil.VerbUnchanged {
			return c.handleAutoBackupResourcesDeletionSuccess(targetRef)
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
		// write sidecar injection failure/success event
		ref, rerr := util.GetWorkloadReference(w)
		if err != nil && rerr != nil {
			return false, err
		} else if err != nil && rerr == nil {
			return false, c.handleSidecarInjectionFailure(ref, err)
		} else if err == nil && rerr != nil {
			return true, nil
		}
		return true, c.handleSidecarInjectionSuccess(ref)

	} else if oldbc != nil && newbc == nil {
		// there was BackupConfiguration before but it does not exist now.
		// this means BackupConfiguration has been removed.
		// in this case, we have to delete the backup sidecar container
		// and remove respective annotations from the workload.
		c.ensureBackupSidecarDeleted(w)
		// write sidecar deletion failure/success event
		ref, rerr := util.GetWorkloadReference(w)
		if rerr != nil {
			return true, nil
		}
		return true, c.handleSidecarDeletionSuccess(ref)
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
		c.ensureWorkloadSidecarDeleted(w, oldRestic)
		return true, nil
	}

	return false, nil
}

// ensureAutoBackupResources creates(if does not exist) BackupConfiguration and Repository object for the respective workload
func (c *StashController) ensureAutoBackupResourcesForWorkload(w *wapi.Workload, targetRef *core.ObjectReference) (kutil.VerbType, error) {
	backupBlueprintName, err := meta_util.GetStringValue(w.Annotations, api_v1beta1.KeyBackupBlueprint)
	if err != nil {
		return kutil.VerbUnchanged, err
	}
	paths, err := meta_util.GetStringValue(w.Annotations, api_v1beta1.KeyTargetPaths)
	if err != nil {
		return kutil.VerbUnchanged, err
	}

	v, err := meta_util.GetStringValue(w.Annotations, api_v1beta1.KeyVolumeMounts)
	if err != nil {
		return kutil.VerbUnchanged, err
	}
	// extract volume and mount information from volumeMount annotation
	mounts := strings.Split(v, ",")
	volumeMounts := []core.VolumeMount{}
	for _, m := range mounts {
		vol := strings.Split(m, ":")
		if len(vol) == 3 {
			volumeMounts = append(volumeMounts, core.VolumeMount{Name: vol[0], MountPath: vol[1], SubPath: vol[2]})
		} else if len(vol) == 2 {
			volumeMounts = append(volumeMounts, core.VolumeMount{Name: vol[0], MountPath: vol[1]})
		} else {
			return kutil.VerbUnchanged, fmt.Errorf("invalid volume-mounts annotations. use either 'volName:mountPath' or 'volName:mountPath:subPath' format")
		}
	}
	// read respective BackupBlueprint crd
	backupBlueprint, err := c.stashClient.StashV1beta1().BackupBlueprints().Get(backupBlueprintName, metav1.GetOptions{})
	if err != nil {
		return kutil.VerbUnchanged, err
	}

	// resolve BackupBlueprint's variables
	inputs := make(map[string]string)
	inputs[apis.TargetAPIVersion] = w.APIVersion
	inputs[apis.TargetKind] = strings.ToLower(w.Kind)
	inputs[apis.TargetName] = w.Name
	inputs[apis.TargetNamespace] = w.Namespace

	gvr, err := discovery.ResourceForGVK(c.kubeClient.Discovery(), w.GroupVersionKind())
	if err != nil {
		return kutil.VerbUnchanged, err
	}
	inputs[apis.TargetResource] = gvr.Resource

	err = resolve.ResolveBackupBlueprint(backupBlueprint, inputs)
	if err != nil {
		return kutil.VerbUnchanged, err
	}

	// ensure Repository crd
	verb1, err := c.ensureRepository(backupBlueprint, targetRef, targetRef.Kind)
	if err != nil {
		return kutil.VerbUnchanged, err
	}

	// ensure BackupConfiguration crd
	verb2, err := c.ensureBackupConfiguration(backupBlueprint, strings.Split(paths, ","), volumeMounts, targetRef, targetRef.Kind)
	if err != nil {
		return kutil.VerbUnchanged, err
	}
	// if both of the verb is unchanged then no create/update happened to the auto backup resources
	if verb1 == kutil.VerbUnchanged || verb2 == kutil.VerbUnchanged {
		return kutil.VerbUnchanged, nil
	}
	// auto backup resources has been created/updated
	// we will use this information to write event to AppBinding
	// so, "created" or "updated" verb has same effect to the end result
	// we can return any of them.
	return kutil.VerbCreated, nil
}

// ensureAutoBackupResourcesDeleted deletes(if previously created) BackupConfiguration object for the respective resources
func (c *StashController) ensureAutoBackupResourcesDeleted(targetRef *core.ObjectReference, namespace, prefix string) (kutil.VerbType, error) {
	_, err := c.stashClient.StashV1beta1().BackupConfigurations(namespace).Get(getBackupConfigurationName(targetRef, targetRef.Kind), metav1.GetOptions{})
	if err != nil {
		if kerr.IsNotFound(err) {
			return kutil.VerbUnchanged, nil
		}
		return kutil.VerbUnchanged, err
	}
	// BackupConfiguration exist. so, we have to remove it.
	err = c.stashClient.StashV1beta1().BackupConfigurations(namespace).Delete(getBackupConfigurationName(targetRef, prefix), meta_util.DeleteInBackground())
	if err != nil && !kerr.IsNotFound(err) {
		return kutil.VerbUnchanged, err
	}
	return kutil.VerbDeleted, nil
}

func (c *StashController) ensureRepository(backupBlueprint *api_v1beta1.BackupBlueprint, target *core.ObjectReference, prefix string) (kutil.VerbType, error) {
	meta := metav1.ObjectMeta{
		Name:      getRepositoryName(target, prefix),
		Namespace: target.Namespace,
	}
	_, verb, err := v1alpha1_util.CreateOrPatchRepository(c.stashClient.StashV1alpha1(), meta, func(in *api_v1alpha1.Repository) *api_v1alpha1.Repository {
		in.Spec.Backend = backupBlueprint.Spec.Backend
		return in
	})
	if err != nil {
		return kutil.VerbUnchanged, err
	}
	return verb, nil
}

func (c *StashController) ensureBackupConfiguration(backupBlueprint *api_v1beta1.BackupBlueprint, paths []string, volumeMounts []core.VolumeMount, target *core.ObjectReference, prefix string) (kutil.VerbType, error) {
	meta := metav1.ObjectMeta{
		Name:      getBackupConfigurationName(target, prefix),
		Namespace: target.Namespace,
	}
	_, verb, err := v1beta1_util.CreateOrPatchBackupConfiguration(c.stashClient.StashV1beta1(), meta, func(in *api_v1beta1.BackupConfiguration) *api_v1beta1.BackupConfiguration {
		// set workload as owner of this backupConfiguration object
		core_util.EnsureOwnerReference(&in.ObjectMeta, target)
		in.Spec.Repository.Name = getRepositoryName(target, prefix)
		in.Spec.Target = &api_v1beta1.BackupTarget{
			Ref: api_v1beta1.TargetRef{
				APIVersion: target.APIVersion,
				Kind:       target.Kind,
				Name:       target.Name,
			},
			Paths:        paths,
			VolumeMounts: volumeMounts,
		}

		in.Spec.Schedule = backupBlueprint.Spec.Schedule
		in.Spec.Task = backupBlueprint.Spec.Task
		in.Spec.RetentionPolicy = backupBlueprint.Spec.RetentionPolicy
		in.Spec.RuntimeSettings = backupBlueprint.Spec.RuntimeSettings
		in.Spec.TempDir = backupBlueprint.Spec.TempDir
		return in
	})

	if err != nil {
		return kutil.VerbUnchanged, err
	}
	return verb, nil
}

func getRepositoryName(target *core.ObjectReference, prefix string) string {
	return fmt.Sprintf("%s-%s", strings.ToLower(prefix), target.Name)
}

func getBackupConfigurationName(target *core.ObjectReference, prefix string) string {
	return fmt.Sprintf("%s-%s", strings.ToLower(prefix), target.Name)
}

func (c *StashController) handleAutoBackupResourcesCreationFailure(ref *core.ObjectReference, err error) error {
	log.Warningf("Failed to create auto backup resources for %s %s/%s. Reason: %v", ref.Kind, ref.Namespace, ref.Name, err)

	// write event to respective resource
	_, err2 := eventer.CreateEvent(
		c.kubeClient,
		eventer.EventSourceAutoBackupHandler,
		ref,
		core.EventTypeWarning,
		eventer.EventReasonAutoBackupResourcesCreationFailed,
		fmt.Sprintf("Failed to create auto backup resources for %s %s/%s. Reason: %v", ref.Kind, ref.Namespace, ref.Name, err),
	)
	return err2
}

func (c *StashController) handleAutoBackupResourcesCreationSuccess(ref *core.ObjectReference) error {
	log.Infof("Successfully created auto backup resources for %s %s/%s.", ref.Kind, ref.Namespace, ref.Name)

	// write event to respective resource
	_, err2 := eventer.CreateEvent(
		c.kubeClient,
		eventer.EventSourceAutoBackupHandler,
		ref,
		core.EventTypeWarning,
		eventer.EventReasonAutoBackupResourcesCreationSucceeded,
		fmt.Sprintf("Successfully created auto backup resources for %s %s/%s.", ref.Kind, ref.Namespace, ref.Name),
	)
	return err2
}

func (c *StashController) handleAutoBackupResourcesDeletionFailure(ref *core.ObjectReference, err error) error {
	log.Warningf("Failed to delete auto backup resources for %s %s/%s. Reason: %v", ref.Kind, ref.Namespace, ref.Name, err)

	// write event to respective resource
	_, err2 := eventer.CreateEvent(
		c.kubeClient,
		eventer.EventSourceAutoBackupHandler,
		ref,
		core.EventTypeWarning,
		eventer.EventReasonAutoBackupResourcesDeletionFailed,
		fmt.Sprintf("Failed to deleted auto backup resources for %s %s/%s. Reason: %v", ref.Kind, ref.Namespace, ref.Name, err),
	)
	return err2
}

func (c *StashController) handleAutoBackupResourcesDeletionSuccess(ref *core.ObjectReference) error {
	log.Infof("Successfully deleted auto backup resources for %s %s/%s.", ref.Kind, ref.Namespace, ref.Name)

	// write event to respective resource
	_, err2 := eventer.CreateEvent(
		c.kubeClient,
		eventer.EventSourceAutoBackupHandler,
		ref,
		core.EventTypeWarning,
		eventer.EventReasonAutoBackupResourcesDeletionSucceeded,
		fmt.Sprintf("Successfully deleted auto backup resources for %s %s/%s.", ref.Kind, ref.Namespace, ref.Name),
	)
	return err2
}

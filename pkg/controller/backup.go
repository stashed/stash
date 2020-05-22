/*
Copyright The Stash Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controller

import (
	"context"
	"fmt"
	"strings"

	"stash.appscode.dev/apimachinery/apis"
	api_v1alpha1 "stash.appscode.dev/apimachinery/apis/stash/v1alpha1"
	api_v1beta1 "stash.appscode.dev/apimachinery/apis/stash/v1beta1"
	v1alpha1_util "stash.appscode.dev/apimachinery/client/clientset/versioned/typed/stash/v1alpha1/util"
	v1beta1_util "stash.appscode.dev/apimachinery/client/clientset/versioned/typed/stash/v1beta1/util"
	"stash.appscode.dev/stash/pkg/eventer"
	"stash.appscode.dev/stash/pkg/resolve"
	"stash.appscode.dev/stash/pkg/util"

	"github.com/appscode/go/log"
	appsv1 "k8s.io/api/apps/v1"
	core "k8s.io/api/core/v1"
	kerr "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/errors"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/tools/reference"
	kutil "kmodules.xyz/client-go"
	core_util "kmodules.xyz/client-go/core/v1"
	discovery "kmodules.xyz/client-go/discovery"
	meta_util "kmodules.xyz/client-go/meta"
	appcatalog "kmodules.xyz/custom-resources/apis/appcatalog/v1alpha1"
	store "kmodules.xyz/objectstore-api/api/v1"
	ofst "kmodules.xyz/offshoot-api/api/v1"
	ocapps "kmodules.xyz/openshift/apis/apps/v1"
	wapi "kmodules.xyz/webhook-runtime/apis/workload/v1"
)

type autoBackupOptions struct {
	resourceName          string
	resourceNamespace     string
	resourceOwner         *metav1.OwnerReference
	schedule              string
	backend               store.Backend
	target                *api_v1beta1.BackupTarget
	taskRef               api_v1beta1.TaskRef
	retentionPolicy       api_v1alpha1.RetentionPolicy
	runtimeSettings       ofst.RuntimeSettings
	tempDir               api_v1beta1.EmptyDirSettings
	interimVolumeTemplate *ofst.PersistentVolumeClaim
	backupHistoryLimit    *int32
}

// applyBackupAnnotationLogic check if the workload has backup annotations then ensure respective Repository and BackupConfiguration.
func (c *StashController) applyBackupAnnotationLogic(w *wapi.Workload) error {
	targetRef, err := reference.GetReference(scheme.Scheme, w)
	if err != nil {
		return fmt.Errorf("failed to create reference of %s %s/%s. Reason: %v", w.Kind, w.Namespace, w.Namespace, err)
	}
	// if workload has backup annotations then ensure respective Repository and BackupConfiguration
	if meta_util.HasKey(w.Annotations, api_v1beta1.KeyBackupBlueprint) &&
		meta_util.HasKey(w.Annotations, api_v1beta1.KeyTargetPaths) &&
		meta_util.HasKey(w.Annotations, api_v1beta1.KeyVolumeMounts) {
		// backup annotations found. so, we have to ensure Repository and BackupConfiguration from BackupBlueprint
		verb, err := c.ensureAutoBackupResourcesForWorkload(w)
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
		invoker, err := apis.ExtractBackupInvokerInfo(c.stashClient, api_v1beta1.ResourceKindBackupConfiguration, newbc.Name, newbc.Namespace)
		if err != nil {
			return true, err
		}
		for _, targetInfo := range invoker.TargetsInfo {
			if targetInfo.Target != nil &&
				targetInfo.Target.Ref.Kind == w.Kind &&
				targetInfo.Target.Ref.Name == w.Name {
				err = c.ensureBackupSidecar(w, invoker, targetInfo, caller)
				if err != nil {
					return false, c.handleSidecarInjectionFailure(w, invoker, targetInfo.Target.Ref, err)
				}
				return true, c.handleSidecarInjectionSuccess(w, invoker, targetInfo.Target.Ref)
			}
		}

	} else if oldbc != nil && newbc == nil {
		// there was BackupConfiguration before but it does not exist now.
		// this means BackupConfiguration has been removed.
		// in this case, we have to delete the backup sidecar container
		// and remove respective annotations from the workload.
		c.ensureBackupSidecarDeleted(w)
		// write sidecar deletion failure/success event
		return true, c.handleSidecarDeletionSuccess(w)
	}
	return false, nil
}

func (c *StashController) applyBackupBatchLogic(w *wapi.Workload, caller string) (bool, error) {
	// detect old BackupBatch from annotations if it does exist.
	oldbb, err := util.GetAppliedBackupBatch(w.Annotations)
	if err != nil {
		return false, err
	}
	// find existing BackupBatch for this workload
	newbb, err := util.FindBackupBatch(c.backupBatchLister, w)
	if err != nil {
		return false, err
	}
	// if BackupBatch currently exist for this workload but it is not same as old one,
	// this means BackupBatch has been newly created/updated.
	// in this case, we have to add/update sidecar container accordingly.
	if newbb != nil && !util.BackupBatchEqual(oldbb, newbb) {
		for _, member := range newbb.Spec.Members {
			if member.Target != nil && member.Target.Ref.Kind == w.Kind && member.Target.Ref.Name == w.Name {
				invoker, err := apis.ExtractBackupInvokerInfo(c.stashClient, api_v1beta1.ResourceKindBackupBatch, newbb.Name, newbb.Namespace)
				if err != nil {
					return true, err
				}
				for _, targetInfo := range invoker.TargetsInfo {
					if targetInfo.Target != nil &&
						targetInfo.Target.Ref.Kind == w.Kind &&
						targetInfo.Target.Ref.Name == w.Name {
						err = c.ensureBackupSidecar(w, invoker, targetInfo, caller)
						if err != nil {
							return false, c.handleSidecarInjectionFailure(w, invoker, targetInfo.Target.Ref, err)
						}
						// write sidecar injection failure/success event
						return true, c.handleSidecarInjectionSuccess(w, invoker, targetInfo.Target.Ref)
					}
				}
			}
		}
	} else if oldbb != nil && newbb == nil {
		// there was BackupBatch before but it does not exist now.
		// this means BackupBatch has been removed.
		// in this case, we have to delete the backup sidecar container
		// and remove respective annotations from the workload.
		c.ensureBackupSidecarDeleted(w)
		// write sidecar deletion failure/success event
		return true, c.handleSidecarDeletionSuccess(w)
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

// ensureAutoBackupResources creates(if does not exist) BackupConfiguration and Repository wect for the respective workload
func (c *StashController) ensureAutoBackupResourcesForWorkload(w *wapi.Workload) (kutil.VerbType, error) {
	backupBlueprintName, err := meta_util.GetStringValue(w.Annotations, api_v1beta1.KeyBackupBlueprint)
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

	return c.ensureAutoBackupResources(backupBlueprintName, inputs, w)
}

func (c *StashController) ensureAutoBackupResources(blueprintName string, inputs map[string]string, targetObject interface{}) (kutil.VerbType, error) {
	// read respective BackupBlueprint crd
	backupBlueprint, err := c.stashClient.StashV1beta1().BackupBlueprints().Get(context.TODO(), blueprintName, metav1.GetOptions{})
	if err != nil {
		return kutil.VerbUnchanged, err
	}

	err = resolve.ResolveBackupBlueprint(backupBlueprint, inputs)
	if err != nil {
		return kutil.VerbUnchanged, err
	}

	options, err := extractAutoBackupOptions(backupBlueprint, targetObject)
	if err != nil {
		return kutil.VerbUnchanged, err
	}
	// ensure Repository crd
	verb1, err := c.ensureRepository(options)
	if err != nil {
		return kutil.VerbUnchanged, err
	}

	// ensure BackupConfiguration crd
	verb2, err := c.ensureBackupConfiguration(options)
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

func extractAutoBackupOptions(blueprint *api_v1beta1.BackupBlueprint, targetObject interface{}) (autoBackupOptions, error) {
	if blueprint == nil {
		return autoBackupOptions{}, fmt.Errorf("failed to extract autoBackupOptions. Reason: BackupBlueprint is nil")
	}

	options := autoBackupOptions{
		schedule:              blueprint.Spec.Schedule,
		retentionPolicy:       blueprint.Spec.RetentionPolicy,
		taskRef:               blueprint.Spec.Task,
		runtimeSettings:       blueprint.Spec.RuntimeSettings,
		tempDir:               blueprint.Spec.TempDir,
		backend:               blueprint.Spec.Backend,
		interimVolumeTemplate: blueprint.Spec.InterimVolumeTemplate,
		backupHistoryLimit:    blueprint.Spec.BackupHistoryLimit,
	}

	switch w := targetObject.(type) {
	case *appcatalog.AppBinding:
		options.resourceName = meta_util.ValidNameWithPrefix(util.ResourceKindShortForm(w.Kind), w.Name)
		options.resourceNamespace = w.Namespace
		options.target = &api_v1beta1.BackupTarget{
			Ref: api_v1beta1.TargetRef{
				APIVersion: w.APIVersion,
				Kind:       w.Kind,
				Name:       w.Name,
			},
		}
		options.resourceOwner = metav1.NewControllerRef(w, appcatalog.SchemeGroupVersion.WithKind(w.Kind))

	case *core.PersistentVolumeClaim:
		options.resourceName = meta_util.ValidNameWithPrefix(util.ResourceKindShortForm(w.Kind), w.Name)
		options.resourceNamespace = w.Namespace
		options.target = &api_v1beta1.BackupTarget{
			Ref: api_v1beta1.TargetRef{
				APIVersion: w.APIVersion,
				Kind:       w.Kind,
				Name:       w.Name,
			},
		}
		options.resourceOwner = metav1.NewControllerRef(w, core.SchemeGroupVersion.WithKind(w.Kind))

	case *wapi.Workload:
		options.resourceName = meta_util.ValidNameWithPrefix(util.ResourceKindShortForm(w.Kind), w.Name)
		options.resourceNamespace = w.Namespace

		paths, err := meta_util.GetStringValue(w.Annotations, api_v1beta1.KeyTargetPaths)
		if err != nil {
			return options, err
		}

		v, err := meta_util.GetStringValue(w.Annotations, api_v1beta1.KeyVolumeMounts)
		if err != nil {
			return options, err
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
				return options, fmt.Errorf("invalid volume-mounts annotations. use either 'volName:mountPath' or 'volName:mountPath:subPath' format")
			}
		}
		options.target = &api_v1beta1.BackupTarget{
			Ref: api_v1beta1.TargetRef{
				APIVersion: w.APIVersion,
				Kind:       w.Kind,
				Name:       w.Name,
			},
			Paths:        strings.Split(paths, ","),
			VolumeMounts: volumeMounts,
		}
		options.resourceOwner, err = ownerWorkload(w)
		if err != nil {
			return options, fmt.Errorf("failed to extract autoBackupOptions. Reason: %s", err.Error())
		}
	default:
		return options, fmt.Errorf("failed to extract autoBackupOptions. Reason: unknown target wect")
	}
	return options, nil
}

// ensureAutoBackupResourcesDeleted deletes(if previously created) BackupConfiguration wect for the respective resources
func (c *StashController) ensureAutoBackupResourcesDeleted(targetRef *core.ObjectReference, namespace, prefix string) (kutil.VerbType, error) {
	_, err := c.stashClient.StashV1beta1().BackupConfigurations(namespace).Get(context.TODO(), getBackupConfigurationName(targetRef, targetRef.Kind), metav1.GetOptions{})
	if err != nil {
		if kerr.IsNotFound(err) {
			return kutil.VerbUnchanged, nil
		}
		return kutil.VerbUnchanged, err
	}
	// BackupConfiguration exist. so, we have to remove it.
	err = c.stashClient.StashV1beta1().BackupConfigurations(namespace).Delete(context.TODO(), getBackupConfigurationName(targetRef, prefix), meta_util.DeleteInBackground())
	if err != nil && !kerr.IsNotFound(err) {
		return kutil.VerbUnchanged, err
	}
	return kutil.VerbDeleted, nil
}

func (c *StashController) ensureRepository(opt autoBackupOptions) (kutil.VerbType, error) {
	meta := metav1.ObjectMeta{
		Name:      opt.resourceName,
		Namespace: opt.resourceNamespace,
	}
	_, verb, err := v1alpha1_util.CreateOrPatchRepository(
		context.TODO(),
		c.stashClient.StashV1alpha1(),
		meta,
		func(in *api_v1alpha1.Repository) *api_v1alpha1.Repository {
			in.Spec.Backend = opt.backend
			return in
		}, metav1.PatchOptions{},
	)
	if err != nil {
		return kutil.VerbUnchanged, err
	}
	return verb, nil
}

func (c *StashController) ensureBackupConfiguration(opt autoBackupOptions) (kutil.VerbType, error) {
	meta := metav1.ObjectMeta{
		Name:      opt.resourceName,
		Namespace: opt.resourceNamespace,
	}
	_, verb, err := v1beta1_util.CreateOrPatchBackupConfiguration(
		context.TODO(),
		c.stashClient.StashV1beta1(),
		meta,
		func(in *api_v1beta1.BackupConfiguration) *api_v1beta1.BackupConfiguration {
			// set workload as owner of this backupConfiguration wect
			core_util.EnsureOwnerReference(&in.ObjectMeta, opt.resourceOwner)
			in.Spec.Repository.Name = opt.resourceName

			if opt.target != nil {
				in.Spec.Target = &api_v1beta1.BackupTarget{
					Ref: api_v1beta1.TargetRef{
						APIVersion: opt.target.Ref.APIVersion,
						Kind:       opt.target.Ref.Kind,
						Name:       opt.target.Ref.Name,
					},
					Paths:        opt.target.Paths,
					VolumeMounts: opt.target.VolumeMounts,
				}
			}

			in.Spec.Task = opt.taskRef
			in.Spec.TempDir = opt.tempDir
			in.Spec.Schedule = opt.schedule
			in.Spec.RetentionPolicy = opt.retentionPolicy
			in.Spec.RuntimeSettings = opt.runtimeSettings
			in.Spec.BackupHistoryLimit = opt.backupHistoryLimit
			in.Spec.InterimVolumeTemplate = opt.interimVolumeTemplate

			return in
		},
		metav1.PatchOptions{},
	)

	if err != nil {
		return kutil.VerbUnchanged, err
	}
	return verb, nil
}

func getBackupConfigurationName(target *core.ObjectReference, prefix string) string {
	return meta_util.ValidNameWithPrefix(util.ResourceKindShortForm(prefix), target.Name)
}

func (c *StashController) handleAutoBackupResourcesCreationFailure(ref *core.ObjectReference, err error) error {
	if ref == nil {
		return errors.NewAggregate([]error{err, fmt.Errorf("failed to write autobackup resource creation failure event. Reason: provided ObjectReference is nil")})
	}

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
	if ref == nil {
		return fmt.Errorf("failed to write autobackup resource creation success event. Reason: provided ObjectReference is nil")
	}

	log.Infof("Successfully created auto backup resources for %s %s/%s.", ref.Kind, ref.Namespace, ref.Name)

	// write event to respective resource
	_, err2 := eventer.CreateEvent(
		c.kubeClient,
		eventer.EventSourceAutoBackupHandler,
		ref,
		core.EventTypeNormal,
		eventer.EventReasonAutoBackupResourcesCreationSucceeded,
		fmt.Sprintf("Successfully created auto backup resources for %s %s/%s.", ref.Kind, ref.Namespace, ref.Name),
	)
	return err2
}

func (c *StashController) handleAutoBackupResourcesDeletionFailure(ref *core.ObjectReference, err error) error {
	if ref == nil {
		return errors.NewAggregate([]error{err, fmt.Errorf("failed to write autobackup resource deletion failure event. Reason: provided ObjectReference is nil")})
	}
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
	if ref == nil {
		return fmt.Errorf("failed to write autobackup resource creation success event. Reason: provided ObjectReference is nil")
	}

	log.Infof("Successfully deleted auto backup resources for %s %s/%s.", ref.Kind, ref.Namespace, ref.Name)

	// write event to respective resource
	_, err2 := eventer.CreateEvent(
		c.kubeClient,
		eventer.EventSourceAutoBackupHandler,
		ref,
		core.EventTypeNormal,
		eventer.EventReasonAutoBackupResourcesDeletionSucceeded,
		fmt.Sprintf("Successfully deleted auto backup resources for %s %s/%s.", ref.Kind, ref.Namespace, ref.Name),
	)
	return err2
}

func ownerWorkload(w *wapi.Workload) (*metav1.OwnerReference, error) {
	switch w.Kind {
	case apis.KindDeployment, apis.KindStatefulSet, apis.KindDaemonSet, apis.KindReplicaSet:
		return metav1.NewControllerRef(w, appsv1.SchemeGroupVersion.WithKind(w.Kind)), nil
	case apis.KindReplicationController:
		return metav1.NewControllerRef(w, core.SchemeGroupVersion.WithKind(w.Kind)), nil
	case apis.KindDeploymentConfig:
		return metav1.NewControllerRef(w, ocapps.GroupVersion.WithKind(w.Kind)), nil
	default:
		return nil, fmt.Errorf("failed to set workload as owner. Reason: unknown workload kind")
	}
}

package controller

import (
	"fmt"

	stringz "github.com/appscode/go/strings"
	core_util "github.com/appscode/kutil/core/v1"
	"github.com/appscode/kutil/meta"
	workload "github.com/appscode/kutil/workload/v1"
	api "github.com/appscode/stash/apis/stash/v1alpha1"
	"github.com/appscode/stash/pkg/docker"
	"github.com/appscode/stash/pkg/util"
	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/tools/reference"
)

func (c *StashController) ensureWorkloadSidecar(w *workload.Workload, oldRestic, newRestic *api.Restic) error {
	if c.EnableRBAC {
		sa := stringz.Val(w.Spec.ServiceAccountName, "default")
		ref, err := reference.GetReference(scheme.Scheme, w)
		if err != nil {
			ref = &v1.ObjectReference{
				Name:      w.Name,
				Namespace: w.Namespace,
			}
		}
		err = c.ensureSidecarRoleBinding(ref, sa)
		if err != nil {
			return err
		}
	}

	if newRestic.Spec.Backend.StorageSecretName == "" {
		err := fmt.Errorf("missing repository secret name for Restic %s/%s", newRestic.Namespace, newRestic.Name)
		return err
	}

	_, err := c.KubeClient.CoreV1().Secrets(w.Namespace).Get(newRestic.Spec.Backend.StorageSecretName, metav1.GetOptions{})
	if err != nil {
		return err
	}

	image := docker.Docker{
		Registry: c.DockerRegistry,
		Image:    docker.ImageStash,
		Tag:      c.StashImageTag,
	}
	fmt.Println("========================================\nName:", w.Name, "\nKind:", w.Kind, "\n=================================================")

	workload := api.LocalTypedReference{
		Kind: w.Kind,
		Name: w.Name,
	}

	if newRestic.Spec.Type == api.BackupOffline {
		w.Spec.InitContainers = core_util.UpsertContainer(
			w.Spec.InitContainers,
			util.NewInitContainer(newRestic, workload, image, c.EnableRBAC),
		)
	} else {
		w.Spec.Containers = core_util.UpsertContainer(
			w.Spec.Containers,
			util.NewSidecarContainer(newRestic, workload, image),
		)
	}

	// keep existing image pull secrets
	w.Spec.ImagePullSecrets = core_util.MergeLocalObjectReferences(
		w.Spec.ImagePullSecrets,
		newRestic.Spec.ImagePullSecrets,
	)

	w.Spec.Volumes = util.UpsertScratchVolume(w.Spec.Volumes)
	w.Spec.Volumes = util.UpsertDownwardVolume(w.Spec.Volumes)
	w.Spec.Volumes = util.MergeLocalVolume(w.Spec.Volumes, oldRestic, newRestic)

	if w.Annotations == nil {
		w.Annotations = make(map[string]string)
	}
	r := &api.Restic{
		TypeMeta: metav1.TypeMeta{
			APIVersion: api.SchemeGroupVersion.String(),
			Kind:       api.ResourceKindRestic,
		},
		ObjectMeta: newRestic.ObjectMeta,
		Spec:       newRestic.Spec,
	}
	data, _ := meta.MarshalToJson(r, api.SchemeGroupVersion)
	w.Annotations[api.LastAppliedConfiguration] = string(data)
	w.Annotations[api.VersionTag] = c.StashImageTag

	return nil
}

func (c *StashController) ensureWorkloadSidecarDeleted(w *workload.Workload, restic *api.Restic) error {
	if c.EnableRBAC {
		err := c.ensureSidecarRoleBindingDeleted(w.ObjectMeta)
		if err != nil {
			return err
		}
	}

	if restic.Spec.Type == api.BackupOffline {
		w.Spec.InitContainers = core_util.EnsureContainerDeleted(w.Spec.InitContainers, util.StashContainer)
	} else {
		w.Spec.Containers = core_util.EnsureContainerDeleted(w.Spec.Containers, util.StashContainer)
	}

	w.Spec.Volumes = util.EnsureVolumeDeleted(w.Spec.Volumes, util.ScratchDirVolumeName)
	w.Spec.Volumes = util.EnsureVolumeDeleted(w.Spec.Volumes, util.PodinfoVolumeName)

	if restic.Spec.Backend.Local != nil {
		w.Spec.Volumes = util.EnsureVolumeDeleted(w.Spec.Volumes, util.LocalVolumeName)
	}
	if w.Annotations == nil {
		w.Annotations = make(map[string]string)
	}
	delete(w.Annotations, api.LastAppliedConfiguration)
	delete(w.Annotations, api.VersionTag)

	return nil
}

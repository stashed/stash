package controller

import (
	"fmt"

	"github.com/appscode/go/log"
	stringz "github.com/appscode/go/strings"
	api_v1beta1 "github.com/appscode/stash/apis/stash/v1beta1"
	"github.com/appscode/stash/pkg/docker"
	"github.com/appscode/stash/pkg/util"
	core "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/tools/reference"
	core_util "kmodules.xyz/client-go/core/v1"
	"kmodules.xyz/client-go/meta"
	wapi "kmodules.xyz/webhook-runtime/apis/workload/v1"
)

func (c *StashController) ensureRestoreInitContainer(w *wapi.Workload, rs *api_v1beta1.RestoreSession) error {
	// if RBAC is enabled then ensure ServiceAccount and respective ClusterRole and RoleBinding
	if c.EnableRBAC {
		sa := stringz.Val(w.Spec.Template.Spec.ServiceAccountName, "default")
		ref, err := reference.GetReference(scheme.Scheme, w)
		if err != nil {
			ref = &core.ObjectReference{
				Name:      w.Name,
				Namespace: w.Namespace,
			}
		}
		err = c.ensureRestoreInitContainerRBAC(ref, sa)
		if err != nil {
			return err
		}
	}

	repository, err := c.stashClient.StashV1alpha1().Repositories(rs.Namespace).Get(rs.Spec.Repository.Name, metav1.GetOptions{})
	if err != nil {
		log.Errorf("unable to get repository %s/%s: Reason: %v", rs.Namespace, rs.Spec.Repository.Name, err)
		return err
	}

	if repository.Spec.Backend.StorageSecretName == "" {
		err := fmt.Errorf("missing repository secret name  %s/%s", repository.Namespace, repository.Name)
		return err
	}

	// check if secret exist
	_, err = c.kubeClient.CoreV1().Secrets(w.Namespace).Get(repository.Spec.Backend.StorageSecretName, metav1.GetOptions{})
	if err != nil {
		return err
	}

	if w.Spec.Template.Annotations == nil {
		w.Spec.Template.Annotations = map[string]string{}
	}

	// mark pods with RestoreSession spec hash. used to force restart pods for rc/rs
	w.Spec.Template.Annotations[api_v1beta1.AppliedRestoreSessionSpecHash] = rs.GetSpecHash()

	image := docker.Docker{
		Registry: c.DockerRegistry,
		Image:    docker.ImageStash,
		Tag:      c.StashImageTag,
	}

	// insert restore init container
	w.Spec.Template.Spec.InitContainers = core_util.UpsertContainer(
		w.Spec.Template.Spec.InitContainers,
		util.NewRestoreInitContainer(rs, repository, image, c.EnableRBAC),
	)

	// keep existing image pull secrets and add new image pull secrets if specified in RestoreSession spec.
	if rs.Spec.RuntimeSettings.Pod != nil {
		w.Spec.Template.Spec.ImagePullSecrets = core_util.MergeLocalObjectReferences(
			w.Spec.Template.Spec.ImagePullSecrets,
			rs.Spec.RuntimeSettings.Pod.ImagePullSecrets,
		)
	}
	// add an emptyDir volume for holding temporary files
	w.Spec.Template.Spec.Volumes = util.UpsertTmpVolume(w.Spec.Template.Spec.Volumes, rs.Spec.TempDir)
	// add  downward volume to make some information of the workload accessible to the container
	w.Spec.Template.Spec.Volumes = util.UpsertDownwardVolume(w.Spec.Template.Spec.Volumes)
	// add storage secret as volume to the workload. this is mounted on the restore init container
	w.Spec.Template.Spec.Volumes = util.UpsertSecretVolume(w.Spec.Template.Spec.Volumes, repository.Spec.Backend.StorageSecretName)
	// if Repository uses local volume as backend, append this volume to workload
	w.Spec.Template.Spec.Volumes = util.MergeLocalVolume(w.Spec.Template.Spec.Volumes, &repository.Spec.Backend)

	// add RestoreSession definition as annotation of the workload
	if w.Annotations == nil {
		w.Annotations = make(map[string]string)
	}
	r := &api_v1beta1.RestoreSession{
		TypeMeta: metav1.TypeMeta{
			APIVersion: api_v1beta1.SchemeGroupVersion.String(),
			Kind:       api_v1beta1.ResourceKindRestoreSession,
		},
		ObjectMeta: rs.ObjectMeta,
		Spec:       rs.Spec,
	}
	data, _ := meta.MarshalToJson(r, api_v1beta1.SchemeGroupVersion)
	w.Annotations[api_v1beta1.KeyLastAppliedRestoreSession] = string(data)

	return nil
}

func (c *StashController) ensureRestoreInitContainerDeleted(w *wapi.Workload, rs *api_v1beta1.RestoreSession) error {
	// remove resource hash annotation
	if w.Spec.Template.Annotations != nil {
		delete(w.Spec.Template.Annotations, api_v1beta1.AppliedRestoreSessionSpecHash)
	}
	// remove init-container
	w.Spec.Template.Spec.InitContainers = core_util.EnsureContainerDeleted(w.Spec.Template.Spec.InitContainers, util.StashInitContainer)

	// restore init-container has been removed but workload still may have backup sidecar
	// so removed respective volumes that were added to the workload only if the workload does not have backup sidecar
	if !hasStashContainer(w) {
		// remove the helpers volumes added for init-container
		w.Spec.Template.Spec.Volumes = util.EnsureVolumeDeleted(w.Spec.Template.Spec.Volumes, util.ScratchDirVolumeName)
		w.Spec.Template.Spec.Volumes = util.EnsureVolumeDeleted(w.Spec.Template.Spec.Volumes, util.PodinfoVolumeName)
		w.Spec.Template.Spec.Volumes = util.EnsureVolumeDeleted(w.Spec.Template.Spec.Volumes, util.StashSecretVolume)

		// if stash-local volume was added for local backend, remove it
		w.Spec.Template.Spec.Volumes = util.EnsureVolumeDeleted(w.Spec.Template.Spec.Volumes, util.LocalVolumeName)
	}

	// remove respective annotations
	if w.Annotations != nil {
		delete(w.Annotations, api_v1beta1.KeyLastAppliedRestoreSession)
	}
	return nil
}

package controller

import (
	"fmt"
	"time"

	"github.com/appscode/go/log"
	stringz "github.com/appscode/go/strings"
	"github.com/appscode/stash/apis"
	api_v1alpha1 "github.com/appscode/stash/apis/stash/v1alpha1"
	api_v1beta1 "github.com/appscode/stash/apis/stash/v1beta1"
	"github.com/appscode/stash/pkg/docker"
	"github.com/appscode/stash/pkg/util"
	core "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/tools/clientcmd/api"
	"k8s.io/client-go/tools/reference"
	core_util "kmodules.xyz/client-go/core/v1"
	"kmodules.xyz/client-go/meta"
	wapi "kmodules.xyz/webhook-runtime/apis/workload/v1"
)

func (c *StashController) ensureWorkloadSidecar(w *wapi.Workload, restic *api_v1alpha1.Restic) error {
	if c.EnableRBAC {
		sa := stringz.Val(w.Spec.Template.Spec.ServiceAccountName, "default")
		ref, err := reference.GetReference(scheme.Scheme, w)
		if err != nil {
			ref = &core.ObjectReference{
				Name:      w.Name,
				Namespace: w.Namespace,
			}
		}
		err = c.ensureSidecarRoleBinding(ref, sa)
		if err != nil {
			return err
		}
	}

	if restic.Spec.Backend.StorageSecretName == "" {
		err := fmt.Errorf("missing repository secret name for Restic %s/%s", restic.Namespace, restic.Name)
		return err
	}

	_, err := c.kubeClient.CoreV1().Secrets(w.Namespace).Get(restic.Spec.Backend.StorageSecretName, metav1.GetOptions{})
	if err != nil {
		return err
	}

	if w.Spec.Template.Annotations == nil {
		w.Spec.Template.Annotations = map[string]string{}
	}
	// mark pods with Restic resource version, used to force restart pods for rc/rs
	w.Spec.Template.Annotations[api_v1alpha1.ResourceHash] = restic.GetSpecHash()

	image := docker.Docker{
		Registry: c.DockerRegistry,
		Image:    docker.ImageStash,
		Tag:      c.StashImageTag,
	}
	ref := api_v1alpha1.LocalTypedReference{
		Kind: w.Kind,
		Name: w.Name,
	}

	if restic.Spec.Type == api_v1alpha1.BackupOffline {
		w.Spec.Template.Spec.InitContainers = core_util.UpsertContainer(
			w.Spec.Template.Spec.InitContainers,
			util.NewInitContainer(restic, ref, image, c.EnableRBAC),
		)
	} else {
		w.Spec.Template.Spec.Containers = core_util.UpsertContainer(
			w.Spec.Template.Spec.Containers,
			util.NewSidecarContainer(restic, ref, image, c.EnableRBAC),
		)
	}

	// keep existing image pull secrets
	w.Spec.Template.Spec.ImagePullSecrets = core_util.MergeLocalObjectReferences(
		w.Spec.Template.Spec.ImagePullSecrets,
		restic.Spec.ImagePullSecrets,
	)

	w.Spec.Template.Spec.Volumes = util.UpsertScratchVolume(w.Spec.Template.Spec.Volumes)
	w.Spec.Template.Spec.Volumes = util.UpsertDownwardVolume(w.Spec.Template.Spec.Volumes)
	// if repository backend is local backend, mount this inside sidecar container
	w.Spec.Template.Spec.Volumes = util.MergeLocalVolume(w.Spec.Template.Spec.Volumes, &restic.Spec.Backend)

	if w.Annotations == nil {
		w.Annotations = make(map[string]string)
	}
	r := &api_v1alpha1.Restic{
		TypeMeta: metav1.TypeMeta{
			APIVersion: api.SchemeGroupVersion.String(),
			Kind:       api_v1alpha1.ResourceKindRestic,
		},
		ObjectMeta: restic.ObjectMeta,
		Spec:       restic.Spec,
	}
	data, _ := meta.MarshalToJson(r, api_v1alpha1.SchemeGroupVersion)
	w.Annotations[api_v1alpha1.LastAppliedConfiguration] = string(data)
	w.Annotations[apis.VersionTag] = c.StashImageTag

	return nil
}

func (c *StashController) ensureWorkloadSidecarDeleted(w *wapi.Workload, restic *api_v1alpha1.Restic) error {

	if w.Spec.Template.Annotations != nil {
		// mark pods with restic resource version, used to force restart pods for rc/rs
		delete(w.Spec.Template.Annotations, api_v1alpha1.ResourceHash)
	}

	if restic.Spec.Type == api_v1alpha1.BackupOffline {
		w.Spec.Template.Spec.InitContainers = core_util.EnsureContainerDeleted(w.Spec.Template.Spec.InitContainers, util.StashContainer)
	} else {
		w.Spec.Template.Spec.Containers = core_util.EnsureContainerDeleted(w.Spec.Template.Spec.Containers, util.StashContainer)
	}
	// backup sidecar/init-container has been removed but workload still may have restore init-container
	// so removed respective volumes that were added to the workload only if the workload does not have restore init-container
	if !hasStashContainer(w) {
		w.Spec.Template.Spec.Volumes = util.EnsureVolumeDeleted(w.Spec.Template.Spec.Volumes, util.ScratchDirVolumeName)
		w.Spec.Template.Spec.Volumes = util.EnsureVolumeDeleted(w.Spec.Template.Spec.Volumes, util.PodinfoVolumeName)

		if restic.Spec.Backend.Local != nil {
			w.Spec.Template.Spec.Volumes = util.EnsureVolumeDeleted(w.Spec.Template.Spec.Volumes, util.LocalVolumeName)
		}
	}
	if w.Annotations != nil {
		delete(w.Annotations, api_v1alpha1.LastAppliedConfiguration)
		delete(w.Annotations, apis.VersionTag)
	}
	return nil
}

func (c *StashController) ensureBackupSidecar(w *wapi.Workload, bc *api_v1beta1.BackupConfiguration) error {
	if c.EnableRBAC {
		sa := stringz.Val(w.Spec.Template.Spec.ServiceAccountName, "default")
		ref, err := reference.GetReference(scheme.Scheme, w)
		if err != nil {
			ref = &core.ObjectReference{
				Name:       w.Name,
				Namespace:  w.Namespace,
				APIVersion: w.APIVersion,
			}
		}
		err = c.ensureSidecarRoleBinding(ref, sa)
		if err != nil {
			return err
		}
	}

	repository, err := c.stashClient.StashV1alpha1().Repositories(bc.Namespace).Get(bc.Spec.Repository.Name, metav1.GetOptions{})
	if err != nil {
		log.Errorf("unable to get repository %s/%s: Reason: %v", bc.Namespace, bc.Spec.Repository.Name, err)
		return err
	}

	if repository.Spec.Backend.StorageSecretName == "" {
		return fmt.Errorf("missing repository secret name  %s/%s", repository.Namespace, repository.Name)
	}

	// check if secret exist
	_, err = c.kubeClient.CoreV1().Secrets(w.Namespace).Get(repository.Spec.Backend.StorageSecretName, metav1.GetOptions{})
	if err != nil {
		return err
	}

	if w.Spec.Template.Annotations == nil {
		w.Spec.Template.Annotations = map[string]string{}
	}
	// mark pods with BackupConfiguration spec hash. used to force restart pods for rc/rs
	w.Spec.Template.Annotations[api_v1beta1.AppliedBackupConfigurationSpecHash] = bc.GetSpecHash()

	image := docker.Docker{
		Registry: c.DockerRegistry,
		Image:    docker.ImageStash,
		Tag:      c.StashImageTag,
	}

	w.Spec.Template.Spec.Containers = core_util.UpsertContainer(
		w.Spec.Template.Spec.Containers,
		util.NewBackupSidecarContainer(bc, &repository.Spec.Backend, image, c.EnableRBAC),
	)

	// keep existing image pull secrets
	if bc.Spec.RuntimeSettings.Pod != nil {
		w.Spec.Template.Spec.ImagePullSecrets = core_util.MergeLocalObjectReferences(
			w.Spec.Template.Spec.ImagePullSecrets,
			bc.Spec.RuntimeSettings.Pod.ImagePullSecrets,
		)
	}

	w.Spec.Template.Spec.Volumes = util.UpsertTmpVolume(w.Spec.Template.Spec.Volumes, bc.Spec.TempDir)
	w.Spec.Template.Spec.Volumes = util.UpsertDownwardVolume(w.Spec.Template.Spec.Volumes)
	w.Spec.Template.Spec.Volumes = util.UpsertSecretVolume(w.Spec.Template.Spec.Volumes, repository.Spec.Backend.StorageSecretName)
	// if Repository uses local volume as backend, append this volume to workload.
	// otherwise, restic will not be able to access the backend
	w.Spec.Template.Spec.Volumes = util.MergeLocalVolume(w.Spec.Template.Spec.Volumes, &repository.Spec.Backend)

	if w.Annotations == nil {
		w.Annotations = make(map[string]string)
	}
	r := &api_v1beta1.BackupConfiguration{
		TypeMeta: metav1.TypeMeta{
			APIVersion: api.SchemeGroupVersion.String(),
			Kind:       api_v1beta1.ResourceKindBackupConfiguration,
		},
		ObjectMeta: bc.ObjectMeta,
		Spec:       bc.Spec,
	}
	data, _ := meta.MarshalToJson(r, api_v1beta1.SchemeGroupVersion)
	w.Annotations[api_v1beta1.KeyLastAppliedBackupConfiguration] = string(data)

	return nil
}

func (c *StashController) ensureBackupSidecarDeleted(w *wapi.Workload, bc *api_v1beta1.BackupConfiguration) error {
	// remove resource hash annotation
	if w.Spec.Template.Annotations != nil {
		delete(w.Spec.Template.Annotations, api_v1beta1.AppliedBackupConfigurationSpecHash)
	}
	// remove sidecar container
	w.Spec.Template.Spec.Containers = core_util.EnsureContainerDeleted(w.Spec.Template.Spec.Containers, util.StashContainer)

	// backup sidecar has been removed but workload still may have restore init-container
	// so removed respective volumes that were added to the workload only if the workload does not have restore init-container
	if !hasStashContainer(w) {
		// remove the helpers volumes that were added for sidecar
		w.Spec.Template.Spec.Volumes = util.EnsureVolumeDeleted(w.Spec.Template.Spec.Volumes, util.ScratchDirVolumeName)
		w.Spec.Template.Spec.Volumes = util.EnsureVolumeDeleted(w.Spec.Template.Spec.Volumes, util.PodinfoVolumeName)
		w.Spec.Template.Spec.Volumes = util.EnsureVolumeDeleted(w.Spec.Template.Spec.Volumes, util.StashSecretVolume)

		// if stash-local volume was added for local backend, remove it
		w.Spec.Template.Spec.Volumes = util.EnsureVolumeDeleted(w.Spec.Template.Spec.Volumes, util.LocalVolumeName)
	}

	// remove respective annotations
	if w.Annotations != nil {
		delete(w.Annotations, api_v1beta1.KeyLastAppliedBackupConfiguration)
	}

	return nil
}

// ensureWorkloadLatestState check if the workload's pod has latest update of workload specification
// if a pod does not have latest update, it deletes that pod so that new pod start with updated spec
func (c *StashController) ensureWorkloadLatestState(w *wapi.Workload) (bool, error) {
	stateChanged := false

	err := wait.PollImmediate(3*time.Second, 5*time.Minute, func() (done bool, err error) {
		r, err := metav1.LabelSelectorAsSelector(w.Spec.Selector)
		if err != nil {
			return false, err
		}
		// list all pods of this workload
		pods, err := c.kubeClient.CoreV1().Pods(w.Namespace).List(metav1.ListOptions{LabelSelector: r.String()})
		if err != nil {
			if errors.IsUnauthorized(err) || errors.IsForbidden(err) {
				return false, err
			}
			return false, nil // ignore temporary server errors
		}

		workloadSidecarState := hasStashSidecar(w.Spec.Template.Spec.Containers)
		workloadInitContainerState := hasStashInitContainer(w.Spec.Template.Spec.InitContainers)
		workloadBackupResourceHash := util.GetString(w.Spec.Template.Annotations, api_v1beta1.AppliedBackupConfigurationSpecHash)
		workloadResticResourceHash := util.GetString(w.Spec.Template.Annotations, api_v1alpha1.ResourceHash)
		workloadRestoreResourceHash := util.GetString(w.Spec.Template.Annotations, api_v1beta1.AppliedRestoreSessionSpecHash)

		// identify the pods that does not have latest update.
		// we have to restart these pods so that it starts with latest update
		var podsToRestart []core.Pod
		for _, pod := range pods.Items {
			if !isPodOwnedByWorkload(w, pod) {
				continue
			}
			podSidecarState := hasStashSidecar(pod.Spec.Containers)
			podInitContainerState := hasStashInitContainer(pod.Spec.InitContainers)
			podBackupResourceHash := util.GetString(pod.Annotations, api_v1beta1.AppliedBackupConfigurationSpecHash)
			podResticResourceHash := util.GetString(pod.Annotations, api_v1alpha1.ResourceHash)
			podRestoreResourceHash := util.GetString(pod.Annotations, api_v1beta1.AppliedRestoreSessionSpecHash)

			if workloadSidecarState != podSidecarState ||
				workloadInitContainerState != podInitContainerState ||
				workloadBackupResourceHash != podBackupResourceHash ||
				workloadResticResourceHash != podResticResourceHash ||
				workloadRestoreResourceHash != podRestoreResourceHash {

				podsToRestart = append(podsToRestart, pod)
			}
		}

		if len(podsToRestart) == 0 {
			return true, nil // done
		}
		stateChanged = true
		for _, pod := range podsToRestart {
			c.kubeClient.CoreV1().Pods(w.Namespace).Delete(pod.Name, &metav1.DeleteOptions{})
		}
		return false, nil // try again
	})
	if err != nil {
		return false, err
	}

	return stateChanged, nil
}

func isPodOwnedByWorkload(w *wapi.Workload, pod core.Pod) bool {
	for _, ref := range pod.OwnerReferences {
		if ref.Kind == w.Kind && ref.Name == w.Name {
			return true
		}
	}
	return false
}

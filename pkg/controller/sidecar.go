package controller

import (
	"fmt"
	"time"

	stringz "github.com/appscode/go/strings"
	wapi "github.com/appscode/kubernetes-webhook-util/apis/workload/v1"
	core_util "github.com/appscode/kutil/core/v1"
	"github.com/appscode/kutil/meta"
	api "github.com/appscode/stash/apis/stash/v1alpha1"
	"github.com/appscode/stash/pkg/docker"
	"github.com/appscode/stash/pkg/util"
	core "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/tools/reference"
)

func (c *StashController) ensureWorkloadSidecar(w *wapi.Workload, oldRestic, newRestic *api.Restic) error {
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

	if newRestic.Spec.Backend.StorageSecretName == "" {
		err := fmt.Errorf("missing repository secret name for Restic %s/%s", newRestic.Namespace, newRestic.Name)
		return err
	}

	_, err := c.kubeClient.CoreV1().Secrets(w.Namespace).Get(newRestic.Spec.Backend.StorageSecretName, metav1.GetOptions{})
	if err != nil {
		return err
	}

	if w.Spec.Template.Annotations == nil {
		w.Spec.Template.Annotations = map[string]string{}
	}
	// mark pods with restic resource version, used to force restart pods for rc/rs
	w.Spec.Template.Annotations[api.ResourceHash] = newRestic.GetSpecHash()

	image := docker.Docker{
		Registry: c.DockerRegistry,
		Image:    docker.ImageStash,
		Tag:      c.StashImageTag,
	}
	ref := api.LocalTypedReference{
		Kind: w.Kind,
		Name: w.Name,
	}

	if newRestic.Spec.Type == api.BackupOffline {
		w.Spec.Template.Spec.InitContainers = core_util.UpsertContainer(
			w.Spec.Template.Spec.InitContainers,
			util.NewInitContainer(newRestic, ref, image, c.EnableRBAC),
		)
	} else {
		w.Spec.Template.Spec.Containers = core_util.UpsertContainer(
			w.Spec.Template.Spec.Containers,
			util.NewSidecarContainer(newRestic, ref, image, c.EnableRBAC),
		)
	}

	// keep existing image pull secrets
	w.Spec.Template.Spec.ImagePullSecrets = core_util.MergeLocalObjectReferences(
		w.Spec.Template.Spec.ImagePullSecrets,
		newRestic.Spec.ImagePullSecrets,
	)

	w.Spec.Template.Spec.Volumes = util.UpsertScratchVolume(w.Spec.Template.Spec.Volumes)
	w.Spec.Template.Spec.Volumes = util.UpsertDownwardVolume(w.Spec.Template.Spec.Volumes)
	w.Spec.Template.Spec.Volumes = util.MergeLocalVolume(w.Spec.Template.Spec.Volumes, oldRestic, newRestic)

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

func (c *StashController) ensureWorkloadSidecarDeleted(w *wapi.Workload, restic *api.Restic) error {
	if c.EnableRBAC {
		err := c.ensureSidecarRoleBindingDeleted(w.ObjectMeta)
		if err != nil {
			return err
		}
	}

	if w.Spec.Template.Annotations != nil {
		// mark pods with restic resource version, used to force restart pods for rc/rs
		delete(w.Spec.Template.Annotations, api.ResourceHash)
	}

	if restic.Spec.Type == api.BackupOffline {
		w.Spec.Template.Spec.InitContainers = core_util.EnsureContainerDeleted(w.Spec.Template.Spec.InitContainers, util.StashContainer)
	} else {
		w.Spec.Template.Spec.Containers = core_util.EnsureContainerDeleted(w.Spec.Template.Spec.Containers, util.StashContainer)
	}

	w.Spec.Template.Spec.Volumes = util.EnsureVolumeDeleted(w.Spec.Template.Spec.Volumes, util.ScratchDirVolumeName)
	w.Spec.Template.Spec.Volumes = util.EnsureVolumeDeleted(w.Spec.Template.Spec.Volumes, util.PodinfoVolumeName)

	if restic.Spec.Backend.Local != nil {
		w.Spec.Template.Spec.Volumes = util.EnsureVolumeDeleted(w.Spec.Template.Spec.Volumes, util.LocalVolumeName)
	}
	if w.Annotations != nil {
		delete(w.Annotations, api.LastAppliedConfiguration)
		delete(w.Annotations, api.VersionTag)
	}
	return nil
}

func (c *StashController) forceRestartPods(w *wapi.Workload, restic *api.Restic) error {
	var sidecarAdded bool
	if w.Annotations != nil {
		_, sidecarAdded = w.Annotations[api.LastAppliedConfiguration]
	}

	return wait.PollImmediateInfinite(3*time.Second, func() (done bool, err error) {
		r, err := metav1.LabelSelectorAsSelector(w.Spec.Selector)
		if err != nil {
			return false, err
		}
		pods, err := c.kubeClient.CoreV1().Pods(w.Namespace).List(metav1.ListOptions{LabelSelector: r.String()})
		if err != nil {
			if errors.IsUnauthorized(err) || errors.IsForbidden(err) {
				return false, err
			}
			return false, nil // ignore temporary server errors
		}

		var podsToRestart []core.Pod
		for _, pod := range pods.Items {
			containers := pod.Spec.Containers
			if restic.Spec.Type == api.BackupOffline {
				containers = pod.Spec.InitContainers
			}

			if sidecarAdded {
				found := false
				for _, c := range containers {
					if c.Name == util.StashContainer && util.GetString(pod.Annotations, api.ResourceHash) == restic.GetSpecHash() {
						found = true
						break
					}
				}
				if !found {
					podsToRestart = append(podsToRestart, pod)
				}
			} else {
				found := false
				for _, c := range containers {
					if c.Name == util.StashContainer {
						found = true
						break
					}
				}
				if found {
					podsToRestart = append(podsToRestart, pod)
				}
			}
		}
		if len(podsToRestart) == 0 {
			return true, nil // done
		}
		for _, pod := range podsToRestart {
			c.kubeClient.CoreV1().Pods(w.Namespace).Delete(pod.Name, &metav1.DeleteOptions{})
		}
		return false, nil // try again
	})
}

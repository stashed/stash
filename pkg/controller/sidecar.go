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
	"fmt"
	"time"

	"stash.appscode.dev/apimachinery/apis"
	api_v1alpha1 "stash.appscode.dev/apimachinery/apis/stash/v1alpha1"
	api_v1beta1 "stash.appscode.dev/apimachinery/apis/stash/v1beta1"
	"stash.appscode.dev/apimachinery/pkg/docker"
	"stash.appscode.dev/stash/pkg/eventer"
	stash_rbac "stash.appscode.dev/stash/pkg/rbac"
	"stash.appscode.dev/stash/pkg/util"

	"github.com/appscode/go/log"
	stringz "github.com/appscode/go/strings"
	errors2 "github.com/appscode/go/util/errors"
	core "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/tools/clientcmd/api"
	core_util "kmodules.xyz/client-go/core/v1"
	"kmodules.xyz/client-go/meta"
	wapi "kmodules.xyz/webhook-runtime/apis/workload/v1"
)

func (c *StashController) ensureWorkloadSidecar(w *wapi.Workload, restic *api_v1alpha1.Restic, caller string) error {
	sa := stringz.Val(w.Spec.Template.Spec.ServiceAccountName, "default")
	owner, err := ownerWorkload(w)
	if err != nil {
		return err
	}
	//Don't create RBAC stuff when the caller is webhook to make the webhooks side effect free.
	if caller != apis.CallerWebhook {
		err = stash_rbac.EnsureSidecarRoleBinding(c.kubeClient, owner, w.Namespace, sa, nil)
		if err != nil {
			return err
		}
	}

	if restic.Spec.Backend.StorageSecretName == "" {
		err := fmt.Errorf("missing repository secret name for Restic %s/%s", restic.Namespace, restic.Name)
		return err
	}

	_, err = c.kubeClient.CoreV1().Secrets(w.Namespace).Get(restic.Spec.Backend.StorageSecretName, metav1.GetOptions{})
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
	localRef := api_v1alpha1.LocalTypedReference{
		Kind: w.Kind,
		Name: w.Name,
	}

	if restic.Spec.Type == api_v1alpha1.BackupOffline {
		w.Spec.Template.Spec.InitContainers = core_util.UpsertContainer(
			w.Spec.Template.Spec.InitContainers,
			util.NewInitContainer(restic, localRef, image),
		)
	} else {
		w.Spec.Template.Spec.Containers = core_util.UpsertContainer(
			w.Spec.Template.Spec.Containers,
			util.NewSidecarContainer(restic, localRef, image),
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

func (c *StashController) ensureWorkloadSidecarDeleted(w *wapi.Workload, restic *api_v1alpha1.Restic) {

	if w.Spec.Template.Annotations != nil {
		// mark pods with restic resource version, used to force restart pods for rc/rs
		delete(w.Spec.Template.Annotations, api_v1alpha1.ResourceHash)
	}

	if restic.Spec.Type == api_v1alpha1.BackupOffline {
		w.Spec.Template.Spec.InitContainers = core_util.EnsureContainerDeleted(w.Spec.Template.Spec.InitContainers, apis.StashContainer)
	} else {
		w.Spec.Template.Spec.Containers = core_util.EnsureContainerDeleted(w.Spec.Template.Spec.Containers, apis.StashContainer)
	}
	// backup sidecar/init-container has been removed but workload still may have restore init-container
	// so removed respective volumes that were added to the workload only if the workload does not have restore init-container
	if !util.HasStashContainer(w) {
		w.Spec.Template.Spec.Volumes = util.EnsureVolumeDeleted(w.Spec.Template.Spec.Volumes, apis.ScratchDirVolumeName)
		w.Spec.Template.Spec.Volumes = util.EnsureVolumeDeleted(w.Spec.Template.Spec.Volumes, apis.PodinfoVolumeName)

		if restic.Spec.Backend.Local != nil {
			w.Spec.Template.Spec.Volumes = util.EnsureVolumeDeleted(w.Spec.Template.Spec.Volumes, apis.LocalVolumeName)
		}
	}
	if w.Annotations != nil {
		delete(w.Annotations, api_v1alpha1.LastAppliedConfiguration)
		delete(w.Annotations, apis.VersionTag)
	}
}

func (c *StashController) ensureBackupSidecar(w *wapi.Workload, invoker apis.Invoker, targetInfo apis.TargetInfo, caller string) error {
	sa := stringz.Val(w.Spec.Template.Spec.ServiceAccountName, "default")
	owner, err := ownerWorkload(w)
	if err != nil {
		return err
	}

	//Don't create RBAC stuff when the caller is webhook to make the webhooks side effect free.
	if caller != apis.CallerWebhook {
		err = stash_rbac.EnsureSidecarRoleBinding(c.kubeClient, owner, invoker.ObjectMeta.Namespace, sa, invoker.Labels)
		if err != nil {
			return err
		}
	}

	repository, err := c.stashClient.StashV1alpha1().Repositories(invoker.ObjectMeta.Namespace).Get(invoker.Repository, metav1.GetOptions{})
	if err != nil {
		log.Errorf("unable to get repository %s/%s: Reason: %v", invoker.ObjectMeta.Namespace, invoker.Repository, err)
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
	w.Spec.Template.Annotations[api_v1beta1.AppliedBackupInvokerSpecHash] = invoker.Hash

	if targetInfo.Target == nil {
		return fmt.Errorf("target is nil")
	}

	image := docker.Docker{
		Registry: c.DockerRegistry,
		Image:    docker.ImageStash,
		Tag:      c.StashImageTag,
	}
	w.Spec.Template.Spec.Containers = core_util.UpsertContainer(
		w.Spec.Template.Spec.Containers,
		util.NewBackupSidecarContainer(invoker, targetInfo, &repository.Spec.Backend, image),
	)

	w.Spec.Template.Spec.Volumes = util.UpsertTmpVolume(w.Spec.Template.Spec.Volumes, targetInfo.TempDir)
	w.Spec.Template.Spec.Volumes = util.UpsertDownwardVolume(w.Spec.Template.Spec.Volumes)
	w.Spec.Template.Spec.Volumes = util.UpsertSecretVolume(w.Spec.Template.Spec.Volumes, repository.Spec.Backend.StorageSecretName)
	// if Repository uses local volume as backend, append this volume to workload.
	// otherwise, restic will not be able to access the backend
	w.Spec.Template.Spec.Volumes = util.MergeLocalVolume(w.Spec.Template.Spec.Volumes, &repository.Spec.Backend)

	if w.Annotations == nil {
		w.Annotations = make(map[string]string)
	}
	w.Annotations[api_v1beta1.KeyLastAppliedBackupInvoker] = string(invoker.ObjectJson)
	w.Annotations[api_v1beta1.KeyLastAppliedBackupInvokerKind] = invoker.ObjectRef.Kind

	return nil
}

func (c *StashController) ensureBackupSidecarDeleted(w *wapi.Workload) {
	// remove resource hash annotation
	if w.Spec.Template.Annotations != nil {
		delete(w.Spec.Template.Annotations, api_v1beta1.AppliedBackupInvokerSpecHash)
	}
	// remove sidecar container
	w.Spec.Template.Spec.Containers = core_util.EnsureContainerDeleted(w.Spec.Template.Spec.Containers, apis.StashContainer)

	// backup sidecar has been removed but workload still may have restore init-container
	// so removed respective volumes that were added to the workload only if the workload does not have restore init-container
	if !util.HasStashContainer(w) {
		// remove the helpers volumes that were added for sidecar
		w.Spec.Template.Spec.Volumes = util.EnsureVolumeDeleted(w.Spec.Template.Spec.Volumes, apis.ScratchDirVolumeName)
		w.Spec.Template.Spec.Volumes = util.EnsureVolumeDeleted(w.Spec.Template.Spec.Volumes, apis.PodinfoVolumeName)
		w.Spec.Template.Spec.Volumes = util.EnsureVolumeDeleted(w.Spec.Template.Spec.Volumes, apis.StashSecretVolume)

		// if stash-local volume was added for local backend, remove it
		w.Spec.Template.Spec.Volumes = util.EnsureVolumeDeleted(w.Spec.Template.Spec.Volumes, apis.LocalVolumeName)
	}

	// remove respective annotations
	if w.Annotations != nil {
		delete(w.Annotations, api_v1beta1.KeyLastAppliedBackupInvoker)
	}
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

		workloadSidecarState := util.HasStashSidecar(w.Spec.Template.Spec.Containers)
		workloadInitContainerState := util.HasStashInitContainer(w.Spec.Template.Spec.InitContainers)
		workloadBackupInvokerResourceHash := util.GetString(w.Spec.Template.Annotations, api_v1beta1.AppliedBackupInvokerSpecHash)
		workloadResticResourceHash := util.GetString(w.Spec.Template.Annotations, api_v1alpha1.ResourceHash)
		workloadRestoreResourceHash := util.GetString(w.Spec.Template.Annotations, api_v1beta1.AppliedRestoreSessionSpecHash)

		// identify the pods that does not have latest update.
		// we have to restart these pods so that it starts with latest update
		var podsToRestart []core.Pod
		for _, pod := range pods.Items {
			if !isPodOwnedByWorkload(w, pod) {
				continue
			}
			podSidecarState := util.HasStashSidecar(pod.Spec.Containers)
			podInitContainerState := util.HasStashInitContainer(pod.Spec.InitContainers)
			podBackupInvokerResourceHash := util.GetString(pod.Annotations, api_v1beta1.AppliedBackupInvokerSpecHash)
			podResticResourceHash := util.GetString(pod.Annotations, api_v1alpha1.ResourceHash)
			podRestoreResourceHash := util.GetString(pod.Annotations, api_v1beta1.AppliedRestoreSessionSpecHash)

			if workloadSidecarState != podSidecarState ||
				workloadInitContainerState != podInitContainerState ||
				workloadBackupInvokerResourceHash != podBackupInvokerResourceHash ||
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
			err := c.kubeClient.CoreV1().Pods(w.Namespace).Delete(pod.Name, &metav1.DeleteOptions{})
			if err != nil {
				log.Errorln(err)
			}
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

func (c *StashController) handleSidecarInjectionFailure(w *wapi.Workload, invoker apis.Invoker, tref api_v1beta1.TargetRef, err error) error {
	log.Warningf("Failed to inject stash sidecar into %s %s/%s. Reason: %v", w.Kind, w.Namespace, w.Name, err)

	// Failed to inject stash sidecar. So, set "StashSidecarInjected" condition to "False".
	cerr := c.setSidecarInjectedConditionToFalse(invoker, tref, err)

	// write event to respective resource
	_, err2 := eventer.CreateEvent(
		c.kubeClient,
		eventer.EventSourceWorkloadController,
		w.Object,
		core.EventTypeWarning,
		eventer.EventReasonSidecarInjectionFailed,
		fmt.Sprintf("Failed to inject stash sidecar into %s %s/%s. Reason: %v", w.Kind, w.Namespace, w.Name, err),
	)
	return errors2.NewAggregate([]error{err2, cerr})
}

func (c *StashController) handleSidecarInjectionSuccess(w *wapi.Workload, invoker apis.Invoker, tref api_v1beta1.TargetRef) error {
	log.Infof("Successfully injected stash sidecar into %s %s/%s.", w.Kind, w.Namespace, w.Name)

	// Set "StashSidecarInjected" condition to "True"
	cerr := c.setSidecarInjectedConditionToTrue(invoker, tref)

	// write event to respective resource
	_, err2 := eventer.CreateEvent(
		c.kubeClient,
		eventer.EventSourceWorkloadController,
		w.Object,
		core.EventTypeNormal,
		eventer.EventReasonSidecarInjectionSucceeded,
		fmt.Sprintf("Successfully injected stash sidecar into %s %s/%s.", w.Kind, w.Namespace, w.Name),
	)
	return errors2.NewAggregate([]error{err2, cerr})
}

func (c *StashController) handleSidecarDeletionSuccess(w *wapi.Workload) error {
	log.Infof("Successfully removed stash sidecar from %s %s/%s.", w.Kind, w.Namespace, w.Name)

	// write event to respective resource
	_, err2 := eventer.CreateEvent(
		c.kubeClient,
		eventer.EventSourceWorkloadController,
		w.Object,
		core.EventTypeNormal,
		eventer.EventReasonSidecarDeletionSucceeded,
		fmt.Sprintf("Successfully removed stash sidecar from %s %s/%s.", w.Kind, w.Namespace, w.Name),
	)
	return err2
}

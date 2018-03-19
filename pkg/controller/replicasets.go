package controller

import (
	"fmt"

	"github.com/appscode/go/log"
	stringz "github.com/appscode/go/strings"
	"github.com/appscode/kutil/admission"
	hooks "github.com/appscode/kutil/admission/v1beta1"
	core_util "github.com/appscode/kutil/core/v1"
	ext_util "github.com/appscode/kutil/extensions/v1beta1"
	"github.com/appscode/kutil/meta"
	"github.com/appscode/kutil/tools/queue"
	api "github.com/appscode/stash/apis/stash/v1alpha1"
	"github.com/appscode/stash/pkg/docker"
	"github.com/appscode/stash/pkg/util"
	"github.com/golang/glog"
	"k8s.io/api/core/v1"
	extensions "k8s.io/api/extensions/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/reference"
	"k8s.io/kubernetes/pkg/apis/apps"
)

func (c *StashController) NewReplicaSetWebhook() hooks.AdmissionHook {
	return hooks.NewGenericWebhook(
		schema.GroupVersionResource{
			Group:    "admission.stash.appscode.com",
			Version:  "v1alpha1",
			Resource: "replicasets",
		},
		"replicaset",
		[]string{apps.GroupName, extensions.GroupName},
		apps.SchemeGroupVersion.WithKind("ReplicaSet"),
		nil,
		&admission.ResourceHandlerFuncs{
			CreateFunc: func(obj runtime.Object) (runtime.Object, error) {
				modObj, _, err := c.mutateReplicaSet(obj.(*extensions.ReplicaSet))
				return modObj, err

			},
			UpdateFunc: func(oldObj, newObj runtime.Object) (runtime.Object, error) {
				modObj, _, err := c.mutateReplicaSet(newObj.(*extensions.ReplicaSet))
				return modObj, err
			},
		},
	)
}

func (c *StashController) initReplicaSetWatcher() {
	c.rsInformer = c.kubeInformerFactory.Extensions().V1beta1().ReplicaSets().Informer()
	c.rsQueue = queue.New("ReplicaSet", c.MaxNumRequeues, c.NumThreads, c.runReplicaSetInjector)
	c.rsInformer.AddEventHandler(queue.DefaultEventHandler(c.rsQueue.GetQueue()))
	c.rsLister = c.kubeInformerFactory.Extensions().V1beta1().ReplicaSets().Lister()
}

// syncToStdout is the business logic of the controller. In this controller it simply prints
// information about the deployment to stdout. In case an error happened, it has to simply return the error.
// The retry logic should not be part of the business logic.
func (c *StashController) runReplicaSetInjector(key string) error {
	obj, exists, err := c.rsInformer.GetIndexer().GetByKey(key)
	if err != nil {
		glog.Errorf("Fetching object with key %s from store failed with %v", key, err)
		return err
	}

	if !exists {
		// Below we will warm up our cache with a ReplicaSet, so that we will see a delete for one d
		glog.Warningf("ReplicaSet %s does not exist anymore\n", key)

		ns, name, err := cache.SplitMetaNamespaceKey(key)
		if err != nil {
			return err
		}
		util.DeleteConfigmapLock(c.KubeClient, ns, api.LocalTypedReference{Kind: api.KindReplicaSet, Name: name})
	} else {
		rs := obj.(*extensions.ReplicaSet)
		glog.Infof("Sync/Add/Update for ReplicaSet %s\n", rs.GetName())

		if !ext_util.IsOwnedByDeployment(rs) {
			modObj, modified, err := c.mutateReplicaSet(rs.DeepCopy())
			if err != nil {
				return err
			}

			patchedObj := &extensions.ReplicaSet{}
			if modified {
				patchedObj, _, err = ext_util.PatchReplicaSet(c.KubeClient, rs, func(obj *extensions.ReplicaSet) *extensions.ReplicaSet {
					return modObj
				})
				if err != nil {
					return err
				}
			}

			// ReplicaSet does not have RollingUpdate strategy. We must delete old pods manually to get patched state.
			if restartType := util.GetString(patchedObj.Annotations, util.ForceRestartType); restartType != "" {
				err := c.forceRestartRSPods(patchedObj, restartType, api.BackupType(util.GetString(patchedObj.Annotations, util.BackupType)))
				if err != nil {
					return err
				}
				return ext_util.WaitUntilReplicaSetReady(c.KubeClient, patchedObj.ObjectMeta)
			}
		}
	}
	return nil
}

func (c *StashController) mutateReplicaSet(rs *extensions.ReplicaSet) (*extensions.ReplicaSet, bool, error) {
	oldRestic, err := util.GetAppliedRestic(rs.Annotations)
	if err != nil {
		return nil, false, err
	}

	newRestic, err := util.FindRestic(c.RstLister, rs.ObjectMeta)
	if err != nil {
		log.Errorf("Error while searching Restic for ReplicaSet %s/%s.", rs.Name, rs.Namespace)
		return nil, false, err
	}

	if newRestic != nil && !util.ResticEqual(oldRestic, newRestic) {
		if !newRestic.Spec.Paused {
			modObj, err := c.ensureReplicaSetSidecar(rs, oldRestic, newRestic)
			if err != nil {
				return nil, false, err
			}
			return modObj, true, nil
		}
	} else if oldRestic != nil && newRestic == nil {
		modObj, err := c.ensureReplicaSetSidecarDeleted(rs, oldRestic)
		if err != nil {
			return nil, false, err
		}
		return modObj, true, nil
	}
	return rs, false, nil
}
func (c *StashController) ensureReplicaSetSidecar(obj *extensions.ReplicaSet, oldRestic, newRestic *api.Restic) (*extensions.ReplicaSet, error) {
	if c.EnableRBAC {
		sa := stringz.Val(obj.Spec.Template.Spec.ServiceAccountName, "default")
		ref, err := reference.GetReference(scheme.Scheme, obj)
		if err != nil {
			ref = &v1.ObjectReference{
				Name:      obj.Name,
				Namespace: obj.Namespace,
			}
		}
		err = c.ensureSidecarRoleBinding(ref, sa)
		if err != nil {
			return nil, err
		}
	}

	if newRestic.Spec.Backend.StorageSecretName == "" {
		return nil, fmt.Errorf("missing repository secret name for Restic %s/%s", newRestic.Namespace, newRestic.Name)
	}
	_, err := c.KubeClient.CoreV1().Secrets(obj.Namespace).Get(newRestic.Spec.Backend.StorageSecretName, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}

	image := docker.Docker{
		Registry: c.DockerRegistry,
		Image:    docker.ImageStash,
		Tag:      c.StashImageTag,
	}

	workload := api.LocalTypedReference{
		Kind: api.KindReplicaSet,
		Name: obj.Name,
	}

	if newRestic.Spec.Type == api.BackupOffline {
		obj.Spec.Template.Spec.InitContainers = core_util.UpsertContainer(
			obj.Spec.Template.Spec.InitContainers,
			util.NewInitContainer(newRestic, workload, image, c.EnableRBAC),
		)
	} else {
		obj.Spec.Template.Spec.Containers = core_util.UpsertContainer(
			obj.Spec.Template.Spec.Containers,
			util.NewSidecarContainer(newRestic, workload, image),
		)
	}

	// keep existing image pull secrets
	obj.Spec.Template.Spec.ImagePullSecrets = core_util.MergeLocalObjectReferences(
		obj.Spec.Template.Spec.ImagePullSecrets,
		newRestic.Spec.ImagePullSecrets,
	)

	obj.Spec.Template.Spec.Volumes = util.UpsertScratchVolume(obj.Spec.Template.Spec.Volumes)
	obj.Spec.Template.Spec.Volumes = util.UpsertDownwardVolume(obj.Spec.Template.Spec.Volumes)
	obj.Spec.Template.Spec.Volumes = util.MergeLocalVolume(obj.Spec.Template.Spec.Volumes, oldRestic, newRestic)

	if obj.Annotations == nil {
		obj.Annotations = make(map[string]string)
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
	obj.Annotations[api.LastAppliedConfiguration] = string(data)
	obj.Annotations[api.VersionTag] = c.StashImageTag

	obj.Annotations[util.ForceRestartType] = util.SideCarAdded
	obj.Annotations[util.BackupType] = string(newRestic.Spec.Type)

	return obj, nil
}

func (c *StashController) ensureReplicaSetSidecarDeleted(obj *extensions.ReplicaSet, restic *api.Restic) (*extensions.ReplicaSet, error) {
	if c.EnableRBAC {
		err := c.ensureSidecarRoleBindingDeleted(obj.ObjectMeta)
		if err != nil {
			return nil, err
		}
	}

	if restic.Spec.Type == api.BackupOffline {
		obj.Spec.Template.Spec.InitContainers = core_util.EnsureContainerDeleted(obj.Spec.Template.Spec.InitContainers, util.StashContainer)
	} else {
		obj.Spec.Template.Spec.Containers = core_util.EnsureContainerDeleted(obj.Spec.Template.Spec.Containers, util.StashContainer)
	}
	obj.Spec.Template.Spec.Volumes = util.EnsureVolumeDeleted(obj.Spec.Template.Spec.Volumes, util.ScratchDirVolumeName)
	obj.Spec.Template.Spec.Volumes = util.EnsureVolumeDeleted(obj.Spec.Template.Spec.Volumes, util.PodinfoVolumeName)
	if restic.Spec.Backend.Local != nil {
		obj.Spec.Template.Spec.Volumes = util.EnsureVolumeDeleted(obj.Spec.Template.Spec.Volumes, util.LocalVolumeName)
	}
	if obj.Annotations != nil {
		delete(obj.Annotations, api.LastAppliedConfiguration)
		delete(obj.Annotations, api.VersionTag)
	}

	if obj.Annotations == nil {
		obj.Annotations = make(map[string]string)
	}
	obj.Annotations[util.ForceRestartType] = util.SideCarRemoved
	obj.Annotations[util.BackupType] = string(restic.Spec.Type)

	err := util.DeleteConfigmapLock(c.KubeClient, obj.Namespace, api.LocalTypedReference{Kind: api.KindReplicaSet, Name: obj.Name})
	if err != nil {
		return nil, err
	}
	return obj, nil
}

func (c *StashController) forceRestartRSPods(rs *extensions.ReplicaSet, restartType string, backupType api.BackupType) error {
	rs, _, err := ext_util.PatchReplicaSet(c.KubeClient, rs, func(obj *extensions.ReplicaSet) *extensions.ReplicaSet {
		delete(obj.Annotations, util.ForceRestartType)
		delete(obj.Annotations, util.BackupType)
		return obj
	})
	if err != nil {
		return err
	}

	if restartType == util.SideCarAdded {
		err := util.WaitUntilSidecarAdded(c.KubeClient, rs.Namespace, rs.Spec.Selector, backupType)
		if err != nil {
			return err
		}
	} else if restartType == util.SideCarRemoved {
		err := util.WaitUntilSidecarRemoved(c.KubeClient, rs.Namespace, rs.Spec.Selector, backupType)
		if err != nil {
			return err
		}
	}
	return nil
}

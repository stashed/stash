package controller

import (
	"fmt"

	"github.com/appscode/go/log"
	stringz "github.com/appscode/go/strings"
	"github.com/appscode/kutil/admission"
	hooks "github.com/appscode/kutil/admission/v1beta1"
	core_util "github.com/appscode/kutil/core/v1"
	"github.com/appscode/kutil/meta"
	"github.com/appscode/kutil/tools/queue"
	api "github.com/appscode/stash/apis/stash/v1alpha1"
	"github.com/appscode/stash/pkg/docker"
	"github.com/appscode/stash/pkg/util"
	"github.com/golang/glog"
	core "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/reference"
)

func (c *StashController) NewReplicationControllerWebhook() hooks.AdmissionHook {
	return hooks.NewGenericWebhook(
		schema.GroupVersionResource{
			Group:    "admission.stash.appscode.com",
			Version:  "v1alpha1",
			Resource: "replicationcontrollers",
		},
		"replicationcontroller",
		[]string{core.GroupName},
		core.SchemeGroupVersion.WithKind("ReplicationController"),
		nil,
		&admission.ResourceHandlerFuncs{
			CreateFunc: func(obj runtime.Object) (runtime.Object, error) {
				modObj, _, err := c.mutateReplicationController(obj.(*core.ReplicationController))
				return modObj, err

			},
			UpdateFunc: func(oldObj, newObj runtime.Object) (runtime.Object, error) {
				modObj, _, err := c.mutateReplicationController(newObj.(*core.ReplicationController))
				return modObj, err
			},
		},
	)
}

func (c *StashController) initRCWatcher() {
	c.rcInformer = c.kubeInformerFactory.Core().V1().ReplicationControllers().Informer()
	c.rcQueue = queue.New("ReplicationController", c.MaxNumRequeues, c.NumThreads, c.runRCInjector)
	c.rcInformer.AddEventHandler(queue.DefaultEventHandler(c.rcQueue.GetQueue()))
	c.rcLister = c.kubeInformerFactory.Core().V1().ReplicationControllers().Lister()
}

// syncToStdout is the business logic of the controller. In this controller it simply prints
// information about the deployment to stdout. In case an error happened, it has to simply return the error.
// The retry logic should not be part of the business logic.
func (c *StashController) runRCInjector(key string) error {
	obj, exists, err := c.rcInformer.GetIndexer().GetByKey(key)
	if err != nil {
		glog.Errorf("Fetching object with key %s from store failed with %v", key, err)
		return err
	}

	if !exists {
		// Below we will warm up our cache with a ReplicationController, so that we will see a delete for one d
		glog.Warningf("ReplicationController %s does not exist anymore\n", key)

		ns, name, err := cache.SplitMetaNamespaceKey(key)
		if err != nil {
			return err
		}
		util.DeleteConfigmapLock(c.KubeClient, ns, api.LocalTypedReference{Kind: api.KindReplicationController, Name: name})
	} else {
		rc := obj.(*core.ReplicationController)
		glog.Infof("Sync/Add/Update for ReplicationController %s\n", rc.GetName())

		modObj, modified, err := c.mutateReplicationController(rc.DeepCopy())
		if err != nil {
			return err
		}

		patchedObj := &core.ReplicationController{}
		if modified {
			patchedObj, _, err = core_util.PatchRC(c.KubeClient, rc, func(obj *core.ReplicationController) *core.ReplicationController {
				return modObj
			})
			if err != nil {
				return err
			}
		}

		// ReplicationController does not have RollingUpdate strategy. We must delete old pods manually to get patched state.
		if restartType := util.GetString(patchedObj.Annotations, util.ForceRestartType); restartType != "" {
			err := c.forceRestartRCPods(patchedObj, restartType, api.BackupType(util.GetString(patchedObj.Annotations, util.BackupType)))
			if err != nil {
				return err
			}
			return core_util.WaitUntilRCReady(c.KubeClient, patchedObj.ObjectMeta)
		}
	}
	return nil
}

func (c *StashController) mutateReplicationController(rc *core.ReplicationController) (*core.ReplicationController, bool, error) {
	oldRestic, err := util.GetAppliedRestic(rc.Annotations)
	if err != nil {
		return nil, false, err
	}
	newRestic, err := util.FindRestic(c.RstLister, rc.ObjectMeta)
	if err != nil {
		log.Errorf("Error while searching Restic for ReplicationController %s/%s.", rc.Name, rc.Namespace)
		return nil, false, err
	}

	if newRestic != nil && !util.ResticEqual(oldRestic, newRestic) {
		if !newRestic.Spec.Paused {
			modObj, err := c.ensureReplicationControllerSidecar(rc, oldRestic, newRestic)
			if err != nil {
				return nil, false, err
			}
			return modObj, true, nil
		}
	} else if oldRestic != nil && newRestic == nil {
		modObj, err := c.ensureReplicationControllerSidecarDeleted(rc, oldRestic)
		if err != nil {
			return nil, false, err
		}
		return modObj, true, nil
	}
	return rc, false, nil
}

func (c *StashController) ensureReplicationControllerSidecar(obj *core.ReplicationController, oldRestic, newRestic *api.Restic) (*core.ReplicationController, error) {
	if c.EnableRBAC {
		sa := stringz.Val(obj.Spec.Template.Spec.ServiceAccountName, "default")
		ref, err := reference.GetReference(scheme.Scheme, obj)
		if err != nil {
			ref = &core.ObjectReference{
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
		Kind: api.KindReplicationController,
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

func (c *StashController) ensureReplicationControllerSidecarDeleted(obj *core.ReplicationController, restic *api.Restic) (*core.ReplicationController, error) {
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

	err := util.DeleteConfigmapLock(c.KubeClient, obj.Namespace, api.LocalTypedReference{Kind: api.KindReplicationController, Name: obj.Name})
	if err != nil {
		return nil, err
	}
	return obj, nil
}

func (c *StashController) forceRestartRCPods(rc *core.ReplicationController, restartType string, backupType api.BackupType) error {
	rc, _, err := core_util.PatchRC(c.KubeClient, rc, func(obj *core.ReplicationController) *core.ReplicationController {
		delete(obj.Annotations, util.ForceRestartType)
		delete(obj.Annotations, util.BackupType)
		return obj
	})
	if err != nil {
		return err
	}

	if restartType == util.SideCarAdded {
		err := util.WaitUntilSidecarAdded(c.KubeClient, rc.Namespace, &metav1.LabelSelector{MatchLabels: rc.Spec.Selector}, backupType)
		if err != nil {
			return err
		}
	} else if restartType == util.SideCarRemoved {
		err := util.WaitUntilSidecarRemoved(c.KubeClient, rc.Namespace, &metav1.LabelSelector{MatchLabels: rc.Spec.Selector}, backupType)
		if err != nil {
			return err
		}
	}
	return nil
}

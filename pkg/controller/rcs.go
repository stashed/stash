package controller

import (
	"fmt"

	"github.com/appscode/go/log"
	stringz "github.com/appscode/go/strings"
	core_util "github.com/appscode/kutil/core/v1"
	"github.com/appscode/kutil/meta"
	"github.com/appscode/kutil/tools/queue"
	api "github.com/appscode/stash/apis/stash/v1alpha1"
	"github.com/appscode/stash/pkg/docker"
	"github.com/appscode/stash/pkg/util"
	"github.com/golang/glog"
	core "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/reference"
)

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
		util.DeleteConfigmapLock(c.kubeClient, ns, api.LocalTypedReference{Kind: api.KindReplicationController, Name: name})
	} else {
		rc := obj.(*core.ReplicationController)
		glog.Infof("Sync/Add/Update for ReplicationController %s\n", rc.GetName())

		if util.ToBeInitializedByPeer(rc.Initializers) {
			glog.Warningf("Not stash's turn to initialize %s\n", rc.GetName())
			return nil
		}

		oldRestic, err := util.GetAppliedRestic(rc.Annotations)
		if err != nil {
			return err
		}
		newRestic, err := util.FindRestic(c.rstLister, rc.ObjectMeta)
		if err != nil {
			log.Errorf("Error while searching Restic for ReplicationController %s/%s.", rc.Name, rc.Namespace)
			return err
		}

		if newRestic != nil && !util.ResticEqual(oldRestic, newRestic) {
			if !newRestic.Spec.Paused {
				if newRestic.Spec.Type == api.BackupOffline && *rc.Spec.Replicas > 1 {
					return fmt.Errorf("cannot perform offline backup for rc with replicas > 1")
				}
				return c.EnsureReplicationControllerSidecar(rc, oldRestic, newRestic)
			}
		} else if oldRestic != nil && newRestic == nil {
			return c.EnsureReplicationControllerSidecarDeleted(rc, oldRestic)
		}

		// not restic workload, just remove the pending stash initializer
		if util.ToBeInitializedBySelf(rc.Initializers) {
			_, _, err = core_util.PatchRC(c.kubeClient, rc, func(obj *core.ReplicationController) *core.ReplicationController {
				fmt.Println("Removing pending stash initializer for", obj.Name)
				if len(obj.Initializers.Pending) == 1 {
					obj.Initializers = nil
				} else {
					obj.Initializers.Pending = obj.Initializers.Pending[1:]
				}
				return obj
			})
			if err != nil {
				log.Errorf("Error while removing pending stash initializer for %s/%s. Reason: %s", rc.Name, rc.Namespace, err)
				return err
			}
		}
	}
	return nil
}

func (c *StashController) EnsureReplicationControllerSidecar(resource *core.ReplicationController, old, new *api.Restic) (err error) {
	image := docker.Docker{
		Registry: c.DockerRegistry,
		Image:    docker.ImageStash,
		Tag:      c.StashImageTag,
	}

	if new.Spec.Backend.StorageSecretName == "" {
		err = fmt.Errorf("missing repository secret name for Restic %s/%s", new.Namespace, new.Name)
		return
	}
	_, err = c.kubeClient.CoreV1().Secrets(resource.Namespace).Get(new.Spec.Backend.StorageSecretName, metav1.GetOptions{})
	if err != nil {
		return err
	}

	if c.EnableRBAC {
		sa := stringz.Val(resource.Spec.Template.Spec.ServiceAccountName, "default")
		ref, err := reference.GetReference(scheme.Scheme, resource)
		if err != nil {
			return err
		}
		err = c.ensureSidecarRoleBinding(ref, sa)
		if err != nil {
			return err
		}
	}

	resource, _, err = core_util.PatchRC(c.kubeClient, resource, func(obj *core.ReplicationController) *core.ReplicationController {
		if util.ToBeInitializedBySelf(obj.Initializers) {
			fmt.Println("Removing pending stash initializer for", obj.Name)
			if len(obj.Initializers.Pending) == 1 {
				obj.Initializers = nil
			} else {
				obj.Initializers.Pending = obj.Initializers.Pending[1:]
			}
		}

		workload := api.LocalTypedReference{
			Kind: api.KindReplicationController,
			Name: obj.Name,
		}

		if new.Spec.Type == api.BackupOffline {
			obj.Spec.Template.Spec.InitContainers = core_util.UpsertContainer(
				obj.Spec.Template.Spec.InitContainers,
				util.NewInitContainer(new, workload, image, c.EnableRBAC),
			)
		} else {
			obj.Spec.Template.Spec.Containers = core_util.UpsertContainer(
				obj.Spec.Template.Spec.Containers,
				util.NewSidecarContainer(new, workload, image),
			)
		}

		// keep existing image pull secrets
		obj.Spec.Template.Spec.ImagePullSecrets = core_util.MergeLocalObjectReferences(
			obj.Spec.Template.Spec.ImagePullSecrets,
			new.Spec.ImagePullSecrets,
		)

		obj.Spec.Template.Spec.Volumes = util.UpsertScratchVolume(obj.Spec.Template.Spec.Volumes)
		obj.Spec.Template.Spec.Volumes = util.UpsertDownwardVolume(obj.Spec.Template.Spec.Volumes)
		obj.Spec.Template.Spec.Volumes = util.MergeLocalVolume(obj.Spec.Template.Spec.Volumes, old, new)

		if obj.Annotations == nil {
			obj.Annotations = make(map[string]string)
		}
		r := &api.Restic{
			TypeMeta: metav1.TypeMeta{
				APIVersion: api.SchemeGroupVersion.String(),
				Kind:       api.ResourceKindRestic,
			},
			ObjectMeta: new.ObjectMeta,
			Spec:       new.Spec,
		}
		data, _ := meta.MarshalToJson(r, api.SchemeGroupVersion)
		obj.Annotations[api.LastAppliedConfiguration] = string(data)
		obj.Annotations[api.VersionTag] = c.StashImageTag

		return obj
	})
	if err != nil {
		return
	}

	err = core_util.WaitUntilRCReady(c.kubeClient, resource.ObjectMeta)
	if err != nil {
		return
	}
	err = util.WaitUntilSidecarAdded(c.kubeClient, resource.Namespace, &metav1.LabelSelector{MatchLabels: resource.Spec.Selector}, new.Spec.Type)
	return err
}

func (c *StashController) EnsureReplicationControllerSidecarDeleted(resource *core.ReplicationController, restic *api.Restic) (err error) {
	if c.EnableRBAC {
		err := c.ensureSidecarRoleBindingDeleted(resource.ObjectMeta)
		if err != nil {
			return err
		}
	}

	resource, _, err = core_util.PatchRC(c.kubeClient, resource, func(obj *core.ReplicationController) *core.ReplicationController {
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
		return obj
	})
	if err != nil {
		return
	}

	err = core_util.WaitUntilRCReady(c.kubeClient, resource.ObjectMeta)
	if err != nil {
		return
	}
	err = util.WaitUntilSidecarRemoved(c.kubeClient, resource.Namespace, &metav1.LabelSelector{MatchLabels: resource.Spec.Selector}, restic.Spec.Type)
	if err != nil {
		return
	}
	util.DeleteConfigmapLock(c.kubeClient, resource.Namespace, api.LocalTypedReference{Kind: api.KindReplicationController, Name: resource.Name})
	return err
}

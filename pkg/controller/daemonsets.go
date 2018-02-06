package controller

import (
	"fmt"

	"github.com/appscode/go/log"
	stringz "github.com/appscode/go/strings"
	core_util "github.com/appscode/kutil/core/v1"
	ext_util "github.com/appscode/kutil/extensions/v1beta1"
	"github.com/appscode/kutil/meta"
	"github.com/appscode/kutil/tools/queue"
	api "github.com/appscode/stash/apis/stash/v1alpha1"
	"github.com/appscode/stash/pkg/docker"
	"github.com/appscode/stash/pkg/util"
	"github.com/golang/glog"
	extensions "k8s.io/api/extensions/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/tools/reference"
)

func (c *StashController) initDaemonSetWatcher() {
	c.dsInformer = c.kubeInformerFactory.Extensions().V1beta1().DaemonSets().Informer()
	c.dsQueue = queue.New("DaemonSet", c.options.MaxNumRequeues, c.options.NumThreads, c.runDaemonSetInjector)
	c.dsInformer.AddEventHandler(queue.DefaultEventHandler(c.dsQueue.GetQueue()))
	c.dsLister = c.kubeInformerFactory.Extensions().V1beta1().DaemonSets().Lister()
}

// syncToStdout is the business logic of the controller. In this controller it simply prints
// information about the deployment to stdout. In case an error happened, it has to simply return the error.
// The retry logic should not be part of the business logic.
func (c *StashController) runDaemonSetInjector(key string) error {
	obj, exists, err := c.dsInformer.GetIndexer().GetByKey(key)
	if err != nil {
		glog.Errorf("Fetching object with key %s from store failed with %v", key, err)
		return err
	}

	if !exists {
		// Below we will warm up our cache with a DaemonSet, so that we will see a delete for one d
		glog.Warningf("DaemonSet %s does not exist anymore\n", key)
	} else {
		ds := obj.(*extensions.DaemonSet)
		glog.Infof("Sync/Add/Update for DaemonSet %s\n", ds.GetName())

		if util.ToBeInitializedByPeer(ds.Initializers) {
			glog.Warningf("Not stash's turn to initialize %s\n", ds.GetName())
			return nil
		}

		oldRestic, err := util.GetAppliedRestic(ds.Annotations)
		if err != nil {
			return err
		}
		newRestic, err := util.FindRestic(c.rstLister, ds.ObjectMeta)
		if err != nil {
			log.Errorf("Error while searching Restic for DaemonSet %s/%s.", ds.Name, ds.Namespace)
			return err
		}

		if newRestic != nil && !util.ResticEqual(oldRestic, newRestic) {
			if newRestic.Spec.Paused == false {
				return c.EnsureDaemonSetSidecar(ds, oldRestic, newRestic)
			}
		} else if oldRestic != nil && newRestic == nil {
			return c.EnsureDaemonSetSidecarDeleted(ds, oldRestic)
		}

		// not restic workload, just remove the pending stash initializer
		if util.ToBeInitializedBySelf(ds.Initializers) {
			_, _, err = ext_util.PatchDaemonSet(c.k8sClient, ds, func(obj *extensions.DaemonSet) *extensions.DaemonSet {
				fmt.Println("Removing pending stash initializer for", obj.Name)
				if len(obj.Initializers.Pending) == 1 {
					obj.Initializers = nil
				} else {
					obj.Initializers.Pending = obj.Initializers.Pending[1:]
				}
				return obj
			})
			if err != nil {
				log.Errorf("Error while removing pending stash initializer for %s/%s. Reason: %s", ds.Name, ds.Namespace, err)
				return err
			}
		}
	}
	return nil
}

func (c *StashController) EnsureDaemonSetSidecar(resource *extensions.DaemonSet, old, new *api.Restic) (err error) {
	image := docker.Docker{
		Registry: c.options.DockerRegistry,
		Image:    docker.ImageStash,
		Tag:      c.options.StashImageTag,
	}

	if new.Spec.Backend.StorageSecretName == "" {
		err = fmt.Errorf("missing repository secret name for Restic %s/%s", new.Namespace, new.Name)
		return
	}
	_, err = c.k8sClient.CoreV1().Secrets(resource.Namespace).Get(new.Spec.Backend.StorageSecretName, metav1.GetOptions{})
	if err != nil {
		return err
	}

	if c.options.EnableRBAC {
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

	resource, _, err = ext_util.PatchDaemonSet(c.k8sClient, resource, func(obj *extensions.DaemonSet) *extensions.DaemonSet {
		if util.ToBeInitializedBySelf(obj.Initializers) {
			fmt.Println("Removing pending stash initializer for", obj.Name)
			if len(obj.Initializers.Pending) == 1 {
				obj.Initializers = nil
			} else {
				obj.Initializers.Pending = obj.Initializers.Pending[1:]
			}
		}

		workload := api.LocalTypedReference{
			Kind: api.KindDaemonSet,
			Name: obj.Name,
		}

		if new.Spec.Type == api.BackupOffline {
			obj.Spec.Template.Spec.InitContainers = core_util.UpsertContainer(
				obj.Spec.Template.Spec.InitContainers,
				util.NewInitContainer(new, workload, image, c.options.EnableRBAC),
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
		obj.Annotations[api.VersionTag] = c.options.StashImageTag
		return obj
	})
	if err != nil {
		return
	}

	err = ext_util.WaitUntilDaemonSetReady(c.k8sClient, resource.ObjectMeta)
	if err != nil {
		return
	}
	err = util.WaitUntilSidecarAdded(c.k8sClient, resource.Namespace, resource.Spec.Selector, new.Spec.Type)
	return
}

func (c *StashController) EnsureDaemonSetSidecarDeleted(resource *extensions.DaemonSet, restic *api.Restic) (err error) {
	if c.options.EnableRBAC {
		err := c.ensureSidecarRoleBindingDeleted(resource.ObjectMeta)
		if err != nil {
			return err
		}
	}

	resource, _, err = ext_util.PatchDaemonSet(c.k8sClient, resource, func(obj *extensions.DaemonSet) *extensions.DaemonSet {
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

	err = ext_util.WaitUntilDaemonSetReady(c.k8sClient, resource.ObjectMeta)
	if err != nil {
		return
	}
	err = util.WaitUntilSidecarRemoved(c.k8sClient, resource.Namespace, resource.Spec.Selector, restic.Spec.Type)
	return
}

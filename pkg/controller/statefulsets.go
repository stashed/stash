package controller

import (
	"fmt"

	"github.com/appscode/go/log"
	stringz "github.com/appscode/go/strings"
	apps_util "github.com/appscode/kutil/apps/v1beta1"
	core_util "github.com/appscode/kutil/core/v1"
	"github.com/appscode/kutil/meta"
	"github.com/appscode/kutil/tools/queue"
	api "github.com/appscode/stash/apis/stash/v1alpha1"
	"github.com/appscode/stash/pkg/docker"
	"github.com/appscode/stash/pkg/util"
	"github.com/golang/glog"
	apps "k8s.io/api/apps/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/tools/reference"
)

func (c *StashController) initStatefulSetWatcher() {
	c.ssInformer = c.kubeInformerFactory.Apps().V1beta1().StatefulSets().Informer()
	c.ssQueue = queue.New("StatefulSet", c.options.MaxNumRequeues, c.options.NumThreads, c.runStatefulSetInjector)
	c.ssInformer.AddEventHandler(queue.DefaultEventHandler(c.ssQueue.GetQueue()))
	c.ssLister = c.kubeInformerFactory.Apps().V1beta1().StatefulSets().Lister()
}

// syncToStdout is the business logic of the controller. In this controller it simply prints
// information about the deployment to stdout. In case an error happened, it has to simply return the error.
// The retry logic should not be part of the business logic.
func (c *StashController) runStatefulSetInjector(key string) error {
	obj, exists, err := c.ssInformer.GetIndexer().GetByKey(key)
	if err != nil {
		glog.Errorf("Fetching object with key %s from store failed with %v", key, err)
		return err
	}

	if !exists {
		// Below we will warm up our cache with a StatefulSet, so that we will see a delete for one d
		glog.Warningf("StatefulSet %s does not exist anymore\n", key)
	} else {
		ss := obj.(*apps.StatefulSet)
		glog.Infof("Sync/Add/Update for StatefulSet %s\n", ss.GetName())

		if util.ToBeInitializedByPeer(ss.Initializers) {
			glog.Warningf("Not stash's turn to initialize %s\n", ss.GetName())
			return nil
		}

		if util.ToBeInitializedBySelf(ss.Initializers) {
			// StatefulSets are supported during initializer phase
			oldBackup, err := util.GetAppliedBackup(ss.Annotations)
			if err != nil {
				return err
			}
			newBackup, err := util.FindBackup(c.rstLister, ss.ObjectMeta)
			if err != nil {
				log.Errorf("Error while searching Backup for StatefulSet %s/%s.", ss.Name, ss.Namespace)
				return err
			}

			if newBackup != nil && !util.BackupEqual(oldBackup, newBackup) {
				if !newBackup.Spec.Paused {
					return c.EnsureStatefulSetSidecar(ss, oldBackup, newBackup)
				}
			} else if oldBackup != nil && newBackup == nil {
				return c.EnsureStatefulSetSidecarDeleted(ss, oldBackup)
			}

			// not restic workload, just remove the pending stash initializer
			_, _, err = apps_util.PatchStatefulSet(c.k8sClient, ss, func(obj *apps.StatefulSet) *apps.StatefulSet {
				fmt.Println("Removing pending stash initializer for", obj.Name)
				if len(obj.Initializers.Pending) == 1 {
					obj.Initializers = nil
				} else {
					obj.Initializers.Pending = obj.Initializers.Pending[1:]
				}
				return obj
			})
			if err != nil {
				log.Errorf("Error while removing pending stash initializer for %s/%s. Reason: %s", ss.Name, ss.Namespace, err)
				return err
			}
		}
	}
	return nil
}

func (c *StashController) EnsureStatefulSetSidecar(resource *apps.StatefulSet, old, new *api.Backup) (err error) {
	image := docker.Docker{
		Registry: c.options.DockerRegistry,
		Image:    docker.ImageStash,
		Tag:      c.options.StashImageTag,
	}

	if new.Spec.Backend.StorageSecretName == "" {
		err = fmt.Errorf("missing repository secret name for Backup %s/%s", new.Namespace, new.Name)
		return
	}
	_, err = c.k8sClient.CoreV1().Secrets(resource.Namespace).Get(new.Spec.Backend.StorageSecretName, metav1.GetOptions{})
	if err != nil {
		return
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

	resource, _, err = apps_util.PatchStatefulSet(c.k8sClient, resource, func(obj *apps.StatefulSet) *apps.StatefulSet {
		if util.ToBeInitializedBySelf(obj.Initializers) {
			fmt.Println("Removing pending stash initializer for", obj.Name)
			if len(obj.Initializers.Pending) == 1 {
				obj.Initializers = nil
			} else {
				obj.Initializers.Pending = obj.Initializers.Pending[1:]
			}
		}

		workload := api.LocalTypedReference{
			Kind: api.KindStatefulSet,
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
		r := &api.Backup{
			TypeMeta: metav1.TypeMeta{
				APIVersion: api.SchemeGroupVersion.String(),
				Kind:       api.ResourceKindBackup,
			},
			ObjectMeta: new.ObjectMeta,
			Spec:       new.Spec,
		}
		data, _ := meta.MarshalToJson(r, api.SchemeGroupVersion)
		obj.Annotations[api.LastAppliedConfiguration] = string(data)
		obj.Annotations[api.VersionTag] = c.options.StashImageTag

		obj.Spec.UpdateStrategy.Type = apps.RollingUpdateStatefulSetStrategyType

		return obj
	})
	if err != nil {
		return
	}

	err = apps_util.WaitUntilStatefulSetReady(c.k8sClient, resource.ObjectMeta)
	return err
}

func (c *StashController) EnsureStatefulSetSidecarDeleted(resource *apps.StatefulSet, restic *api.Backup) (err error) {
	if c.options.EnableRBAC {
		err := c.ensureSidecarRoleBindingDeleted(resource.ObjectMeta)
		if err != nil {
			return err
		}
	}

	resource, _, err = apps_util.PatchStatefulSet(c.k8sClient, resource, func(obj *apps.StatefulSet) *apps.StatefulSet {
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
		obj.Spec.UpdateStrategy.Type = apps.RollingUpdateStatefulSetStrategyType
		return obj
	})
	if err != nil {
		return
	}

	err = apps_util.WaitUntilStatefulSetReady(c.k8sClient, resource.ObjectMeta)
	return err
}

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
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/reference"
)

func (c *StashController) initDeploymentWatcher() {
	c.dpInformer = c.kubeInformerFactory.Apps().V1beta1().Deployments().Informer()
	c.dpQueue = queue.New("Deployment", c.options.MaxNumRequeues, c.options.NumThreads, c.runDeploymentInjector)
	c.dpInformer.AddEventHandler(queue.DefaultEventHandler(c.dpQueue.GetQueue()))
	c.dpLister = c.kubeInformerFactory.Apps().V1beta1().Deployments().Lister()
}

// syncToStdout is the business logic of the controller. In this controller it simply prints
// information about the deployment to stdout. In case an error happened, it has to simply return the error.
// The retry logic should not be part of the business logic.
func (c *StashController) runDeploymentInjector(key string) error {
	obj, exists, err := c.dpInformer.GetIndexer().GetByKey(key)
	if err != nil {
		glog.Errorf("Fetching object with key %s from store failed with %v", key, err)
		return err
	}

	if !exists {
		// Below we will warm up our cache with a Deployment, so that we will see a delete for one d
		glog.Warningf("Deployment %s does not exist anymore\n", key)

		ns, name, err := cache.SplitMetaNamespaceKey(key)
		if err != nil {
			return err
		}
		util.DeleteConfigmapLock(c.k8sClient, ns, api.LocalTypedReference{Kind: api.KindDeployment, Name: name})
	} else {
		dp := obj.(*apps.Deployment)
		glog.Infof("Sync/Add/Update for Deployment %s\n", dp.GetName())

		if util.ToBeInitializedByPeer(dp.Initializers) {
			glog.Warningf("Not stash's turn to initialize %s\n", dp.GetName())
			return nil
		}

		oldBackup, err := util.GetAppliedBackup(dp.Annotations)
		if err != nil {
			return err
		}
		newBackup, err := util.FindBackup(c.rstLister, dp.ObjectMeta)
		if err != nil {
			log.Errorf("Error while searching Backup for Deployment %s/%s.", dp.Name, dp.Namespace)
			return err
		}
		if newBackup != nil && !util.BackupEqual(oldBackup, newBackup) {
			if !newBackup.Spec.Paused {
				if newBackup.Spec.Type == api.BackupOffline && *dp.Spec.Replicas > 1 {
					return fmt.Errorf("cannot perform offline backup for deployment with replicas > 1")
				}
				return c.EnsureDeploymentSidecar(dp, oldBackup, newBackup)
			}
		} else if oldBackup != nil && newBackup == nil {
			return c.EnsureDeploymentSidecarDeleted(dp, oldBackup)
		}

		// not restic workload, just remove the pending stash initializer
		if util.ToBeInitializedBySelf(dp.Initializers) {
			_, _, err = apps_util.PatchDeployment(c.k8sClient, dp, func(obj *apps.Deployment) *apps.Deployment {
				fmt.Println("Removing pending stash initializer for", obj.Name)
				if len(obj.Initializers.Pending) == 1 {
					obj.Initializers = nil
				} else {
					obj.Initializers.Pending = obj.Initializers.Pending[1:]
				}
				return obj
			})
			if err != nil {
				log.Errorf("Error while removing pending stash initializer for %s/%s. Reason: %s", dp.Name, dp.Namespace, err)
				return err
			}
		}
	}
	return nil
}

func (c *StashController) EnsureDeploymentSidecar(resource *apps.Deployment, old, new *api.Backup) (err error) {
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

	resource, _, err = apps_util.PatchDeployment(c.k8sClient, resource, func(obj *apps.Deployment) *apps.Deployment {
		if util.ToBeInitializedBySelf(obj.Initializers) {
			fmt.Println("Removing pending stash initializer for", obj.Name)
			if len(obj.Initializers.Pending) == 1 {
				obj.Initializers = nil
			} else {
				obj.Initializers.Pending = obj.Initializers.Pending[1:]
			}
		}

		workload := api.LocalTypedReference{
			Kind: api.KindDeployment,
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

		return obj
	})
	if err != nil {
		return
	}

	err = apps_util.WaitUntilDeploymentReady(c.k8sClient, resource.ObjectMeta)
	return err
}

func (c *StashController) EnsureDeploymentSidecarDeleted(resource *apps.Deployment, restic *api.Backup) (err error) {
	if c.options.EnableRBAC {
		err = c.ensureSidecarRoleBindingDeleted(resource.ObjectMeta)
		if err != nil {
			return
		}
	}

	resource, _, err = apps_util.PatchDeployment(c.k8sClient, resource, func(obj *apps.Deployment) *apps.Deployment {
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
	err = apps_util.WaitUntilDeploymentReady(c.k8sClient, resource.ObjectMeta)
	if err != nil {
		return
	}
	util.DeleteConfigmapLock(c.k8sClient, resource.Namespace, api.LocalTypedReference{Kind: api.KindDeployment, Name: resource.Name})
	return err
}

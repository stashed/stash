package controller

import (
	"fmt"

	"github.com/appscode/go/log"
	stringz "github.com/appscode/go/strings"
	"github.com/appscode/kutil/admission"
	hooks "github.com/appscode/kutil/admission/v1beta1"
	apps_util "github.com/appscode/kutil/apps/v1beta1"
	core_util "github.com/appscode/kutil/core/v1"
	"github.com/appscode/kutil/meta"
	"github.com/appscode/kutil/tools/queue"
	api "github.com/appscode/stash/apis/stash/v1alpha1"
	"github.com/appscode/stash/pkg/docker"
	"github.com/appscode/stash/pkg/util"
	"github.com/golang/glog"
	apps "k8s.io/api/apps/v1beta1"
	extensions "k8s.io/api/extensions/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/reference"
)

func (c *StashController) NewDeploymentWebhook() hooks.AdmissionHook {
	return hooks.NewGenericWebhook(
		schema.GroupVersionResource{
			Group:    "admission.stash.appscode.com",
			Version:  "v1alpha1",
			Resource: "deployments",
		},
		"deployment",
		[]string{apps.GroupName, extensions.GroupName},
		apps.SchemeGroupVersion.WithKind("Deployment"),
		nil,
		&admission.ResourceHandlerFuncs{
			CreateFunc: func(obj runtime.Object) (runtime.Object, error) {
				return nil, nil

			},
			UpdateFunc: func(oldObj, newObj runtime.Object) (runtime.Object, error) {
				return nil, nil
			},
		},
	)
}

func (c *StashController) initDeploymentWatcher() {
	c.dpInformer = c.kubeInformerFactory.Apps().V1beta1().Deployments().Informer()
	c.dpQueue = queue.New("Deployment", c.MaxNumRequeues, c.NumThreads, c.runDeploymentInjector)
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
		util.DeleteConfigmapLock(c.KubeClient, ns, api.LocalTypedReference{Kind: api.KindDeployment, Name: name})
	} else {
		dp := obj.(*apps.Deployment)
		glog.Infof("Sync/Add/Update for Deployment %s\n", dp.GetName())

		oldRestic, err := util.GetAppliedRestic(dp.Annotations)
		if err != nil {
			return err
		}
		newRestic, err := util.FindRestic(c.RstLister, dp.ObjectMeta)
		if err != nil {
			log.Errorf("Error while searching Restic for Deployment %s/%s.", dp.Name, dp.Namespace)
			return err
		}

		if newRestic != nil && c.EnableRBAC {
			sa := stringz.Val(dp.Spec.Template.Spec.ServiceAccountName, "default")
			ref, err := reference.GetReference(scheme.Scheme, dp)
			if err != nil {
				return err
			}
			err = c.EnsureSidecarRoleBinding(ref, sa)
			if err != nil {
				return err
			}
		}

		if newRestic != nil && !util.ResticEqual(oldRestic, newRestic) {
			if !newRestic.Spec.Paused {
				return c.EnsureDeploymentSidecar(dp, oldRestic, newRestic)
			}
		} else if oldRestic != nil && newRestic == nil {
			return c.EnsureDeploymentSidecarDeleted(dp, oldRestic)
		}
	}
	return nil
}

func (c *StashController) EnsureDeploymentSidecar(resource *apps.Deployment, old, new *api.Restic) (err error) {

	if new.Spec.Backend.StorageSecretName == "" {
		err = fmt.Errorf("missing repository secret name for Restic %s/%s", new.Namespace, new.Name)
		return
	}
	_, err = c.KubeClient.CoreV1().Secrets(resource.Namespace).Get(new.Spec.Backend.StorageSecretName, metav1.GetOptions{})
	if err != nil {
		return err
	}

	resource, _, err = apps_util.PatchDeployment(c.KubeClient, resource, func(obj *apps.Deployment) *apps.Deployment {
		return c.DeploymentSidecarInjectionTransformerFunc(obj, old, new)
	})
	if err != nil {
		return
	}

	err = apps_util.WaitUntilDeploymentReady(c.KubeClient, resource.ObjectMeta)
	return err
}

func (c *StashController) EnsureDeploymentSidecarDeleted(resource *apps.Deployment, restic *api.Restic) (err error) {
	if c.EnableRBAC {
		err = c.ensureSidecarRoleBindingDeleted(resource.ObjectMeta)
		if err != nil {
			return
		}
	}

	resource, _, err = apps_util.PatchDeployment(c.KubeClient, resource, func(obj *apps.Deployment) *apps.Deployment {
		objMod := c.DeploymentSidecarDeletionTransformerFunc(obj, restic)
		return objMod
	})
	if err != nil {
		return
	}

	err = apps_util.WaitUntilDeploymentReady(c.KubeClient, resource.ObjectMeta)
	if err != nil {
		return
	}
	util.DeleteConfigmapLock(c.KubeClient, resource.Namespace, api.LocalTypedReference{Kind: api.KindDeployment, Name: resource.Name})
	return err
}

func (c *StashController) DeploymentSidecarInjectionTransformerFunc(obj *apps.Deployment, oldRestic, newRestic *api.Restic) *apps.Deployment {
	image := docker.Docker{
		Registry: c.DockerRegistry,
		Image:    docker.ImageStash,
		Tag:      c.StashImageTag,
	}

	workload := api.LocalTypedReference{
		Kind: api.KindDeployment,
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

	return obj
}

func (c *StashController) DeploymentSidecarDeletionTransformerFunc(obj *apps.Deployment, restic *api.Restic) *apps.Deployment {
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
}

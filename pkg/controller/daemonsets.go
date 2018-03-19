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
	apps "k8s.io/api/apps/v1beta1"
	"k8s.io/api/core/v1"
	extensions "k8s.io/api/extensions/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/tools/reference"
)

func (c *StashController) NewDaemonSetWebhook() hooks.AdmissionHook {
	return hooks.NewGenericWebhook(
		schema.GroupVersionResource{
			Group:    "daemonset.admission.stash.appscode.com",
			Version:  "v1alpha1",
			Resource: "daemonsets",
		},
		"daemonset",
		[]string{apps.GroupName, extensions.GroupName},
		extensions.SchemeGroupVersion.WithKind("DaemonSet"),
		nil,
		&admission.ResourceHandlerFuncs{
			CreateFunc: func(obj runtime.Object) (runtime.Object, error) {
				modObj, _, err := c.mutateDaemonSet(obj.(*extensions.DaemonSet))
				return modObj, err

			},
			UpdateFunc: func(oldObj, newObj runtime.Object) (runtime.Object, error) {
				modObj, _, err := c.mutateDaemonSet(newObj.(*extensions.DaemonSet))
				return modObj, err
			},
		},
	)
}

func (c *StashController) initDaemonSetWatcher() {
	c.dsInformer = c.kubeInformerFactory.Extensions().V1beta1().DaemonSets().Informer()
	c.dsQueue = queue.New("DaemonSet", c.MaxNumRequeues, c.NumThreads, c.runDaemonSetInjector)
	c.dsInformer.AddEventHandler(queue.DefaultEventHandler(c.dsQueue.GetQueue()))
	c.dsLister = c.kubeInformerFactory.Extensions().V1beta1().DaemonSets().Lister()
}

// syncToStdout is the business logic of the controller. In this controller it simply prints
// information about the daemonset to stdout. In case an error happened, it has to simply return the error.
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
		modObj, modified, err := c.mutateDaemonSet(ds.DeepCopy())
		if err != nil {
			return err
		}

		if modified {
			patchedObj, _, err := ext_util.PatchDaemonSet(c.KubeClient, ds, func(obj *extensions.DaemonSet) *extensions.DaemonSet {
				return modObj
			})
			if err != nil {
				return err
			}
			return ext_util.WaitUntilDaemonSetReady(c.KubeClient, patchedObj.ObjectMeta)
		}

	}
	return nil
}

func (c *StashController) mutateDaemonSet(ds *extensions.DaemonSet) (*extensions.DaemonSet, bool, error) {
	oldRestic, err := util.GetAppliedRestic(ds.Annotations)
	if err != nil {
		return nil, false, err
	}

	newRestic, err := util.FindRestic(c.RstLister, ds.ObjectMeta)
	if err != nil {
		log.Errorf("Error while searching Restic for DaemonSet %s/%s.", ds.Name, ds.Namespace)
		return nil, false, err
	}

	if newRestic != nil && !util.ResticEqual(oldRestic, newRestic) {
		if !newRestic.Spec.Paused {
			modObj, err := c.ensureDaemonSetSidecar(ds, oldRestic, newRestic)
			if err != nil {
				return nil, false, err
			}
			return modObj, true, nil
		}
	} else if oldRestic != nil && newRestic == nil {
		modObj, err := c.ensureDaemonSetSidecarDeleted(ds, oldRestic)
		if err != nil {
			return nil, false, err
		}
		return modObj, true, nil
	}
	return ds, false, nil
}
func (c *StashController) ensureDaemonSetSidecar(obj *extensions.DaemonSet, oldRestic, newRestic *api.Restic) (*extensions.DaemonSet, error) {
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
		err := fmt.Errorf("missing repository secret name for Restic %s/%s", newRestic.Namespace, newRestic.Name)
		return nil, err
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
		Kind: api.KindDaemonSet,
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

	obj.Spec.UpdateStrategy.Type = extensions.RollingUpdateDaemonSetStrategyType
	if obj.Spec.UpdateStrategy.RollingUpdate == nil ||
		obj.Spec.UpdateStrategy.RollingUpdate.MaxUnavailable == nil ||
		obj.Spec.UpdateStrategy.RollingUpdate.MaxUnavailable.IntValue() == 0 {
		count := intstr.FromInt(1)
		obj.Spec.UpdateStrategy.RollingUpdate = &extensions.RollingUpdateDaemonSet{
			MaxUnavailable: &count,
		}
	}

	return obj, nil
}

func (c *StashController) ensureDaemonSetSidecarDeleted(obj *extensions.DaemonSet, restic *api.Restic) (*extensions.DaemonSet, error) {
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
	return obj, nil
}

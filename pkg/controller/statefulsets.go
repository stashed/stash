package controller

import (
	"fmt"

	"github.com/appscode/go/log"
	stringz "github.com/appscode/go/strings"
	hooks "github.com/appscode/kutil/admission/v1beta1"
	apps_util "github.com/appscode/kutil/apps/v1beta1"
	core_util "github.com/appscode/kutil/core/v1"
	"github.com/appscode/kutil/meta"
	"github.com/appscode/kutil/tools/queue"
	api "github.com/appscode/stash/apis/stash/v1alpha1"
	"github.com/appscode/stash/pkg/docker"
	"github.com/appscode/stash/pkg/util"
	"github.com/golang/glog"
	"k8s.io/apimachinery/pkg/runtime/schema"
	//oneliner "github.com/the-redback/go-oneliners"
	"github.com/appscode/kutil/admission"
	apps "k8s.io/api/apps/v1beta1"
	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/tools/reference"
)

func (c *StashController) NewStatefulSetWebhook() hooks.AdmissionHook {
	return hooks.NewGenericWebhook(
		schema.GroupVersionResource{
			Group:    "admission.stash.appscode.com",
			Version:  "v1alpha1",
			Resource: "statefulsets",
		},
		"statefulset",
		[]string{apps.GroupName},
		apps.SchemeGroupVersion.WithKind("StatefulSet"),
		nil,
		&admission.ResourceHandlerFuncs{
			CreateFunc: func(obj runtime.Object) (runtime.Object, error) {
				modObj, _, err := c.mutateStatefulSet(obj.(*apps.StatefulSet))
				return modObj, err

			},
			UpdateFunc: func(oldObj, newObj runtime.Object) (runtime.Object, error) {
				modObj, _, err := c.mutateStatefulSet(newObj.(*apps.StatefulSet))
				return modObj, err
			},
		},
	)
}

func (c *StashController) initStatefulSetWatcher() {
	c.ssInformer = c.kubeInformerFactory.Apps().V1beta1().StatefulSets().Informer()
	c.ssQueue = queue.New("StatefulSet", c.MaxNumRequeues, c.NumThreads, c.runStatefulSetInjector)
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

		modObj, modified, err := c.mutateStatefulSet(ss.DeepCopy())
		if err != nil {
			return nil
		}

		if modified {
			patchedObj, _, err := apps_util.PatchStatefulSet(c.KubeClient, ss, func(obj *apps.StatefulSet) *apps.StatefulSet {
				return modObj
			})
			if err != nil {
				return err
			}

			return apps_util.WaitUntilStatefulSetReady(c.KubeClient, patchedObj.ObjectMeta)
		}
	}
	return nil
}

func (c *StashController) mutateStatefulSet(ss *apps.StatefulSet) (*apps.StatefulSet, bool, error) {
	oldRestic, err := util.GetAppliedRestic(ss.Annotations)
	if err != nil {
		return nil, false, err
	}

	newRestic, err := util.FindRestic(c.RstLister, ss.ObjectMeta)
	if err != nil {
		log.Errorf("Error while searching Restic for StatefulSet %s/%s.", ss.Name, ss.Namespace)
		return nil, false, err
	}

	if newRestic != nil && !util.ResticEqual(oldRestic, newRestic) {
		if !newRestic.Spec.Paused {
			modObj, err := c.ensureStatefulSetSidecar(ss, oldRestic, newRestic)
			if err != nil {
				return nil, false, err
			}
			return modObj, true, nil
		}
	} else if oldRestic != nil && newRestic == nil {
		modObj, err := c.ensureStatefulSetSidecarDeleted(ss, oldRestic)
		if err != nil {
			return nil, false, err
		}
		return modObj, true, nil
	}
	return ss, false, nil
}
func (c *StashController) ensureStatefulSetSidecar(obj *apps.StatefulSet, oldRestic, newRestic *api.Restic) (*apps.StatefulSet, error) {
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
		Kind: api.KindStatefulSet,
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

	obj.Spec.UpdateStrategy.Type = apps.RollingUpdateStatefulSetStrategyType

	return obj, nil
}

func (c *StashController) ensureStatefulSetSidecarDeleted(obj *apps.StatefulSet, restic *api.Restic) (*apps.StatefulSet, error) {
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

	obj.Spec.UpdateStrategy.Type = apps.RollingUpdateStatefulSetStrategyType

	return obj, nil
}

package controller

import (
	"errors"
	"fmt"
	"reflect"

	"github.com/appscode/go-version"
	acrt "github.com/appscode/go/runtime"
	"github.com/appscode/kutil"
	"github.com/appscode/log"
	sapi "github.com/appscode/stash/api"
	"github.com/appscode/stash/pkg/util"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/apimachinery/pkg/watch"
	apiv1 "k8s.io/client-go/pkg/api/v1"
	apps "k8s.io/client-go/pkg/apis/apps/v1beta1"
	"k8s.io/client-go/tools/cache"
)

// Blocks caller. Intended to be called as a Go routine.
func (c *Controller) WatchStatefulSets() {
	// Skip v
	if info, err := c.kubeClient.Discovery().ServerVersion(); err == nil {
		v1_6, err := version.NewConstraint("<= 1.6")
		if err != nil {
			log.Warningf("Skipping watching StatefulSet for, Reason: %v", err)
			return
		}
		v, err := version.NewVersion(info.Major + "." + info.Minor)
		if err != nil {
			log.Warningf("Skipping watching StatefulSet for, Reason: %v", err)
			return
		}
		if v1_6.Check(v) {
			log.Warningf("Skipping watching StatefulSet for Kubernetes version: %s.%s.%s[%s]", info.Major, info.Minor, info.String())
			return
		}
	}
	if !util.IsPreferredAPIResource(c.kubeClient, apps.SchemeGroupVersion.String(), "StatefulSet") {
		log.Warningf("Skipping watching non-preferred GroupVersion:%s Kind:%s", apps.SchemeGroupVersion.String(), "StatefulSet")
		return
	}

	defer acrt.HandleCrash()

	lw := &cache.ListWatch{
		ListFunc: func(opts metav1.ListOptions) (runtime.Object, error) {
			return c.kubeClient.AppsV1beta1().StatefulSets(apiv1.NamespaceAll).List(metav1.ListOptions{})
		},
		WatchFunc: func(options metav1.ListOptions) (watch.Interface, error) {
			return c.kubeClient.AppsV1beta1().StatefulSets(apiv1.NamespaceAll).Watch(metav1.ListOptions{})
		},
	}
	_, ctrl := cache.NewInformer(lw,
		&apps.StatefulSet{},
		c.syncPeriod,
		cache.ResourceEventHandlerFuncs{
			AddFunc: func(obj interface{}) {
				if resource, ok := obj.(*apps.StatefulSet); ok {
					log.Infof("StatefulSet %s@%s added", resource.Name, resource.Namespace)

					restic, err := util.FindRestic(c.stashClient, resource.ObjectMeta)
					if err != nil {
						log.Errorf("Error while searching Restic for StatefulSet %s@%s.", resource.Name, resource.Namespace)
						return
					}
					if restic == nil {
						log.Errorf("No Restic found for StatefulSet %s@%s.", resource.Name, resource.Namespace)
						return
					}
					c.EnsureStatefulSetSidecar(resource, nil, restic)
				}
			},
			UpdateFunc: func(old, new interface{}) {
				oldObj, ok := old.(*apps.StatefulSet)
				if !ok {
					log.Errorln(errors.New("Invalid StatefulSet object"))
					return
				}
				newObj, ok := new.(*apps.StatefulSet)
				if !ok {
					log.Errorln(errors.New("Invalid StatefulSet object"))
					return
				}
				if !reflect.DeepEqual(oldObj.Labels, newObj.Labels) {
					oldRestic, err := util.FindRestic(c.stashClient, oldObj.ObjectMeta)
					if err != nil {
						log.Errorf("Error while searching Restic for StatefulSet %s@%s.", oldObj.Name, oldObj.Namespace)
						return
					}
					newRestic, err := util.FindRestic(c.stashClient, newObj.ObjectMeta)
					if err != nil {
						log.Errorf("Error while searching Restic for StatefulSet %s@%s.", newObj.Name, newObj.Namespace)
						return
					}
					if util.ResticEqual(oldRestic, newRestic) {
						return
					}
					if newRestic != nil {
						c.EnsureStatefulSetSidecar(newObj, oldRestic, newRestic)
					} else if oldRestic != nil {
						c.EnsureStatefulSetSidecarDeleted(newObj, oldRestic)
					}
				}
			},
		},
	)
	ctrl.Run(wait.NeverStop)
}

func (c *Controller) EnsureStatefulSetSidecar(resource *apps.StatefulSet, old, new *sapi.Restic) (err error) {
	defer func() {
		if err != nil {
			sidecarFailedToDelete()
			return
		}
		sidecarSuccessfullyAdd()
	}()

	if new.Spec.Backend.StorageSecretName == "" {
		err = fmt.Errorf("Missing repository secret name for Restic %s@%s.", new.Name, new.Namespace)
		return
	}
	_, err = c.kubeClient.CoreV1().Secrets(resource.Namespace).Get(new.Spec.Backend.StorageSecretName, metav1.GetOptions{})
	if err != nil {
		return err
	}

	if name := util.GetString(resource.Annotations, sapi.ConfigName); name != "" && name != new.Name {
		log.Infof("Restic %s sidecar already added for StatefulSet %s@%s.", name, resource.Name, resource.Namespace)
		return nil
	}

	_, err = kutil.PatchStatefulSet(c.kubeClient, resource, func(resource *apps.StatefulSet) {
		resource.Spec.Template.Spec.Containers = kutil.UpsertContainer(resource.Spec.Template.Spec.Containers, util.CreateSidecarContainer(new, c.SidecarImageTag, "StatefulSet/"+resource.Name))
		resource.Spec.Template.Spec.Volumes = util.UpsertScratchVolume(resource.Spec.Template.Spec.Volumes)
		resource.Spec.Template.Spec.Volumes = util.UpsertDownwardVolume(resource.Spec.Template.Spec.Volumes)
		resource.Spec.Template.Spec.Volumes = util.MergeLocalVolume(resource.Spec.Template.Spec.Volumes, old, new)

		if resource.Annotations == nil {
			resource.Annotations = make(map[string]string)
		}
		resource.Annotations[sapi.ConfigName] = new.Name
		resource.Annotations[sapi.VersionTag] = c.SidecarImageTag
	})
	if err != nil {
		return
	}

	err = kutil.WaitUntilStatefulSetReady(c.kubeClient, resource.ObjectMeta)
	if err != nil {
		return
	}
	err = util.WaitUntilSidecarAdded(c.kubeClient, resource.Namespace, resource.Spec.Selector)
	return err
}

func (c *Controller) EnsureStatefulSetSidecarDeleted(resource *apps.StatefulSet, restic *sapi.Restic) (err error) {
	defer func() {
		if err != nil {
			sidecarFailedToDelete()
			return
		}
		sidecarSuccessfullyDeleted()
	}()

	if name := util.GetString(resource.Annotations, sapi.ConfigName); name == "" {
		log.Infof("Restic sidecar already removed for StatefulSet %s@%s.", resource.Name, resource.Namespace)
		return nil
	}

	_, err = kutil.PatchStatefulSet(c.kubeClient, resource, func(resource *apps.StatefulSet) {
		resource.Spec.Template.Spec.Containers = kutil.EnsureContainerDeleted(resource.Spec.Template.Spec.Containers, util.StashContainer)
		resource.Spec.Template.Spec.Volumes = util.EnsureVolumeDeleted(resource.Spec.Template.Spec.Volumes, util.ScratchDirVolumeName)
		resource.Spec.Template.Spec.Volumes = util.EnsureVolumeDeleted(resource.Spec.Template.Spec.Volumes, util.PodinfoVolumeName)
		if restic.Spec.Backend.Local != nil {
			resource.Spec.Template.Spec.Volumes = util.EnsureVolumeDeleted(resource.Spec.Template.Spec.Volumes, util.LocalVolumeName)
		}
		if resource.Annotations != nil {
			delete(resource.Annotations, sapi.ConfigName)
			delete(resource.Annotations, sapi.VersionTag)
		}
	})
	if err != nil {
		return
	}

	err = kutil.WaitUntilStatefulSetReady(c.kubeClient, resource.ObjectMeta)
	if err != nil {
		return
	}
	err = util.WaitUntilSidecarRemoved(c.kubeClient, resource.Namespace, resource.Spec.Selector)
	return err
}

package controller

import (
	"errors"
	"fmt"
	"reflect"
	"time"

	acrt "github.com/appscode/go/runtime"
	"github.com/appscode/log"
	sapi "github.com/appscode/stash/api"
	"github.com/appscode/stash/pkg/util"
	kerr "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/apimachinery/pkg/watch"
	apiv1 "k8s.io/client-go/pkg/api/v1"
	apps "k8s.io/client-go/pkg/apis/apps/v1beta1"
	"k8s.io/client-go/tools/cache"
)

// Blocks caller. Intended to be called as a Go routine.
func (c *Controller) WatchDeploymentApps() {
	if !util.IsPreferredAPIResource(c.kubeClient, apps.SchemeGroupVersion.String(), "Deployment") {
		log.Warningf("Skipping watching non-preferred GroupVersion:%s Kind:%s", apps.SchemeGroupVersion.String(), "Deployment")
		return
	}

	defer acrt.HandleCrash()

	lw := &cache.ListWatch{
		ListFunc: func(opts metav1.ListOptions) (runtime.Object, error) {
			return c.kubeClient.AppsV1beta1().Deployments(apiv1.NamespaceAll).List(metav1.ListOptions{})
		},
		WatchFunc: func(options metav1.ListOptions) (watch.Interface, error) {
			return c.kubeClient.AppsV1beta1().Deployments(apiv1.NamespaceAll).Watch(metav1.ListOptions{})
		},
	}
	_, ctrl := cache.NewInformer(lw,
		&apps.Deployment{},
		c.syncPeriod,
		cache.ResourceEventHandlerFuncs{
			AddFunc: func(obj interface{}) {
				if resource, ok := obj.(*apps.Deployment); ok {
					log.Infof("Deployment %s@%s added", resource.Name, resource.Namespace)

					restic, err := util.FindRestic(c.stashClient, resource.ObjectMeta)
					if err != nil {
						log.Errorf("Error while searching Restic for Deployment %s@%s.", resource.Name, resource.Namespace)
						return
					}
					if restic == nil {
						log.Errorf("No Restic found for Deployment %s@%s.", resource.Name, resource.Namespace)
						return
					}
					c.EnsureDeploymentAppSidecar(resource, nil, restic)
				}
			},
			UpdateFunc: func(old, new interface{}) {
				oldObj, ok := old.(*apps.Deployment)
				if !ok {
					log.Errorln(errors.New("Invalid Deployment object"))
					return
				}
				newObj, ok := new.(*apps.Deployment)
				if !ok {
					log.Errorln(errors.New("Invalid Deployment object"))
					return
				}
				if !reflect.DeepEqual(oldObj.Labels, newObj.Labels) {
					oldRestic, err := util.FindRestic(c.stashClient, oldObj.ObjectMeta)
					if err != nil {
						log.Errorf("Error while searching Restic for Deployment %s@%s.", oldObj.Name, oldObj.Namespace)
						return
					}
					newRestic, err := util.FindRestic(c.stashClient, newObj.ObjectMeta)
					if err != nil {
						log.Errorf("Error while searching Restic for Deployment %s@%s.", newObj.Name, newObj.Namespace)
						return
					}
					if util.ResticEqual(oldRestic, newRestic) {
						return
					}
					if newRestic != nil {
						c.EnsureDeploymentAppSidecar(newObj, oldRestic, newRestic)
					} else if oldRestic != nil {
						c.EnsureDeploymentAppSidecarDeleted(newObj, oldRestic)
					}
				}
			},
		},
	)
	ctrl.Run(wait.NeverStop)
}

func (c *Controller) EnsureDeploymentAppSidecar(resource *apps.Deployment, old, new *sapi.Restic) (err error) {
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

	attempt := 0
	for ; attempt < maxAttempts; attempt = attempt + 1 {
		if name := util.GetString(resource.Annotations, sapi.ConfigName); name != "" && name != new.Name {
			log.Infof("Restic %s sidecar already added for Deployment %s@%s.", name, resource.Name, resource.Namespace)
			return nil
		}

		resource.Spec.Template.Spec.Containers = util.UpsertContainer(resource.Spec.Template.Spec.Containers, util.CreateSidecarContainer(new, c.SidecarImageTag, "Deployment/"+resource.Name))
		resource.Spec.Template.Spec.Volumes = util.UpsertScratchVolume(resource.Spec.Template.Spec.Volumes)
		resource.Spec.Template.Spec.Volumes = util.UpsertDownwardVolume(resource.Spec.Template.Spec.Volumes)
		resource.Spec.Template.Spec.Volumes = util.MergeLocalVolume(resource.Spec.Template.Spec.Volumes, old, new)

		if resource.Annotations == nil {
			resource.Annotations = make(map[string]string)
		}
		resource.Annotations[sapi.ConfigName] = new.Name
		resource.Annotations[sapi.VersionTag] = c.SidecarImageTag
		_, err = c.kubeClient.AppsV1beta1().Deployments(resource.Namespace).Update(resource)
		if err == nil {
			break
		}
		log.Errorf("Attempt %d failed to add sidecar for Deployment %s@%s due to %s.", attempt, resource.Name, resource.Namespace, err)
		time.Sleep(updateRetryInterval)
		if kerr.IsConflict(err) {
			resource, err = c.kubeClient.AppsV1beta1().Deployments(resource.Namespace).Get(resource.Name, metav1.GetOptions{})
			if err != nil {
				return
			}
		}
	}
	if attempt >= maxAttempts {
		err = fmt.Errorf("Failed to add sidecar for Deployment %s@%s after %d attempts.", resource.Name, resource.Namespace, attempt)
		return
	}

	err = util.WaitUntilDeploymentAppReady(c.kubeClient, resource.ObjectMeta)
	if err != nil {
		return
	}
	err = util.WaitUntilSidecarAdded(c.kubeClient, resource.Namespace, resource.Spec.Selector)
	return err
}

func (c *Controller) EnsureDeploymentAppSidecarDeleted(resource *apps.Deployment, restic *sapi.Restic) (err error) {
	defer func() {
		if err != nil {
			sidecarFailedToDelete()
			return
		}
		sidecarSuccessfullyDeleted()
	}()

	attempt := 0
	for ; attempt < maxAttempts; attempt = attempt + 1 {
		if name := util.GetString(resource.Annotations, sapi.ConfigName); name == "" {
			log.Infof("Restic sidecar already removed for Deployment %s@%s.", resource.Name, resource.Namespace)
			return nil
		}

		resource.Spec.Template.Spec.Containers = util.EnsureContainerDeleted(resource.Spec.Template.Spec.Containers, util.StashContainer)
		resource.Spec.Template.Spec.Volumes = util.EnsureVolumeDeleted(resource.Spec.Template.Spec.Volumes, util.ScratchDirVolumeName)
		resource.Spec.Template.Spec.Volumes = util.EnsureVolumeDeleted(resource.Spec.Template.Spec.Volumes, util.PodinfoVolumeName)
		if restic.Spec.Backend.Local != nil {
			resource.Spec.Template.Spec.Volumes = util.EnsureVolumeDeleted(resource.Spec.Template.Spec.Volumes, util.LocalVolumeName)
		}
		if resource.Annotations != nil {
			delete(resource.Annotations, sapi.ConfigName)
			delete(resource.Annotations, sapi.VersionTag)
		}
		_, err = c.kubeClient.AppsV1beta1().Deployments(resource.Namespace).Update(resource)
		if err == nil {
			break
		}
		log.Errorf("Attempt %d failed to delete sidecar for Deployment %s@%s due to %s.", attempt, resource.Name, resource.Namespace, err)
		time.Sleep(updateRetryInterval)
		if kerr.IsConflict(err) {
			resource, err = c.kubeClient.AppsV1beta1().Deployments(resource.Namespace).Get(resource.Name, metav1.GetOptions{})
			if err != nil {
				return
			}
		}
	}
	if attempt >= maxAttempts {
		err = fmt.Errorf("Failed to delete sidecar for Deployment %s@%s after %d attempts.", resource.Name, resource.Namespace, attempt)
		return
	}

	err = util.WaitUntilDeploymentAppReady(c.kubeClient, resource.ObjectMeta)
	if err != nil {
		return
	}
	err = util.WaitUntilSidecarRemoved(c.kubeClient, resource.Namespace, resource.Spec.Selector)
	return err
}

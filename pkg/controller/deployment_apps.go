package controller

import (
	acrt "github.com/appscode/go/runtime"
	"github.com/appscode/log"
	sapi "github.com/appscode/stash/api"
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
	if !c.IsPreferredAPIResource(apps.SchemeGroupVersion.String(), "Deployment") {
		log.Warningf("Skipping watching non-preferred GroupVersion:%s Kind:%s", apps.SchemeGroupVersion.String(), "Deployment")
		return
	}

	defer acrt.HandleCrash()

	lw := &cache.ListWatch{
		ListFunc: func(opts metav1.ListOptions) (runtime.Object, error) {
			return c.KubeClient.AppsV1beta1().Deployments(apiv1.NamespaceAll).List(metav1.ListOptions{})
		},
		WatchFunc: func(options metav1.ListOptions) (watch.Interface, error) {
			return c.KubeClient.AppsV1beta1().Deployments(apiv1.NamespaceAll).Watch(metav1.ListOptions{})
		},
	}
	_, ctrl := cache.NewInformer(lw,
		&apps.Deployment{},
		c.SyncPeriod,
		cache.ResourceEventHandlerFuncs{
			AddFunc: func(obj interface{}) {
				if resource, ok := obj.(*apps.Deployment); ok {
					log.Infof("Deployment %s@%s added", resource.Name, resource.Namespace)

					if name := getString(resource.Annotations, sapi.ConfigName); name != "" {
						log.Infof("Restic sidecar already exists for Deployment %s@%s.", resource.Name, resource.Namespace)
						return
					}

					restic, err := c.FindRestic(resource.ObjectMeta)
					if err != nil {
						log.Errorf("Error while searching Restic for Deployment %s@%s.", resource.Name, resource.Namespace)
						return
					}
					if restic == nil {
						log.Errorf("No Restic found for Deployment %s@%s.", resource.Name, resource.Namespace)
						return
					}
					c.EnsureDeploymentAppSidecar(resource, restic)
				}
			},
		},
	)
	ctrl.Run(wait.NeverStop)
}

func (c *Controller) EnsureDeploymentAppSidecar(resource *apps.Deployment, restic *sapi.Restic) {
	resource.Spec.Template.Spec.Containers = append(resource.Spec.Template.Spec.Containers, c.GetSidecarContainer(restic))
	resource.Spec.Template.Spec.Volumes = append(resource.Spec.Template.Spec.Volumes, restic.Spec.Destination.Volume)
	if resource.Annotations == nil {
		resource.Annotations = make(map[string]string)
	}
	resource.Annotations[sapi.ConfigName] = restic.Name
	resource.Annotations[sapi.VersionTag] = c.SidecarImageTag

	resource, err := c.KubeClient.AppsV1beta1().Deployments(resource.Namespace).Update(resource)
	if kerr.IsNotFound(err) {
		return
	} else if err != nil {
		sidecarFailedToAdd()
		log.Errorf("Failed to add sidecar for Deployment %s@%s.", resource.Name, resource.Namespace)
		return
	}
	sidecarSuccessfullyAdd()
	c.restartPods(resource.Namespace, resource.Spec.Selector)
}

func (c *Controller) EnsureDeploymentAppSidecarDeleted(resource *apps.Deployment, restic *sapi.Restic) error {
	resource.Spec.Template.Spec.Containers = removeContainer(resource.Spec.Template.Spec.Containers, ContainerName)
	resource.Spec.Template.Spec.Volumes = removeVolume(resource.Spec.Template.Spec.Volumes, restic.Spec.Destination.Volume.Name)
	if resource.Annotations != nil {
		delete(resource.Annotations, sapi.ConfigName)
		delete(resource.Annotations, sapi.VersionTag)
	}

	resource, err := c.KubeClient.AppsV1beta1().Deployments(resource.Namespace).Update(resource)
	if kerr.IsNotFound(err) {
		return nil
	} else if err != nil {
		sidecarFailedToAdd()
		log.Errorf("Failed to add sidecar for Deployment %s@%s.", resource.Name, resource.Namespace)
		return err
	}
	sidecarSuccessfullyDeleted()
	c.restartPods(resource.Namespace, resource.Spec.Selector)
	return nil
}

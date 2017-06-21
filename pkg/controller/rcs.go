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
	"k8s.io/client-go/tools/cache"
)

// Blocks caller. Intended to be called as a Go routine.
func (c *Controller) WatchReplicationControllers() {
	if !c.IsPreferredAPIResource("core/v1", "ReplicationController") {
		return
	}

	defer acrt.HandleCrash()

	lw := &cache.ListWatch{
		ListFunc: func(opts metav1.ListOptions) (runtime.Object, error) {
			return c.KubeClient.CoreV1().ReplicationControllers(apiv1.NamespaceAll).List(metav1.ListOptions{})
		},
		WatchFunc: func(options metav1.ListOptions) (watch.Interface, error) {
			return c.KubeClient.CoreV1().ReplicationControllers(apiv1.NamespaceAll).Watch(metav1.ListOptions{})
		},
	}
	_, ctrl := cache.NewInformer(lw,
		&apiv1.ReplicationController{},
		c.SyncPeriod,
		cache.ResourceEventHandlerFuncs{
			AddFunc: func(obj interface{}) {
				if resource, ok := obj.(*apiv1.ReplicationController); ok {
					log.Infof("ReplicationController %s@%s added", resource.Name, resource.Namespace)

					if name := getString(resource.Annotations, sapi.ConfigName); name != "" {
						log.Infof("Restic sidecar already exists for ReplicationController %s@%s.", resource.Name, resource.Namespace)
						return
					}

					restic, err := c.FindRestic(resource.ObjectMeta)
					if err != nil {
						log.Errorf("Error while searching Restic for ReplicationController %s@%s.", resource.Name, resource.Namespace)
						return
					}
					if restic != nil {
						log.Errorf("No Restic found for ReplicationController %s@%s.", resource.Name, resource.Namespace)
						return
					}
					c.EnsureReplicationControllerSidecar(resource, restic)
				}
			},
		},
	)
	ctrl.Run(wait.NeverStop)
}

func (c *Controller) EnsureReplicationControllerSidecar(resouce *apiv1.ReplicationController, restic *sapi.Restic) {
	resouce.Spec.Template.Spec.Containers = append(resouce.Spec.Template.Spec.Containers, c.GetSidecarContainer(restic))
	resouce.Spec.Template.Spec.Volumes = append(resouce.Spec.Template.Spec.Volumes, restic.Spec.Destination.Volume)
	if resouce.Annotations == nil {
		resouce.Annotations = make(map[string]string)
	}
	resouce.Annotations[sapi.ConfigName] = restic.Name
	resouce.Annotations[sapi.VersionTag] = c.SidecarImageTag

	resouce, err := c.KubeClient.CoreV1().ReplicationControllers(resouce.Namespace).Update(resouce)
	if kerr.IsNotFound(err) {
		return
	} else if err != nil {
		sidecarFailedToAdd()
		log.Errorf("Failed to add sidecar for ReplicationController %s@%s.", resouce.Name, resouce.Namespace)
		return
	}
	sidecarSuccessfullyAdd()
	c.restartPods(resouce.Namespace, &metav1.LabelSelector{MatchLabels: resouce.Spec.Selector})
}

func (c *Controller) EnsureReplicationControllerSidecarDeleted(resource *apiv1.ReplicationController, restic *sapi.Restic) error {
	resource.Spec.Template.Spec.Containers = removeContainer(resource.Spec.Template.Spec.Containers, ContainerName)
	resource.Spec.Template.Spec.Volumes = removeVolume(resource.Spec.Template.Spec.Volumes, restic.Spec.Destination.Volume.Name)
	if resource.Annotations != nil {
		delete(resource.Annotations, sapi.ConfigName)
		delete(resource.Annotations, sapi.VersionTag)
	}

	resouce, err := c.KubeClient.CoreV1().ReplicationControllers(resource.Namespace).Update(resource)
	if kerr.IsNotFound(err) {
		return nil
	} else if err != nil {
		sidecarFailedToAdd()
		log.Errorf("Failed to add sidecar for ReplicationController %s@%s.", resouce.Name, resouce.Namespace)
		return err
	}
	sidecarSuccessfullyDeleted()
	c.restartPods(resouce.Namespace, &metav1.LabelSelector{MatchLabels: resouce.Spec.Selector})
	return nil
}

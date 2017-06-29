package controller

import (
	"errors"

	acrt "github.com/appscode/go/runtime"
	"github.com/appscode/log"
	sapi "github.com/appscode/stash/api"
	"github.com/appscode/stash/pkg/util"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/apimachinery/pkg/watch"
	apiv1 "k8s.io/client-go/pkg/api/v1"
	apps "k8s.io/client-go/pkg/apis/apps/v1beta1"
	extensions "k8s.io/client-go/pkg/apis/extensions/v1beta1"
	"k8s.io/client-go/tools/cache"
)

// Blocks caller. Intended to be called as a Go routine.
func (c *Controller) WatchRestics() {
	defer acrt.HandleCrash()

	lw := &cache.ListWatch{
		ListFunc: func(opts metav1.ListOptions) (runtime.Object, error) {
			return c.stashClient.Restics(apiv1.NamespaceAll).List(metav1.ListOptions{})
		},
		WatchFunc: func(options metav1.ListOptions) (watch.Interface, error) {
			return c.stashClient.Restics(apiv1.NamespaceAll).Watch(metav1.ListOptions{})
		},
	}
	_, ctrl := cache.NewInformer(lw,
		&sapi.Restic{},
		c.syncPeriod,
		cache.ResourceEventHandlerFuncs{
			AddFunc: func(obj interface{}) {
				if resource, ok := obj.(*sapi.Restic); ok {
					c.EnsureSidecar(nil, resource)
				}
			},
			UpdateFunc: func(old, new interface{}) {
				oldObj, ok := old.(*sapi.Restic)
				if !ok {
					log.Errorln(errors.New("Invalid Restic object"))
					return
				}
				newObj, ok := new.(*sapi.Restic)
				if !ok {
					log.Errorln(errors.New("Invalid Restic object"))
					return
				}
				if !util.ResticEqual(oldObj, newObj) {
					c.EnsureSidecar(oldObj, newObj)
				}
			},
			DeleteFunc: func(obj interface{}) {
				if resource, ok := obj.(*sapi.Restic); ok {
					c.EnsureSidecarDeleted(resource)
				}
			},
		},
	)
	ctrl.Run(wait.NeverStop)
}

func (c *Controller) EnsureSidecar(old, new *sapi.Restic) {
	var oldOpt, newOpt *metav1.ListOptions
	if old != nil {
		oldSelector, err := metav1.LabelSelectorAsSelector(&old.Spec.Selector)
		if err != nil {
			return
		}
		oldOpt = &metav1.ListOptions{LabelSelector: oldSelector.String()}
	}

	newSelector, err := metav1.LabelSelectorAsSelector(&new.Spec.Selector)
	if err != nil {
		return
	}
	newOpt = &metav1.ListOptions{LabelSelector: newSelector.String()}

	{
		oldObjs := make(map[string]apiv1.ReplicationController)
		if oldOpt != nil {
			if resources, err := c.kubeClient.CoreV1().ReplicationControllers(new.Namespace).List(*oldOpt); err == nil {
				for _, resource := range resources.Items {
					oldObjs[resource.Name] = resource
				}
			}
		}

		if resources, err := c.kubeClient.CoreV1().ReplicationControllers(new.Namespace).List(*newOpt); err == nil {
			for _, resource := range resources.Items {
				delete(oldObjs, resource.Name)
				go c.EnsureReplicationControllerSidecar(&resource, old, new)
			}
		}
		for _, resource := range oldObjs {
			go c.EnsureReplicationControllerSidecarDeleted(&resource, old)
		}
	}

	{
		oldObjs := make(map[string]extensions.ReplicaSet)
		if oldOpt != nil {
			if resources, err := c.kubeClient.ExtensionsV1beta1().ReplicaSets(new.Namespace).List(*oldOpt); err == nil {
				for _, resource := range resources.Items {
					oldObjs[resource.Name] = resource
				}
			}
		}

		if resources, err := c.kubeClient.ExtensionsV1beta1().ReplicaSets(new.Namespace).List(*newOpt); err == nil {
			for _, resource := range resources.Items {
				delete(oldObjs, resource.Name)
				go c.EnsureReplicaSetSidecar(&resource, old, new)
			}
		}
		for _, resource := range oldObjs {
			go c.EnsureReplicaSetSidecarDeleted(&resource, old)
		}
	}

	{
		oldObjs := make(map[string]extensions.Deployment)
		if oldOpt != nil {
			if resources, err := c.kubeClient.ExtensionsV1beta1().Deployments(new.Namespace).List(*oldOpt); err == nil {
				for _, resource := range resources.Items {
					oldObjs[resource.Name] = resource
				}
			}
		}

		if resources, err := c.kubeClient.ExtensionsV1beta1().Deployments(new.Namespace).List(*newOpt); err == nil {
			for _, resource := range resources.Items {
				delete(oldObjs, resource.Name)
				go c.EnsureDeploymentExtensionSidecar(&resource, old, new)
			}
		}
		for _, resource := range oldObjs {
			go c.EnsureDeploymentExtensionSidecarDeleted(&resource, old)
		}
	}

	{
		if util.IsPreferredAPIResource(c.kubeClient, apps.SchemeGroupVersion.String(), "Deployment") {
			oldObjs := make(map[string]apps.Deployment)
			if oldOpt != nil {
				if resources, err := c.kubeClient.AppsV1beta1().Deployments(new.Namespace).List(*oldOpt); err == nil {
					for _, resource := range resources.Items {
						oldObjs[resource.Name] = resource
					}
				}
			}

			if resources, err := c.kubeClient.AppsV1beta1().Deployments(new.Namespace).List(*newOpt); err == nil {
				for _, resource := range resources.Items {
					delete(oldObjs, resource.Name)
					go c.EnsureDeploymentAppSidecar(&resource, old, new)
				}
			}
			for _, resource := range oldObjs {
				go c.EnsureDeploymentAppSidecarDeleted(&resource, old)
			}
		}
	}

	{
		oldObjs := make(map[string]extensions.DaemonSet)
		if oldOpt != nil {
			if resources, err := c.kubeClient.ExtensionsV1beta1().DaemonSets(new.Namespace).List(*oldOpt); err == nil {
				for _, resource := range resources.Items {
					oldObjs[resource.Name] = resource
				}
			}
		}

		if resources, err := c.kubeClient.ExtensionsV1beta1().DaemonSets(new.Namespace).List(*newOpt); err == nil {
			for _, resource := range resources.Items {
				delete(oldObjs, resource.Name)
				go c.EnsureDaemonSetSidecar(&resource, old, new)
			}
		}
		for _, resource := range oldObjs {
			go c.EnsureDaemonSetSidecarDeleted(&resource, old)
		}
	}
}

func (c *Controller) EnsureSidecarDeleted(restic *sapi.Restic) {
	sel, err := metav1.LabelSelectorAsSelector(&restic.Spec.Selector)
	if err != nil {
		return
	}
	opt := metav1.ListOptions{LabelSelector: sel.String()}

	if resources, err := c.kubeClient.CoreV1().ReplicationControllers(restic.Namespace).List(opt); err == nil {
		for _, resource := range resources.Items {
			go c.EnsureReplicationControllerSidecarDeleted(&resource, restic)
		}
	}

	if resources, err := c.kubeClient.ExtensionsV1beta1().ReplicaSets(restic.Namespace).List(opt); err == nil {
		for _, resource := range resources.Items {
			go c.EnsureReplicaSetSidecarDeleted(&resource, restic)
		}
	}
	if resources, err := c.kubeClient.ExtensionsV1beta1().Deployments(restic.Namespace).List(opt); err == nil {
		for _, resource := range resources.Items {
			go c.EnsureDeploymentExtensionSidecarDeleted(&resource, restic)
		}
	}
	if resources, err := c.kubeClient.AppsV1beta1().Deployments(restic.Namespace).List(opt); err == nil {
		for _, resource := range resources.Items {
			go c.EnsureDeploymentAppSidecarDeleted(&resource, restic)
		}
	}
	if resources, err := c.kubeClient.ExtensionsV1beta1().DaemonSets(restic.Namespace).List(opt); err == nil {
		for _, resource := range resources.Items {
			go c.EnsureDaemonSetSidecarDeleted(&resource, restic)
		}
	}
}

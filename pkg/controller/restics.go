package controller

import (
	"errors"
	"reflect"

	acrt "github.com/appscode/go/runtime"
	"github.com/appscode/log"
	sapi "github.com/appscode/stash/api"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/apimachinery/pkg/watch"
	apiv1 "k8s.io/client-go/pkg/api/v1"
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
				if !reflect.DeepEqual(oldObj.Spec, newObj.Spec) {
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
	sel, err := metav1.LabelSelectorAsSelector(&new.Spec.Selector)
	if err != nil {
		return
	}
	opt := metav1.ListOptions{LabelSelector: sel.String()}

	if resources, err := c.kubeClient.CoreV1().ReplicationControllers(new.Namespace).List(opt); err == nil {
		for _, resource := range resources.Items {
			go c.EnsureReplicationControllerSidecar(&resource, old, new)
		}
	}

	if resources, err := c.kubeClient.ExtensionsV1beta1().ReplicaSets(new.Namespace).List(opt); err == nil {
		for _, resource := range resources.Items {
			go c.EnsureReplicaSetSidecar(&resource, old, new)
		}
	}

	if resources, err := c.kubeClient.ExtensionsV1beta1().Deployments(new.Namespace).List(opt); err == nil {
		for _, resource := range resources.Items {
			go c.EnsureDeploymentExtensionSidecar(&resource, old, new)
		}
	}

	if resources, err := c.kubeClient.AppsV1beta1().Deployments(new.Namespace).List(opt); err == nil {
		for _, resource := range resources.Items {
			go c.EnsureDeploymentAppSidecar(&resource, old, new)
		}
	}

	if resources, err := c.kubeClient.ExtensionsV1beta1().DaemonSets(new.Namespace).List(opt); err == nil {
		for _, resource := range resources.Items {
			go c.EnsureDaemonSetSidecar(&resource, old, new)
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

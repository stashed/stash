package watcher

import (
	acs "github.com/appscode/k8s-addons/client/clientset"
	"github.com/appscode/k8s-addons/pkg/events"
	kapi "k8s.io/kubernetes/pkg/api"
	clientset "k8s.io/kubernetes/pkg/client/clientset_generated/internalclientset"
	"k8s.io/kubernetes/pkg/fields"
	"k8s.io/kubernetes/pkg/runtime"
	"k8s.io/kubernetes/pkg/watch"
)

func IngressListFunc(c clientset.Interface) func(kapi.ListOptions) (runtime.Object, error) {
	return func(opts kapi.ListOptions) (runtime.Object, error) {
		return c.Extensions().Ingresses(kapi.NamespaceAll).List(opts)
	}
}

func IngressWatchFunc(c clientset.Interface) func(options kapi.ListOptions) (watch.Interface, error) {
	return func(options kapi.ListOptions) (watch.Interface, error) {
		return c.Extensions().Ingresses(kapi.NamespaceAll).Watch(options)
	}
}

func ExtendedIngressListFunc(c acs.AppsCodeExtensionInterface) func(kapi.ListOptions) (runtime.Object, error) {
	return func(opts kapi.ListOptions) (runtime.Object, error) {
		return c.Ingress(kapi.NamespaceAll).List(opts)
	}
}

func ExtendedIngressWatchFunc(c acs.AppsCodeExtensionInterface) func(options kapi.ListOptions) (watch.Interface, error) {
	return func(options kapi.ListOptions) (watch.Interface, error) {
		return c.Ingress(kapi.NamespaceAll).Watch(options)
	}
}

func DaemonSetListFunc(c clientset.Interface) func(kapi.ListOptions) (runtime.Object, error) {
	return func(opts kapi.ListOptions) (runtime.Object, error) {
		return c.Extensions().DaemonSets(kapi.NamespaceAll).List(opts)
	}
}

func DaemonSetWatchFunc(c clientset.Interface) func(options kapi.ListOptions) (watch.Interface, error) {
	return func(options kapi.ListOptions) (watch.Interface, error) {
		return c.Extensions().DaemonSets(kapi.NamespaceAll).Watch(options)
	}
}

func ReplicaSetListFunc(c clientset.Interface) func(kapi.ListOptions) (runtime.Object, error) {
	return func(opts kapi.ListOptions) (runtime.Object, error) {
		return c.Extensions().ReplicaSets(kapi.NamespaceAll).List(opts)
	}
}

func ReplicaSetWatchFunc(c clientset.Interface) func(options kapi.ListOptions) (watch.Interface, error) {
	return func(options kapi.ListOptions) (watch.Interface, error) {
		return c.Extensions().ReplicaSets(kapi.NamespaceAll).Watch(options)
	}
}

func StatefulSetListFunc(c clientset.Interface) func(kapi.ListOptions) (runtime.Object, error) {
	return func(opts kapi.ListOptions) (runtime.Object, error) {
		return c.Apps().StatefulSets(kapi.NamespaceAll).List(opts)
	}
}

func StatefulSetWatchFunc(c clientset.Interface) func(options kapi.ListOptions) (watch.Interface, error) {
	return func(options kapi.ListOptions) (watch.Interface, error) {
		return c.Apps().StatefulSets(kapi.NamespaceAll).Watch(options)
	}
}

func AlertListFunc(c acs.AppsCodeExtensionInterface) func(kapi.ListOptions) (runtime.Object, error) {
	return func(opts kapi.ListOptions) (runtime.Object, error) {
		return c.Alert(kapi.NamespaceAll).List(opts)
	}
}

func AlertWatchFunc(c acs.AppsCodeExtensionInterface) func(options kapi.ListOptions) (watch.Interface, error) {
	return func(options kapi.ListOptions) (watch.Interface, error) {
		return c.Alert(kapi.NamespaceAll).Watch(options)
	}
}

func CertificateListFunc(c acs.AppsCodeExtensionInterface) func(kapi.ListOptions) (runtime.Object, error) {
	return func(opts kapi.ListOptions) (runtime.Object, error) {
		return c.Certificate(kapi.NamespaceAll).List(opts)
	}
}

func CertificateWatchFunc(c acs.AppsCodeExtensionInterface) func(options kapi.ListOptions) (watch.Interface, error) {
	return func(options kapi.ListOptions) (watch.Interface, error) {
		return c.Certificate(kapi.NamespaceAll).Watch(options)
	}
}

func AlertEventListFunc(c clientset.Interface) func(kapi.ListOptions) (runtime.Object, error) {
	return func(opts kapi.ListOptions) (runtime.Object, error) {
		sets := fields.Set{
			kapi.EventTypeField:         kapi.EventTypeNormal,
			kapi.EventReasonField:       events.EventReasonAlertAcknowledgement.String(),
			kapi.EventInvolvedKindField: events.ObjectKindAlert.String(),
		}
		fieldSelector := fields.SelectorFromSet(sets)

		opts.FieldSelector = fieldSelector
		return c.Core().Events(kapi.NamespaceAll).List(opts)
	}
}

func AlertEventWatchFunc(c clientset.Interface) func(options kapi.ListOptions) (watch.Interface, error) {
	return func(options kapi.ListOptions) (watch.Interface, error) {
		sets := fields.Set{
			kapi.EventTypeField:         kapi.EventTypeNormal,
			kapi.EventReasonField:       events.EventReasonAlertAcknowledgement.String(),
			kapi.EventInvolvedKindField: events.ObjectKindAlert.String(),
		}
		fieldSelector := fields.SelectorFromSet(sets)

		options.FieldSelector = fieldSelector
		return c.Core().Events(kapi.NamespaceAll).Watch(options)
	}
}

func NamespaceListFunc(c clientset.Interface) func(kapi.ListOptions) (runtime.Object, error) {
	return func(opts kapi.ListOptions) (runtime.Object, error) {
		return c.Core().Namespaces().List(opts)
	}
}

func NamespaceWatchFunc(c clientset.Interface) func(options kapi.ListOptions) (watch.Interface, error) {
	return func(options kapi.ListOptions) (watch.Interface, error) {
		return c.Core().Namespaces().Watch(options)
	}
}

func PodListFunc(c clientset.Interface) func(kapi.ListOptions) (runtime.Object, error) {
	return func(opts kapi.ListOptions) (runtime.Object, error) {
		return c.Core().Pods(kapi.NamespaceAll).List(opts)
	}
}

func PodWatchFunc(c clientset.Interface) func(options kapi.ListOptions) (watch.Interface, error) {
	return func(options kapi.ListOptions) (watch.Interface, error) {
		return c.Core().Pods(kapi.NamespaceAll).Watch(options)
	}
}

func ServiceListFunc(c clientset.Interface) func(kapi.ListOptions) (runtime.Object, error) {
	return func(opts kapi.ListOptions) (runtime.Object, error) {
		return c.Core().Services(kapi.NamespaceAll).List(opts)
	}
}

func ServiceWatchFunc(c clientset.Interface) func(options kapi.ListOptions) (watch.Interface, error) {
	return func(options kapi.ListOptions) (watch.Interface, error) {
		return c.Core().Services(kapi.NamespaceAll).Watch(options)
	}
}

func ReplicationControllerWatchFunc(c clientset.Interface) func(options kapi.ListOptions) (watch.Interface, error) {
	return func(options kapi.ListOptions) (watch.Interface, error) {
		return c.Core().ReplicationControllers(kapi.NamespaceAll).Watch(options)
	}
}

func ReplicationControllerListFunc(c clientset.Interface) func(kapi.ListOptions) (runtime.Object, error) {
	return func(opts kapi.ListOptions) (runtime.Object, error) {
		return c.Core().ReplicationControllers(kapi.NamespaceAll).List(opts)
	}
}

func EndpointListFunc(c clientset.Interface) func(kapi.ListOptions) (runtime.Object, error) {
	return func(opts kapi.ListOptions) (runtime.Object, error) {
		return c.Core().Endpoints(kapi.NamespaceAll).List(opts)
	}
}

func EndpointWatchFunc(c clientset.Interface) func(options kapi.ListOptions) (watch.Interface, error) {
	return func(options kapi.ListOptions) (watch.Interface, error) {
		return c.Core().Endpoints(kapi.NamespaceAll).Watch(options)
	}
}

func NodeListFunc(c clientset.Interface) func(kapi.ListOptions) (runtime.Object, error) {
	return func(opts kapi.ListOptions) (runtime.Object, error) {
		return c.Core().Nodes().List(opts)
	}
}

func NodeWatchFunc(c clientset.Interface) func(options kapi.ListOptions) (watch.Interface, error) {
	return func(options kapi.ListOptions) (watch.Interface, error) {
		return c.Core().Nodes().Watch(options)
	}
}

func DeploymentListFunc(c clientset.Interface) func(kapi.ListOptions) (runtime.Object, error) {
	return func(opts kapi.ListOptions) (runtime.Object, error) {
		return c.Extensions().Deployments(kapi.NamespaceAll).List(opts)
	}
}

func DeploymentWatchFunc(c clientset.Interface) func(options kapi.ListOptions) (watch.Interface, error) {
	return func(options kapi.ListOptions) (watch.Interface, error) {
		return c.Extensions().Deployments(kapi.NamespaceAll).Watch(options)
	}
}

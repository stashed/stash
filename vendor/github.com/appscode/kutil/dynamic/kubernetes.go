package dynamic

import (
	"context"
	"fmt"
	"time"

	"github.com/appscode/kutil"
	"github.com/appscode/kutil/core/v1"
	discovery_util "github.com/appscode/kutil/discovery"
	"github.com/pkg/errors"
	core "k8s.io/api/core/v1"
	kerr "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	utilerrors "k8s.io/apimachinery/pkg/util/errors"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
	watchtools "k8s.io/client-go/tools/watch"
)

func UntilHasLabel(config *rest.Config, gvk schema.GroupVersionKind, namespace, name string, key string, value *string, timeout time.Duration) (out string, err error) {
	return untilHasKey(config, gvk, namespace, name, func(obj metav1.Object) map[string]string { return obj.GetLabels() }, key, value, timeout)
}

func UntilHasAnnotation(config *rest.Config, gvk schema.GroupVersionKind, namespace, name string, key string, value *string, timeout time.Duration) (out string, err error) {
	return untilHasKey(config, gvk, namespace, name, func(obj metav1.Object) map[string]string { return obj.GetAnnotations() }, key, value, timeout)
}

func untilHasKey(
	config *rest.Config,
	gvk schema.GroupVersionKind,
	namespace, name string,
	fn func(metav1.Object) map[string]string,
	key string, value *string,
	timeout time.Duration) (out string, err error) {

	ctx := context.Background()
	if timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, timeout)
		defer cancel()
	}

	kc := kubernetes.NewForConfigOrDie(config)
	dc, err := dynamic.NewForConfig(config)
	if err != nil {
		return
	}

	gvr, err := discovery_util.ResourceForGVK(kc.Discovery(), gvk)
	if err != nil {
		return
	}

	var ri dynamic.ResourceInterface
	if namespace != "" {
		ri = dc.Resource(gvr).Namespace(namespace)
	} else {
		ri = dc.Resource(gvr)
	}

	lw := &cache.ListWatch{
		ListFunc: func(options metav1.ListOptions) (runtime.Object, error) {
			options.FieldSelector = fields.OneTermEqualSelector(kutil.ObjectNameField, name).String()
			return ri.List(options)
		},
		WatchFunc: func(options metav1.ListOptions) (watch.Interface, error) {
			options.FieldSelector = fields.OneTermEqualSelector(kutil.ObjectNameField, name).String()
			return ri.Watch(options)
		},
	}

	_, err = watchtools.UntilWithSync(ctx,
		lw,
		&unstructured.Unstructured{},
		nil,
		func(event watch.Event) (bool, error) {
			switch event.Type {
			case watch.Deleted:
				return false, nil
			case watch.Error:
				return false, errors.Wrap(err, "error watching")
			case watch.Added, watch.Modified:
				m, e2 := meta.Accessor(event.Object)
				if e2 != nil {
					return false, e2
				}
				var ok bool
				if out, ok = fn(m)[key]; ok && (value == nil || *value == out) {
					return true, nil
				}
				return false, nil // continue
			default:
				return false, fmt.Errorf("unexpected event type: %v", event.Type)
			}
		},
	)
	return
}

func DetectWorkload(config *rest.Config, resource schema.GroupVersionResource, namespace, name string) (*unstructured.Unstructured, schema.GroupVersionResource, error) {
	kc := kubernetes.NewForConfigOrDie(config)
	dc, err := dynamic.NewForConfig(config)
	if err != nil {
		return nil, resource, err
	}

	obj, err := dc.Resource(resource).Namespace(namespace).Get(name, metav1.GetOptions{})
	if err != nil {
		return nil, resource, err
	}
	return findWorkload(kc, dc, resource, obj)
}

func findWorkload(kc kubernetes.Interface, dc dynamic.Interface, resource schema.GroupVersionResource, obj *unstructured.Unstructured) (*unstructured.Unstructured, schema.GroupVersionResource, error) {
	m, err := meta.Accessor(obj)
	if err != nil {
		return nil, resource, err
	}
	for _, ref := range m.GetOwnerReferences() {
		if ref.Controller != nil && *ref.Controller {
			gvk := schema.FromAPIVersionAndKind(ref.APIVersion, ref.Kind)
			gvr, err := discovery_util.ResourceForGVK(kc.Discovery(), gvk)
			if err != nil {
				return nil, gvr, err
			}
			parent, err := dc.Resource(gvr).Namespace(m.GetNamespace()).Get(ref.Name, metav1.GetOptions{})
			if err != nil {
				return nil, gvr, err
			}
			return findWorkload(kc, dc, gvr, parent)
		}
	}
	return obj, resource, nil
}

func RemoveOwnerReferenceForItems(
	c dynamic.Interface,
	gvr schema.GroupVersionResource,
	namespace string,
	items []string,
	ref *core.ObjectReference,
) error {
	var ri dynamic.ResourceInterface
	if namespace == "" {
		ri = c.Resource(gvr)
	} else {
		ri = c.Resource(gvr).Namespace(namespace)
	}

	var errs []error
	for _, name := range items {
		item, err := ri.Get(name, metav1.GetOptions{})
		if err != nil {
			if !kerr.IsNotFound(err) {
				errs = append(errs, err)
			}
			continue
		}
		if _, _, err := Patch(c, gvr, item, func(in *unstructured.Unstructured) *unstructured.Unstructured {
			v1.RemoveOwnerReference(in, ref)
			return in
		}); err != nil && !kerr.IsNotFound(err) {
			errs = append(errs, err)
		}
	}
	return utilerrors.NewAggregate(errs)
}

func RemoveOwnerReferenceForSelector(
	c dynamic.Interface,
	gvr schema.GroupVersionResource,
	namespace string,
	selector labels.Selector,
	ref *core.ObjectReference,
) error {
	var ri dynamic.ResourceInterface
	if namespace == "" {
		ri = c.Resource(gvr)
	} else {
		ri = c.Resource(gvr).Namespace(namespace)
	}

	list, err := ri.List(metav1.ListOptions{LabelSelector: selector.String()})
	if err != nil {
		return err
	}

	var errs []error
	for _, item := range list.Items {
		if _, _, err := Patch(c, gvr, &item, func(in *unstructured.Unstructured) *unstructured.Unstructured {
			v1.RemoveOwnerReference(in, ref)
			return in
		}); err != nil && !kerr.IsNotFound(err) {
			errs = append(errs, err)
		}
	}
	return utilerrors.NewAggregate(errs)
}

func EnsureOwnerReferenceForItems(
	c dynamic.Interface,
	gvr schema.GroupVersionResource,
	namespace string,
	items []string,
	ref *core.ObjectReference,
) error {
	var ri dynamic.ResourceInterface
	if namespace == "" {
		ri = c.Resource(gvr)
	} else {
		ri = c.Resource(gvr).Namespace(namespace)
	}

	var errs []error
	for _, name := range items {
		item, err := ri.Get(name, metav1.GetOptions{})
		if err != nil {
			if !kerr.IsNotFound(err) {
				errs = append(errs, err)
			}
			continue
		}
		if _, _, err := Patch(c, gvr, item, func(in *unstructured.Unstructured) *unstructured.Unstructured {
			v1.EnsureOwnerReference(in, ref)
			return in
		}); err != nil && !kerr.IsNotFound(err) {
			errs = append(errs, err)
		}
	}
	return utilerrors.NewAggregate(errs)
}

func EnsureOwnerReferenceForSelector(
	c dynamic.Interface,
	gvr schema.GroupVersionResource,
	namespace string,
	selector labels.Selector,
	ref *core.ObjectReference,
) error {
	var ri dynamic.ResourceInterface
	if namespace == "" {
		ri = c.Resource(gvr)
	} else {
		ri = c.Resource(gvr).Namespace(namespace)
	}
	list, err := ri.List(metav1.ListOptions{LabelSelector: selector.String()})
	if err != nil {
		return err
	}

	var errs []error
	for _, item := range list.Items {
		if _, _, err := Patch(c, gvr, &item, func(in *unstructured.Unstructured) *unstructured.Unstructured {
			v1.EnsureOwnerReference(in, ref)
			return in
		}); err != nil && !kerr.IsNotFound(err) {
			errs = append(errs, err)
		}
	}
	return utilerrors.NewAggregate(errs)
}

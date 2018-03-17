package plugin

import (
	"fmt"

	"github.com/appscode/go/log"
	stringz "github.com/appscode/go/strings"
	"github.com/appscode/kutil/admission"
	"github.com/appscode/stash/pkg/controller"
	"github.com/appscode/stash/pkg/util"
	"k8s.io/api/core/v1"
	extensions "k8s.io/api/extensions/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

type DaemonSetMutator struct {
	Ctrl *controller.StashController
}

var _ admission.ResourceHandler = &DaemonSetMutator{}

func (handler *DaemonSetMutator) OnCreate(obj runtime.Object) (runtime.Object, error) {
	return handler.mutate(obj)
}

func (handler *DaemonSetMutator) OnUpdate(oldObj, newObj runtime.Object) (runtime.Object, error) {
	return handler.mutate(newObj)
}

func (handler *DaemonSetMutator) OnDelete(obj runtime.Object) error {
	// nothing to do
	return nil
}

func (handler *DaemonSetMutator) mutate(obj runtime.Object) (runtime.Object, error) {
	ds := obj.(*extensions.DaemonSet).DeepCopy()

	oldRestic, err := util.GetAppliedRestic(ds.Annotations)
	if err != nil {
		return nil, err
	}
	newRestic, err := util.FindRestic(handler.Ctrl.RstLister, ds.ObjectMeta)
	if err != nil {
		log.Errorf("Error while searching Restic for DaemonSet %s/%s.", ds.Name, ds.Namespace)
		return nil, err
	}

	if newRestic != nil && handler.Ctrl.EnableRBAC {
		sa := stringz.Val(ds.Spec.Template.Spec.ServiceAccountName, "default")
		if err != nil {
			return nil, err
		}
		ref := &v1.ObjectReference{
			Name:      ds.Name,
			Namespace: ds.Namespace,
		}
		err = handler.Ctrl.EnsureSidecarRoleBinding(ref, sa)
		if err != nil {
			return nil, err
		}
	}

	if newRestic != nil && !util.ResticEqual(oldRestic, newRestic) {
		if !newRestic.Spec.Paused {
			if newRestic.Spec.Backend.StorageSecretName == "" {
				err = fmt.Errorf("missing repository secret name for Restic %s/%s", newRestic.Namespace, newRestic.Name)
				return nil, err
			}
			_, err = handler.Ctrl.KubeClient.CoreV1().Secrets(ds.Namespace).Get(newRestic.Spec.Backend.StorageSecretName, metav1.GetOptions{})
			if err != nil {
				return nil, err
			}
			return handler.Ctrl.DaemonSetSidecarInjectionTransformerFunc(ds, oldRestic, newRestic), nil
		}
	} else if oldRestic != nil && newRestic == nil {
		return handler.Ctrl.DaemonSetSidecarDeletionTransformerFunc(ds, oldRestic), nil
	}
	return ds, nil
}

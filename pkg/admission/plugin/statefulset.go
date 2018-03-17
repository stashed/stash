package plugin

import (
	"fmt"

	"github.com/appscode/go/log"
	stringz "github.com/appscode/go/strings"
	"github.com/appscode/kutil/admission"
	"github.com/appscode/stash/pkg/controller"
	"github.com/appscode/stash/pkg/util"
	apps "k8s.io/api/apps/v1beta1"
	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

type StatefulSetMutator struct {
	Ctrl *controller.StashController
}

var _ admission.ResourceHandler = &StatefulSetMutator{}

func (handler *StatefulSetMutator) OnCreate(obj runtime.Object) (runtime.Object, error) {
	return handler.mutate(obj)
}

func (handler *StatefulSetMutator) OnUpdate(oldObj, newObj runtime.Object) (runtime.Object, error) {
	return handler.mutate(newObj)
}

func (handler *StatefulSetMutator) OnDelete(obj runtime.Object) error {
	// nothing to do
	fmt.Println("================== OnDelete() called")
	return nil
}

func (handler *StatefulSetMutator) mutate(obj runtime.Object) (runtime.Object, error) {
	ss := obj.(*apps.StatefulSet).DeepCopy()
	oldRestic, err := util.GetAppliedRestic(ss.Annotations)
	if err != nil {
		return nil, err
	}

	newRestic, err := util.FindRestic(handler.Ctrl.RstLister, ss.ObjectMeta)
	if err != nil {
		log.Errorf("Error while searching Restic for StatefulSet %s/%s.", ss.Name, ss.Namespace)
		return nil, err
	}

	if newRestic != nil && handler.Ctrl.EnableRBAC {
		sa := stringz.Val(ss.Spec.Template.Spec.ServiceAccountName, "default")
		if err != nil {
			return nil, err
		}
		ref := &v1.ObjectReference{
			Name:      ss.Name,
			Namespace: ss.Namespace,
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
			_, err = handler.Ctrl.KubeClient.CoreV1().Secrets(ss.Namespace).Get(newRestic.Spec.Backend.StorageSecretName, metav1.GetOptions{})
			if err != nil {
				return nil, err
			}
			return handler.Ctrl.StatefulSetSidecarInjectionTransformerFunc(ss, oldRestic, newRestic), nil
		}
	} else if oldRestic != nil && newRestic == nil {
		return handler.Ctrl.StatefulSetSidecarDeletionTransformerFunc(ss, oldRestic), nil
	}
	return ss, nil
}

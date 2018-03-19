package plugin

import (
	"fmt"

	"github.com/appscode/go/log"
	stringz "github.com/appscode/go/strings"
	"github.com/appscode/kutil/admission"
	"github.com/appscode/stash/pkg/controller"
	"github.com/appscode/stash/pkg/util"
	core "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

type ReplicationControllerMutator struct {
	Ctrl *controller.StashController
}

var _ admission.ResourceHandler = &ReplicationControllerMutator{}

func (handler *ReplicationControllerMutator) OnCreate(obj runtime.Object) (runtime.Object, error) {
	return handler.mutate(obj)
}

func (handler *ReplicationControllerMutator) OnUpdate(oldObj, newObj runtime.Object) (runtime.Object, error) {
	return handler.mutate(newObj)
}

func (handler *ReplicationControllerMutator) OnDelete(obj runtime.Object) error {
	// nothing to do
	return nil
}

func (handler *ReplicationControllerMutator) mutate(obj runtime.Object) (runtime.Object, error) {
	rc := obj.(*core.ReplicationController).DeepCopy()
	oldRestic, err := util.GetAppliedRestic(rc.Annotations)
	if err != nil {
		return nil, err
	}

	newRestic, err := util.FindRestic(handler.Ctrl.RstLister, rc.ObjectMeta)
	if err != nil {
		log.Errorf("Error while searching Restic for ReplicationController %s/%s.", rc.Name, rc.Namespace)
		return nil, err
	}

	if newRestic != nil && handler.Ctrl.EnableRBAC {
		sa := stringz.Val(rc.Spec.Template.Spec.ServiceAccountName, "default")
		if err != nil {
			return nil, err
		}
		ref := &core.ObjectReference{
			Name:      rc.Name,
			Namespace: rc.Namespace,
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
			_, err = handler.Ctrl.KubeClient.CoreV1().Secrets(rc.Namespace).Get(newRestic.Spec.Backend.StorageSecretName, metav1.GetOptions{})
			if err != nil {
				return nil, err
			}

			if rc.Annotations == nil {
				rc.Annotations = make(map[string]string)
			}
			rc.Annotations[util.ForceRestartType] = util.SideCarAdded
			rc.Annotations[util.BackupType] = string(newRestic.Spec.Type)
			return handler.Ctrl.ReplicationControllerSidecarInjectionTransformerFunc(rc, oldRestic, newRestic), nil
		}
	} else if oldRestic != nil && newRestic == nil {

		if rc.Annotations == nil {
			rc.Annotations = make(map[string]string)
		}
		rc.Annotations[util.ForceRestartType] = util.SideCarRemoved
		rc.Annotations[util.BackupType] = string(oldRestic.Spec.Type)

		return handler.Ctrl.ReplicationControllerSidecarDeletionTransformerFunc(rc, oldRestic), nil
	}
	return rc, nil
}

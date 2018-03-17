package plugin

import (
	"fmt"

	"github.com/appscode/go/log"
	stringz "github.com/appscode/go/strings"
	"github.com/appscode/kutil/admission"
	ext_util "github.com/appscode/kutil/extensions/v1beta1"
	"github.com/appscode/stash/pkg/controller"
	"github.com/appscode/stash/pkg/util"
	"k8s.io/api/core/v1"
	extensions "k8s.io/api/extensions/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

type ReplicaSetMutator struct {
	Ctrl *controller.StashController
}

var _ admission.ResourceHandler = &ReplicaSetMutator{}

func (handler *ReplicaSetMutator) OnCreate(obj runtime.Object) (runtime.Object, error) {
	return handler.mutate(obj)
}

func (handler *ReplicaSetMutator) OnUpdate(oldObj, newObj runtime.Object) (runtime.Object, error) {
	return handler.mutate(newObj)
}

func (handler *ReplicaSetMutator) OnDelete(obj runtime.Object) error {
	// nothing to do
	return nil
}

func (handler *ReplicaSetMutator) mutate(obj runtime.Object) (runtime.Object, error) {
	rs := obj.(*extensions.ReplicaSet).DeepCopy()

	if !ext_util.IsOwnedByDeployment(rs) {
		oldRestic, err := util.GetAppliedRestic(rs.Annotations)
		if err != nil {
			return nil, err
		}

		newRestic, err := util.FindRestic(handler.Ctrl.RstLister, rs.ObjectMeta)
		if err != nil {
			log.Errorf("Error while searching Restic for ReplicaSet %s/%s.", rs.Name, rs.Namespace)
			return nil, err
		}

		if newRestic != nil && handler.Ctrl.EnableRBAC {
			sa := stringz.Val(rs.Spec.Template.Spec.ServiceAccountName, "default")
			if err != nil {
				return nil, err
			}
			ref := &v1.ObjectReference{
				Name:      rs.Name,
				Namespace: rs.Namespace,
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
				_, err = handler.Ctrl.KubeClient.CoreV1().Secrets(rs.Namespace).Get(newRestic.Spec.Backend.StorageSecretName, metav1.GetOptions{})
				if err != nil {
					return nil, err
				}

				if rs.Annotations == nil {
					rs.Annotations = make(map[string]string)
				}
				rs.Annotations[util.ForceRestartType] = util.SideCarAdded
				rs.Annotations[util.BackupType] = string(newRestic.Spec.Type)

				return handler.Ctrl.ReplicaSetSidecarInjectionTransformerFunc(rs, oldRestic, newRestic), nil
			}
		} else if oldRestic != nil && newRestic == nil {
			if rs.Annotations == nil {
				rs.Annotations = make(map[string]string)
			}
			rs.Annotations[util.ForceRestartType] = util.SideCarRemoved
			rs.Annotations[util.BackupType] = string(oldRestic.Spec.Type)

			return handler.Ctrl.ReplicaSetSidecarDeletionTransformerFunc(rs, oldRestic), nil
		}
	}
	return rs, nil
}

package plugin

import (
	"fmt"

	"github.com/appscode/go/log"
	stringz "github.com/appscode/go/strings"
	"github.com/appscode/stash/pkg/controller"
	"github.com/appscode/stash/pkg/util"
	apps "k8s.io/api/apps/v1beta1"
	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type DeploymentMutator struct {
	Ctrl *controller.StashController
}

func (handler *DeploymentMutator) OnCreate(obj interface{}) (interface{}, error) {
	return handler.MutateDeployment(obj)
}

func (handler *DeploymentMutator) OnUpdate(oldObj, newObj interface{}) (interface{}, error) {
	return handler.MutateDeployment(newObj)
}

func (handler *DeploymentMutator) OnDelete(obj interface{}) error {
	// nothing to do
	return nil
}

func (handler *DeploymentMutator) MutateDeployment(obj interface{}) (interface{}, error) {
	dp := obj.(*apps.Deployment).DeepCopy()

	oldRestic, err := util.GetAppliedRestic(dp.Annotations)
	if err != nil {
		return nil, err
	}

	newRestic, err := util.FindRestic(handler.Ctrl.RstLister, dp.ObjectMeta)
	if err != nil {
		log.Errorf("Error while searching Restic for Deployment %s/%s.", dp.Name, dp.Namespace)
		return nil, err
	}

	if newRestic != nil && handler.Ctrl.EnableRBAC {
		sa := stringz.Val(dp.Spec.Template.Spec.ServiceAccountName, "default")
		if err != nil {
			return nil, err
		}
		ref := &v1.ObjectReference{
			Name:      dp.Name,
			Namespace: dp.Namespace,
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
			_, err = handler.Ctrl.KubeClient.CoreV1().Secrets(dp.Namespace).Get(newRestic.Spec.Backend.StorageSecretName, metav1.GetOptions{})
			if err != nil {
				return nil, err
			}
			return handler.Ctrl.DeploymentSidecarInjectionTransformerFunc(dp, oldRestic, newRestic), nil
		}
	} else if oldRestic != nil && newRestic == nil {
		return handler.Ctrl.DeploymentSidecarDeletionTransformerFunc(dp, oldRestic), nil
	}
	return dp, nil
}

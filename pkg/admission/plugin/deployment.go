package plugin

import (
	"fmt"

	"github.com/appscode/go/log"
	stringz "github.com/appscode/go/strings"
	core_util "github.com/appscode/kutil/core/v1"
	api "github.com/appscode/stash/apis/stash/v1alpha1"
	cs "github.com/appscode/stash/client/clientset/versioned"
	"github.com/appscode/stash/pkg/docker"
	"github.com/appscode/stash/pkg/util"
	//oneliner "github.com/the-redback/go-oneliners"
	apps "k8s.io/api/apps/v1beta1"
	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/tools/reference"
)

type DeploymentMutator struct {
	KubeClient     kubernetes.Interface
	StashClient    cs.Interface
	DockerRegistry string
	StashImageTag  string
	EnableRBAC     bool
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
	newRestic, err := FindNewRestic(handler.StashClient, dp.ObjectMeta)
	if err != nil {
		log.Errorf("Error while searching Restic for Deployment %s/%s.", dp.Name, dp.Namespace)
		return nil, err
	}
	if newRestic != nil && !util.ResticEqual(oldRestic, newRestic) {
		if !newRestic.Spec.Paused {
			return handler.ensureDeploymentSidecar(dp, oldRestic, newRestic)
		}
	} else if oldRestic != nil && newRestic == nil {
		return handler.ensureDeploymentSidecarDeleted(dp, oldRestic)
	}
	return dp, nil
}

func (c *DeploymentMutator) ensureDeploymentSidecar(obj *apps.Deployment, old, new *api.Restic) (*apps.Deployment, error) {
	image := docker.Docker{
		Registry: c.DockerRegistry,
		Image:    docker.ImageStash,
		Tag:      c.StashImageTag,
	}

	if new.Spec.Backend.StorageSecretName == "" {
		return nil, fmt.Errorf("missing repository secret name for Restic %s/%s", new.Namespace, new.Name)
	}
	_, err := c.KubeClient.CoreV1().Secrets(obj.Namespace).Get(new.Spec.Backend.StorageSecretName, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}

	if c.EnableRBAC {
		sa := stringz.Val(obj.Spec.Template.Spec.ServiceAccountName, "default")
		ref, err := reference.GetReference(scheme.Scheme, obj)
		// create rbac rules even through there is no valid reference.
		if err != nil {
			ref = &v1.ObjectReference{
				Name:      obj.Name,
				Namespace: obj.Namespace,
				Kind:      obj.Kind,
			}
		}
		err = c.ensureSidecarRoleBinding(ref, sa)
		if err != nil {
			return nil, err
		}
	}

	workload := api.LocalTypedReference{
		Kind: api.KindDeployment,
		Name: obj.Name,
	}

	if new.Spec.Type == api.BackupOffline {
		obj.Spec.Template.Spec.InitContainers = core_util.UpsertContainer(
			obj.Spec.Template.Spec.InitContainers,
			util.NewInitContainer(new, workload, image, c.EnableRBAC),
		)
	} else {
		obj.Spec.Template.Spec.Containers = core_util.UpsertContainer(
			obj.Spec.Template.Spec.Containers,
			util.NewSidecarContainer(new, workload, image),
		)
	}

	// keep existing image pull secrets
	obj.Spec.Template.Spec.ImagePullSecrets = core_util.MergeLocalObjectReferences(
		obj.Spec.Template.Spec.ImagePullSecrets,
		new.Spec.ImagePullSecrets,
	)

	obj.Spec.Template.Spec.Volumes = util.UpsertScratchVolume(obj.Spec.Template.Spec.Volumes)
	obj.Spec.Template.Spec.Volumes = util.UpsertDownwardVolume(obj.Spec.Template.Spec.Volumes)
	obj.Spec.Template.Spec.Volumes = util.MergeLocalVolume(obj.Spec.Template.Spec.Volumes, old, new)

	return obj, nil
}

func (c *DeploymentMutator) ensureDeploymentSidecarDeleted(obj *apps.Deployment, restic *api.Restic) (interface{}, error) {

	if restic.Spec.Type == api.BackupOffline {
		obj.Spec.Template.Spec.InitContainers = core_util.EnsureContainerDeleted(obj.Spec.Template.Spec.InitContainers, util.StashContainer)
	} else {
		obj.Spec.Template.Spec.Containers = core_util.EnsureContainerDeleted(obj.Spec.Template.Spec.Containers, util.StashContainer)
	}
	obj.Spec.Template.Spec.Volumes = util.EnsureVolumeDeleted(obj.Spec.Template.Spec.Volumes, util.ScratchDirVolumeName)
	obj.Spec.Template.Spec.Volumes = util.EnsureVolumeDeleted(obj.Spec.Template.Spec.Volumes, util.PodinfoVolumeName)
	if restic.Spec.Backend.Local != nil {
		obj.Spec.Template.Spec.Volumes = util.EnsureVolumeDeleted(obj.Spec.Template.Spec.Volumes, util.LocalVolumeName)
	}
	return obj, nil
}

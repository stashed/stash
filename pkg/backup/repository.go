package backup

import (
	"github.com/appscode/go/log"
	api "github.com/appscode/stash/apis/stash/v1alpha1"
	"github.com/appscode/stash/client/clientset/versioned/typed/stash/v1alpha1/util"
)

func (c *Controller) createRepositoryCrdIfNotExist(restic *api.Restic, prefix string) (*api.Repository, error) {
	repository := &api.Repository{}
	repository.Namespace = restic.Namespace
	repository.Name = c.opt.Workload.GetRepositoryCRDName(c.opt.PodName, c.opt.NodeName)

	repository.Labels = map[string]string{
		"restic":        restic.Name,
		"workload-kind": c.opt.Workload.Kind,
		"workload-name": c.opt.Workload.Name,
	}

	switch c.opt.Workload.Kind {
	case api.KindStatefulSet:
		repository.Labels["pod-name"] = c.opt.PodName
	case api.KindDaemonSet:
		repository.Labels["node-name"] = c.opt.NodeName
	}

	repository.Spec.Backend = *restic.Spec.Backend.DeepCopy()
	if repository.Spec.Backend.Local != nil {
		repository.Spec.Backend.Local.SubPath = prefix
	} else if repository.Spec.Backend.Azure != nil {
		repository.Spec.Backend.Azure.Prefix = prefix
	} else if repository.Spec.Backend.B2 != nil {
		repository.Spec.Backend.B2.Prefix = prefix
	} else if repository.Spec.Backend.GCS != nil {
		repository.Spec.Backend.GCS.Prefix = prefix
	} else if repository.Spec.Backend.S3 != nil {
		repository.Spec.Backend.S3.Prefix = prefix
	} else if repository.Spec.Backend.Swift != nil {
		repository.Spec.Backend.Swift.Prefix = prefix
	}

	repo, _, err := util.CreateOrPatchRepository(c.stashClient.StashV1alpha1(), repository.ObjectMeta, func(in *api.Repository) *api.Repository {
		in.Spec = repository.Spec
		return in
	})
	if err == nil {
		log.Infof("Repository %v created", repository.Name)
	}
	return repo, err
}

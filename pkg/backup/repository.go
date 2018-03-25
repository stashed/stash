package backup

import (
	"strings"

	"github.com/appscode/go/log"
	api "github.com/appscode/stash/apis/stash/v1alpha1"
	kerr "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func (c *Controller) createRepositoryCrdIfNotExist(restic *api.Restic) error {
	repository := &api.Repository{}
	switch c.opt.Workload.Kind {
	case api.KindDeployment, api.KindReplicaSet, api.KindReplicationController:
		repository.Name = strings.ToLower(c.opt.Workload.Kind) + "." + c.opt.Workload.Name
	case api.KindStatefulSet:
		repository.Name = strings.ToLower(c.opt.Workload.Kind) + "." + c.opt.PodName
	case api.KindDaemonSet:
		repository.Name = strings.ToLower(c.opt.Workload.Kind) + "." + c.opt.Workload.Name + "." + c.opt.NodeName
	}

	repository.Namespace = restic.Namespace
	_, err := c.stashClient.StashV1alpha1().Repositories(repository.Namespace).Get(repository.Name, metav1.GetOptions{})
	if err != nil && kerr.IsNotFound(err) {
		repository.Labels = map[string]string{
			"restic":        restic.Name,
			"workload-kind": c.opt.Workload.Kind,
			"workload-name": c.opt.Workload.Name,
		}

		switch c.opt.Workload.Kind {
		case api.KindStatefulSet:
			repository.Labels = map[string]string{
				"pod-name": c.opt.PodName,
			}
		case api.KindDaemonSet:
			repository.Labels = map[string]string{
				"node-name": c.opt.NodeName,
			}
		}

		repository.Spec.Backend = restic.Spec.Backend
		_, err = c.stashClient.StashV1alpha1().Repositories(repository.Namespace).Create(repository)
		if err == nil {
			log.Infof("Repository %v created", repository.Name)
		}
		return err
	}
	return err
}

package backup

import (
	"strings"

	"github.com/appscode/go/log"
	api "github.com/appscode/stash/apis/stash/v1alpha1"
	kerr "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func (c *Controller) createRepositoryCrdIfNotExist(restic *api.Restic, backupDir string) error {
	repository := &api.Repository{}
	repository.Namespace = restic.Namespace
	repository.Name = c.getRepositoryCrdName(restic)

	_, err := c.stashClient.StashV1alpha1().Repositories(repository.Namespace).Get(repository.Name, metav1.GetOptions{})
	if err != nil && kerr.IsNotFound(err) {
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

		repository.Spec.Backend = restic.Spec.Backend
		repository.Spec.BackupPath = backupDir
		_, err = c.stashClient.StashV1alpha1().Repositories(repository.Namespace).Create(repository)
		if err == nil {
			log.Infof("Repository %v created", repository.Name)
		}
		return err
	}
	return err
}

func (c *Controller) getRepositoryCrdName(restic *api.Restic) string {
	name := ""
	switch c.opt.Workload.Kind {
	case api.KindDeployment, api.KindReplicaSet, api.KindReplicationController:
		name = strings.ToLower(c.opt.Workload.Kind) + "." + c.opt.Workload.Name
	case api.KindStatefulSet:
		name = strings.ToLower(c.opt.Workload.Kind) + "." + c.opt.PodName
	case api.KindDaemonSet:
		name = strings.ToLower(c.opt.Workload.Kind) + "." + c.opt.Workload.Name + "." + c.opt.NodeName
	}
	return name
}

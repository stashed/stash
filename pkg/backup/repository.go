/*
Copyright The Stash Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package backup

import (
	"stash.appscode.dev/apimachinery/apis"
	api "stash.appscode.dev/apimachinery/apis/stash/v1alpha1"
	"stash.appscode.dev/apimachinery/client/clientset/versioned/typed/stash/v1alpha1/util"

	"github.com/appscode/go/log"
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
	case apis.KindStatefulSet:
		repository.Labels["pod-name"] = c.opt.PodName
	case apis.KindDaemonSet:
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

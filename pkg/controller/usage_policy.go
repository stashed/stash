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

package controller

import (
	"context"
	"fmt"

	api_v1alpha1 "stash.appscode.dev/apimachinery/apis/stash/v1alpha1"
	v1alpha1_util "stash.appscode.dev/apimachinery/client/clientset/versioned/typed/stash/v1alpha1/util"
	"stash.appscode.dev/apimachinery/pkg/invoker"

	"gomodules.xyz/x/arrays"
	kerr "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	kmapi "kmodules.xyz/client-go/api/v1"
)

type repoReferenceHandler interface {
	invoker.MetadataHandler
	invoker.RepositoryGetter
}

func (c *StashController) validateAgainstUsagePolicy(repo kmapi.ObjectReference, curNamespace string) error {
	if repo.Namespace == "" {
		repo.Namespace = curNamespace
	}

	repository, err := c.repoLister.Repositories(repo.Namespace).Get(repo.Name)
	if err != nil {
		if kerr.IsNotFound(err) {
			return nil
		}
		return err
	}

	namespace, err := c.kubeClient.CoreV1().Namespaces().Get(context.Background(), curNamespace, metav1.GetOptions{})
	if err != nil {
		return err
	}

	if !repository.UsageAllowed(namespace) {
		return fmt.Errorf("Namespace %q is not allowed to refer Repository %q of %q namespace. Please, check the `usagePolicy` of the Repository.", curNamespace, repo.Name, repo.Namespace)
	}

	return nil
}

func (c *StashController) upsertRepositoryReferences(inv repoReferenceHandler) error {
	repository, err := c.repoLister.Repositories(inv.GetRepoRef().Namespace).Get(inv.GetRepoRef().Name)
	if err != nil {
		if kerr.IsNotFound(err) {
			return nil
		}
		return err
	}

	reference := kmapi.TypedObjectReference{
		Kind:      inv.GetTypeMeta().Kind,
		Namespace: inv.GetObjectMeta().Namespace,
		Name:      inv.GetObjectMeta().Name,
	}

	// Don't insert anything if the reference already exists
	found, _ := arrays.Contains(repository.Status.References, reference)
	if found {
		return nil
	}

	repository.Status.References = append(repository.Status.References, reference)

	_, err = v1alpha1_util.UpdateRepositoryStatus(
		context.TODO(),
		c.stashClient.StashV1alpha1(),
		repository.ObjectMeta,
		func(in *api_v1alpha1.RepositoryStatus) (types.UID, *api_v1alpha1.RepositoryStatus) {
			in.References = repository.Status.References
			return repository.UID, in
		},
		metav1.UpdateOptions{},
	)

	return err
}

func (c *StashController) deleteRepositoryReferences(inv repoReferenceHandler) error {
	repository, err := c.repoLister.Repositories(inv.GetRepoRef().Namespace).Get(inv.GetRepoRef().Name)
	if err != nil {
		if kerr.IsNotFound(err) {
			return nil
		}
		return err
	}

	reference := kmapi.TypedObjectReference{
		Kind:      inv.GetTypeMeta().Kind,
		Namespace: inv.GetObjectMeta().Namespace,
		Name:      inv.GetObjectMeta().Name,
	}

	// Delete the reference if it exists
	found, indx := arrays.Contains(repository.Status.References, reference)
	if !found {
		return nil
	}

	repository.Status.References[indx] = repository.Status.References[len(repository.Status.References)-1]
	repository.Status.References = repository.Status.References[:len(repository.Status.References)-1]

	_, err = v1alpha1_util.UpdateRepositoryStatus(
		context.TODO(),
		c.stashClient.StashV1alpha1(),
		repository.ObjectMeta,
		func(in *api_v1alpha1.RepositoryStatus) (types.UID, *api_v1alpha1.RepositoryStatus) {
			in.References = repository.Status.References
			return repository.UID, in
		},
		metav1.UpdateOptions{},
	)

	return err
}

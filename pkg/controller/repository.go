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

	"stash.appscode.dev/apimachinery/apis"
	"stash.appscode.dev/apimachinery/apis/stash"
	api "stash.appscode.dev/apimachinery/apis/stash/v1alpha1"
	stash_util "stash.appscode.dev/apimachinery/client/clientset/versioned/typed/stash/v1alpha1/util"
	"stash.appscode.dev/stash/pkg/util"

	"github.com/golang/glog"
	"gomodules.xyz/stow"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	core_util "kmodules.xyz/client-go/core/v1"
	"kmodules.xyz/client-go/tools/queue"
	"kmodules.xyz/objectstore-api/osm"
	"kmodules.xyz/webhook-runtime/admission"
	hooks "kmodules.xyz/webhook-runtime/admission/v1beta1"
	webhook "kmodules.xyz/webhook-runtime/admission/v1beta1/generic"
)

func (c *StashController) NewRepositoryWebhook() hooks.AdmissionHook {
	return webhook.NewGenericWebhook(
		schema.GroupVersionResource{
			Group:    "admission.stash.appscode.com",
			Version:  "v1alpha1",
			Resource: "repositoryvalidators",
		},
		"repositoryvalidator",
		[]string{stash.GroupName},
		api.SchemeGroupVersion.WithKind("Repository"),
		nil,
		&admission.ResourceHandlerFuncs{
			CreateFunc: func(obj runtime.Object) (runtime.Object, error) {
				return nil, obj.(*api.Repository).IsValid()
			},
			UpdateFunc: func(oldObj, newObj runtime.Object) (runtime.Object, error) {
				return nil, newObj.(*api.Repository).IsValid()
			},
		},
	)
}
func (c *StashController) initRepositoryWatcher() {
	c.repoInformer = c.stashInformerFactory.Stash().V1alpha1().Repositories().Informer()
	c.repoQueue = queue.New("Repository", c.MaxNumRequeues, c.NumThreads, c.runRepositoryReconciler)
	c.repoInformer.AddEventHandler(queue.NewReconcilableHandler(c.repoQueue.GetQueue()))
	c.repoLister = c.stashInformerFactory.Stash().V1alpha1().Repositories().Lister()
}

func (c *StashController) runRepositoryReconciler(key string) error {
	obj, exist, err := c.repoInformer.GetIndexer().GetByKey(key)
	if err != nil {
		glog.Errorf("Fetching object with key %s from store failed with %v", key, err)
		return err
	}

	if !exist {
		glog.Warningf("Repository %s does not exist anymore\n", key)
	} else {
		glog.Infof("Sync/Add/Update for Repository %s", key)

		repo := obj.(*api.Repository)

		if repo.DeletionTimestamp != nil {
			if core_util.HasFinalizer(repo.ObjectMeta, apis.RepositoryFinalizer) {
				// ignore invalid repository objects (eg: created by xray).
				if repo.IsValid() == nil && repo.Spec.WipeOut {
					err = c.deleteResticRepository(repo)
					if err != nil {
						return err
					}
				}
				_, _, err = stash_util.PatchRepository(
					context.TODO(),
					c.stashClient.StashV1alpha1(),
					repo,
					func(in *api.Repository) *api.Repository {
						in.ObjectMeta = core_util.RemoveFinalizer(in.ObjectMeta, apis.RepositoryFinalizer)
						return in
					},
					metav1.PatchOptions{},
				)
				return err
			}
		} else {
			_, _, err = stash_util.PatchRepository(
				context.TODO(),
				c.stashClient.StashV1alpha1(),
				repo,
				func(in *api.Repository) *api.Repository {
					in.ObjectMeta = core_util.AddFinalizer(in.ObjectMeta, apis.RepositoryFinalizer)
					return in
				},
				metav1.PatchOptions{},
			)
			return err
		}
	}
	return nil
}

func (c *StashController) deleteResticRepository(repository *api.Repository) error {
	cfg, err := osm.NewOSMContext(c.kubeClient, repository.Spec.Backend, repository.Namespace)
	if err != nil {
		return err
	}

	loc, err := stow.Dial(cfg.Provider, cfg.Config)
	if err != nil {
		return err
	}

	bucket, prefix, err := util.GetBucketAndPrefix(&repository.Spec.Backend)
	if err != nil {
		return err
	}
	prefix = prefix + "/"

	container, err := loc.Container(bucket)
	if err != nil {
		return err
	}

	cursor := stow.CursorStart
	for {
		items, next, err := container.Items(prefix, cursor, 50)
		if err != nil {
			return err
		}
		for _, item := range items {
			if err := container.RemoveItem(item.ID()); err != nil {
				return err
			}
		}
		cursor = next
		if stow.IsCursorEnd(cursor) {
			break
		}
	}

	return nil
}

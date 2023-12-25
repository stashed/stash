/*
Copyright AppsCode Inc. and Contributors

Licensed under the AppsCode Community License 1.0.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    https://github.com/appscode/licenses/raw/1.0.0/AppsCode-Community-1.0.0.md

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

	"stash.appscode.dev/apimachinery/apis"
	"stash.appscode.dev/apimachinery/apis/stash"
	api_v1alpha1 "stash.appscode.dev/apimachinery/apis/stash/v1alpha1"
	api_v1beta1 "stash.appscode.dev/apimachinery/apis/stash/v1beta1"
	stash_util "stash.appscode.dev/apimachinery/client/clientset/versioned/typed/stash/v1alpha1/util"
	"stash.appscode.dev/stash/pkg/util"

	"gomodules.xyz/stow"
	core "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/cache"
	"k8s.io/klog/v2"
	core_util "kmodules.xyz/client-go/core/v1"
	"kmodules.xyz/client-go/tools/queue"
	"kmodules.xyz/objectstore-api/osm"
	"kmodules.xyz/webhook-runtime/admission"
	hooks "kmodules.xyz/webhook-runtime/admission/v1beta1"
	webhook "kmodules.xyz/webhook-runtime/admission/v1beta1/generic"
)

type repositoryReconciler struct {
	ctrl       *StashController
	logger     klog.Logger
	repository *api_v1alpha1.Repository
}

func (c *StashController) NewRepositoryWebhook() hooks.AdmissionHook {
	return webhook.NewGenericWebhook(
		schema.GroupVersionResource{
			Group:    "admission.stash.appscode.com",
			Version:  "v1alpha1",
			Resource: "repositoryvalidators",
		},
		"repositoryvalidator",
		[]string{stash.GroupName},
		api_v1alpha1.SchemeGroupVersion.WithKind("Repository"),
		nil,
		&admission.ResourceHandlerFuncs{
			CreateFunc: func(obj runtime.Object) (runtime.Object, error) {
				return nil, obj.(*api_v1alpha1.Repository).IsValid()
			},
			UpdateFunc: func(oldObj, newObj runtime.Object) (runtime.Object, error) {
				return nil, newObj.(*api_v1alpha1.Repository).IsValid()
			},
		},
	)
}

func (c *StashController) initRepositoryWatcher() {
	c.repoInformer = c.stashInformerFactory.Stash().V1alpha1().Repositories().Informer()
	c.repoQueue = queue.New(api_v1alpha1.ResourceKindRepository, c.MaxNumRequeues, c.NumThreads, c.runRepositoryReconciler)
	if c.auditor != nil {
		c.auditor.ForGVK(c.repoInformer, api_v1alpha1.SchemeGroupVersion.WithKind(api_v1alpha1.ResourceKindRepository))
	}
	_, _ = c.repoInformer.AddEventHandler(queue.NewReconcilableHandler(c.repoQueue.GetQueue(), core.NamespaceAll))
	c.repoLister = c.stashInformerFactory.Stash().V1alpha1().Repositories().Lister()
}

func (c *StashController) runRepositoryReconciler(key string) error {
	obj, exist, err := c.repoInformer.GetIndexer().GetByKey(key)
	if err != nil {
		klog.ErrorS(err, "Failed to fetch object from indexer",
			apis.ObjectKind, api_v1alpha1.ResourceKindRepository,
			apis.ObjectKey, key,
		)
		return err
	}

	if !exist {
		klog.V(4).InfoS("Object doesn't exist anymore",
			apis.ObjectKind, api_v1alpha1.ResourceKindRepository,
			apis.ObjectKey, key,
		)
	} else {
		repo := obj.(*api_v1alpha1.Repository)

		logger := klog.NewKlogr().WithValues(
			apis.ObjectKind, api_v1alpha1.ResourceKindRepository,
			apis.ObjectName, repo.Name,
			apis.ObjectNamespace, repo.Namespace,
		)
		logger.V(4).Info("Received Sync/Add/Update event")

		r := repositoryReconciler{
			ctrl:       c,
			logger:     logger,
			repository: repo,
		}

		if r.repository.DeletionTimestamp != nil {
			if err := r.cleanupOffshoots(); err != nil {
				return err
			}
		} else {
			if err := r.ensureFinalizer(); err != nil {
				return err
			}
		}
		if err := r.requeueReferences(); err != nil {
			return err
		}
		return r.updateObservedGeneration()
	}
	return nil
}

func (r *repositoryReconciler) ensureFinalizer() error {
	if !core_util.HasFinalizer(r.repository.ObjectMeta, apis.RepositoryFinalizer) {
		var err error
		r.repository, _, err = stash_util.PatchRepository(
			context.TODO(),
			r.ctrl.stashClient.StashV1alpha1(),
			r.repository,
			func(in *api_v1alpha1.Repository) *api_v1alpha1.Repository {
				in.ObjectMeta = core_util.AddFinalizer(in.ObjectMeta, apis.RepositoryFinalizer)
				return in
			},
			metav1.PatchOptions{},
		)
		if err != nil {
			return err
		}
	}
	return nil
}

func (r *repositoryReconciler) cleanupOffshoots() error {
	if core_util.HasFinalizer(r.repository.ObjectMeta, apis.RepositoryFinalizer) {
		// ignore invalid repository objects (eg: created by xray).
		if r.repository.IsValid() == nil && r.repository.Spec.WipeOut {
			err := r.deleteResticRepository()
			if err != nil {
				return err
			}
		}

		var err error
		r.repository, _, err = stash_util.PatchRepository(
			context.TODO(),
			r.ctrl.stashClient.StashV1alpha1(),
			r.repository,
			func(in *api_v1alpha1.Repository) *api_v1alpha1.Repository {
				in.ObjectMeta = core_util.RemoveFinalizer(in.ObjectMeta, apis.RepositoryFinalizer)
				return in
			},
			metav1.PatchOptions{},
		)
		if err != nil {
			return err
		}
	}
	return nil
}

func (r *repositoryReconciler) deleteResticRepository() error {
	cfg, err := osm.NewOSMContext(r.ctrl.kubeClient, r.repository.Spec.Backend, r.repository.Namespace)
	if err != nil {
		return err
	}

	loc, err := stow.Dial(cfg.Provider, cfg.Config)
	if err != nil {
		return err
	}

	bucket, prefix, err := util.GetBucketAndPrefix(&r.repository.Spec.Backend)
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

func (r *repositoryReconciler) requeueReferences() error {
	for _, ref := range r.repository.Status.References {
		switch ref.Kind {
		case api_v1beta1.ResourceKindRestoreSession:
			restoresession, err := r.ctrl.restoreSessionLister.RestoreSessions(ref.Namespace).Get(ref.Name)
			if err != nil {
				return err
			}
			key, err := cache.MetaNamespaceKeyFunc(restoresession)
			if err != nil {
				return err
			}
			r.ctrl.restoreSessionQueue.GetQueue().Add(key)

		case api_v1beta1.ResourceKindBackupConfiguration:
			backupconfiguration, err := r.ctrl.bcLister.BackupConfigurations(ref.Namespace).Get(ref.Name)
			if err != nil {
				return err
			}
			key, err := cache.MetaNamespaceKeyFunc(backupconfiguration)
			if err != nil {
				return err
			}
			r.ctrl.bcQueue.GetQueue().Add(key)

		default:
			return fmt.Errorf("reference kind %q is unknown", ref.Kind)
		}
	}

	return nil
}

func (r *repositoryReconciler) updateObservedGeneration() error {
	if r.repository.DeletionTimestamp == nil && r.repository.Status.ObservedGeneration != r.repository.Generation {
		var err error
		r.repository, err = stash_util.UpdateRepositoryStatus(
			context.TODO(),
			r.ctrl.stashClient.StashV1alpha1(),
			r.repository.ObjectMeta,
			func(in *api_v1alpha1.RepositoryStatus) (types.UID, *api_v1alpha1.RepositoryStatus) {
				in.ObservedGeneration = r.repository.ObjectMeta.Generation
				return r.repository.UID, in
			},
			metav1.UpdateOptions{},
		)
		return err
	}
	return nil
}

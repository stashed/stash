package controller

import (
	"github.com/appscode/kubernetes-webhook-util/admission"
	hooks "github.com/appscode/kubernetes-webhook-util/admission/v1beta1"
	webhook "github.com/appscode/kubernetes-webhook-util/admission/v1beta1/generic"
	core_util "github.com/appscode/kutil/core/v1"
	"github.com/appscode/kutil/tools/queue"
	"github.com/appscode/stash/apis/stash"
	api "github.com/appscode/stash/apis/stash/v1alpha1"
	stash_util "github.com/appscode/stash/client/clientset/versioned/typed/stash/v1alpha1/util"
	"github.com/appscode/stash/pkg/util"
	"github.com/golang/glog"
	"github.com/graymeta/stow"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"kmodules.xyz/objectstore-api/osm"
)

func (c *StashController) NewRepositoryWebhook() hooks.AdmissionHook {
	return webhook.NewGenericWebhook(
		schema.GroupVersionResource{
			Group:    "admission.stash.appscode.com",
			Version:  "v1alpha1",
			Resource: "repositories",
		},
		"repository",
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
	c.repoInformer.AddEventHandler(queue.DefaultEventHandler(c.repoQueue.GetQueue()))
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
			if core_util.HasFinalizer(repo.ObjectMeta, util.RepositoryFinalizer) {
				// ignore invalid repository objects (eg: created by xray).
				if repo.IsValid() == nil && repo.Spec.WipeOut {
					err = c.deleteResticRepository(repo)
					if err != nil {
						return err
					}
				}
				_, _, err = stash_util.PatchRepository(c.stashClient.StashV1alpha1(), repo, func(in *api.Repository) *api.Repository {
					in.ObjectMeta = core_util.RemoveFinalizer(in.ObjectMeta, util.RepositoryFinalizer)
					return in
				})
				return err
			}
		} else {
			_, _, err = stash_util.PatchRepository(c.stashClient.StashV1alpha1(), repo, func(in *api.Repository) *api.Repository {
				in.ObjectMeta = core_util.AddFinalizer(in.ObjectMeta, util.RepositoryFinalizer)
				return in
			})
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

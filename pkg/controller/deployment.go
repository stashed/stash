package controller

import (
	"fmt"

	"github.com/appscode/go/log"
	stringz "github.com/appscode/go/strings"
	"github.com/appscode/kutil"
	apps_util "github.com/appscode/kutil/apps/v1beta1"
	core_util "github.com/appscode/kutil/core/v1"
	api "github.com/appscode/stash/apis/stash/v1alpha1"
	"github.com/appscode/stash/pkg/util"
	"github.com/golang/glog"
	apps "k8s.io/api/apps/v1beta1"
	core "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	rt "k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/watch"
	apps_listers "k8s.io/client-go/listers/apps/v1beta1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"
)

func (c *StashController) initDeploymentWatcher() {
	lw := &cache.ListWatch{
		ListFunc: func(options metav1.ListOptions) (rt.Object, error) {
			options.IncludeUninitialized = true
			return c.k8sClient.AppsV1beta1().Deployments(core.NamespaceAll).List(options)
		},
		WatchFunc: func(options metav1.ListOptions) (watch.Interface, error) {
			options.IncludeUninitialized = true
			return c.k8sClient.AppsV1beta1().Deployments(core.NamespaceAll).Watch(options)
		},
	}

	// create the workqueue
	c.dpQueue = workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), "deployment")

	// Bind the workqueue to a cache with the help of an informer. This way we make sure that
	// whenever the cache is updated, the pod key is added to the workqueue.
	// Note that when we finally process the item from the workqueue, we might see a newer version
	// of the Deployment than the version which was responsible for triggering the update.
	c.dpIndexer, c.dpInformer = cache.NewIndexerInformer(lw, &apps.Deployment{}, c.options.ResyncPeriod, cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			key, err := cache.MetaNamespaceKeyFunc(obj)
			if err == nil {
				c.dpQueue.Add(key)
			}
		},
		UpdateFunc: func(old interface{}, new interface{}) {
			key, err := cache.MetaNamespaceKeyFunc(new)
			if err == nil {
				c.dpQueue.Add(key)
			}
		},
		DeleteFunc: func(obj interface{}) {
			// IndexerInformer uses a delta queue, therefore for deletes we have to use this
			// key function.
			key, err := cache.DeletionHandlingMetaNamespaceKeyFunc(obj)
			if err == nil {
				c.dpQueue.Add(key)
			}
		},
	}, cache.Indexers{cache.NamespaceIndex: cache.MetaNamespaceIndexFunc})
	c.dpLister = apps_listers.NewDeploymentLister(c.dpIndexer)
}

func (c *StashController) runDeploymentWatcher() {
	for c.processNextDeployment() {
	}
}

func (c *StashController) processNextDeployment() bool {
	// Wait until there is a new item in the working queue
	key, quit := c.dpQueue.Get()
	if quit {
		return false
	}
	// Tell the queue that we are done with processing this key. This unblocks the key for other workers
	// This allows safe parallel processing because two deployments with the same key are never processed in
	// parallel.
	defer c.dpQueue.Done(key)

	// Invoke the method containing the business logic
	err := c.runDeploymentInjector(key.(string))
	if err == nil {
		// Forget about the #AddRateLimited history of the key on every successful synchronization.
		// This ensures that future processing of updates for this key is not delayed because of
		// an outdated error history.
		c.dpQueue.Forget(key)
		return true
	}
	log.Errorf("Failed to process Deployment %v. Reason: %s", key, err)

	// This controller retries 5 times if something goes wrong. After that, it stops trying.
	if c.dpQueue.NumRequeues(key) < c.options.MaxNumRequeues {
		glog.Infof("Error syncing deployment %v: %v", key, err)

		// Re-enqueue the key rate limited. Based on the rate limiter on the
		// queue and the re-enqueue history, the key will be processed later again.
		c.dpQueue.AddRateLimited(key)
		return true
	}

	c.dpQueue.Forget(key)
	// Report to an external entity that, even after several retries, we could not successfully process this key
	runtime.HandleError(err)
	glog.Infof("Dropping deployment %q out of the queue: %v", key, err)
	return true
}

// syncToStdout is the business logic of the controller. In this controller it simply prints
// information about the deployment to stdout. In case an error happened, it has to simply return the error.
// The retry logic should not be part of the business logic.
func (c *StashController) runDeploymentInjector(key string) error {
	obj, exists, err := c.dpIndexer.GetByKey(key)
	if err != nil {
		glog.Errorf("Fetching object with key %s from store failed with %v", key, err)
		return err
	}

	if !exists {
		// Below we will warm up our cache with a Deployment, so that we will see a delete for one d
		fmt.Printf("Deployment %s does not exist anymore\n", key)

		ns, name, err := cache.SplitMetaNamespaceKey(key)
		if err != nil {
			return err
		}
		util.DeleteConfigmapLock(c.k8sClient, ns, api.LocalTypedReference{Kind: api.AppKindDeployment, Name: name})
	} else {
		dp := obj.(*apps.Deployment)
		fmt.Printf("Sync/Add/Update for Deployment %s\n", dp.GetName())

		if !util.CheckWorkloadInitializer(dp.Initializers) {
			fmt.Printf("Not stash's turn to initialize %s\n", dp.GetName())
			return nil
		}

		oldRestic, err := util.GetAppliedRestic(dp.Annotations)
		if err != nil {
			return err
		}
		newRestic, err := util.FindRestic(c.rstLister, dp.ObjectMeta)
		if err != nil {
			log.Errorf("Error while searching Restic for Deployment %s/%s.", dp.Name, dp.Namespace)
			return err
		}
		if util.ResticEqual(oldRestic, newRestic) {
			return nil
		}
		if newRestic != nil {
			return c.EnsureDeploymentSidecar(dp, oldRestic, newRestic)
		} else if oldRestic != nil {
			return c.EnsureDeploymentSidecarDeleted(dp, oldRestic)
		}

		// not restic workload, just remove the pending stash initializer
		if util.ShouldRemovePendingInitializer(dp.Initializers) {
			_, err = apps_util.PatchDeployment(c.k8sClient, dp, func(obj *apps.Deployment) *apps.Deployment {
				fmt.Println("Removing pending stash initializer for", obj.Name)
				if len(obj.Initializers.Pending) == 1 {
					obj.Initializers = nil
				} else {
					obj.Initializers.Pending = obj.Initializers.Pending[1:]
				}
				return obj
			})
			if err != nil {
				log.Errorf("Error while removing pending stash initializer for %s/%s. Reason: %s", dp.Name, dp.Namespace, err)
				return err
			}
		}
	}
	return nil
}

func (c *StashController) EnsureDeploymentSidecar(resource *apps.Deployment, old, new *api.Restic) (err error) {
	if new.Spec.Backend.StorageSecretName == "" {
		err = fmt.Errorf("missing repository secret name for Restic %s/%s", new.Namespace, new.Name)
		return
	}
	_, err = c.k8sClient.CoreV1().Secrets(resource.Namespace).Get(new.Spec.Backend.StorageSecretName, metav1.GetOptions{})
	if err != nil {
		return err
	}

	if c.options.EnableRBAC {
		sa := stringz.Val(resource.Spec.Template.Spec.ServiceAccountName, "default")
		err := c.ensureRoleBinding(kutil.GetObjectReference(resource, apps.SchemeGroupVersion), sa)
		if err != nil {
			return err
		}
	}

	resource, err = apps_util.PatchDeployment(c.k8sClient, resource, func(obj *apps.Deployment) *apps.Deployment {
		if util.ShouldRemovePendingInitializer(obj.Initializers) {
			fmt.Println("Removing pending stash initializer for", obj.Name)
			if len(obj.Initializers.Pending) == 1 {
				obj.Initializers = nil
			} else {
				obj.Initializers.Pending = obj.Initializers.Pending[1:]
			}
		}

		workload := api.LocalTypedReference{
			Kind: api.AppKindDeployment,
			Name: obj.Name,
		}
		obj.Spec.Template.Spec.Containers = core_util.UpsertContainer(obj.Spec.Template.Spec.Containers, util.CreateSidecarContainer(new, c.options.SidecarImageTag, workload))
		obj.Spec.Template.Spec.Volumes = util.UpsertScratchVolume(obj.Spec.Template.Spec.Volumes)
		obj.Spec.Template.Spec.Volumes = util.UpsertDownwardVolume(obj.Spec.Template.Spec.Volumes)
		obj.Spec.Template.Spec.Volumes = util.MergeLocalVolume(obj.Spec.Template.Spec.Volumes, old, new)

		if obj.Annotations == nil {
			obj.Annotations = make(map[string]string)
		}
		r := &api.Restic{
			TypeMeta: metav1.TypeMeta{
				APIVersion: api.SchemeGroupVersion.String(),
				Kind:       api.ResourceKindRestic,
			},
			ObjectMeta: new.ObjectMeta,
			Spec:       new.Spec,
		}
		data, _ := kutil.MarshalToJson(r, api.SchemeGroupVersion)
		obj.Annotations[api.LastAppliedConfiguration] = string(data)
		obj.Annotations[api.VersionTag] = c.options.SidecarImageTag

		return obj
	})
	if err != nil {
		return
	}

	err = apps_util.WaitUntilDeploymentReady(c.k8sClient, resource.ObjectMeta)
	if err != nil {
		return
	}
	err = util.WaitUntilSidecarAdded(c.k8sClient, resource.Namespace, resource.Spec.Selector)
	return err
}

func (c *StashController) EnsureDeploymentSidecarDeleted(resource *apps.Deployment, restic *api.Restic) (err error) {
	if c.options.EnableRBAC {
		err := c.ensureRoleBindingDeleted(resource.ObjectMeta)
		if err != nil {
			return err
		}
	}

	resource, err = apps_util.PatchDeployment(c.k8sClient, resource, func(obj *apps.Deployment) *apps.Deployment {
		obj.Spec.Template.Spec.Containers = core_util.EnsureContainerDeleted(obj.Spec.Template.Spec.Containers, util.StashContainer)
		obj.Spec.Template.Spec.Volumes = util.EnsureVolumeDeleted(obj.Spec.Template.Spec.Volumes, util.ScratchDirVolumeName)
		obj.Spec.Template.Spec.Volumes = util.EnsureVolumeDeleted(obj.Spec.Template.Spec.Volumes, util.PodinfoVolumeName)
		if restic.Spec.Backend.Local != nil {
			obj.Spec.Template.Spec.Volumes = util.EnsureVolumeDeleted(obj.Spec.Template.Spec.Volumes, util.LocalVolumeName)
		}
		if obj.Annotations != nil {
			delete(obj.Annotations, api.LastAppliedConfiguration)
			delete(obj.Annotations, api.VersionTag)
		}
		return obj
	})
	if err != nil {
		return
	}

	err = apps_util.WaitUntilDeploymentReady(c.k8sClient, resource.ObjectMeta)
	if err != nil {
		return
	}
	err = util.WaitUntilSidecarRemoved(c.k8sClient, resource.Namespace, resource.Spec.Selector)
	if err != nil {
		return
	}
	util.DeleteConfigmapLock(c.k8sClient, resource.Namespace, api.LocalTypedReference{Kind: api.AppKindDeployment, Name: resource.Name})
	return err
}

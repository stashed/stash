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

	"stash.appscode.dev/apimachinery/apis"
	stash_rbac "stash.appscode.dev/stash/pkg/rbac"
	"stash.appscode.dev/stash/pkg/util"

	appsv1 "k8s.io/api/apps/v1"
	core "k8s.io/api/core/v1"
	kerr "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/tools/cache"
	"k8s.io/klog/v2"
	apps_util "kmodules.xyz/client-go/apps/v1"
	"kmodules.xyz/client-go/tools/queue"
	"kmodules.xyz/webhook-runtime/admission"
	hooks "kmodules.xyz/webhook-runtime/admission/v1beta1"
	webhook "kmodules.xyz/webhook-runtime/admission/v1beta1/workload"
	wapi "kmodules.xyz/webhook-runtime/apis/workload/v1"
	wcs "kmodules.xyz/webhook-runtime/client/workload/v1"
)

func (c *StashController) NewStatefulSetWebhook() hooks.AdmissionHook {
	return webhook.NewWorkloadWebhook(
		schema.GroupVersionResource{
			Group:    "admission.stash.appscode.com",
			Version:  "v1alpha1",
			Resource: "statefulsetmutators",
		},
		"statefulsetmutator",
		"StatefulSetMutator",
		nil,
		&admission.ResourceHandlerFuncs{
			CreateFunc: func(obj runtime.Object) (runtime.Object, error) {
				w := obj.(*wapi.Workload)
				// apply stash backup/restore logic on this workload
				modified, err := c.applyStashLogic(w, apis.CallerWebhook)
				if err != nil {
					return w, err
				}
				if modified {
					err := setRollingUpdate(w)
					if err != nil {
						return w, err
					}
				}
				return w, err

			},
			UpdateFunc: func(oldObj, newObj runtime.Object) (runtime.Object, error) {
				w := newObj.(*wapi.Workload)
				// apply stash backup/restore logic on this workload
				modified, err := c.applyStashLogic(w, apis.CallerWebhook)
				if err != nil {
					return w, err
				}
				if modified {
					err := setRollingUpdate(w)
					if err != nil {
						return w, err
					}
				}
				return w, err
			},
		},
	)
}

func (c *StashController) initStatefulSetWatcher() {
	c.ssInformer = c.kubeInformerFactory.Apps().V1().StatefulSets().Informer()
	c.ssQueue = queue.New("StatefulSet", c.MaxNumRequeues, c.NumThreads, c.runStatefulSetInjector)
	c.ssInformer.AddEventHandler(queue.DefaultEventHandler(c.ssQueue.GetQueue(), core.NamespaceAll))
	c.ssLister = c.kubeInformerFactory.Apps().V1().StatefulSets().Lister()
}

// syncToStdout is the business logic of the controller. In this controller it simply prints
// information about the deployment to stdout. In case an error happened, it has to simply return the error.
// The retry logic should not be part of the business logic.
func (c *StashController) runStatefulSetInjector(key string) error {
	obj, exists, err := c.ssInformer.GetIndexer().GetByKey(key)
	if err != nil {
		klog.Errorf("Fetching object with key %s from store failed with %v", key, err)
		return err
	}

	if !exists {
		// Below we will warm up our cache with a StatefulSet, so that we will see a delete for one d
		klog.Warningf("StatefulSet %s does not exist anymore\n", key)

		ns, name, err := cache.SplitMetaNamespaceKey(key)
		if err != nil {
			return err
		}
		// workload does not exist anymore. so delete respective ConfigMapLocks if exist
		err = util.DeleteAllConfigMapLocks(c.kubeClient, ns, name, apis.KindStatefulSet)
		if err != nil && !kerr.IsNotFound(err) {
			return err
		}

	} else {
		klog.Infof("Sync/Add/Update for StatefulSet %s", key)

		ss := obj.(*appsv1.StatefulSet).DeepCopy()
		ss.GetObjectKind().SetGroupVersionKind(appsv1.SchemeGroupVersion.WithKind(apis.KindStatefulSet))

		// convert StatefulSet into a common object (Workload type) so that
		// we don't need to re-write stash logic for StatefulSet separately
		w, err := wcs.ConvertToWorkload(ss.DeepCopy())
		if err != nil {
			klog.Errorf("failed to convert StatefulSet %s/%s to workload type. Reason: %v", ss.Namespace, ss.Name, err)
			return err
		}

		// apply stash backup/restore logic on this workload
		modified, err := c.applyStashLogic(w, apis.CallerController)
		if err != nil {
			klog.Errorf("failed to apply stash logic on StatefulSet %s/%s. Reason: %v", ss.Namespace, ss.Name, err)
			return err
		}
		if modified {
			// set update strategy RollingUpdate so that pods automatically restart after patch
			err := setRollingUpdate(w)
			if err != nil {
				return err
			}

			// workload has been modified. patch the workload so that respective pods start with the updated spec
			_, _, err = apps_util.PatchStatefulSetObject(context.TODO(), c.kubeClient, ss, w.Object.(*appsv1.StatefulSet), metav1.PatchOptions{})
			if err != nil {
				klog.Errorf("failed to update statefulset %s/%s. Reason: %v", ss.Namespace, ss.Name, err)
				return err
			}

			// wait until newly patched StatefulSet pods are ready
			err = util.WaitUntilStatefulSetReady(c.kubeClient, ss.ObjectMeta)
			if err != nil {
				return err
			}
		}

		// if the workload does not have any stash sidecar/init-container then
		// delete respective ConfigMapLock and RBAC stuffs if exist
		err = c.ensureUnnecessaryConfigMapLockDeleted(w)
		if err != nil {
			return err
		}
		err = stash_rbac.EnsureUnnecessaryWorkloadRBACDeleted(c.kubeClient, w)
		if err != nil {
			return err
		}
	}
	return nil
}

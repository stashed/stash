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
	"stash.appscode.dev/apimachinery/apis"
	stash_rbac "stash.appscode.dev/stash/pkg/rbac"
	"stash.appscode.dev/stash/pkg/util"

	core "k8s.io/api/core/v1"
	kerr "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/tools/cache"
	"k8s.io/klog/v2"
	"kmodules.xyz/client-go/discovery"
	"kmodules.xyz/client-go/tools/queue"
	ocapps "kmodules.xyz/openshift/apis/apps/v1"
	"kmodules.xyz/webhook-runtime/admission"
	hooks "kmodules.xyz/webhook-runtime/admission/v1beta1"
	webhook "kmodules.xyz/webhook-runtime/admission/v1beta1/workload"
	wapi "kmodules.xyz/webhook-runtime/apis/workload/v1"
	wcs "kmodules.xyz/webhook-runtime/client/workload/v1"
)

func (c *StashController) NewDeploymentConfigWebhook() hooks.AdmissionHook {
	return webhook.NewWorkloadWebhook(
		schema.GroupVersionResource{
			Group:    "admission.stash.appscode.com",
			Version:  "v1alpha1",
			Resource: "deploymentconfigmutators",
		},
		"deploymentconfigmutator",
		"DeploymentConfigMutator",
		nil,
		&admission.ResourceHandlerFuncs{
			CreateFunc: func(obj runtime.Object) (runtime.Object, error) {
				w := obj.(*wapi.Workload)
				r := workloadReconciler{
					ctrl:     c,
					workload: w,
					logger: klog.NewKlogr().WithValues(
						apis.ObjectKind, apis.KindDeploymentConfig,
						apis.ObjectName, w.Name,
						apis.ObjectNamespace, w.Namespace,
					),
				}
				err := r.reconcile(apis.CallerWebhook)
				return w, err
			},
			UpdateFunc: func(oldObj, newObj runtime.Object) (runtime.Object, error) {
				w := newObj.(*wapi.Workload)
				r := workloadReconciler{
					ctrl:     c,
					workload: w,
					logger: klog.NewKlogr().WithValues(
						apis.ObjectKind, apis.KindDeploymentConfig,
						apis.ObjectName, w.Name,
						apis.ObjectNamespace, w.Namespace,
					),
				}
				err := r.reconcile(apis.CallerWebhook)
				return w, err
			},
		},
	)
}

func (c *StashController) initDeploymentConfigWatcher() {
	if !discovery.IsPreferredAPIResource(c.kubeClient.Discovery(), ocapps.GroupVersion.String(), apis.KindDeploymentConfig) {
		klog.Warningf("Skipping watching non-preferred GroupVersion:%s Kind:%s", ocapps.GroupVersion.String(), apis.KindDeploymentConfig)
		return
	}
	c.dcInformer = c.ocInformerFactory.Apps().V1().DeploymentConfigs().Informer()
	c.dcQueue = queue.New(apis.KindDeploymentConfig, c.MaxNumRequeues, c.NumThreads, c.processDeploymentConfigEvent)
	_, _ = c.dcInformer.AddEventHandler(queue.DefaultEventHandler(c.dcQueue.GetQueue(), core.NamespaceAll))
	c.dcLister = c.ocInformerFactory.Apps().V1().DeploymentConfigs().Lister()
}

// syncToStdout is the business logic of the controller. In this controller it simply prints
// information about the deployment to stdout. In case an error happened, it has to simply return the error.
// The retry logic should not be part of the business logic.
func (c *StashController) processDeploymentConfigEvent(key string) error {
	obj, exists, err := c.dcInformer.GetIndexer().GetByKey(key)
	if err != nil {
		klog.ErrorS(err, "Failed to fetch object from indexer",
			apis.ObjectKind, apis.KindDeploymentConfig,
			apis.ObjectKey, key,
		)
		return err
	}

	if !exists {
		klog.V(4).InfoS("Object doesn't exist anymore",
			apis.ObjectKind, apis.KindDeploymentConfig,
			apis.ObjectKey, key,
		)

		ns, name, err := cache.SplitMetaNamespaceKey(key)
		if err != nil {
			return err
		}
		// workload does not exist anymore. so delete respective ConfigMapLocks if exist
		err = util.DeleteAllConfigMapLocks(c.kubeClient, ns, name, apis.KindDeploymentConfig)
		if err != nil && !kerr.IsNotFound(err) {
			return err
		}
	} else {
		dc := obj.(*ocapps.DeploymentConfig).DeepCopy()
		dc.GetObjectKind().SetGroupVersionKind(ocapps.GroupVersion.WithKind(apis.KindDeploymentConfig))

		logger := klog.NewKlogr().WithValues(
			apis.ObjectKind, apis.KindDeploymentConfig,
			apis.ObjectName, dc.Name,
			apis.ObjectNamespace, dc.Namespace,
		)
		logger.V(4).Info("Received Sync/Add/Update event")

		// convert DeploymentConfig into a common object (Workload type) so that
		// we don't need to re-write stash logic for DeploymentConfig separately
		w, err := wcs.ConvertToWorkload(dc.DeepCopy())
		if err != nil {
			logger.Error(err, "Failed to convert into generic workload type")
			return err
		}

		r := workloadReconciler{
			ctrl:     c,
			logger:   logger,
			workload: w,
		}
		if err := r.reconcile(apis.CallerController); err != nil {
			r.logger.Error(err, "Failed to reconcile workload")
			return err
		}

		// if the workload does not have any stash sidecar/init-container then
		// delete respective ConfigMapLock and RBAC stuffs if exist
		if err := c.ensureUnnecessaryConfigMapLockDeleted(w); err != nil {
			return err
		}
		return stash_rbac.EnsureUnnecessaryWorkloadRBACDeleted(c.kubeClient, logger, w)
	}
	return nil
}

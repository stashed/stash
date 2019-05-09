package controller

import (
	"github.com/appscode/go/log"
	"github.com/golang/glog"
	kerr "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/tools/cache"
	"kmodules.xyz/client-go/discovery"
	"kmodules.xyz/client-go/tools/queue"
	ocapps "kmodules.xyz/openshift/apis/apps/v1"
	ocapps_util "kmodules.xyz/openshift/client/clientset/versioned/typed/apps/v1/util"
	"kmodules.xyz/webhook-runtime/admission"
	hooks "kmodules.xyz/webhook-runtime/admission/v1beta1"
	webhook "kmodules.xyz/webhook-runtime/admission/v1beta1/workload"
	wapi "kmodules.xyz/webhook-runtime/apis/workload/v1"
	wcs "kmodules.xyz/webhook-runtime/client/workload/v1"
	"stash.appscode.dev/stash/apis"
	"stash.appscode.dev/stash/pkg/util"
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
				// apply stash backup/restore logic on this workload
				_, err := c.applyStashLogic(w, util.CallerWebhook)
				return w, err

			},
			UpdateFunc: func(oldObj, newObj runtime.Object) (runtime.Object, error) {
				w := newObj.(*wapi.Workload)
				// apply stash backup/restore logic on this workload
				_, err := c.applyStashLogic(w, util.CallerWebhook)
				return w, err
			},
		},
	)
}

func (c *StashController) initDeploymentConfigWatcher() {
	if !discovery.IsPreferredAPIResource(c.kubeClient.Discovery(), ocapps.GroupVersion.String(), apis.KindDeploymentConfig) {
		log.Warningf("Skipping watching non-preferred GroupVersion:%s Kind:%s", ocapps.GroupVersion.String(), apis.KindDeploymentConfig)
		return
	}
	c.dcInformer = c.ocInformerFactory.Apps().V1().DeploymentConfigs().Informer()
	c.dcQueue = queue.New(apis.KindDeploymentConfig, c.MaxNumRequeues, c.NumThreads, c.runDeploymentConfigProcessor)
	c.dcInformer.AddEventHandler(queue.DefaultEventHandler(c.dcQueue.GetQueue()))
	c.dcLister = c.ocInformerFactory.Apps().V1().DeploymentConfigs().Lister()
}

// syncToStdout is the business logic of the controller. In this controller it simply prints
// information about the deployment to stdout. In case an error happened, it has to simply return the error.
// The retry logic should not be part of the business logic.
func (c *StashController) runDeploymentConfigProcessor(key string) error {
	obj, exists, err := c.dcInformer.GetIndexer().GetByKey(key)
	if err != nil {
		glog.Errorf("Fetching object with key %s from store failed with %v", key, err)
		return err
	}

	if !exists {
		// Below we will warm up our cache with a DeploymentConfig, so that we will see a delete for one deployment
		glog.Warningf("DeploymentConfig %s does not exist anymore\n", key)

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
		glog.Infof("Sync/Add/Update for DeploymentConfig %s", key)

		dc := obj.(*ocapps.DeploymentConfig).DeepCopy()
		dc.GetObjectKind().SetGroupVersionKind(ocapps.GroupVersion.WithKind(apis.KindDeploymentConfig))

		// convert DeploymentConfig into a common object (Workload type) so that
		// we don't need to re-write stash logic for DeploymentConfig separately
		w, err := wcs.ConvertToWorkload(dc.DeepCopy())
		if err != nil {
			glog.Errorf("failed to convert DeploymentConfig %s/%s to workload type. Reason: %v", dc.Namespace, dc.Name, err)
			return err
		}

		// apply stash backup/restore logic on this workload
		modified, err := c.applyStashLogic(w, util.CallerController)
		if err != nil {
			glog.Errorf("failed to apply stash logic on DeploymentConfig %s/%s. Reason: %v", dc.Namespace, dc.Name, err)
			return err
		}

		if modified {
			// workload has been modified. Patch the workload so that respective pods start with the updated spec
			_, _, err := ocapps_util.PatchDeploymentConfigObject(c.ocClient, dc, w.Object.(*ocapps.DeploymentConfig))
			if err != nil {
				glog.Errorf("failed to update DeploymentConfig %s/%s. Reason: %v", dc.Namespace, dc.Name, err)
				return err
			}

			//TODO: Should we force restart all pods while restore?
			// otherwise one pod will restore while others are writing/reading?

			// wait until newly patched deploymentconfigs pods are ready
			err = util.WaitUntilDeploymentConfigReady(c.ocClient, dc.ObjectMeta)
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
		err = c.ensureUnnecessaryWorkloadRBACDeleted(w)
		if err != nil {
			return err
		}
	}
	return nil
}

package controller

import (
	"github.com/appscode/go/log"
	"github.com/appscode/kubernetes-webhook-util/admission"
	hooks "github.com/appscode/kubernetes-webhook-util/admission/v1beta1"
	webhook "github.com/appscode/kubernetes-webhook-util/admission/v1beta1/workload"
	workload "github.com/appscode/kubernetes-webhook-util/workload/v1"
	"github.com/appscode/kutil/tools/queue"
	apps_util "github.com/appscode/ocutil/apps/v1"
	api "github.com/appscode/stash/apis/stash/v1alpha1"
	"github.com/appscode/stash/pkg/util"
	"github.com/golang/glog"
	apps "github.com/openshift/api/apps/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/tools/cache"
)

func (c *StashController) NewDeploymentConfigWebhook() hooks.AdmissionHook {
	return webhook.NewWorkloadWebhook(
		schema.GroupVersionResource{
			Group:    "admission.stash.appscode.com",
			Version:  "v1alpha1",
			Resource: "deploymentconfigs",
		},
		"deploymentconfig",
		"DeploymentConfig",
		nil,
		&admission.ResourceHandlerFuncs{
			CreateFunc: func(obj runtime.Object) (runtime.Object, error) {
				w := obj.(*workload.Workload)
				_, _, err := c.mutateDeploymentConfig(w)
				return w, err

			},
			UpdateFunc: func(oldObj, newObj runtime.Object) (runtime.Object, error) {
				w := newObj.(*workload.Workload)
				_, _, err := c.mutateDeploymentConfig(w)
				return w, err
			},
		},
	)
}

func (c *StashController) initDeploymentConfigWatcher() {
	c.dcInformer = c.ocInformerFactory.Apps().V1().DeploymentConfigs().Informer()
	c.dcQueue = queue.New("DeploymentConfig", c.MaxNumRequeues, c.NumThreads, c.runDeploymentConfigInjector)
	c.dcInformer.AddEventHandler(queue.DefaultEventHandler(c.dcQueue.GetQueue()))
	c.dcLister = c.ocInformerFactory.Apps().V1().DeploymentConfigs().Lister()
}

// syncToStdout is the business logic of the controller. In this controller it simply prints
// information about the deploymentconfig to stdout. In case an error happened, it has to simply return the error.
// The retry logic should not be part of the business logic.
func (c *StashController) runDeploymentConfigInjector(key string) error {
	obj, exists, err := c.dcInformer.GetIndexer().GetByKey(key)
	if err != nil {
		glog.Errorf("Fetching object with key %s from store failed with %v", key, err)
		return err
	}

	if !exists {
		// Below we will warm up our cache with a DeploymentConfig, so that we will see a delete for one d
		glog.Warningf("DeploymentConfig %s does not exist anymore\n", key)

		ns, name, err := cache.SplitMetaNamespaceKey(key)
		if err != nil {
			return err
		}
		util.DeleteConfigmapLock(c.kubeClient, ns, api.LocalTypedReference{Kind: api.KindDeploymentConfig, Name: name})
	} else {
		glog.Infof("Sync/Add/Update for DeploymentConfig %s\n", key)

		dc := obj.(*apps.DeploymentConfig).DeepCopy()
		dc.GetObjectKind().SetGroupVersionKind(apps.SchemeGroupVersion.WithKind(api.KindDeploymentConfig))

		w, err := workload.ConvertToWorkload(dc.DeepCopy())
		if err != nil {
			return nil
		}
		// mutateDeploymentConfig add or remove sidecar to DeploymentConfig when necessary
		_, modified, err := c.mutateDeploymentConfig(w)
		if err != nil {
			return err
		}
		if modified {
			_, _, err := apps_util.PatchDeploymentConfigObject(c.ocClient, dc, w.Object.(*apps.DeploymentConfig))
			if err != nil {
				return err
			}
			return apps_util.WaitUntilDeploymentConfigReady(c.ocClient, dc.ObjectMeta)
		}
	}
	return nil
}

func (c *StashController) mutateDeploymentConfig(w *workload.Workload) (*api.Restic, bool, error) {
	oldRestic, err := util.GetAppliedRestic(w.Annotations)
	if err != nil {
		return nil, false, err
	}

	newRestic, err := util.FindRestic(c.rstLister, w.ObjectMeta)
	if err != nil {
		log.Errorf("Error while searching Restic for DeploymentConfig %s/%s.", w.Name, w.Namespace)
		return nil, false, err
	}

	if newRestic != nil && !util.ResticEqual(oldRestic, newRestic) {
		if !newRestic.Spec.Paused {
			err := c.ensureWorkloadSidecar(w, oldRestic, newRestic)
			if err != nil {
				return nil, false, err
			}
			workload.ApplyWorkload(w.Object, w)
			return newRestic, true, nil
		}
	} else if oldRestic != nil && newRestic == nil {
		err := c.ensureWorkloadSidecarDeleted(w, oldRestic)
		if err != nil {
			return nil, false, err
		}
		workload.ApplyWorkload(w.Object, w)
		err = util.DeleteConfigmapLock(c.kubeClient, w.Namespace, api.LocalTypedReference{Kind: api.KindDeploymentConfig, Name: w.Name})
		if err != nil {
			return nil, false, err
		}
		return oldRestic, true, nil
	}

	return oldRestic, false, nil
}

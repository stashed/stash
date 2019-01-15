package controller

import (
	"fmt"

	"github.com/appscode/go/log"
	"github.com/appscode/kubernetes-webhook-util/admission"
	hooks "github.com/appscode/kubernetes-webhook-util/admission/v1beta1"
	webhook "github.com/appscode/kubernetes-webhook-util/admission/v1beta1/generic"
	"github.com/appscode/kutil/tools/queue"
	"github.com/appscode/stash/apis"
	"github.com/appscode/stash/apis/stash"
	api "github.com/appscode/stash/apis/stash/v1alpha1"
	stash_util "github.com/appscode/stash/client/clientset/versioned/typed/stash/v1alpha1/util"
	"github.com/appscode/stash/pkg/docker"
	"github.com/appscode/stash/pkg/eventer"
	"github.com/appscode/stash/pkg/util"
	"github.com/golang/glog"
	core "k8s.io/api/core/v1"
	kerr "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/reference"
)

const (
	RecoveryEventComponent = "stash-recovery"
)

func (c *StashController) NewRecoveryWebhook() hooks.AdmissionHook {
	return webhook.NewGenericWebhook(
		schema.GroupVersionResource{
			Group:    "admission.stash.appscode.com",
			Version:  "v1alpha1",
			Resource: "recoveries",
		},
		"recovery",
		[]string{stash.GroupName},
		api.SchemeGroupVersion.WithKind("Recovery"),
		nil,
		&admission.ResourceHandlerFuncs{
			CreateFunc: func(obj runtime.Object) (runtime.Object, error) {
				return nil, obj.(*api.Recovery).IsValid()
			},
			UpdateFunc: func(oldObj, newObj runtime.Object) (runtime.Object, error) {
				return nil, newObj.(*api.Recovery).IsValid()
			},
		},
	)
}

func (c *StashController) initRecoveryWatcher() {
	c.recInformer = c.stashInformerFactory.Stash().V1alpha1().Recoveries().Informer()
	c.recQueue = queue.New("Recovery", c.MaxNumRequeues, c.NumThreads, c.runRecoveryInjector)
	c.recInformer.AddEventHandler(&cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			if r, ok := obj.(*api.Recovery); ok {
				if err := r.IsValid(); err != nil {
					ref, rerr := reference.GetReference(scheme.Scheme, r)
					if rerr == nil {
						c.recorder.Eventf(
							ref,
							core.EventTypeWarning,
							eventer.EventReasonInvalidRecovery,
							"Reason %v",
							err,
						)
					}
					return
				}
				queue.Enqueue(c.recQueue.GetQueue(), obj)
			}
		},
		UpdateFunc: func(oldObj, newObj interface{}) {
			oldRes, ok := oldObj.(*api.Recovery)
			if !ok {
				log.Errorln("Invalid Recovery object")
				return
			}
			newRes, ok := newObj.(*api.Recovery)
			if !ok {
				log.Errorln("Invalid Recovery object")
				return
			}
			if err := newRes.IsValid(); err != nil {
				ref, rerr := reference.GetReference(scheme.Scheme, newRes)
				if rerr == nil {
					c.recorder.Eventf(
						ref,
						core.EventTypeWarning,
						eventer.EventReasonInvalidRecovery,
						"Reason %v",
						err,
					)
				}
				return
			} else if !util.RecoveryEqual(oldRes, newRes) {
				queue.Enqueue(c.recQueue.GetQueue(), newObj)
			}
		},
		DeleteFunc: func(obj interface{}) {
			queue.Enqueue(c.recQueue.GetQueue(), obj)
		},
	})
	c.recLister = c.stashInformerFactory.Stash().V1alpha1().Recoveries().Lister()
}

// syncToStdout is the business logic of the controller. In this controller it simply prints
// information about the deployment to stdout. In case an error happened, it has to simply return the error.
// The retry logic should not be part of the business logic.
func (c *StashController) runRecoveryInjector(key string) error {
	obj, exists, err := c.recInformer.GetIndexer().GetByKey(key)
	if err != nil {
		glog.Errorf("Fetching object with key %s from store failed with %v", key, err)
		return err
	}

	if !exists {
		// Below we will warm up our cache with a Recovery, so that we will see a delete for one d
		glog.Warningf("Recovery %s does not exist anymore\n", key)
		return nil
	}

	d := obj.(*api.Recovery)
	glog.Infof("Sync/Add/Update for Recovery %s", d.GetName())
	return c.runRecoveryJob(d)
}

func (c *StashController) runRecoveryJob(rec *api.Recovery) error {
	if rec.Status.Phase == api.RecoverySucceeded || rec.Status.Phase == api.RecoveryRunning {
		return nil
	}

	image := docker.Docker{
		Registry: c.DockerRegistry,
		Image:    docker.ImageStash,
		Tag:      c.StashImageTag,
	}

	job, err := util.NewRecoveryJob(c.stashClient, rec, image)
	if err != nil {
		eventer.CreateEvent(c.kubeClient, RecoveryEventComponent, rec, core.EventTypeWarning, eventer.EventReasonJobFailedToCreate, err.Error())
		return err
	}
	if c.EnableRBAC {
		job.Spec.Template.Spec.ServiceAccountName = job.Name
	}

	job, err = c.kubeClient.BatchV1().Jobs(rec.Namespace).Create(job)
	if err != nil {
		if kerr.IsAlreadyExists(err) {
			log.Infoln("Skipping to create recovery job. Reason: job already exist")
			return nil
		}
		log.Errorln(err)

		stash_util.UpdateRecoveryStatus(c.stashClient.StashV1alpha1(), rec, func(in *api.RecoveryStatus) *api.RecoveryStatus {
			in.Phase = api.RecoveryFailed
			return in
		}, apis.EnableStatusSubresource)
		ref, rerr := reference.GetReference(scheme.Scheme, rec)
		if rerr == nil {
			eventer.CreateEvent(c.kubeClient, RecoveryEventComponent, ref, core.EventTypeWarning, eventer.EventReasonJobFailedToCreate, err.Error())
		}
		return err
	}

	if c.EnableRBAC {
		ref, err := reference.GetReference(scheme.Scheme, job)
		if err != nil {
			return err
		}
		if err := c.ensureRecoveryRBAC(ref); err != nil {
			err = fmt.Errorf("error ensuring rbac for recovery job %s, reason: %s", job.Name, err)
			eventer.CreateEvent(c.kubeClient, RecoveryEventComponent, rec, core.EventTypeWarning, eventer.EventReasonJobFailedToCreate, err.Error())
			return err
		}

		if err := c.ensureRepoReaderRBAC(ref, rec); err != nil {
			err = fmt.Errorf("error ensuring repository-reader rbac for recovery job %s, reason: %s", job.Name, err)
			eventer.CreateEvent(c.kubeClient, RecoveryEventComponent, rec, core.EventTypeWarning, eventer.EventReasonJobFailedToCreate, err.Error())
			return err
		}
	}

	log.Infoln("Recovery job created:", job.Name)
	ref, rerr := reference.GetReference(scheme.Scheme, rec)
	if rerr == nil {
		c.recorder.Eventf(ref, core.EventTypeNormal, eventer.EventReasonJobCreated, "Recovery job created: %s", job.Name)
	}
	stash_util.UpdateRecoveryStatus(c.stashClient.StashV1alpha1(), rec, func(in *api.RecoveryStatus) *api.RecoveryStatus {
		in.Phase = api.RecoveryRunning
		return in
	}, apis.EnableStatusSubresource)

	return nil
}

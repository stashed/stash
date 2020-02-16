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
	"fmt"

	"stash.appscode.dev/apimachinery/apis"
	"stash.appscode.dev/apimachinery/apis/stash"
	api "stash.appscode.dev/apimachinery/apis/stash/v1alpha1"
	stash_util "stash.appscode.dev/apimachinery/client/clientset/versioned/typed/stash/v1alpha1/util"
	"stash.appscode.dev/apimachinery/pkg/docker"
	"stash.appscode.dev/stash/pkg/eventer"
	stash_rbac "stash.appscode.dev/stash/pkg/rbac"
	"stash.appscode.dev/stash/pkg/util"

	"github.com/appscode/go/log"
	"github.com/golang/glog"
	batchv1 "k8s.io/api/batch/v1"
	core "k8s.io/api/core/v1"
	kerr "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/errors"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/reference"
	"kmodules.xyz/client-go/tools/queue"
	"kmodules.xyz/webhook-runtime/admission"
	hooks "kmodules.xyz/webhook-runtime/admission/v1beta1"
	webhook "kmodules.xyz/webhook-runtime/admission/v1beta1/generic"
)

const (
	RecoveryEventComponent = "stash-recovery"
)

func (c *StashController) NewRecoveryWebhook() hooks.AdmissionHook {
	return webhook.NewGenericWebhook(
		schema.GroupVersionResource{
			Group:    "admission.stash.appscode.com",
			Version:  "v1alpha1",
			Resource: "recoveryvalidators",
		},
		"recoveryvalidator",
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
		_, err2 := eventer.CreateEvent(c.kubeClient, RecoveryEventComponent, rec, core.EventTypeWarning, eventer.EventReasonJobFailedToCreate, err.Error())
		if err2 != nil {
			return err
		}
		return err
	}
	job.Spec.Template.Spec.ServiceAccountName = job.Name

	job, err = c.kubeClient.BatchV1().Jobs(rec.Namespace).Create(job)
	if err != nil {
		if kerr.IsAlreadyExists(err) {
			log.Infoln("Skipping to create recovery job. Reason: job already exist")
			return nil
		}
		log.Errorln(err)

		_, err2 := stash_util.UpdateRecoveryStatus(c.stashClient.StashV1alpha1(), rec, func(in *api.RecoveryStatus) *api.RecoveryStatus {
			in.Phase = api.RecoveryFailed
			return in
		})
		if err2 != nil {
			return err
		}
		_, err2 = eventer.CreateEvent(c.kubeClient, RecoveryEventComponent, rec, core.EventTypeWarning, eventer.EventReasonJobFailedToCreate, err.Error())
		if err2 != nil {
			return err
		}
		return err
	}

	owner := metav1.NewControllerRef(job, batchv1.SchemeGroupVersion.WithKind(apis.KindJob))

	if err := stash_rbac.EnsureRecoveryRBAC(c.kubeClient, owner, rec.Namespace); err != nil {
		err = fmt.Errorf("error ensuring rbac for recovery job %s, reason: %s", job.Name, err)
		_, err2 := eventer.CreateEvent(c.kubeClient, RecoveryEventComponent, rec, core.EventTypeWarning, eventer.EventReasonJobFailedToCreate, err.Error())
		return errors.NewAggregate([]error{err, err2})
	}

	if err := stash_rbac.EnsureRepoReaderRBAC(c.kubeClient, c.stashClient, owner, rec); err != nil {
		err = fmt.Errorf("error ensuring repository-reader rbac for recovery job %s, reason: %s", job.Name, err)
		_, err2 := eventer.CreateEvent(c.kubeClient, RecoveryEventComponent, rec, core.EventTypeWarning, eventer.EventReasonJobFailedToCreate, err.Error())
		return errors.NewAggregate([]error{err, err2})
	}

	log.Infoln("Recovery job created:", job.Name)
	ref, rerr := reference.GetReference(scheme.Scheme, rec)
	if rerr == nil {
		c.recorder.Eventf(ref, core.EventTypeNormal, eventer.EventReasonJobCreated, "Recovery job created: %s", job.Name)
	}
	_, err = stash_util.UpdateRecoveryStatus(c.stashClient.StashV1alpha1(), rec, func(in *api.RecoveryStatus) *api.RecoveryStatus {
		in.Phase = api.RecoveryRunning
		return in
	})

	return err
}

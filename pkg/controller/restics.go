package controller

import (
	"fmt"

	"github.com/appscode/go/log"
	batch_util "github.com/appscode/kutil/batch/v1beta1"
	core_util "github.com/appscode/kutil/core/v1"
	ext_util "github.com/appscode/kutil/extensions/v1beta1"
	"github.com/appscode/kutil/tools/queue"
	api "github.com/appscode/stash/apis/stash/v1alpha1"
	"github.com/appscode/stash/pkg/docker"
	"github.com/appscode/stash/pkg/eventer"
	"github.com/appscode/stash/pkg/util"
	"github.com/golang/glog"
	batch "k8s.io/api/batch/v1beta1"
	core "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/reference"
)

func (c *StashController) initBackupWatcher() {
	c.rstInformer = c.stashInformerFactory.Stash().V1alpha1().Backups().Informer()
	c.rstQueue = queue.New("Backup", c.options.MaxNumRequeues, c.options.NumThreads, c.runBackupInjector)
	c.rstInformer.AddEventHandler(&cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			if r, ok := obj.(*api.Backup); ok {
				if err := r.IsValid(); err != nil {
					c.recorder.Eventf(
						r.ObjectReference(),
						core.EventTypeWarning,
						eventer.EventReasonInvalidBackup,
						"Reason %v",
						err,
					)
					return
				}
				queue.Enqueue(c.rstQueue.GetQueue(), obj)
			}
		},
		UpdateFunc: func(oldObj, newObj interface{}) {
			oldRes, ok := oldObj.(*api.Backup)
			if !ok {
				log.Errorln("Invalid Backup object")
				return
			}
			newRes, ok := newObj.(*api.Backup)
			if !ok {
				log.Errorln("Invalid Backup object")
				return
			}
			if err := newRes.IsValid(); err != nil {
				c.recorder.Eventf(
					newRes.ObjectReference(),
					core.EventTypeWarning,
					eventer.EventReasonInvalidBackup,
					"Reason %v",
					err,
				)
				return
			} else if !util.BackupEqual(oldRes, newRes) {
				queue.Enqueue(c.rstQueue.GetQueue(), newObj)
			}
		},
		DeleteFunc: func(obj interface{}) {
			queue.Enqueue(c.rstQueue.GetQueue(), obj)
		},
	})
	c.rstLister = c.stashInformerFactory.Stash().V1alpha1().Backups().Lister()
}

// syncToStdout is the business logic of the controller. In this controller it simply prints
// information about the deployment to stdout. In case an error happened, it has to simply return the error.
// The retry logic should not be part of the business logic.
func (c *StashController) runBackupInjector(key string) error {
	obj, exists, err := c.rstInformer.GetIndexer().GetByKey(key)
	if err != nil {
		glog.Errorf("Fetching object with key %s from store failed with %v", key, err)
		return err
	}

	if !exists {
		// Below we will warm up our cache with a Backup, so that we will see a delete for one d
		glog.Warningf("Backup %s does not exist anymore\n", key)

		namespace, name, err := cache.SplitMetaNamespaceKey(key)
		if err != nil {
			return err
		}
		c.EnsureSidecarDeleted(namespace, name)
	} else {
		restic := obj.(*api.Backup)
		glog.Infof("Sync/Add/Update for Backup %s\n", restic.GetName())

		if restic.Spec.Type == api.BackupOffline {
			c.EnsureKubectlCronJob(restic)
		}
		c.EnsureSidecar(restic)
		c.EnsureSidecarDeleted(restic.Namespace, restic.Name)
	}
	return nil
}

func (c *StashController) EnsureKubectlCronJob(restic *api.Backup) error {
	image := docker.Docker{
		Registry: c.options.DockerRegistry,
		Image:    docker.ImageKubectl,
		Tag:      c.options.KubectlImageTag,
	}

	meta := metav1.ObjectMeta{
		Name:      util.KubectlCronPrefix + restic.Name,
		Namespace: restic.Namespace,
	}

	selector, err := metav1.LabelSelectorAsSelector(&restic.Spec.Selector)
	if err != nil {
		return err
	}

	cronJob, _, err := batch_util.CreateOrPatchCronJob(c.k8sClient, meta, func(in *batch.CronJob) *batch.CronJob {
		// set restic as cron-job owner
		in.OwnerReferences = []metav1.OwnerReference{
			{
				APIVersion: api.SchemeGroupVersion.String(),
				Kind:       api.ResourceKindBackup,
				Name:       restic.Name,
				UID:        restic.UID,
			},
		}

		if in.Labels == nil {
			in.Labels = map[string]string{}
		}
		in.Labels["app"] = util.AppLabelStash
		in.Labels[util.AnnotationBackup] = restic.Name
		in.Labels[util.AnnotationOperation] = util.OperationDeletePods

		// spec
		in.Spec.Schedule = restic.Spec.Schedule
		if in.Spec.JobTemplate.Labels == nil {
			in.Spec.JobTemplate.Labels = map[string]string{}
		}
		in.Spec.JobTemplate.Labels["app"] = util.AppLabelStash
		in.Spec.JobTemplate.Labels[util.AnnotationBackup] = restic.Name
		in.Spec.JobTemplate.Labels[util.AnnotationOperation] = util.OperationDeletePods

		in.Spec.JobTemplate.Spec.Template.Spec.Containers = core_util.UpsertContainer(
			in.Spec.JobTemplate.Spec.Template.Spec.Containers,
			core.Container{
				Name:  util.KubectlContainer,
				Image: image.ToContainerImage(),
				Args: []string{
					"kubectl",
					"delete",
					"pods",
					"-l " + selector.String(),
				},
			})
		in.Spec.JobTemplate.Spec.Template.Spec.ImagePullSecrets = restic.Spec.ImagePullSecrets

		in.Spec.JobTemplate.Spec.Template.Spec.RestartPolicy = core.RestartPolicyNever
		if c.options.EnableRBAC {
			in.Spec.JobTemplate.Spec.Template.Spec.ServiceAccountName = in.Name
		}
		return in
	})
	if err != nil {
		return err
	}

	if c.options.EnableRBAC {
		ref, err := reference.GetReference(scheme.Scheme, cronJob)
		if err != nil {
			return err
		}
		if err = c.ensureKubectlRBAC(ref); err != nil {
			return fmt.Errorf("error ensuring rbac for kubectl cron job %s, reason: %s\n", meta.Name, err)
		}
	}

	return nil
}

func (c *StashController) EnsureSidecar(restic *api.Backup) {
	sel, err := metav1.LabelSelectorAsSelector(&restic.Spec.Selector)
	if err != nil {
		c.recorder.Eventf(
			restic.ObjectReference(),
			core.EventTypeWarning,
			eventer.EventReasonInvalidBackup,
			"Reason: %s",
			err.Error(),
		)
		return
	}
	{
		if resources, err := c.dpLister.Deployments(restic.Namespace).List(sel); err == nil {
			for _, resource := range resources {
				key, err := cache.MetaNamespaceKeyFunc(resource)
				if err == nil {
					c.dpQueue.GetQueue().Add(key)
				}
			}
		}
	}
	{
		if resources, err := c.dsLister.DaemonSets(restic.Namespace).List(sel); err == nil {
			for _, resource := range resources {
				key, err := cache.MetaNamespaceKeyFunc(resource)
				if err == nil {
					c.dsQueue.GetQueue().Add(key)
				}
			}
		}
	}
	//{
	//	if resources, err := c.ssLister.StatefulSets(restic.Namespace).List(sel); err == nil {
	//		for _, resource := range resources {
	//			key, err := cache.MetaNamespaceKeyFunc(resource)
	//			if err == nil {
	//				c.ssQueue.GetQueue().Add(key)
	//			}
	//		}
	//	}
	//}
	{
		if resources, err := c.rcLister.ReplicationControllers(restic.Namespace).List(sel); err == nil {
			for _, resource := range resources {
				key, err := cache.MetaNamespaceKeyFunc(resource)
				if err == nil {
					c.rcQueue.GetQueue().Add(key)
				}
			}
		}
	}
	{
		if resources, err := c.rsLister.ReplicaSets(restic.Namespace).List(sel); err == nil {
			for _, resource := range resources {
				// If owned by a Deployment, skip it.
				if ext_util.IsOwnedByDeployment(resource) {
					continue
				}
				key, err := cache.MetaNamespaceKeyFunc(resource)
				if err == nil {
					c.rsQueue.GetQueue().Add(key)
				}
			}
		}
	}
}

func (c *StashController) EnsureSidecarDeleted(namespace, name string) {
	if resources, err := c.dpLister.Deployments(namespace).List(labels.Everything()); err == nil {
		for _, resource := range resources {
			restic, err := util.GetAppliedBackup(resource.Annotations)
			if err != nil {
				if ref, e2 := reference.GetReference(scheme.Scheme, resource); e2 == nil {
					c.recorder.Eventf(
						ref,
						core.EventTypeWarning,
						eventer.EventReasonInvalidBackup,
						"Reason: %s",
						err.Error(),
					)
				}
			} else if restic != nil && restic.Namespace == namespace && restic.Name == name {
				key, err := cache.MetaNamespaceKeyFunc(resource)
				if err == nil {
					c.dpQueue.GetQueue().Add(key)
				}
			}
		}
	}
	if resources, err := c.dsLister.DaemonSets(namespace).List(labels.Everything()); err == nil {
		for _, resource := range resources {
			restic, err := util.GetAppliedBackup(resource.Annotations)
			if err != nil {
				if ref, e2 := reference.GetReference(scheme.Scheme, resource); e2 == nil {
					c.recorder.Eventf(
						ref,
						core.EventTypeWarning,
						eventer.EventReasonInvalidBackup,
						"Reason: %s",
						err.Error(),
					)
				}
			} else if restic != nil && restic.Namespace == namespace && restic.Name == name {
				key, err := cache.MetaNamespaceKeyFunc(resource)
				if err == nil {
					c.dsQueue.GetQueue().Add(key)
				}
			}
		}
	}
	//if resources, err := c.ssLister.StatefulSets(namespace).List(labels.Everything()); err == nil {
	//	for _, resource := range resources {
	//		restic, err := util.GetAppliedBackup(resource.Annotations)
	//		if err != nil {
	//			c.recorder.Eventf(
	//				kutil.GetObjectReference(resource, apps.SchemeGroupVersion),
	//				core.EventTypeWarning,
	//				eventer.EventReasonInvalidBackup,
	//				"Reason: %s",
	//				err.Error(),
	//			)
	//		} else if restic != nil && restic.Namespace == namespace && restic.Name == name {
	//			key, err := cache.MetaNamespaceKeyFunc(resource)
	//			if err == nil {
	//				c.ssQueue.GetQueue().Add(key)
	//			}
	//		}
	//	}
	//}
	if resources, err := c.rcLister.ReplicationControllers(namespace).List(labels.Everything()); err == nil {
		for _, resource := range resources {
			restic, err := util.GetAppliedBackup(resource.Annotations)
			if err != nil {
				if ref, e2 := reference.GetReference(scheme.Scheme, resource); e2 == nil {
					c.recorder.Eventf(
						ref,
						core.EventTypeWarning,
						eventer.EventReasonInvalidBackup,
						"Reason: %s",
						err.Error(),
					)
				}
			} else if restic != nil && restic.Namespace == namespace && restic.Name == name {
				key, err := cache.MetaNamespaceKeyFunc(resource)
				if err == nil {
					c.rcQueue.GetQueue().Add(key)
				}
			}
		}
	}
	if resources, err := c.rsLister.ReplicaSets(namespace).List(labels.Everything()); err == nil {
		for _, resource := range resources {
			restic, err := util.GetAppliedBackup(resource.Annotations)
			if err != nil {
				if ref, e2 := reference.GetReference(scheme.Scheme, resource); e2 == nil {
					c.recorder.Eventf(
						ref,
						core.EventTypeWarning,
						eventer.EventReasonInvalidBackup,
						"Reason: %s",
						err.Error(),
					)
				}
			} else if restic != nil && restic.Namespace == namespace && restic.Name == name {
				key, err := cache.MetaNamespaceKeyFunc(resource)
				if err == nil {
					c.rsQueue.GetQueue().Add(key)
				}
			}
		}
	}
}

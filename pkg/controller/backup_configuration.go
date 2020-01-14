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
	"strings"

	"stash.appscode.dev/stash/apis"
	api_v1beta1 "stash.appscode.dev/stash/apis/stash/v1beta1"
	v1beta1_util "stash.appscode.dev/stash/client/clientset/versioned/typed/stash/v1beta1/util"
	"stash.appscode.dev/stash/pkg/docker"
	"stash.appscode.dev/stash/pkg/eventer"
	stash_rbac "stash.appscode.dev/stash/pkg/rbac"
	"stash.appscode.dev/stash/pkg/util"

	"github.com/appscode/go/log"
	"github.com/golang/glog"
	batch_v1beta1 "k8s.io/api/batch/v1beta1"
	core "k8s.io/api/core/v1"
	kerr "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/errors"
	"k8s.io/client-go/tools/cache"
	batch_util "kmodules.xyz/client-go/batch/v1beta1"
	core_util "kmodules.xyz/client-go/core/v1"
	"kmodules.xyz/client-go/tools/queue"
	workload_api "kmodules.xyz/webhook-runtime/apis/workload/v1"
)

// TODO: Add validator that will reject to create BackupConfiguration if any Restic exist for target workload

func (c *StashController) initBackupConfigurationWatcher() {
	c.bcInformer = c.stashInformerFactory.Stash().V1beta1().BackupConfigurations().Informer()
	c.bcQueue = queue.New(api_v1beta1.ResourceKindBackupConfiguration, c.MaxNumRequeues, c.NumThreads, c.runBackupConfigurationProcessor)
	c.bcInformer.AddEventHandler(queue.NewReconcilableHandler(c.bcQueue.GetQueue()))
	c.bcLister = c.stashInformerFactory.Stash().V1beta1().BackupConfigurations().Lister()
}

// syncToStdout is the business logic of the controller. In this controller it simply prints
// information about the deployment to stdout. In case an error happened, it has to simply return the error.
// The retry logic should not be part of the business logic.
func (c *StashController) runBackupConfigurationProcessor(key string) error {
	obj, exists, err := c.bcInformer.GetIndexer().GetByKey(key)
	if err != nil {
		glog.Errorf("Fetching object with key %s from store failed with %v", key, err)
		return err
	}
	if !exists {
		glog.Warningf("BackupConfiguration %s does not exit anymore\n", key)
		return nil
	}

	backupConfiguration := obj.(*api_v1beta1.BackupConfiguration)
	glog.Infof("Sync/Add/Update for BackupConfiguration %s", backupConfiguration.GetName())
	// process syc/add/update event
	invoker, err := apis.ExtractBackupInvokerInfo(c.stashClient, api_v1beta1.ResourceKindBackupConfiguration, backupConfiguration.Name, backupConfiguration.Namespace)
	if err != nil {
		return err
	}
	err = c.applyBackupInvokerReconciliationLogic(invoker)
	if err != nil {
		return err
	}

	// We have successfully completed respective stuffs for the current state of this resource.
	// Hence, let's set observed generation as same as the current generation.
	_, err = v1beta1_util.UpdateBackupConfigurationStatus(c.stashClient.StashV1beta1(), backupConfiguration, func(in *api_v1beta1.BackupConfigurationStatus) *api_v1beta1.BackupConfigurationStatus {
		in.ObservedGeneration = backupConfiguration.Generation
		return in
	})
	return err
}

func (c *StashController) applyBackupInvokerReconciliationLogic(invoker apis.Invoker) error {
	// check if BackupBatch is being deleted. if it is being deleted then delete respective resources.
	if invoker.ObjectMeta.DeletionTimestamp != nil {
		if core_util.HasFinalizer(invoker.ObjectMeta, api_v1beta1.StashKey) {
			for _, targetInfo := range invoker.TargetsInfo {
				if targetInfo.Target != nil {
					err := c.EnsureV1beta1SidecarDeleted(targetInfo.Target.Ref, invoker.ObjectMeta.Namespace)
					if err != nil {
						return c.handleWorkloadControllerTriggerFailure(invoker.ObjectRef, err)
					}
				}
			}

			if err := c.EnsureBackupTriggeringCronJobDeleted(invoker); err != nil {
				return err
			}
			// Remove finalizer
			return invoker.RemoveFinalizer()
		}
	} else {
		err := invoker.AddFinalizer()
		if err != nil {
			return err
		}
		// skip if BackupBatch paused
		if invoker.Paused {
			log.Infof("Skipping processing for invoker %s %s/%s. Reason: The invoker is paused.", invoker.ObjectRef.Kind, invoker.ObjectMeta.Namespace, invoker.ObjectMeta.Name)
			return nil
		}

		for _, targetInfo := range invoker.TargetsInfo {
			if targetInfo.Target != nil && invoker.Driver != api_v1beta1.VolumeSnapshotter &&
				util.BackupModel(targetInfo.Target.Ref.Kind) == apis.ModelSidecar {
				if err := c.EnsureV1beta1Sidecar(targetInfo.Target.Ref, invoker.ObjectMeta.Namespace); err != nil {
					return c.handleWorkloadControllerTriggerFailure(invoker.ObjectRef, err)
				}
			}
		}
		// create a CronJob that will create BackupSession on each schedule
		err = c.EnsureBackupTriggeringCronJob(invoker)
		if err != nil {
			return c.handleCronJobCreationFailure(invoker.ObjectRef, err)
		}
	}
	return nil
}

// EnsureV1beta1SidecarDeleted send an event to workload respective controller
// the workload controller will take care of removing respective sidecar
func (c *StashController) EnsureV1beta1SidecarDeleted(targetRef api_v1beta1.TargetRef, namespace string) error {
	return c.sendEventToWorkloadQueue(
		targetRef.Kind,
		namespace,
		targetRef.Name,
	)
}

// EnsureV1beta1Sidecar send an event to workload respective controller
// the workload controller will take care of injecting backup sidecar
func (c *StashController) EnsureV1beta1Sidecar(targetRef api_v1beta1.TargetRef, namespace string) error {
	return c.sendEventToWorkloadQueue(
		targetRef.Kind,
		namespace,
		targetRef.Name,
	)
}

func (c *StashController) sendEventToWorkloadQueue(kind, namespace, resourceName string) error {
	switch kind {
	case workload_api.KindDeployment:
		if resource, err := c.dpLister.Deployments(namespace).Get(resourceName); err == nil {
			key, err := cache.MetaNamespaceKeyFunc(resource)
			if err == nil {
				c.dpQueue.GetQueue().Add(key)
			}
			return err
		}
	case workload_api.KindDaemonSet:
		if resource, err := c.dsLister.DaemonSets(namespace).Get(resourceName); err == nil {
			key, err := cache.MetaNamespaceKeyFunc(resource)
			if err == nil {
				c.dsQueue.GetQueue().Add(key)
			}
			return err
		}
	case workload_api.KindStatefulSet:
		if resource, err := c.ssLister.StatefulSets(namespace).Get(resourceName); err == nil {
			key, err := cache.MetaNamespaceKeyFunc(resource)
			if err == nil {
				c.ssQueue.GetQueue().Add(key)
			}
		}
	case workload_api.KindReplicationController:
		if resource, err := c.rcLister.ReplicationControllers(namespace).Get(resourceName); err == nil {
			key, err := cache.MetaNamespaceKeyFunc(resource)
			if err == nil {
				c.rcQueue.GetQueue().Add(key)
			}
			return err
		}
	case workload_api.KindReplicaSet:
		if resource, err := c.rsLister.ReplicaSets(namespace).Get(resourceName); err == nil {
			key, err := cache.MetaNamespaceKeyFunc(resource)
			if err == nil {
				c.rsQueue.GetQueue().Add(key)
			}
			return err
		}
	case workload_api.KindDeploymentConfig:
		if c.ocClient != nil && c.dcLister != nil {
			if resource, err := c.dcLister.DeploymentConfigs(namespace).Get(resourceName); err == nil {
				key, err := cache.MetaNamespaceKeyFunc(resource)
				if err == nil {
					c.dcQueue.GetQueue().Add(key)
				}
				return err
			}
		}
	}
	return nil
}

// EnsureBackupTriggeringCronJob creates a Kubernetes CronJob for the respective backup invoker
// the CornJob will create a BackupSession object in each schedule
// respective BackupSession controller will watch this BackupSession object and take backup instantly
func (c *StashController) EnsureBackupTriggeringCronJob(invoker apis.Invoker) error {
	image := docker.Docker{
		Registry: c.DockerRegistry,
		Image:    docker.ImageStash,
		Tag:      c.StashImageTag,
	}

	meta := metav1.ObjectMeta{
		Name:      getBackupCronJobName(invoker.ObjectMeta.Name),
		Namespace: invoker.ObjectMeta.Namespace,
		Labels:    invoker.Labels,
	}

	// ensure respective ClusterRole,RoleBinding,ServiceAccount etc.
	var serviceAccountName string

	if invoker.RuntimeSettings.Pod != nil && invoker.RuntimeSettings.Pod.ServiceAccountName != "" {
		// ServiceAccount has been specified, so use it.
		serviceAccountName = invoker.RuntimeSettings.Pod.ServiceAccountName
	} else {
		// ServiceAccount hasn't been specified. so create new one with same name as BackupConfiguration object.
		serviceAccountName = meta.Name

		_, _, err := core_util.CreateOrPatchServiceAccount(c.kubeClient, meta, func(in *core.ServiceAccount) *core.ServiceAccount {
			core_util.EnsureOwnerReference(&in.ObjectMeta, invoker.OwnerRef)
			return in
		})
		if err != nil {
			return err
		}
	}

	// now ensure RBAC stuff for this CronJob
	err := stash_rbac.EnsureCronJobRBAC(c.kubeClient, invoker.OwnerRef, invoker.ObjectMeta.Namespace, serviceAccountName, c.getBackupSessionCronJobPSPNames(), invoker.Labels)
	if err != nil {
		return err
	}

	_, _, err = batch_util.CreateOrPatchCronJob(c.kubeClient, meta, func(in *batch_v1beta1.CronJob) *batch_v1beta1.CronJob {
		//set backup invoker object as cron-job owner
		core_util.EnsureOwnerReference(&in.ObjectMeta, invoker.OwnerRef)

		in.Spec.Schedule = invoker.Schedule
		in.Spec.JobTemplate.Labels = invoker.Labels
		// ensure that job gets deleted on completion
		in.Spec.JobTemplate.Labels[apis.KeyDeleteJobOnCompletion] = apis.AllowDeletingJobOnCompletion

		in.Spec.JobTemplate.Spec.Template.Spec.Containers = core_util.UpsertContainer(
			in.Spec.JobTemplate.Spec.Template.Spec.Containers,
			core.Container{
				Name:            apis.StashContainer,
				ImagePullPolicy: core.PullIfNotPresent,
				Image:           image.ToContainerImage(),
				Args: []string{
					"create-backupsession",
					fmt.Sprintf("--invoker-name=%s", invoker.OwnerRef.Name),
					fmt.Sprintf("--invoker-type=%s", invoker.OwnerRef.Kind),
				},
			})
		in.Spec.JobTemplate.Spec.Template.Spec.RestartPolicy = core.RestartPolicyNever
		in.Spec.JobTemplate.Spec.Template.Spec.ServiceAccountName = serviceAccountName
		// insert default pod level security context
		in.Spec.JobTemplate.Spec.Template.Spec.SecurityContext = util.UpsertDefaultPodSecurityContext(in.Spec.JobTemplate.Spec.Template.Spec.SecurityContext)
		return in
	})

	return err
}

// EnsureBackupTriggeringCronJobDeleted ensure that the CronJob of the respective backup invoker has it as owner.
// Kuebernetes garbage collector will take care of removing the CronJob
func (c *StashController) EnsureBackupTriggeringCronJobDeleted(invoker apis.Invoker) error {
	cur, err := c.kubeClient.BatchV1beta1().CronJobs(invoker.ObjectMeta.Namespace).Get(getBackupCronJobName(invoker.ObjectMeta.Name), metav1.GetOptions{})
	if err != nil {
		if kerr.IsNotFound(err) {
			return nil
		}
		return err
	}
	_, _, err = batch_util.PatchCronJob(c.kubeClient, cur, func(in *batch_v1beta1.CronJob) *batch_v1beta1.CronJob {
		core_util.EnsureOwnerReference(&in.ObjectMeta, invoker.OwnerRef)
		return in
	})
	return err
}

func getBackupCronJobName(name string) string {
	return strings.ReplaceAll(name, ".", "-")
}

func (c *StashController) handleCronJobCreationFailure(ref *core.ObjectReference, err error) error {
	if ref == nil {
		return errors.NewAggregate([]error{err, fmt.Errorf("failed to write cronjob creation failure event. Reason: provided ObjectReference is nil")})
	}

	var eventSource string
	switch ref.Kind {
	case api_v1beta1.ResourceKindBackupConfiguration:
		eventSource = eventer.EventSourceBackupConfigurationController
	case api_v1beta1.ResourceKindBackupBatch:
		eventSource = eventer.EventSourceBackupBatchController
	default:
		return errors.NewAggregate([]error{err, fmt.Errorf("failed to write cronjob creation failure event. Reason: Stash does not create cron job for %s", ref.Kind)})
	}
	// write log
	log.Warningf("failed to create CronJob for %s %s/%s. Reason: %v", ref.Kind, ref.Namespace, ref.Name, err)

	// write event to Backup invoker
	_, err2 := eventer.CreateEvent(
		c.kubeClient,
		eventSource,
		ref,
		core.EventTypeWarning,
		eventer.EventReasonCronJobCreationFailed,
		fmt.Sprintf("failed to ensure CronJob for %s  %s/%s. Reason: %v", ref.Kind, ref.Namespace, ref.Name, err))
	return errors.NewAggregate([]error{err, err2})
}

func (c *StashController) handleWorkloadControllerTriggerFailure(ref *core.ObjectReference, err error) error {
	if ref == nil {
		return errors.NewAggregate([]error{err, fmt.Errorf("failed to write workload controller triggering failure event. Reason: provided ObjectReference is nil")})
	}
	var eventSource string
	switch ref.Kind {
	case api_v1beta1.ResourceKindBackupConfiguration:
		eventSource = eventer.EventSourceBackupConfigurationController
	case api_v1beta1.ResourceKindBackupBatch:
		eventSource = eventer.EventSourceBackupBatchController
	case api_v1beta1.ResourceKindRestoreSession:
		eventSource = eventer.EventSourceRestoreSessionController
	}

	log.Warningf("failed to trigger workload controller for %s %s/%s. Reason: %v", ref.Kind, ref.Namespace, ref.Name, err)

	// write event to backup invoker/RestoreSession
	_, err2 := eventer.CreateEvent(
		c.kubeClient,
		eventSource,
		ref,
		core.EventTypeWarning,
		eventer.EventReasonWorkloadControllerTriggeringFailed,
		fmt.Sprintf("failed to trigger workload controller for %s %s/%s. Reason: %v", ref.Kind, ref.Namespace, ref.Name, err),
	)
	return errors.NewAggregate([]error{err, err2})
}

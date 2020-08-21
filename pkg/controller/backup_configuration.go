/*
Copyright AppsCode Inc. and Contributors

Licensed under the PolyForm Noncommercial License 1.0.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    https://github.com/appscode/licenses/raw/1.0.0/PolyForm-Noncommercial-1.0.0.md

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controller

import (
	"context"
	"fmt"
	"strings"
	"time"

	"stash.appscode.dev/apimachinery/apis"
	api_v1beta1 "stash.appscode.dev/apimachinery/apis/stash/v1beta1"
	v1beta1_util "stash.appscode.dev/apimachinery/client/clientset/versioned/typed/stash/v1beta1/util"
	"stash.appscode.dev/apimachinery/pkg/conditions"
	"stash.appscode.dev/apimachinery/pkg/docker"
	"stash.appscode.dev/stash/pkg/eventer"
	stash_rbac "stash.appscode.dev/stash/pkg/rbac"
	"stash.appscode.dev/stash/pkg/util"

	"github.com/appscode/go/log"
	"github.com/appscode/go/types"
	"github.com/golang/glog"
	batch_v1beta1 "k8s.io/api/batch/v1beta1"
	core "k8s.io/api/core/v1"
	kerr "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/errors"
	"k8s.io/client-go/tools/cache"
	batch_util "kmodules.xyz/client-go/batch/v1beta1"
	core_util "kmodules.xyz/client-go/core/v1"
	meta2 "kmodules.xyz/client-go/meta"
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
	err = c.applyBackupInvokerReconciliationLogic(invoker, key)
	if err != nil {
		return err
	}

	// We have successfully completed respective stuffs for the current state of this resource.
	// Hence, let's set observed generation as same as the current generation.
	_, err = v1beta1_util.UpdateBackupConfigurationStatus(
		context.TODO(),
		c.stashClient.StashV1beta1(),
		backupConfiguration.ObjectMeta,
		func(in *api_v1beta1.BackupConfigurationStatus) *api_v1beta1.BackupConfigurationStatus {
			in.ObservedGeneration = backupConfiguration.Generation
			return in
		},
		metav1.UpdateOptions{},
	)
	return err
}

func (c *StashController) applyBackupInvokerReconciliationLogic(invoker apis.Invoker, key string) error {
	// check if backup invoker is being deleted. if it is being deleted then delete respective resources.
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
	}
	err := invoker.AddFinalizer()
	if err != nil {
		return err
	}

	if invoker.Driver == "" || invoker.Driver == api_v1beta1.ResticSnapshotter {
		// Check whether Repository exist or not
		repository, err := c.stashClient.StashV1alpha1().Repositories(invoker.ObjectMeta.Namespace).Get(context.TODO(), invoker.Repository, metav1.GetOptions{})
		if err != nil {
			if kerr.IsNotFound(err) {
				glog.Infof("Repository %s/%s for invoker: %s %s/%s does not exist.\nRequeueing after 5 seconds......",
					invoker.ObjectMeta.Namespace,
					invoker.Repository,
					invoker.TypeMeta.Kind,
					invoker.ObjectMeta.Namespace,
					invoker.ObjectMeta.Name,
				)
				err2 := conditions.SetRepositoryFoundConditionToFalse(invoker)
				if err2 != nil {
					return err2
				}
				return c.requeueInvoker(invoker, key, 5*time.Second)
			}
			err2 := conditions.SetRepositoryFoundConditionToUnknown(invoker, err)
			return errors.NewAggregate([]error{err, err2})
		}
		err = conditions.SetRepositoryFoundConditionToTrue(invoker)
		if err != nil {
			return err
		}

		// Check whether the backend Secret exist or not
		secret, err := c.kubeClient.CoreV1().Secrets(repository.Namespace).Get(context.TODO(), repository.Spec.Backend.StorageSecretName, metav1.GetOptions{})
		if err != nil {
			if kerr.IsNotFound(err) {
				glog.Infof("Backend Secret %s/%s does not exist for Repository %s/%s.\nRequeueing after 5 seconds......",
					secret.Namespace,
					secret.Name,
					repository.Namespace,
					repository.Name,
				)
				err2 := conditions.SetBackendSecretFoundConditionToFalse(invoker, secret.Name)
				if err2 != nil {
					return err2
				}
				return c.requeueInvoker(invoker, key, 5*time.Second)
			}
			err2 := conditions.SetBackendSecretFoundConditionToUnknown(invoker, secret.Name, err)
			return errors.NewAggregate([]error{err, err2})
		}
		err = conditions.SetBackendSecretFoundConditionToTrue(invoker, secret.Name)
		if err != nil {
			return err
		}
	}

	someTargetMissing := false
	for _, targetInfo := range invoker.TargetsInfo {
		if targetInfo.Target != nil {
			tref := targetInfo.Target.Ref
			wc := util.WorkloadClients{
				KubeClient:       c.kubeClient,
				OcClient:         c.ocClient,
				StashClient:      c.stashClient,
				CRDClient:        c.crdClient,
				AppCatalogClient: c.appCatalogClient,
			}
			targetExist, err := wc.IsTargetExist(tref, invoker.ObjectMeta.Namespace)
			if err != nil {
				glog.Errorf("Failed to check whether %s %s %s/%s exist or not. Reason: %v.",
					tref.APIVersion,
					tref.Kind,
					invoker.ObjectMeta.Namespace,
					tref.Name,
					err.Error(),
				)
				// Set the "BackupTargetFound" condition to "Unknown"
				cerr := conditions.SetBackupTargetFoundConditionToUnknown(invoker, tref, err)
				return errors.NewAggregate([]error{err, cerr})
			}
			if !targetExist {
				// Target does not exist. Log the information.
				someTargetMissing = true
				glog.Infof("Backup target %s %s %s/%s does not exist.",
					tref.APIVersion,
					tref.Kind,
					invoker.ObjectMeta.Namespace,
					tref.Name)
				err = conditions.SetBackupTargetFoundConditionToFalse(invoker, tref)
				if err != nil {
					return err
				}
				// Process next target.
				continue
			}
			// Backup target exist. So, set "BackupTargetFound" condition to "True"
			err = conditions.SetBackupTargetFoundConditionToTrue(invoker, tref)
			if err != nil {
				return err
			}
			// For sidecar model, ensure the stash sidecar
			if (invoker.Driver == "" || invoker.Driver == api_v1beta1.ResticSnapshotter) && util.BackupModel(tref.Kind) == apis.ModelSidecar {
				err := c.EnsureV1beta1Sidecar(tref, invoker.ObjectMeta.Namespace)
				if err != nil {
					return c.handleWorkloadControllerTriggerFailure(invoker.ObjectRef, err)
				}
			}
		}

	}
	// If some backup targets are missing, then retry after some time.
	if someTargetMissing {
		glog.Infof("Some targets are missing for backup invoker: %s %s/%s.\nRequeueing after 5 seconds......",
			invoker.TypeMeta.Kind,
			invoker.ObjectMeta.Namespace,
			invoker.ObjectMeta.Name,
		)
		return c.requeueInvoker(invoker, key, 5*time.Second)
	}
	// create a CronJob that will create BackupSession on each schedule
	err = c.EnsureBackupTriggeringCronJob(invoker)
	if err != nil {
		// Failed to ensure the backup triggering CronJob. So, set "CronJobCreated" condition to "False"
		cerr := conditions.SetCronJobCreatedConditionToFalse(invoker, err)
		return c.handleCronJobCreationFailure(invoker.ObjectRef, errors.NewAggregate([]error{err, cerr}))
	}
	// Successfully ensured the backup triggering CronJob. So, set "CronJobCreated" condition to "True"
	return conditions.SetCronJobCreatedConditionToTrue(invoker)
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
		Image:    c.StashImage,
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
		// ServiceAccount hasn't been specified. so create new one with same name as BackupConfiguration object prefixed with stash-backup.
		serviceAccountName = meta.Name

		_, _, err := core_util.CreateOrPatchServiceAccount(
			context.TODO(),
			c.kubeClient,
			meta,
			func(in *core.ServiceAccount) *core.ServiceAccount {
				core_util.EnsureOwnerReference(&in.ObjectMeta, invoker.OwnerRef)
				return in
			},
			metav1.PatchOptions{},
		)
		if err != nil {
			return err
		}
	}

	// now ensure RBAC stuff for this CronJob
	err := stash_rbac.EnsureCronJobRBAC(c.kubeClient, invoker.OwnerRef, invoker.ObjectMeta.Namespace, serviceAccountName, c.getBackupSessionCronJobPSPNames(), invoker.Labels)
	if err != nil {
		return err
	}

	// if the Stash is using a private registry, then ensure the image pull secrets
	var imagePullSecrets []core.LocalObjectReference
	if c.ImagePullSecrets != nil {
		imagePullSecrets, err = c.ensureImagePullSecrets(invoker.ObjectMeta, invoker.OwnerRef)
		if err != nil {
			return err
		}
	}
	_, _, err = batch_util.CreateOrPatchCronJob(
		context.TODO(),
		c.kubeClient,
		meta,
		func(in *batch_v1beta1.CronJob) *batch_v1beta1.CronJob {
			//set backup invoker object as cron-job owner
			core_util.EnsureOwnerReference(&in.ObjectMeta, invoker.OwnerRef)

			in.Spec.Schedule = invoker.Schedule
			in.Spec.Suspend = types.BoolP(invoker.Paused) // this ensure that the CronJob is suspended when the backup invoker is paused.
			in.Spec.JobTemplate.Labels = invoker.Labels
			// ensure that job gets deleted on completion
			in.Spec.JobTemplate.Labels[apis.KeyDeleteJobOnCompletion] = apis.AllowDeletingJobOnCompletion

			container := core.Container{
				Name:            apis.StashContainer,
				ImagePullPolicy: core.PullIfNotPresent,
				Image:           image.ToContainerImage(),
				Args: []string{
					"create-backupsession",
					fmt.Sprintf("--invoker-name=%s", invoker.OwnerRef.Name),
					fmt.Sprintf("--invoker-kind=%s", invoker.OwnerRef.Kind),
				},
			}
			// only apply the container level runtime settings that make sense for the CronJob
			if invoker.RuntimeSettings.Container != nil {
				container.Resources = invoker.RuntimeSettings.Container.Resources
				container.Env = invoker.RuntimeSettings.Container.Env
				container.EnvFrom = invoker.RuntimeSettings.Container.EnvFrom
				container.SecurityContext = invoker.RuntimeSettings.Container.SecurityContext
			}

			in.Spec.JobTemplate.Spec.Template.Spec.Containers = core_util.UpsertContainer(
				in.Spec.JobTemplate.Spec.Template.Spec.Containers, container)
			in.Spec.JobTemplate.Spec.Template.Spec.RestartPolicy = core.RestartPolicyNever
			in.Spec.JobTemplate.Spec.Template.Spec.ServiceAccountName = serviceAccountName
			in.Spec.JobTemplate.Spec.Template.Spec.ImagePullSecrets = imagePullSecrets

			// only apply the pod level runtime settings that make sense for the CronJob
			if invoker.RuntimeSettings.Pod != nil {
				if len(invoker.RuntimeSettings.Pod.ImagePullSecrets) != 0 {
					in.Spec.JobTemplate.Spec.Template.Spec.ImagePullSecrets = invoker.RuntimeSettings.Pod.ImagePullSecrets
				}
				if invoker.RuntimeSettings.Pod.SecurityContext != nil {
					in.Spec.JobTemplate.Spec.Template.Spec.SecurityContext = invoker.RuntimeSettings.Pod.SecurityContext
				}
			}

			return in
		},
		metav1.PatchOptions{},
	)

	return err
}

// EnsureBackupTriggeringCronJobDeleted ensure that the CronJob of the respective backup invoker has it as owner.
// Kuebernetes garbage collector will take care of removing the CronJob
func (c *StashController) EnsureBackupTriggeringCronJobDeleted(invoker apis.Invoker) error {
	cur, err := c.kubeClient.BatchV1beta1().CronJobs(invoker.ObjectMeta.Namespace).Get(context.TODO(), getBackupCronJobName(invoker.ObjectMeta.Name), metav1.GetOptions{})
	if err != nil {
		if kerr.IsNotFound(err) {
			return nil
		}
		return err
	}
	_, _, err = batch_util.PatchCronJob(
		context.TODO(),
		c.kubeClient,
		cur,
		func(in *batch_v1beta1.CronJob) *batch_v1beta1.CronJob {
			core_util.EnsureOwnerReference(&in.ObjectMeta, invoker.OwnerRef)
			return in
		},
		metav1.PatchOptions{},
	)
	return err
}

func getBackupCronJobName(name string) string {
	return meta2.ValidCronJobNameWithPrefix(apis.PrefixStashBackup, strings.ReplaceAll(name, ".", "-"))
}

func (c *StashController) handleCronJobCreationFailure(ref *core.ObjectReference, err error) error {
	if ref == nil {
		return errors.NewAggregate([]error{err, fmt.Errorf("failed to write cronjob creation failure event. Reason: provided ObjectReference is nil")})
	}

	// write log
	log.Warningf("failed to create CronJob for %s %s/%s. Reason: %v", ref.Kind, ref.Namespace, ref.Name, err)

	// write event to Backup invoker
	_, err2 := eventer.CreateEvent(
		c.kubeClient,
		eventer.EventSourceBackupConfigurationController,
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

func (c *StashController) requeueInvoker(invoker apis.Invoker, key string, delay time.Duration) error {
	switch invoker.TypeMeta.Kind {
	case api_v1beta1.ResourceKindBackupConfiguration:
		c.bcQueue.GetQueue().AddAfter(key, delay)
	default:
		return fmt.Errorf("unable to requeue. Reason: Backup invoker %s  %s is not supported",
			invoker.TypeMeta.APIVersion,
			invoker.TypeMeta.Kind,
		)
	}
	return nil
}

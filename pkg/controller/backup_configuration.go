package controller

import (
	"fmt"

	api_v1beta1 "github.com/appscode/stash/apis/stash/v1beta1"
	stash_scheme "github.com/appscode/stash/client/clientset/versioned/scheme"
	stash_v1beta1_util "github.com/appscode/stash/client/clientset/versioned/typed/stash/v1beta1/util"
	"github.com/appscode/stash/pkg/docker"
	"github.com/appscode/stash/pkg/util"
	"github.com/golang/glog"
	batch_v1beta1 "k8s.io/api/batch/v1beta1"
	core "k8s.io/api/core/v1"
	kerr "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/reference"
	batch_util "kmodules.xyz/client-go/batch/v1beta1"
	core_util "kmodules.xyz/client-go/core/v1"
	"kmodules.xyz/client-go/tools/queue"
	workload_api "kmodules.xyz/webhook-runtime/apis/workload/v1"
)

// TODO: Add validator that will reject to create BackupConfiguration if any Restic exist for target workload

func (c *StashController) initBackupConfigurationWatcher() {
	c.bcInformer = c.stashInformerFactory.Stash().V1beta1().BackupConfigurations().Informer()
	c.bcQueue = queue.New(api_v1beta1.ResourceKindBackupConfiguration, c.MaxNumRequeues, c.NumThreads, c.runBackupConfigurationProcessor)
	c.bcInformer.AddEventHandler(queue.DefaultEventHandler(c.bcQueue.GetQueue()))
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

	} else {
		backupConfiguration := obj.(*api_v1beta1.BackupConfiguration)
		glog.Infof("Sync/Add/Update for BackupConfiguration %s", backupConfiguration.GetName())
		// check if BackupConfiguration is being deleted. if it is being deleted then delete respective resources.
		if backupConfiguration.DeletionTimestamp != nil {
			if core_util.HasFinalizer(backupConfiguration.ObjectMeta, api_v1beta1.StashKey) {
				if err = c.EnsureV1beta1SidecarDeleted(backupConfiguration); err != nil {
					return err
				}
				if err = c.EnsureCronJobDeleted(backupConfiguration); err != nil {
					return err
				}
				// Remove finalizer
				_, _, err = stash_v1beta1_util.PatchBackupConfiguration(c.stashClient.StashV1beta1(), backupConfiguration, func(in *api_v1beta1.BackupConfiguration) *api_v1beta1.BackupConfiguration {
					in.ObjectMeta = core_util.RemoveFinalizer(in.ObjectMeta, api_v1beta1.StashKey)
					return in

				})
				if err != nil {
					return err
				}
			}
		} else {
			// add a finalizer so that we can remove respective resources before this BackupConfiguration is deleted
			_, _, err := stash_v1beta1_util.PatchBackupConfiguration(c.stashClient.StashV1beta1(), backupConfiguration, func(in *api_v1beta1.BackupConfiguration) *api_v1beta1.BackupConfiguration {
				in.ObjectMeta = core_util.AddFinalizer(in.ObjectMeta, api_v1beta1.StashKey)
				return in
			})
			if err != nil {
				return err
			}

			if backupConfiguration.Spec.Target != nil &&
				util.BackupModel(backupConfiguration.Spec.Target.Ref.Kind) == util.ModelSidecar {
				if err := c.EnsureV1beta1Sidecar(backupConfiguration); err != nil {
					return err
				}
			}
			// create a CronJob that will create BackupSession on each schedule
			return c.EnsureCronJob(backupConfiguration)
		}
	}
	return nil
}

// EnsureV1beta1SidecarDeleted send an event to workload respective controller
// the workload controller will take care of removing respective sidecar
func (c *StashController) EnsureV1beta1SidecarDeleted(backupConfiguration *api_v1beta1.BackupConfiguration) error {
	if backupConfiguration == nil {
		return fmt.Errorf("BackupConfiguration is nil")
	}

	if backupConfiguration.Spec.Target != nil {
		return c.sendEventToWorkloadQueue(
			backupConfiguration.Spec.Target.Ref.Kind,
			backupConfiguration.Namespace,
			backupConfiguration.Spec.Target.Ref.Name,
		)
	}
	return nil
}

// EnsureV1beta1Sidecar send an event to workload respective controller
// the workload controller will take care of injecting backup sidecar
func (c *StashController) EnsureV1beta1Sidecar(backupConfiguration *api_v1beta1.BackupConfiguration) error {
	if backupConfiguration == nil {
		return fmt.Errorf("BackupConfiguration is nil")
	}

	return c.sendEventToWorkloadQueue(
		backupConfiguration.Spec.Target.Ref.Kind,
		backupConfiguration.Namespace,
		backupConfiguration.Spec.Target.Ref.Name,
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
	}
	return nil
}

// EnsureCronJob creates a Kubernetes CronJob for a BackupConfiguration object
// the CornJob will create a BackupSession object in each schedule
// respective BackupSession controller will watch this BackupSession object and take backup instantly
func (c *StashController) EnsureCronJob(backupConfiguration *api_v1beta1.BackupConfiguration) error {
	if backupConfiguration == nil {
		return fmt.Errorf("BackupConfiguration is nil")
	}
	image := docker.Docker{
		Registry: c.DockerRegistry,
		Image:    docker.ImageStash,
		Tag:      c.StashImageTag,
	}

	meta := metav1.ObjectMeta{
		Name:      backupConfiguration.Name,
		Namespace: backupConfiguration.Namespace,
	}
	ref, err := reference.GetReference(stash_scheme.Scheme, backupConfiguration)
	if err != nil {
		return err
	}

	// if RBAC is enabled then ensure respective ClusterRole,RoleBinding,ServiceAccount etc.
	serviceAccountName := "default"

	if c.EnableRBAC {
		if backupConfiguration.Spec.RuntimeSettings.Pod != nil &&
			backupConfiguration.Spec.RuntimeSettings.Pod.ServiceAccountName != "" {
			// ServiceAccount has been specified, so use it.
			serviceAccountName = backupConfiguration.Spec.RuntimeSettings.Pod.ServiceAccountName
		} else {
			// ServiceAccount hasn't been specified. so create new one with same name as BackupConfiguration object.
			serviceAccountName = meta.Name

			_, _, err := core_util.CreateOrPatchServiceAccount(c.kubeClient, meta, func(in *core.ServiceAccount) *core.ServiceAccount {
				core_util.EnsureOwnerReference(&in.ObjectMeta, ref)
				if in.Labels == nil {
					in.Labels = map[string]string{}
				}
				in.Labels[util.LabelApp] = util.AppLabelStash
				return in
			})
			if err != nil {
				return err
			}
		}

		// now ensure RBAC stuff for this CronJob
		err := c.ensureCronJobRBAC(ref, serviceAccountName)
		if err != nil {
			return err
		}
	}
	_, _, err = batch_util.CreateOrPatchCronJob(c.kubeClient, meta, func(in *batch_v1beta1.CronJob) *batch_v1beta1.CronJob {
		//set backup-configuration as cron-job owner
		core_util.EnsureOwnerReference(&in.ObjectMeta, ref)

		in.Spec.Schedule = backupConfiguration.Spec.Schedule
		if in.Spec.JobTemplate.Labels == nil {
			in.Spec.JobTemplate.Labels = map[string]string{}
		}
		in.Spec.JobTemplate.Labels[util.LabelApp] = util.AppLabelStash
		in.Spec.JobTemplate.Spec.Template.Spec.Containers = core_util.UpsertContainer(
			in.Spec.JobTemplate.Spec.Template.Spec.Containers,
			core.Container{
				Name:            util.StashContainer,
				ImagePullPolicy: core.PullIfNotPresent,
				Image:           image.ToContainerImage(),
				Args: []string{
					"backup-session",
					fmt.Sprintf("--backupsession.name=%s", backupConfiguration.Name),
					fmt.Sprintf("--backupsession.namespace=%s", backupConfiguration.Namespace),
				},
			})
		in.Spec.JobTemplate.Spec.Template.Spec.RestartPolicy = core.RestartPolicyNever
		if c.EnableRBAC {
			in.Spec.JobTemplate.Spec.Template.Spec.ServiceAccountName = serviceAccountName
		}
		return in
	})

	return err
}

// EnsureCronJobDelete ensure that respective CronJob of a BackupConfiguration has it as owner.
// Kuebernetes garbage collector will take care of removing the CronJob
func (c *StashController) EnsureCronJobDeleted(backupConfiguration *api_v1beta1.BackupConfiguration) error {
	ref, err := reference.GetReference(stash_scheme.Scheme, backupConfiguration)
	if err != nil {
		return err
	}

	cur, err := c.kubeClient.BatchV1beta1().CronJobs(backupConfiguration.Namespace).Get(backupConfiguration.Name, metav1.GetOptions{})
	if err != nil {
		if kerr.IsNotFound(err) {
			return nil
		}
		return err
	}

	_, _, err = batch_util.PatchCronJob(c.kubeClient, cur, func(in *batch_v1beta1.CronJob) *batch_v1beta1.CronJob {
		core_util.EnsureOwnerReference(&in.ObjectMeta, ref)
		return in
	})
	return err
}

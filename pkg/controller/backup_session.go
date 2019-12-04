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
	"sort"
	"strconv"
	"strings"
	"time"

	"stash.appscode.dev/stash/apis"
	"stash.appscode.dev/stash/apis/stash"
	api_v1beta1 "stash.appscode.dev/stash/apis/stash/v1beta1"
	stash_scheme "stash.appscode.dev/stash/client/clientset/versioned/scheme"
	stash_util "stash.appscode.dev/stash/client/clientset/versioned/typed/stash/v1beta1/util"
	"stash.appscode.dev/stash/pkg/docker"
	"stash.appscode.dev/stash/pkg/eventer"
	stash_rbac "stash.appscode.dev/stash/pkg/rbac"
	"stash.appscode.dev/stash/pkg/resolve"
	"stash.appscode.dev/stash/pkg/restic"
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
	"k8s.io/client-go/tools/clientcmd/api"
	"k8s.io/client-go/tools/reference"
	batch_util "kmodules.xyz/client-go/batch/v1"
	core_util "kmodules.xyz/client-go/core/v1"
	"kmodules.xyz/client-go/meta"
	"kmodules.xyz/client-go/tools/queue"
	"kmodules.xyz/webhook-runtime/admission"
	hooks "kmodules.xyz/webhook-runtime/admission/v1beta1"
	webhook "kmodules.xyz/webhook-runtime/admission/v1beta1/generic"
)

const (
	BackupJobPrefix                = "backup"
	VolumeSnapshotPrefix           = "vs"
	PromJobBackupSessionController = "stash-backupsession-controller"
)

func (c *StashController) NewBackupSessionWebhook() hooks.AdmissionHook {
	return webhook.NewGenericWebhook(
		schema.GroupVersionResource{
			Group:    "admission.stash.appscode.com",
			Version:  "v1beta1",
			Resource: "backupsessionvalidators",
		},
		"backupsessionvalidator",
		[]string{stash.GroupName},
		api.SchemeGroupVersion.WithKind(api_v1beta1.ResourceKindBackupSession),
		nil,
		&admission.ResourceHandlerFuncs{
			CreateFunc: func(obj runtime.Object) (runtime.Object, error) {
				return nil, obj.(*api_v1beta1.BackupSession).IsValid()
			},
			UpdateFunc: func(oldObj, newObj runtime.Object) (runtime.Object, error) {
				// should not allow spec update
				if !meta.Equal(oldObj.(*api_v1beta1.BackupSession).Spec, newObj.(*api_v1beta1.BackupSession).Spec) {
					return nil, fmt.Errorf("BackupSession spec is immutable")
				}
				return nil, nil
			},
		},
	)
}

// process only add events
func (c *StashController) initBackupSessionWatcher() {
	c.backupSessionInformer = c.stashInformerFactory.Stash().V1beta1().BackupSessions().Informer()
	c.backupSessionQueue = queue.New(api_v1beta1.ResourceKindBackupSession, c.MaxNumRequeues, c.NumThreads, c.runBackupSessionProcessor)
	c.backupSessionInformer.AddEventHandler(queue.DefaultEventHandler(c.backupSessionQueue.GetQueue()))
	c.backupSessionLister = c.stashInformerFactory.Stash().V1beta1().BackupSessions().Lister()
}

func (c *StashController) runBackupSessionProcessor(key string) error {
	obj, exists, err := c.backupSessionInformer.GetIndexer().GetByKey(key)
	if err != nil {
		glog.Errorf("Fetching object with key %s from store failed with %v", key, err)
		return err
	}
	if !exists {
		glog.Warningf("BackupSession %s does not exist anymore\n", key)
		return nil
	}

	backupSession := obj.(*api_v1beta1.BackupSession)
	glog.Infof("Sync/Add/Update for BackupSession %s", backupSession.GetName())
	// process sync/add/update event
	return c.applyBackupSessionReconciliationLogic(backupSession)
}

func (c *StashController) applyBackupSessionReconciliationLogic(backupSession *api_v1beta1.BackupSession) error {
	if backupSession.Status.Phase == api_v1beta1.BackupSessionFailed ||
		backupSession.Status.Phase == api_v1beta1.BackupSessionSucceeded {
		log.Infof("Skipping processing BackupSession %s/%s. Reason: phase is %q.", backupSession.Namespace, backupSession.Name, backupSession.Status.Phase)
		return nil
	}
	// check whether backup session is completed or running and set it's phase accordingly
	phase, err := c.getBackupSessionPhase(backupSession)

	if phase == api_v1beta1.BackupSessionFailed {
		// one or more hosts has failed to complete their backup process.
		// mark entire backup session as failure.
		// individual hosts has updated their respective stats and has sent respective metrics.
		// now, just set BackupSession phase "Failed" and create an event.
		return c.setBackupSessionFailed(backupSession, err)
	} else if phase == api_v1beta1.BackupSessionSucceeded {
		// all hosts has completed their backup process successfully.
		// individual hosts has updated their respective stats and has sent respective metrics.
		// now, just set BackupSession phase "Succeeded" and create an event.
		return c.setBackupSessionSucceeded(backupSession)
	} else if phase == api_v1beta1.BackupSessionRunning {
		log.Infof("Skipping processing BackupSession %s/%s. Reason: phase is %q.", backupSession.Namespace, backupSession.Name, backupSession.Status.Phase)
		return nil
	} else if phase == api_v1beta1.BackupSessionSkipped {
		log.Infof("Skipping processing BackupSession %s/%s. Reason: previously skipped.", backupSession.Namespace, backupSession.Name)
		return nil
	}

	// backup process for this BackupSession has not started. so let's start backup process
	// get BackupConfiguration/BackupBatch for BackupSession
	if backupSession.Spec.Invoker.Kind == api_v1beta1.ResourceKindBackupBatch {
		return c.invokeBackupBatch(backupSession)
	}
	return c.invokeBackupConfiguration(backupSession)
}

func (c *StashController) invokeBackupBatch(backupSession *api_v1beta1.BackupSession) error {
	backupBatch, err := c.stashClient.StashV1beta1().BackupBatches(backupSession.Namespace).Get(
		backupSession.Spec.Invoker.Name,
		metav1.GetOptions{},
	)
	if err != nil {
		return fmt.Errorf("can't get BackupBatch for BackupSession %s/%s, Reason: %s", backupBatch.Namespace, backupBatch.Name, err)
	}

	// skip if BackupBatch paused
	if backupBatch.Spec.Paused {
		log.Infof("Skipping processing BackupSession %s/%s. Reason: BackupBatch is paused.", backupSession.Namespace, backupSession.Name)
		return c.setBackupSessionSkipped(backupSession, fmt.Sprintf("BackupBatch %s/%s is paused", backupBatch.Namespace, backupBatch.Name))
	}

	// skip if backup model is sidecar.
	// for sidecar model controller inside sidecar will take care of it.
	for _, backupConfigTemp := range backupBatch.Spec.BackupConfigurationTemplates {
		if backupConfigTemp.Spec.Target != nil && backupBatch.Spec.Driver != api_v1beta1.VolumeSnapshotter && util.BackupModel(backupConfigTemp.Spec.Target.Ref.Kind) == util.ModelSidecar {
			log.Infof("Skipping processing BackupSession %s/%s. Reason: Backup model is sidecar. Controller inside sidecar will take care of it.", backupSession.Namespace, backupSession.Name)
			backupSession, err = c.setBackupSessionRunning(backupConfigTemp.Spec.Target, backupBatch.Spec.Driver, backupSession)
			if err != nil {
				return err
			}
		}
	}

	// create BackupBatch object reference
	// and config for VolumeSnapshotter Job/backup Job
	ref, err := reference.GetReference(stash_scheme.Scheme, backupBatch)
	if err != nil {
		return err
	}
	config := backupOption{
		objMeta:         backupBatch.ObjectMeta,
		offshootLabels:  backupBatch.OffshootLabels(),
		ref:             ref,
		repositoryName:  backupBatch.Spec.Repository.Name,
		retentionPolicy: backupBatch.Spec.RetentionPolicy,
	}

	// if VolumeSnapshotter driver is used then ensure VolumeSnapshotter job and return
	if backupBatch.Spec.Driver == api_v1beta1.VolumeSnapshotter {
		for _, backupConfigTemp := range backupBatch.Spec.BackupConfigurationTemplates {
			if backupConfigTemp.Spec.Target != nil {
				config.backupTarget = backupConfigTemp.Spec.Target
				config.runtimeSettings = backupConfigTemp.Spec.RuntimeSettings
				err = c.ensureVolumeSnapshotterJob(config, backupSession)
				if err != nil {
					return c.handleBackupJobCreationFailure(backupSession, err)
				}
				backupSession, err = c.setBackupSessionRunning(backupConfigTemp.Spec.Target, backupBatch.Spec.Driver, backupSession)
				if err != nil {
					return err
				}
			}
		}
		return nil
	}

	// Restic driver has been used. Now, create a backup job
	for i, backupConfigTemp := range backupBatch.Spec.BackupConfigurationTemplates {
		if backupConfigTemp.Spec.Target != nil && util.BackupModel(backupConfigTemp.Spec.Target.Ref.Kind) != util.ModelSidecar {
			config.runtimeSettings = backupConfigTemp.Spec.RuntimeSettings
			config.backupTarget = backupConfigTemp.Spec.Target
			config.interimVolumeTemplate = backupConfigTemp.Spec.InterimVolumeTemplate
			config.tempDir = backupConfigTemp.Spec.TempDir
			config.taskRef = backupConfigTemp.Spec.Task
			err = c.ensureBackupJob(config, backupSession, i)
			if err != nil {
				// failed to ensure backup job. set BackupSession phase "Failed" and send failure metrics.
				return c.handleBackupJobCreationFailure(backupSession, err)
			}
			// Backup job has been created successfully. Set BackupSession phase "Running"
			backupSession, err = c.setBackupSessionRunning(backupConfigTemp.Spec.Target, backupBatch.Spec.Driver, backupSession)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func (c *StashController) invokeBackupConfiguration(backupSession *api_v1beta1.BackupSession) error {
	backupConfig, err := c.stashClient.StashV1beta1().BackupConfigurations(backupSession.Namespace).Get(
		backupSession.Spec.Invoker.Name,
		metav1.GetOptions{},
	)
	if err != nil {
		return fmt.Errorf("can't get BackupConfiguration for BackupSession %s/%s, Reason: %s", backupSession.Namespace, backupSession.Name, err)
	}

	// skip if BackupConfiguration paused
	if backupConfig.Spec.Paused {
		log.Infof("Skipping processing BackupSession %s/%s. Reason: Backup Configuration is paused.", backupSession.Namespace, backupSession.Name)
		return c.setBackupSessionSkipped(backupSession, fmt.Sprintf("BackupConfiguration %s/%s is paused", backupConfig.Namespace, backupConfig.Name))
	}

	// skip if backup model is sidecar.
	// for sidecar model controller inside sidecar will take care of it.
	if backupConfig.Spec.Target != nil && backupConfig.Spec.Driver != api_v1beta1.VolumeSnapshotter && util.BackupModel(backupConfig.Spec.Target.Ref.Kind) == util.ModelSidecar {
		log.Infof("Skipping processing BackupSession %s/%s. Reason: Backup model is sidecar. Controller inside sidecar will take care of it.", backupSession.Namespace, backupSession.Name)
		_, err = c.setBackupSessionRunning(backupConfig.Spec.Target, backupConfig.Spec.Driver, backupSession)
		return err
	}

	// create BackupConfiguration object reference
	// and config for VolumeSnapshotter Job/backup Job
	ref, err := reference.GetReference(stash_scheme.Scheme, backupConfig)
	if err != nil {
		return err
	}
	config := backupOption{
		objMeta:         backupConfig.ObjectMeta,
		offshootLabels:  backupConfig.OffshootLabels(),
		ref:             ref,
		repositoryName:  backupConfig.Spec.Repository.Name,
		retentionPolicy: backupConfig.Spec.RetentionPolicy,
	}

	// if VolumeSnapshotter driver is used then ensure VolumeSnapshotter job
	if backupConfig.Spec.Target != nil && backupConfig.Spec.Driver == api_v1beta1.VolumeSnapshotter {
		config.backupTarget = backupConfig.Spec.Target
		config.runtimeSettings = backupConfig.Spec.RuntimeSettings
		err = c.ensureVolumeSnapshotterJob(config, backupSession)
		if err != nil {
			return c.handleBackupJobCreationFailure(backupSession, err)
		}
		// VolumeSnapshotter job has been created successfully. Set BackupSession phase "Running"
		_, err = c.setBackupSessionRunning(backupConfig.Spec.Target, backupConfig.Spec.Driver, backupSession)
		return err
	}

	// Restic driver has been used. Now, create a backup job
	config.runtimeSettings = backupConfig.Spec.RuntimeSettings
	config.backupTarget = backupConfig.Spec.Target
	config.interimVolumeTemplate = backupConfig.Spec.InterimVolumeTemplate
	config.tempDir = backupConfig.Spec.TempDir
	config.taskRef = backupConfig.Spec.Task
	err = c.ensureBackupJob(config, backupSession, 0)
	if err != nil {
		// failed to ensure backup job. set BackupSession phase "Failed" and send failure metrics.
		return c.handleBackupJobCreationFailure(backupSession, err)
	}
	// Backup job has been created successfully. Set BackupSession phase "Running"
	_, err = c.setBackupSessionRunning(backupConfig.Spec.Target, backupConfig.Spec.Driver, backupSession)
	return err
}

func (c *StashController) ensureBackupJob(config backupOption, backupSession *api_v1beta1.BackupSession, index int) error {
	jobMeta := metav1.ObjectMeta{
		Name:      getBackupJobName(backupSession, strconv.Itoa(index)),
		Namespace: config.objMeta.Namespace,
		Labels:    config.offshootLabels,
	}

	var serviceAccountName string

	// if RBAC is enabled then ensure respective RBAC stuffs
	if config.runtimeSettings.Pod != nil && config.runtimeSettings.Pod.ServiceAccountName != "" {
		serviceAccountName = config.runtimeSettings.Pod.ServiceAccountName
	} else {
		// ServiceAccount hasn't been specified. so create new one.
		serviceAccountName = getBackupJobServiceAccountName(config.ref.Name)
		saMeta := metav1.ObjectMeta{
			Name:      serviceAccountName,
			Namespace: config.objMeta.Namespace,
			Labels:    config.offshootLabels,
		}
		_, _, err := core_util.CreateOrPatchServiceAccount(c.kubeClient, saMeta, func(in *core.ServiceAccount) *core.ServiceAccount {
			core_util.EnsureOwnerReference(&in.ObjectMeta, config.ref)
			return in
		})
		if err != nil {
			return err
		}
	}

	psps, err := c.getBackupJobPSPNames(config.taskRef)
	if err != nil {
		return err
	}

	err = stash_rbac.EnsureBackupJobRBAC(c.kubeClient, config.ref, serviceAccountName, psps, config.offshootLabels)
	if err != nil {
		return err
	}

	// get repository for backupConfig
	repository, err := c.stashClient.StashV1alpha1().Repositories(config.objMeta.Namespace).Get(
		config.repositoryName,
		metav1.GetOptions{},
	)
	if err != nil {
		return err
	}

	// resolve task template

	explicitInputs := make(map[string]string)
	for _, param := range config.taskRef.Params {
		explicitInputs[param.Name] = param.Value
	}

	repoInputs, err := c.inputsForRepository(repository)
	if err != nil {
		return fmt.Errorf("cannot resolve implicit inputs for Repository %s/%s, reason: %s", repository.Namespace, repository.Name, err)
	}

	if backupSession.Spec.Invoker.Kind == api_v1beta1.ResourceKindBackupBatch {
		repoInputs[apis.RepositoryPrefix] = fmt.Sprintf("%s-%s-%s_%s", repoInputs[apis.RepositoryPrefix], config.objMeta.Namespace, config.backupTarget.Ref.Kind, config.backupTarget.Ref.Name)
	}

	bcInputs, err := c.inputsForBackupConfig(config)
	if err != nil {
		return fmt.Errorf("cannot resolve implicit inputs for BackupConfiguration %s/%s, reason: %s", config.ref.Namespace, config.ref.Name, err)
	}

	implicitInputs := core_util.UpsertMap(repoInputs, bcInputs)
	implicitInputs[apis.Namespace] = backupSession.Namespace
	implicitInputs[apis.BackupSession] = backupSession.Name

	taskResolver := resolve.TaskResolver{
		StashClient:     c.stashClient,
		TaskName:        config.taskRef.Name,
		Inputs:          core_util.UpsertMap(explicitInputs, implicitInputs), // TODO: reverse priority ???
		RuntimeSettings: config.runtimeSettings,
		TempDir:         config.tempDir,
	}
	podSpec, err := taskResolver.GetPodSpec()
	if err != nil {
		return fmt.Errorf("can't get PodSpec for BackupConfiguration %s/%s, reason: %s", config.objMeta.Namespace, config.objMeta.Name, err)
	}
	// for local backend, attach volume to all containers
	if repository.Spec.Backend.Local != nil {
		podSpec = util.AttachLocalBackend(podSpec, *repository.Spec.Backend.Local)
	}

	// upsert InterimVolume to hold the backup/restored data temporarily
	backupSessionRef, err := reference.GetReference(stash_scheme.Scheme, backupSession)
	if err != nil {
		return err
	}
	podSpec, err = util.UpsertInterimVolume(c.kubeClient, podSpec, config.interimVolumeTemplate, backupSessionRef)
	if err != nil {
		return err
	}

	// create Backup Job
	_, _, err = batch_util.CreateOrPatchJob(c.kubeClient, jobMeta, func(in *batchv1.Job) *batchv1.Job {
		// set BackupSession as owner of this Job
		core_util.EnsureOwnerReference(&in.ObjectMeta, backupSessionRef)

		in.Spec.Template.Spec = podSpec
		in.Spec.Template.Spec.ServiceAccountName = serviceAccountName
		return in
	})

	return err
}

func (c *StashController) ensureVolumeSnapshotterJob(config backupOption, backupSession *api_v1beta1.BackupSession) error {
	jobMeta := metav1.ObjectMeta{
		Name:      getVolumeSnapshotterJobName(config.backupTarget.Ref, backupSession.Name),
		Namespace: config.objMeta.Namespace,
		Labels:    config.offshootLabels,
	}

	//ensure respective RBAC stuffs
	//Create new ServiceAccount
	serviceAccountName := config.objMeta.Name
	saMeta := metav1.ObjectMeta{
		Name:      serviceAccountName,
		Namespace: config.objMeta.Namespace,
		Labels:    config.offshootLabels,
	}
	_, _, err := core_util.CreateOrPatchServiceAccount(c.kubeClient, saMeta, func(in *core.ServiceAccount) *core.ServiceAccount {
		core_util.EnsureOwnerReference(&in.ObjectMeta, config.ref)
		return in
	})
	if err != nil {
		return err
	}

	err = stash_rbac.EnsureVolumeSnapshotterJobRBAC(c.kubeClient, config.ref, serviceAccountName, config.offshootLabels)
	if err != nil {
		return err
	}

	image := docker.Docker{
		Registry: c.DockerRegistry,
		Image:    docker.ImageStash,
		Tag:      c.StashImageTag,
	}

	jobTemplate, err := util.NewVolumeSnapshotterJob(backupSession, config.backupTarget, config.runtimeSettings, image)
	if err != nil {
		return err
	}

	// Create VolumeSnapshotter job
	_, _, err = batch_util.CreateOrPatchJob(c.kubeClient, jobMeta, func(in *batchv1.Job) *batchv1.Job {
		// set BackupSession as owner of this Job
		core_util.EnsureOwnerReference(&in.ObjectMeta, config.ref)

		in.Labels = config.offshootLabels
		in.Spec.Template = *jobTemplate
		in.Spec.Template.Spec.ServiceAccountName = serviceAccountName
		return in
	})

	return err
}

func (c *StashController) setBackupSessionFailed(backupSession *api_v1beta1.BackupSession, backupErr error) error {

	// set BackupSession phase to "Failed"
	updatedBackupSession, err := stash_util.UpdateBackupSessionStatus(c.stashClient.StashV1beta1(), backupSession, func(in *api_v1beta1.BackupSessionStatus) *api_v1beta1.BackupSessionStatus {
		in.Phase = api_v1beta1.BackupSessionFailed
		in.Targets = backupSession.Status.Targets
		return in
	})
	if err != nil {
		return errors.NewAggregate([]error{backupErr, err})
	}

	// write failure event to BackupSession
	_, _ = eventer.CreateEvent(
		c.kubeClient,
		eventer.EventSourceBackupSessionController,
		backupSession,
		core.EventTypeWarning,
		eventer.EventReasonBackupSessionFailed,
		fmt.Sprintf("Backup session failed to complete. Reason: %v", backupErr),
	)

	var backupHistoryLimit *int32

	// send backup session specific metrics
	metricsOpt := &restic.MetricsOptions{
		Enabled:        true,
		PushgatewayURL: util.PushgatewayLocalURL,
		JobName:        PromJobBackupSessionController,
	}

	if backupSession.Spec.Invoker.Kind == api_v1beta1.ResourceKindBackupBatch {
		backupBatch, err := c.stashClient.StashV1beta1().BackupBatches(backupSession.Namespace).Get(backupSession.Spec.Invoker.Name, metav1.GetOptions{})
		if err != nil {
			return err
		}

		backupHistoryLimit = backupBatch.Spec.BackupHistoryLimit

		for _, backupConfigTemp := range backupBatch.Spec.BackupConfigurationTemplates {
			for _, target := range backupSession.Status.Targets {
				if backupConfigTemp.Spec.Target != nil && backupConfigTemp.Spec.Target.Ref.Kind == target.Ref.Kind &&
					backupConfigTemp.Spec.Target.Ref.Name == target.Ref.Name && target.Phase == api_v1beta1.TargetBackupFailed {
					err = metricsOpt.SendBackupSessionMetrics(c.clientConfig, backupSession.Namespace, backupConfigTemp.Spec.Target, backupBatch.Spec.Repository.Name, backupBatch.Spec.Driver, updatedBackupSession.Status)
					if err != nil {
						return err
					}
				}
			}
		}

	} else {
		backupConfig, err := c.stashClient.StashV1beta1().BackupConfigurations(backupSession.Namespace).Get(backupSession.Spec.Invoker.Name, metav1.GetOptions{})
		if err != nil {
			return errors.NewAggregate([]error{backupErr, err})
		}

		backupHistoryLimit = backupConfig.Spec.BackupHistoryLimit

		err = metricsOpt.SendBackupSessionMetrics(c.clientConfig, backupSession.Namespace, backupConfig.Spec.Target, backupConfig.Spec.Repository.Name, backupConfig.Spec.Driver, updatedBackupSession.Status)
		if err != nil {
			return errors.NewAggregate([]error{backupErr, err})
		}
	}

	// cleanup old BackupSessions
	err = c.cleanupBackupHistory(backupSession.Spec.Invoker, backupSession.Namespace, backupHistoryLimit)
	return errors.NewAggregate([]error{backupErr, err})
}

func (c *StashController) setBackupSessionSkipped(backupSession *api_v1beta1.BackupSession, reason string) error {
	// set BackupSession phase to "Skipped"
	_, err := stash_util.UpdateBackupSessionStatus(c.stashClient.StashV1beta1(), backupSession, func(in *api_v1beta1.BackupSessionStatus) *api_v1beta1.BackupSessionStatus {
		in.Phase = api_v1beta1.BackupSessionSkipped
		in.Targets = backupSession.Status.Targets
		return in
	})
	if err != nil {
		return err
	}

	// write skip event
	_, err = eventer.CreateEvent(
		c.kubeClient,
		eventer.EventSourceBackupSessionController,
		backupSession,
		core.EventTypeWarning,
		eventer.EventReasonBackupSessionSkipped,
		reason,
	)
	return err
}

func (c *StashController) setBackupSessionRunning(backupTarget *api_v1beta1.BackupTarget, driver api_v1beta1.Snapshotter, backupSession *api_v1beta1.BackupSession) (*api_v1beta1.BackupSession, error) {
	// find out the total number of hosts in target that will be backed up in this backup session
	totalHosts, err := c.getTotalHosts(backupTarget, backupSession.Namespace, driver)
	if err != nil {
		return nil, err
	}
	// set BackupSession phase to "Running"
	backupSession, err = stash_util.UpdateBackupSessionStatus(c.stashClient.StashV1beta1(), backupSession, func(in *api_v1beta1.BackupSessionStatus) *api_v1beta1.BackupSessionStatus {
		in.Phase = api_v1beta1.BackupSessionRunning
		if backupTarget != nil {
			in.Targets = append(backupSession.Status.Targets, api_v1beta1.Target{
				TotalHosts: totalHosts,
				Ref: api_v1beta1.TargetRef{
					Name: backupTarget.Ref.Name,
					Kind: backupTarget.Ref.Kind,
				},
				Phase: api_v1beta1.TargetBackupRunning,
			})
		}
		return in
	})
	if err != nil {
		return nil, err
	}

	// write event to the BackupSession
	_, err = eventer.CreateEvent(
		c.kubeClient,
		eventer.EventSourceBackupSessionController,
		backupSession,
		core.EventTypeNormal,
		eventer.EventReasonBackupSessionRunning,
		fmt.Sprintf("Backup job has been created succesfully/sidecar is watching the BackupSession."),
	)

	return backupSession, err
}

func (c *StashController) setBackupSessionSucceeded(backupSession *api_v1beta1.BackupSession) error {

	// total backup session duration is the difference between the time when BackupSession was created and current time
	sessionDuration := time.Since(backupSession.CreationTimestamp.Time)

	// set BackupSession phase "Succeeded"
	updatedBackupSession, err := stash_util.UpdateBackupSessionStatus(c.stashClient.StashV1beta1(), backupSession, func(in *api_v1beta1.BackupSessionStatus) *api_v1beta1.BackupSessionStatus {
		in.Phase = api_v1beta1.BackupSessionSucceeded
		in.SessionDuration = sessionDuration.String()
		in.Targets = backupSession.Status.Targets
		return in
	})
	if err != nil {
		return err
	}

	// write event to the BackupSession for successful backup
	_, err = eventer.CreateEvent(
		c.kubeClient,
		eventer.EventSourceBackupSessionController,
		backupSession,
		core.EventTypeNormal,
		eventer.EventReasonBackupSessionSucceeded,
		fmt.Sprintf("Backup session completed successfully"),
	)
	if err != nil {
		log.Errorf("failed to write event in BackupSession %s/%s. Reason: %v", backupSession.Namespace, backupSession.Name, err)
	}

	var backupHistoryLimit *int32

	// send backup session specific metrics
	metricsOpt := &restic.MetricsOptions{
		Enabled:        true,
		PushgatewayURL: util.PushgatewayLocalURL,
		JobName:        PromJobBackupSessionController,
	}

	if backupSession.Spec.Invoker.Kind == api_v1beta1.ResourceKindBackupBatch {
		backupBatch, err := c.stashClient.StashV1beta1().BackupBatches(backupSession.Namespace).Get(backupSession.Spec.Invoker.Name, metav1.GetOptions{})
		if err != nil {
			return err
		}

		backupHistoryLimit = backupBatch.Spec.BackupHistoryLimit

		for _, backupConfigTemp := range backupBatch.Spec.BackupConfigurationTemplates {
			if backupConfigTemp.Spec.Target != nil {
				err = metricsOpt.SendBackupSessionMetrics(c.clientConfig, backupSession.Namespace, backupConfigTemp.Spec.Target, backupBatch.Spec.Repository.Name, backupBatch.Spec.Driver, updatedBackupSession.Status)
				if err != nil {
					return err
				}
			}
		}

	} else {
		backupConfig, err := c.stashClient.StashV1beta1().BackupConfigurations(backupSession.Namespace).Get(backupSession.Spec.Invoker.Name, metav1.GetOptions{})
		if err != nil {
			return err
		}

		backupHistoryLimit = backupConfig.Spec.BackupHistoryLimit

		err = metricsOpt.SendBackupSessionMetrics(c.clientConfig, backupSession.Namespace, backupConfig.Spec.Target, backupConfig.Spec.Repository.Name, backupConfig.Spec.Driver, updatedBackupSession.Status)
		if err != nil {
			return err
		}
	}

	// cleanup old BackupSessions
	return c.cleanupBackupHistory(backupSession.Spec.Invoker, backupSession.Namespace, backupHistoryLimit)
}

func (c *StashController) getBackupSessionPhase(backupSession *api_v1beta1.BackupSession) (api_v1beta1.BackupSessionPhase, error) {
	// BackupSession phase is empty or "Pending" then return it. controller will process accordingly
	if backupSession.Status.Phase == "" ||
		backupSession.Status.Phase == api_v1beta1.BackupSessionPending {
		return api_v1beta1.BackupSessionPending, nil
	}

	// all target hasn't completed it's backup. BackupSession phase must be "Running".
	updatedTargetPhase := 0
	for _, target := range backupSession.Status.Targets {
		if target.Phase == api_v1beta1.TargetBackupSucceeded ||
			target.Phase == api_v1beta1.TargetBackupFailed {
			updatedTargetPhase = updatedTargetPhase + 1
		}
	}

	if updatedTargetPhase != len(backupSession.Status.Targets) {
		return api_v1beta1.BackupSessionRunning, nil
	}

	// check if any of the host has failed to take backup. if any of them has failed, then consider entire backup session as a failure.
	var errList []error
	for _, target := range backupSession.Status.Targets {
		if target.Phase == api_v1beta1.TargetBackupFailed {
			errList = append(errList, fmt.Errorf("backup failed for target: %s/%s", target.Ref.Kind, target.Ref.Name))
		}
	}

	if errList != nil {
		return api_v1beta1.BackupSessionFailed, errors.NewAggregate(errList)
	}

	// backup has been completed successfully
	return api_v1beta1.BackupSessionSucceeded, nil
}

func (c *StashController) handleBackupJobCreationFailure(backupSession *api_v1beta1.BackupSession, err error) error {
	log.Warningln("failed to ensure backup job. Reason: ", err)

	// write event to BackupSession
	_, _ = eventer.CreateEvent(
		c.kubeClient,
		eventer.EventSourceBackupSessionController,
		backupSession,
		core.EventTypeWarning,
		eventer.EventReasonBackupJobCreationFailed,
		fmt.Sprintf("failed to create backup job for BackupSession %s/%s. Reason: %v", backupSession.Namespace, backupSession.Name, err),
	)

	// set BackupSession phase failed
	return c.setBackupSessionFailed(backupSession, err)
}

func getBackupJobName(backupSession *api_v1beta1.BackupSession, index string) string {
	return meta.ValidNameWithPefixNSuffix(BackupJobPrefix, strings.ReplaceAll(backupSession.Name, ".", "-"), index)
}

func getBackupJobServiceAccountName(name string) string {
	return strings.ReplaceAll(name, ".", "-")
}

func getVolumeSnapshotterJobName(targetRef api_v1beta1.TargetRef, name string) string {
	parts := strings.Split(name, "-")
	lastPart := parts[len(parts)-1]
	return meta.ValidNameWithPrefix(VolumeSnapshotPrefix, fmt.Sprintf("%s-%s-%s", util.ResourceKindShortForm(targetRef.Kind), targetRef.Name, lastPart))
}

// cleanupBackupHistory deletes old BackupSessions and theirs associate resources according to BackupHistoryLimit
func (c *StashController) cleanupBackupHistory(backupInvokerRef api_v1beta1.BackupInvokerRef, namespace string, backupHistoryLimit *int32) error {
	// default history limit is 1
	historyLimit := int32(1)
	if backupHistoryLimit != nil {
		historyLimit = *backupHistoryLimit
	}

	// BackupSession use BackupConfiguration name as label. We can use this label as selector to list only the BackupSession
	// of this particular BackupConfiguration.
	label := metav1.LabelSelector{
		MatchLabels: map[string]string{
			util.LabelInvokerType: backupInvokerRef.Kind,
			util.LabelInvokerName: backupInvokerRef.Name,
		},
	}
	selector, err := metav1.LabelSelectorAsSelector(&label)
	if err != nil {
		return err
	}

	// list all the BackupSessions of this particular BackupConfiguration
	bsList, err := c.backupSessionLister.BackupSessions(namespace).List(selector)
	if err != nil {
		return err
	}

	// sort BackupSession according to creation timestamp. keep latest BackupSession first.
	sort.Slice(bsList, func(i, j int) bool {
		return bsList[i].CreationTimestamp.After(bsList[j].CreationTimestamp.Time)
	})

	// delete the BackupSession that does not fit within the history limit
	for i := int(historyLimit); i < len(bsList); i++ {
		err = c.stashClient.StashV1beta1().BackupSessions(namespace).Delete(bsList[i].Name, meta.DeleteInBackground())
		if err != nil && !(kerr.IsNotFound(err) || kerr.IsGone(err)) {
			return err
		}
	}
	return nil
}

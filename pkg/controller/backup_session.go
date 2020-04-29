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
	"os"
	"sort"
	"strconv"
	"strings"
	"time"

	"stash.appscode.dev/apimachinery/apis"
	"stash.appscode.dev/apimachinery/apis/stash"
	api_v1beta1 "stash.appscode.dev/apimachinery/apis/stash/v1beta1"
	stash_util "stash.appscode.dev/apimachinery/client/clientset/versioned/typed/stash/v1beta1/util"
	"stash.appscode.dev/apimachinery/pkg/docker"
	"stash.appscode.dev/apimachinery/pkg/restic"
	"stash.appscode.dev/stash/pkg/eventer"
	stash_rbac "stash.appscode.dev/stash/pkg/rbac"
	"stash.appscode.dev/stash/pkg/resolve"
	"stash.appscode.dev/stash/pkg/util"

	"github.com/appscode/go/log"
	"github.com/appscode/go/types"
	"github.com/golang/glog"
	batchv1 "k8s.io/api/batch/v1"
	core "k8s.io/api/core/v1"
	kerr "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/errors"
	"k8s.io/client-go/tools/clientcmd/api"
	batch_util "kmodules.xyz/client-go/batch/v1"
	core_util "kmodules.xyz/client-go/core/v1"
	"kmodules.xyz/client-go/meta"
	"kmodules.xyz/client-go/tools/queue"
	"kmodules.xyz/webhook-runtime/admission"
	hooks "kmodules.xyz/webhook-runtime/admission/v1beta1"
	webhook "kmodules.xyz/webhook-runtime/admission/v1beta1/generic"
)

const (
	BackupExecutorSidecar   = "sidecar"
	BackupExecutorCSIDriver = "csi-driver"
	BackupExecutorJob       = "job"
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

	// backup process for this BackupSession has not started. so let's start backup process
	// get backup Invoker
	invoker, err := apis.ExtractBackupInvokerInfo(c.stashClient, backupSession.Spec.Invoker.Kind, backupSession.Spec.Invoker.Name, backupSession.Namespace)
	if err != nil {
		return err
	}

	// check whether backup session is completed or running and set it's phase accordingly
	phase, err := c.getBackupSessionPhase(backupSession)

	// if backup process completed ( Failed or Succeeded), execute postBackup hook
	if (phase == api_v1beta1.BackupSessionFailed || phase == api_v1beta1.BackupSessionSucceeded) &&
		invoker.Hooks != nil && invoker.Hooks.PostBackup != nil {
		err = util.ExecuteHook(c.clientConfig, invoker.Hooks, apis.PostBackupHook, os.Getenv("MY_POD_NAME"), os.Getenv("MY_POD_NAMESPACE"))
		if err != nil {
			return c.setBackupSessionFailed(invoker, backupSession, err)
		}
	}

	if phase == api_v1beta1.BackupSessionFailed {
		// one or more hosts has failed to complete their backup process.
		// mark entire backup session as failure.
		// individual hosts has updated their respective stats and has sent respective metrics.
		// now, just set BackupSession phase "Failed" and create an event.
		return c.setBackupSessionFailed(invoker, backupSession, err)
	} else if phase == api_v1beta1.BackupSessionSucceeded {
		// all hosts has completed their backup process successfully.
		// individual hosts has updated their respective stats and has sent respective metrics.
		// now, just set BackupSession phase "Succeeded" and create an event.
		return c.setBackupSessionSucceeded(invoker, backupSession)
	} else if phase == api_v1beta1.BackupSessionRunning {
		log.Infof("Skipping processing BackupSession %s/%s. Reason: phase is %q.", backupSession.Namespace, backupSession.Name, backupSession.Status.Phase)
		return nil
	}

	// if preBackup hook exist, then execute preBackupHook
	if invoker.Hooks != nil && invoker.Hooks.PreBackup != nil {
		err = util.ExecuteHook(c.clientConfig, invoker.Hooks, apis.PreBackupHook, os.Getenv("MY_POD_NAME"), os.Getenv("MY_POD_NAMESPACE"))
		if err != nil {
			return c.setBackupSessionFailed(invoker, backupSession, err)
		}
	}

	for i, targetInfo := range invoker.TargetsInfo {
		if targetInfo.Target != nil {
			switch backupExecutor(invoker, targetInfo.Target.Ref) {
			case BackupExecutorSidecar:
				// Backup model is sidecar. For sidecar model, controller inside sidecar will take care of it.
				log.Infof("Skipping processing BackupSession %s/%s for target %s %s/%s. Reason: Backup model is sidecar."+
					"Controller inside sidecar will take care of it.",
					backupSession.Namespace,
					backupSession.Name,
					targetInfo.Target.Ref.Kind,
					backupSession.Namespace,
					targetInfo.Target.Ref.Name,
				)
			case BackupExecutorCSIDriver:
				// VolumeSnapshotter driver has been used. So, ensure VolumeSnapshotter job
				err = c.ensureVolumeSnapshotterJob(invoker, targetInfo, backupSession, i)
				if err != nil {
					return c.handleBackupJobCreationFailure(invoker, backupSession, err)
				}
			case BackupExecutorJob:
				err = c.ensureBackupJob(invoker, targetInfo, backupSession, i)
				if err != nil {
					// failed to ensure backup job. set BackupSession phase "Failed" and send failure metrics.
					return c.handleBackupJobCreationFailure(invoker, backupSession, err)
				}
			default:
				return fmt.Errorf("unable to identify backup executor entity")
			}

			// Set BackupSession phase "Running"
			backupSession, err = c.setTargetPhaseRunning(targetInfo.Target, invoker.Driver, backupSession)
			if err != nil {
				return err
			}
		}
	}
	_, err = c.setBackupSessionRunning(backupSession)
	return err
}

func (c *StashController) ensureBackupJob(invoker apis.Invoker, targetInfo apis.TargetInfo, backupSession *api_v1beta1.BackupSession, index int) error {
	jobMeta := metav1.ObjectMeta{
		Name:      getBackupJobName(backupSession, strconv.Itoa(index)),
		Namespace: backupSession.Namespace,
		Labels:    invoker.Labels,
	}

	var serviceAccountName string

	// if RBAC is enabled then ensure respective RBAC stuffs
	if targetInfo.RuntimeSettings.Pod != nil && targetInfo.RuntimeSettings.Pod.ServiceAccountName != "" {
		serviceAccountName = targetInfo.RuntimeSettings.Pod.ServiceAccountName
	} else {
		// ServiceAccount hasn't been specified. so create new one.
		serviceAccountName = getBackupJobServiceAccountName(invoker.ObjectMeta.Name, strconv.Itoa(index))
		saMeta := metav1.ObjectMeta{
			Name:      serviceAccountName,
			Namespace: invoker.ObjectMeta.Namespace,
			Labels:    invoker.Labels,
		}
		_, _, err := core_util.CreateOrPatchServiceAccount(c.kubeClient, saMeta, func(in *core.ServiceAccount) *core.ServiceAccount {
			core_util.EnsureOwnerReference(&in.ObjectMeta, invoker.OwnerRef)
			return in
		})
		if err != nil {
			return err
		}
	}

	psps, err := c.getBackupJobPSPNames(targetInfo.Task)
	if err != nil {
		return err
	}

	err = stash_rbac.EnsureBackupJobRBAC(c.kubeClient, invoker.OwnerRef, invoker.ObjectMeta.Namespace, serviceAccountName, psps, invoker.Labels)
	if err != nil {
		return err
	}

	// get repository for backupConfig
	repository, err := c.stashClient.StashV1alpha1().Repositories(invoker.ObjectMeta.Namespace).Get(invoker.Repository, metav1.GetOptions{})
	if err != nil {
		return err
	}

	// resolve task template
	explicitInputs := make(map[string]string)
	for _, param := range targetInfo.Task.Params {
		explicitInputs[param.Name] = param.Value
	}

	repoInputs, err := c.inputsForRepository(repository)
	if err != nil {
		return fmt.Errorf("cannot resolve implicit inputs for Repository %s/%s, reason: %s", repository.Namespace, repository.Name, err)
	}

	if backupSession.Spec.Invoker.Kind == api_v1beta1.ResourceKindBackupBatch {
		repoInputs[apis.RepositoryPrefix] = fmt.Sprintf("%s/%s/%s", repoInputs[apis.RepositoryPrefix], strings.ToLower(targetInfo.Target.Ref.Kind), targetInfo.Target.Ref.Name)
	}

	bcInputs, err := c.inputsForBackupConfig(invoker, targetInfo)
	if err != nil {
		return fmt.Errorf("cannot resolve implicit inputs for backup invoker  %s %s/%s, reason: %s", invoker.ObjectRef.Kind, invoker.ObjectRef.Namespace, invoker.ObjectRef.Name, err)
	}

	implicitInputs := core_util.UpsertMap(repoInputs, bcInputs)
	implicitInputs[apis.Namespace] = backupSession.Namespace
	implicitInputs[apis.BackupSession] = backupSession.Name

	// add docker image specific input
	implicitInputs[apis.StashDockerRegistry] = c.DockerRegistry
	implicitInputs[apis.StashDockerImage] = apis.ImageStash
	implicitInputs[apis.StashImageTag] = c.StashImageTag

	taskResolver := resolve.TaskResolver{
		StashClient:     c.stashClient,
		TaskName:        targetInfo.Task.Name,
		Inputs:          core_util.UpsertMap(explicitInputs, implicitInputs), // TODO: reverse priority ???
		RuntimeSettings: targetInfo.RuntimeSettings,
		TempDir:         targetInfo.TempDir,
	}

	// if preBackup or postBackup Hook is specified, add their specific inputs
	if targetInfo.Hooks != nil && targetInfo.Hooks.PreBackup != nil {
		taskResolver.PreTaskHookInput = make(map[string]string)
		taskResolver.PreTaskHookInput[apis.HookType] = apis.PreBackupHook
	}
	if targetInfo.Hooks != nil && targetInfo.Hooks.PostBackup != nil {
		taskResolver.PostTaskHookInput = make(map[string]string)
		taskResolver.PostTaskHookInput[apis.HookType] = apis.PostBackupHook
	}

	podSpec, err := taskResolver.GetPodSpec(invoker.ObjectRef.Kind, invoker.ObjectRef.Name, targetInfo.Target.Ref.Kind, targetInfo.Target.Ref.Name)
	if err != nil {
		return fmt.Errorf("can't get PodSpec for backup invoker %s/%s, reason: %s", invoker.ObjectMeta.Namespace, invoker.ObjectMeta.Name, err)
	}
	// for local backend, attach volume to all containers
	if repository.Spec.Backend.Local != nil {
		podSpec = util.AttachLocalBackend(podSpec, *repository.Spec.Backend.Local)
	}

	ownerBackupSession := metav1.NewControllerRef(backupSession, api_v1beta1.SchemeGroupVersion.WithKind(api_v1beta1.ResourceKindBackupSession))

	// upsert InterimVolume to hold the backup/restored data temporarily
	podSpec, err = util.UpsertInterimVolume(c.kubeClient, podSpec, targetInfo.InterimVolumeTemplate.ToCorePVC(), invoker.ObjectMeta.Namespace, ownerBackupSession)
	if err != nil {
		return err
	}

	// create Backup Job
	_, _, err = batch_util.CreateOrPatchJob(c.kubeClient, jobMeta, func(in *batchv1.Job) *batchv1.Job {
		// set BackupSession as owner of this Job so that the it get cleaned automatically
		// when the BackupSession gets deleted according to backupHistoryLimit
		core_util.EnsureOwnerReference(&in.ObjectMeta, ownerBackupSession)

		in.Spec.Template.Spec = podSpec
		in.Spec.Template.Spec.ServiceAccountName = serviceAccountName
		in.Spec.BackoffLimit = types.Int32P(1)
		return in
	})

	return err
}

func (c *StashController) ensureVolumeSnapshotterJob(invoker apis.Invoker, targetInfo apis.TargetInfo, backupSession *api_v1beta1.BackupSession, index int) error {
	jobMeta := metav1.ObjectMeta{
		Name:      getVolumeSnapshotterJobName(targetInfo.Target.Ref, backupSession.Name),
		Namespace: backupSession.Namespace,
		Labels:    invoker.Labels,
	}

	var serviceAccountName string
	// Ensure respective RBAC stuffs
	if targetInfo.RuntimeSettings.Pod != nil && targetInfo.RuntimeSettings.Pod.ServiceAccountName != "" {
		serviceAccountName = targetInfo.RuntimeSettings.Pod.ServiceAccountName
	} else {
		// Create new ServiceAccount
		serviceAccountName = getVolumeSnapshotterServiceAccountName(invoker.ObjectMeta.Name, strconv.Itoa(index))
		saMeta := metav1.ObjectMeta{
			Name:      serviceAccountName,
			Namespace: invoker.ObjectMeta.Namespace,
			Labels:    invoker.Labels,
		}
		_, _, err := core_util.CreateOrPatchServiceAccount(c.kubeClient, saMeta, func(in *core.ServiceAccount) *core.ServiceAccount {
			core_util.EnsureOwnerReference(&in.ObjectMeta, invoker.OwnerRef)
			return in
		})
		if err != nil {
			return err
		}
	}
	err := stash_rbac.EnsureVolumeSnapshotterJobRBAC(c.kubeClient, invoker.OwnerRef, invoker.ObjectMeta.Namespace, serviceAccountName, invoker.Labels)
	if err != nil {
		return err
	}

	image := docker.Docker{
		Registry: c.DockerRegistry,
		Image:    docker.ImageStash,
		Tag:      c.StashImageTag,
	}

	jobTemplate, err := util.NewVolumeSnapshotterJob(backupSession, targetInfo.Target, targetInfo.RuntimeSettings, image)
	if err != nil {
		return err
	}

	ownerBackupSession := metav1.NewControllerRef(backupSession, api_v1beta1.SchemeGroupVersion.WithKind(api_v1beta1.ResourceKindBackupSession))
	// Create VolumeSnapshotter job
	_, _, err = batch_util.CreateOrPatchJob(c.kubeClient, jobMeta, func(in *batchv1.Job) *batchv1.Job {
		// set BackupSession as owner of this Job so that the it get cleaned automatically
		// when the BackupSession gets deleted according to backupHistoryLimit
		core_util.EnsureOwnerReference(&in.ObjectMeta, ownerBackupSession)

		in.Labels = invoker.Labels
		in.Spec.Template = *jobTemplate
		in.Spec.Template.Spec.ServiceAccountName = serviceAccountName

		in.Spec.BackoffLimit = types.Int32P(1)
		return in
	})

	return err
}

func (c *StashController) setBackupSessionFailed(invoker apis.Invoker, backupSession *api_v1beta1.BackupSession, backupErr error) error {

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

	// send backup session specific metrics
	metricsOpt := &restic.MetricsOptions{
		Enabled:        true,
		PushgatewayURL: apis.PushgatewayLocalURL,
		JobName:        apis.PromJobStashBackup,
	}

	err = metricsOpt.SendBackupSessionMetrics(c.clientConfig, invoker, updatedBackupSession.Status)
	if err != nil {
		return err
	}
	// cleanup old BackupSessions
	err = c.cleanupBackupHistory(backupSession.Spec.Invoker, backupSession.Namespace, invoker.BackupHistoryLimit)
	return errors.NewAggregate([]error{backupErr, err})
}

func (c *StashController) setTargetPhaseRunning(target *api_v1beta1.BackupTarget, driver api_v1beta1.Snapshotter, backupSession *api_v1beta1.BackupSession) (*api_v1beta1.BackupSession, error) {
	// find out the total number of hosts in target that will be backed up in this backup session
	totalHosts, err := c.getTotalHosts(target, backupSession.Namespace, driver)
	if err != nil {
		return nil, err
	}
	// set target phase to "Running"
	backupSession, err = stash_util.UpdateBackupSessionStatus(c.stashClient.StashV1beta1(), backupSession, func(in *api_v1beta1.BackupSessionStatus) *api_v1beta1.BackupSessionStatus {
		if target != nil {
			in.Targets = upsertTargetStatsEntry(backupSession.Status.Targets, api_v1beta1.Target{
				TotalHosts: totalHosts,
				Ref: api_v1beta1.TargetRef{
					Name: target.Ref.Name,
					Kind: target.Ref.Kind,
				},
				Phase: api_v1beta1.TargetBackupRunning,
			})
		}
		return in
	})
	return backupSession, err
}

func (c *StashController) setBackupSessionRunning(backupSession *api_v1beta1.BackupSession) (*api_v1beta1.BackupSession, error) {
	// set BackupSession phase to "Running"
	backupSession, err := stash_util.UpdateBackupSessionStatus(c.stashClient.StashV1beta1(), backupSession, func(in *api_v1beta1.BackupSessionStatus) *api_v1beta1.BackupSessionStatus {
		in.Phase = api_v1beta1.BackupSessionRunning
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
		"Backup job has been created succesfully/sidecar is watching the BackupSession.",
	)
	return backupSession, err
}

func (c *StashController) setBackupSessionSucceeded(invoker apis.Invoker, backupSession *api_v1beta1.BackupSession) error {

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
		"Backup session completed successfully",
	)
	if err != nil {
		log.Errorf("failed to write event in BackupSession %s/%s. Reason: %v", backupSession.Namespace, backupSession.Name, err)
	}

	// send backup session specific metrics
	metricsOpt := &restic.MetricsOptions{
		Enabled:        true,
		PushgatewayURL: apis.PushgatewayLocalURL,
		JobName:        apis.PromJobStashBackup,
	}

	err = metricsOpt.SendBackupSessionMetrics(c.clientConfig, invoker, updatedBackupSession.Status)
	if err != nil {
		return err
	}

	// cleanup old BackupSessions
	return c.cleanupBackupHistory(backupSession.Spec.Invoker, backupSession.Namespace, invoker.BackupHistoryLimit)
}

func (c *StashController) getBackupSessionPhase(backupSession *api_v1beta1.BackupSession) (api_v1beta1.BackupSessionPhase, error) {
	// BackupSession phase is empty or "Pending" then return it. controller will process accordingly
	if backupSession.Status.Phase == "" ||
		backupSession.Status.Phase == api_v1beta1.BackupSessionPending {
		return api_v1beta1.BackupSessionPending, nil
	}

	// all target hasn't completed it's backup. BackupSession phase must be "Running"
	// and check if any of the host has failed to take backup. if any of them has failed,
	// then consider entire backup session as a failure.
	completedTargets := 0
	var errList []error
	for _, target := range backupSession.Status.Targets {
		if target.Phase == api_v1beta1.TargetBackupSucceeded ||
			target.Phase == api_v1beta1.TargetBackupFailed {
			completedTargets = completedTargets + 1
		}
		if target.Phase == api_v1beta1.TargetBackupFailed {
			errList = append(errList, fmt.Errorf("backup failed for target: %s/%s", target.Ref.Kind, target.Ref.Name))
		}
	}

	if completedTargets != len(backupSession.Status.Targets) {
		return api_v1beta1.BackupSessionRunning, nil
	}

	if errList != nil {
		return api_v1beta1.BackupSessionFailed, errors.NewAggregate(errList)
	}

	// backup has been completed successfully
	return api_v1beta1.BackupSessionSucceeded, nil
}

func (c *StashController) handleBackupJobCreationFailure(invoker apis.Invoker, backupSession *api_v1beta1.BackupSession, err error) error {
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
	return c.setBackupSessionFailed(invoker, backupSession, err)
}

func getBackupJobName(backupSession *api_v1beta1.BackupSession, index string) string {
	return meta.ValidNameWithPefixNSuffix(apis.PrefixStashBackup, strings.ReplaceAll(backupSession.Name, ".", "-"), index)
}

func getBackupJobServiceAccountName(invokerName, index string) string {
	return meta.ValidNameWithPefixNSuffix(apis.PrefixStashBackup, strings.ReplaceAll(invokerName, ".", "-"), index)
}

func getVolumeSnapshotterJobName(targetRef api_v1beta1.TargetRef, name string) string {
	parts := strings.Split(name, "-")
	suffix := parts[len(parts)-1]
	return meta.ValidNameWithPrefix(apis.PrefixStashVolumeSnapshot, fmt.Sprintf("%s-%s-%s", util.ResourceKindShortForm(targetRef.Kind), targetRef.Name, suffix))
}

func getVolumeSnapshotterServiceAccountName(invokerName, index string) string {
	return meta.ValidNameWithPefixNSuffix(apis.PrefixStashVolumeSnapshot, strings.ReplaceAll(invokerName, ".", "-"), index)
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
			apis.LabelInvokerType: backupInvokerRef.Kind,
			apis.LabelInvokerName: backupInvokerRef.Name,
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

func upsertTargetStatsEntry(targetStats []api_v1beta1.Target, newEntry api_v1beta1.Target) []api_v1beta1.Target {
	// already exist, then just update
	for i := range targetStats {
		if targetStats[i].Ref.Kind == newEntry.Ref.Kind && targetStats[i].Ref.Name == newEntry.Ref.Name {
			targetStats[i] = newEntry
			return targetStats
		}
	}
	// target entry does not exist. add new entry
	targetStats = append(targetStats, newEntry)
	return targetStats
}

func backupExecutor(invoker apis.Invoker, tref api_v1beta1.TargetRef) string {
	if (invoker.Driver == "" || invoker.Driver == api_v1beta1.ResticSnapshotter) &&
		util.BackupModel(tref.Kind) == apis.ModelSidecar {
		return BackupExecutorSidecar
	} else if invoker.Driver == api_v1beta1.VolumeSnapshotter {
		return BackupExecutorCSIDriver
	} else {
		return BackupExecutorJob
	}
}

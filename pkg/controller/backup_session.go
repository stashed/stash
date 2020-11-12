/*
Copyright AppsCode Inc. and Contributors

Licensed under the AppsCode Community License 1.0.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    https://github.com/appscode/licenses/raw/1.0.0/AppsCode-Community-1.0.0.md

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
	"os"
	"sort"
	"strconv"
	"strings"
	"time"

	"stash.appscode.dev/apimachinery/apis"
	"stash.appscode.dev/apimachinery/apis/stash"
	api_v1beta1 "stash.appscode.dev/apimachinery/apis/stash/v1beta1"
	stash_util "stash.appscode.dev/apimachinery/client/clientset/versioned/typed/stash/v1beta1/util"
	"stash.appscode.dev/apimachinery/pkg/conditions"
	"stash.appscode.dev/apimachinery/pkg/docker"
	"stash.appscode.dev/apimachinery/pkg/invoker"
	"stash.appscode.dev/apimachinery/pkg/restic"
	"stash.appscode.dev/stash/pkg/eventer"
	stash_rbac "stash.appscode.dev/stash/pkg/rbac"
	"stash.appscode.dev/stash/pkg/resolve"
	"stash.appscode.dev/stash/pkg/util"

	"github.com/golang/glog"
	"gomodules.xyz/pointer"
	"gomodules.xyz/x/log"
	batchv1 "k8s.io/api/batch/v1"
	core "k8s.io/api/core/v1"
	kerr "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/errors"
	"k8s.io/client-go/tools/clientcmd/api"
	kmapi "kmodules.xyz/client-go/api/v1"
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
	// ================= Don't Process Completed/Skipped BackupSession  ===========================
	if backupSession.Status.Phase == api_v1beta1.BackupSessionFailed ||
		backupSession.Status.Phase == api_v1beta1.BackupSessionSucceeded ||
		backupSession.Status.Phase == api_v1beta1.BackupSessionSkipped {
		log.Infof("Skipping processing BackupSession %s/%s. Reason: phase is %q.",
			backupSession.Namespace,
			backupSession.Name,
			backupSession.Status.Phase,
		)
		return nil
	}

	// backup process for this BackupSession has not started. so let's start backup process
	// get backup Invoker
	inv, err := invoker.ExtractBackupInvokerInfo(c.stashClient, backupSession.Spec.Invoker.Kind, backupSession.Spec.Invoker.Name, backupSession.Namespace)
	if err != nil {
		return err
	}

	// ensure that target phases are up to date
	backupSession, err = c.ensureTargetPhases(backupSession)
	if err != nil {
		return err
	}

	// check whether backup session is completed or running and set it's phase accordingly
	phase, err := c.getBackupSessionPhase(backupSession)

	// if current BackupSession phase is "Pending" and there is already another BackupSession in `Running` state for this invoker,
	// then we should skip the current session
	if phase == api_v1beta1.BackupSessionPending {
		// check if there is another BackupSession in running state.
		runningBS, err := c.getRunningBackupSessionForInvoker(inv)
		if err != nil {
			return err
		}
		if runningBS != nil {
			log.Infof("Skipped taking new backup. Reason: Previous BackupSession: %s is %q.",
				runningBS.Name,
				runningBS.Status.Phase,
			)
			return c.setBackupSessionSkipped(inv, backupSession, runningBS)
		}
	}

	var condErr error

	// ==================== Execute Global PostBackup Hooks ===========================
	// if backup process completed ( Failed or Succeeded) and postBackup hook has not been executed yet, execute global postBackup hook
	if backupCompleted(phase) && !globalPostBackupHookExecuted(inv, backupSession) {
		hookErr := util.ExecuteHook(c.clientConfig, inv.Hooks, apis.PostBackupHook, os.Getenv("MY_POD_NAME"), os.Getenv("MY_POD_NAMESPACE"))
		if hookErr != nil {
			backupSession, condErr = conditions.SetGlobalPostBackupHookSucceededConditionToFalse(c.stashClient, backupSession, hookErr)
			return c.setBackupSessionFailed(inv, backupSession, errors.NewAggregate([]error{err, hookErr, condErr}))
		}
		backupSession, condErr = conditions.SetGlobalPostBackupHookSucceededConditionToTrue(c.stashClient, backupSession)
		if condErr != nil {
			return condErr
		}
	}

	// ==================== Set BackupSession Phase ======================================
	if phase == api_v1beta1.BackupSessionFailed {
		// one or more target has failed to complete their backup process.
		// mark entire backup session as failure.
		// now, set BackupSession phase "Failed", create an event, and send respective metrics.
		return c.setBackupSessionFailed(inv, backupSession, err)
	} else if phase == api_v1beta1.BackupSessionSucceeded {
		// all hosts has completed their backup process successfully.
		// now, just set BackupSession phase "Succeeded", create an event, and send respective metrics.
		return c.setBackupSessionSucceeded(inv, backupSession)
	}

	// ==================== Execute Global PreBackup Hook =====================
	// if global preBackup hook exist and not yet executed, then execute the preBackupHook
	if !globalPreBackupHookExecuted(inv, backupSession) {
		err = util.ExecuteHook(c.clientConfig, inv.Hooks, apis.PreBackupHook, os.Getenv("MY_POD_NAME"), os.Getenv("MY_POD_NAMESPACE"))
		if err != nil {
			backupSession, condErr = conditions.SetGlobalPreBackupHookSucceededConditionToFalse(c.stashClient, backupSession, err)
			return c.setBackupSessionFailed(inv, backupSession, errors.NewAggregate([]error{err, condErr}))
		}
		backupSession, condErr = conditions.SetGlobalPreBackupHookSucceededConditionToTrue(c.stashClient, backupSession)
		if condErr != nil {
			return condErr
		}
	}

	// ===================== Run Backup for the Individual Targets ============================
	for i, targetInfo := range inv.TargetsInfo {
		if targetInfo.Target != nil {
			// Skip processing if the backup has been already initiated before for this target
			if invoker.TargetBackupInitiated(targetInfo.Target.Ref, backupSession.Status.Targets) {
				glog.Infof("Skipping initiating backup for %s %s/%s. Reason: Backup has been already initiated for this target.", targetInfo.Target.Ref.Kind, backupSession.ObjectMeta.Namespace, targetInfo.Target.Ref.Name)
				continue
			}
			// ----------------- Ensure Execution Order -------------------
			if inv.ExecutionOrder == api_v1beta1.Sequential &&
				!inv.NextInOrder(targetInfo.Target.Ref, backupSession.Status.Targets) {
				// backup order is sequential and the current target is not yet to be executed.
				// so, set its phase to "Pending".
				glog.Infof("Skipping initiating backup for %s %s/%s. Reason: Backup order is sequential and some previous targets hasn't completed their backup process.", targetInfo.Target.Ref.Kind, backupSession.ObjectMeta.Namespace, targetInfo.Target.Ref.Name)
				backupSession, err = c.setTargetPhasePending(targetInfo.Target.Ref, backupSession)
				if err != nil {
					return err
				}
				continue
			}
			// -------------- Ensure Backup Process for the Target ------------------
			switch backupExecutor(inv, targetInfo.Target.Ref) {
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
				err = c.ensureVolumeSnapshotterJob(inv, targetInfo, backupSession, i)
				if err != nil {
					return c.handleBackupJobCreationFailure(inv, backupSession, err)
				}
			case BackupExecutorJob:
				err = c.ensureBackupJob(inv, targetInfo, backupSession, i)
				if err != nil {
					// failed to ensure backup job. set BackupSession phase "Failed" and send failure metrics.
					return c.handleBackupJobCreationFailure(inv, backupSession, err)
				}
			default:
				return fmt.Errorf("unable to identify backup executor entity")
			}

			// Set target phase "Running"
			backupSession, err = c.setTargetPhaseRunning(inv, i, backupSession)
			if err != nil {
				return err
			}
		}
	}
	// Set BackupSession phase "Running"
	return c.setBackupSessionRunning(backupSession)
}

func (c *StashController) ensureBackupJob(inv invoker.BackupInvoker, targetInfo invoker.BackupTargetInfo, backupSession *api_v1beta1.BackupSession, index int) error {
	jobMeta := metav1.ObjectMeta{
		Name:      getBackupJobName(backupSession, strconv.Itoa(index)),
		Namespace: backupSession.Namespace,
		Labels:    inv.Labels,
	}

	var serviceAccountName string

	// if RBAC is enabled then ensure respective RBAC stuffs
	if targetInfo.RuntimeSettings.Pod != nil && targetInfo.RuntimeSettings.Pod.ServiceAccountName != "" {
		serviceAccountName = targetInfo.RuntimeSettings.Pod.ServiceAccountName
	} else {
		// ServiceAccount hasn't been specified. so create new one.
		serviceAccountName = getBackupJobServiceAccountName(inv.ObjectMeta.Name, strconv.Itoa(index))
		saMeta := metav1.ObjectMeta{
			Name:      serviceAccountName,
			Namespace: inv.ObjectMeta.Namespace,
			Labels:    inv.Labels,
		}
		_, _, err := core_util.CreateOrPatchServiceAccount(
			context.TODO(),
			c.kubeClient,
			saMeta,
			func(in *core.ServiceAccount) *core.ServiceAccount {
				core_util.EnsureOwnerReference(&in.ObjectMeta, inv.OwnerRef)
				return in
			},
			metav1.PatchOptions{},
		)
		if err != nil {
			return err
		}
	}

	psps, err := c.getBackupJobPSPNames(targetInfo.Task)
	if err != nil {
		return err
	}

	err = stash_rbac.EnsureBackupJobRBAC(c.kubeClient, inv.OwnerRef, inv.ObjectMeta.Namespace, serviceAccountName, psps, inv.Labels)
	if err != nil {
		return err
	}
	// Give the ServiceAccount permission to send request to the license handler
	err = stash_rbac.EnsureLicenseReaderClusterRoleBinding(c.kubeClient, inv.OwnerRef, inv.ObjectMeta.Namespace, serviceAccountName, inv.Labels)
	if err != nil {
		return err
	}

	// if the Stash is using a private registry, then ensure the image pull secrets
	var imagePullSecrets []core.LocalObjectReference
	if c.ImagePullSecrets != nil {
		imagePullSecrets, err = c.ensureImagePullSecrets(inv.ObjectMeta, inv.OwnerRef)
		if err != nil {
			return err
		}
	}

	// get repository for backupConfig
	repository, err := c.stashClient.StashV1alpha1().Repositories(inv.ObjectMeta.Namespace).Get(context.TODO(), inv.Repository, metav1.GetOptions{})
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

	bcInputs, err := c.inputsForBackupInvoker(inv, targetInfo)
	if err != nil {
		return fmt.Errorf("cannot resolve implicit inputs for backup invoker  %s %s/%s, reason: %s", inv.ObjectRef.Kind, inv.ObjectRef.Namespace, inv.ObjectRef.Name, err)
	}

	implicitInputs := core_util.UpsertMap(repoInputs, bcInputs)
	implicitInputs[apis.Namespace] = backupSession.Namespace
	implicitInputs[apis.BackupSession] = backupSession.Name

	// add docker image specific input
	implicitInputs[apis.StashDockerRegistry] = c.DockerRegistry
	implicitInputs[apis.StashDockerImage] = c.StashImage
	implicitInputs[apis.StashImageTag] = c.StashImageTag
	// license related inputs
	implicitInputs[apis.LicenseApiService] = c.LicenseApiService

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

	podSpec, err := taskResolver.GetPodSpec(inv.ObjectRef.Kind, inv.ObjectRef.Name, targetInfo.Target.Ref.Kind, targetInfo.Target.Ref.Name)
	if err != nil {
		return fmt.Errorf("can't get PodSpec for backup invoker %s/%s, reason: %s", inv.ObjectMeta.Namespace, inv.ObjectMeta.Name, err)
	}

	ownerBackupSession := metav1.NewControllerRef(backupSession, api_v1beta1.SchemeGroupVersion.WithKind(api_v1beta1.ResourceKindBackupSession))

	// upsert InterimVolume to hold the backup/restored data temporarily
	podSpec, err = util.UpsertInterimVolume(c.kubeClient, podSpec, targetInfo.InterimVolumeTemplate.ToCorePVC(), inv.ObjectMeta.Namespace, ownerBackupSession)
	if err != nil {
		return err
	}

	// create Backup Job
	_, _, err = batch_util.CreateOrPatchJob(
		context.TODO(),
		c.kubeClient,
		jobMeta,
		func(in *batchv1.Job) *batchv1.Job {
			// set BackupSession as owner of this Job so that the it get cleaned automatically
			// when the BackupSession gets deleted according to backupHistoryLimit
			core_util.EnsureOwnerReference(&in.ObjectMeta, ownerBackupSession)
			// pass offshoot labels to job's pod
			in.Spec.Template.Labels = core_util.UpsertMap(in.Spec.Template.Labels, inv.Labels)

			in.Spec.Template.Spec = podSpec
			in.Spec.Template.Spec.ImagePullSecrets = imagePullSecrets
			in.Spec.Template.Spec.ServiceAccountName = serviceAccountName
			in.Spec.BackoffLimit = pointer.Int32P(0)
			return in
		},
		metav1.PatchOptions{},
	)

	return err
}

func (c *StashController) ensureVolumeSnapshotterJob(inv invoker.BackupInvoker, targetInfo invoker.BackupTargetInfo, backupSession *api_v1beta1.BackupSession, index int) error {
	jobMeta := metav1.ObjectMeta{
		Name:      getVolumeSnapshotterJobName(targetInfo.Target.Ref, backupSession.Name),
		Namespace: backupSession.Namespace,
		Labels:    inv.Labels,
	}

	var serviceAccountName string
	// Ensure respective RBAC stuffs
	if targetInfo.RuntimeSettings.Pod != nil && targetInfo.RuntimeSettings.Pod.ServiceAccountName != "" {
		serviceAccountName = targetInfo.RuntimeSettings.Pod.ServiceAccountName
	} else {
		// Create new ServiceAccount
		serviceAccountName = getVolumeSnapshotterServiceAccountName(inv.ObjectMeta.Name, strconv.Itoa(index))
		saMeta := metav1.ObjectMeta{
			Name:      serviceAccountName,
			Namespace: inv.ObjectMeta.Namespace,
			Labels:    inv.Labels,
		}
		_, _, err := core_util.CreateOrPatchServiceAccount(
			context.TODO(),
			c.kubeClient,
			saMeta,
			func(in *core.ServiceAccount) *core.ServiceAccount {
				core_util.EnsureOwnerReference(&in.ObjectMeta, inv.OwnerRef)
				return in
			},
			metav1.PatchOptions{},
		)
		if err != nil {
			return err
		}
	}
	err := stash_rbac.EnsureVolumeSnapshotterJobRBAC(c.kubeClient, inv.OwnerRef, inv.ObjectMeta.Namespace, serviceAccountName, inv.Labels)
	if err != nil {
		return err
	}

	// if the Stash is using a private registry, then ensure the image pull secrets
	var imagePullSecrets []core.LocalObjectReference
	if c.ImagePullSecrets != nil {
		imagePullSecrets, err = c.ensureImagePullSecrets(inv.ObjectMeta, inv.OwnerRef)
		if err != nil {
			return err
		}
	}

	image := docker.Docker{
		Registry: c.DockerRegistry,
		Image:    c.StashImage,
		Tag:      c.StashImageTag,
	}

	jobTemplate, err := util.NewVolumeSnapshotterJob(backupSession, targetInfo.Target, targetInfo.RuntimeSettings, image)
	if err != nil {
		return err
	}

	ownerBackupSession := metav1.NewControllerRef(backupSession, api_v1beta1.SchemeGroupVersion.WithKind(api_v1beta1.ResourceKindBackupSession))
	// Create VolumeSnapshotter job
	_, _, err = batch_util.CreateOrPatchJob(
		context.TODO(),
		c.kubeClient,
		jobMeta,
		func(in *batchv1.Job) *batchv1.Job {
			// set BackupSession as owner of this Job so that the it get cleaned automatically
			// when the BackupSession gets deleted according to backupHistoryLimit
			core_util.EnsureOwnerReference(&in.ObjectMeta, ownerBackupSession)

			in.Labels = inv.Labels
			// pass offshoot labels to job's pod
			in.Spec.Template.Labels = core_util.UpsertMap(in.Spec.Template.Labels, inv.Labels)
			in.Spec.Template = *jobTemplate
			in.Spec.Template.Spec.ImagePullSecrets = imagePullSecrets
			in.Spec.Template.Spec.ServiceAccountName = serviceAccountName

			in.Spec.BackoffLimit = pointer.Int32P(0)
			return in
		},
		metav1.PatchOptions{},
	)

	return err
}

func (c *StashController) ensureTargetPhases(backupSession *api_v1beta1.BackupSession) (*api_v1beta1.BackupSession, error) {
	return stash_util.UpdateBackupSessionStatus(
		context.TODO(),
		c.stashClient.StashV1beta1(),
		backupSession.ObjectMeta,
		func(in *api_v1beta1.BackupSessionStatus) (types.UID, *api_v1beta1.BackupSessionStatus) {
			for i, target := range in.Targets {
				if target.TotalHosts == nil {
					in.Targets[i].Phase = api_v1beta1.TargetBackupPending
					continue
				}
				// if any host failed, then overall target phase should be failed.
				anyHostFailed := false
				for _, hostStats := range target.Stats {
					if hostStats.Phase == api_v1beta1.HostBackupFailed {
						anyHostFailed = true
						break
					}
				}
				if anyHostFailed {
					in.Targets[i].Phase = api_v1beta1.TargetBackupFailed
					continue
				}
				// if some host hasn't completed their backup yet, phase should be running
				if target.TotalHosts != nil && *target.TotalHosts != int32(len(target.Stats)) {
					in.Targets[i].Phase = api_v1beta1.TargetBackupRunning
					continue
				}
				// all host completed their backup and none of them failed. so, phase should be Succeeded.
				in.Targets[i].Phase = api_v1beta1.TargetBackupSucceeded
			}
			return backupSession.UID, in
		},
		metav1.UpdateOptions{},
	)
}

func (c *StashController) setTargetPhasePending(targetRef api_v1beta1.TargetRef, backupSession *api_v1beta1.BackupSession) (*api_v1beta1.BackupSession, error) {
	// set target phase to "Pending"
	backupSession, err := stash_util.UpdateBackupSessionStatus(
		context.TODO(),
		c.stashClient.StashV1beta1(),
		backupSession.ObjectMeta,
		func(in *api_v1beta1.BackupSessionStatus) (types.UID, *api_v1beta1.BackupSessionStatus) {
			in.Targets = upsertTargetStatsEntry(in.Targets, api_v1beta1.BackupTargetStatus{
				Ref:   targetRef,
				Phase: api_v1beta1.TargetBackupPending,
			})
			return backupSession.UID, in
		},
		metav1.UpdateOptions{},
	)
	return backupSession, err
}

func (c *StashController) setTargetPhaseRunning(inv invoker.BackupInvoker, index int, backupSession *api_v1beta1.BackupSession) (*api_v1beta1.BackupSession, error) {
	target := inv.TargetsInfo[index].Target
	// find out the total number of hosts in target that will be backed up in this backup session
	totalHosts, err := c.getTotalHosts(target, backupSession.Namespace, inv.Driver)
	if err != nil {
		return nil, err
	}
	// For Restic driver, set preBackupAction and postBackupAction
	var preBackupActions, postBackupActions []string
	if inv.Driver == api_v1beta1.ResticSnapshotter {
		// assign preBackupAction to the first target
		if index == 0 {
			preBackupActions = []string{apis.InitializeBackendRepository}
		}
		// assign postBackupAction to the last target
		if index == len(inv.TargetsInfo)-1 {
			postBackupActions = []string{apis.ApplyRetentionPolicy, apis.VerifyRepositoryIntegrity, apis.SendRepositoryMetrics}
		}
	}
	// set target phase to "Running"
	backupSession, err = stash_util.UpdateBackupSessionStatus(
		context.TODO(),
		c.stashClient.StashV1beta1(),
		backupSession.ObjectMeta,
		func(in *api_v1beta1.BackupSessionStatus) (types.UID, *api_v1beta1.BackupSessionStatus) {
			if target != nil {
				in.Targets = upsertTargetStatsEntry(in.Targets, api_v1beta1.BackupTargetStatus{
					TotalHosts:        totalHosts,
					Ref:               target.Ref,
					Phase:             api_v1beta1.TargetBackupRunning,
					PreBackupActions:  preBackupActions,
					PostBackupActions: postBackupActions,
				})
			}
			return backupSession.UID, in
		},
		metav1.UpdateOptions{},
	)
	return backupSession, err
}

func (c *StashController) setBackupSessionRunning(backupSession *api_v1beta1.BackupSession) error {
	// set BackupSession phase to "Running"
	backupSession, err := stash_util.UpdateBackupSessionStatus(
		context.TODO(),
		c.stashClient.StashV1beta1(),
		backupSession.ObjectMeta,
		func(in *api_v1beta1.BackupSessionStatus) (types.UID, *api_v1beta1.BackupSessionStatus) {
			in.Phase = api_v1beta1.BackupSessionRunning
			return backupSession.UID, in
		},
		metav1.UpdateOptions{},
	)
	if err != nil {
		return err
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
	return err
}

func (c *StashController) setBackupSessionSucceeded(inv invoker.BackupInvoker, backupSession *api_v1beta1.BackupSession) error {

	// total backup session duration is the difference between the time when BackupSession was created and current time
	sessionDuration := time.Since(backupSession.CreationTimestamp.Time)

	// set BackupSession phase "Succeeded"
	updatedBackupSession, err := stash_util.UpdateBackupSessionStatus(
		context.TODO(),
		c.stashClient.StashV1beta1(),
		backupSession.ObjectMeta,
		func(in *api_v1beta1.BackupSessionStatus) (types.UID, *api_v1beta1.BackupSessionStatus) {
			in.Phase = api_v1beta1.BackupSessionSucceeded
			in.SessionDuration = sessionDuration.String()
			return backupSession.UID, in
		},
		metav1.UpdateOptions{},
	)
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

	// send backup metrics
	metricsOpt := &restic.MetricsOptions{
		Enabled:        true,
		PushgatewayURL: apis.PushgatewayLocalURL,
		JobName:        fmt.Sprintf("%s-%s-%s", strings.ToLower(inv.TypeMeta.Kind), inv.ObjectMeta.Namespace, inv.ObjectMeta.Name),
	}

	// send backup session related metrics
	err = metricsOpt.SendBackupSessionMetrics(inv, updatedBackupSession.Status)
	if err != nil {
		return err
	}

	// send target related metrics
	for _, target := range backupSession.Status.Targets {
		metricErr := metricsOpt.SendBackupTargetMetrics(c.clientConfig, inv, target.Ref, backupSession.Status)
		if metricErr != nil {
			return metricErr
		}
	}

	// cleanup old BackupSessions
	return c.cleanupBackupHistory(backupSession.Spec.Invoker, backupSession.Namespace, inv.BackupHistoryLimit)
}

func (c *StashController) setBackupSessionFailed(inv invoker.BackupInvoker, backupSession *api_v1beta1.BackupSession, backupErr error) error {

	// set BackupSession phase to "Failed"
	updatedBackupSession, err := stash_util.UpdateBackupSessionStatus(
		context.TODO(),
		c.stashClient.StashV1beta1(),
		backupSession.ObjectMeta,
		func(in *api_v1beta1.BackupSessionStatus) (types.UID, *api_v1beta1.BackupSessionStatus) {
			in.Phase = api_v1beta1.BackupSessionFailed
			return backupSession.UID, in
		},
		metav1.UpdateOptions{},
	)
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

	// send metrics
	metricsOpt := &restic.MetricsOptions{
		Enabled:        true,
		PushgatewayURL: apis.PushgatewayLocalURL,
		JobName:        fmt.Sprintf("%s-%s-%s", strings.ToLower(inv.TypeMeta.Kind), inv.ObjectMeta.Namespace, inv.ObjectMeta.Name),
	}

	// send backup session related metrics
	err = metricsOpt.SendBackupSessionMetrics(inv, updatedBackupSession.Status)
	if err != nil {
		return err
	}
	// send target related metrics
	for _, target := range backupSession.Status.Targets {
		metricErr := metricsOpt.SendBackupTargetMetrics(c.clientConfig, inv, target.Ref, backupSession.Status)
		if metricErr != nil {
			return metricErr
		}
	}

	// cleanup old BackupSessions
	err = c.cleanupBackupHistory(backupSession.Spec.Invoker, backupSession.Namespace, inv.BackupHistoryLimit)
	return errors.NewAggregate([]error{backupErr, err})
}

func (c *StashController) setBackupSessionSkipped(inv invoker.BackupInvoker, currentBS, runningBS *api_v1beta1.BackupSession) error {

	// set BackupSession phase to "Skipped"
	_, statusErr := stash_util.UpdateBackupSessionStatus(
		context.TODO(),
		c.stashClient.StashV1beta1(),
		currentBS.ObjectMeta,
		func(in *api_v1beta1.BackupSessionStatus) (types.UID, *api_v1beta1.BackupSessionStatus) {
			in.Phase = api_v1beta1.BackupSessionSkipped
			return currentBS.UID, in
		},
		metav1.UpdateOptions{},
	)

	// write failure event to BackupSession
	_, _ = eventer.CreateEvent(
		c.kubeClient,
		eventer.EventSourceBackupSessionController,
		currentBS,
		core.EventTypeWarning,
		eventer.EventReasonBackupSessionSkipped,
		fmt.Sprintf("Skipped taking new backup. Reason: Previous BackupSession: %s is %q.",
			runningBS.Name,
			runningBS.Status.Phase,
		))

	// cleanup old BackupSessions
	err := c.cleanupBackupHistory(currentBS.Spec.Invoker, currentBS.Namespace, inv.BackupHistoryLimit)
	return errors.NewAggregate([]error{statusErr, err})
}

func (c *StashController) getBackupSessionPhase(backupSession *api_v1beta1.BackupSession) (api_v1beta1.BackupSessionPhase, error) {
	// BackupSession phase is empty or "Pending" then return it. controller will process accordingly
	if backupSession.Status.Phase == "" ||
		backupSession.Status.Phase == api_v1beta1.BackupSessionPending {
		return api_v1beta1.BackupSessionPending, nil
	}

	// If any of the target fail, then mark the entire backup process as "Failed".
	// Mark the entire backup process "Succeeded" only and if only the backup of all targets has succeeded.
	// Otherwise, mark the backup process as "Running".
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
	// If any of the postBackup action was not executed successfully, consider the whole backup process as failure
	if ok, reason := postBackupActionsSucceeded(backupSession.Status.Conditions); !ok {
		return api_v1beta1.BackupSessionFailed, fmt.Errorf(reason)
	}

	// Backup has been completed successfully for all targets.
	return api_v1beta1.BackupSessionSucceeded, nil
}

func (c *StashController) handleBackupJobCreationFailure(inv invoker.BackupInvoker, backupSession *api_v1beta1.BackupSession, err error) error {
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
	return c.setBackupSessionFailed(inv, backupSession, err)
}

func getBackupJobName(backupSession *api_v1beta1.BackupSession, index string) string {
	return meta.ValidNameWithPrefixNSuffix(apis.PrefixStashBackup, strings.ReplaceAll(backupSession.Name, ".", "-"), index)
}

func getBackupJobServiceAccountName(invokerName, index string) string {
	return meta.ValidNameWithPrefixNSuffix(apis.PrefixStashBackup, strings.ReplaceAll(invokerName, ".", "-"), index)
}

func getVolumeSnapshotterJobName(targetRef api_v1beta1.TargetRef, name string) string {
	parts := strings.Split(name, "-")
	suffix := parts[len(parts)-1]
	return meta.ValidNameWithPrefix(apis.PrefixStashVolumeSnapshot, fmt.Sprintf("%s-%s-%s", util.ResourceKindShortForm(targetRef.Kind), targetRef.Name, suffix))
}

func getVolumeSnapshotterServiceAccountName(invokerName, index string) string {
	return meta.ValidNameWithPrefixNSuffix(apis.PrefixStashVolumeSnapshot, strings.ReplaceAll(invokerName, ".", "-"), index)
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
		// delete only the BackupSessions that has completed its backup
		if backupCompleted(bsList[i].Status.Phase) {
			err = c.stashClient.StashV1beta1().BackupSessions(namespace).Delete(context.TODO(), bsList[i].Name, meta.DeleteInBackground())
			if err != nil && !(kerr.IsNotFound(err) || kerr.IsGone(err)) {
				return err
			}
		}
	}
	return nil
}

func upsertTargetStatsEntry(targetStats []api_v1beta1.BackupTargetStatus, newEntry api_v1beta1.BackupTargetStatus) []api_v1beta1.BackupTargetStatus {
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

func backupExecutor(inv invoker.BackupInvoker, tref api_v1beta1.TargetRef) string {
	if (inv.Driver == "" || inv.Driver == api_v1beta1.ResticSnapshotter) &&
		util.BackupModel(tref.Kind) == apis.ModelSidecar {
		return BackupExecutorSidecar
	}
	if inv.Driver == api_v1beta1.VolumeSnapshotter {
		return BackupExecutorCSIDriver
	}
	return BackupExecutorJob
}

func backupCompleted(phase api_v1beta1.BackupSessionPhase) bool {
	return phase == api_v1beta1.BackupSessionFailed ||
		phase == api_v1beta1.BackupSessionSucceeded ||
		phase == api_v1beta1.BackupSessionSkipped
}

func globalPostBackupHookExecuted(inv invoker.BackupInvoker, backupSession *api_v1beta1.BackupSession) bool {
	if inv.Hooks == nil || inv.Hooks.PostBackup == nil {
		return true
	}
	return kmapi.HasCondition(backupSession.Status.Conditions, apis.GlobalPostBackupHookSucceeded) &&
		kmapi.IsConditionTrue(backupSession.Status.Conditions, apis.GlobalPostBackupHookSucceeded)
}

func globalPreBackupHookExecuted(inv invoker.BackupInvoker, backupSession *api_v1beta1.BackupSession) bool {
	if inv.Hooks == nil || inv.Hooks.PreBackup == nil {
		return true
	}
	return kmapi.HasCondition(backupSession.Status.Conditions, apis.GlobalPreBackupHookSucceeded) &&
		kmapi.IsConditionTrue(backupSession.Status.Conditions, apis.GlobalPreBackupHookSucceeded)
}

func postBackupActionsSucceeded(conditions []kmapi.Condition) (bool, string) {
	// Check if the RetentionPolicy was applied properly
	if kmapi.HasCondition(conditions, apis.RetentionPolicyApplied) && !kmapi.IsConditionTrue(conditions, apis.RetentionPolicyApplied) {
		_, cond := kmapi.GetCondition(conditions, apis.RetentionPolicyApplied)
		if cond != nil {
			return false, cond.Message
		}
		return false, "Failed to apply retention policy"
	}

	// Check if the repo integrity was verified properly
	if kmapi.HasCondition(conditions, apis.RepositoryIntegrityVerified) && !kmapi.IsConditionTrue(conditions, apis.RepositoryIntegrityVerified) {
		_, cond := kmapi.GetCondition(conditions, apis.RepositoryIntegrityVerified)
		if cond != nil {
			return false, cond.Message
		}
		return false, "Failed to verify backend repository integrity"
	}

	// Check if the repository metrics was pushed properly
	if kmapi.HasCondition(conditions, apis.RepositoryMetricsPushed) && !kmapi.IsConditionTrue(conditions, apis.RepositoryMetricsPushed) {
		_, cond := kmapi.GetCondition(conditions, apis.RepositoryMetricsPushed)
		if cond != nil {
			return false, cond.Message
		}
		return false, "Failed to push repository metrics"
	}

	return true, ""
}

func (c *StashController) getRunningBackupSessionForInvoker(inv invoker.BackupInvoker) (*api_v1beta1.BackupSession, error) {
	backupSessions, err := c.backupSessionLister.List(labels.SelectorFromSet(map[string]string{
		apis.LabelInvokerName: inv.ObjectMeta.Name,
		apis.LabelInvokerType: inv.TypeMeta.Kind,
	}))
	if err != nil {
		return nil, err
	}
	for i := range backupSessions {
		if backupSessions[i].Status.Phase == api_v1beta1.BackupSessionRunning {
			return backupSessions[i], nil
		}
	}
	return nil, nil
}

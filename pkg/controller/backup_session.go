package controller

import (
	"fmt"
	"time"

	"github.com/appscode/go/log"
	"github.com/appscode/stash/apis"
	"github.com/appscode/stash/apis/stash"
	api_v1beta1 "github.com/appscode/stash/apis/stash/v1beta1"
	stash_scheme "github.com/appscode/stash/client/clientset/versioned/scheme"
	stash_util "github.com/appscode/stash/client/clientset/versioned/typed/stash/v1beta1/util"
	"github.com/appscode/stash/pkg/eventer"
	"github.com/appscode/stash/pkg/resolve"
	"github.com/appscode/stash/pkg/util"
	"github.com/golang/glog"
	batchv1 "k8s.io/api/batch/v1"
	core "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
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
	BackupJobPrefix = "stash-backup-"
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
	c.backupSessionInformer.AddEventHandler(queue.NewObservableHandler(c.backupSessionQueue.GetQueue(), apis.EnableStatusSubresource))
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

	if backupSession.Status.Phase == api_v1beta1.BackupSessionFailed ||
		backupSession.Status.Phase == api_v1beta1.BackupSessionSucceeded {
		log.Infof("Skipping processing BackupSession %s/%s. Reason: phase is %q.", backupSession.Namespace, backupSession.Name, backupSession.Status.Phase)
		return nil
	}

	// check weather backup session is completed or running and set it's phase accordingly
	phase, err := c.getBackupSessionPhase(backupSession)

	if phase == api_v1beta1.BackupSessionFailed {
		return c.setBackupSessionFailed(backupSession, err)
	} else if phase == api_v1beta1.BackupSessionSucceeded {
		return c.setBackupSessionSucceeded(backupSession)
	} else if phase == api_v1beta1.BackupSessionRunning {
		log.Infof("Skipping processing BackupSession %s/%s. Reason: phase is %q.", backupSession.Namespace, backupSession.Name, backupSession.Status.Phase)
		return nil
	} else if phase == api_v1beta1.BackupSessionSkipped {
		log.Infof("Skipping processing BackupSession %s/%s. Reason: previously skipped.", backupSession.Namespace, backupSession.Name)
		return nil
	}

	// backup process for this BackupSession has not started. so let's start backup process
	// get BackupConfiguration for BackupSession
	backupConfig, err := c.stashClient.StashV1beta1().BackupConfigurations(backupSession.Namespace).Get(
		backupSession.Spec.BackupConfiguration.Name,
		metav1.GetOptions{},
	)
	if err != nil {
		return fmt.Errorf("can't get BackupConfiguration for BackupSession %s/%s, Reason: %s", backupSession.Namespace, backupSession.Name, err)
	}

	// skip if BackupConfiguration paused
	if backupConfig.Spec.Paused {
		log.Infof("Skipping processing BackupSession %s/%s. Reason: Backup Configuration is paused.", backupSession.Namespace, backupSession.Name)
		return c.setBackupSessionSkipped(backupSession, "Backup Configuration is paused")
	}

	// skip if backup model is sidecar.
	// for sidecar model controller inside sidecar will take care of it.
	if backupConfig.Spec.Target != nil && util.BackupModel(backupConfig.Spec.Target.Ref.Kind) == util.ModelSidecar {
		log.Infof("Skipping processing BackupSession %s/%s. Reason: Backup model is sidecar. Controller inside sidecar will take care of it.", backupSession.Namespace, backupSession.Name)
		return c.setBackupSessionRunning(backupSession)
	}

	// create backup job
	err = c.ensureBackupJob(backupSession, backupConfig)
	if err != nil {
		return c.setBackupSessionFailed(backupSession, err)
	}

	// job has been created successfully. set BackupSession phase "Running"
	return c.setBackupSessionRunning(backupSession)
}

func (c *StashController) ensureBackupJob(backupSession *api_v1beta1.BackupSession, backupConfig *api_v1beta1.BackupConfiguration) error {

	jobMeta := metav1.ObjectMeta{
		Name:      BackupJobPrefix + backupSession.Name,
		Namespace: backupSession.Namespace,
	}

	backupConfigRef, err := reference.GetReference(stash_scheme.Scheme, backupConfig)
	if err != nil {
		return err
	}

	backupSessionRef, err := reference.GetReference(stash_scheme.Scheme, backupSession)
	if err != nil {
		return err
	}

	serviceAccountName := "default"

	// if RBAC is enabled then ensure respective RBAC stuffs
	if c.EnableRBAC {
		if backupConfig.Spec.RuntimeSettings.Pod != nil && backupConfig.Spec.RuntimeSettings.Pod.ServiceAccountName != "" {
			serviceAccountName = backupConfig.Spec.RuntimeSettings.Pod.ServiceAccountName
		} else {
			// ServiceAccount hasn't been specified. so create new one.
			serviceAccountName = backupConfig.Name
			saMeta := metav1.ObjectMeta{
				Name:      serviceAccountName,
				Namespace: backupConfig.Namespace,
			}
			_, _, err := core_util.CreateOrPatchServiceAccount(c.kubeClient, saMeta, func(in *core.ServiceAccount) *core.ServiceAccount {
				core_util.EnsureOwnerReference(&in.ObjectMeta, backupConfigRef)
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

		err := c.ensureBackupJobRBAC(backupConfigRef, serviceAccountName)
		if err != nil {
			return err
		}
	}

	// get repository for backupConfig
	repository, err := c.stashClient.StashV1alpha1().Repositories(backupConfig.Namespace).Get(
		backupConfig.Spec.Repository.Name,
		metav1.GetOptions{},
	)
	if err != nil {
		return err
	}

	// resolve task template

	explicitInputs := make(map[string]string)
	for _, param := range backupConfig.Spec.Task.Params {
		explicitInputs[param.Name] = param.Value
	}

	repoInputs, err := c.inputsForRepository(repository)
	if err != nil {
		return fmt.Errorf("cannot resolve implicit inputs for Repository %s/%s, reason: %s", repository.Namespace, repository.Name, err)
	}
	bcInputs, err := c.inputsForBackupConfig(*backupConfig)
	if err != nil {
		return fmt.Errorf("cannot resolve implicit inputs for BackupConfiguration %s/%s, reason: %s", backupConfig.Namespace, backupConfig.Name, err)
	}

	implicitInputs := core_util.UpsertMap(repoInputs, bcInputs)
	implicitInputs[apis.Namespace] = backupSession.Namespace
	implicitInputs[apis.BackupSession] = backupSession.Name
	implicitInputs[apis.StatusSubresourceEnabled] = fmt.Sprint(apis.EnableStatusSubresource)

	taskResolver := resolve.TaskResolver{
		StashClient:     c.stashClient,
		TaskName:        backupConfig.Spec.Task.Name,
		Inputs:          core_util.UpsertMap(explicitInputs, implicitInputs), // TODO: reverse priority ???
		RuntimeSettings: backupConfig.Spec.RuntimeSettings,
		TempDir:         backupConfig.Spec.TempDir,
	}
	podSpec, err := taskResolver.GetPodSpec()
	if err != nil {
		return fmt.Errorf("can't get PodSpec for BackupConfiguration %s/%s, reason: %s", backupConfig.Namespace, backupConfig.Name, err)
	}
	// for local backend, attach volume to all containers
	if repository.Spec.Backend.Local != nil {
		podSpec = util.AttachLocalBackend(podSpec, *repository.Spec.Backend.Local)
	}

	// create Backup Job
	_, _, err = batch_util.CreateOrPatchJob(c.kubeClient, jobMeta, func(in *batchv1.Job) *batchv1.Job {
		// set BackupSession as owner of this Job
		core_util.EnsureOwnerReference(&in.ObjectMeta, backupSessionRef)
		in.Labels = map[string]string{
			// job controller should not delete this job on completion
			// use a different label than v1alpha1 job labels to skip deletion from job controller
			// TODO: Remove job controller, cleanup backup-session periodically
			util.LabelApp: util.AppLabelStashV1Beta1,
		}
		in.Spec.Template.Spec = podSpec
		in.Spec.Template.Spec.ServiceAccountName = serviceAccountName

		return in
	})

	return err
}

func (c *StashController) setBackupSessionFailed(backupSession *api_v1beta1.BackupSession, jobErr error) error {

	// set BackupSession phase to "Failed"
	_, err := stash_util.UpdateBackupSessionStatus(c.stashClient.StashV1beta1(), backupSession, func(in *api_v1beta1.BackupSessionStatus) *api_v1beta1.BackupSessionStatus {
		in.Phase = api_v1beta1.BackupSessionFailed
		return in
	}, apis.EnableStatusSubresource)
	if err != nil {
		return err
	}

	// write failure event
	_, err = eventer.CreateEvent(
		c.kubeClient,
		eventer.BackupSessionEventComponent,
		backupSession,
		core.EventTypeWarning,
		eventer.EventReasonBackupSessionFailed,
		jobErr.Error(),
	)

	return err
}

func (c *StashController) setBackupSessionSkipped(backupSession *api_v1beta1.BackupSession, reason string) error {
	// set BackupSession phase to "Skipped"
	_, err := stash_util.UpdateBackupSessionStatus(c.stashClient.StashV1beta1(), backupSession, func(in *api_v1beta1.BackupSessionStatus) *api_v1beta1.BackupSessionStatus {
		in.Phase = api_v1beta1.BackupSessionSkipped
		return in
	}, apis.EnableStatusSubresource)
	if err != nil {
		return err
	}

	// write skip event
	_, err = eventer.CreateEvent(
		c.kubeClient,
		eventer.BackupSessionEventComponent,
		backupSession,
		core.EventTypeWarning,
		eventer.EventReasonBackupSessionSkipped,
		reason,
	)

	return err
}

func (c *StashController) setBackupSessionRunning(backupSession *api_v1beta1.BackupSession) error {

	backupConfig, err := c.stashClient.StashV1beta1().BackupConfigurations(backupSession.Namespace).Get(
		backupSession.Spec.BackupConfiguration.Name,
		metav1.GetOptions{},
	)
	if err != nil {
		return err
	}

	totalHosts, err := c.getTotalHosts(backupConfig.Spec.Target, backupConfig.Namespace)
	if err != nil {
		return err
	}

	// set BackupSession phase to "Running"
	_, err = stash_util.UpdateBackupSessionStatus(c.stashClient.StashV1beta1(), backupSession, func(in *api_v1beta1.BackupSessionStatus) *api_v1beta1.BackupSessionStatus {
		in.Phase = api_v1beta1.BackupSessionRunning
		in.TotalHosts = totalHosts
		return in
	}, apis.EnableStatusSubresource)
	if err != nil {
		return err
	}

	// write job creation success event
	_, err = eventer.CreateEvent(
		c.kubeClient,
		eventer.BackupSessionEventComponent,
		backupSession,
		core.EventTypeNormal,
		eventer.EventReasonBackupSessionJobCreated,
		fmt.Sprintf("backup job has been created succesfully for BackupSession %s/%s", backupSession.Namespace, backupSession.Name),
	)

	return err
}

func (c *StashController) setBackupSessionSucceeded(backupSession *api_v1beta1.BackupSession) error {

	// total backup session duration is sum of individual host backup duration
	var sessionDuration time.Duration
	for _, hostStats := range backupSession.Status.Stats {
		hostBackupDuration, err := time.ParseDuration(hostStats.Duration)
		if err != nil {
			return err
		}
		sessionDuration = sessionDuration + hostBackupDuration
	}

	// update BackupSession status
	_, err := stash_util.UpdateBackupSessionStatus(c.stashClient.StashV1beta1(), backupSession, func(in *api_v1beta1.BackupSessionStatus) *api_v1beta1.BackupSessionStatus {
		in.Phase = api_v1beta1.BackupSessionSucceeded
		in.SessionDuration = sessionDuration.String()
		return in
	}, apis.EnableStatusSubresource)
	if err != nil {
		return err
	}

	// write event for successful backup
	_, err = eventer.CreateEvent(
		c.kubeClient,
		eventer.BackupSessionEventComponent,
		backupSession,
		core.EventTypeNormal,
		eventer.EventReasonSuccessfulBackup,
		fmt.Sprintf("backup has been completed succesfully for BackupSession %s/%s", backupSession.Namespace, backupSession.Name),
	)

	return err
}

func (c *StashController) getBackupSessionPhase(backupSession *api_v1beta1.BackupSession) (api_v1beta1.BackupSessionPhase, error) {
	// BackupSession phase is empty or "Pending" then return it. controller will process accordingly
	if backupSession.Status.TotalHosts == nil ||
		backupSession.Status.Phase == "" ||
		backupSession.Status.Phase == api_v1beta1.BackupSessionPending {
		return api_v1beta1.BackupSessionPending, nil
	}

	// all hosts hasn't completed it's backup. BackupSession phase must be "Running".
	if *backupSession.Status.TotalHosts != int32(len(backupSession.Status.Stats)) {
		return api_v1beta1.BackupSessionRunning, nil
	}

	// check if any of the host has failed to take backup. if any of them has failed, then consider entire backup session as a failure.
	for _, host := range backupSession.Status.Stats {
		if host.Phase == api_v1beta1.HostBackupFailed {
			return api_v1beta1.BackupSessionFailed, fmt.Errorf("backup failed for host: %s. Reason: %s", host.Hostname, host.Error)
		}
	}

	// backup has been completed successfully
	return api_v1beta1.BackupSessionSucceeded, nil
}

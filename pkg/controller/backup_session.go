package controller

import (
	"fmt"

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
			Resource: api_v1beta1.ResourcePluralBackupSession,
		},
		api_v1beta1.ResourceSingularBackupSession,
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
	} else {
		backupSession := obj.(*api_v1beta1.BackupSession)
		glog.Infof("Sync/Add/Update for BackupSession %s", backupSession.GetName())

		// don't process further if the BackupSession already has been processed
		if !util.BackupPending(backupSession.Status.Phase) {
			log.Infof("Skipping processing BackupSession %s/%s. Reason: phase is %s.", backupSession.Namespace, backupSession.Name, backupSession.Status.Phase)
			return nil
		}

		// get BackupConfiguration for BackupSession
		backupConfig, err := c.stashClient.StashV1beta1().BackupConfigurations(backupSession.Namespace).Get(
			backupSession.Spec.BackupConfiguration.Name,
			metav1.GetOptions{},
		)
		if err != nil {
			return fmt.Errorf("can't get BackupConfiguration for BackupSession %s/%s, Reason: %s", backupSession.Namespace, backupSession.Name, err)
		}

		// skip if backup model is sidecar.
		// for sidecar model controller inside sidecar will take care of it.
		if backupConfig.Spec.Target != nil && util.BackupModel(backupConfig.Spec.Target.Ref.Kind) == util.ModelSidecar {
			log.Infof("Skipping processing BackupSession %s/%s. Reason: Backup model is sidecar. Controller inside sidecar will take care of it.", backupSession.Namespace, backupSession.Name)
			return nil
		}

		// create backup job
		err = c.ensureBackupJob(backupSession, backupConfig)
		if err != nil {
			return c.setBackupSessionFailed(backupSession, err)
		}

		// job has been created successfully. set BackupSession phase "Running"
		err = c.setBackupSessionRunning(backupSession)
		if err != nil {
			return err
		}
	}
	return nil
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

	// resolve task
	explicitInputs := make(map[string]string)
	for _, param := range backupConfig.Spec.Task.Params {
		explicitInputs[param.Name] = param.Value
	}

	implicitInputs, err := c.inputsForBackupConfig(*backupConfig)
	if err != nil {
		return fmt.Errorf("cannot resolve implicit inputs for BackupConfiguration %s/%s, reason: %s", backupConfig.Namespace, backupConfig.Name, err)
	}
	implicitInputs[apis.Namespace] = backupSession.Namespace
	implicitInputs[apis.BackupSession] = backupSession.Name
	implicitInputs[apis.StatusSubresourceEnabled] = fmt.Sprint(apis.EnableStatusSubresource)

	taskResolver := resolve.TaskResolver{
		StashClient:     c.stashClient,
		TaskName:        backupConfig.Spec.Task.Name,
		Inputs:          core_util.UpsertMap(explicitInputs, implicitInputs), // TODO: reverse priority ???
		RuntimeSettings: backupConfig.Spec.RuntimeSettings,
	}
	podSpec, err := taskResolver.GetPodSpec()
	if err != nil {
		return fmt.Errorf("can't get PodSpec for BackupConfiguration %s/%s, reason: %s", backupConfig.Namespace, backupConfig.Name, err)
	}

	// create Backup Job
	_, _, err = batch_util.CreateOrPatchJob(c.kubeClient, jobMeta, func(in *batchv1.Job) *batchv1.Job {
		// set BackupSession as owner of this Job
		core_util.EnsureOwnerReference(&in.ObjectMeta, backupSessionRef)
		in.Labels = map[string]string{
			// job controller should not delete this job on completion
			// use a different label than v1alpha1 job labels to skip deletion from job controller
			// TODO: Remove job controller, cleanup backup-session periodically
			util.LabelApp: util.AppLabelStash,
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

func (c *StashController) setBackupSessionRunning(backupSession *api_v1beta1.BackupSession) error {

	// set BackupSession phase to "Running"
	_, err := stash_util.UpdateBackupSessionStatus(c.stashClient.StashV1beta1(), backupSession, func(in *api_v1beta1.BackupSessionStatus) *api_v1beta1.BackupSessionStatus {
		in.Phase = api_v1beta1.BackupSessionRunning
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

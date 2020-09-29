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

package status

import (
	"context"
	"fmt"
	"path/filepath"
	"time"

	"stash.appscode.dev/apimachinery/apis"
	"stash.appscode.dev/apimachinery/apis/stash/v1alpha1"
	"stash.appscode.dev/apimachinery/apis/stash/v1beta1"
	cs "stash.appscode.dev/apimachinery/client/clientset/versioned"
	stash_util "stash.appscode.dev/apimachinery/client/clientset/versioned/typed/stash/v1alpha1/util"
	stash_util_v1beta1 "stash.appscode.dev/apimachinery/client/clientset/versioned/typed/stash/v1beta1/util"
	"stash.appscode.dev/apimachinery/pkg/conditions"
	"stash.appscode.dev/apimachinery/pkg/restic"
	"stash.appscode.dev/stash/pkg/eventer"

	"github.com/appscode/go/log"
	"github.com/appscode/go/util/errors"
	"github.com/appscode/go/wait"
	core "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	kmapi "kmodules.xyz/client-go/api/v1"
)

type UpdateStatusOptions struct {
	Config      *rest.Config
	KubeClient  kubernetes.Interface
	StashClient cs.Interface

	Namespace      string
	Repository     string
	BackupSession  string
	OutputDir      string
	OutputFileName string
	Metrics        restic.MetricsOptions
	TargetRef      v1beta1.TargetRef

	InvokerKind string
	InvokerName string

	SetupOpt restic.SetupOptions
}

func (o UpdateStatusOptions) UpdateBackupStatusFromFile() error {
	// read backup output from file
	log.Infof("Reading backup output from file: %s", filepath.Join(o.OutputDir, o.OutputFileName))
	backupOutput, err := restic.ReadBackupOutput(filepath.Join(o.OutputDir, o.OutputFileName))
	if err != nil {
		return err
	}
	backupSession, err := o.StashClient.StashV1beta1().BackupSessions(o.Namespace).Get(context.TODO(), o.BackupSession, metav1.GetOptions{})
	if err != nil {
		return err
	}
	// get backup Invoker
	invoker, err := apis.ExtractBackupInvokerInfo(o.StashClient, backupSession.Spec.Invoker.Kind, backupSession.Spec.Invoker.Name, backupSession.Namespace)
	if err != nil {
		return err
	}
	for _, targetInfo := range invoker.TargetsInfo {
		if targetInfo.Target != nil &&
			targetInfo.Target.Ref.Kind == o.TargetRef.Kind &&
			targetInfo.Target.Ref.Name == o.TargetRef.Name {
			return o.UpdatePostBackupStatus(backupOutput, invoker, targetInfo)
		}
	}
	return nil
}

func (o UpdateStatusOptions) UpdateRestoreStatusFromFile() error {
	// read restore output from file
	log.Infof("Reading restore output from file: %s", filepath.Join(o.OutputDir, o.OutputFileName))
	restoreOutput, err := restic.ReadRestoreOutput(filepath.Join(o.OutputDir, o.OutputFileName))
	if err != nil {
		return err
	}

	invoker, err := apis.ExtractRestoreInvokerInfo(o.KubeClient, o.StashClient, o.InvokerKind, o.InvokerName, o.Namespace)
	if err != nil {
		return err
	}
	for _, targetInfo := range invoker.TargetsInfo {
		if targetInfo.Target != nil &&
			targetInfo.Target.Ref.Kind == o.TargetRef.Kind &&
			targetInfo.Target.Ref.Name == o.TargetRef.Name {
			return o.UpdatePostRestoreStatus(restoreOutput, invoker, targetInfo)
		}
	}
	return nil
}

func (o UpdateStatusOptions) UpdatePostBackupStatus(backupOutput *restic.BackupOutput, invoker apis.Invoker, targetInfo apis.TargetInfo) error {
	log.Infof("Updating post backup status.......")
	if backupOutput == nil {
		return fmt.Errorf("invalid backup ouputput. Backup output must not be nil")
	}
	// get backup session, update status and create event
	backupSession, err := o.StashClient.StashV1beta1().BackupSessions(o.Namespace).Get(context.TODO(), o.BackupSession, metav1.GetOptions{})
	if err != nil {
		return err
	}

	// If the target has been assigned some post-backup actions, execute them
	// Only execute the postBackupActions if the backup of current host/hosts has succeeded
	if backupSucceeded(backupOutput) {
		err = o.executePostBackupActions(invoker, backupSession, targetInfo.Target.Ref)
		if err != nil {
			return err
		}
	}

	// add or update entry for each host in BackupSession status + create event
	backupSession, err = stash_util_v1beta1.UpdateBackupSessionStatus(context.TODO(), o.StashClient.StashV1beta1(), backupSession.ObjectMeta, func(in *v1beta1.BackupSessionStatus) *v1beta1.BackupSessionStatus {
		for i := range in.Targets {
			if apis.TargetMatched(in.Targets[i].Ref, targetInfo.Target.Ref) {
				in.Targets[i].Stats = stash_util_v1beta1.UpsertHost(in.Targets[i].Stats, backupOutput.BackupTargetStatus.Stats...)
			}
		}
		return in
	}, metav1.UpdateOptions{})
	if err != nil {
		return err
	}

	//overallBackupSucceeded := true
	for _, hostStats := range backupOutput.BackupTargetStatus.Stats {
		// create event to the BackupSession
		var eventType, eventReason, eventMessage string
		if hostStats.Error != "" {
			//overallBackupSucceeded = false
			eventType = core.EventTypeWarning
			eventReason = eventer.EventReasonHostBackupFailed
			eventMessage = fmt.Sprintf("backup failed for host %q of %q/%q. Reason: %s", hostStats.Hostname, o.TargetRef.Kind, o.TargetRef.Name, hostStats.Error)
		} else {
			eventType = core.EventTypeNormal
			eventReason = eventer.EventReasonHostBackupSucceded
			eventMessage = fmt.Sprintf("backup succeeded for host %s of %q/%q.", hostStats.Hostname, o.TargetRef.Kind, o.TargetRef.Name)
		}
		_, err = eventer.CreateEvent(
			o.KubeClient,
			eventer.EventSourceStatusUpdater,
			backupSession,
			eventType,
			eventReason,
			eventMessage,
		)
		if err != nil {
			log.Errorf("failed to create event for %s %s/%s. Reason: %v",
				invoker.TypeMeta.Kind,
				invoker.ObjectMeta.Namespace,
				invoker.ObjectMeta.Name,
				err,
			)
		}
	}

	// if metrics enabled then send backup host specific metrics to the Prometheus pushgateway
	if o.Metrics.Enabled && targetInfo.Target != nil {
		return o.Metrics.SendBackupHostMetrics(o.Config, invoker, targetInfo.Target.Ref, backupOutput)
	}
	return nil
}

func (o UpdateStatusOptions) UpdatePostRestoreStatus(restoreOutput *restic.RestoreOutput, invoker apis.RestoreInvoker, targetInfo apis.RestoreTargetInfo) error {
	if restoreOutput == nil {
		return fmt.Errorf("invalid restore output. Restore output must not be nil")
	}
	// add or update entry for each host in restore invoker status
	var err error
	log.Infof("Updating hosts status for restore target %s %s/%s.",
		targetInfo.Target.Ref.Kind,
		invoker.ObjectMeta.Namespace,
		targetInfo.Target.Ref.Name,
	)
	invoker.Status, err = invoker.UpdateRestoreInvokerStatus(apis.RestoreInvokerStatus{
		TargetStatus: []v1beta1.RestoreMemberStatus{
			{
				Ref:   targetInfo.Target.Ref,
				Stats: restoreOutput.RestoreTargetStatus.Stats,
			},
		},
	})
	if err != nil {
		return err
	}
	// create event for each hosts
	for _, hostStats := range restoreOutput.RestoreTargetStatus.Stats {
		if err != nil {
			return err
		}
		// create event to the RestoreSession
		var eventType, eventReason, eventMessage string
		if hostStats.Error != "" {
			eventType = core.EventTypeWarning
			eventReason = eventer.EventReasonHostRestoreFailed
			eventMessage = fmt.Sprintf("restore failed for host %q. Reason: %s", hostStats.Hostname, hostStats.Error)
		} else {
			eventType = core.EventTypeNormal
			eventReason = eventer.EventReasonHostRestoreSucceeded
			eventMessage = fmt.Sprintf("restore succeeded for host %q", hostStats.Hostname)
		}
		err = invoker.CreateEvent(eventType, eventer.EventSourceStatusUpdater, eventReason, eventMessage)
		if err != nil {
			log.Errorf("failed to create event for %s %s/%s. Reason: %v",
				invoker.TypeMeta.Kind,
				invoker.ObjectMeta.Namespace,
				invoker.ObjectMeta.Name,
				err,
			)
		}
	}
	// if metrics enabled then send metrics to the Prometheus pushgateway
	if o.Metrics.Enabled && targetInfo.Target != nil {
		return o.Metrics.SendRestoreHostMetrics(o.Config, invoker, targetInfo.Target.Ref, restoreOutput)
	}
	return nil
}

func (o UpdateStatusOptions) executePostBackupActions(invoker apis.Invoker, backupSession *v1beta1.BackupSession, curTarget v1beta1.TargetRef) error {
	var repoStats restic.RepositoryStats
	for _, targetStatus := range backupSession.Status.Targets {
		if apis.TargetMatched(targetStatus.Ref, curTarget) {
			// check if it has any post-backup action assigned to it
			if len(targetStatus.PostBackupActions) > 0 {
				// For StatefulSet and DaemonSet, only the last host will run these PostBackupActions
				if curTarget.Kind == apis.KindStatefulSet || curTarget.Kind == apis.KindDaemonSet {
					if len(targetStatus.Stats) != (int(*targetStatus.TotalHosts) - 1) {
						log.Infof("Skipping running PostBackupActions. Reason: Only the last host will execute the post backup actions for %s", curTarget.Kind)
						return nil
					}
				}
				// wait until all other targets/hosts has completed their backup.
				err := o.waitUntilOtherHostsCompleted(backupSession, curTarget)
				if err != nil {
					return fmt.Errorf("failed to execute PostBackupActions. Reason: %v ", err.Error())
				}

				// execute the post-backup actions
				for _, action := range targetStatus.PostBackupActions {
					switch action {
					case apis.ApplyRetentionPolicy:
						if !kmapi.HasCondition(backupSession.Status.Conditions, apis.RetentionPolicyApplied) {
							log.Infoln("Applying retention policy.....")
							w, err := restic.NewResticWrapper(o.SetupOpt)
							if err != nil {
								_, condErr := conditions.SetRetentionPolicyAppliedConditionToFalse(o.StashClient, backupSession, err)
								return errors.NewAggregate([]error{err, condErr})
							}
							res, err := w.ApplyRetentionPolicies(invoker.RetentionPolicy)
							if err != nil {
								log.Warningf("Failed to apply retention policy. Reason: %s", err.Error())
								_, condErr := conditions.SetRetentionPolicyAppliedConditionToFalse(o.StashClient, backupSession, err)
								return errors.NewAggregate([]error{err, condErr})
							}
							if res != nil {
								repoStats.SnapshotCount = res.SnapshotCount
								repoStats.SnapshotsRemovedOnLastCleanup = res.SnapshotsRemovedOnLastCleanup
							}
							_, err = conditions.SetRetentionPolicyAppliedConditionToTrue(o.StashClient, backupSession)
							if err != nil {
								return err
							}
						}
					case apis.VerifyRepositoryIntegrity:
						if !kmapi.HasCondition(backupSession.Status.Conditions, apis.RepositoryIntegrityVerified) {
							log.Infoln("Verifying repository integrity...........")
							w, err := restic.NewResticWrapper(o.SetupOpt)
							if err != nil {
								_, condErr := conditions.SetRepositoryIntegrityVerifiedConditionToFalse(o.StashClient, backupSession, err)
								return errors.NewAggregate([]error{err, condErr})
							}
							res, err := w.VerifyRepositoryIntegrity()
							if err != nil {
								log.Warningf("Failed to verify Repository integrity. Reason: %s", err.Error())
								_, condErr := conditions.SetRepositoryIntegrityVerifiedConditionToFalse(o.StashClient, backupSession, err)
								return errors.NewAggregate([]error{err, condErr})
							}
							if res != nil {
								repoStats.Integrity = res.Integrity
								repoStats.Size = res.Size
							}
							_, err = conditions.SetRepositoryIntegrityVerifiedConditionToTrue(o.StashClient, backupSession)
							if err != nil {
								return err
							}
						}
					case apis.SendRepositoryMetrics:
						// if metrics enabled then send metrics to the Prometheus pushgateway
						if o.Metrics.Enabled && !kmapi.HasCondition(backupSession.Status.Conditions, apis.RepositoryMetricsPushed) {
							log.Infoln("Pushing repository metrics...........")
							err := o.Metrics.SendRepositoryMetrics(o.Config, invoker, repoStats)
							if err != nil {
								log.Warningf("Failed to send Repository metrics. Reason: %s", err.Error())
								_, condErr := conditions.SetRepositoryMetricsPushedConditionToFalse(o.StashClient, backupSession, err)
								return errors.NewAggregate([]error{err, condErr})
							}
							_, err = conditions.SetRepositoryMetricsPushedConditionToTrue(o.StashClient, backupSession)
							if err != nil {
								return err
							}
						}
					default:
						return fmt.Errorf("unknown PostBackupAction: %s", action)
					}
				}
				// now, update the repository status
				if repoStats.Integrity != nil {
					log.Infoln("Updating repository status......")
					repo, err := o.StashClient.StashV1alpha1().Repositories(o.Namespace).Get(context.TODO(), invoker.Repository, metav1.GetOptions{})
					if err != nil {
						return err
					}
					_, err = stash_util.UpdateRepositoryStatus(context.TODO(), o.StashClient.StashV1alpha1(), repo.ObjectMeta, func(in *v1alpha1.RepositoryStatus) *v1alpha1.RepositoryStatus {
						in.Integrity = repoStats.Integrity
						in.SnapshotCount = repoStats.SnapshotCount
						in.SnapshotsRemovedOnLastCleanup = repoStats.SnapshotsRemovedOnLastCleanup
						in.TotalSize = repoStats.Size
						in.LastBackupTime = &backupSession.CreationTimestamp
						return in
					}, metav1.UpdateOptions{})
					return err
				}
			}
		}
	}
	return nil
}

func (o UpdateStatusOptions) waitUntilOtherHostsCompleted(backupSession *v1beta1.BackupSession, curTarget v1beta1.TargetRef) error {
	return wait.PollImmediate(5*time.Second, 30*time.Minute, func() (done bool, err error) {
		log.Infof("Waiting for all other targets/hosts to complete their backup.....")
		newBackupSession, err := o.StashClient.StashV1beta1().BackupSessions(backupSession.Namespace).Get(context.TODO(), backupSession.Name, metav1.GetOptions{})
		if err != nil {
			return false, err
		}
		for _, target := range newBackupSession.Status.Targets {
			if apis.TargetMatched(target.Ref, curTarget) {
				// If all the other hosts complete their backup process, they should add their entry into target.Stats field.
				// So, the target.Stats should have one less entry than the totalHosts because the current host hasn't added its entry yet.
				if len(target.Stats) != (int(*target.TotalHosts) - 1) {
					return false, nil
				}
			} else {
				if !targetBackupCompleted(target.Phase) {
					return false, nil
				}
			}
		}
		return true, nil
	})
}

func targetBackupCompleted(phase v1beta1.TargetPhase) bool {
	return phase == v1beta1.TargetBackupSucceeded || phase == v1beta1.TargetBackupFailed
}

func backupSucceeded(output *restic.BackupOutput) bool {
	for _, stat := range output.BackupTargetStatus.Stats {
		if stat.Phase == v1beta1.HostBackupFailed {
			return false
		}
	}
	return true
}

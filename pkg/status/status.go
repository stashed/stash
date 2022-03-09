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
	"stash.appscode.dev/apimachinery/pkg/conditions"
	"stash.appscode.dev/apimachinery/pkg/invoker"
	"stash.appscode.dev/apimachinery/pkg/metrics"
	"stash.appscode.dev/apimachinery/pkg/restic"
	"stash.appscode.dev/stash/pkg/eventer"

	core "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/errors"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/klog/v2"
	kmapi "kmodules.xyz/client-go/api/v1"
)

type UpdateStatusOptions struct {
	Config      *rest.Config
	KubeClient  kubernetes.Interface
	StashClient cs.Interface

	Namespace      string
	BackupSession  string
	OutputDir      string
	OutputFileName string
	InvokerKind    string
	InvokerName    string

	StorageSecret kmapi.ObjectReference

	Metrics   metrics.MetricsOptions
	TargetRef v1beta1.TargetRef
	SetupOpt  restic.SetupOptions
}

func (o UpdateStatusOptions) UpdateBackupStatusFromFile() error {
	// read backup output from file
	klog.Infof("Reading backup output from file: %s", filepath.Join(o.OutputDir, o.OutputFileName))
	backupOutput, err := restic.ReadBackupOutput(filepath.Join(o.OutputDir, o.OutputFileName))
	if err != nil {
		return err
	}
	return o.UpdatePostBackupStatus(backupOutput)
}

func (o UpdateStatusOptions) UpdateRestoreStatusFromFile() error {
	// read restore output from file
	klog.Infof("Reading restore output from file: %s", filepath.Join(o.OutputDir, o.OutputFileName))
	restoreOutput, err := restic.ReadRestoreOutput(filepath.Join(o.OutputDir, o.OutputFileName))
	if err != nil {
		return err
	}

	inv, err := invoker.NewRestoreInvoker(o.KubeClient, o.StashClient, o.InvokerKind, o.InvokerName, o.Namespace)
	if err != nil {
		return err
	}
	for _, targetInfo := range inv.GetTargetInfo() {
		if targetInfo.Target != nil &&
			targetInfo.Target.Ref.Kind == o.TargetRef.Kind &&
			targetInfo.Target.Ref.Name == o.TargetRef.Name {
			return o.UpdatePostRestoreStatus(restoreOutput, inv, targetInfo)
		}
	}
	return nil
}

func (o UpdateStatusOptions) UpdatePostBackupStatus(backupOutput *restic.BackupOutput) error {
	klog.Infof("Updating post backup status.......")
	if backupOutput == nil {
		return fmt.Errorf("invalid backup ouputput. Backup output must not be nil")
	}
	// get backup session, update status and create event
	backupSession, err := o.StashClient.StashV1beta1().BackupSessions(o.Namespace).Get(context.TODO(), o.BackupSession, metav1.GetOptions{})
	if err != nil {
		return err
	}

	session := invoker.NewBackupSessionHandler(o.StashClient, backupSession)

	inv, err := session.GetInvoker()
	if err != nil {
		return err
	}

	var targetInfo invoker.BackupTargetInfo
	for _, t := range inv.GetTargetInfo() {
		if t.Target != nil &&
			t.Target.Ref.Kind == o.TargetRef.Kind &&
			t.Target.Ref.Name == o.TargetRef.Name {
			targetInfo = t
		}
	}

	var statusErr error
	// If the target has been assigned some post-backup actions, execute them
	// Only execute the postBackupActions if the backup of current host/hosts has succeeded
	// We should update the target status even if the post-backup actions failed
	if backupSucceeded(backupOutput) {
		statusErr = o.executePostBackupActions(inv, session, targetInfo.Target.Ref, len(backupOutput.BackupTargetStatus.Stats))
	}

	err = session.UpdateStatus(&v1beta1.BackupSessionStatus{
		Targets: []v1beta1.BackupTargetStatus{
			{
				Ref:   targetInfo.Target.Ref,
				Stats: backupOutput.BackupTargetStatus.Stats,
			},
		},
	})
	if err != nil {
		statusErr = errors.NewAggregate([]error{statusErr, err})
	}

	// create status against the BackupSession for each hosts
	for _, hostStats := range backupOutput.BackupTargetStatus.Stats {
		var eventType, eventReason, eventMessage string
		if hostStats.Error != "" {
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
			klog.Errorf("failed to create event for %s %s/%s. Reason: %v",
				inv.GetTypeMeta().Kind,
				inv.GetObjectMeta().Namespace,
				inv.GetObjectMeta().Name,
				err,
			)
		}
	}

	// if metrics enabled then send backup host specific metrics to the Prometheus pushgateway
	if o.Metrics.Enabled && targetInfo.Target != nil {
		err = o.Metrics.SendBackupHostMetrics(o.Config, inv, targetInfo.Target.Ref, backupOutput)
		if err != nil {
			statusErr = errors.NewAggregate([]error{statusErr, err})
		}
	}
	return statusErr
}

func (o UpdateStatusOptions) UpdatePostRestoreStatus(restoreOutput *restic.RestoreOutput, inv invoker.RestoreInvoker, targetInfo invoker.RestoreTargetInfo) error {
	if restoreOutput == nil {
		return fmt.Errorf("invalid restore output. Restore output must not be nil")
	}
	// add or update entry for each host in restore invoker status
	var err error
	klog.Infof("Updating hosts status for restore target %s %s/%s.",
		targetInfo.Target.Ref.Kind,
		inv.GetObjectMeta().Namespace,
		targetInfo.Target.Ref.Name,
	)
	err = inv.UpdateStatus(invoker.RestoreInvokerStatus{
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
	// create event against the RestoreSession for each hosts
	for _, hostStats := range restoreOutput.RestoreTargetStatus.Stats {
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
		err = inv.CreateEvent(eventType, eventer.EventSourceStatusUpdater, eventReason, eventMessage)
		if err != nil {
			klog.Errorf("failed to create event for %s %s/%s. Reason: %v",
				inv.GetTypeMeta().Kind,
				inv.GetObjectMeta().Namespace,
				inv.GetObjectMeta().Name,
				err,
			)
		}
	}
	// if metrics enabled then send metrics to the Prometheus pushgateway
	if o.Metrics.Enabled && targetInfo.Target != nil {
		return o.Metrics.SendRestoreHostMetrics(o.Config, inv, targetInfo.Target.Ref, restoreOutput)
	}
	return nil
}

func (o UpdateStatusOptions) executePostBackupActions(inv invoker.BackupInvoker, session *invoker.BackupSessionHandler, curTarget v1beta1.TargetRef, numCurHosts int) error {
	var repoStats restic.RepositoryStats
	for _, targetStatus := range session.GetTargetStatus() {
		if invoker.TargetMatched(targetStatus.Ref, curTarget) {
			// check if it has any post-backup action assigned to it
			if len(targetStatus.PostBackupActions) > 0 {
				// For StatefulSet and DaemonSet, only the last host will run these PostBackupActions
				if curTarget.Kind == apis.KindStatefulSet || curTarget.Kind == apis.KindDaemonSet {
					if len(targetStatus.Stats) != (int(*targetStatus.TotalHosts) - numCurHosts) {
						klog.Infof("Skipping running PostBackupActions. Reason: Only the last host will execute the post backup actions for %s", curTarget.Kind)
						return nil
					}
				}
				// wait until all other targets/hosts has completed their backup.
				err := o.waitUntilOtherHostsCompleted(session.GetBackupSession(), curTarget, numCurHosts)
				if err != nil {
					return fmt.Errorf("failed to execute PostBackupActions. Reason: %v ", err.Error())
				}

				// execute the post-backup actions
				for _, action := range targetStatus.PostBackupActions {
					switch action {
					case v1beta1.ApplyRetentionPolicy:
						if !kmapi.HasCondition(session.GetConditions(), v1beta1.RetentionPolicyApplied) {
							klog.Infoln("Applying retention policy.....")
							w, err := restic.NewResticWrapper(o.SetupOpt)
							if err != nil {
								condErr := conditions.SetRetentionPolicyAppliedConditionToFalse(session, err)
								return errors.NewAggregate([]error{err, condErr})
							}
							res, err := w.ApplyRetentionPolicies(inv.GetRetentionPolicy())
							if err != nil {
								klog.Warningf("Failed to apply retention policy. Reason: %s", err.Error())
								condErr := conditions.SetRetentionPolicyAppliedConditionToFalse(session, err)
								return errors.NewAggregate([]error{err, condErr})
							}
							if res != nil {
								repoStats.SnapshotCount = res.SnapshotCount
								repoStats.SnapshotsRemovedOnLastCleanup = res.SnapshotsRemovedOnLastCleanup
							}
							err = conditions.SetRetentionPolicyAppliedConditionToTrue(session)
							if err != nil {
								return err
							}
						}
					case v1beta1.VerifyRepositoryIntegrity:
						if !kmapi.HasCondition(session.GetConditions(), v1beta1.RepositoryIntegrityVerified) {
							klog.Infoln("Verifying repository integrity...........")
							w, err := restic.NewResticWrapper(o.SetupOpt)
							if err != nil {
								condErr := conditions.SetRepositoryIntegrityVerifiedConditionToFalse(session, err)
								return errors.NewAggregate([]error{err, condErr})
							}
							res, err := w.VerifyRepositoryIntegrity()
							if err != nil {
								klog.Warningf("Failed to verify Repository integrity. Reason: %s", err.Error())
								condErr := conditions.SetRepositoryIntegrityVerifiedConditionToFalse(session, err)
								return errors.NewAggregate([]error{err, condErr})
							}
							if res != nil {
								repoStats.Integrity = res.Integrity
								repoStats.Size = res.Size
							}
							err = conditions.SetRepositoryIntegrityVerifiedConditionToTrue(session)
							if err != nil {
								return err
							}
						}
					case v1beta1.SendRepositoryMetrics:
						// if metrics enabled then send metrics to the Prometheus pushgateway
						if o.Metrics.Enabled && !kmapi.HasCondition(session.GetConditions(), v1beta1.RepositoryMetricsPushed) {
							klog.Infoln("Pushing repository metrics...........")
							err := o.Metrics.SendRepositoryMetrics(o.Config, inv, repoStats)
							if err != nil {
								klog.Warningf("Failed to send Repository metrics. Reason: %s", err.Error())
								condErr := conditions.SetRepositoryMetricsPushedConditionToFalse(session, err)
								return errors.NewAggregate([]error{err, condErr})
							}
							return conditions.SetRepositoryMetricsPushedConditionToTrue(session)
						}
					default:
						return fmt.Errorf("unknown PostBackupAction: %s", action)
					}
				}
				// now, update the repository status
				if repoStats.Integrity != nil {
					klog.Infoln("Updating repository status......")
					repo, err := inv.GetRepository()
					if err != nil {
						return err
					}
					startTime := session.GetObjectMeta().CreationTimestamp
					_, err = stash_util.UpdateRepositoryStatus(
						context.TODO(),
						o.StashClient.StashV1alpha1(),
						repo.ObjectMeta,
						func(in *v1alpha1.RepositoryStatus) (types.UID, *v1alpha1.RepositoryStatus) {
							in.Integrity = repoStats.Integrity
							in.SnapshotCount = repoStats.SnapshotCount
							in.SnapshotsRemovedOnLastCleanup = repoStats.SnapshotsRemovedOnLastCleanup
							in.TotalSize = repoStats.Size
							in.LastBackupTime = &startTime
							return repo.UID, in
						}, metav1.UpdateOptions{})
					return err
				}
			}
		}
	}
	return nil
}

func (o UpdateStatusOptions) waitUntilOtherHostsCompleted(backupSession *v1beta1.BackupSession, curTarget v1beta1.TargetRef, numCurHosts int) error {
	return wait.PollImmediate(5*time.Second, 30*time.Minute, func() (done bool, err error) {
		klog.Infof("Waiting for all other targets/hosts to complete their backup.....")
		newBackupSession, err := o.StashClient.StashV1beta1().BackupSessions(backupSession.Namespace).Get(context.TODO(), backupSession.Name, metav1.GetOptions{})
		if err != nil {
			return false, err
		}
		for _, target := range newBackupSession.Status.Targets {
			if invoker.TargetMatched(target.Ref, curTarget) {
				// If all the other hosts complete their backup process, they should add their entry into target.Stats field.
				// So, the target.Stats should have (target.TotalHosts - numCurHosts) entry.
				if len(target.Stats) != (int(*target.TotalHosts) - numCurHosts) {
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

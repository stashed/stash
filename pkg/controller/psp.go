package controller

import (
	api_v1beta1 "stash.appscode.dev/stash/apis/stash/v1beta1"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	DefaultBackupSessionCronJobPSPName = "stash-backupsession-cron"
	DefaultBackupJobPSPName            = "stash-backup-job"
	DefaultRestoreJobPSPName           = "stash-restore-job"
)

func (c *StashController) getBackupSessionCronJobPSPNames() []string {
	// BackupSession cron does not need any custom PSP. So, default minimum privileged
	return []string{DefaultBackupSessionCronJobPSPName}
}

func (c *StashController) getBackupJobPSPNames(backupConfig *api_v1beta1.BackupConfiguration) ([]string, error) {
	// if task field is empty then return default backup job psp
	if backupConfig.Spec.Task.Name == "" {
		return []string{DefaultBackupJobPSPName}, nil
	}

	// find out task and then functions. finally, get psp names from the functions
	task, err := c.stashClient.StashV1beta1().Tasks().Get(backupConfig.Spec.Task.Name, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}

	var psps []string
	for _, step := range task.Spec.Steps {
		fn, err := c.stashClient.StashV1beta1().Functions().Get(step.Name, metav1.GetOptions{})
		if err != nil {
			return nil, err
		}
		if fn.Spec.PodSecurityPolicyName != "" {
			psps = append(psps, fn.Spec.PodSecurityPolicyName)
		}
	}

	if len(psps) != 0 {
		return psps, nil
	}

	// if no PSP name is specified, then return default PSP for backup job
	return []string{DefaultBackupJobPSPName}, nil
}

func (c *StashController) getRestoreJobPSPNames(restoreSession *api_v1beta1.RestoreSession) ([]string, error) {
	// if task field is empty then return default restore job psp
	if restoreSession.Spec.Task.Name == "" {
		return []string{DefaultRestoreJobPSPName}, nil
	}

	// find out task and then functions. finally, get psp names from the functions
	task, err := c.stashClient.StashV1beta1().Tasks().Get(restoreSession.Spec.Task.Name, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}

	var psps []string
	for _, step := range task.Spec.Steps {
		fn, err := c.stashClient.StashV1beta1().Functions().Get(step.Name, metav1.GetOptions{})
		if err != nil {
			return nil, err
		}
		if fn.Spec.PodSecurityPolicyName != "" {
			psps = append(psps, fn.Spec.PodSecurityPolicyName)
		}
	}

	if len(psps) != 0 {
		return psps, nil
	}

	// if no PSP name is specified, then return default PSP for restore job
	return []string{DefaultRestoreJobPSPName}, nil
}

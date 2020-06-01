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
	"context"

	api_v1beta1 "stash.appscode.dev/apimachinery/apis/stash/v1beta1"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	BaselinePSP = "baseline"
)

func (c *StashController) getBackupSessionCronJobPSPNames() []string {
	// BackupSession cron does not need any custom PSP. So, default minimum privileged
	return []string{BaselinePSP}
}

func (c *StashController) getBackupJobPSPNames(taskRef api_v1beta1.TaskRef) ([]string, error) {
	// if task field is empty then return default backup job psp
	if taskRef.Name == "" {
		return []string{BaselinePSP}, nil
	}

	// find out task and then functions. finally, get psp names from the functions
	task, err := c.stashClient.StashV1beta1().Tasks().Get(context.TODO(), taskRef.Name, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}

	var psps []string
	for _, step := range task.Spec.Steps {
		fn, err := c.stashClient.StashV1beta1().Functions().Get(context.TODO(), step.Name, metav1.GetOptions{})
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
	return []string{BaselinePSP}, nil
}

func (c *StashController) getRestoreJobPSPNames(restoreSession *api_v1beta1.RestoreSession) ([]string, error) {
	// if task field is empty then return default restore job psp
	if restoreSession.Spec.Task.Name == "" {
		return []string{BaselinePSP}, nil
	}

	// find out task and then functions. finally, get psp names from the functions
	task, err := c.stashClient.StashV1beta1().Tasks().Get(context.TODO(), restoreSession.Spec.Task.Name, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}

	var psps []string
	for _, step := range task.Spec.Steps {
		fn, err := c.stashClient.StashV1beta1().Functions().Get(context.TODO(), step.Name, metav1.GetOptions{})
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
	return []string{BaselinePSP}, nil
}

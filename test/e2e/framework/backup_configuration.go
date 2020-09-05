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

package framework

import (
	"context"
	"strconv"
	"strings"

	"stash.appscode.dev/apimachinery/apis"
	"stash.appscode.dev/apimachinery/apis/stash/v1alpha1"
	"stash.appscode.dev/apimachinery/apis/stash/v1beta1"

	"github.com/appscode/go/crypto/rand"
	. "github.com/onsi/gomega"
	batchv1 "k8s.io/api/batch/v1"
	batch_v1beta1 "k8s.io/api/batch/v1beta1"
	core "k8s.io/api/core/v1"
	kerr "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	meta_util "kmodules.xyz/client-go/meta"
)

func (fi *Invocation) GetBackupConfiguration(repoName string, transformFuncs ...func(bc *v1beta1.BackupConfiguration)) *v1beta1.BackupConfiguration {
	backupConfig := &v1beta1.BackupConfiguration{
		ObjectMeta: metav1.ObjectMeta{
			Name:      rand.WithUniqSuffix(fi.app),
			Namespace: fi.namespace,
		},
		Spec: v1beta1.BackupConfigurationSpec{
			Repository: core.LocalObjectReference{
				Name: repoName,
			},
			// some workloads such as StatefulSet or DaemonSet may take long to complete backup. so, giving a fixed short interval is not always feasible.
			// hence, set the schedule to a large interval so that no backup schedule appear before completing running backup
			// we will use manual triggering for taking backup. this will help us to avoid waiting for fixed interval before each backup
			// and the problem mentioned above
			Schedule: "59 * * * *",
			RetentionPolicy: v1alpha1.RetentionPolicy{
				Name:     "keep-last-5",
				KeepLast: 5,
				Prune:    true,
			},
		},
	}
	// transformFuncs provides a array of functions that made test specific change on the BackupConfiguration
	// apply these test specific changes
	for _, fn := range transformFuncs {
		fn(backupConfig)
	}

	return backupConfig
}

func (fi *Invocation) CreateBackupConfiguration(backupCfg v1beta1.BackupConfiguration) error {
	_, err := fi.StashClient.StashV1beta1().BackupConfigurations(backupCfg.Namespace).Create(context.TODO(), &backupCfg, metav1.CreateOptions{})
	return err
}

func (fi *Invocation) DeleteBackupConfiguration(backupCfg v1beta1.BackupConfiguration) error {
	err := fi.StashClient.StashV1beta1().BackupConfigurations(backupCfg.Namespace).Delete(context.TODO(), backupCfg.Name, metav1.DeleteOptions{})
	if err != nil && !kerr.IsNotFound(err) {
		return err
	}
	return nil
}

func (f *Framework) EventuallyCronJobCreated(meta metav1.ObjectMeta) GomegaAsyncAssertion {
	return Eventually(
		func() bool {
			_, err := f.GetCronJob(meta)
			if err == nil && !kerr.IsNotFound(err) {
				return true
			}
			return false
		},
		WaitTimeOut,
		PullInterval,
	)
}

func (f *Framework) GetCronJob(meta metav1.ObjectMeta) (*batch_v1beta1.CronJob, error) {
	return f.KubeClient.BatchV1beta1().CronJobs(meta.Namespace).Get(context.TODO(), getBackupCronJobName(meta), metav1.GetOptions{})
}

func (f *Framework) EventuallyCronJobSuspended(meta metav1.ObjectMeta) GomegaAsyncAssertion {
	return Eventually(
		func() bool {
			cronJob, err := f.KubeClient.BatchV1beta1().CronJobs(meta.Namespace).Get(context.TODO(), getBackupCronJobName(meta), metav1.GetOptions{})
			if err != nil {
				return false
			}
			return *cronJob.Spec.Suspend
		},
		WaitTimeOut,
		PullInterval,
	)
}

func (f *Framework) EventuallyCronJobResumed(meta metav1.ObjectMeta) GomegaAsyncAssertion {
	return Eventually(
		func() bool {
			cronJob, err := f.KubeClient.BatchV1beta1().CronJobs(meta.Namespace).Get(context.TODO(), getBackupCronJobName(meta), metav1.GetOptions{})
			if err != nil {
				return false
			}
			return !*cronJob.Spec.Suspend
		},
		WaitTimeOut,
		PullInterval,
	)
}

func (f *Framework) EventuallyBackupConfigurationCreated(meta metav1.ObjectMeta) GomegaAsyncAssertion {
	return Eventually(
		func() bool {
			_, err := f.StashClient.StashV1beta1().BackupConfigurations(meta.Namespace).Get(context.TODO(), meta.Name, metav1.GetOptions{})
			if err == nil && !kerr.IsNotFound(err) {
				return true
			}
			return false
		},
		WaitTimeOut,
		PullInterval,
	)
}

func (f *Framework) GetBackupJob(backupSessionName string) (*batchv1.Job, error) {
	return f.KubeClient.BatchV1().Jobs(f.namespace).Get(context.TODO(), getBackupJobName(backupSessionName, strconv.Itoa(0)), metav1.GetOptions{})
}

func getBackupCronJobName(objMeta metav1.ObjectMeta) string {
	return meta_util.ValidCronJobNameWithPrefix(apis.PrefixStashBackup, strings.ReplaceAll(objMeta.Name, ".", "-"))
}

func getBackupJobName(backupSessionName string, index string) string {
	return meta_util.ValidNameWithPrefixNSuffix(apis.PrefixStashBackup, strings.ReplaceAll(backupSessionName, ".", "-"), index)
}

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
	"fmt"
	"strings"

	"stash.appscode.dev/apimachinery/apis"
	"stash.appscode.dev/apimachinery/apis/stash/v1beta1"

	"github.com/appscode/go/crypto/rand"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func (f *Framework) EventuallyBackupSessionPhase(meta metav1.ObjectMeta) GomegaAsyncAssertion {
	return Eventually(
		func() (phase v1beta1.BackupSessionPhase) {
			bs, err := f.StashClient.StashV1beta1().BackupSessions(meta.Namespace).Get(context.TODO(), meta.Name, metav1.GetOptions{})
			Expect(err).NotTo(HaveOccurred())
			return bs.Status.Phase
		}, WaitTimeOut, PullInterval)
}

func (f *Framework) EventuallyBackupProcessCompleted(meta metav1.ObjectMeta) GomegaAsyncAssertion {
	return Eventually(
		func() bool {
			bs, err := f.StashClient.StashV1beta1().BackupSessions(meta.Namespace).Get(context.TODO(), meta.Name, metav1.GetOptions{})
			if err != nil {
				Expect(err).NotTo(HaveOccurred())
				return false
			}
			if bs.Status.Phase == v1beta1.BackupSessionSucceeded ||
				bs.Status.Phase == v1beta1.BackupSessionFailed ||
				bs.Status.Phase == v1beta1.BackupSessionUnknown {
				return true
			}
			return false
		}, WaitTimeOut, PullInterval)
}

func (f *Framework) EventuallyBackupSessionCreated(meta metav1.ObjectMeta) GomegaAsyncAssertion {
	return Eventually(
		func() bool {
			backupsnlist, err := f.StashClient.StashV1beta1().BackupSessions(meta.Namespace).List(context.TODO(), metav1.ListOptions{})
			Expect(err).NotTo(HaveOccurred())
			return len(backupsnlist.Items) > 0
		}, WaitTimeOut, PullInterval)
}

func (f *Framework) GetBackupSession(meta metav1.ObjectMeta) (*v1beta1.BackupSession, error) {
	backupsnlist, err := f.StashClient.StashV1beta1().BackupSessions(meta.Namespace).List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return nil, err
	}
	if len(backupsnlist.Items) > 0 {
		for _, bs := range backupsnlist.Items {
			if strings.HasPrefix(bs.Name, meta.Name) {
				return &bs, nil
			}
		}
	}
	return nil, fmt.Errorf("no BackupSession found")
}

func (fi *Invocation) TriggerInstantBackup(objMeta metav1.ObjectMeta, invokerRef v1beta1.BackupInvokerRef) (*v1beta1.BackupSession, error) {
	backupSession := &v1beta1.BackupSession{
		ObjectMeta: metav1.ObjectMeta{
			Name:      rand.WithUniqSuffix(objMeta.Name),
			Namespace: objMeta.Namespace,
			Labels: map[string]string{
				apis.LabelApp:         apis.AppLabelStash,
				apis.LabelInvokerType: invokerRef.Kind,
				apis.LabelInvokerName: invokerRef.Name,
			},
		},
		Spec: v1beta1.BackupSessionSpec{
			Invoker: v1beta1.BackupInvokerRef{
				APIGroup: v1beta1.SchemeGroupVersion.Group,
				Kind:     invokerRef.Kind,
				Name:     invokerRef.Name,
			},
		},
	}

	return fi.StashClient.StashV1beta1().BackupSessions(backupSession.Namespace).Create(context.TODO(), backupSession, metav1.CreateOptions{})
}

func (fi *Invocation) EventuallyBackupCount(invokerMeta metav1.ObjectMeta, invokerKind string) GomegaAsyncAssertion {
	return Eventually(func() int64 {
		count, err := fi.GetSuccessfulBackupSessionCount(invokerMeta, invokerKind)
		if err != nil {
			return 0
		}
		return count
	}, WaitTimeOut, PullInterval)
}

func (fi *Invocation) GetSuccessfulBackupSessionCount(invokerMeta metav1.ObjectMeta, invokerKind string) (int64, error) {
	backupSessions, err := fi.StashClient.StashV1beta1().BackupSessions(fi.namespace).List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return 0, err
	}

	count := int64(0)
	for _, bs := range backupSessions.Items {
		if bs.Spec.Invoker.Kind == invokerKind &&
			bs.Spec.Invoker.Name == invokerMeta.Name &&
			bs.Status.Phase == v1beta1.BackupSessionSucceeded {
			count++
		}
	}
	return count, nil
}

func (fi *Invocation) EventuallyRunningBackupCompleted(invokerMeta metav1.ObjectMeta, invokerKind string) GomegaAsyncAssertion {
	return Eventually(func() bool {
		backupSessions, err := fi.StashClient.StashV1beta1().BackupSessions(fi.namespace).List(context.TODO(), metav1.ListOptions{})
		if err != nil {
			return false
		}
		for _, bs := range backupSessions.Items {
			if bs.Spec.Invoker.Kind == invokerKind &&
				bs.Spec.Invoker.Name == invokerMeta.Name &&
				bs.Status.Phase == v1beta1.BackupSessionRunning {
				return false
			}
		}
		return true
	}, WaitTimeOut, PullInterval)
}

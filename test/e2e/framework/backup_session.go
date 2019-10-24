package framework

import (
	"fmt"
	"strings"

	"stash.appscode.dev/stash/apis/stash/v1beta1"
	"stash.appscode.dev/stash/pkg/util"

	"github.com/appscode/go/crypto/rand"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func (f *Framework) EventuallyBackupSessionPhase(meta metav1.ObjectMeta) GomegaAsyncAssertion {
	return Eventually(
		func() (phase v1beta1.BackupSessionPhase) {
			bs, err := f.StashClient.StashV1beta1().BackupSessions(meta.Namespace).Get(meta.Name, metav1.GetOptions{})
			Expect(err).NotTo(HaveOccurred())
			return bs.Status.Phase
		},
	)
}

func (f *Framework) EventuallyBackupProcessCompleted(meta metav1.ObjectMeta) GomegaAsyncAssertion {
	return Eventually(
		func() bool {
			bs, err := f.StashClient.StashV1beta1().BackupSessions(meta.Namespace).Get(meta.Name, metav1.GetOptions{})
			if err != nil {
				return false
			}
			if bs.Status.Phase == v1beta1.BackupSessionSucceeded ||
				bs.Status.Phase == v1beta1.BackupSessionFailed ||
				bs.Status.Phase == v1beta1.BackupSessionSkipped ||
				bs.Status.Phase == v1beta1.BackupSessionUnknown {
				return true
			}
			return false
		},
	)
}

func (f *Framework) EventuallyBackupSessionCreated(meta metav1.ObjectMeta) GomegaAsyncAssertion {
	return Eventually(
		func() bool {
			backupsnlist, err := f.StashClient.StashV1beta1().BackupSessions(meta.Namespace).List(metav1.ListOptions{})
			Expect(err).NotTo(HaveOccurred())
			return len(backupsnlist.Items) > 0
		},
	)
}

func (f *Framework) GetBackupSession(meta metav1.ObjectMeta) (*v1beta1.BackupSession, error) {
	backupsnlist, err := f.StashClient.StashV1beta1().BackupSessions(meta.Namespace).List(metav1.ListOptions{})
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

func (f *Framework) EventuallyBackupSessionTotalHost(meta metav1.ObjectMeta) GomegaAsyncAssertion {
	return Eventually(
		func() (totalhost *int32) {
			bs, err := f.StashClient.StashV1beta1().BackupSessions(meta.Namespace).Get(meta.Name, metav1.GetOptions{})
			Expect(err).NotTo(HaveOccurred())
			return bs.Status.TotalHosts
		},
	)
}

func (f *Invocation) TriggerInstantBackup(backupConfig *v1beta1.BackupConfiguration) (*v1beta1.BackupSession, error) {
	backupSession := &v1beta1.BackupSession{
		ObjectMeta: metav1.ObjectMeta{
			Name:      rand.WithUniqSuffix(backupConfig.Name),
			Namespace: backupConfig.Namespace,
			Labels: map[string]string{
				util.LabelApp:                 util.AppLabelStash,
				util.LabelBackupConfiguration: backupConfig.Name,
			},
		},
		Spec: v1beta1.BackupSessionSpec{
			Invoker: v1beta1.BackupInvokerRef{
				APIGroup: v1beta1.SchemeGroupVersion.Group,
				Kind:     v1beta1.ResourceKindBackupConfiguration,
				Name:     backupConfig.Name,
			},
		},
	}

	return f.StashClient.StashV1beta1().BackupSessions(backupSession.Namespace).Create(backupSession)
}

package framework

import (
	"fmt"
	"time"

	"github.com/appscode/stash/apis/stash/v1beta1"
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

func (f *Framework) EventuallyBackupSessionCreated(meta metav1.ObjectMeta) GomegaAsyncAssertion {
	return Eventually(
		func() bool {
			backupsnlist, err := f.StashClient.StashV1beta1().BackupSessions(meta.Namespace).List(metav1.ListOptions{})
			Expect(err).NotTo(HaveOccurred())
			if len(backupsnlist.Items) > 0 {
				return true
			}
			return false
		},
		time.Minute*7,
		time.Second*5,
	)
}

func (f *Framework) GetBackupSession(meta metav1.ObjectMeta) (*v1beta1.BackupSession, error) {
	backupsnlist, err := f.StashClient.StashV1beta1().BackupSessions(meta.Namespace).List(metav1.ListOptions{})
	if err != nil {
		return nil, err
	}
	if len(backupsnlist.Items) > 0 {
		return &backupsnlist.Items[0], nil
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

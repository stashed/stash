package framework

import (
	"fmt"
	"time"

	kerr "k8s.io/apimachinery/pkg/api/errors"

	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"stash.appscode.dev/stash/apis/stash/v1beta1"
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

func (f *Framework) EventuallyBackupSessionNotCreated(meta metav1.ObjectMeta) GomegaAsyncAssertion {
	try := 0
	return Eventually(
		func() bool {
			try = try + 1
			backupsnlist, err := f.StashClient.StashV1beta1().BackupSessions(meta.Namespace).List(metav1.ListOptions{})
			Expect(err).NotTo(HaveOccurred())
			if len(backupsnlist.Items) > 0 {
				return true
			}
			if try > 1 {
				if kerr.IsNotFound(err) {
					Expect(err).To(HaveOccurred())
				}
				return true
			}
			return false
		},
		time.Minute*3,
		time.Minute*2,
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

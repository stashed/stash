package framework

import (
	"time"

	"github.com/appscode/go/crypto/rand"
	. "github.com/onsi/gomega"
	core "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"stash.appscode.dev/stash/apis"
	"stash.appscode.dev/stash/apis/stash/v1beta1"
)

func (f *Invocation) RestoreSession(repoName string, targetref v1beta1.TargetRef, rules []v1beta1.Rule) v1beta1.RestoreSession {
	return v1beta1.RestoreSession{
		ObjectMeta: metav1.ObjectMeta{
			Name:      rand.WithUniqSuffix(f.app),
			Namespace: f.namespace,
		},
		Spec: v1beta1.RestoreSessionSpec{
			Repository: core.LocalObjectReference{
				Name: repoName,
			},
			Rules: rules,
			Target: &v1beta1.RestoreTarget{
				Ref: targetref,
				VolumeMounts: []core.VolumeMount{
					{
						Name:      TestSourceDataVolumeName,
						MountPath: TestSourceDataMountPath,
					},
				},
			},
		},
	}
}

func (f *Invocation) CreateRestoreSession(restoreSession v1beta1.RestoreSession) error {
	_, err := f.StashClient.StashV1beta1().RestoreSessions(restoreSession.Namespace).Create(&restoreSession)
	return err
}

func (f Invocation) DeleteRestoreSession(meta metav1.ObjectMeta) error {
	err := f.StashClient.StashV1beta1().RestoreSessions(meta.Namespace).Delete(meta.Name, &metav1.DeleteOptions{})
	return err
}

func (f *Framework) EventuallyRestoreSessionPhase(meta metav1.ObjectMeta) GomegaAsyncAssertion {
	return Eventually(func() v1beta1.RestoreSessionPhase {
		restoreSession, err := f.StashClient.StashV1beta1().RestoreSessions(meta.Namespace).Get(meta.Name, metav1.GetOptions{})
		Expect(err).NotTo(HaveOccurred())
		return restoreSession.Status.Phase
	},
		time.Minute*7,
		time.Second*7,
	)
}

func (f *Invocation) PvcRestoreTarget(pvcName string) *v1beta1.RestoreTarget {
	return &v1beta1.RestoreTarget{
		Ref: v1beta1.TargetRef{
			APIVersion: "v1",
			Kind:       apis.KindPersistentVolumeClaim,
			Name:       pvcName,
		},
		VolumeMounts: []core.VolumeMount{
			{
				Name:      TestSourceDataVolumeName,
				MountPath: TestSourceDataMountPath,
			},
		},
	}
}

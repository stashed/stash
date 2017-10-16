package framework

import (
	"time"

	"github.com/appscode/go/crypto/rand"
	api "github.com/appscode/stash/apis/stash/v1alpha1"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	apiv1 "k8s.io/client-go/pkg/api/v1"
)

func (fi *Invocation) RecoveryForRestic(resticName string) api.Recovery {
	return api.Recovery{
		TypeMeta: metav1.TypeMeta{
			APIVersion: api.SchemeGroupVersion.String(),
			Kind:       api.ResourceKindRecovery,
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      rand.WithUniqSuffix("stash"),
			Namespace: fi.namespace,
		},
		Spec: api.RecoverySpec{
			Restic: resticName,
			VolumeMounts: []apiv1.VolumeMount{
				{
					Name:      TestRecoveredDataVolumeName,
					MountPath: TestRecoveredDataMountPath,
				},
			},
		},
		Volumes: []apiv1.Volume{
			{
				Name: TestRecoveredDataVolumeName,
				VolumeSource: apiv1.VolumeSource{
					EmptyDir: &apiv1.EmptyDirVolumeSource{},
				},
			},
		},
	}
}

func (f *Framework) CreateRecovery(obj api.Recovery) error {
	_, err := f.StashClient.Recoveries(obj.Namespace).Create(&obj)
	return err
}

func (f *Framework) DeleteRecovery(meta metav1.ObjectMeta) error {
	return f.StashClient.Recoveries(meta.Namespace).Delete(meta.Name, deleteInForeground())
}

func (f *Framework) EventuallyRecoverySucceed(meta metav1.ObjectMeta) GomegaAsyncAssertion {
	return Eventually(func() bool {
		obj, err := f.StashClient.Recoveries(meta.Namespace).Get(meta.Name, metav1.GetOptions{})
		Expect(err).NotTo(HaveOccurred())
		return obj.Status.RecoveryStatus == "SUCCEED"
	}, time.Minute*5, time.Minute*2)
}

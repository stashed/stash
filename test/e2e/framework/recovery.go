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
			Volumes: []apiv1.Volume{
				{
					Name: TestSourceDataVolumeName,
					VolumeSource: apiv1.VolumeSource{
						HostPath: &apiv1.HostPathVolumeSource{
							Path: "/data/stash-test/restic-restored",
						},
					},
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
		return obj.Status == "SUCCEED"
	}, time.Minute*5, time.Second*5)
}

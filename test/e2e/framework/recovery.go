package framework

import (
	"time"

	"github.com/appscode/go/crypto/rand"
	api "github.com/appscode/stash/apis/stash/v1alpha1"
	. "github.com/onsi/gomega"
	core "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func (fi *Invocation) RecoveryForRestic(restic api.Restic) api.Recovery {
	paths := make([]string, 0)
	for _, fg := range restic.Spec.FileGroups {
		paths = append(paths, fg.Path)
	}
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
			Paths:   paths,
			Backend: restic.Spec.Backend,
			RecoveryVolumes: []api.RecoveryVolume{
				{
					Name:      TestSourceDataVolumeName,
					MountPath: restic.Spec.VolumeMounts[0].MountPath,
					VolumeSource: core.VolumeSource{
						HostPath: &core.HostPathVolumeSource{
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
		return obj.Status.Phase == api.RecoverySucceeded
	}, time.Minute*5, time.Second*5)
}

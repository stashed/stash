package framework

import (
	"time"

	"github.com/appscode/go/crypto/rand"
	api "github.com/appscode/stash/apis/stash/v1alpha1"
	. "github.com/onsi/gomega"
	core "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	TestRecoveredVolumePath = "/data/stash-test/restic-restored"
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
			RecoveredVolumes: []api.LocalSpec{
				{
					MountPath: restic.Spec.VolumeMounts[0].MountPath,
					VolumeSource: core.VolumeSource{
						HostPath: &core.HostPathVolumeSource{
							Path: TestRecoveredVolumePath,
						},
					},
				},
			},
		},
	}
}

func (f *Framework) CreateRecovery(obj api.Recovery) error {
	_, err := f.StashClient.StashV1alpha1().Recoveries(obj.Namespace).Create(&obj)
	return err
}

func (f *Framework) DeleteRecovery(meta metav1.ObjectMeta) error {
	return f.StashClient.StashV1alpha1().Recoveries(meta.Namespace).Delete(meta.Name, deleteInBackground())
}

func (f *Framework) EventuallyRecoverySucceed(meta metav1.ObjectMeta) GomegaAsyncAssertion {
	return Eventually(func() bool {
		obj, err := f.StashClient.StashV1alpha1().Recoveries(meta.Namespace).Get(meta.Name, metav1.GetOptions{})
		Expect(err).NotTo(HaveOccurred())
		return obj.Status.Phase == api.RecoverySucceeded
	}, time.Minute*5, time.Second*5)
}

func (f *Framework) EventuallyRecoveredData(meta metav1.ObjectMeta, restic *api.Restic) GomegaAsyncAssertion {
	return Eventually(func() []string {
		recoveredData, err := f.ReadDataFromMountedDir(meta, restic)
		if err != nil {
			return nil
		}
		return recoveredData
	}, time.Minute*5, time.Second*5)
}

func (f *Framework) ReadDataFromMountedDir(meta metav1.ObjectMeta, restic *api.Restic) ([]string, error) {
	pod, err := f.GetPod(meta)
	if err != nil {
		return nil, err
	}
	datas := make([]string, 0)
	for _, fg := range restic.Spec.FileGroups {
		data, err := f.ExecOnPod(pod, "ls", fg.Path)
		if err != nil {
			return nil, err
		}
		datas = append(datas, data)
	}
	return datas, nil
}

/*
Copyright The Stash Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package framework

import (
	"path/filepath"
	"time"

	api "stash.appscode.dev/stash/apis/stash/v1alpha1"

	"github.com/appscode/go/crypto/rand"
	. "github.com/onsi/gomega"
	core "k8s.io/api/core/v1"
	kerr "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	store "kmodules.xyz/objectstore-api/api/v1"
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
			Paths: paths,
			RecoveredVolumes: []store.LocalSpec{
				{
					MountPath: restic.Spec.VolumeMounts[0].MountPath,
					VolumeSource: core.VolumeSource{
						HostPath: &core.HostPathVolumeSource{
							Path: filepath.Join(TestRecoveredVolumePath, fi.app),
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
	err := f.StashClient.StashV1alpha1().Recoveries(meta.Namespace).Delete(meta.Name, deleteInBackground())
	if err != nil && !kerr.IsNotFound(err) {
		return err
	}
	return nil
}

func (f *Framework) EventuallyRecoverySucceed(meta metav1.ObjectMeta) GomegaAsyncAssertion {
	return Eventually(func() bool {
		obj, err := f.StashClient.StashV1alpha1().Recoveries(meta.Namespace).Get(meta.Name, metav1.GetOptions{})
		Expect(err).NotTo(HaveOccurred())

		return obj.Status.Phase == api.RecoverySucceeded
	}, time.Minute*5, time.Second*5)
}

func (f *Framework) EventuallyRecoveredData(meta metav1.ObjectMeta, paths []string) GomegaAsyncAssertion {
	return Eventually(func() []string {
		recoveredData, err := f.ReadDataFromMountedDir(meta, paths)
		if err != nil {
			return nil
		}
		return recoveredData
	}, time.Minute*5, time.Second*5)
}

func (f *Framework) ReadDataFromMountedDir(meta metav1.ObjectMeta, paths []string) ([]string, error) {
	pod, err := f.GetPod(meta)
	if err != nil {
		return nil, err
	}

	datas := make([]string, 0)
	for _, path := range paths {
		data, err := f.ExecOnPod(pod, "ls", "-R", path)
		if err != nil {
			return nil, err
		}
		datas = append(datas, data)
	}
	return datas, nil
}

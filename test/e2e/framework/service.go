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
	core "k8s.io/api/core/v1"
	kerr "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	core_util "kmodules.xyz/client-go/core/v1"
)

const (
	TEST_HEADLESS_SERVICE = "headless"
)

func (fi *Invocation) HeadlessService(label string) core.Service {
	return core.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      TEST_HEADLESS_SERVICE,
			Namespace: fi.namespace,
		},
		Spec: core.ServiceSpec{
			Selector: map[string]string{
				"app": label,
			},
			ClusterIP: core.ClusterIPNone,
			Ports: []core.ServicePort{
				{
					Name: "http",
					Port: 80,
				},
			},
		},
	}
}

func (f *Framework) CreateService(obj core.Service) (*core.Service, error) {
	return f.KubeClient.CoreV1().Services(obj.Namespace).Create(&obj)
}

func (f *Framework) DeleteService(meta metav1.ObjectMeta) error {
	err := f.KubeClient.CoreV1().Services(meta.Namespace).Delete(meta.Name, deleteInForeground())
	if err != nil && !kerr.IsNotFound(err) {
		return err
	}
	return nil
}

func (f *Framework) CreateOrPatchService(obj core.Service) error {
	_, _, err := core_util.CreateOrPatchService(f.KubeClient, obj.ObjectMeta, func(in *core.Service) *core.Service {
		in.Spec = obj.Spec
		return in
	})
	return err
}

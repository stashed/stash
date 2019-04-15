package framework

import (
	core "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	core_util "kmodules.xyz/client-go/core/v1"
)

const (
	TEST_HEADLESS_SERVICE = "headless"
)

func (fi *Invocation) HeadlessService() core.Service {
	return core.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      TEST_HEADLESS_SERVICE,
			Namespace: fi.namespace,
		},
		Spec: core.ServiceSpec{
			Selector: map[string]string{
				"app": fi.app,
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

func (f *Framework) CreateService(obj core.Service) error {
	_, err := f.KubeClient.CoreV1().Services(obj.Namespace).Create(&obj)
	return err
}

func (f *Framework) DeleteService(meta metav1.ObjectMeta) error {
	return f.KubeClient.CoreV1().Services(meta.Namespace).Delete(meta.Name, deleteInForeground())
}

func (f *Framework) CreateOrPatchService(obj core.Service) error {
	_, _, err := core_util.CreateOrPatchService(f.KubeClient, obj.ObjectMeta, func(in *core.Service) *core.Service {
		in.Spec = obj.Spec
		return in
	})
	return err
}

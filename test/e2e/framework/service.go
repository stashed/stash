package framework

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	apiv1 "k8s.io/client-go/pkg/api/v1"
)

const (
	TEST_HEADLESS_SERVICE = "headless"
)

func (fi *Invocation) HeadlessService() apiv1.Service {
	return apiv1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      TEST_HEADLESS_SERVICE,
			Namespace: fi.namespace,
		},
		Spec: apiv1.ServiceSpec{
			Selector: map[string]string{
				"app": fi.app,
			},
			ClusterIP: apiv1.ClusterIPNone,
			Ports: []apiv1.ServicePort{
				{
					Name: "http",
					Port: 80,
				},
			},
		},
	}
}

func (f *Framework) CreateService(obj apiv1.Service) error {
	_, err := f.KubeClient.CoreV1().Services(obj.Namespace).Create(&obj)
	return err
}

func (f *Framework) DeleteService(meta metav1.ObjectMeta) error {
	return f.KubeClient.CoreV1().Services(meta.Namespace).Delete(meta.Name, deleteInForeground())
}

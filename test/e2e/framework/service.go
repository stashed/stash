package framework

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	apiv1 "k8s.io/client-go/pkg/api/v1"
)

const (
	TEST_HEADLESS_SERVICE = "headless"
)

func (f *Framework) HeadlessService(namespace string) apiv1.Service {
	return apiv1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      TEST_HEADLESS_SERVICE,
			Namespace: namespace,
		},
		Spec: apiv1.ServiceSpec{},
	}
}

func (f *Framework) CreateService(obj apiv1.Service) error {
	_, err := f.kubeClient.CoreV1().Services(obj.Namespace).Create(&obj)
	return err
}

func (f *Framework) DeleteService(meta metav1.ObjectMeta) error {
	return f.kubeClient.CoreV1().Services(meta.Namespace).Delete(meta.Name, &metav1.DeleteOptions{})
}

package framework

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	apiv1 "k8s.io/client-go/pkg/api/v1"
)

func (f *Framework) Namespace() string {
	return f.namespace
}

func (f *Framework) CreateNamespace() error {
	obj := apiv1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: f.namespace,
		},
	}
	_, err := f.KubeClient.CoreV1().Namespaces().Create(&obj)
	return err
}

func (f *Framework) DeleteNamespace() error {
	return f.KubeClient.CoreV1().Namespaces().Delete(f.namespace, deleteInBackground())
}

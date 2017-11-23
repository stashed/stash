package framework

import (
	core "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	kerr "k8s.io/apimachinery/pkg/api/errors"
)

func (f *Framework) Namespace() string {
	return f.namespace
}

func (f *Framework) CreateNamespace() error {
	obj := core.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: f.namespace,
		},
	}
	if _, err := f.KubeClient.CoreV1().Namespaces().Create(&obj); err != nil && kerr.IsAlreadyExists(err) {
		return err
	}
	return nil
}

func (f *Framework) DeleteNamespace() error {
	return f.KubeClient.CoreV1().Namespaces().Delete(f.namespace, deleteInBackground())
}

package framework

import (
	"github.com/appscode/go/crypto/rand"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	apiv1 "k8s.io/client-go/pkg/api/v1"
)

func (f *Framework) Namespace() apiv1.Namespace {
	return apiv1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: rand.WithUniqSuffix("test-stash"),
		},
	}
}

func (f *Framework) CreateNamespace(obj apiv1.Namespace) error {
	_, err := f.KubeClient.CoreV1().Namespaces().Create(&obj)
	return err
}

func (f *Framework) DeleteNamespace(meta metav1.ObjectMeta) error {
	return f.KubeClient.CoreV1().Namespaces().Delete(meta.Name, &metav1.DeleteOptions{})
}

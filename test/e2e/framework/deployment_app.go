package framework

import (
	"github.com/appscode/go/crypto/rand"
	"github.com/appscode/go/types"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	apps "k8s.io/client-go/pkg/apis/apps/v1beta1"
)

func (f *Framework) DeploymentApp() apps.Deployment {
	return apps.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      rand.WithUniqSuffix("stash"),
			Namespace: f.namespace,
			Labels: map[string]string{
				"app": "stash-e2e",
			},
		},
		Spec: apps.DeploymentSpec{
			Replicas: types.Int32P(1),
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app": "nginx",
				},
			},
			Template: f.PodTemplate(),
		},
	}
}

func (f *Framework) CreateDeploymentApp(obj apps.Deployment) error {
	_, err := f.kubeClient.AppsV1beta1().Deployments(obj.Namespace).Create(&obj)
	return err
}

func (f *Framework) DeleteDeploymentApp(meta metav1.ObjectMeta) error {
	return f.kubeClient.AppsV1beta1().Deployments(meta.Namespace).Delete(meta.Name, &metav1.DeleteOptions{})
}

func (f *Framework) EventuallyDeploymentApp(meta metav1.ObjectMeta) GomegaAsyncAssertion {
	return Eventually(func() *apps.Deployment {
		obj, err := f.kubeClient.AppsV1beta1().Deployments(meta.Namespace).Get(meta.Name, metav1.GetOptions{})
		Expect(err).NotTo(HaveOccurred())
		return obj
	})
}

package framework

import (
	"fmt"
	"time"

	"github.com/appscode/go/crypto/rand"
	"github.com/appscode/go/types"
	"github.com/appscode/log"
	. "github.com/onsi/gomega"
	kerr "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	apps "k8s.io/client-go/pkg/apis/apps/v1beta1"
)

func (fi *Invocation) DeploymentApp() apps.Deployment {
	return apps.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      rand.WithUniqSuffix("stash"),
			Namespace: fi.namespace,
			Labels: map[string]string{
				"app": fi.app,
			},
		},
		Spec: apps.DeploymentSpec{
			Replicas: types.Int32P(1),
			Template: fi.PodTemplate(),
		},
	}
}

func (f *Framework) CreateDeploymentApp(obj apps.Deployment) error {
	_, err := f.kubeClient.AppsV1beta1().Deployments(obj.Namespace).Create(&obj)
	return err
}

func (f *Framework) DeleteDeploymentApp(meta metav1.ObjectMeta) error {
	return f.kubeClient.AppsV1beta1().Deployments(meta.Namespace).Delete(meta.Name, deleteInForeground())
}

func (f *Framework) UpdateDeploymentApp(meta metav1.ObjectMeta, transformer func(apps.Deployment) apps.Deployment) error {
	attempt := 0
	for ; attempt < maxAttempts; attempt = attempt + 1 {
		cur, err := f.kubeClient.AppsV1beta1().Deployments(meta.Namespace).Get(meta.Name, metav1.GetOptions{})
		if kerr.IsNotFound(err) {
			return nil
		} else if err == nil {
			modified := transformer(*cur)
			_, err = f.kubeClient.AppsV1beta1().Deployments(cur.Namespace).Update(&modified)
			if err == nil {
				return nil
			}
		}
		log.Errorf("Attempt %d failed to update Deployment %s@%s due to %s.", attempt, cur.Name, cur.Namespace, err)
		time.Sleep(updateRetryInterval)
	}
	return fmt.Errorf("Failed to update Deployment %s@%s after %d attempts.", meta.Name, meta.Namespace, attempt)
}

func (f *Framework) EventuallyDeploymentApp(meta metav1.ObjectMeta) GomegaAsyncAssertion {
	return Eventually(func() *apps.Deployment {
		obj, err := f.kubeClient.AppsV1beta1().Deployments(meta.Namespace).Get(meta.Name, metav1.GetOptions{})
		Expect(err).NotTo(HaveOccurred())
		return obj
	})
}

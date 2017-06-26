package framework

import (
	"github.com/appscode/go/crypto/rand"
	"github.com/appscode/go/types"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/types"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	extensions "k8s.io/client-go/pkg/apis/extensions/v1beta1"
)

func (f *Framework) ReplicaSet() extensions.ReplicaSet {
	return extensions.ReplicaSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      rand.WithUniqSuffix("stash"),
			Namespace: f.namespace,
			Labels: map[string]string{
				"app": "stash-e2e",
			},
		},
		Spec: extensions.ReplicaSetSpec{
			Replicas: types.Int32P(1),
			Template: f.PodTemplate(),
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app": "stash-e2e",
				},
			},
		},
	}
}

func (f *Framework) CreateReplicaSet(obj extensions.ReplicaSet) error {
	_, err := f.kubeClient.ExtensionsV1beta1().ReplicaSets(obj.Namespace).Create(&obj)
	return err
}

func (f *Framework) DeleteReplicaSet(meta metav1.ObjectMeta) error {
	return f.kubeClient.ExtensionsV1beta1().ReplicaSets(meta.Namespace).Delete(meta.Name, &metav1.DeleteOptions{})
}

func (f *Framework) WaitForReplicaSetCondition(meta metav1.ObjectMeta, condition GomegaMatcher) {
	Eventually(func() *extensions.ReplicaSet {
		obj, err := f.kubeClient.ExtensionsV1beta1().ReplicaSets(meta.Namespace).Get(meta.Name, metav1.GetOptions{})
		Expect(err).NotTo(HaveOccurred())
		return obj
	}).Should(condition)
}

func (f *Framework) WaitUntilReplicaSetCondition(meta metav1.ObjectMeta, condition GomegaMatcher) {
	Eventually(func() *extensions.ReplicaSet {
		obj, err := f.kubeClient.ExtensionsV1beta1().ReplicaSets(meta.Namespace).Get(meta.Name, metav1.GetOptions{})
		Expect(err).NotTo(HaveOccurred())
		return obj
	}).ShouldNot(condition)
}

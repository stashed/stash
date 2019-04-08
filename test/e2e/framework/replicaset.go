package framework

import (
	"github.com/appscode/go/crypto/rand"
	"github.com/appscode/go/types"
	. "github.com/onsi/gomega"
	apps "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func (fi *Invocation) ReplicaSet(pvcName string) apps.ReplicaSet {
	labels := map[string]string{
		"app":  fi.app,
		"kind": "replicaset",
	}
	return apps.ReplicaSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      rand.WithUniqSuffix("stash"),
			Namespace: fi.namespace,
			Labels:    labels,
		},
		Spec: apps.ReplicaSetSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: labels,
			},
			Replicas: types.Int32P(1),
			Template: fi.PodTemplate(labels, pvcName),
		},
	}
}

func (f *Framework) CreateReplicaSet(obj apps.ReplicaSet) (*apps.ReplicaSet, error) {
	return f.KubeClient.AppsV1().ReplicaSets(obj.Namespace).Create(&obj)
}

func (f *Framework) DeleteReplicaSet(meta metav1.ObjectMeta) error {
	return f.KubeClient.AppsV1().ReplicaSets(meta.Namespace).Delete(meta.Name, deleteInBackground())
}

func (f *Framework) EventuallyReplicaSet(meta metav1.ObjectMeta) GomegaAsyncAssertion {
	return Eventually(func() *apps.ReplicaSet {
		obj, err := f.KubeClient.AppsV1().ReplicaSets(meta.Namespace).Get(meta.Name, metav1.GetOptions{})
		Expect(err).NotTo(HaveOccurred())
		return obj
	})
}

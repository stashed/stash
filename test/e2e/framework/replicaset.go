package framework

import (
	"github.com/appscode/go/crypto/rand"
	"github.com/appscode/go/types"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	extensions "k8s.io/client-go/pkg/apis/extensions/v1beta1"
)

func (f *Framework) Replicaset(namespace string) extensions.ReplicaSet {
	return extensions.ReplicaSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      rand.WithUniqSuffix("stash"),
			Namespace: namespace,
			Labels: map[string]string{
				"app": "stash-e2e",
			},
		},
		Spec: extensions.ReplicaSetSpec{
			Replicas: types.Int32P(1),
			Template: f.PodTemplate(),
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app": "busybox",
				},
			},
		},
	}
}

func (f *Framework) CreateReplicaset(obj extensions.ReplicaSet) error {
	_, err := f.KubeClient.ExtensionsV1beta1().ReplicaSets(obj.Namespace).Create(&obj)
	return err
}

func (f *Framework) DeleteReplicaset(meta metav1.ObjectMeta) error {
	return f.KubeClient.ExtensionsV1beta1().ReplicaSets(meta.Namespace).Delete(meta.Name, &metav1.DeleteOptions{})
}

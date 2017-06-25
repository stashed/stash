package framework

import (
	"github.com/appscode/go/crypto/rand"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	extensions "k8s.io/client-go/pkg/apis/extensions/v1beta1"
)

func (f *Framework) Daemonset(namespace string) extensions.DaemonSet {
	return extensions.DaemonSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      rand.WithUniqSuffix("stash"),
			Namespace: namespace,
			Labels: map[string]string{
				"app": "stash-e2e",
			},
		},
		Spec: extensions.DaemonSetSpec{
			Template: f.PodTemplate(),
		},
	}
}

func (f *Framework) CreateDaemonset(obj extensions.DaemonSet) error {
	_, err := f.KubeClient.ExtensionsV1beta1().DaemonSets(obj.Namespace).Create(&obj)
	return err
}

func (f *Framework) DeleteDaemonset(meta metav1.ObjectMeta) error {
	return f.KubeClient.ExtensionsV1beta1().DaemonSets(meta.Namespace).Delete(meta.Name, &metav1.DeleteOptions{})
}

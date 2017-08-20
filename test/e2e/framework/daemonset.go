package framework

import (
	"github.com/appscode/go/crypto/rand"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	extensions "k8s.io/client-go/pkg/apis/extensions/v1beta1"
)

func (fi *Invocation) DaemonSet() extensions.DaemonSet {
	daemon := extensions.DaemonSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      rand.WithUniqSuffix("stash"),
			Namespace: fi.namespace,
			Labels: map[string]string{
				"app": fi.app,
			},
		},
		Spec: extensions.DaemonSetSpec{
			Template: fi.PodTemplate(),
		},
	}
	if nodes, err := fi.KubeClient.CoreV1().Nodes().List(metav1.ListOptions{}); err == nil {
		if len(nodes.Items) > 0 {
			daemon.Spec.Template.Spec.NodeSelector = map[string]string{
				"kubernetes.io/hostname": nodes.Items[0].Labels["kubernetes.io/hostname"],
			}
		}
	}
	return daemon
}

func (f *Framework) CreateDaemonSet(obj extensions.DaemonSet) error {
	_, err := f.KubeClient.ExtensionsV1beta1().DaemonSets(obj.Namespace).Create(&obj)
	return err
}

func (f *Framework) DeleteDaemonSet(meta metav1.ObjectMeta) error {
	return f.KubeClient.ExtensionsV1beta1().DaemonSets(meta.Namespace).Delete(meta.Name, deleteInForeground())
}

func (f *Framework) EventuallyDaemonSet(meta metav1.ObjectMeta) GomegaAsyncAssertion {
	return Eventually(func() *extensions.DaemonSet {
		obj, err := f.KubeClient.ExtensionsV1beta1().DaemonSets(meta.Namespace).Get(meta.Name, metav1.GetOptions{})
		Expect(err).NotTo(HaveOccurred())
		return obj
	})
}

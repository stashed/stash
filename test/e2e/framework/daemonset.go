package framework

import (
	"github.com/appscode/go/crypto/rand"
	. "github.com/onsi/gomega"
	apps "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

func (fi *Invocation) DaemonSet() apps.DaemonSet {
	daemon := apps.DaemonSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      rand.WithUniqSuffix("stash"),
			Namespace: fi.namespace,
			Labels: map[string]string{
				"app": fi.app,
			},
		},
		Spec: apps.DaemonSetSpec{
			Template: fi.PodTemplate(),
			UpdateStrategy: apps.DaemonSetUpdateStrategy{
				RollingUpdate: &apps.RollingUpdateDaemonSet{MaxUnavailable: &intstr.IntOrString{IntVal: 0}},
			},
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

func (f *Framework) CreateDaemonSet(obj apps.DaemonSet) (*apps.DaemonSet, error) {
	return f.KubeClient.AppsV1().DaemonSets(obj.Namespace).Create(&obj)
}

func (f *Framework) DeleteDaemonSet(meta metav1.ObjectMeta) error {
	return f.KubeClient.AppsV1().DaemonSets(meta.Namespace).Delete(meta.Name, deleteInBackground())
}

func (f *Framework) EventuallyDaemonSet(meta metav1.ObjectMeta) GomegaAsyncAssertion {
	return Eventually(func() *apps.DaemonSet {
		obj, err := f.KubeClient.AppsV1().DaemonSets(meta.Namespace).Get(meta.Name, metav1.GetOptions{})
		Expect(err).NotTo(HaveOccurred())
		return obj
	})
}

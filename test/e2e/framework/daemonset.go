package framework

import (
	"time"

	"github.com/appscode/go/crypto/rand"
	. "github.com/onsi/gomega"
	apps "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/util/intstr"
)

func (fi *Invocation) DaemonSet(pvcName string) apps.DaemonSet {
	labels := map[string]string{
		"app":  fi.app,
		"kind": "daemonset",
	}
	daemon := apps.DaemonSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      rand.WithUniqSuffix("stash"),
			Namespace: fi.namespace,
			Labels:    labels,
		},
		Spec: apps.DaemonSetSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: labels,
			},
			Template: fi.PodTemplate(labels, pvcName),
			UpdateStrategy: apps.DaemonSetUpdateStrategy{
				RollingUpdate: &apps.RollingUpdateDaemonSet{MaxUnavailable: &intstr.IntOrString{IntVal: 1}},
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

func (f *Framework) EventuallyPodAccessible(meta metav1.ObjectMeta) GomegaAsyncAssertion {
	return Eventually(func() bool {
		labelSelector := fields.SelectorFromSet(meta.Labels)
		podList, err := f.KubeClient.CoreV1().Pods(meta.Namespace).List(metav1.ListOptions{LabelSelector: labelSelector.String()})
		Expect(err).NotTo(HaveOccurred())

		for _, pod := range podList.Items {
			_, err := f.ExecOnPod(&pod, "ls", "-R")
			if err == nil {
				return true
			}
		}
		return false
	},
		time.Minute*2,
		time.Second*2,
	)

}

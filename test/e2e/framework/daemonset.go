package framework

import (
	"fmt"
	"time"

	"github.com/appscode/go/crypto/rand"
	"github.com/appscode/log"
	. "github.com/onsi/gomega"
	kerr "k8s.io/apimachinery/pkg/api/errors"
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
	if nodes, err := fi.kubeClient.CoreV1().Nodes().List(metav1.ListOptions{}); err == nil {
		if len(nodes.Items) > 0 {
			daemon.Spec.Template.Spec.NodeSelector = map[string]string{
				"kubernetes.io/hostname": nodes.Items[0].Labels["kubernetes.io/hostname"],
			}
		}
	}
	return daemon
}

func (f *Framework) CreateDaemonSet(obj extensions.DaemonSet) error {
	_, err := f.kubeClient.ExtensionsV1beta1().DaemonSets(obj.Namespace).Create(&obj)
	return err
}

func (f *Framework) DeleteDaemonSet(meta metav1.ObjectMeta) error {
	return f.kubeClient.ExtensionsV1beta1().DaemonSets(meta.Namespace).Delete(meta.Name, deleteInForeground())
}

func (f *Framework) UpdateDaemonSet(meta metav1.ObjectMeta, transformer func(extensions.DaemonSet) extensions.DaemonSet) error {
	attempt := 0
	for ; attempt < maxAttempts; attempt = attempt + 1 {
		cur, err := f.kubeClient.ExtensionsV1beta1().DaemonSets(meta.Namespace).Get(meta.Name, metav1.GetOptions{})
		if kerr.IsNotFound(err) {
			return nil
		} else if err == nil {
			modified := transformer(*cur)
			_, err = f.kubeClient.ExtensionsV1beta1().DaemonSets(cur.Namespace).Update(&modified)
			if err == nil {
				return nil
			}
		}
		log.Errorf("Attempt %d failed to update DaemonSet %s@%s due to %s.", attempt, cur.Name, cur.Namespace, err)
		time.Sleep(updateRetryInterval)
	}
	return fmt.Errorf("Failed to update DaemonSet %s@%s after %d attempts.", meta.Name, meta.Namespace, attempt)
}

func (f *Framework) EventuallyDaemonSet(meta metav1.ObjectMeta) GomegaAsyncAssertion {
	return Eventually(func() *extensions.DaemonSet {
		obj, err := f.kubeClient.ExtensionsV1beta1().DaemonSets(meta.Namespace).Get(meta.Name, metav1.GetOptions{})
		Expect(err).NotTo(HaveOccurred())
		return obj
	})
}

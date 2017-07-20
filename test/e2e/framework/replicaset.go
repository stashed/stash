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
	extensions "k8s.io/client-go/pkg/apis/extensions/v1beta1"
)

func (fi *Invocation) ReplicaSet() extensions.ReplicaSet {
	return extensions.ReplicaSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      rand.WithUniqSuffix("stash"),
			Namespace: fi.namespace,
			Labels: map[string]string{
				"app": fi.app,
			},
		},
		Spec: extensions.ReplicaSetSpec{
			Replicas: types.Int32P(1),
			Template: fi.PodTemplate(),
		},
	}
}

func (f *Framework) CreateReplicaSet(obj extensions.ReplicaSet) error {
	_, err := f.kubeClient.ExtensionsV1beta1().ReplicaSets(obj.Namespace).Create(&obj)
	return err
}

func (f *Framework) DeleteReplicaSet(meta metav1.ObjectMeta) error {
	return f.kubeClient.ExtensionsV1beta1().ReplicaSets(meta.Namespace).Delete(meta.Name, deleteInForeground())
}

func (f *Framework) UpdateReplicaSet(meta metav1.ObjectMeta, transformer func(extensions.ReplicaSet) extensions.ReplicaSet) error {
	attempt := 0
	for ; attempt < maxAttempts; attempt = attempt + 1 {
		cur, err := f.kubeClient.ExtensionsV1beta1().ReplicaSets(meta.Namespace).Get(meta.Name, metav1.GetOptions{})
		if kerr.IsNotFound(err) {
			return nil
		} else if err == nil {
			modified := transformer(*cur)
			_, err = f.kubeClient.ExtensionsV1beta1().ReplicaSets(cur.Namespace).Update(&modified)
			if err == nil {
				return nil
			}
		}
		log.Errorf("Attempt %d failed to update ReplicaSet %s@%s due to %s.", attempt, cur.Name, cur.Namespace, err)
		time.Sleep(updateRetryInterval)
	}
	return fmt.Errorf("Failed to update ReplicaSet %s@%s after %d attempts.", meta.Name, meta.Namespace, attempt)
}

func (f *Framework) EventuallyReplicaSet(meta metav1.ObjectMeta) GomegaAsyncAssertion {
	return Eventually(func() *extensions.ReplicaSet {
		obj, err := f.kubeClient.ExtensionsV1beta1().ReplicaSets(meta.Namespace).Get(meta.Name, metav1.GetOptions{})
		Expect(err).NotTo(HaveOccurred())
		return obj
	})
}

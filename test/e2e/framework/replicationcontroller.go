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
	apiv1 "k8s.io/client-go/pkg/api/v1"
)

func (f *Invocation) ReplicationController() apiv1.ReplicationController {
	podTemplate := f.PodTemplate()
	return apiv1.ReplicationController{
		ObjectMeta: metav1.ObjectMeta{
			Name:      rand.WithUniqSuffix("stash"),
			Namespace: f.namespace,
			Labels: map[string]string{
				"app": f.app,
			},
		},
		Spec: apiv1.ReplicationControllerSpec{
			Replicas: types.Int32P(1),
			Template: &podTemplate,
		},
	}
}

func (f *Framework) CreateReplicationController(obj apiv1.ReplicationController) error {
	_, err := f.kubeClient.CoreV1().ReplicationControllers(obj.Namespace).Create(&obj)
	return err
}

func (f *Framework) UpdateReplicationController(meta metav1.ObjectMeta, transformer func(apiv1.ReplicationController) apiv1.ReplicationController) error {
	attempt := 0
	for ; attempt < maxAttempts; attempt = attempt + 1 {
		cur, err := f.kubeClient.CoreV1().ReplicationControllers(meta.Namespace).Get(meta.Name, metav1.GetOptions{})
		if kerr.IsNotFound(err) {
			return nil
		} else if err == nil {
			modified := transformer(*cur)
			_, err = f.kubeClient.CoreV1().ReplicationControllers(cur.Namespace).Update(&modified)
			if err == nil {
				return nil
			}
		}
		log.Errorf("Attempt %d failed to update ReplicationController %s@%s due to %s.", attempt, cur.Name, cur.Namespace, err)
		time.Sleep(updateRetryInterval)
	}
	return fmt.Errorf("Failed to update ReplicationController %s@%s after %d attempts.", meta.Name, meta.Namespace, attempt)
}

func (f *Framework) DeleteReplicationController(meta metav1.ObjectMeta) error {
	return f.kubeClient.CoreV1().ReplicationControllers(meta.Namespace).Delete(meta.Name, deleteInForeground())
}

func (f *Framework) EventuallyReplicationController(meta metav1.ObjectMeta) GomegaAsyncAssertion {
	return Eventually(func() *apiv1.ReplicationController {
		obj, err := f.kubeClient.CoreV1().ReplicationControllers(meta.Namespace).Get(meta.Name, metav1.GetOptions{})
		Expect(err).NotTo(HaveOccurred())
		return obj
	})
}

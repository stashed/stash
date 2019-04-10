package framework

import (
	"github.com/appscode/go/crypto/rand"
	"github.com/appscode/go/types"
	. "github.com/onsi/gomega"
	core "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func (fi *Invocation) ReplicationController(pvcName string) core.ReplicationController {
	labels := map[string]string{
		"app":  fi.app,
		"kind": "replicationcontroller",
	}
	podTemplate := fi.PodTemplate(labels, pvcName)
	return core.ReplicationController{
		ObjectMeta: metav1.ObjectMeta{
			Name:      rand.WithUniqSuffix("stash"),
			Namespace: fi.namespace,
			Labels: map[string]string{
				"app": fi.app,
			},
		},
		Spec: core.ReplicationControllerSpec{
			Replicas: types.Int32P(1),
			Template: &podTemplate,
		},
	}
}

func (f *Framework) CreateReplicationController(obj core.ReplicationController) (*core.ReplicationController, error) {
	return f.KubeClient.CoreV1().ReplicationControllers(obj.Namespace).Create(&obj)
}

func (f *Framework) DeleteReplicationController(meta metav1.ObjectMeta) error {
	return f.KubeClient.CoreV1().ReplicationControllers(meta.Namespace).Delete(meta.Name, deleteInBackground())
}

func (f *Framework) EventuallyReplicationController(meta metav1.ObjectMeta) GomegaAsyncAssertion {
	return Eventually(func() *core.ReplicationController {
		obj, err := f.KubeClient.CoreV1().ReplicationControllers(meta.Namespace).Get(meta.Name, metav1.GetOptions{})
		Expect(err).NotTo(HaveOccurred())
		return obj
	})
}

package framework

import (
	"github.com/appscode/go/crypto/rand"
	"github.com/appscode/go/types"
	. "github.com/onsi/gomega"
	apps "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func (fi *Invocation) StatefulSet() apps.StatefulSet {
	labels := map[string]string{
		"app":  fi.app,
		"kind": "statefulset",
	}
	return apps.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      rand.WithUniqSuffix("stash"),
			Namespace: fi.namespace,
			Labels:    labels,
		},
		Spec: apps.StatefulSetSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: labels,
			},
			Replicas:    types.Int32P(1),
			Template:    fi.PodTemplate(labels),
			ServiceName: TEST_HEADLESS_SERVICE,
			UpdateStrategy: apps.StatefulSetUpdateStrategy{
				Type: apps.RollingUpdateStatefulSetStrategyType,
			},
		},
	}
}

func (f *Framework) CreateStatefulSet(obj apps.StatefulSet) (*apps.StatefulSet, error) {
	return f.KubeClient.AppsV1().StatefulSets(obj.Namespace).Create(&obj)
}

func (f *Framework) DeleteStatefulSet(meta metav1.ObjectMeta) error {
	return f.KubeClient.AppsV1().StatefulSets(meta.Namespace).Delete(meta.Name, deleteInBackground())
}

func (f *Framework) EventuallyStatefulSet(meta metav1.ObjectMeta) GomegaAsyncAssertion {
	return Eventually(func() *apps.StatefulSet {
		obj, err := f.KubeClient.AppsV1().StatefulSets(meta.Namespace).Get(meta.Name, metav1.GetOptions{})
		Expect(err).NotTo(HaveOccurred())
		return obj
	})
}

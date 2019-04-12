package framework

import (
	"github.com/appscode/go/crypto/rand"
	"github.com/appscode/go/types"
	. "github.com/onsi/gomega"
	apps "k8s.io/api/apps/v1"
	core "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func (fi *Invocation) StatefulSet(pvcName string) apps.StatefulSet {
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
			Template:    fi.PodTemplate(labels, pvcName),
			ServiceName: TEST_HEADLESS_SERVICE,
			UpdateStrategy: apps.StatefulSetUpdateStrategy{
				Type: apps.RollingUpdateStatefulSetStrategyType,
			},
		},
	}
}

func (fi *Invocation) StatefulSetForV1beta1API() apps.StatefulSet {
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
			Replicas:    types.Int32P(3),
			ServiceName: TEST_HEADLESS_SERVICE,
			Template: core.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: labels,
				},
				Spec: core.PodSpec{
					Containers: []core.Container{
						{
							Name:            "busybox",
							Image:           "busybox",
							ImagePullPolicy: core.PullIfNotPresent,
							Command: []string{
								"sleep",
								"3600",
							},
							VolumeMounts: []core.VolumeMount{
								{
									Name:      TestSourceDataVolumeName,
									MountPath: TestSourceDataMountPath,
								},
							},
						},
					},
				},
			},
			VolumeClaimTemplates: []core.PersistentVolumeClaim{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: TestSourceDataVolumeName,
					},
					Spec: core.PersistentVolumeClaimSpec{
						AccessModes: []core.PersistentVolumeAccessMode{
							core.ReadWriteOnce,
						},
						StorageClassName: types.StringP(fi.StorageClass),
						Resources: core.ResourceRequirements{
							Requests: core.ResourceList{
								core.ResourceName(core.ResourceStorage): resource.MustParse("1Gi"),
							},
						},
					},
				},
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

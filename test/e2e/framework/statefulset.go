package framework

import (
	"github.com/appscode/go/crypto/rand"
	"github.com/appscode/go/types"
	api "github.com/appscode/stash/apis/stash/v1alpha1"
	"github.com/appscode/stash/pkg/util"
	. "github.com/onsi/gomega"
	apps "k8s.io/api/apps/v1beta1"
	core "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func (fi *Invocation) StatefulSet(r api.Restic, sidecarImageTag string) apps.StatefulSet {
	resource := apps.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      rand.WithUniqSuffix("stash"),
			Namespace: r.Namespace,
			Labels: map[string]string{
				"app": fi.app,
			},
		},
		Spec: apps.StatefulSetSpec{
			Replicas:    types.Int32P(1),
			Template:    fi.PodTemplate(),
			ServiceName: TEST_HEADLESS_SERVICE,
			UpdateStrategy: apps.StatefulSetUpdateStrategy{
				Type: apps.RollingUpdateStatefulSetStrategyType,
			},
		},
	}

	workload := api.LocalTypedReference{
		Kind: api.KindStatefulSet,
		Name: resource.Name,
	}
	resource.Spec.Template.Spec.Containers = append(resource.Spec.Template.Spec.Containers, util.CreateSidecarContainer(&r, sidecarImageTag, workload))
	resource.Spec.Template.Spec.Volumes = util.UpsertScratchVolume(resource.Spec.Template.Spec.Volumes)
	resource.Spec.Template.Spec.Volumes = util.UpsertDownwardVolume(resource.Spec.Template.Spec.Volumes)
	if r.Spec.Backend.Local != nil {
		resource.Spec.Template.Spec.Volumes = append(resource.Spec.Template.Spec.Volumes, core.Volume{Name: util.LocalVolumeName, VolumeSource: r.Spec.Backend.Local.VolumeSource})
	}
	return resource
}

func (fi *Invocation) StatefulSetWitInitContainer(r api.Restic, sidecarImageTag string) apps.StatefulSet {
	resource := apps.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      rand.WithUniqSuffix("stash"),
			Namespace: r.Namespace,
			Labels: map[string]string{
				"app": fi.app,
			},
		},
		Spec: apps.StatefulSetSpec{
			Replicas:    types.Int32P(1),
			Template:    fi.PodTemplate(),
			ServiceName: TEST_HEADLESS_SERVICE,
			UpdateStrategy: apps.StatefulSetUpdateStrategy{
				Type: apps.RollingUpdateStatefulSetStrategyType,
			},
		},
	}

	workload := api.LocalTypedReference{
		Kind: api.KindStatefulSet,
		Name: resource.Name,
	}
	resource.Spec.Template.Spec.InitContainers = append(resource.Spec.Template.Spec.InitContainers, util.CreateInitContainer(&r, sidecarImageTag, workload, false))
	resource.Spec.Template.Spec.Volumes = util.UpsertScratchVolume(resource.Spec.Template.Spec.Volumes)
	resource.Spec.Template.Spec.Volumes = util.UpsertDownwardVolume(resource.Spec.Template.Spec.Volumes)
	if r.Spec.Backend.Local != nil {
		resource.Spec.Template.Spec.Volumes = append(resource.Spec.Template.Spec.Volumes, core.Volume{Name: util.LocalVolumeName, VolumeSource: r.Spec.Backend.Local.VolumeSource})
	}
	return resource
}

func (f *Framework) CreateStatefulSet(obj apps.StatefulSet) (*apps.StatefulSet, error) {
	return f.KubeClient.AppsV1beta1().StatefulSets(obj.Namespace).Create(&obj)
}

func (f *Framework) DeleteStatefulSet(meta metav1.ObjectMeta) error {
	return f.KubeClient.AppsV1beta1().StatefulSets(meta.Namespace).Delete(meta.Name, deleteInForeground())
}

func (f *Framework) EventuallyStatefulSet(meta metav1.ObjectMeta) GomegaAsyncAssertion {
	return Eventually(func() *apps.StatefulSet {
		obj, err := f.KubeClient.AppsV1beta1().StatefulSets(meta.Namespace).Get(meta.Name, metav1.GetOptions{})
		Expect(err).NotTo(HaveOccurred())
		return obj
	})
}

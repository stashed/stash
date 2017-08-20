package framework

import (
	"github.com/appscode/go/crypto/rand"
	"github.com/appscode/go/types"
	sapi "github.com/appscode/stash/api"
	"github.com/appscode/stash/pkg/util"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	apiv1 "k8s.io/client-go/pkg/api/v1"
	apps "k8s.io/client-go/pkg/apis/apps/v1beta1"
)

func (fi *Invocation) StatefulSet(r sapi.Restic) apps.StatefulSet {
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
		},
	}
	resource.Spec.Template.Spec.Containers = append(resource.Spec.Template.Spec.Containers, util.CreateSidecarContainer(&r, "canary", "ss/"+resource.Name))
	resource.Spec.Template.Spec.Volumes = util.UpsertScratchVolume(resource.Spec.Template.Spec.Volumes)
	resource.Spec.Template.Spec.Volumes = util.UpsertDownwardVolume(resource.Spec.Template.Spec.Volumes)
	if r.Spec.Backend.Local != nil {
		resource.Spec.Template.Spec.Volumes = append(resource.Spec.Template.Spec.Volumes, apiv1.Volume{Name: util.LocalVolumeName, VolumeSource: r.Spec.Backend.Local.VolumeSource})
	}
	return resource
}

func (f *Framework) CreateStatefulSet(obj apps.StatefulSet) error {
	_, err := f.KubeClient.AppsV1beta1().StatefulSets(obj.Namespace).Create(&obj)
	return err
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

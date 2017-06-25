package framework

import (
	"github.com/appscode/go/crypto/rand"
	"github.com/appscode/go/types"
	sapi "github.com/appscode/stash/api"
	"github.com/appscode/stash/pkg/util"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	apps "k8s.io/client-go/pkg/apis/apps/v1beta1"
)

func (f *Framework) StatefulSet(r sapi.Restic) apps.StatefulSet {
	resource := apps.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      rand.WithUniqSuffix("stash"),
			Namespace: r.Namespace,
			Labels: map[string]string{
				"app": "stash-e2e",
			},
		},
		Spec: apps.StatefulSetSpec{
			Replicas:    types.Int32P(1),
			Template:    f.PodTemplate(),
			ServiceName: TEST_HEADLESS_SERVICE,
		},
	}
	resource.Spec.Template.Spec.Containers = append(resource.Spec.Template.Spec.Containers, util.GetSidecarContainer(&r, "canary", resource.Name, true))
	resource.Spec.Template.Spec.Volumes = util.AddScratchVolume(resource.Spec.Template.Spec.Volumes)
	resource.Spec.Template.Spec.Volumes = util.AddDownwardVolume(resource.Spec.Template.Spec.Volumes)
	if r.Spec.Backend.Local != nil {
		resource.Spec.Template.Spec.Volumes = append(resource.Spec.Template.Spec.Volumes, r.Spec.Backend.Local.Volume)
	}
	return resource
}

func (f *Framework) CreateStatefulSet(obj apps.StatefulSet) error {
	_, err := f.kubeClient.AppsV1beta1().StatefulSets(obj.Namespace).Create(&obj)
	return err
}

func (f *Framework) DeleteStatefulset(meta metav1.ObjectMeta) error {
	return f.kubeClient.AppsV1beta1().StatefulSets(meta.Namespace).Delete(meta.Name, &metav1.DeleteOptions{})
}

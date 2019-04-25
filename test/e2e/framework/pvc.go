package framework

import (
	"github.com/appscode/go/crypto/rand"
	core "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func (f *Invocation) GetPersistentVolumeClaim() *core.PersistentVolumeClaim {
	return &core.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name:      rand.WithUniqSuffix("pvc"),
			Namespace: f.namespace,
		},
		Spec: core.PersistentVolumeClaimSpec{
			AccessModes: []core.PersistentVolumeAccessMode{
				core.ReadWriteOnce,
			},
			StorageClassName: &f.StorageClass,
			Resources: core.ResourceRequirements{
				Requests: core.ResourceList{
					core.ResourceName(core.ResourceStorage): resource.MustParse("10Mi"),
				},
			},
		},
	}
}

func (f *Invocation) CreatePersistentVolumeClaim(pvc *core.PersistentVolumeClaim) error {
	_, err := f.KubeClient.CoreV1().PersistentVolumeClaims(pvc.Namespace).Create(pvc)
	return err
}

func (f *Invocation) DeletePersistentVolumeClaim(meta metav1.ObjectMeta) error {
	return f.KubeClient.CoreV1().PersistentVolumeClaims(meta.Namespace).Delete(meta.Name, deleteInForeground())
}

func (f *Invocation) GetPersistentVolume() *core.PersistentVolume {
	labels := map[string]string{
		"app":  f.app,
		"kind": "deployment",
		"type": "local",
	}
	return &core.PersistentVolume{
		ObjectMeta: metav1.ObjectMeta{
			Name:      rand.WithUniqSuffix("pv"),
			Namespace: f.namespace,
			Labels:    labels,
		},
		Spec: core.PersistentVolumeSpec{
			AccessModes: []core.PersistentVolumeAccessMode{
				core.ReadWriteOnce,
			},
			Capacity: core.ResourceList{
				core.ResourceName(core.ResourceStorage): resource.MustParse("10Mi"),
			},
			PersistentVolumeSource: core.PersistentVolumeSource{
				HostPath: &core.HostPathVolumeSource{
					Path: "/tmp/data",
				},
			},
		},
	}
}

func (f *Invocation) CreatePersistentVolume(pv *core.PersistentVolume) error {
	_, err := f.KubeClient.CoreV1().PersistentVolumes().Create(pv)
	return err
}

func (f *Invocation) DeletePersistentVolume(meta metav1.ObjectMeta) error {
	return f.KubeClient.CoreV1().PersistentVolumes().Delete(meta.Name, &metav1.DeleteOptions{})
}

package framework

import (
	"fmt"

	"github.com/appscode/go/crypto/rand"
	. "github.com/onsi/ginkgo"
	core "k8s.io/api/core/v1"
	kerr "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func (f *Invocation) PersistentVolumeClaim(name string) *core.PersistentVolumeClaim {
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

func (f *Framework) CreatePersistentVolumeClaim(pvc *core.PersistentVolumeClaim) (*core.PersistentVolumeClaim, error) {
	return f.KubeClient.CoreV1().PersistentVolumeClaims(pvc.Namespace).Create(pvc)
}

func (f *Invocation) DeletePersistentVolumeClaim(meta metav1.ObjectMeta) error {
	err := f.KubeClient.CoreV1().PersistentVolumeClaims(meta.Namespace).Delete(meta.Name, deleteInForeground())
	if err != nil && !kerr.IsNotFound(err) {
		return err
	}
	return nil
}

func (f *Invocation) CreateNewPVC(name string) (*core.PersistentVolumeClaim, error) {
	// Generate PVC definition
	pvc := f.PersistentVolumeClaim(name)

	By(fmt.Sprintf("Creating PVC: %s/%s", pvc.Namespace, pvc.Name))
	createdPVC, err := f.CreatePersistentVolumeClaim(pvc)
	f.AppendToCleanupList(createdPVC)

	return createdPVC, err
}

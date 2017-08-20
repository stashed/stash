package framework

import (
	"fmt"
	"time"

	"github.com/appscode/go/crypto/rand"
	"github.com/appscode/log"
	tapi "github.com/appscode/stash/api"
	. "github.com/onsi/gomega"
	kerr "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	apiv1 "k8s.io/client-go/pkg/api/v1"
)

func (fi *Invocation) _restic() tapi.Restic {
	return tapi.Restic{
		TypeMeta: metav1.TypeMeta{
			APIVersion: tapi.SchemeGroupVersion.String(),
			Kind:       tapi.ResourceKindRestic,
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      rand.WithUniqSuffix("stash"),
			Namespace: fi.namespace,
		},
		Spec: tapi.ResticSpec{
			Selector: metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app": fi.app,
				},
			},
			FileGroups: []tapi.FileGroup{
				{
					Path: TestSourceDataMountPath,
					RetentionPolicy: tapi.RetentionPolicy{
						KeepLast: 5,
					},
				},
			},
			Schedule: "@every 15s",
			VolumeMounts: []apiv1.VolumeMount{
				{
					Name:      TestSourceDataVolumeName,
					MountPath: TestSourceDataMountPath,
				},
			},
		},
	}
}

func (fi *Invocation) ResticForLocalBackend() tapi.Restic {
	r := fi._restic()
	r.Spec.Backend = tapi.Backend{
		StorageSecretName: "",
		Local: &tapi.LocalSpec{
			Path: "/safe/data",
			VolumeSource: apiv1.VolumeSource{
				EmptyDir: &apiv1.EmptyDirVolumeSource{},
			},
		},
	}
	return r
}

func (fi *Invocation) ResticForS3Backend() tapi.Restic {
	r := fi._restic()
	r.Spec.Backend = tapi.Backend{
		StorageSecretName: "",
		S3: &tapi.S3Spec{
			Endpoint: "s3.amazonaws.com",
			Bucket:   "stash-qa",
			Prefix:   fi.app,
		},
	}
	return r
}

func (fi *Invocation) ResticForGCSBackend() tapi.Restic {
	r := fi._restic()
	r.Spec.Backend = tapi.Backend{
		StorageSecretName: "",
		GCS: &tapi.GCSSpec{
			Bucket: "stash-qa",
			Prefix: fi.app,
		},
	}
	return r
}

func (fi *Invocation) ResticForAzureBackend() tapi.Restic {
	r := fi._restic()
	r.Spec.Backend = tapi.Backend{
		StorageSecretName: "",
		Azure: &tapi.AzureSpec{
			Container: "stashqa",
			Prefix:    fi.app,
		},
	}
	return r
}

func (fi *Invocation) ResticForSwiftBackend() tapi.Restic {
	r := fi._restic()
	r.Spec.Backend = tapi.Backend{
		StorageSecretName: "",
		Swift: &tapi.SwiftSpec{
			Container: "stash-qa",
			Prefix:    fi.app,
		},
	}
	return r
}

func (f *Framework) CreateRestic(obj tapi.Restic) error {
	_, err := f.StashClient.Restics(obj.Namespace).Create(&obj)
	return err
}

func (f *Framework) DeleteRestic(meta metav1.ObjectMeta) error {
	return f.StashClient.Restics(meta.Namespace).Delete(meta.Name, deleteInForeground())
}

func (f *Framework) UpdateRestic(meta metav1.ObjectMeta, transformer func(tapi.Restic) tapi.Restic) error {
	attempt := 0
	for ; attempt < maxAttempts; attempt = attempt + 1 {
		cur, err := f.StashClient.Restics(meta.Namespace).Get(meta.Name)
		if kerr.IsNotFound(err) {
			return nil
		} else if err == nil {
			modified := transformer(*cur)
			_, err = f.StashClient.Restics(cur.Namespace).Update(&modified)
			if err == nil {
				return nil
			}
		}
		log.Errorf("Attempt %d failed to update Restic %s@%s due to %s.", attempt, cur.Name, cur.Namespace, err)
		time.Sleep(updateRetryInterval)
	}
	return fmt.Errorf("Failed to update Restic %s@%s after %d attempts.", meta.Name, meta.Namespace, attempt)
}

func (f *Framework) EventuallyRestic(meta metav1.ObjectMeta) GomegaAsyncAssertion {
	return Eventually(func() *tapi.Restic {
		obj, err := f.StashClient.Restics(meta.Namespace).Get(meta.Name)
		Expect(err).NotTo(HaveOccurred())
		return obj
	})
}

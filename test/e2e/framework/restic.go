package framework

import (
	"fmt"
	"time"

	"github.com/appscode/go/crypto/rand"
	"github.com/appscode/log"
	sapi "github.com/appscode/stash/api"
	"github.com/appscode/stash/client/clientset"
	. "github.com/onsi/gomega"
	kerr "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	apiv1 "k8s.io/client-go/pkg/api/v1"
)

func (f *Invocation) _restic() sapi.Restic {
	return sapi.Restic{
		TypeMeta: metav1.TypeMeta{
			APIVersion: sapi.SchemeGroupVersion.String(),
			Kind:       clientset.ResourceKindRestic,
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      rand.WithUniqSuffix("stash"),
			Namespace: f.namespace,
		},
		Spec: sapi.ResticSpec{
			Selector: metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app": f.app,
				},
			},
			FileGroups: []sapi.FileGroup{
				{
					Path: TestSourceDataMountPath,
					RetentionPolicy: sapi.RetentionPolicy{
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

func (f *Invocation) ResticForLocalBackend() sapi.Restic {
	r := f._restic()
	r.Spec.Backend = sapi.Backend{
		RepositorySecretName: "",
		Local: &sapi.LocalSpec{
			Path: "/safe/data",
			VolumeSource: apiv1.VolumeSource{
				EmptyDir: &apiv1.EmptyDirVolumeSource{},
			},
		},
	}
	return r
}

func (f *Invocation) ResticForS3Backend() sapi.Restic {
	r := f._restic()
	r.Spec.Backend = sapi.Backend{
		RepositorySecretName: "",
		S3: &sapi.S3Spec{
			Endpoint: "s3.amazonaws.com",
			Bucket:   "stash-qa",
			Prefix:   f.app,
		},
	}
	return r
}

func (f *Invocation) ResticForGCSBackend() sapi.Restic {
	r := f._restic()
	r.Spec.Backend = sapi.Backend{
		RepositorySecretName: "",
		GCS: &sapi.GCSSpec{
			Bucket: "stash-qa",
			Prefix: f.app,
		},
	}
	return r
}

func (f *Invocation) ResticForAzureBackend() sapi.Restic {
	r := f._restic()
	r.Spec.Backend = sapi.Backend{
		RepositorySecretName: "",
		Azure: &sapi.AzureSpec{
			Container: "stashqa",
			Prefix:    f.app,
		},
	}
	return r
}

func (f *Invocation) ResticForSwiftBackend() sapi.Restic {
	r := f._restic()
	r.Spec.Backend = sapi.Backend{
		RepositorySecretName: "",
		Swift: &sapi.SwiftSpec{
			Container: "stash-qa",
			Prefix:    f.app,
		},
	}
	return r
}

func (f *Framework) CreateRestic(obj sapi.Restic) error {
	_, err := f.stashClient.Restics(obj.Namespace).Create(&obj)
	return err
}

func (f *Framework) DeleteRestic(meta metav1.ObjectMeta) error {
	return f.stashClient.Restics(meta.Namespace).Delete(meta.Name, deleteInForeground())
}

func (f *Framework) UpdateRestic(meta metav1.ObjectMeta, transformer func(sapi.Restic) sapi.Restic) error {
	attempt := 0
	for ; attempt < maxAttempts; attempt = attempt + 1 {
		cur, err := f.stashClient.Restics(meta.Namespace).Get(meta.Name)
		if kerr.IsNotFound(err) {
			return nil
		} else if err == nil {
			modified := transformer(*cur)
			_, err = f.stashClient.Restics(cur.Namespace).Update(&modified)
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
	return Eventually(func() *sapi.Restic {
		obj, err := f.stashClient.Restics(meta.Namespace).Get(meta.Name)
		Expect(err).NotTo(HaveOccurred())
		return obj
	})
}

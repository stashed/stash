package framework

import (
	"fmt"
	"time"

	"github.com/appscode/go/crypto/rand"
	"github.com/appscode/go/log"
	api "github.com/appscode/stash/apis/stash/v1alpha1"
	. "github.com/onsi/gomega"
	kerr "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	apiv1 "k8s.io/client-go/pkg/api/v1"
)

func (fi *Invocation) _restic() api.Restic {
	return api.Restic{
		TypeMeta: metav1.TypeMeta{
			APIVersion: api.SchemeGroupVersion.String(),
			Kind:       api.ResourceKindRestic,
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      rand.WithUniqSuffix("stash"),
			Namespace: fi.namespace,
		},
		Spec: api.ResticSpec{
			Selector: metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app": fi.app,
				},
			},
			FileGroups: []api.FileGroup{
				{
					Path: TestSourceDataMountPath,
					RetentionPolicy: api.RetentionPolicy{
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

func (fi *Invocation) ResticForLocalBackend() api.Restic {
	r := fi._restic()
	r.Spec.Backend = api.Backend{
		StorageSecretName: "",
		Local: &api.LocalSpec{
			Path: "/safe/data",
			VolumeSource: apiv1.VolumeSource{
				EmptyDir: &apiv1.EmptyDirVolumeSource{},
			},
		},
	}
	return r
}

func (fi *Invocation) ResticForS3Backend() api.Restic {
	r := fi._restic()
	r.Spec.Backend = api.Backend{
		StorageSecretName: "",
		S3: &api.S3Spec{
			Endpoint: "s3.amazonaws.com",
			Bucket:   "stash-qa",
			Prefix:   fi.app,
		},
	}
	return r
}

func (fi *Invocation) ResticForGCSBackend() api.Restic {
	r := fi._restic()
	r.Spec.Backend = api.Backend{
		StorageSecretName: "",
		GCS: &api.GCSSpec{
			Bucket: "stash-qa",
			Prefix: fi.app,
		},
	}
	return r
}

func (fi *Invocation) ResticForAzureBackend() api.Restic {
	r := fi._restic()
	r.Spec.Backend = api.Backend{
		StorageSecretName: "",
		Azure: &api.AzureSpec{
			Container: "stashqa",
			Prefix:    fi.app,
		},
	}
	return r
}

func (fi *Invocation) ResticForSwiftBackend() api.Restic {
	r := fi._restic()
	r.Spec.Backend = api.Backend{
		StorageSecretName: "",
		Swift: &api.SwiftSpec{
			Container: "stash-qa",
			Prefix:    fi.app,
		},
	}
	return r
}

func (f *Framework) CreateRestic(obj api.Restic) error {
	_, err := f.StashClient.Restics(obj.Namespace).Create(&obj)
	return err
}

func (f *Framework) DeleteRestic(meta metav1.ObjectMeta) error {
	return f.StashClient.Restics(meta.Namespace).Delete(meta.Name, deleteInForeground())
}

func (f *Framework) UpdateRestic(meta metav1.ObjectMeta, transformer func(api.Restic) api.Restic) error {
	attempt := 0
	for ; attempt < maxAttempts; attempt = attempt + 1 {
		cur, err := f.StashClient.Restics(meta.Namespace).Get(meta.Name, metav1.GetOptions{})
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
	return Eventually(func() *api.Restic {
		obj, err := f.StashClient.Restics(meta.Namespace).Get(meta.Name, metav1.GetOptions{})
		Expect(err).NotTo(HaveOccurred())
		return obj
	})
}

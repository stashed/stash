package framework

import (
	"github.com/appscode/go/crypto/rand"
	. "github.com/onsi/gomega"
	core "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	store "kmodules.xyz/objectstore-api/api/v1"
	api "stash.appscode.dev/stash/apis/stash/v1alpha1"
	stash_util "stash.appscode.dev/stash/client/clientset/versioned/typed/stash/v1alpha1/util"
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
					Path:                TestSourceDataMountPath,
					RetentionPolicyName: "keep-last-5",
				},
			},
			Schedule: "@every 15s",
			VolumeMounts: []core.VolumeMount{
				{
					Name:      TestSourceDataVolumeName,
					MountPath: TestSourceDataMountPath,
				},
			},
			RetentionPolicies: []api.RetentionPolicy{
				{
					Name:     "keep-last-5",
					KeepLast: 5,
				},
			},
		},
	}
}

func (fi *Invocation) ResticForLocalBackend() api.Restic {
	r := fi._restic()
	r.Spec.Backend = store.Backend{
		StorageSecretName: "",
		Local: &store.LocalSpec{
			MountPath: "/safe/data",
			VolumeSource: core.VolumeSource{
				EmptyDir: &core.EmptyDirVolumeSource{},
			},
		},
	}
	return r
}

func (fi *Invocation) ResticForHostPathLocalBackend() api.Restic {
	r := fi._restic()
	r.Spec.Backend = store.Backend{
		StorageSecretName: "",
		Local: &store.LocalSpec{
			MountPath: "/safe/data",
			VolumeSource: core.VolumeSource{
				HostPath: &core.HostPathVolumeSource{
					Path: "/data/stash-test/restic-repo",
				},
			},
		},
	}
	return r
}

func (fi *Invocation) ResticForS3Backend() api.Restic {
	r := fi._restic()
	r.Spec.Backend = store.Backend{
		StorageSecretName: "",
		S3: &store.S3Spec{
			Endpoint: "s3.amazonaws.com",
			Bucket:   "stash-qa",
			Prefix:   fi.app,
		},
	}
	return r
}

func (fi *Invocation) ResticForMinioBackend(address string) api.Restic {
	r := fi._restic()
	r.Spec.Backend = store.Backend{
		StorageSecretName: "",
		S3: &store.S3Spec{
			Endpoint: address,
			Bucket:   "stash-qa",
			Prefix:   fi.app,
		},
	}
	return r
}

func (fi *Invocation) ResticForDOBackend() api.Restic {
	r := fi._restic()
	r.Spec.Backend = store.Backend{
		StorageSecretName: "",
		S3: &store.S3Spec{
			Endpoint: "nyc3.digitaloceanspaces.com",
			Bucket:   "stash-qa",
			Prefix:   fi.app,
		},
	}
	return r
}

func (fi *Invocation) ResticForGCSBackend() api.Restic {
	r := fi._restic()
	r.Spec.Backend = store.Backend{
		StorageSecretName: "",
		GCS: &store.GCSSpec{
			Bucket: "stash-qa",
			Prefix: fi.app,
		},
	}
	return r
}

func (fi *Invocation) ResticForAzureBackend() api.Restic {
	r := fi._restic()
	r.Spec.Backend = store.Backend{
		StorageSecretName: "",
		Azure: &store.AzureSpec{
			Container: "stashqa",
			Prefix:    fi.app,
		},
	}
	return r
}

func (fi *Invocation) ResticForSwiftBackend() api.Restic {
	r := fi._restic()
	r.Spec.Backend = store.Backend{
		StorageSecretName: "",
		Swift: &store.SwiftSpec{
			Container: "stash-qa",
			Prefix:    fi.app,
		},
	}
	return r
}

func (fi *Invocation) ResticForB2Backend() api.Restic {
	r := fi._restic()
	r.Spec.Backend = store.Backend{
		StorageSecretName: "",
		B2: &store.B2Spec{
			Bucket: "stash-qa",
			Prefix: fi.app,
		},
	}
	return r
}

func (f *Framework) CreateRestic(obj api.Restic) error {
	_, err := f.StashClient.StashV1alpha1().Restics(obj.Namespace).Create(&obj)
	return err
}

func (f *Framework) DeleteRestic(meta metav1.ObjectMeta) error {
	return f.StashClient.StashV1alpha1().Restics(meta.Namespace).Delete(meta.Name, deleteInForeground())
}

func (f *Framework) UpdateRestic(meta metav1.ObjectMeta, transformer func(*api.Restic) *api.Restic) error {
	_, err := stash_util.TryUpdateRestic(f.StashClient.StashV1alpha1(), meta, transformer)
	return err
}

func (f *Framework) CreateOrPatchRestic(meta metav1.ObjectMeta, transformer func(*api.Restic) *api.Restic) error {
	_, _, err := stash_util.CreateOrPatchRestic(f.StashClient.StashV1alpha1(), meta, transformer)
	return err

}

func (f *Framework) CreateOrPatchRepository(meta metav1.ObjectMeta, transformer func(repository *api.Repository) *api.Repository) error {
	_, _, err := stash_util.CreateOrPatchRepository(f.StashClient.StashV1alpha1(), meta, transformer)
	return err

}

func (f *Framework) EventuallyRestic(meta metav1.ObjectMeta) GomegaAsyncAssertion {
	return Eventually(func() *api.Restic {
		obj, err := f.StashClient.StashV1alpha1().Restics(meta.Namespace).Get(meta.Name, metav1.GetOptions{})
		Expect(err).NotTo(HaveOccurred())
		return obj
	})
}

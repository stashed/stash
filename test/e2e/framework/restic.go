package framework

import (
	"github.com/appscode/go/crypto/rand"
	api "github.com/appscode/stash/apis/stash/v1alpha1"
	stash_util "github.com/appscode/stash/client/typed/stash/v1alpha1/util"
	. "github.com/onsi/gomega"
	core "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func (fi *Invocation) _backup() api.Backup {
	return api.Backup{
		TypeMeta: metav1.TypeMeta{
			APIVersion: api.SchemeGroupVersion.String(),
			Kind:       api.ResourceKindBackup,
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      rand.WithUniqSuffix("stash"),
			Namespace: fi.namespace,
		},
		Spec: api.BackupSpec{
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

func (fi *Invocation) BackupForLocalBackend() api.Backup {
	r := fi._backup()
	r.Spec.Backend = api.Backend{
		StorageSecretName: "",
		Local: &api.LocalSpec{
			MountPath: "/safe/data",
			VolumeSource: core.VolumeSource{
				EmptyDir: &core.EmptyDirVolumeSource{},
			},
		},
	}
	return r
}

func (fi *Invocation) BackupForHostPathLocalBackend() api.Backup {
	r := fi._backup()
	r.Spec.Backend = api.Backend{
		StorageSecretName: "",
		Local: &api.LocalSpec{
			MountPath: "/safe/data",
			VolumeSource: core.VolumeSource{
				HostPath: &core.HostPathVolumeSource{
					Path: "/data/stash-test/backup-repo",
				},
			},
		},
	}
	return r
}

func (fi *Invocation) BackupForS3Backend() api.Backup {
	r := fi._backup()
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

func (fi *Invocation) BackupForMinioBackend(address string) api.Backup {
	r := fi._backup()
	r.Spec.Backend = api.Backend{
		StorageSecretName: "",
		S3: &api.S3Spec{
			Endpoint: address,
			Bucket:   "stash-qa",
			Prefix:   fi.app,
		},
	}
	return r
}

func (fi *Invocation) BackupForDOBackend() api.Backup {
	r := fi._backup()
	r.Spec.Backend = api.Backend{
		StorageSecretName: "",
		S3: &api.S3Spec{
			Endpoint: "nyc3.digitaloceanspaces.com",
			Bucket:   "stash-qa",
			Prefix:   fi.app,
		},
	}
	return r
}

func (fi *Invocation) BackupForGCSBackend() api.Backup {
	r := fi._backup()
	r.Spec.Backend = api.Backend{
		StorageSecretName: "",
		GCS: &api.GCSSpec{
			Bucket: "stash-qa",
			Prefix: fi.app,
		},
	}
	return r
}

func (fi *Invocation) BackupForAzureBackend() api.Backup {
	r := fi._backup()
	r.Spec.Backend = api.Backend{
		StorageSecretName: "",
		Azure: &api.AzureSpec{
			Container: "stashqa",
			Prefix:    fi.app,
		},
	}
	return r
}

func (fi *Invocation) BackupForSwiftBackend() api.Backup {
	r := fi._backup()
	r.Spec.Backend = api.Backend{
		StorageSecretName: "",
		Swift: &api.SwiftSpec{
			Container: "stash-qa",
			Prefix:    fi.app,
		},
	}
	return r
}

func (fi *Invocation) BackupForB2Backend() api.Backup {
	r := fi._backup()
	r.Spec.Backend = api.Backend{
		StorageSecretName: "",
		B2: &api.B2Spec{
			Bucket: "stash-qa",
			Prefix: fi.app,
		},
	}
	return r
}

func (f *Framework) CreateBackup(obj api.Backup) error {
	_, err := f.StashClient.StashV1alpha1().Backups(obj.Namespace).Create(&obj)
	return err
}

func (f *Framework) DeleteBackup(meta metav1.ObjectMeta) error {
	return f.StashClient.StashV1alpha1().Backups(meta.Namespace).Delete(meta.Name, deleteInForeground())
}

func (f *Framework) UpdateBackup(meta metav1.ObjectMeta, transformer func(*api.Backup) *api.Backup) error {
	_, err := stash_util.TryUpdateBackup(f.StashClient.StashV1alpha1(), meta, transformer)
	return err
}

func (f *Framework) CreateOrPatchBackup(meta metav1.ObjectMeta, transformer func(*api.Backup) *api.Backup) error {
	_, _, err := stash_util.CreateOrPatchBackup(f.StashClient.StashV1alpha1(), meta, transformer)
	return err

}

func (f *Framework) EventuallyBackup(meta metav1.ObjectMeta) GomegaAsyncAssertion {
	return Eventually(func() *api.Backup {
		obj, err := f.StashClient.StashV1alpha1().Backups(meta.Namespace).Get(meta.Name, metav1.GetOptions{})
		Expect(err).NotTo(HaveOccurred())
		return obj
	})
}

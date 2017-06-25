package framework

import (
	"github.com/appscode/go/crypto/rand"
	"github.com/appscode/stash/pkg/cli"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	apiv1 "k8s.io/client-go/pkg/api/v1"
)

const (
	TEST_RESTIC_PASSWORD = "not@secret"
)

func (f *Framework) SecretForLocalBackend(namespace string) apiv1.Secret {
	return apiv1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      rand.WithUniqSuffix("gcs"),
			Namespace: namespace,
		},
		Data: map[string][]byte{
			cli.RESTIC_PASSWORD: []byte(TEST_RESTIC_PASSWORD),
		},
	}
}

func (f *Framework) SecretForS3Backend(namespace string) apiv1.Secret {
	return apiv1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      rand.WithUniqSuffix("gcs"),
			Namespace: namespace,
		},
		Data: map[string][]byte{
			cli.RESTIC_PASSWORD: []byte(TEST_RESTIC_PASSWORD),
		},
	}
}

func (f *Framework) SecretForGCSBackend(namespace string) apiv1.Secret {
	return apiv1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      rand.WithUniqSuffix("gcs"),
			Namespace: namespace,
		},
		Data: map[string][]byte{
			cli.RESTIC_PASSWORD: []byte(TEST_RESTIC_PASSWORD),
		},
	}
}

func (f *Framework) SecretForAzureBackend(namespace string) apiv1.Secret {
	return apiv1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      rand.WithUniqSuffix("gcs"),
			Namespace: namespace,
		},
		Data: map[string][]byte{
			cli.RESTIC_PASSWORD: []byte(TEST_RESTIC_PASSWORD),
		},
	}
}

// TODO: Add more methods for Swift, Backblaze B2, Rest server backend.

func (f *Framework) CreateSecret(obj apiv1.Secret) error {
	_, err := f.KubeClient.CoreV1().Secrets(obj.Namespace).Create(&obj)
	return err
}

func (f *Framework) DeleteSecret(meta metav1.ObjectMeta) error {
	return f.KubeClient.CoreV1().Secrets(meta.Namespace).Delete(meta.Name, &metav1.DeleteOptions{})
}

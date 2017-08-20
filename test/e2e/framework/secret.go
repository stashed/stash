package framework

import (
	"io/ioutil"
	"os"

	"github.com/appscode/go/crypto/rand"
	"github.com/appscode/stash/pkg/cli"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	apiv1 "k8s.io/client-go/pkg/api/v1"
)

const (
	TEST_RESTIC_PASSWORD = "not@secret"
)

func (fi *Invocation) SecretForLocalBackend() apiv1.Secret {
	return apiv1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      rand.WithUniqSuffix(fi.app + "-local"),
			Namespace: fi.namespace,
		},
		Data: map[string][]byte{
			cli.RESTIC_PASSWORD: []byte(TEST_RESTIC_PASSWORD),
		},
	}
}

func (fi *Invocation) SecretForS3Backend() apiv1.Secret {
	if os.Getenv(cli.AWS_ACCESS_KEY_ID) == "" ||
		os.Getenv(cli.AWS_SECRET_ACCESS_KEY) == "" {
		return apiv1.Secret{}
	}

	return apiv1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      rand.WithUniqSuffix(fi.app + "-s3"),
			Namespace: fi.namespace,
		},
		Data: map[string][]byte{
			cli.RESTIC_PASSWORD:       []byte(TEST_RESTIC_PASSWORD),
			cli.AWS_ACCESS_KEY_ID:     []byte(os.Getenv(cli.AWS_ACCESS_KEY_ID)),
			cli.AWS_SECRET_ACCESS_KEY: []byte(os.Getenv(cli.AWS_SECRET_ACCESS_KEY)),
		},
	}
}

func (fi *Invocation) SecretForGCSBackend() apiv1.Secret {
	if os.Getenv(cli.GOOGLE_PROJECT_ID) == "" ||
		(os.Getenv(cli.GOOGLE_APPLICATION_CREDENTIALS) == "" && os.Getenv(cli.GOOGLE_SERVICE_ACCOUNT_JSON_KEY) == "") {
		return apiv1.Secret{}
	}

	jsonKey := os.Getenv(cli.GOOGLE_SERVICE_ACCOUNT_JSON_KEY)
	if jsonKey == "" {
		if keyBytes, err := ioutil.ReadFile(os.Getenv(cli.GOOGLE_APPLICATION_CREDENTIALS)); err == nil {
			jsonKey = string(keyBytes)
		}
	}
	return apiv1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      rand.WithUniqSuffix(fi.app + "-gcs"),
			Namespace: fi.namespace,
		},
		Data: map[string][]byte{
			cli.RESTIC_PASSWORD:                 []byte(TEST_RESTIC_PASSWORD),
			cli.GOOGLE_PROJECT_ID:               []byte(os.Getenv(cli.GOOGLE_PROJECT_ID)),
			cli.GOOGLE_SERVICE_ACCOUNT_JSON_KEY: []byte(jsonKey),
		},
	}
}

func (fi *Invocation) SecretForAzureBackend() apiv1.Secret {
	if os.Getenv(cli.AZURE_ACCOUNT_NAME) == "" ||
		os.Getenv(cli.AZURE_ACCOUNT_KEY) == "" {
		return apiv1.Secret{}
	}

	return apiv1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      rand.WithUniqSuffix(fi.app + "-azure"),
			Namespace: fi.namespace,
		},
		Data: map[string][]byte{
			cli.RESTIC_PASSWORD:    []byte(TEST_RESTIC_PASSWORD),
			cli.AZURE_ACCOUNT_NAME: []byte(os.Getenv(cli.AZURE_ACCOUNT_NAME)),
			cli.AZURE_ACCOUNT_KEY:  []byte(os.Getenv(cli.AZURE_ACCOUNT_KEY)),
		},
	}
}

func (fi *Invocation) SecretForSwiftBackend() apiv1.Secret {
	if os.Getenv(cli.OS_AUTH_URL) == "" ||
		(os.Getenv(cli.OS_TENANT_ID) == "" && os.Getenv(cli.OS_TENANT_NAME) == "") ||
		os.Getenv(cli.OS_USERNAME) == "" ||
		os.Getenv(cli.OS_PASSWORD) == "" {
		return apiv1.Secret{}
	}

	return apiv1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      rand.WithUniqSuffix(fi.app + "-swift"),
			Namespace: fi.namespace,
		},
		Data: map[string][]byte{
			cli.RESTIC_PASSWORD: []byte(TEST_RESTIC_PASSWORD),
			cli.OS_AUTH_URL:     []byte(os.Getenv(cli.OS_AUTH_URL)),
			cli.OS_TENANT_ID:    []byte(os.Getenv(cli.OS_TENANT_ID)),
			cli.OS_TENANT_NAME:  []byte(os.Getenv(cli.OS_TENANT_NAME)),
			cli.OS_USERNAME:     []byte(os.Getenv(cli.OS_USERNAME)),
			cli.OS_PASSWORD:     []byte(os.Getenv(cli.OS_PASSWORD)),
			cli.OS_REGION_NAME:  []byte(os.Getenv(cli.OS_REGION_NAME)),
		},
	}
}

// TODO: Add more methods for Swift, Backblaze B2, Rest server backend.

func (f *Framework) CreateSecret(obj apiv1.Secret) error {
	_, err := f.KubeClient.CoreV1().Secrets(obj.Namespace).Create(&obj)
	return err
}

func (f *Framework) DeleteSecret(meta metav1.ObjectMeta) error {
	return f.KubeClient.CoreV1().Secrets(meta.Namespace).Delete(meta.Name, deleteInForeground())
}

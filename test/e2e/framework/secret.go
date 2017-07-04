package framework

import (
	"fmt"
	"io/ioutil"
	"os"
	"time"

	"github.com/appscode/go/crypto/rand"
	"github.com/appscode/log"
	"github.com/appscode/stash/pkg/cli"
	kerr "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	apiv1 "k8s.io/client-go/pkg/api/v1"
)

const (
	TEST_RESTIC_PASSWORD = "not@secret"
)

func (f *Invocation) SecretForLocalBackend() apiv1.Secret {
	return apiv1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      rand.WithUniqSuffix(f.app + "-local"),
			Namespace: f.namespace,
		},
		Data: map[string][]byte{
			cli.RESTIC_PASSWORD: []byte(TEST_RESTIC_PASSWORD),
		},
	}
}

func (f *Invocation) SecretForS3Backend() apiv1.Secret {
	if os.Getenv(cli.AWS_ACCESS_KEY_ID) == "" ||
		os.Getenv(cli.AWS_SECRET_ACCESS_KEY) == "" {
		return apiv1.Secret{}
	}

	return apiv1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      rand.WithUniqSuffix(f.app + "-s3"),
			Namespace: f.namespace,
		},
		Data: map[string][]byte{
			cli.RESTIC_PASSWORD:       []byte(TEST_RESTIC_PASSWORD),
			cli.AWS_ACCESS_KEY_ID:     []byte(os.Getenv(cli.AWS_ACCESS_KEY_ID)),
			cli.AWS_SECRET_ACCESS_KEY: []byte(os.Getenv(cli.AWS_SECRET_ACCESS_KEY)),
		},
	}
}

func (f *Invocation) SecretForGCSBackend() apiv1.Secret {
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
			Name:      rand.WithUniqSuffix(f.app + "-gcs"),
			Namespace: f.namespace,
		},
		Data: map[string][]byte{
			cli.RESTIC_PASSWORD:                 []byte(TEST_RESTIC_PASSWORD),
			cli.GOOGLE_PROJECT_ID:               []byte(os.Getenv(cli.GOOGLE_PROJECT_ID)),
			cli.GOOGLE_SERVICE_ACCOUNT_JSON_KEY: []byte(jsonKey),
		},
	}
}

func (f *Invocation) SecretForAzureBackend() apiv1.Secret {
	if os.Getenv(cli.AZURE_ACCOUNT_NAME) == "" ||
		os.Getenv(cli.AZURE_ACCOUNT_KEY) == "" {
		return apiv1.Secret{}
	}

	return apiv1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      rand.WithUniqSuffix(f.app + "-azure"),
			Namespace: f.namespace,
		},
		Data: map[string][]byte{
			cli.RESTIC_PASSWORD:    []byte(TEST_RESTIC_PASSWORD),
			cli.AZURE_ACCOUNT_NAME: []byte(os.Getenv(cli.AZURE_ACCOUNT_NAME)),
			cli.AZURE_ACCOUNT_KEY:  []byte(os.Getenv(cli.AZURE_ACCOUNT_KEY)),
		},
	}
}

func (f *Invocation) SecretForSwiftBackend() apiv1.Secret {
	if os.Getenv(cli.OS_AUTH_URL) == "" ||
		( os.Getenv(cli.OS_TENANT_ID) == "" && os.Getenv(cli.OS_TENANT_NAME) == "" ) ||
		os.Getenv(cli.OS_USERNAME) == "" ||
		os.Getenv(cli.OS_PASSWORD) == "" {
		return apiv1.Secret{}
	}

	return apiv1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      rand.WithUniqSuffix(f.app + "-swift"),
			Namespace: f.namespace,
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
	_, err := f.kubeClient.CoreV1().Secrets(obj.Namespace).Create(&obj)
	return err
}

func (f *Framework) UpdateSecret(meta metav1.ObjectMeta, transformer func(apiv1.Secret) apiv1.Secret) error {
	attempt := 0
	for ; attempt < maxAttempts; attempt = attempt + 1 {
		cur, err := f.kubeClient.CoreV1().Secrets(meta.Namespace).Get(meta.Name, metav1.GetOptions{})
		if kerr.IsNotFound(err) {
			return nil
		} else if err == nil {
			modified := transformer(*cur)
			_, err = f.kubeClient.CoreV1().Secrets(cur.Namespace).Update(&modified)
			if err == nil {
				return nil
			}
		}
		log.Errorf("Attempt %d failed to update Secret %s@%s due to %s.", attempt, cur.Name, cur.Namespace, err)
		time.Sleep(updateRetryInterval)
	}
	return fmt.Errorf("Failed to update Secret %s@%s after %d attempts.", meta.Name, meta.Namespace, attempt)
}

func (f *Framework) DeleteSecret(meta metav1.ObjectMeta) error {
	return f.kubeClient.CoreV1().Secrets(meta.Namespace).Delete(meta.Name, deleteInForeground())
}

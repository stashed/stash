package restic

import (
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	sapi "github.com/appscode/stash/api"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	clientset "k8s.io/client-go/kubernetes"
)

const (
	RESTIC_REPOSITORY = "RESTIC_REPOSITORY"
	RESTIC_PASSWORD   = "RESTIC_PASSWORD"

	AWS_ACCESS_KEY_ID     = "AWS_ACCESS_KEY_ID"
	AWS_SECRET_ACCESS_KEY = "AWS_SECRET_ACCESS_KEY"

	GOOGLE_PROJECT_ID               = "GOOGLE_PROJECT_ID"
	GOOGLE_SERVICE_ACCOUNT_JSON_KEY = "GOOGLE_SERVICE_ACCOUNT_JSON_KEY"
	GOOGLE_APPLICATION_CREDENTIALS  = "GOOGLE_APPLICATION_CREDENTIALS"

	AZURE_ACCOUNT_NAME = "AZURE_ACCOUNT_NAME"
	AZURE_ACCOUNT_KEY  = "AZURE_ACCOUNT_KEY"
)

func ExportEnvVars(client clientset.Interface, resource *sapi.Restic, prefixHostname bool, scratchDir string) error {
	backend := resource.Spec.Backend
	if backend.RepositorySecretName == "" {
		return errors.New("Missing repository secret name")
	}
	secret, err := client.CoreV1().Secrets(resource.Namespace).Get(backend.RepositorySecretName, metav1.GetOptions{})
	if err != nil {
		return err
	}
	if v, ok := secret.Data[RESTIC_PASSWORD]; !ok {
		return errors.New("Missing repository password")
	} else {
		os.Setenv(RESTIC_PASSWORD, string(v))
	}

	hostname := ""
	if prefixHostname {
		hostname, err = os.Hostname()
		if err != nil {
			return err
		}
	}

	if backend.Local != nil {
		r := backend.Local.Path
		os.Setenv(RESTIC_REPOSITORY, filepath.Join(r, hostname))
	} else if backend.S3 != nil {
		r := fmt.Sprintf("s3:%s:%s:%s", backend.S3.Endpoint, backend.S3.Bucket, backend.S3.Prefix)
		os.Setenv(RESTIC_REPOSITORY, filepath.Join(r, hostname))
		os.Setenv(AWS_ACCESS_KEY_ID, string(secret.Data[AWS_ACCESS_KEY_ID]))
		os.Setenv(AWS_SECRET_ACCESS_KEY, string(secret.Data[AWS_SECRET_ACCESS_KEY]))
	} else if backend.GCS != nil {
		r := fmt.Sprintf("gs:%s:%s:%s", backend.GCS.Location, backend.GCS.Bucket, backend.GCS.Prefix)
		os.Setenv(RESTIC_REPOSITORY, filepath.Join(r, hostname))
		os.Setenv(GOOGLE_PROJECT_ID, string(secret.Data[GOOGLE_PROJECT_ID]))
		jsonKeyPath := filepath.Join(scratchDir, "gcs_sa.json")
		err = ioutil.WriteFile(jsonKeyPath, secret.Data[GOOGLE_SERVICE_ACCOUNT_JSON_KEY], 600)
		if err != nil {
			return err
		}
		os.Setenv(GOOGLE_APPLICATION_CREDENTIALS, jsonKeyPath)
	} else if backend.Azure != nil {
		r := fmt.Sprintf("azure:%s:%s", backend.Azure.Container, backend.Azure.Prefix)
		os.Setenv(RESTIC_REPOSITORY, filepath.Join(r, hostname))
		os.Setenv(AZURE_ACCOUNT_NAME, string(secret.Data[AZURE_ACCOUNT_NAME]))
		os.Setenv(AZURE_ACCOUNT_KEY, string(secret.Data[AZURE_ACCOUNT_KEY]))
	}
	return nil
}

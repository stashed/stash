package restic

import (
	"errors"
	"os"

	sapi "github.com/appscode/stash/api"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	clientset "k8s.io/client-go/kubernetes"
)

const (
	RESTIC_REPOSITORY = "RESTIC_REPOSITORY"
	RESTIC_PASSWORD   = "RESTIC_PASSWORD"

	AWS_ACCESS_KEY_ID     = "AWS_ACCESS_KEY_ID"
	AWS_SECRET_ACCESS_KEY = "AWS_SECRET_ACCESS_KEY"

	GOOGLE_PROJECT_ID              = "GOOGLE_PROJECT_ID"
	GOOGLE_APPLICATION_CREDENTIALS = "GOOGLE_APPLICATION_CREDENTIALS"

	ACCOUNT_NAME = "ACCOUNT_NAME"
	ACCOUNT_KEY  = "ACCOUNT_KEY"
)

func PrepareEnv(client clientset.Interface, resource *sapi.Restic) error {
	if resource.Spec.Backend.RepositorySecretName == "" {
		return errors.New("Missing repository secret name")
	}
	secret, err := client.CoreV1().Secrets(resource.Namespace).Get(resource.Spec.Backend.RepositorySecretName, metav1.GetOptions{})
	if err != nil {
		return err
	}
	if v, ok := secret.Data[RESTIC_PASSWORD]; !ok {
		return errors.New("Missing repository password")
	} else {
		os.Setenv(RESTIC_PASSWORD, string(v))
	}

	if resource.Spec.Backend.Local != nil {
		// TODO: suffix pod for statefulsets
		os.Setenv(RESTIC_REPOSITORY, resource.Spec.Backend.Local.Repository())
	} else if resource.Spec.Backend.S3 != nil {
		os.Setenv(RESTIC_REPOSITORY, resource.Spec.Backend.S3.Repository())
		os.Setenv(AWS_ACCESS_KEY_ID, string(secret.Data[AWS_ACCESS_KEY_ID]))
		os.Setenv(AWS_SECRET_ACCESS_KEY, string(secret.Data[AWS_SECRET_ACCESS_KEY]))
	} else if resource.Spec.Backend.GCS != nil {
		os.Setenv(RESTIC_REPOSITORY, resource.Spec.Backend.GCS.Repository())
		os.Setenv(GOOGLE_PROJECT_ID, string(secret.Data[GOOGLE_PROJECT_ID]))
		os.Setenv(GOOGLE_APPLICATION_CREDENTIALS, string(secret.Data[GOOGLE_APPLICATION_CREDENTIALS])) // TODO; Write the File first
	} else if resource.Spec.Backend.S3 != nil {
		os.Setenv(RESTIC_REPOSITORY, resource.Spec.Backend.Azure.Repository())
		os.Setenv(ACCOUNT_NAME, string(secret.Data[ACCOUNT_NAME]))
		os.Setenv(ACCOUNT_KEY, string(secret.Data[ACCOUNT_KEY]))
	}
	return nil
}

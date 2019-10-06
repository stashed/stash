package v1

import (
	"io/ioutil"
	"log"
	"net/url"

	"github.com/appscode/go/os"
	"github.com/pkg/errors"
	core "k8s.io/api/core/v1"
)

const (
	ProviderLocal = "local"
	ProviderS3    = "s3"
	ProviderGCS   = "gcs"
	ProviderAzure = "azure"
	ProviderSwift = "swift"
	ProviderB2    = "b2"
	ProviderRest  = "rest"
)

// Container returns name of the bucket
func (backend Backend) Container() (string, error) {
	if backend.Local != nil {
		return backend.Local.MountPath, nil
	} else if backend.S3 != nil {
		return backend.S3.Bucket, nil
	} else if backend.GCS != nil {
		return backend.GCS.Bucket, nil
	} else if backend.Azure != nil {
		return backend.Azure.Container, nil
	} else if backend.Swift != nil {
		return backend.Swift.Container, nil
	} else if backend.Rest != nil {
		u, err := url.Parse(backend.Rest.URL)
		if err != nil {
			return "", err
		}
		return u.Host, nil
	}
	return "", errors.New("failed to get container. Reason: Unknown backend type.")
}

// Location returns the location of backend (<provider>:<bucket name>)
func (backend Backend) Location() (string, error) {
	if backend.S3 != nil {
		return "s3:" + backend.S3.Bucket, nil
	} else if backend.GCS != nil {
		return "gs:" + backend.GCS.Bucket, nil
	} else if backend.Azure != nil {
		return "azure:" + backend.Azure.Container, nil
	} else if backend.Local != nil {
		return "local:" + backend.Local.MountPath, nil
	} else if backend.Swift != nil {
		return "swift:" + backend.Swift.Container, nil
	}
	return "", errors.New("no storage provider is configured")
}

// ToVolumeAndMount returns volumes and mounts for local backend
func (l LocalSpec) ToVolumeAndMount(volName string) (core.Volume, core.VolumeMount) {
	vol := core.Volume{
		Name:         volName,
		VolumeSource: *l.VolumeSource.DeepCopy(), // avoid defaulting in MutatingWebhook
	}
	mnt := core.VolumeMount{
		Name:      volName,
		MountPath: l.MountPath,
		SubPath:   l.SubPath,
	}
	return vol, mnt
}

// Prefix returns the prefix used in the backend
func (backend Backend) Prefix() (string, error) {
	if backend.Local != nil {
		return "", nil
	} else if backend.S3 != nil {
		return backend.S3.Prefix, nil
	} else if backend.GCS != nil {
		return backend.GCS.Prefix, nil
	} else if backend.Azure != nil {
		return backend.Azure.Prefix, nil
	} else if backend.Swift != nil {
		return backend.Swift.Prefix, nil
	} else if backend.Rest != nil {
		u, err := url.Parse(backend.Rest.URL)
		if err != nil {
			return "", err
		}
		return u.Path, nil
	}
	return "", errors.New("failed to get prefix. Reason: Unknown backend type.")
}

// Provider returns the provider of the backend
func (backend Backend) Provider() (string, error) {
	if backend.Local != nil {
		return ProviderLocal, nil
	} else if backend.S3 != nil {
		return ProviderS3, nil
	} else if backend.GCS != nil {
		return ProviderGCS, nil
	} else if backend.Azure != nil {
		return ProviderAzure, nil
	} else if backend.Swift != nil {
		return ProviderSwift, nil
	} else if backend.B2 != nil {
		return ProviderB2, nil
	} else if backend.Rest != nil {
		return ProviderRest, nil
	}
	return "", errors.New("unknown provider.")
}

// MaxConnections returns maximum parallel connection to use to connect with the backend
// returns 0 if not specified
func (backend Backend) MaxConnections() int {
	if backend.GCS != nil {
		return backend.GCS.MaxConnections
	} else if backend.Azure != nil {
		return backend.Azure.MaxConnections
	} else if backend.B2 != nil {
		return backend.B2.MaxConnections
	}
	return 0
}

// Endpoint returns endpoint of Restic rest server and S3/S3 compatible backend
func (backend Backend) Endpoint() (string, bool) {
	if backend.S3 != nil {
		return backend.S3.Endpoint, true
	} else if backend.Rest != nil {
		return backend.Rest.URL, true
	}
	return "", false
}

const (
	GCPSACredentialJson = "sa.json"
)

func GoogleServiceAccountFromEnv() string {
	if data := os.Getenv(GOOGLE_SERVICE_ACCOUNT_JSON_KEY); len(data) > 0 {
		return data
	}
	if data, err := ioutil.ReadFile(os.Getenv(GOOGLE_APPLICATION_CREDENTIALS)); err == nil {
		return string(data)
	}
	log.Println("GOOGLE_SERVICE_ACCOUNT_JSON_KEY and GOOGLE_APPLICATION_CREDENTIALS are empty")
	return ""
}

func GoogleCredentialsFromEnv() map[string][]byte {
	sa := GoogleServiceAccountFromEnv()
	if len(sa) == 0 {
		return map[string][]byte{}
	}
	return map[string][]byte{
		GCPSACredentialJson: []byte(sa),
	}
}

const (
	AzureClientSecret   = "client-secret"
	AzureSubscriptionID = "subscription-id"
	AzureTenantID       = "tenant-id"
	AzureClientID       = "client-id"
)

func AzureCredentialsFromEnv() map[string][]byte {
	subscriptionID := os.Getenv("AZURE_SUBSCRIPTION_ID")
	tenantID := os.Getenv("AZURE_TENANT_ID")
	clientID := os.Getenv("AZURE_CLIENT_ID")
	clientSecret := os.Getenv("AZURE_CLIENT_SECRET")
	if len(subscriptionID) == 0 || len(tenantID) == 0 || len(clientID) == 0 || len(clientSecret) == 0 {
		log.Println("Azure credentials for empty")
		return map[string][]byte{}
	}

	return map[string][]byte{
		AzureSubscriptionID: []byte(subscriptionID),
		AzureTenantID:       []byte(tenantID),
		AzureClientID:       []byte(clientID),
		AzureClientSecret:   []byte(clientSecret),
	}
}

const (
	AWSCredentialAccessKeyKey = "access_key"
	AWSCredentialSecretKeyKey = "secret_key"
)

func AWSCredentialsFromEnv() map[string][]byte {
	awsAccessKeyId := os.Getenv("AWS_ACCESS_KEY_ID")
	awsSecretAccessKey := os.Getenv("AWS_SECRET_ACCESS_KEY")
	if len(awsAccessKeyId) == 0 || len(awsSecretAccessKey) == 0 {
		log.Println("AWS credentials for empty")
		return map[string][]byte{}
	}

	return map[string][]byte{
		AWSCredentialAccessKeyKey: []byte(awsAccessKeyId),
		AWSCredentialSecretKeyKey: []byte(awsSecretAccessKey),
	}
}

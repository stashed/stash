package v1

import (
	"path/filepath"
	"strings"

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
	if backend.S3 != nil {
		return backend.S3.Bucket, nil
	} else if backend.GCS != nil {
		return backend.GCS.Bucket, nil
	} else if backend.Azure != nil {
		return backend.Azure.Container, nil
	} else if backend.Local != nil {
		return backend.Local.MountPath, nil
	} else if backend.Swift != nil {
		return backend.Swift.Container, nil
	} else if backend.Rest != nil {
		return backend.Rest.URL, nil
	}
	return "", errors.New("no storage provider is configured")
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
	} else if backend.Rest != nil {
		return "rest:" + backend.Rest.URL, nil
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

// GetBucketAndPrefix return bucket and the prefix used in the backend
func (backend Backend) GetBucketAndPrefix() (string, string, error) {
	if backend.Local != nil {
		return "", filepath.Join(backend.Local.MountPath, strings.TrimPrefix(backend.Local.SubPath, "/")), nil
	} else if backend.S3 != nil {
		return backend.S3.Bucket, strings.TrimPrefix(backend.S3.Prefix, backend.S3.Bucket+"/"), nil
	} else if backend.GCS != nil {
		return backend.GCS.Bucket, backend.GCS.Prefix, nil
	} else if backend.Azure != nil {
		return backend.Azure.Container, backend.Azure.Prefix, nil
	} else if backend.Swift != nil {
		return backend.Swift.Container, backend.Swift.Prefix, nil
	} else if backend.Rest != nil {
		return "", "", nil
	}
	return "", "", errors.New("unknown backend type.")
}

// GetProvider returns the provider of the backend
func (backend Backend) GetProvider() (string, error) {
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

// GetMaxConnections returns maximum parallel connection to use to connect with the backend
// returns 0 if not specified
func (backend Backend) GetMaxConnections() int {
	if backend.GCS != nil {
		return backend.GCS.MaxConnections
	} else if backend.Azure != nil {
		return backend.Azure.MaxConnections
	} else if backend.B2 != nil {
		return backend.B2.MaxConnections
	}
	return 0
}

// GetEndpoint returns endpoint of S3/S3 compatible backend
func (backend Backend) GetEndpoint() string {
	if backend.S3 != nil {
		return backend.S3.Endpoint
	}
	return ""
}

// GetRestUrl returns the URL of REST backend
func (backend *Backend) GetRestUrl() string {
	if backend.Rest != nil {
		return backend.Rest.URL
	}
	return ""
}

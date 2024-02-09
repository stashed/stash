/*
Copyright AppsCode Inc. and Contributors

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package v1

import (
	"net/url"

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
	} else if backend.B2 != nil {
		return backend.B2.Bucket, nil
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
	} else if backend.B2 != nil {
		return "b2:" + backend.B2.Bucket, nil
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
	} else if backend.B2 != nil {
		return backend.B2.Prefix, nil
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
func (backend Backend) MaxConnections() int64 {
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

// Region returns region of S3/S3 compatible backend
func (backend Backend) Region() (string, bool) {
	if backend.S3 != nil {
		return backend.S3.Region, true
	}
	return "", false
}

// InsecureTLS returns insecureTLS of S3/S3 compatible backend
func (backend Backend) InsecureTLS() bool {
	if backend.S3 != nil {
		return backend.S3.InsecureTLS
	}
	return false
}

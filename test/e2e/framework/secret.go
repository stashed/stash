/*
Copyright AppsCode Inc. and Contributors

Licensed under the AppsCode Community License 1.0.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    https://github.com/appscode/licenses/raw/1.0.0/AppsCode-Community-1.0.0.md

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package framework

import (
	"context"
	"os"

	"stash.appscode.dev/apimachinery/pkg/restic"

	"gomodules.xyz/x/crypto/rand"
	core "k8s.io/api/core/v1"
	kerr "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	googleconsts "kmodules.xyz/constants/google"
)

const (
	TEST_RESTIC_PASSWORD = "not@secret"
)

func (fi *Invocation) SecretForS3Backend() core.Secret {
	if os.Getenv(restic.AWS_ACCESS_KEY_ID) == "" ||
		os.Getenv(restic.AWS_SECRET_ACCESS_KEY) == "" {
		return core.Secret{}
	}

	return core.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      rand.WithUniqSuffix(fi.app + "-s3"),
			Namespace: fi.namespace,
		},
		Data: map[string][]byte{
			restic.RESTIC_PASSWORD:       []byte(TEST_RESTIC_PASSWORD),
			restic.AWS_ACCESS_KEY_ID:     []byte(os.Getenv(restic.AWS_ACCESS_KEY_ID)),
			restic.AWS_SECRET_ACCESS_KEY: []byte(os.Getenv(restic.AWS_SECRET_ACCESS_KEY)),
		},
	}
}

const (
	DO_ACCESS_KEY_ID     = "DO_ACCESS_KEY_ID"
	DO_SECRET_ACCESS_KEY = "DO_SECRET_ACCESS_KEY"
)

func (fi *Invocation) SecretForDOBackend() core.Secret {
	if os.Getenv(DO_ACCESS_KEY_ID) == "" ||
		os.Getenv(DO_SECRET_ACCESS_KEY) == "" {
		return core.Secret{}
	}

	return core.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      rand.WithUniqSuffix(fi.app + "-s3"),
			Namespace: fi.namespace,
		},
		Data: map[string][]byte{
			restic.RESTIC_PASSWORD:       []byte(TEST_RESTIC_PASSWORD),
			restic.AWS_ACCESS_KEY_ID:     []byte(os.Getenv(DO_ACCESS_KEY_ID)),
			restic.AWS_SECRET_ACCESS_KEY: []byte(os.Getenv(DO_SECRET_ACCESS_KEY)),
		},
	}
}

func (fi *Invocation) SecretForGCSBackend() core.Secret {
	jsonKey := googleconsts.ServiceAccountFromEnv()

	if jsonKey == "" || os.Getenv(googleconsts.GOOGLE_PROJECT_ID) == "" {
		return core.Secret{}
	}

	return core.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      rand.WithUniqSuffix(fi.app + "-gcs"),
			Namespace: fi.namespace,
		},
		Data: map[string][]byte{
			restic.RESTIC_PASSWORD:                 []byte(TEST_RESTIC_PASSWORD),
			restic.GOOGLE_PROJECT_ID:               []byte(os.Getenv(restic.GOOGLE_PROJECT_ID)),
			restic.GOOGLE_SERVICE_ACCOUNT_JSON_KEY: []byte(jsonKey),
		},
	}
}

func (fi *Invocation) SecretForAzureBackend() core.Secret {
	if os.Getenv(restic.AZURE_ACCOUNT_NAME) == "" ||
		os.Getenv(restic.AZURE_ACCOUNT_KEY) == "" {
		return core.Secret{}
	}

	return core.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      rand.WithUniqSuffix(fi.app + "-azure"),
			Namespace: fi.namespace,
		},
		Data: map[string][]byte{
			restic.RESTIC_PASSWORD:    []byte(TEST_RESTIC_PASSWORD),
			restic.AZURE_ACCOUNT_NAME: []byte(os.Getenv(restic.AZURE_ACCOUNT_NAME)),
			restic.AZURE_ACCOUNT_KEY:  []byte(os.Getenv(restic.AZURE_ACCOUNT_KEY)),
		},
	}
}

func (fi *Invocation) SecretForSwiftBackend() core.Secret {
	if os.Getenv(restic.OS_AUTH_URL) == "" ||
		(os.Getenv(restic.OS_TENANT_ID) == "" && os.Getenv(restic.OS_TENANT_NAME) == "") ||
		os.Getenv(restic.OS_USERNAME) == "" ||
		os.Getenv(restic.OS_PASSWORD) == "" {
		return core.Secret{}
	}

	return core.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      rand.WithUniqSuffix(fi.app + "-swift"),
			Namespace: fi.namespace,
		},
		Data: map[string][]byte{
			restic.RESTIC_PASSWORD: []byte(TEST_RESTIC_PASSWORD),
			restic.OS_AUTH_URL:     []byte(os.Getenv(restic.OS_AUTH_URL)),
			restic.OS_TENANT_ID:    []byte(os.Getenv(restic.OS_TENANT_ID)),
			restic.OS_TENANT_NAME:  []byte(os.Getenv(restic.OS_TENANT_NAME)),
			restic.OS_USERNAME:     []byte(os.Getenv(restic.OS_USERNAME)),
			restic.OS_PASSWORD:     []byte(os.Getenv(restic.OS_PASSWORD)),
			restic.OS_REGION_NAME:  []byte(os.Getenv(restic.OS_REGION_NAME)),
		},
	}
}

func (fi *Invocation) SecretForB2Backend() core.Secret {
	if os.Getenv(restic.B2_ACCOUNT_ID) == "" ||
		os.Getenv(restic.B2_ACCOUNT_KEY) == "" {
		return core.Secret{}
	}

	return core.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      rand.WithUniqSuffix(fi.app + "-b2"),
			Namespace: fi.namespace,
		},
		Data: map[string][]byte{
			restic.RESTIC_PASSWORD: []byte(TEST_RESTIC_PASSWORD),
			restic.B2_ACCOUNT_ID:   []byte(os.Getenv(restic.B2_ACCOUNT_ID)),
			restic.B2_ACCOUNT_KEY:  []byte(os.Getenv(restic.B2_ACCOUNT_KEY)),
		},
	}
}

func (fi *Invocation) SecretForMinioBackend(includeCacert bool) core.Secret {
	secret := core.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      rand.WithUniqSuffix(fi.app + "-minio"),
			Namespace: fi.namespace,
		},
		Data: map[string][]byte{
			restic.RESTIC_PASSWORD:       []byte(TEST_RESTIC_PASSWORD),
			restic.AWS_ACCESS_KEY_ID:     []byte(MINIO_ACCESS_KEY_ID),
			restic.AWS_SECRET_ACCESS_KEY: []byte(MINIO_SECRET_ACCESS_KEY),
		},
	}
	if includeCacert {
		secret.Data[restic.CA_CERT_DATA] = fi.CertStore.CACertBytes()
	}
	return secret
}

func (fi *Invocation) SecretForRestBackend(includeCacert bool, username, password string) core.Secret {
	secret := core.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      rand.WithUniqSuffix(fi.app + "-rest"),
			Namespace: fi.namespace,
		},
		Data: map[string][]byte{
			restic.RESTIC_PASSWORD:      []byte(TEST_RESTIC_PASSWORD),
			restic.REST_SERVER_USERNAME: []byte(username),
			restic.REST_SERVER_PASSWORD: []byte(password),
		},
	}
	if includeCacert {
		secret.Data[restic.CA_CERT_DATA] = fi.CertStore.CACertBytes()
	}
	return secret
}

func (fi *Invocation) SecretForRegistry(dockerCfgJson []byte) core.Secret {
	return core.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      rand.WithUniqSuffix(fi.app + "-docker"),
			Namespace: fi.namespace,
		},
		Type: core.SecretTypeDockerConfigJson,
		Data: map[string][]byte{
			core.DockerConfigJsonKey: dockerCfgJson,
		},
	}
}

// TODO: Add more methods for Swift, Backblaze B2, Rest server backend.

func (f *Framework) CreateSecret(obj core.Secret) (*core.Secret, error) {
	return f.KubeClient.CoreV1().Secrets(obj.Namespace).Create(context.TODO(), &obj, metav1.CreateOptions{})
}

func (f *Framework) DeleteSecret(meta metav1.ObjectMeta) error {
	err := f.KubeClient.CoreV1().Secrets(meta.Namespace).Delete(context.TODO(), meta.Name, *deleteInForeground())
	if err != nil && !kerr.IsNotFound(err) {
		return err
	}
	return nil
}

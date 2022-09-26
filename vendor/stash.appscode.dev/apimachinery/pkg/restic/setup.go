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

package restic

import (
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"

	storage "kmodules.xyz/objectstore-api/api/v1"
)

const (
	RESTIC_REPOSITORY   = "RESTIC_REPOSITORY"
	RESTIC_PASSWORD     = "RESTIC_PASSWORD"
	RESTIC_PROGRESS_FPS = "RESTIC_PROGRESS_FPS"
	TMPDIR              = "TMPDIR"

	AWS_ACCESS_KEY_ID     = "AWS_ACCESS_KEY_ID"
	AWS_SECRET_ACCESS_KEY = "AWS_SECRET_ACCESS_KEY"
	AWS_DEFAULT_REGION    = "AWS_DEFAULT_REGION"

	GOOGLE_PROJECT_ID               = "GOOGLE_PROJECT_ID"
	GOOGLE_SERVICE_ACCOUNT_JSON_KEY = "GOOGLE_SERVICE_ACCOUNT_JSON_KEY"
	GOOGLE_APPLICATION_CREDENTIALS  = "GOOGLE_APPLICATION_CREDENTIALS"

	AZURE_ACCOUNT_NAME = "AZURE_ACCOUNT_NAME"
	AZURE_ACCOUNT_KEY  = "AZURE_ACCOUNT_KEY"

	REST_SERVER_USERNAME = "REST_SERVER_USERNAME"
	REST_SERVER_PASSWORD = "REST_SERVER_PASSWORD"

	B2_ACCOUNT_ID  = "B2_ACCOUNT_ID"
	B2_ACCOUNT_KEY = "B2_ACCOUNT_KEY"

	// For keystone v1 authentication
	ST_AUTH = "ST_AUTH"
	ST_USER = "ST_USER"
	ST_KEY  = "ST_KEY"
	// For keystone v2 authentication (some variables are optional)
	OS_AUTH_URL    = "OS_AUTH_URL"
	OS_REGION_NAME = "OS_REGION_NAME"
	OS_USERNAME    = "OS_USERNAME"
	OS_PASSWORD    = "OS_PASSWORD"
	OS_TENANT_ID   = "OS_TENANT_ID"
	OS_TENANT_NAME = "OS_TENANT_NAME"
	// For keystone v3 authentication (some variables are optional)
	OS_USER_DOMAIN_NAME    = "OS_USER_DOMAIN_NAME"
	OS_PROJECT_NAME        = "OS_PROJECT_NAME"
	OS_PROJECT_DOMAIN_NAME = "OS_PROJECT_DOMAIN_NAME"
	// For keystone v3 application credential authentication (application credential id)
	OS_APPLICATION_CREDENTIAL_ID     = "OS_APPLICATION_CREDENTIAL_ID"
	OS_APPLICATION_CREDENTIAL_SECRET = "OS_APPLICATION_CREDENTIAL_SECRET"
	// For keystone v3 application credential authentication (application credential name)
	OS_APPLICATION_CREDENTIAL_NAME = "OS_APPLICATION_CREDENTIAL_NAME"
	// For authentication based on tokens
	OS_STORAGE_URL = "OS_STORAGE_URL"
	OS_AUTH_TOKEN  = "OS_AUTH_TOKEN"

	// For using certs in Minio server or REST server
	CA_CERT_DATA = "CA_CERT_DATA"

	// ref: https://github.com/restic/restic/blob/master/doc/manual_rest.rst#temporary-files
	resticCacheDir = "restic-cache"
)

func (w *ResticWrapper) setupEnv() error {
	// Set progress report frequency.
	// 0.016666 is for one report per minute.
	// ref: https://restic.readthedocs.io/en/stable/manual_rest.html
	w.sh.SetEnv(RESTIC_PROGRESS_FPS, "0.016666")

	if w.config.StorageSecret == nil {
		return errors.New("missing storage Secret")
	}

	if err := w.exportSecretKey(RESTIC_PASSWORD, true); err != nil {
		return err
	}

	tmpDir, err := os.MkdirTemp(w.config.ScratchDir, "tmp-")
	if err != nil {
		return err
	}
	w.sh.SetEnv(TMPDIR, tmpDir)

	if _, ok := w.config.StorageSecret.Data[CA_CERT_DATA]; ok {
		filePath, err := w.writeSecretKeyToFile(CA_CERT_DATA, "ca.crt")
		if err != nil {
			return err
		}
		w.config.CacertFile = filePath
	}

	if w.config.EnableCache {
		cacheDir := filepath.Join(w.config.ScratchDir, resticCacheDir)
		if err := os.MkdirAll(cacheDir, 0o755); err != nil {
			return err
		}
	}

	// path = strings.TrimPrefix(path, "/")

	switch w.config.Provider {

	case storage.ProviderLocal:
		r := w.config.Bucket
		if err := os.MkdirAll(r, 0o755); err != nil {
			return err
		}
		w.sh.SetEnv(RESTIC_REPOSITORY, r)

	case storage.ProviderS3:
		r := fmt.Sprintf("s3:%s/%s", w.config.Endpoint, filepath.Join(w.config.Bucket, w.config.Path))
		w.sh.SetEnv(RESTIC_REPOSITORY, r)

		if err := w.exportSecretKey(AWS_ACCESS_KEY_ID, false); err != nil {
			return err
		}

		if err := w.exportSecretKey(AWS_SECRET_ACCESS_KEY, false); err != nil {
			return err
		}

		if w.config.Region != "" {
			w.sh.SetEnv(AWS_DEFAULT_REGION, w.config.Region)
		}

	case storage.ProviderGCS:
		r := fmt.Sprintf("gs:%s:/%s", w.config.Bucket, w.config.Path)
		w.sh.SetEnv(RESTIC_REPOSITORY, r)

		if err := w.exportSecretKey(GOOGLE_PROJECT_ID, false); err != nil {
			return err
		}

		if w.isSecretKeyExist(GOOGLE_SERVICE_ACCOUNT_JSON_KEY) {
			filePath, err := w.writeSecretKeyToFile(GOOGLE_SERVICE_ACCOUNT_JSON_KEY, GOOGLE_SERVICE_ACCOUNT_JSON_KEY)
			if err != nil {
				return err
			}
			w.sh.SetEnv(GOOGLE_APPLICATION_CREDENTIALS, filePath)
		}
	case storage.ProviderAzure:
		r := fmt.Sprintf("azure:%s:/%s", w.config.Bucket, w.config.Path)
		w.sh.SetEnv(RESTIC_REPOSITORY, r)

		if err := w.exportSecretKey(AZURE_ACCOUNT_NAME, false); err != nil {
			return err
		}

		if err := w.exportSecretKey(AZURE_ACCOUNT_KEY, false); err != nil {
			return err
		}

	case storage.ProviderSwift:
		r := fmt.Sprintf("swift:%s:/%s", w.config.Bucket, w.config.Path)
		w.sh.SetEnv(RESTIC_REPOSITORY, r)

		// For keystone v1 authentication
		// Necessary Envs:
		// ST_AUTH
		// ST_USER
		// ST_KEY
		if err := w.exportSecretKey(ST_AUTH, false); err != nil {
			return err
		}

		if err := w.exportSecretKey(ST_USER, false); err != nil {
			return err
		}

		if err := w.exportSecretKey(ST_KEY, false); err != nil {
			return err
		}

		// For keystone v2 authentication (some variables are optional)
		// Necessary Envs:
		// OS_AUTH_URL
		// OS_REGION_NAME
		// OS_USERNAME
		// OS_PASSWORD
		// OS_TENANT_ID
		// OS_TENANT_NAME
		if err := w.exportSecretKey(OS_AUTH_URL, false); err != nil {
			return err
		}

		if err := w.exportSecretKey(OS_REGION_NAME, false); err != nil {
			return err
		}

		if err := w.exportSecretKey(OS_USERNAME, false); err != nil {
			return err
		}

		if err := w.exportSecretKey(OS_PASSWORD, false); err != nil {
			return err
		}

		if err := w.exportSecretKey(OS_TENANT_ID, false); err != nil {
			return err
		}

		if err := w.exportSecretKey(OS_TENANT_NAME, false); err != nil {
			return err
		}

		// For keystone v3 authentication (some variables are optional)
		// Necessary Envs:
		// OS_AUTH_URL (already set in v2 authentication section)
		// OS_REGION_NAME (already set in v2 authentication section)
		// OS_USERNAME (already set in v2 authentication section)
		// OS_PASSWORD (already set in v2 authentication section)
		// OS_USER_DOMAIN_NAME
		// OS_PROJECT_NAME
		// OS_PROJECT_DOMAIN_NAME
		if err := w.exportSecretKey(OS_USER_DOMAIN_NAME, false); err != nil {
			return err
		}

		if err := w.exportSecretKey(OS_PROJECT_NAME, false); err != nil {
			return err
		}

		if err := w.exportSecretKey(OS_PROJECT_DOMAIN_NAME, false); err != nil {
			return err
		}

		// For keystone v3 application credential authentication (application credential id)
		// Necessary Envs:
		// OS_AUTH_URL (already set in v2 authentication section)
		// OS_APPLICATION_CREDENTIAL_ID
		// OS_APPLICATION_CREDENTIAL_SECRET
		if err := w.exportSecretKey(OS_APPLICATION_CREDENTIAL_ID, false); err != nil {
			return err
		}

		if err := w.exportSecretKey(OS_APPLICATION_CREDENTIAL_SECRET, false); err != nil {
			return err
		}

		// For keystone v3 application credential authentication (application credential name)
		// Necessary Envs:
		// OS_AUTH_URL (already set in v2 authentication section)
		// OS_USERNAME (already set in v2 authentication section)
		// OS_USER_DOMAIN_NAME (already set in v3 authentication section)
		// OS_APPLICATION_CREDENTIAL_NAME
		// OS_APPLICATION_CREDENTIAL_SECRET (already set in v3 authentication with credential id section)
		if err := w.exportSecretKey(OS_APPLICATION_CREDENTIAL_NAME, false); err != nil {
			return err
		}

		// For authentication based on tokens
		// Necessary Envs:
		// OS_STORAGE_URL
		// OS_AUTH_TOKEN
		if err := w.exportSecretKey(OS_STORAGE_URL, false); err != nil {
			return err
		}

		if err := w.exportSecretKey(OS_AUTH_TOKEN, false); err != nil {
			return err
		}

	case storage.ProviderB2:
		r := fmt.Sprintf("b2:%s:/%s", w.config.Bucket, w.config.Path)
		w.sh.SetEnv(RESTIC_REPOSITORY, r)

		if err := w.exportSecretKey(B2_ACCOUNT_ID, true); err != nil {
			return err
		}

		if err := w.exportSecretKey(B2_ACCOUNT_KEY, true); err != nil {
			return err
		}

	case storage.ProviderRest:
		u, err := url.Parse(w.config.Endpoint)
		if err != nil {
			return err
		}

		if username, hasUserKey := w.config.StorageSecret.Data[REST_SERVER_USERNAME]; hasUserKey {
			if password, hasPassKey := w.config.StorageSecret.Data[REST_SERVER_PASSWORD]; hasPassKey {
				u.User = url.UserPassword(string(username), string(password))
			} else {
				u.User = url.User(string(username))
			}
		}
		// u.Path = filepath.Join(u.Path, w.config.Path) // path integrated with url
		r := fmt.Sprintf("rest:%s", u.String())
		w.sh.SetEnv(RESTIC_REPOSITORY, r)
	}

	return nil
}

func (w *ResticWrapper) exportSecretKey(key string, required bool) error {
	if v, ok := w.config.StorageSecret.Data[key]; !ok {
		if required {
			return fmt.Errorf("storage Secret missing %s key", key)
		}
	} else {
		w.sh.SetEnv(key, string(v))
	}
	return nil
}

func (w *ResticWrapper) isSecretKeyExist(key string) bool {
	_, ok := w.config.StorageSecret.Data[key]
	return ok
}

func (w *ResticWrapper) writeSecretKeyToFile(key, name string) (string, error) {
	v, ok := w.config.StorageSecret.Data[key]
	if !ok {
		return "", fmt.Errorf("storage Secret missing %s key", key)
	}

	tmpDir := w.GetEnv(TMPDIR)
	filePath := filepath.Join(tmpDir, name)

	if err := os.WriteFile(filePath, v, 0o755); err != nil {
		return "", err
	}
	return filePath, nil
}

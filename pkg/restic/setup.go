package restic

import (
	"fmt"
	"io/ioutil"
	"net/url"
	"os"
	"path/filepath"
)

const (
	ProviderLocal = "local"
	ProviderS3    = "s3"
	ProviderGCS   = "gcs"
	ProviderAzure = "azure"
	ProviderSwift = "swift"
	ProviderB2    = "b2"
	ProviderRest  = "rest"

	RESTIC_REPOSITORY = "RESTIC_REPOSITORY"
	RESTIC_PASSWORD   = "RESTIC_PASSWORD"
	TMPDIR            = "TMPDIR"

	AWS_ACCESS_KEY_ID     = "AWS_ACCESS_KEY_ID"
	AWS_SECRET_ACCESS_KEY = "AWS_SECRET_ACCESS_KEY"

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
	resticTempDir  = "restic-tmp"
	resticCacheDir = "restic-cache"
)

func (w *ResticWrapper) setupEnv() error {

	if v, err := ioutil.ReadFile(filepath.Join(w.config.SecretDir, RESTIC_PASSWORD)); err != nil {
		return err
	} else {
		w.sh.SetEnv(RESTIC_PASSWORD, string(v))
	}

	if _, err := os.Stat(filepath.Join(w.config.SecretDir, CA_CERT_DATA)); err == nil {
		// ca-cert file exists
		w.config.CacertFile = filepath.Join(w.config.SecretDir, CA_CERT_DATA)
	}

	tmpDir := filepath.Join(w.config.ScratchDir, resticTempDir)
	if err := os.MkdirAll(tmpDir, 0755); err != nil {
		return err
	}
	w.sh.SetEnv(TMPDIR, tmpDir)

	if w.config.EnableCache {
		cacheDir := filepath.Join(w.config.ScratchDir, resticCacheDir)
		if err := os.MkdirAll(cacheDir, 0755); err != nil {
			return err
		}
	}

	//path = strings.TrimPrefix(path, "/")

	switch w.config.Provider {

	case ProviderLocal:
		r := w.config.Path
		if err := os.MkdirAll(r, 0755); err != nil {
			return err
		}
		w.sh.SetEnv(RESTIC_REPOSITORY, r)

	case ProviderS3:
		r := fmt.Sprintf("s3:%s/%s", w.config.Endpoint, filepath.Join(w.config.Bucket, w.config.Path))
		w.sh.SetEnv(RESTIC_REPOSITORY, r)

		if v, err := ioutil.ReadFile(filepath.Join(w.config.SecretDir, AWS_ACCESS_KEY_ID)); err == nil {
			w.sh.SetEnv(AWS_ACCESS_KEY_ID, string(v))
		}
		if v, err := ioutil.ReadFile(filepath.Join(w.config.SecretDir, AWS_SECRET_ACCESS_KEY)); err == nil {
			w.sh.SetEnv(AWS_SECRET_ACCESS_KEY, string(v))
		}

	case ProviderGCS:
		r := fmt.Sprintf("gs:%s:/%s", w.config.Bucket, w.config.Path)
		w.sh.SetEnv(RESTIC_REPOSITORY, r)

		if v, err := ioutil.ReadFile(filepath.Join(w.config.SecretDir, GOOGLE_PROJECT_ID)); err == nil {
			w.sh.SetEnv(GOOGLE_PROJECT_ID, string(v))
		}

		if _, err := os.Stat(filepath.Join(w.config.SecretDir, GOOGLE_SERVICE_ACCOUNT_JSON_KEY)); err == nil {
			// json key file exists
			w.sh.SetEnv(GOOGLE_APPLICATION_CREDENTIALS, filepath.Join(w.config.SecretDir, GOOGLE_SERVICE_ACCOUNT_JSON_KEY))
		}

	case ProviderAzure:
		r := fmt.Sprintf("azure:%s:/%s", w.config.Bucket, w.config.Path)
		w.sh.SetEnv(RESTIC_REPOSITORY, r)

		if v, err := ioutil.ReadFile(filepath.Join(w.config.SecretDir, AZURE_ACCOUNT_NAME)); err == nil {
			w.sh.SetEnv(AZURE_ACCOUNT_NAME, string(v))
		}
		if v, err := ioutil.ReadFile(filepath.Join(w.config.SecretDir, AZURE_ACCOUNT_KEY)); err == nil {
			w.sh.SetEnv(AZURE_ACCOUNT_KEY, string(v))
		}

	case ProviderSwift:
		r := fmt.Sprintf("swift:%s:/%s", w.config.Bucket, w.config.Path)
		w.sh.SetEnv(RESTIC_REPOSITORY, r)

		// For keystone v1 authentication
		// Necessary Envs:
		// ST_AUTH
		// ST_USER
		// ST_KEY
		if v, err := ioutil.ReadFile(filepath.Join(w.config.SecretDir, ST_AUTH)); err == nil {
			w.sh.SetEnv(ST_AUTH, string(v))
		}
		if v, err := ioutil.ReadFile(filepath.Join(w.config.SecretDir, ST_USER)); err == nil {
			w.sh.SetEnv(ST_USER, string(v))
		}
		if v, err := ioutil.ReadFile(filepath.Join(w.config.SecretDir, ST_KEY)); err == nil {
			w.sh.SetEnv(ST_KEY, string(v))
		}

		// For keystone v2 authentication (some variables are optional)
		// Necessary Envs:
		// OS_AUTH_URL
		// OS_REGION_NAME
		// OS_USERNAME
		// OS_PASSWORD
		// OS_TENANT_ID
		// OS_TENANT_NAME
		if v, err := ioutil.ReadFile(filepath.Join(w.config.SecretDir, OS_AUTH_URL)); err == nil {
			w.sh.SetEnv(OS_AUTH_URL, string(v))
		}
		if v, err := ioutil.ReadFile(filepath.Join(w.config.SecretDir, OS_REGION_NAME)); err == nil {
			w.sh.SetEnv(OS_REGION_NAME, string(v))
		}
		if v, err := ioutil.ReadFile(filepath.Join(w.config.SecretDir, OS_USERNAME)); err == nil {
			w.sh.SetEnv(OS_USERNAME, string(v))
		}
		if v, err := ioutil.ReadFile(filepath.Join(w.config.SecretDir, OS_PASSWORD)); err == nil {
			w.sh.SetEnv(OS_PASSWORD, string(v))
		}
		if v, err := ioutil.ReadFile(filepath.Join(w.config.SecretDir, OS_TENANT_ID)); err == nil {
			w.sh.SetEnv(OS_TENANT_ID, string(v))
		}
		if v, err := ioutil.ReadFile(filepath.Join(w.config.SecretDir, OS_TENANT_NAME)); err == nil {
			w.sh.SetEnv(OS_TENANT_NAME, string(v))
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
		if v, err := ioutil.ReadFile(filepath.Join(w.config.SecretDir, OS_USER_DOMAIN_NAME)); err == nil {
			w.sh.SetEnv(OS_USER_DOMAIN_NAME, string(v))
		}
		if v, err := ioutil.ReadFile(filepath.Join(w.config.SecretDir, OS_PROJECT_NAME)); err == nil {
			w.sh.SetEnv(OS_PROJECT_NAME, string(v))
		}
		if v, err := ioutil.ReadFile(filepath.Join(w.config.SecretDir, OS_PROJECT_DOMAIN_NAME)); err == nil {
			w.sh.SetEnv(OS_PROJECT_DOMAIN_NAME, string(v))
		}

		// For keystone v3 application credential authentication (application credential id)
		// Necessary Envs:
		// OS_AUTH_URL (already set in v2 authentication section)
		// OS_APPLICATION_CREDENTIAL_ID
		// OS_APPLICATION_CREDENTIAL_SECRET
		if v, err := ioutil.ReadFile(filepath.Join(w.config.SecretDir, OS_APPLICATION_CREDENTIAL_ID)); err == nil {
			w.sh.SetEnv(OS_APPLICATION_CREDENTIAL_ID, string(v))
		}
		if v, err := ioutil.ReadFile(filepath.Join(w.config.SecretDir, OS_APPLICATION_CREDENTIAL_SECRET)); err == nil {
			w.sh.SetEnv(OS_APPLICATION_CREDENTIAL_SECRET, string(v))
		}

		// For keystone v3 application credential authentication (application credential name)
		// Necessary Envs:
		// OS_AUTH_URL (already set in v2 authentication section)
		// OS_USERNAME (already set in v2 authentication section)
		// OS_USER_DOMAIN_NAME (already set in v3 authentication section)
		// OS_APPLICATION_CREDENTIAL_NAME
		// OS_APPLICATION_CREDENTIAL_SECRET (already set in v3 authentication with credential id section)
		if v, err := ioutil.ReadFile(filepath.Join(w.config.SecretDir, OS_APPLICATION_CREDENTIAL_NAME)); err == nil {
			w.sh.SetEnv(OS_APPLICATION_CREDENTIAL_NAME, string(v))
		}

		// For authentication based on tokens
		// Necessary Envs:
		// OS_STORAGE_URL
		// OS_AUTH_TOKEN
		if v, err := ioutil.ReadFile(filepath.Join(w.config.SecretDir, OS_STORAGE_URL)); err == nil {
			w.sh.SetEnv(OS_STORAGE_URL, string(v))
		}
		if v, err := ioutil.ReadFile(filepath.Join(w.config.SecretDir, OS_AUTH_TOKEN)); err == nil {
			w.sh.SetEnv(OS_AUTH_TOKEN, string(v))
		}

	case ProviderB2:
		r := fmt.Sprintf("b2:%s:/%s", w.config.Bucket, w.config.Path)
		w.sh.SetEnv(RESTIC_REPOSITORY, r)

		if v, err := ioutil.ReadFile(filepath.Join(w.config.SecretDir, B2_ACCOUNT_ID)); err == nil {
			w.sh.SetEnv(B2_ACCOUNT_ID, string(v))
		}

		if v, err := ioutil.ReadFile(filepath.Join(w.config.SecretDir, B2_ACCOUNT_KEY)); err == nil {
			w.sh.SetEnv(B2_ACCOUNT_KEY, string(v))
		}

	case ProviderRest:
		u, err := url.Parse(w.config.URL)
		if err != nil {
			return err
		}
		if username, err := ioutil.ReadFile(filepath.Join(w.config.SecretDir, REST_SERVER_USERNAME)); err == nil {
			if password, err := ioutil.ReadFile(filepath.Join(w.config.SecretDir, REST_SERVER_PASSWORD)); err == nil {
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

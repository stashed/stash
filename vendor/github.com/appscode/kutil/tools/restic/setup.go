package restic

import (
	"fmt"
	"io/ioutil"
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
	// For authentication based on tokens
	OS_STORAGE_URL = "OS_STORAGE_URL"
	OS_AUTH_TOKEN  = "OS_AUTH_TOKEN"

	// For using certs in Minio server or REST server
	CA_CERT_DATA = "CA_CERT_DATA"
)

func (w *ResticWrapper) SetupEnv(provider, bucket, endpoint, path, secretDir string) error {

	if v, err := ioutil.ReadFile(filepath.Join(secretDir, RESTIC_PASSWORD)); err != nil {
		return err
	} else {
		w.sh.SetEnv(RESTIC_PASSWORD, string(v))
	}

	if v, err := ioutil.ReadFile(filepath.Join(secretDir, CA_CERT_DATA)); err == nil {
		certDir := filepath.Join(w.scratchDir, "cacerts")
		if err2 := os.MkdirAll(certDir, 0755); err2 != nil {
			return err
		}

		w.cacertFile = filepath.Join(certDir, "ca.crt")
		if err3 := ioutil.WriteFile(w.cacertFile, v, 0755); err3 != nil {
			return err
		}
	}

	tmpDir := filepath.Join(w.scratchDir, "restic-tmp")
	if err := os.MkdirAll(tmpDir, 0755); err != nil {
		return err
	}
	w.sh.SetEnv(TMPDIR, tmpDir)

	//path = strings.TrimPrefix(path, "/")

	switch provider {

	case ProviderLocal:
		r := path
		if err := os.MkdirAll(r, 0755); err != nil {
			return err
		}
		w.sh.SetEnv(RESTIC_REPOSITORY, r)

	case ProviderS3:
		r := fmt.Sprintf("s3:%s/%s", endpoint, filepath.Join(bucket, path))
		w.sh.SetEnv(RESTIC_REPOSITORY, r)

		if v, err := ioutil.ReadFile(filepath.Join(secretDir, AWS_ACCESS_KEY_ID)); err == nil {
			w.sh.SetEnv(AWS_ACCESS_KEY_ID, string(v))
		}
		if v, err := ioutil.ReadFile(filepath.Join(secretDir, AWS_SECRET_ACCESS_KEY)); err == nil {
			w.sh.SetEnv(AWS_SECRET_ACCESS_KEY, string(v))
		}

	case ProviderGCS:
		r := fmt.Sprintf("gs:%s:/%s", bucket, path)
		w.sh.SetEnv(RESTIC_REPOSITORY, r)

		if v, err := ioutil.ReadFile(filepath.Join(secretDir, GOOGLE_PROJECT_ID)); err == nil {
			w.sh.SetEnv(GOOGLE_PROJECT_ID, string(v))
		}

		jsonKeyPath := filepath.Join(w.scratchDir, "gcs_sa.json")
		if v, err := ioutil.ReadFile(filepath.Join(secretDir, GOOGLE_SERVICE_ACCOUNT_JSON_KEY)); err == nil {
			err2 := ioutil.WriteFile(jsonKeyPath, v, 0600)
			if err != nil {
				return err2
			}
			w.sh.SetEnv(GOOGLE_APPLICATION_CREDENTIALS, jsonKeyPath)
		}

	case ProviderAzure:
		r := fmt.Sprintf("azure:%s:/%s", bucket, path)
		w.sh.SetEnv(RESTIC_REPOSITORY, r)

		if v, err := ioutil.ReadFile(filepath.Join(secretDir, AZURE_ACCOUNT_NAME)); err == nil {
			w.sh.SetEnv(AZURE_ACCOUNT_NAME, string(v))
		}
		if v, err := ioutil.ReadFile(filepath.Join(secretDir, AZURE_ACCOUNT_KEY)); err == nil {
			w.sh.SetEnv(AZURE_ACCOUNT_KEY, string(v))
		}

	case ProviderSwift:
		r := fmt.Sprintf("swift:%s:/%s", bucket, path)
		w.sh.SetEnv(RESTIC_REPOSITORY, r)

		// For keystone v1 authentication
		if v, err := ioutil.ReadFile(filepath.Join(secretDir, ST_AUTH)); err == nil {
			w.sh.SetEnv(ST_AUTH, string(v))
		}
		if v, err := ioutil.ReadFile(filepath.Join(secretDir, ST_USER)); err == nil {
			w.sh.SetEnv(ST_USER, string(v))
		}
		if v, err := ioutil.ReadFile(filepath.Join(secretDir, ST_KEY)); err == nil {
			w.sh.SetEnv(ST_KEY, string(v))
		}

		// For keystone v2 authentication (some variables are optional)
		if v, err := ioutil.ReadFile(filepath.Join(secretDir, OS_AUTH_URL)); err == nil {
			w.sh.SetEnv(OS_AUTH_URL, string(v))
		}
		if v, err := ioutil.ReadFile(filepath.Join(secretDir, OS_REGION_NAME)); err == nil {
			w.sh.SetEnv(OS_REGION_NAME, string(v))
		}
		if v, err := ioutil.ReadFile(filepath.Join(secretDir, OS_USERNAME)); err == nil {
			w.sh.SetEnv(OS_USERNAME, string(v))
		}
		if v, err := ioutil.ReadFile(filepath.Join(secretDir, OS_PASSWORD)); err == nil {
			w.sh.SetEnv(OS_PASSWORD, string(v))
		}
		if v, err := ioutil.ReadFile(filepath.Join(secretDir, OS_TENANT_ID)); err == nil {
			w.sh.SetEnv(OS_TENANT_ID, string(v))
		}
		if v, err := ioutil.ReadFile(filepath.Join(secretDir, OS_TENANT_NAME)); err == nil {
			w.sh.SetEnv(OS_TENANT_NAME, string(v))
		}

		// For keystone v3 authentication (some variables are optional)
		if v, err := ioutil.ReadFile(filepath.Join(secretDir, OS_USER_DOMAIN_NAME)); err == nil {
			w.sh.SetEnv(OS_USER_DOMAIN_NAME, string(v))
		}
		if v, err := ioutil.ReadFile(filepath.Join(secretDir, OS_PROJECT_NAME)); err == nil {
			w.sh.SetEnv(OS_PROJECT_NAME, string(v))
		}
		if v, err := ioutil.ReadFile(filepath.Join(secretDir, OS_PROJECT_DOMAIN_NAME)); err == nil {
			w.sh.SetEnv(OS_PROJECT_DOMAIN_NAME, string(v))
		}

		// For authentication based on tokens
		if v, err := ioutil.ReadFile(filepath.Join(secretDir, OS_STORAGE_URL)); err == nil {
			w.sh.SetEnv(OS_STORAGE_URL, string(v))
		}
		if v, err := ioutil.ReadFile(filepath.Join(secretDir, OS_AUTH_TOKEN)); err == nil {
			w.sh.SetEnv(OS_AUTH_TOKEN, string(v))
		}

	case ProviderB2:
		r := fmt.Sprintf("b2:%s:/%s", bucket, path)
		w.sh.SetEnv(RESTIC_REPOSITORY, r)

		if v, err := ioutil.ReadFile(filepath.Join(secretDir, B2_ACCOUNT_ID)); err == nil {
			w.sh.SetEnv(B2_ACCOUNT_ID, string(v))
		}

		if v, err := ioutil.ReadFile(filepath.Join(secretDir, B2_ACCOUNT_KEY)); err == nil {
			w.sh.SetEnv(B2_ACCOUNT_KEY, string(v))
		}
	}

	return nil
}

package scheduler

import (
	"errors"
	"fmt"
	"io/ioutil"
	"net/url"
	"os"
	"path/filepath"

	sapi "github.com/appscode/stash/api"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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
)

func (c *controller) SetEnvVars(resource *sapi.Restic) error {
	backend := resource.Spec.Backend
	if backend.RepositorySecretName == "" {
		return errors.New("Missing repository secret name")
	}
	secret, err := c.KubeClient.CoreV1().Secrets(resource.Namespace).Get(backend.RepositorySecretName, metav1.GetOptions{})
	if err != nil {
		return err
	}
	if v, ok := secret.Data[RESTIC_PASSWORD]; !ok {
		return errors.New("Missing repository password")
	} else {
		c.sh.SetEnv(RESTIC_PASSWORD, string(v))
	}

	hostname := ""
	if c.opt.PrefixHostname {
		hostname, err = os.Hostname()
		if err != nil {
			return err
		}
	}

	if backend.Local != nil {
		r := backend.Local.Path
		c.sh.SetEnv(RESTIC_REPOSITORY, filepath.Join(r, hostname))
	} else if backend.S3 != nil {
		r := fmt.Sprintf("s3:%s:%s:%s", backend.S3.Endpoint, backend.S3.Bucket, backend.S3.Prefix)
		c.sh.SetEnv(RESTIC_REPOSITORY, filepath.Join(r, hostname))
		c.sh.SetEnv(AWS_ACCESS_KEY_ID, string(secret.Data[AWS_ACCESS_KEY_ID]))
		c.sh.SetEnv(AWS_SECRET_ACCESS_KEY, string(secret.Data[AWS_SECRET_ACCESS_KEY]))
	} else if backend.GCS != nil {
		r := fmt.Sprintf("gs:%s:%s:%s", backend.GCS.Location, backend.GCS.Bucket, backend.GCS.Prefix)
		c.sh.SetEnv(RESTIC_REPOSITORY, filepath.Join(r, hostname))
		c.sh.SetEnv(GOOGLE_PROJECT_ID, string(secret.Data[GOOGLE_PROJECT_ID]))
		jsonKeyPath := filepath.Join(c.opt.ScratchDir, "gcs_sa.json")
		err = ioutil.WriteFile(jsonKeyPath, secret.Data[GOOGLE_SERVICE_ACCOUNT_JSON_KEY], 600)
		if err != nil {
			return err
		}
		c.sh.SetEnv(GOOGLE_APPLICATION_CREDENTIALS, jsonKeyPath)
	} else if backend.Azure != nil {
		r := fmt.Sprintf("azure:%s:%s", backend.Azure.Container, backend.Azure.Prefix)
		c.sh.SetEnv(RESTIC_REPOSITORY, filepath.Join(r, hostname))
		c.sh.SetEnv(AZURE_ACCOUNT_NAME, string(secret.Data[AZURE_ACCOUNT_NAME]))
		c.sh.SetEnv(AZURE_ACCOUNT_KEY, string(secret.Data[AZURE_ACCOUNT_KEY]))
	} else if backend.Swift != nil {
		r := fmt.Sprintf("swift:%s:%s", backend.Swift.Container, backend.Swift.Prefix)
		c.sh.SetEnv(RESTIC_REPOSITORY, filepath.Join(r, hostname))
		// For keystone v1 authentication
		c.sh.SetEnv(ST_AUTH, string(secret.Data[ST_AUTH]))
		c.sh.SetEnv(ST_USER, string(secret.Data[ST_USER]))
		c.sh.SetEnv(ST_KEY, string(secret.Data[ST_KEY]))
		// For keystone v2 authentication (some variables are optional)
		c.sh.SetEnv(OS_AUTH_URL, string(secret.Data[OS_AUTH_URL]))
		c.sh.SetEnv(OS_REGION_NAME, string(secret.Data[OS_REGION_NAME]))
		c.sh.SetEnv(OS_USERNAME, string(secret.Data[OS_USERNAME]))
		c.sh.SetEnv(OS_PASSWORD, string(secret.Data[OS_PASSWORD]))
		c.sh.SetEnv(OS_TENANT_ID, string(secret.Data[OS_TENANT_ID]))
		c.sh.SetEnv(OS_TENANT_NAME, string(secret.Data[OS_TENANT_NAME]))
		// For keystone v3 authentication (some variables are optional)
		c.sh.SetEnv(OS_USER_DOMAIN_NAME, string(secret.Data[OS_USER_DOMAIN_NAME]))
		c.sh.SetEnv(OS_PROJECT_NAME, string(secret.Data[AZURE_ACCOUNT_NAME]))
		c.sh.SetEnv(AZURE_ACCOUNT_NAME, string(secret.Data[OS_PROJECT_NAME]))
		c.sh.SetEnv(OS_PROJECT_DOMAIN_NAME, string(secret.Data[OS_PROJECT_DOMAIN_NAME]))
		// For authentication based on tokens
		c.sh.SetEnv(OS_STORAGE_URL, string(secret.Data[OS_STORAGE_URL]))
		c.sh.SetEnv(OS_AUTH_TOKEN, string(secret.Data[OS_AUTH_TOKEN]))
	} else if backend.Rest != nil {
		u, err := url.Parse(backend.Rest.URL)
		if err != nil {
			return err
		}
		if username, ok := secret.Data[REST_SERVER_USERNAME]; ok {
			if password, ok := secret.Data[REST_SERVER_PASSWORD]; ok {
				u.User = url.UserPassword(string(username), string(password))
			} else {
				u.User = url.User(string(username))
			}
		}
		u.Path = filepath.Join(u.Path, hostname) // TODO: check
		r := fmt.Sprintf("rest:%s", u.String())
		c.sh.SetEnv(RESTIC_REPOSITORY, r)
	} else if backend.B2 != nil {
		r := fmt.Sprintf("b2:%s:%s", backend.B2.Bucket, backend.B2.Prefix)
		c.sh.SetEnv(RESTIC_REPOSITORY, filepath.Join(r, hostname))
		c.sh.SetEnv(B2_ACCOUNT_ID, string(secret.Data[B2_ACCOUNT_ID]))
		c.sh.SetEnv(B2_ACCOUNT_KEY, string(secret.Data[B2_ACCOUNT_KEY]))
	}
	return nil
}

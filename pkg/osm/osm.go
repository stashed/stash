package osm
import (
	"bytes"
	"net/url"
	"strconv"
	"strings"

	stringz "github.com/appscode/go/strings"
	"github.com/appscode/go/types"
	otx "github.com/appscode/osm/context"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/credentials/stscreds"
	"github.com/aws/aws-sdk-go/aws/session"
	_s3 "github.com/aws/aws-sdk-go/service/s3"
	"github.com/ghodss/yaml"
	"github.com/graymeta/stow"
	"github.com/graymeta/stow/azure"
	gcs "github.com/graymeta/stow/google"
	"github.com/graymeta/stow/local"
	"github.com/graymeta/stow/s3"
	"github.com/graymeta/stow/swift"
	"github.com/pkg/errors"
	core "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	api "github.com/appscode/stash/apis/stash/v1alpha1"
)

const (
	SecretMountPath = "/etc/osm"
)

func NewOSMSecret(client kubernetes.Interface, backend api.Backend) (*core.Secret, error) {
	osmCtx, err := NewOSMContext(client, snapshot.Spec.SnapshotStorageSpec, snapshot.Namespace)
	if err != nil {
		return nil, err
	}
	osmCfg := &otx.OSMConfig{
		CurrentContext: osmCtx.Name,
		Contexts:       []*otx.Context{osmCtx},
	}
	osmBytes, err := yaml.Marshal(osmCfg)
	if err != nil {
		return nil, err
	}
	return &core.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      snapshot.OSMSecretName(),
			Namespace: snapshot.Namespace,
		},
		Data: map[string][]byte{
			"config": osmBytes,
		},
	}, nil
}

func CheckBucketAccess(client kubernetes.Interface, repository *api.Repository, namespace string) error {
	cfg, err := NewOSMContext(client, repository.Spec.Backend, namespace)
	if err != nil {
		return err
	}
	loc, err := stow.Dial(cfg.Provider, cfg.Config)
	if err != nil {
		return err
	}
	c, err := repository.Spec.Backend
	if err != nil {
		return err
	}
	container, err := loc.Container(c)
	if err != nil {
		return err
	}
	r := bytes.NewReader([]byte("CheckBucketAccess"))
	item, err := container.Put(".kubedb", r, r.Size(), nil)
	if err != nil {
		return err
	}
	if err := container.RemoveItem(item.ID()); err != nil {
		return err
	}
	return nil
}

func NewOSMContext(client kubernetes.Interface, repository *api.Repository, namespace string) (*otx.Context, error) {
	config := make(map[string][]byte)

	if repository.Spec.Backend.StorageSecretName != "" {
		secret, err := client.CoreV1().Secrets(namespace).Get(repository.Spec.Backend.StorageSecretName, metav1.GetOptions{})
		if err != nil {
			return nil, err
		}
		config = secret.Data
	}

	nc := &otx.Context{
		Name:   "stash",
		Config: stow.ConfigMap{},
	}

	if repository.Spec.Backend.S3 != nil {
		nc.Provider = s3.Kind

		keyID, foundKeyID := config[api.AWS_ACCESS_KEY_ID]
		key, foundKey := config[api.AWS_SECRET_ACCESS_KEY]
		if foundKey && foundKeyID {
			nc.Config[s3.ConfigAccessKeyID] = string(keyID)
			nc.Config[s3.ConfigSecretKey] = string(key)
			nc.Config[s3.ConfigAuthType] = "accesskey"
		} else {
			nc.Config[s3.ConfigAuthType] = "iam"
		}
		if strings.HasSuffix(repository.Spec.Backend.S3.Endpoint, ".amazonaws.com") {
			// find region
			var sess *session.Session
			var err error
			if nc.Config[s3.ConfigAuthType] == "iam" {
				sess, err = session.NewSessionWithOptions(session.Options{
					Config: *aws.NewConfig(),
					// Support MFA when authing using assumed roles.
					SharedConfigState:       session.SharedConfigEnable,
					AssumeRoleTokenProvider: stscreds.StdinTokenProvider,
				})
			} else {
				config := &aws.Config{
					Credentials: credentials.NewStaticCredentials(string(keyID), string(key), ""),
					Region:      aws.String("us-east-1"),
				}
				sess, err = session.NewSessionWithOptions(session.Options{
					Config: *config,
					// Support MFA when authing using assumed roles.
					SharedConfigState:       session.SharedConfigEnable,
					AssumeRoleTokenProvider: stscreds.StdinTokenProvider,
				})
			}
			if err != nil {
				return nil, err
			}
			svc := _s3.New(sess)
			out, err := svc.GetBucketLocation(&_s3.GetBucketLocationInput{
				Bucket: types.StringP(repository.Spec.Backend.S3.Bucket),
			})
			nc.Config[s3.ConfigRegion] = stringz.Val(types.String(out.LocationConstraint), "us-east-1")
		} else {
			nc.Config[s3.ConfigEndpoint] = repository.Spec.Backend.S3.Endpoint
			if u, err := url.Parse(repository.Spec.Backend.S3.Endpoint); err == nil {
				nc.Config[s3.ConfigDisableSSL] = strconv.FormatBool(u.Scheme == "http")
			}
		}
		return nc, nil
	} else if repository.Spec.Backend.GCS != nil {
		nc.Provider = gcs.Kind
		nc.Config[gcs.ConfigProjectId] = string(config[api.GOOGLE_PROJECT_ID])
		nc.Config[gcs.ConfigJSON] = string(config[api.GOOGLE_SERVICE_ACCOUNT_JSON_KEY])
		return nc, nil
	} else if repository.Spec.Backend.Azure != nil {
		nc.Provider = azure.Kind
		nc.Config[azure.ConfigAccount] = string(config[api.AZURE_ACCOUNT_NAME])
		nc.Config[azure.ConfigKey] = string(config[api.AZURE_ACCOUNT_KEY])
		return nc, nil
	} else if repository.Spec.Backend.Local != nil {
		nc.Provider = local.Kind
		nc.Config[local.ConfigKeyPath] = repository.Spec.Backend.Local.MountPath
		return nc, nil
	} else if repository.Spec.Backend.Swift != nil {
		nc.Provider = swift.Kind
		// https://github.com/restic/restic/blob/master/src/restic/backend/swift/config.go
		for _, val := range []struct {
			stowKey   string
			secretKey string
		}{
			// v2/v3 repository.Spec.Backendific
			{swift.ConfigUsername, api.OS_USERNAME},
			{swift.ConfigKey, api.OS_PASSWORD},
			{swift.ConfigRegion, api.OS_REGION_NAME},
			{swift.ConfigTenantAuthURL, api.OS_AUTH_URL},

			// v3 repository.Spec.Backendific
			{swift.ConfigDomain, api.OS_USER_DOMAIN_NAME},
			{swift.ConfigTenantName, api.OS_PROJECT_NAME},
			{swift.ConfigTenantDomain, api.OS_PROJECT_DOMAIN_NAME},

			// v2 repository.Spec.Backendific
			{swift.ConfigTenantId, api.OS_TENANT_ID},
			{swift.ConfigTenantName, api.OS_TENANT_NAME},

			// v1 repository.Spec.Backendific
			{swift.ConfigTenantAuthURL, api.ST_AUTH},
			{swift.ConfigUsername, api.ST_USER},
			{swift.ConfigKey, api.ST_KEY},

			// Manual authentication
			{swift.ConfigStorageURL, api.OS_STORAGE_URL},
			{swift.ConfigAuthToken, api.OS_AUTH_TOKEN},
		} {
			if _, exists := nc.Config.Config(val.stowKey); !exists {
				nc.Config[val.stowKey] = string(config[val.secretKey])
			}
		}
		return nc, nil
	}
	return nil, errors.New("no storage provider is configured")
}

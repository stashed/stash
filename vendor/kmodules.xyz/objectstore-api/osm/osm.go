package osm

import (
	"net/url"
	"path/filepath"
	"strconv"
	"strings"

	stringz "github.com/appscode/go/strings"
	"github.com/appscode/go/types"
	otx "github.com/appscode/osm/context"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/credentials/stscreds"
	"github.com/aws/aws-sdk-go/aws/ec2metadata"
	"github.com/aws/aws-sdk-go/aws/session"
	_s3 "github.com/aws/aws-sdk-go/service/s3"
	"github.com/ghodss/yaml"
	"github.com/pkg/errors"
	"gomodules.xyz/stow"
	"gomodules.xyz/stow/azure"
	gcs "gomodules.xyz/stow/google"
	"gomodules.xyz/stow/local"
	"gomodules.xyz/stow/s3"
	"gomodules.xyz/stow/swift"
	core "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	api "kmodules.xyz/objectstore-api/api/v1"
)

const (
	SecretMountPath = "/etc/osm"
	CaCertFileName  = "ca.crt"
)

// NewOSMSecret creates a secret that contains the config file of OSM.
// So, generally, if this secret is mounted in `etc/osm`,
// the tree of `/etc/osm` directory will be similar to,
//
// /etc/osm
// └── config
//
// However, if the EndPoint is `S3 Minio Server`, then the secret will contain two file,
// `config` and `ca.crt`. So, the tree of the file path will look as,
//
// /etc/osm
// ├── ca.crt
// └── config

func NewOSMSecret(client kubernetes.Interface, name, namespace string, spec api.Backend) (*core.Secret, error) {
	osmCtx, err := NewOSMContext(client, spec, namespace)
	if err != nil {
		return nil, err
	}
	cacertData, certDataFound := osmCtx.Config[s3.ConfigCACertData]
	if certDataFound {
		// assume that CA cert file is mounted at SecretMountPath directory
		osmCtx.Config[s3.ConfigCACertFile] = filepath.Join(SecretMountPath, CaCertFileName)
		delete(osmCtx.Config, s3.ConfigCACertData)
	}

	osmCfg := &otx.OSMConfig{
		CurrentContext: osmCtx.Name,
		Contexts:       []*otx.Context{osmCtx},
	}
	osmBytes, err := yaml.Marshal(osmCfg)
	if err != nil {
		return nil, err
	}
	out := &core.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Data: map[string][]byte{
			"config": osmBytes,
		},
	}
	if certDataFound {
		// inject ca cert data as a file into the osm secret so that CaCertFileName exists.
		out.Data[CaCertFileName] = []byte(cacertData)
	}
	return out, nil
}

func CheckBucketAccess(client kubernetes.Interface, spec api.Backend, namespace string) error {
	cfg, err := NewOSMContext(client, spec, namespace)
	if err != nil {
		return err
	}
	loc, err := stow.Dial(cfg.Provider, cfg.Config)
	if err != nil {
		return err
	}
	bucket, err := spec.Container()
	if err != nil {
		return err
	}
	c, err := loc.Container(bucket)
	if err != nil {
		return err
	}
	return c.HasWriteAccess()
}

func NewOSMContext(client kubernetes.Interface, spec api.Backend, namespace string) (*otx.Context, error) {
	config := make(map[string][]byte)

	if spec.StorageSecretName != "" {
		secret, err := client.CoreV1().Secrets(namespace).Get(spec.StorageSecretName, metav1.GetOptions{})
		if err != nil {
			return nil, err
		}
		config = secret.Data
	}

	nc := &otx.Context{
		Name:   "objectstore",
		Config: stow.ConfigMap{},
	}

	if spec.S3 != nil {
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
		if spec.S3.Endpoint == "" || strings.HasSuffix(spec.S3.Endpoint, ".amazonaws.com") {
			// Using s3 and not s3-compatible service like minio or rook, etc. Now, find region
			var sess *session.Session
			var err error
			if nc.Config[s3.ConfigAuthType] == "iam" {
				// The aws sdk does not currently support automatically setting the region based on an instances placement.
				// This automatically sets region based on ec2 instance metadata when running on EC2.
				// ref: https://docs.aws.amazon.com/sdk-for-javascript/v2/developer-guide/setting-region.html#setting-region-order-of-precedence
				var c aws.Config
				if s, e := session.NewSession(); e == nil {
					if region, e := ec2metadata.New(s).Region(); e == nil {
						c.WithRegion(region)
					}
				}
				sess, err = session.NewSessionWithOptions(session.Options{
					Config: c,
					// Support MFA when authing using assumed roles.
					SharedConfigState:       session.SharedConfigEnable,
					AssumeRoleTokenProvider: stscreds.StdinTokenProvider,
				})
			} else {
				sess, err = session.NewSessionWithOptions(session.Options{
					Config: aws.Config{
						Credentials: credentials.NewStaticCredentials(string(keyID), string(key), ""),
						Region:      aws.String("us-east-1"),
					},
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
				Bucket: types.StringP(spec.S3.Bucket),
			})
			if err != nil {
				return nil, err
			}
			nc.Config[s3.ConfigRegion] = stringz.Val(types.String(out.LocationConstraint), "us-east-1")
		} else {
			nc.Config[s3.ConfigEndpoint] = spec.S3.Endpoint
			u, err := url.Parse(spec.S3.Endpoint)
			if err != nil {
				return nil, err
			}
			nc.Config[s3.ConfigDisableSSL] = strconv.FormatBool(u.Scheme == "http")

			cacertData, ok := config[api.CA_CERT_DATA]
			if ok && u.Scheme == "https" {
				nc.Config[s3.ConfigCACertData] = string(cacertData)
			}
		}
		return nc, nil
	} else if spec.GCS != nil {
		nc.Provider = gcs.Kind
		nc.Config[gcs.ConfigProjectId] = string(config[api.GOOGLE_PROJECT_ID])
		nc.Config[gcs.ConfigJSON] = string(config[api.GOOGLE_SERVICE_ACCOUNT_JSON_KEY])
		return nc, nil
	} else if spec.Azure != nil {
		nc.Provider = azure.Kind
		nc.Config[azure.ConfigAccount] = string(config[api.AZURE_ACCOUNT_NAME])
		nc.Config[azure.ConfigKey] = string(config[api.AZURE_ACCOUNT_KEY])
		return nc, nil
	} else if spec.Local != nil {
		nc.Provider = local.Kind
		nc.Config[local.ConfigKeyPath] = spec.Local.MountPath
		return nc, nil
	} else if spec.Swift != nil {
		nc.Provider = swift.Kind
		// https://github.com/restic/restic/blob/master/src/restic/backend/swift/config.go
		for _, val := range []struct {
			stowKey   string
			secretKey string
		}{
			// v2/v3 specific
			{swift.ConfigUsername, api.OS_USERNAME},
			{swift.ConfigKey, api.OS_PASSWORD},
			{swift.ConfigRegion, api.OS_REGION_NAME},
			{swift.ConfigTenantAuthURL, api.OS_AUTH_URL},

			// v3 specific
			{swift.ConfigDomain, api.OS_USER_DOMAIN_NAME},
			{swift.ConfigTenantName, api.OS_PROJECT_NAME},
			{swift.ConfigTenantDomain, api.OS_PROJECT_DOMAIN_NAME},

			// v2 specific
			{swift.ConfigTenantId, api.OS_TENANT_ID},
			{swift.ConfigTenantName, api.OS_TENANT_NAME},

			// v1 specific
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

package s3

import (
	"crypto/tls"
	"crypto/x509"
	"io/ioutil"
	"net"
	"net/http"
	"net/url"
	"time"

	"fmt"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/graymeta/stow"
	"github.com/pkg/errors"
)

// Kind represents the name of the location/storage type.
const Kind = "s3"

var (
	authTypeAccessKey = "accesskey"
	authTypeIAM       = "iam"
)

const (
	// ConfigAuthType is an optional argument that defines whether to use an IAM role or access key based auth
	ConfigAuthType = "auth_type"

	// ConfigAccessKeyID is one key of a pair of AWS credentials.
	ConfigAccessKeyID = "access_key_id"

	// ConfigSecretKey is one key of a pair of AWS credentials.
	ConfigSecretKey = "secret_key"

	// ConfigToken is an optional argument which is required when providing
	// credentials with temporary access.
	// ConfigToken = "token"

	// ConfigRegion represents the region/availability zone of the session.
	ConfigRegion = "region"

	// ConfigEndpoint is optional config value for changing s3 endpoint
	// used for e.g. minio.io
	ConfigEndpoint = "endpoint"

	// ConfigCACertFile is optional config value for providing path to cacert file for custom endpoint like Minio
	// to establish TLS secure connection
	ConfigCACertFile = "cacert_file"

	// ConfigDisableSSL is optional config value for disabling SSL support on custom endpoints
	// Its default value is "false", to disable SSL set it to "true".
	ConfigDisableSSL = "disable_ssl"
)

func init() {
	validatefn := func(config stow.Config) error {
		authType, ok := config.Config(ConfigAuthType)
		if !ok || authType == "" {
			authType = authTypeAccessKey
		}

		if !(authType == authTypeAccessKey || authType == authTypeIAM) {
			return errors.New("invalid auth_type")
		}

		if authType == authTypeAccessKey {
			_, ok := config.Config(ConfigAccessKeyID)
			if !ok {
				return errors.New("missing Access Key ID")
			}

			_, ok = config.Config(ConfigSecretKey)
			if !ok {
				return errors.New("missing Secret Key")
			}
		}
		return nil
	}
	makefn := func(config stow.Config) (stow.Location, error) {

		authType, ok := config.Config(ConfigAuthType)
		if !ok || authType == "" {
			authType = authTypeAccessKey
		}

		if !(authType == authTypeAccessKey || authType == authTypeIAM) {
			return nil, errors.New("invalid auth_type")
		}

		if authType == authTypeAccessKey {
			_, ok := config.Config(ConfigAccessKeyID)
			if !ok {
				return nil, errors.New("missing Access Key ID")
			}

			_, ok = config.Config(ConfigSecretKey)
			if !ok {
				return nil, errors.New("missing Secret Key")
			}
		}

		// Create a new client (s3 session)
		client, endpoint, err := newS3Client(config)
		if err != nil {
			return nil, err
		}

		// Create a location with given config and client (s3 session).
		loc := &location{
			config:         config,
			client:         client,
			customEndpoint: endpoint,
		}

		return loc, nil
	}

	kindfn := func(u *url.URL) bool {
		return u.Scheme == Kind
	}

	stow.Register(Kind, makefn, kindfn, validatefn)
}

// Attempts to create a session based on the information given.
func newS3Client(config stow.Config) (client *s3.S3, endpoint string, err error) {
	authType, _ := config.Config(ConfigAuthType)
	accessKeyID, _ := config.Config(ConfigAccessKeyID)
	secretKey, _ := config.Config(ConfigSecretKey)
	//	token, _ := config.Config(ConfigToken)

	if authType == "" {
		authType = authTypeAccessKey
	}

	awsConfig := aws.NewConfig().
		WithHTTPClient(http.DefaultClient).
		WithMaxRetries(aws.UseServiceDefaultRetries).
		WithLogger(aws.NewDefaultLogger()).
		WithLogLevel(aws.LogOff).
		WithSleepDelay(time.Sleep)

	region, ok := config.Config(ConfigRegion)
	if ok {
		awsConfig.WithRegion(region)
	} else {
		awsConfig.WithRegion("us-east-1")
	}

	if authType == authTypeAccessKey {
		awsConfig.WithCredentials(credentials.NewStaticCredentials(accessKeyID, secretKey, ""))
	}

	endpoint, ok = config.Config(ConfigEndpoint)
	if ok {
		awsConfig.WithEndpoint(endpoint).
			WithS3ForcePathStyle(true)
	}

	disableSSL, ok := config.Config(ConfigDisableSSL)
	if ok && disableSSL == "true" {
		awsConfig.WithDisableSSL(true)
	}

	cacertFile, ok := config.Config(ConfigCACertFile)
	if ok {
		awsConfig.HTTPClient.Transport, err = newSecureTransport(cacertFile)
		if err != nil {
			return nil, "", err
		}
	}

	sess, err := session.NewSession(awsConfig)
	if err != nil {
		return nil, "", fmt.Errorf("failed to create S3 session. Reason: %s", err)
	}

	s3Client := s3.New(sess)

	return s3Client, endpoint, nil
}

func newSecureTransport(cacertFile string) (http.RoundTripper, error) {
	if cacertFile == "" {
		return nil, fmt.Errorf("invalid root certificate directory")
	}

	tr := &http.Transport{
		Proxy: http.ProxyFromEnvironment,
		DialContext: (&net.Dialer{
			Timeout:   30 * time.Second,
			KeepAlive: 30 * time.Second,
			DualStack: true,
		}).DialContext,
		MaxIdleConns:          100,
		MaxIdleConnsPerHost:   100,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
		TLSClientConfig:       &tls.Config{},
	}

	pool := x509.NewCertPool()

	cacert, err := ioutil.ReadFile(cacertFile)
	if err != nil {
		return nil, errors.Errorf("unable to read root certificate: %v", err)
	}
	if ok := pool.AppendCertsFromPEM(cacert); !ok {
		return nil, errors.Errorf("cannot parse root certificate from %q", cacertFile)
	}
	tr.TLSClientConfig.RootCAs = pool

	return tr, nil
}

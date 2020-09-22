package swift

import (
	"errors"
	"fmt"
	"net/http"
	"net/url"

	"github.com/ncw/swift"
	"gomodules.xyz/stow"
)

// Config key constants.
const (
	ConfigUsername      = "username"
	ConfigKey           = "key"
	ConfigTenantName    = "tenant_name"
	ConfigTenantAuthURL = "tenant_auth_url"
	ConfigDomain        = "domain"
	ConfigRegion        = "region"
	ConfigTenantId      = "tenant_id"
	ConfigTenantDomain  = "tenant_domain"
	ConfigTrustId       = "trust_id"
	ConfigStorageURL    = "storage_url"
	ConfigAuthToken     = "auth_token"
)

// Kind is the kind of Location this package provides.
const Kind = "swift"

func init() {
	validatefn := func(config stow.Config) error {
		_, ok := config.Config(ConfigUsername)
		if !ok {
			return errors.New("missing account username")
		}
		_, ok = config.Config(ConfigKey)
		if !ok {
			return errors.New("missing api key")
		}
		_, ok = config.Config(ConfigTenantName)
		if !ok {
			return errors.New("missing tenant name")
		}
		_, ok = config.Config(ConfigTenantAuthURL)
		if !ok {
			return errors.New("missing tenant auth url")
		}
		return nil
	}
	makefn := func(config stow.Config) (stow.Location, error) {
		_, ok := config.Config(ConfigUsername)
		if !ok {
			return nil, errors.New("missing account username")
		}
		_, ok = config.Config(ConfigKey)
		if !ok {
			return nil, errors.New("missing api key")
		}
		_, ok = config.Config(ConfigTenantName)
		if !ok {
			return nil, errors.New("missing tenant name")
		}
		_, ok = config.Config(ConfigTenantAuthURL)
		if !ok {
			return nil, errors.New("missing tenant auth url")
		}
		l := &location{
			config: config,
		}
		var err error
		l.client, err = newSwiftClient(l.config)
		if err != nil {
			return nil, err
		}
		return l, nil
	}
	kindfn := func(u *url.URL) bool {
		return u.Scheme == Kind
	}
	stow.Register(Kind, makefn, kindfn, validatefn)
}

func newSwiftClient(cfg stow.Config) (*swift.Connection, error) {
	username, _ := cfg.Config(ConfigUsername)
	key, _ := cfg.Config(ConfigKey)
	tenantName, _ := cfg.Config(ConfigTenantName)
	tenantAuthURL, _ := cfg.Config(ConfigTenantAuthURL)
	domain, _ := cfg.Config(ConfigDomain)
	region, _ := cfg.Config(ConfigRegion)
	tenantId, _ := cfg.Config(ConfigTenantId)
	tenantDomain, _ := cfg.Config(ConfigTenantDomain)
	trustId, _ := cfg.Config(ConfigTrustId)
	storageURL, _ := cfg.Config(ConfigStorageURL)
	authToken, _ := cfg.Config(ConfigAuthToken)
	client := swift.Connection{
		UserName:     username,
		ApiKey:       key,
		AuthUrl:      tenantAuthURL,
		Tenant:       tenantName, // Name of the tenant (v2 auth only)
		Domain:       domain,
		Region:       region,
		TenantId:     tenantId,
		TenantDomain: tenantDomain,
		TrustId:      trustId,
		StorageUrl:   storageURL,
		AuthToken:    authToken,
		// Add Default transport
		Transport: http.DefaultTransport,
	}
	err := client.Authenticate()
	if err != nil {
		return nil, fmt.Errorf("unable to authenticate, reason: %v", err)
	}
	return &client, nil
}

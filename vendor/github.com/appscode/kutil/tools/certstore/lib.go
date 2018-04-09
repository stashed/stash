package certstore

import (
	"crypto/rsa"
	"crypto/x509"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strings"

	netz "github.com/appscode/go/net"
	"github.com/pkg/errors"
	"github.com/spf13/afero"
	"k8s.io/client-go/util/cert"
)

type CertStore struct {
	fs           afero.Fs
	dir          string
	organization []string
	prefix       string
	ca           string
	caKey        *rsa.PrivateKey
	caCert       *x509.Certificate
}

func NewCertStore(fs afero.Fs, dir string, organization ...string) (*CertStore, error) {
	err := fs.MkdirAll(dir, 0755)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to create dir `%s`", dir)
	}
	return &CertStore{fs: fs, dir: dir, ca: "ca", organization: append([]string(nil), organization...)}, nil
}

func (s *CertStore) InitCA(prefix ...string) error {
	err := s.LoadCA(prefix...)
	if err == nil {
		return nil
	}
	return s.NewCA(prefix...)
}

func (s *CertStore) LoadCA(prefix ...string) error {
	if err := s.prep(prefix...); err != nil {
		return err
	}

	if s.PairExists(s.ca) {
		var err error
		s.caCert, s.caKey, err = s.Read(s.ca)
		return err
	}

	// only ca key found, extract ca cert from it.
	if _, err := s.fs.Stat(s.KeyFile(s.ca)); err == nil {
		keyBytes, err := afero.ReadFile(s.fs, s.KeyFile(s.ca))
		if err != nil {
			return errors.Wrapf(err, "failed to read private key `%s`", s.KeyFile(s.ca))
		}
		key, err := cert.ParsePrivateKeyPEM(keyBytes)
		if err != nil {
			return errors.Wrapf(err, "failed to parse private key `%s`", s.KeyFile(s.ca))
		}
		rsaKey, ok := key.(*rsa.PrivateKey)
		if !ok {
			return errors.Errorf("private key `%s` is not a rsa private key", s.KeyFile(s.ca))
		}
		return s.createCAFromKey(rsaKey)
	}

	return os.ErrNotExist
}

func (s *CertStore) NewCA(prefix ...string) error {
	if err := s.prep(prefix...); err != nil {
		return err
	}

	key, err := cert.NewPrivateKey()
	if err != nil {
		return errors.Wrap(err, "failed to generate private key")
	}
	return s.createCAFromKey(key)
}

func (s *CertStore) prep(prefix ...string) error {
	switch len(prefix) {
	case 0:
		s.prefix = ""
	case 1:
		s.prefix = strings.ToLower(strings.Trim(strings.TrimSpace(prefix[0]), "-")) + "-"
	default:
		return fmt.Errorf("multiple ca prefix given: %v", prefix)
	}
	return nil
}

func (s *CertStore) createCAFromKey(key *rsa.PrivateKey) error {
	var err error

	cfg := cert.Config{
		CommonName:   s.ca,
		Organization: s.organization,
		AltNames: cert.AltNames{
			IPs: []net.IP{net.ParseIP("127.0.0.1")},
		},
	}
	crt, err := cert.NewSelfSignedCACert(cfg, key)
	if err != nil {
		return errors.Wrap(err, "failed to generate self-signed certificate")
	}
	err = s.Write(s.ca, crt, key)
	if err != nil {
		return err
	}

	s.caCert = crt
	s.caKey = key
	return nil
}

func (s *CertStore) Location() string {
	return s.dir
}

func (s *CertStore) CAName() string {
	return s.ca
}

func (s *CertStore) CACert() []byte {
	return cert.EncodeCertPEM(s.caCert)
}

func (s *CertStore) CAKey() []byte {
	return cert.EncodePrivateKeyPEM(s.caKey)
}

func (s *CertStore) NewHostCertPair() ([]byte, []byte, error) {
	var sans cert.AltNames
	publicIPs, privateIPs, _ := netz.HostIPs()
	for _, ip := range publicIPs {
		sans.IPs = append(sans.IPs, net.ParseIP(ip))
	}
	for _, ip := range privateIPs {
		sans.IPs = append(sans.IPs, net.ParseIP(ip))
	}
	return s.NewServerCertPair("127.0.0.1", sans)
}

func (s *CertStore) NewServerCertPair(cn string, sans cert.AltNames) ([]byte, []byte, error) {
	cfg := cert.Config{
		CommonName:   cn,
		Organization: s.organization,
		AltNames:     sans,
		Usages:       []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
	}
	key, err := cert.NewPrivateKey()
	if err != nil {
		return nil, nil, errors.Wrap(err, "failed to generate private key")
	}
	crt, err := cert.NewSignedCert(cfg, key, s.caCert, s.caKey)
	if err != nil {
		return nil, nil, errors.Wrap(err, "failed to generate server certificate")
	}
	return cert.EncodeCertPEM(crt), cert.EncodePrivateKeyPEM(key), nil
}

// NewPeerCertPair is used to create cert pair that can serve as both server and client.
// This is used to issue peer certificates for etcd.
func (s *CertStore) NewPeerCertPair(cn string, sans cert.AltNames) ([]byte, []byte, error) {
	cfg := cert.Config{
		CommonName:   cn,
		Organization: s.organization,
		AltNames:     sans,
		Usages:       []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth, x509.ExtKeyUsageClientAuth},
	}
	key, err := cert.NewPrivateKey()
	if err != nil {
		return nil, nil, errors.Wrap(err, "failed to generate private key")
	}
	crt, err := cert.NewSignedCert(cfg, key, s.caCert, s.caKey)
	if err != nil {
		return nil, nil, errors.Wrap(err, "failed to generate peer certificate")
	}
	return cert.EncodeCertPEM(crt), cert.EncodePrivateKeyPEM(key), nil
}

func (s *CertStore) NewClientCertPair(cn string, organization ...string) ([]byte, []byte, error) {
	cfg := cert.Config{
		CommonName:   cn,
		Organization: organization,
		Usages:       []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
	}
	key, err := cert.NewPrivateKey()
	if err != nil {
		return nil, nil, errors.Wrap(err, "failed to generate private key")
	}
	crt, err := cert.NewSignedCert(cfg, key, s.caCert, s.caKey)
	if err != nil {
		return nil, nil, errors.Wrap(err, "failed to generate client certificate")
	}
	return cert.EncodeCertPEM(crt), cert.EncodePrivateKeyPEM(key), nil
}

func (s *CertStore) IsExists(name string) bool {
	if _, err := s.fs.Stat(s.CertFile(name)); err == nil {
		return true
	}
	if _, err := s.fs.Stat(s.KeyFile(name)); err == nil {
		return true
	}
	return false
}

func (s *CertStore) PairExists(name string) bool {
	if _, err := s.fs.Stat(s.CertFile(name)); err == nil {
		if _, err := s.fs.Stat(s.KeyFile(name)); err == nil {
			return true
		}
	}
	return false
}

func (s *CertStore) CertFile(name string) string {
	filename := strings.ToLower(name) + ".crt"
	if s.prefix != "" {
		filename = s.prefix + filename
	}
	return filepath.Join(s.dir, filename)
}

func (s *CertStore) KeyFile(name string) string {
	filename := strings.ToLower(name) + ".key"
	if s.prefix != "" {
		filename = s.prefix + filename
	}
	return filepath.Join(s.dir, filename)
}

func (s *CertStore) Write(name string, crt *x509.Certificate, key *rsa.PrivateKey) error {
	if err := afero.WriteFile(s.fs, s.CertFile(name), cert.EncodeCertPEM(crt), 0644); err != nil {
		return errors.Wrapf(err, "failed to write `%s`", s.CertFile(name))
	}
	if err := afero.WriteFile(s.fs, s.KeyFile(name), cert.EncodePrivateKeyPEM(key), 0600); err != nil {
		return errors.Wrapf(err, "failed to write `%s`", s.KeyFile(name))
	}
	return nil
}

func (s *CertStore) WriteBytes(name string, crt, key []byte) error {
	if err := afero.WriteFile(s.fs, s.CertFile(name), crt, 0644); err != nil {
		return errors.Wrapf(err, "failed to write `%s`", s.CertFile(name))
	}
	if err := afero.WriteFile(s.fs, s.KeyFile(name), key, 0600); err != nil {
		return errors.Wrapf(err, "failed to write `%s`", s.KeyFile(name))
	}
	return nil
}

func (s *CertStore) Read(name string) (*x509.Certificate, *rsa.PrivateKey, error) {
	crtBytes, err := afero.ReadFile(s.fs, s.CertFile(name))
	if err != nil {
		return nil, nil, errors.Wrapf(err, "failed to read certificate `%s`", s.CertFile(name))
	}
	crt, err := cert.ParseCertsPEM(crtBytes)
	if err != nil {
		return nil, nil, errors.Wrapf(err, "failed to parse certificate `%s`", s.CertFile(name))
	}

	keyBytes, err := afero.ReadFile(s.fs, s.KeyFile(name))
	if err != nil {
		return nil, nil, errors.Wrapf(err, "failed to read private key `%s`", s.KeyFile(name))
	}
	key, err := cert.ParsePrivateKeyPEM(keyBytes)
	if err != nil {
		return nil, nil, errors.Wrapf(err, "failed to parse private key `%s`", s.KeyFile(name))
	}
	return crt[0], key.(*rsa.PrivateKey), nil
}

func (s *CertStore) ReadBytes(name string) ([]byte, []byte, error) {
	crtBytes, err := afero.ReadFile(s.fs, s.CertFile(name))
	if err != nil {
		return nil, nil, errors.Wrapf(err, "failed to read certificate `%s`", s.CertFile(name))
	}

	keyBytes, err := afero.ReadFile(s.fs, s.KeyFile(name))
	if err != nil {
		return nil, nil, errors.Wrapf(err, "failed to read private key `%s`", s.KeyFile(name))
	}
	return crtBytes, keyBytes, nil
}

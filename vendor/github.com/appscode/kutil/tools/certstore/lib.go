package certstore

import (
	"crypto/rsa"
	"crypto/x509"
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

	caKey  *rsa.PrivateKey
	caCert *x509.Certificate
}

func NewCertStore(fs afero.Fs, dir string, organization ...string) (*CertStore, error) {
	err := fs.MkdirAll(dir, 0755)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to create dir `%s`", dir)
	}
	return &CertStore{fs: fs, dir: dir, organization: append([]string(nil), organization...)}, nil
}

func (cs *CertStore) InitCA() error {
	err := cs.LoadCA()
	if err == nil {
		return nil
	}
	return cs.NewCA()
}

func (cs *CertStore) LoadCA() error {
	if cs.PairExists("ca") {
		var err error
		cs.caCert, cs.caKey, err = cs.Read("ca")
		return err
	}
	return os.ErrNotExist
}

func (cs *CertStore) NewCA() error {
	var err error

	key, err := cert.NewPrivateKey()
	if err != nil {
		return errors.Wrap(err, "failed to generate private key")
	}
	cfg := cert.Config{
		CommonName:   "ca",
		Organization: cs.organization,
		AltNames: cert.AltNames{
			IPs: []net.IP{net.ParseIP("127.0.0.1")},
		},
	}
	crt, err := cert.NewSelfSignedCACert(cfg, key)
	if err != nil {
		return errors.Wrap(err, "failed to generate self-signed certificate")
	}
	err = cs.Write("ca", crt, key)
	if err != nil {
		return err
	}

	cs.caCert = crt
	cs.caKey = key
	return nil
}

func (cs *CertStore) Location() string {
	return cs.dir
}

func (cs *CertStore) CACert() []byte {
	return cert.EncodeCertPEM(cs.caCert)
}

func (cs *CertStore) CAKey() []byte {
	return cert.EncodePrivateKeyPEM(cs.caKey)
}

func (cs *CertStore) NewHostCertPair() ([]byte, []byte, error) {
	var sans cert.AltNames
	publicIPs, privateIPs, _ := netz.HostIPs()
	for _, ip := range publicIPs {
		sans.IPs = append(sans.IPs, net.ParseIP(ip))
	}
	for _, ip := range privateIPs {
		sans.IPs = append(sans.IPs, net.ParseIP(ip))
	}
	return cs.NewServerCertPair("127.0.0.1", sans)
}

func (cs *CertStore) NewServerCertPair(cn string, sans cert.AltNames) ([]byte, []byte, error) {
	cfg := cert.Config{
		CommonName:   cn,
		Organization: cs.organization,
		AltNames:     sans,
		Usages:       []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
	}
	key, err := cert.NewPrivateKey()
	if err != nil {
		return nil, nil, errors.Wrap(err, "failed to generate private key")
	}
	crt, err := cert.NewSignedCert(cfg, key, cs.caCert, cs.caKey)
	if err != nil {
		return nil, nil, errors.Wrap(err, "failed to generate server certificate")
	}
	return cert.EncodeCertPEM(crt), cert.EncodePrivateKeyPEM(key), nil
}

func (cs *CertStore) NewClientCertPair(cn string, organization ...string) ([]byte, []byte, error) {
	cfg := cert.Config{
		CommonName:   cn,
		Organization: organization,
		Usages:       []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
	}
	key, err := cert.NewPrivateKey()
	if err != nil {
		return nil, nil, errors.Wrap(err, "failed to generate private key")
	}
	crt, err := cert.NewSignedCert(cfg, key, cs.caCert, cs.caKey)
	if err != nil {
		return nil, nil, errors.Wrap(err, "failed to generate server certificate")
	}
	return cert.EncodeCertPEM(crt), cert.EncodePrivateKeyPEM(key), nil
}

func (cs *CertStore) IsExists(name string) bool {
	if _, err := cs.fs.Stat(cs.CertFile(name)); err == nil {
		return true
	}
	if _, err := cs.fs.Stat(cs.KeyFile(name)); err == nil {
		return true
	}
	return false
}

func (cs *CertStore) PairExists(name string) bool {
	if _, err := cs.fs.Stat(cs.CertFile(name)); err == nil {
		if _, err := cs.fs.Stat(cs.KeyFile(name)); err == nil {
			return true
		}
	}
	return false
}

func (cs *CertStore) CertFile(name string) string {
	return filepath.Join(cs.dir, strings.ToLower(name)+".crt")
}

func (cs *CertStore) KeyFile(name string) string {
	return filepath.Join(cs.dir, strings.ToLower(name)+".key")
}

func (cs *CertStore) Write(name string, crt *x509.Certificate, key *rsa.PrivateKey) error {
	if err := afero.WriteFile(cs.fs, cs.CertFile(name), cert.EncodeCertPEM(crt), 0644); err != nil {
		return errors.Wrapf(err, "failed to write `%s`", cs.CertFile(name))
	}
	if err := afero.WriteFile(cs.fs, cs.KeyFile(name), cert.EncodePrivateKeyPEM(key), 0600); err != nil {
		return errors.Wrapf(err, "failed to write `%s`", cs.KeyFile(name))
	}
	return nil
}

func (cs *CertStore) WriteBytes(name string, crt, key []byte) error {
	if err := afero.WriteFile(cs.fs, cs.CertFile(name), crt, 0644); err != nil {
		return errors.Wrapf(err, "failed to write `%s`", cs.CertFile(name))
	}
	if err := afero.WriteFile(cs.fs, cs.KeyFile(name), key, 0600); err != nil {
		return errors.Wrapf(err, "failed to write `%s`", cs.KeyFile(name))
	}
	return nil
}

func (cs *CertStore) Read(name string) (*x509.Certificate, *rsa.PrivateKey, error) {
	crtBytes, err := afero.ReadFile(cs.fs, cs.CertFile(name))
	if err != nil {
		return nil, nil, errors.Wrapf(err, "failed to read certificate `%s`", cs.CertFile(name))
	}
	crt, err := cert.ParseCertsPEM(crtBytes)
	if err != nil {
		return nil, nil, errors.Wrapf(err, "failed to parse certificate `%s`", cs.CertFile(name))
	}

	keyBytes, err := afero.ReadFile(cs.fs, cs.KeyFile(name))
	if err != nil {
		return nil, nil, errors.Wrapf(err, "failed to read private key `%s`", cs.KeyFile(name))
	}
	key, err := cert.ParsePrivateKeyPEM(keyBytes)
	if err != nil {
		return nil, nil, errors.Wrapf(err, "failed to parse private key `%s`", cs.KeyFile(name))
	}
	return crt[0], key.(*rsa.PrivateKey), nil
}

func (cs *CertStore) ReadBytes(name string) ([]byte, []byte, error) {
	crtBytes, err := afero.ReadFile(cs.fs, cs.CertFile(name))
	if err != nil {
		return nil, nil, errors.Wrapf(err, "failed to read certificate `%s`", cs.CertFile(name))
	}

	keyBytes, err := afero.ReadFile(cs.fs, cs.KeyFile(name))
	if err != nil {
		return nil, nil, errors.Wrapf(err, "failed to read private key `%s`", cs.KeyFile(name))
	}
	return crtBytes, keyBytes, nil
}

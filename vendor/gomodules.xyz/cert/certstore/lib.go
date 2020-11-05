package certstore

import (
	"context"
	"crypto/rsa"
	"crypto/x509"
	"fmt"
	"net"
	"os"
	"path"
	"strings"

	"github.com/pkg/errors"
	"gomodules.xyz/blobfs"
	"gomodules.xyz/cert"
	netz "gomodules.xyz/x/net"
)

func SANsForNames(s string, names ...string) cert.AltNames {
	return cert.AltNames{
		DNSNames: append([]string{s}, names...),
	}
}

func SANsForIPs(s string, ips ...string) cert.AltNames {
	addrs := make([]net.IP, 0, len(ips))
	for _, ip := range ips {
		if v := net.ParseIP(ip); v != nil {
			addrs = append(addrs, v)
		}
	}
	return cert.AltNames{
		DNSNames: []string{s},
		IPs:      addrs,
	}
}

type CertStore struct {
	fs           *blobfs.BlobFS
	dir          string
	organization []string
	prefix       string
	ca           string
	caKey        *rsa.PrivateKey
	caCert       *x509.Certificate
}

func New(fs *blobfs.BlobFS, dir string, organization ...string) (*CertStore, error) {
	return &CertStore{fs: fs, dir: dir, ca: "ca", organization: append([]string(nil), organization...)}, nil
}

func (s *CertStore) InitCA(prefix ...string) error {
	err := s.LoadCA(prefix...)
	if err == nil {
		return nil
	}
	return s.NewCA(prefix...)
}

func (s *CertStore) SetCA(crtBytes, keyBytes []byte) error {
	crt, err := cert.ParseCertsPEM(crtBytes)
	if err != nil {
		return errors.Wrap(err, "failed to parse ca certificate")
	}

	key, err := cert.ParsePrivateKeyPEM(keyBytes)
	if err != nil {
		return errors.Wrap(err, "failed to parse ca private key")
	}

	s.caCert = crt[0]
	s.caKey = key.(*rsa.PrivateKey)
	return s.Write(s.ca, s.caCert, s.caKey)
}

func (s *CertStore) LoadCA(prefix ...string) error {
	if err := s.prep(prefix...); err != nil {
		return err
	}

	if s.PairExists(s.ca, prefix...) {
		var err error
		s.caCert, s.caKey, err = s.Read(s.ca)
		return err
	}

	// only ca key found, extract ca cert from it.
	if found, err := s.fs.Exists(context.TODO(), s.KeyFile(s.ca)); err == nil && found {
		keyBytes, err := s.fs.ReadFile(context.TODO(), s.KeyFile(s.ca))
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
			DNSNames: []string{s.ca},
			IPs:      []net.IP{net.ParseIP("127.0.0.1")},
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

func (s *CertStore) CACert() *x509.Certificate {
	return s.caCert
}

func (s *CertStore) CACertBytes() []byte {
	return cert.EncodeCertPEM(s.caCert)
}

func (s *CertStore) CAKey() *rsa.PrivateKey {
	return s.caKey
}

func (s *CertStore) CAKeyBytes() []byte {
	return cert.EncodePrivateKeyPEM(s.caKey)
}

func (s *CertStore) NewHostCertPair() (*x509.Certificate, *rsa.PrivateKey, error) {
	sans := cert.AltNames{
		IPs: []net.IP{net.ParseIP("127.0.0.1")},
	}
	publicIPs, privateIPs, _ := netz.HostIPs()
	for _, ip := range publicIPs {
		sans.IPs = append(sans.IPs, net.ParseIP(ip))
	}
	for _, ip := range privateIPs {
		sans.IPs = append(sans.IPs, net.ParseIP(ip))
	}
	return s.NewServerCertPair(sans)
}

func (s *CertStore) NewHostCertPairBytes() ([]byte, []byte, error) {
	sans := cert.AltNames{
		IPs: []net.IP{net.ParseIP("127.0.0.1")},
	}
	publicIPs, privateIPs, _ := netz.HostIPs()
	for _, ip := range publicIPs {
		sans.IPs = append(sans.IPs, net.ParseIP(ip))
	}
	for _, ip := range privateIPs {
		sans.IPs = append(sans.IPs, net.ParseIP(ip))
	}
	return s.NewServerCertPairBytes(sans)
}

func (s *CertStore) NewServerCertPair(sans cert.AltNames) (*x509.Certificate, *rsa.PrivateKey, error) {
	cfg := cert.Config{
		CommonName:   getCN(sans),
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
	return crt, key, nil
}

func (s *CertStore) NewServerCertPairBytes(sans cert.AltNames) ([]byte, []byte, error) {
	crt, key, err := s.NewServerCertPair(sans)
	if err != nil {
		return nil, nil, err
	}
	return cert.EncodeCertPEM(crt), cert.EncodePrivateKeyPEM(key), nil
}

// NewPeerCertPair is used to create cert pair that can serve as both server and client.
// This is used to issue peer certificates for etcd.
func (s *CertStore) NewPeerCertPair(sans cert.AltNames) (*x509.Certificate, *rsa.PrivateKey, error) {
	cfg := cert.Config{
		CommonName:   getCN(sans),
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
	return crt, key, nil
}

func (s *CertStore) NewPeerCertPairBytes(sans cert.AltNames) ([]byte, []byte, error) {
	crt, key, err := s.NewPeerCertPair(sans)
	if err != nil {
		return nil, nil, err
	}
	return cert.EncodeCertPEM(crt), cert.EncodePrivateKeyPEM(key), nil
}

func (s *CertStore) NewClientCertPair(sans cert.AltNames, organization ...string) (*x509.Certificate, *rsa.PrivateKey, error) {
	cfg := cert.Config{
		CommonName:   getCN(sans),
		Organization: organization,
		AltNames:     sans,
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
	return crt, key, nil
}

func (s *CertStore) NewClientCertPairBytes(sans cert.AltNames, organization ...string) ([]byte, []byte, error) {
	crt, key, err := s.NewClientCertPair(sans, organization...)
	if err != nil {
		return nil, nil, err
	}
	return cert.EncodeCertPEM(crt), cert.EncodePrivateKeyPEM(key), nil
}

func (s *CertStore) IsExists(name string, prefix ...string) bool {
	if err := s.prep(prefix...); err != nil {
		panic(err)
	}

	if found, err := s.fs.Exists(context.TODO(), s.CertFile(name)); err == nil && found {
		return true
	}
	if found, err := s.fs.Exists(context.TODO(), s.KeyFile(name)); err == nil && found {
		return true
	}
	return false
}

func (s *CertStore) PairExists(name string, prefix ...string) bool {
	if err := s.prep(prefix...); err != nil {
		panic(err)
	}

	if f1, err := s.fs.Exists(context.TODO(), s.CertFile(name)); err == nil && f1 {
		if f2, err := s.fs.Exists(context.TODO(), s.KeyFile(name)); err == nil && f2 {
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
	return path.Join(s.dir, filename)
}

func (s *CertStore) KeyFile(name string) string {
	filename := strings.ToLower(name) + ".key"
	if s.prefix != "" {
		filename = s.prefix + filename
	}
	return path.Join(s.dir, filename)
}

func (s *CertStore) Write(name string, crt *x509.Certificate, key *rsa.PrivateKey) error {
	if err := s.fs.WriteFile(context.TODO(), s.CertFile(name), cert.EncodeCertPEM(crt)); err != nil {
		return errors.Wrapf(err, "failed to write `%s`", s.CertFile(name))
	}
	if err := s.fs.WriteFile(context.TODO(), s.KeyFile(name), cert.EncodePrivateKeyPEM(key)); err != nil {
		return errors.Wrapf(err, "failed to write `%s`", s.KeyFile(name))
	}
	return nil
}

func (s *CertStore) WriteBytes(name string, crt, key []byte) error {
	if err := s.fs.WriteFile(context.TODO(), s.CertFile(name), crt); err != nil {
		return errors.Wrapf(err, "failed to write `%s`", s.CertFile(name))
	}
	if err := s.fs.WriteFile(context.TODO(), s.KeyFile(name), key); err != nil {
		return errors.Wrapf(err, "failed to write `%s`", s.KeyFile(name))
	}
	return nil
}

func (s *CertStore) Read(name string) (*x509.Certificate, *rsa.PrivateKey, error) {
	crtBytes, err := s.fs.ReadFile(context.TODO(), s.CertFile(name))
	if err != nil {
		return nil, nil, errors.Wrapf(err, "failed to read certificate `%s`", s.CertFile(name))
	}
	crt, err := cert.ParseCertsPEM(crtBytes)
	if err != nil {
		return nil, nil, errors.Wrapf(err, "failed to parse certificate `%s`", s.CertFile(name))
	}

	keyBytes, err := s.fs.ReadFile(context.TODO(), s.KeyFile(name))
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
	crtBytes, err := s.fs.ReadFile(context.TODO(), s.CertFile(name))
	if err != nil {
		return nil, nil, errors.Wrapf(err, "failed to read certificate `%s`", s.CertFile(name))
	}

	keyBytes, err := s.fs.ReadFile(context.TODO(), s.KeyFile(name))
	if err != nil {
		return nil, nil, errors.Wrapf(err, "failed to read private key `%s`", s.KeyFile(name))
	}
	return crtBytes, keyBytes, nil
}

// RFC 5280
// When the subjectAltName extension contains a domain name system
// label, the domain name MUST be stored in the dNSName (an IA5String).
// The name MUST be in the "preferred name syntax", as specified by
// Section 3.5 of RFC1034 and as modified by Section 2.1 of
// RFC1123. Note that while uppercase and lowercase letters are
// allowed in domain names, no significance is attached to the case.
// ref: https://security.stackexchange.com/a/150776/27304
func merge(cn string, sans []string) []string {
	var found bool
	for _, name := range sans {
		if strings.EqualFold(name, cn) {
			found = true
			break
		}
	}
	if !found {
		return append(sans, cn)
	}
	return sans
}

func getCN(sans cert.AltNames) string {
	if len(sans.DNSNames) > 0 {
		return sans.DNSNames[0]
	}
	if len(sans.IPs) > 0 {
		return sans.IPs[0].String()
	}
	return ""
}

package cert

import (
	"bytes"
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
)

func ToX509CombinedKeyPair(cert *tls.Certificate) ([]byte, error) {
	var buf bytes.Buffer

	for _, c := range cert.Certificate {
		if err := pem.Encode(&buf, &pem.Block{
			Type:    "CERTIFICATE",
			Headers: nil,
			Bytes:   c,
		}); err != nil {
			return nil, err
		}
	}

	// https://golang.org/src/crypto/tls/tls.go?s=7880:7947#L370
	key, err := x509.MarshalPKCS8PrivateKey(cert.PrivateKey)
	if err != nil {
		return nil, err
	}
	err = pem.Encode(&buf, &pem.Block{
		Type:    "PRIVATE KEY",
		Headers: nil,
		Bytes:   key,
	})
	if err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

func ToX509KeyPair(cert *tls.Certificate) (certPEMBlock, keyPEMBlock []byte, err error) {
	var buf bytes.Buffer

	for _, c := range cert.Certificate {
		if err := pem.Encode(&buf, &pem.Block{
			Type:    "CERTIFICATE",
			Headers: nil,
			Bytes:   c,
		}); err != nil {
			return nil, nil, err
		}
	}
	certPEMBlock = buf.Bytes()
	buf.Reset()

	// https://golang.org/src/crypto/tls/tls.go?s=7880:7947#L370
	key, err := x509.MarshalPKCS8PrivateKey(cert.PrivateKey)
	if err != nil {
		return nil, nil, err
	}
	err = pem.Encode(&buf, &pem.Block{
		Type:    "PRIVATE KEY",
		Headers: nil,
		Bytes:   key,
	})
	if err != nil {
		return nil, nil, err
	}
	keyPEMBlock = buf.Bytes()

	return
}

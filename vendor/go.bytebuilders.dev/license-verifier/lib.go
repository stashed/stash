/*
Copyright AppsCode Inc.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package verifier

import (
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"strings"

	"github.com/pkg/errors"
	"gomodules.xyz/sets"
)

type Options struct {
	ClusterUID  string
	ProductName string
	CACert      []byte
	License     []byte
}

func VerifyLicense(opts *Options) error {
	if opts == nil {
		return fmt.Errorf("missing license")
	}
	cert, err := parseCertificate(opts.License)
	if err != nil {
		return err
	}

	// First, create the set of root certificates. For this example we only
	// have one. It's also possible to omit this in order to use the
	// default root set of the current operating system.
	roots := x509.NewCertPool()
	ok := roots.AppendCertsFromPEM(opts.CACert)
	if !ok {
		return errors.New("failed to parse root certificate")
	}

	crtopts := x509.VerifyOptions{
		DNSName: opts.ClusterUID,
		Roots:   roots,
		KeyUsages: []x509.ExtKeyUsage{
			x509.ExtKeyUsageClientAuth,
		},
	}

	// wildcard certificate
	if strings.HasPrefix(cert.Subject.CommonName, "*.") {
		caCert, err := parseCertificate(opts.CACert)
		if err != nil {
			return err
		}
		if len(caCert.Subject.Organization) > 0 {
			crtopts.DNSName = "*." + caCert.Subject.Organization[0]
		}
	}

	if _, err := cert.Verify(crtopts); err != nil {
		return errors.Wrap(err, "failed to verify certificate")
	}
	if !sets.NewString(cert.Subject.Organization...).Has(opts.ProductName) {
		return fmt.Errorf("license was not issued for %s", opts.ProductName)
	}
	return nil
}

func parseCertificate(data []byte) (*x509.Certificate, error) {
	block, _ := pem.Decode(data)
	if block == nil {
		// This probably is a JWT token, should be check for that when ready
		return nil, errors.New("failed to parse certificate PEM")
	}
	return x509.ParseCertificate(block.Bytes)
}

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

package info

import (
	"bytes"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"io"
	"net/http"
	"net/url"
	"path"
	"strconv"
	"strings"
	"unicode"

	"go.bytebuilders.dev/license-verifier/apis/licenses"

	"github.com/PuerkitoBio/purell"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/sets"
)

var (
	EnforceLicense string
	LicenseCA      string

	ProductOwnerName string
	ProductOwnerUID  string

	ProductName string // This has been renamed to Features
	ProductUID  string

	prodDomain = "byte.builders"
	qaDomain   = "appscode.ninja"

	registrationAPIPath  = "api/v1/register"
	LicenseIssuerAPIPath = "api/v1/license/issue"
)

func Features() []string {
	return ParseFeatures(ProductName)
}

func ParseFeatures(features string) []string {
	out := strings.FieldsFunc(features, func(r rune) bool {
		return unicode.IsSpace(r) || r == ',' || r == ';'
	})
	return sets.NewString(out...).List()
}

func SkipLicenseVerification() bool {
	v, _ := strconv.ParseBool(EnforceLicense)
	return !v
}

func MustRegistrationAPIEndpoint() string {
	r, err := RegistrationAPIEndpoint()
	if err != nil {
		panic(err)
	}
	return r
}

func RegistrationAPIEndpoint(override ...string) (string, error) {
	u, err := APIServerAddress(override...)
	if err != nil {
		return "", err
	}
	u.Path = path.Join(u.Path, registrationAPIPath)
	return u.String(), nil
}

func MustLicenseIssuerAPIEndpoint() string {
	r, err := LicenseIssuerAPIEndpoint()
	if err != nil {
		panic(err)
	}
	return r
}

func LicenseIssuerAPIEndpoint(override ...string) (string, error) {
	u, err := APIServerAddress(override...)
	if err != nil {
		return "", err
	}
	u.Path = path.Join(u.Path, LicenseIssuerAPIPath)
	return u.String(), nil
}

func MustAPIServerAddress() *url.URL {
	u, err := APIServerAddress()
	if err != nil {
		panic(err)
	}
	return u
}

func APIServerAddress(override ...string) (*url.URL, error) {
	if len(override) > 0 && override[0] != "" {
		nu, err := purell.NormalizeURLString(override[0],
			purell.FlagsUsuallySafeGreedy|purell.FlagRemoveDuplicateSlashes)
		if err != nil {
			return nil, err
		}
		return url.Parse(nu)
	}

	if SkipLicenseVerification() {
		return url.Parse("https://api." + qaDomain)
	}
	return url.Parse("https://api." + prodDomain)
}

func HostedEndpoint(u string) (bool, error) {
	nu, err := purell.NormalizeURLString(u,
		purell.FlagsUsuallySafeGreedy|purell.FlagRemoveDuplicateSlashes)
	if err != nil {
		return false, err
	}
	u2, err := url.Parse(nu)
	if err != nil {
		return false, err
	}
	return HostedDomain(u2.Hostname()), nil
}

func HostedDomain(d string) bool {
	return d == prodDomain ||
		d == qaDomain ||
		strings.HasSuffix(d, "."+prodDomain) ||
		strings.HasSuffix(d, "."+qaDomain)
}

func LoadLicenseCA() ([]byte, error) {
	if LicenseCA != "" {
		return []byte(LicenseCA), nil
	}

	resp, err := http.Get("https://licenses.appscode.com/certificates/ca.crt")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var buf bytes.Buffer
	_, err = io.Copy(&buf, resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		return nil, apierrors.NewGenericServerResponse(
			resp.StatusCode,
			http.MethodPost,
			schema.GroupResource{Group: licenses.GroupName, Resource: "License"},
			"LicenseCA",
			buf.String(),
			0,
			false,
		)
	}
	return buf.Bytes(), nil
}

func ParseCertificate(data []byte) (*x509.Certificate, error) {
	block, _ := pem.Decode(data)
	if block == nil {
		// This probably is a JWT token, should be check for that when ready
		return nil, errors.New("failed to parse certificate PEM")
	}
	return x509.ParseCertificate(block.Bytes)
}

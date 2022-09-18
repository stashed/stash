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

	prodAddress          = "https://byte.builders"
	qaAddress            = "https://appscode.ninja"
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

func RegistrationAPIEndpoint() string {
	u := APIServerAddress()
	u.Path = path.Join(u.Path, registrationAPIPath)
	return u.String()
}

func LicenseIssuerAPIEndpoint() string {
	u := APIServerAddress()
	u.Path = path.Join(u.Path, LicenseIssuerAPIPath)
	return u.String()
}

func APIServerAddress() *url.URL {
	if SkipLicenseVerification() {
		u, _ := url.Parse(qaAddress)
		return u
	}
	u, _ := url.Parse(prodAddress)
	return u
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

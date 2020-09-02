/*
Copyright AppsCode Inc. and Contributors

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

package v1

import (
	"reflect"

	"github.com/imdario/mergo"
	core "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/sets"
)

type TLSConfig struct {
	// IssuerRef is a reference to a Certificate Issuer.
	// +optional
	IssuerRef *core.TypedLocalObjectReference `json:"issuerRef,omitempty" protobuf:"bytes,1,opt,name=issuerRef"`

	// Certificate provides server and/or client certificate options used by application pods.
	// These options are passed to a cert-manager Certificate object.
	// xref: https://github.com/jetstack/cert-manager/blob/v0.16.0/pkg/apis/certmanager/v1beta1/types_certificate.go#L82-L162
	// +optional
	Certificates []CertificateSpec `json:"certificates,omitempty" protobuf:"bytes,2,rep,name=certificates"`
}

type CertificateSpec struct {
	// Alias represents the identifier of the certificate.
	Alias string `json:"alias" protobuf:"bytes,1,opt,name=alias"`

	// Specifies the k8s secret name that holds the certificates.
	// Default to <resource-name>-<cert-alias>-cert.
	// +optional
	SecretName string `json:"secretName,omitempty" protobuf:"bytes,2,opt,name=secretName"`

	// KeyEncoding is the private key cryptography standards (PKCS)
	// for this certificate's private key to be encoded in.
	// If provided, allowed values are "pkcs1" and "pkcs8".
	// If KeyEncoding is not specified, then PKCS#1 will be used by default.
	// +optional
	KeyEncoding string `json:"keyEncoding,omitempty" protobuf:"bytes,3,opt,name=keyEncoding"`

	// Full X509 name specification (https://golang.org/pkg/crypto/x509/pkix/#Name).
	// +optional
	Subject *X509Subject `json:"subject,omitempty" protobuf:"bytes,4,opt,name=subject"`

	// Certificate default Duration
	// +optional
	Duration *metav1.Duration `json:"duration,omitempty" protobuf:"bytes,5,opt,name=duration"`

	// Certificate renew before expiration duration
	// +optional
	RenewBefore *metav1.Duration `json:"renewBefore,omitempty" protobuf:"bytes,6,opt,name=renewBefore"`

	// DNSNames is a list of subject alt names to be used on the Certificate.
	// +optional
	DNSNames []string `json:"dnsNames,omitempty" protobuf:"bytes,7,rep,name=dnsNames"`

	// IPAddresses is a list of IP addresses to be used on the Certificate
	// +optional
	IPAddresses []string `json:"ipAddresses,omitempty" protobuf:"bytes,8,rep,name=ipAddresses"`

	// URISANs is a list of URI Subject Alternative Names to be set on this
	// Certificate.
	// +optional
	URISANs []string `json:"uriSANs,omitempty" protobuf:"bytes,9,rep,name=uriSANs"`

	// EmailSANs is a list of email subjectAltNames to be set on the Certificate.
	// +optional
	EmailSANs []string `json:"emailSANs,omitempty" protobuf:"bytes,10,rep,name=emailSANs"`
}

// X509Subject Full X509 name specification
type X509Subject struct {
	// Organizations to be used on the Certificate.
	// +optional
	Organizations []string `json:"organizations,omitempty" protobuf:"bytes,1,rep,name=organizations"`
	// Countries to be used on the CertificateSpec.
	// +optional
	Countries []string `json:"countries,omitempty" protobuf:"bytes,2,rep,name=countries"`
	// Organizational Units to be used on the CertificateSpec.
	// +optional
	OrganizationalUnits []string `json:"organizationalUnits,omitempty" protobuf:"bytes,3,rep,name=organizationalUnits"`
	// Cities to be used on the CertificateSpec.
	// +optional
	Localities []string `json:"localities,omitempty" protobuf:"bytes,4,rep,name=localities"`
	// State/Provinces to be used on the CertificateSpec.
	// +optional
	Provinces []string `json:"provinces,omitempty" protobuf:"bytes,5,rep,name=provinces"`
	// Street addresses to be used on the CertificateSpec.
	// +optional
	StreetAddresses []string `json:"streetAddresses,omitempty" protobuf:"bytes,6,rep,name=streetAddresses"`
	// Postal codes to be used on the CertificateSpec.
	// +optional
	PostalCodes []string `json:"postalCodes,omitempty" protobuf:"bytes,7,rep,name=postalCodes"`
	// Serial number to be used on the CertificateSpec.
	// +optional
	SerialNumber string `json:"serialNumber,omitempty" protobuf:"bytes,8,opt,name=serialNumber"`
}

// HasCertificate returns "true" if the desired certificate provided in "aliaS" is present in the certificate list.
// Otherwise, it returns "false".
func HasCertificate(certificates []CertificateSpec, alias string) bool {
	for i := range certificates {
		if certificates[i].Alias == alias {
			return true
		}
	}
	return false
}

// GetCertificate returns a pointer to the desired certificate referred by "aliaS". Otherwise, it returns nil.
func GetCertificate(certificates []CertificateSpec, alias string) (int, *CertificateSpec) {
	for i := range certificates {
		c := certificates[i]
		if c.Alias == alias {
			return i, &c
		}
	}
	return -1, nil
}

// SetCertificate add/update the desired certificate to the certificate list.
func SetCertificate(certificates []CertificateSpec, newCertificate CertificateSpec) []CertificateSpec {
	idx, _ := GetCertificate(certificates, newCertificate.Alias)
	if idx != -1 {
		certificates[idx] = newCertificate
	} else {
		certificates = append(certificates, newCertificate)
	}
	return certificates
}

// GetCertificateSecretName returns the name of secret for a certificate alias.
func GetCertificateSecretName(certificates []CertificateSpec, alias string) (string, bool) {
	idx, cert := GetCertificate(certificates, alias)
	if idx == -1 {
		return "", false
	}
	return cert.SecretName, cert.SecretName != ""
}

// SetMissingSpecForCertificate sets the missing spec fields for a certificate.
// If the certificate does not exist, it will add a new certificate with the desired spec.
func SetMissingSpecForCertificate(certificates []CertificateSpec, spec CertificateSpec) []CertificateSpec {
	idx, _ := GetCertificate(certificates, spec.Alias)
	if idx != -1 {
		err := mergo.Merge(&certificates[idx], spec, mergo.WithTransformers(stringSetMerger{}))
		if err != nil {
			panic(err)
		}
	} else {
		certificates = append(certificates, spec)
	}
	return certificates
}

// SetSpecForCertificate sets the spec for a certificate.
// If the certificate does not exist, it will add a new certificate with the desired spec.
// Otherwise, the spec will be overwritten.
func SetSpecForCertificate(certificates []CertificateSpec, spec CertificateSpec) []CertificateSpec {
	idx, _ := GetCertificate(certificates, spec.Alias)
	if idx != -1 {
		certificates[idx] = spec
	} else {
		certificates = append(certificates, spec)
	}
	return certificates
}

// SetMissingSecretNameForCertificate sets the missing secret name for a certificate.
// If the certificate does not exist, it will add a new certificate with the desired secret name.
func SetMissingSecretNameForCertificate(certificates []CertificateSpec, alias, secretName string) []CertificateSpec {
	idx, _ := GetCertificate(certificates, alias)
	if idx != -1 {
		if certificates[idx].SecretName == "" {
			certificates[idx].SecretName = secretName
		}
	} else {
		certificates = append(certificates, CertificateSpec{
			Alias:      alias,
			SecretName: secretName,
		})
	}
	return certificates
}

// SetSecretNameForCertificate sets the secret name for a certificate.
// If the certificate does not exist, it will add a new certificate with the desired secret name.
// Otherwise, the secret name will be overwritten.
func SetSecretNameForCertificate(certificates []CertificateSpec, alias, secretName string) []CertificateSpec {
	idx, _ := GetCertificate(certificates, alias)
	if idx != -1 {
		certificates[idx].SecretName = secretName
	} else {
		certificates = append(certificates, CertificateSpec{
			Alias:      alias,
			SecretName: secretName,
		})
	}
	return certificates
}

// RemoveCertificate remove a certificate from the certificate list referred by "aliaS" parameter.
func RemoveCertificate(certificates []CertificateSpec, alias string) []CertificateSpec {
	idx, _ := GetCertificate(certificates, alias)
	if idx == -1 {
		// The desired certificate is not present in the certificate list. So, nothing to do.
		return certificates
	}
	return append(certificates[:idx], certificates[idx+1:]...)
}

type stringSetMerger struct {
}

func (t stringSetMerger) Transformer(typ reflect.Type) func(dst, src reflect.Value) error {
	if typ == reflect.TypeOf([]string{}) {
		return func(dst, src reflect.Value) error {
			if dst.CanSet() {
				if dst.Len() <= 1 && src.Len() == 0 {
					return nil
				}
				if dst.Len() == 0 && src.Len() == 1 {
					dst.Set(src)
					return nil
				}

				out := sets.NewString()
				for i := 0; i < dst.Len(); i++ {
					out.Insert(dst.Index(i).String())
				}
				for i := 0; i < src.Len(); i++ {
					out.Insert(src.Index(i).String())
				}
				dst.Set(reflect.ValueOf(out.List()))
			}
			return nil
		}
	}
	return nil
}

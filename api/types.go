package kube

import (
	"time"

	"k8s.io/kubernetes/pkg/api"
	"k8s.io/kubernetes/pkg/api/unversioned"
)

type Certificate struct {
	unversioned.TypeMeta `json:",inline,omitempty"`
	api.ObjectMeta       `json:"metadata,omitempty"`
	Spec                 CertificateSpec   `json:"spec,omitempty"`
	Status               CertificateStatus `json:"status,omitempty"`
}

type CertificateSpec struct {
	// Tries to obtain a single certificate using all domains passed into Domains.
	// The first domain in domains is used for the CommonName field of the certificate, all other
	// domains are added using the Subject Alternate Names extension.
	Domains []string `json:"domains,omitempty"`

	// DNS Provider.
	Provider string `json:"provider,omitempty"`
	Email    string `json:"email,omitempty"`

	// This is the ingress Reference that will be used if provider is http
	HTTPProviderIngressReference api.ObjectReference `json:"httpProviderIngressReference,omitempty"`

	// ProviderCredentialSecretName is used to create the acme client, that will do
	// needed processing in DNS.
	ProviderCredentialSecretName string `json:"providerCredentialSecretName,omitempty"`

	// Secret contains ACMEUser information. If empty tries to find an Secret via domains
	// if not found create an ACMEUser and stores as a secret.
	ACMEUserSecretName string `json:"acmeUserSecretName"`

	// ACME server that will be used to obtain this certificate.
	ACMEServerURL string `json:"acmeStagingURL"`
}

type CertificateStatus struct {
	CertificateObtained bool                   `json:"certificateObtained"`
	Message             string                 `json:"message"`
	Created             time.Time              `json:"created,omitempty"`
	ACMEUserSecretName  string                 `json:"acmeUserSecretName,omitempty"`
	Details             ACMECertificateDetails `json:"details,omitempty"`
}

type ACMECertificateDetails struct {
	Domain        string `json:"domain"`
	CertURL       string `json:"certUrl"`
	CertStableURL string `json:"certStableUrl"`
	AccountRef    string `json:"accountRef,omitempty"`
}

type CertificateList struct {
	unversioned.TypeMeta `json:",inline"`
	unversioned.ListMeta `json:"metadata,omitempty"`
	Items                []Certificate `json:"items,omitempty"`
}

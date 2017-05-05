package api

import (
	"k8s.io/kubernetes/pkg/api"
	"k8s.io/kubernetes/pkg/api/unversioned"
	"k8s.io/kubernetes/pkg/util/intstr"
)

const (
	EngressKey  = "ingress.appscode.com"
	EngressKind = EngressKey + "/kind"
)

// ExtendedIngress types for appscode.
type Ingress struct {
	unversioned.TypeMeta `json:",inline"`
	// Standard object's metadata.
	// More info: http://releases.k8s.io/release-1.2/docs/devel/api-conventions.md#metadata
	api.ObjectMeta `json:"metadata,omitempty"`

	// Spec is the desired state of the ExtendedIngress.
	// More info: http://releases.k8s.io/release-1.2/docs/devel/api-conventions.md#spec-and-status
	Spec ExtendedIngressSpec `json:"spec,omitempty"`

	// Status is the current state of the ExtendedIngress.
	// More info: http://releases.k8s.io/release-1.2/docs/devel/api-conventions.md#spec-and-status
	Status ExtendedIngressStatus `json:"status,omitempty"`
}

// ExtendedIngressList is a collection of ExtendedIngress.
type IngressList struct {
	unversioned.TypeMeta `json:",inline"`
	// Standard object's metadata.
	// More info: http://releases.k8s.io/release-1.2/docs/devel/api-conventions.md#metadata
	unversioned.ListMeta `json:"metadata,omitempty"`

	// Items is the list of ExtendedIngress.
	Items []Ingress `json:"items"`
}

// ExtendedIngressSpec describes the ExtendedIngress the user wishes to exist.
type ExtendedIngressSpec struct {
	// A default backend capable of servicing requests that don't match any
	// rule. At least one of 'backend' or 'rules' must be specified. This field
	// is optional to allow the loadbalancer controller or defaulting logic to
	// specify a global default.
	Backend *ExtendedIngressBackend `json:"backend,omitempty"`

	// TLS is the TLS configuration. Currently the ExtendedIngress only supports a single TLS
	// port, 443, and assumes TLS termination. If multiple members of this
	// list specify different hosts, they will be multiplexed on the same
	// port according to the hostname specified through the SNI TLS extension.
	TLS []ExtendedIngressTLS `json:"tls,omitempty"`

	// A list of host rules used to configure the ExtendedIngress. If unspecified, or
	// no rule matches, all traffic is sent to the default backend.
	Rules []ExtendedIngressRule `json:"rules,omitempty"`
	// TODO: Add the ability to specify load-balancer IP through claims
}

// ExtendedIngressTLS describes the transport layer security associated with an ExtendedIngress.
type ExtendedIngressTLS struct {
	// Hosts are a list of hosts included in the TLS certificate. The values in
	// this list must match the name/s used in the tlsSecret. Defaults to the
	// wildcard host setting for the loadbalancer controller fulfilling this
	// ExtendedIngress, if left unspecified.
	Hosts []string `json:"hosts,omitempty"`
	// SecretName is the name of the secret used to terminate SSL traffic on 443.
	// Field is left optional to allow SSL routing based on SNI hostname alone.
	// If the SNI host in a listener conflicts with the "Host" header field used
	// by an ExtendedIngressRule, the SNI host is used for termination and value of the
	// Host header is used for routing.
	SecretName string `json:"secretName,omitempty"`
}

// ExtendedIngressStatus describe the current state of the ExtendedIngress.
type ExtendedIngressStatus struct {
	// LoadBalancer contains the current status of the load-balancer.
	LoadBalancer api.LoadBalancerStatus `json:"loadBalancer,omitempty"`
}

// ExtendedIngressRule represents the rules mapping the paths under a specified host to
// the related backend services. Incoming requests are first evaluated for a host
// match, then routed to the backend associated with the matching ExtendedIngressRuleValue.
type ExtendedIngressRule struct {
	// Host is the fully qualified domain name of a network host, as defined
	// by RFC 3986. Note the following deviations from the "host" part of the
	// URI as defined in the RFC:
	// 1. IPs are not allowed. Currently an ExtendedIngressRuleValue can only apply to the
	//	  IP in the Spec of the parent ExtendedIngress.
	// 2. The `:` delimiter is not respected because ports are not allowed.
	//	  Currently the port of an ExtendedIngress is implicitly :80 for http and
	//	  :443 for https.
	// Both these may change in the future.
	// Incoming requests are matched against the host before the ExtendedIngressRuleValue.
	// If the host is unspecified, the ExtendedIngress routes all traffic based on the
	// specified ExtendedIngressRuleValue.
	Host string `json:"host,omitempty"`
	// ExtendedIngressRuleValue represents a rule to route requests for this ExtendedIngressRule.
	// If unspecified, the rule defaults to a http catch-all. Whether that sends
	// just traffic matching the host to the default backend or all traffic to the
	// default backend, is left to the controller fulfilling the ExtendedIngress. Http is
	// currently the only supported ExtendedIngressRuleValue.
	ExtendedIngressRuleValue `json:",inline,omitempty"`
}

// ExtendedIngressRuleValue represents a rule to apply against incoming requests. If the
// rule is satisfied, the request is routed to the specified backend. Currently
// mixing different types of rules in a single ExtendedIngress is disallowed, so exactly
// one of the following must be set.
type ExtendedIngressRuleValue struct {
	//TODO:
	// 1. Consider renaming this resource and the associated rules so they
	// aren't tied to ExtendedIngress. They can be used to route intra-cluster traffic.
	// 2. Consider adding fields for ExtendedIngress-type specific global options
	// usable by a loadbalancer, like http keep-alive.

	HTTP *HTTPExtendedIngressRuleValue `json:"http,omitempty"`

	TCP []TCPExtendedIngressRuleValue `json:"tcp,omitempty"`
}

// HTTPExtendedIngressRuleValue is a list of http selectors pointing to backends.
// In the example: http://<host>/<path>?<searchpart> -> backend where
// where parts of the url correspond to RFC 3986, this resource will be used
// to match against everything after the last '/' and before the first '?'
// or '#'.
type HTTPExtendedIngressRuleValue struct {
	// A collection of paths that map requests to backends.
	Paths []HTTPExtendedIngressPath `json:"paths"`
	// TODO: Consider adding fields for ExtendedIngress-type specific global
	// options usable by a loadbalancer, like http keep-alive.
}

type TCPExtendedIngressRuleValue struct {
	// port to listen tcp connections.
	Port intstr.IntOrString `json:"port,omitempty"`

	// SSl cert used to terminate ssl in this port
	SecretName string `json:"secretName,omitempty"`

	// Backend to forward the requests.
	Backend IngressBackend `json:"backend,omitempty"`

	// Application-Layer Protocol Negotiation (ALPN) is a Transport Layer Security (TLS)
	// extension for application layer protocol negotiation. ALPN allows the application
	// layer to negotiate which protocol should be performed over a secure connection in a
	// manner which avoids additional round trips and which is independent of the application
	// layer protocols. It is used by HTTP/2.
	// If provided a list of alpn will be added to port as alpn option1,option2,...
	// If SecretName is Provided this secret will be used to terminate SSL with alpn options.
	// If Secret name is not provided backend server is responsible for handling SSL.
	ALPN []string `json:"alpn,omitempty"`
}

// HTTPExtendedIngressPath associates a path regex with a backend. Incoming urls matching
// the path are forwarded to the backend.
type HTTPExtendedIngressPath struct {
	// Path is a extended POSIX regex as defined by IEEE Std 1003.1,
	// (i.e this follows the egrep/unix syntax, not the perl syntax)
	// matched against the path of an incoming request. Currently it can
	// contain characters disallowed from the conventional "path"
	// part of a URL as defined by RFC 3986. Paths must begin with
	// a '/'. If unspecified, the path defaults to a catch all sending
	// traffic to the backend.
	Path string `json:"path,omitempty"`

	// Backend defines the referenced service endpoint to which the traffic
	// will be forwarded to.
	Backend ExtendedIngressBackend `json:"backend,omitempty"`
}

type IngressBackend struct {
	// Host names to forward traffic to. If empty traffic will be
	// forwarded to all subsets instance.
	// If set only matched hosts will get the traffic.
	// This is an handy way to send traffic to Specific
	// StatefulSet pod.
	// IE. Setting [web-0] will send traffic to only web-0 host
	// for this StatefulSet, https://kubernetes.io/docs/tutorials/stateful-application/basic-stateful-set/#creating-a-statefulset
	HostNames []string `json:"hostNames,omitempty"`

	// Specifies the name of the referenced service.
	ServiceName string `json:"serviceName,omitempty"`

	// Specifies the port of the referenced service.
	ServicePort intstr.IntOrString `json:"servicePort,omitempty"`
}

// ExtendedIngressBackend describes all endpoints for a given service and port.
type ExtendedIngressBackend struct {
	// TODO (@sadlil) Consider Embedding IngressBackend.

	// Host names to forward traffic to. If empty traffic will be
	// forwarded to all subsets instance.
	// If set only matched hosts will get the traffic.
	// This is an handy way to send traffic to Specific
	// StatefulSet pod.
	// IE. Setting [web-0] will send traffic to only web-0 host
	// for this StatefulSet, https://kubernetes.io/docs/tutorials/stateful-application/basic-stateful-set/#creating-a-statefulset
	HostNames []string `json:"hostNames,omitempty"`

	// Specifies the name of the referenced service.
	ServiceName string `json:"serviceName,omitempty"`

	// Specifies the port of the referenced service.
	ServicePort intstr.IntOrString `json:"servicePort,omitempty"`

	// Path rewrite rules with haproxy formatted regex.
	RewriteRule []string `json:"rewriteRule,omitempty"`

	// Header rules to modifies the header.
	HeaderRule []string `json:"headerRule,omitempty"`
}

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
	CreationTime        *unversioned.Time      `json:"creationTime,omitempty"`
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

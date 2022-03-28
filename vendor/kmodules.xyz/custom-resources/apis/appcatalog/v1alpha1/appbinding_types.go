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

package v1alpha1

import (
	core "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

const (
	ResourceKindApp = "AppBinding"
	ResourceApps    = "appbindings"
	ResourceApp     = "appbinding"
)

// AppBinding defines a generic user application.

// +genclient
// +genclient:skipVerbs=updateStatus
// +k8s:openapi-gen=true
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// +kubebuilder:object:root=true
// +kubebuilder:resource:path=appbindings,singular=appbinding,categories={catalog,appscode,all}
// +kubebuilder:printcolumn:name="Type",type="string",JSONPath=".spec.type"
// +kubebuilder:printcolumn:name="Version",type="string",JSONPath=".spec.version"
// +kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp"
type AppBinding struct {
	metav1.TypeMeta   `json:",inline,omitempty"`
	metav1.ObjectMeta `json:"metadata,omitempty" protobuf:"bytes,1,opt,name=metadata"`
	Spec              AppBindingSpec `json:"spec,omitempty" protobuf:"bytes,2,opt,name=spec"`
}

// AppBindingSpec is the spec for app
type AppBindingSpec struct {
	// Type used to facilitate programmatic handling of application.
	// +optional
	Type AppType `json:"type,omitempty" protobuf:"bytes,1,opt,name=type,casttype=AppType"`

	// Version used to facilitate programmatic handling of application.
	// +optional
	Version string `json:"version,omitempty" protobuf:"bytes,2,opt,name=version"`

	// ClientConfig defines how to communicate with the app.
	// Required
	ClientConfig ClientConfig `json:"clientConfig" protobuf:"bytes,3,opt,name=clientConfig"`

	// Secret is the name of the secret to create in the AppBinding's
	// namespace that will hold the credentials associated with the AppBinding.
	Secret *core.LocalObjectReference `json:"secret,omitempty" protobuf:"bytes,4,opt,name=secret"`

	// List of transformations that should be applied to the credentials
	// associated with the ServiceBinding before they are inserted into the Secret.
	SecretTransforms []SecretTransform `json:"secretTransforms,omitempty" protobuf:"bytes,5,rep,name=secretTransforms"`

	// Parameters is a set of the parameters to be used to connect to the
	// app. The inline YAML/JSON payload to be translated into equivalent
	// JSON object.
	//
	// The Parameters field is NOT secret or secured in any way and should
	// NEVER be used to hold sensitive information. To set parameters that
	// contain secret information, you should ALWAYS store that information
	// in a Secret.
	//
	// +optional
	// +kubebuilder:validation:EmbeddedResource
	// +kubebuilder:pruning:PreserveUnknownFields
	Parameters *runtime.RawExtension `json:"parameters,omitempty" protobuf:"bytes,6,opt,name=parameters"`

	// TLSSecret is the name of the secret that will hold
	// the client certificate and private key associated with the AppBinding.
	TLSSecret *core.LocalObjectReference `json:"tlsSecret,omitempty" protobuf:"bytes,7,opt,name=tlsSecret"`
}

type AppType string

const (
	// AppTypeOpaque is the default. A generic application.
	AppTypeOpaque AppType = "Opaque"
)

// ClientConfig contains the information to make a connection with an app
type ClientConfig struct {
	// `url` gives the location of the app, in standard URL form
	// (`[scheme://]host:port/path`). Exactly one of `url` or `service`
	// must be specified.
	//
	// The `host` should not refer to a service running in the cluster; use
	// the `service` field instead. The host might be resolved via external
	// DNS in some apiservers (e.g., `kube-apiserver` cannot resolve
	// in-cluster DNS as that would be a layering violation). `host` may
	// also be an IP address.
	//
	// A path is optional, and if present may be any string permissible in
	// a URL. You may use the path to pass an arbitrary string to the
	// app, for example, a cluster identifier.
	//
	// Attempting to use a user or basic auth e.g. "user:password@" is not
	// allowed. Fragments ("#...") and query parameters ("?...") are not
	// allowed, either.
	//
	// +optional
	URL *string `json:"url,omitempty" protobuf:"bytes,1,opt,name=url"`

	// `service` is a reference to the service for this app. Either
	// `service` or `url` must be specified.
	//
	// If the webhook is running within the cluster, then you should use `service`.
	//
	// +optional
	Service *ServiceReference `json:"service,omitempty" protobuf:"bytes,2,opt,name=service"`

	// InsecureSkipTLSVerify disables TLS certificate verification when communicating with this app.
	// This is strongly discouraged.  You should use the CABundle instead.
	InsecureSkipTLSVerify bool `json:"insecureSkipTLSVerify,omitempty" protobuf:"varint,3,opt,name=insecureSkipTLSVerify"`

	// CABundle is a PEM encoded CA bundle which will be used to validate the serving certificate of this app.
	// +optional
	CABundle []byte `json:"caBundle,omitempty" protobuf:"bytes,4,opt,name=caBundle"`

	// ServerName is used to verify the hostname on the returned
	// certificates unless InsecureSkipVerify is given. It is also included
	// in the client's handshake to support virtual hosting unless it is
	// an IP address.
	ServerName string `json:"serverName,omitempty" protobuf:"bytes,5,opt,name=serverName"`
}

// ServiceReference holds a reference to Service.legacy.k8s.io
type ServiceReference struct {
	// Specifies which scheme to use, for example: http, https
	// If specified, then it will applied as prefix in this format: scheme://
	// If not specified, then nothing will be prefixed
	Scheme string `json:"scheme" protobuf:"bytes,1,opt,name=scheme"`

	// `namespace` is the namespace of the service.
	// +optional
	Namespace string `json:"namespace,omitempty" protobuf:"bytes,2,opt,name=namespace"`

	// `name` is the name of the service.
	// Required
	Name string `json:"name" protobuf:"bytes,3,opt,name=name"`

	// The port that will be exposed by this app.
	Port int32 `json:"port" protobuf:"varint,4,opt,name=port"`

	// `path` is an optional URL path which will be sent in any request to
	// this service.
	// +optional
	Path string `json:"path,omitempty" protobuf:"bytes,5,opt,name=path"`

	// `query` is optional encoded query string, without '?' which will be
	// sent in any request to this service.
	// +optional
	Query string `json:"query,omitempty" protobuf:"bytes,6,opt,name=query"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// AppBindingList is a list of Apps
type AppBindingList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty" protobuf:"bytes,1,opt,name=metadata"`
	// Items is a list of AppBinding CRD objects
	Items []AppBinding `json:"items,omitempty" protobuf:"bytes,2,rep,name=items"`
}

type AppReference struct {
	// `namespace` is the namespace of the app.
	// Required
	Namespace string `json:"namespace" protobuf:"bytes,1,opt,name=namespace"`

	// `name` is the name of the app.
	// Required
	Name string `json:"name" protobuf:"bytes,2,opt,name=name"`

	// Parameters is a set of the parameters to be used to override default
	// parameters. The inline YAML/JSON payload to be translated into equivalent
	// JSON object.
	//
	// The Parameters field is NOT secret or secured in any way and should
	// NEVER be used to hold sensitive information.
	//
	// +optional
	// +kubebuilder:validation:EmbeddedResource
	// +kubebuilder:pruning:PreserveUnknownFields
	Parameters *runtime.RawExtension `json:"parameters,omitempty" protobuf:"bytes,3,opt,name=parameters"`
}

type AppBindingMeta interface {
	Name() string
	Type() AppType
}

// ObjectReference contains enough information to let you locate the
// referenced object.
type ObjectReference struct {
	// Namespace of the referent.
	Namespace string `json:"namespace,omitempty" protobuf:"bytes,1,opt,name=namespace"`
	// Name of the referent.
	Name string `json:"name,omitempty" protobuf:"bytes,2,opt,name=name"`
}

// ref: https://github.com/kubernetes-sigs/service-catalog/blob/37b874716ad709a175e426f5f5638322a600849f/pkg/apis/servicecatalog/v1beta1/types.go#L1397

// SecretTransform is a single transformation that is applied to the
// credentials returned from the broker before they are inserted into
// the Secret associated with the ServiceBinding.
// Because different brokers providing the same type of service may
// each return a different credentials structure, users can specify
// the transformations that should be applied to the Secret to adapt
// its entries to whatever the service consumer expects.
// For example, the credentials returned by the broker may include the
// key "USERNAME", but the consumer requires the username to be
// exposed under the key "DB_USER" instead. To have the Service
// Catalog transform the Secret, the following SecretTransform must
// be specified in ServiceBinding.spec.secretTransform:
// - {"renameKey": {"from": "USERNAME", "to": "DB_USER"}}
// Only one of the SecretTransform's members may be specified.
type SecretTransform struct {
	// RenameKey represents a transform that renames a credentials Secret entry's key
	RenameKey *RenameKeyTransform `json:"renameKey,omitempty" protobuf:"bytes,1,opt,name=renameKey"`
	// AddKey represents a transform that adds an additional key to the credentials Secret
	AddKey *AddKeyTransform `json:"addKey,omitempty" protobuf:"bytes,2,opt,name=addKey"`
	// AddKeysFrom represents a transform that merges all the entries of an existing Secret
	// into the credentials Secret
	AddKeysFrom *AddKeysFromTransform `json:"addKeysFrom,omitempty" protobuf:"bytes,3,opt,name=addKeysFrom"`
	// RemoveKey represents a transform that removes a credentials Secret entry
	RemoveKey *RemoveKeyTransform `json:"removeKey,omitempty" protobuf:"bytes,4,opt,name=removeKey"`
}

// RenameKeyTransform specifies that one of the credentials keys returned
// from the broker should be renamed and stored under a different key
// in the Secret.
// For example, given the following credentials entry:
//     "USERNAME": "johndoe"
// and the following RenameKeyTransform:
//     {"from": "USERNAME", "to": "DB_USER"}
// the following entry will appear in the Secret:
//     "DB_USER": "johndoe"
type RenameKeyTransform struct {
	// The name of the key to rename
	From string `json:"from" protobuf:"bytes,1,opt,name=from"`
	// The new name for the key
	To string `json:"to" protobuf:"bytes,2,opt,name=to"`
}

// AddKeyTransform specifies that Service Catalog should add an
// additional entry to the Secret associated with the ServiceBinding.
// For example, given the following AddKeyTransform:
//     {"key": "CONNECTION_POOL_SIZE", "stringValue": "10"}
// the following entry will appear in the Secret:
//     "CONNECTION_POOL_SIZE": "10"
// Note that this transform should only be used to add non-sensitive
// (non-secret) values. To add sensitive information, the
// AddKeysFromTransform should be used instead.
type AddKeyTransform struct {
	// The name of the key to add
	Key string `json:"key" protobuf:"bytes,1,opt,name=key"`
	// The binary value (possibly non-string) to add to the Secret under the specified key. If both
	// value and stringValue are specified, then value is ignored and stringValue is stored.
	// +optional
	Value []byte `json:"value,omitempty" protobuf:"bytes,2,opt,name=value"`
	// The string (non-binary) value to add to the Secret under the specified key.
	// +optional
	StringValue *string `json:"stringValue,omitempty" protobuf:"bytes,3,opt,name=stringValue"`
}

// AddKeysFromTransform specifies that Service Catalog should merge
// an existing secret into the Secret associated with the ServiceBinding.
// For example, given the following AddKeysFromTransform:
//     {"secretRef": {"namespace": "foo", "name": "bar"}}
// the entries of the Secret "bar" from Namespace "foo" will be merged into
// the credentials Secret.
type AddKeysFromTransform struct {
	// The reference to the Secret that should be merged into the credentials Secret.
	SecretRef *core.LocalObjectReference `json:"secretRef,omitempty" protobuf:"bytes,1,opt,name=secretRef"`
}

// RemoveKeyTransform specifies that one of the credentials keys returned
// from the broker should not be included in the credentials Secret.
type RemoveKeyTransform struct {
	// The key to remove from the Secret
	Key string `json:"key" protobuf:"bytes,1,opt,name=key"`
}

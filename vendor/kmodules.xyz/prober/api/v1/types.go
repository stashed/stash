/*
Copyright The Kmodules Authors.

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
	core "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

// Handler defines a specific action that should be taken
// TODO: pass structured data to these actions, and document that data here.
type Handler struct {
	// One and only one of the following should be specified.
	// Exec specifies the action to take.
	// +optional
	Exec *core.ExecAction `json:"exec,omitempty" protobuf:"bytes,1,opt,name=exec"`
	// HTTPGet specifies the http Get request to perform.
	// +optional
	HTTPGet *core.HTTPGetAction `json:"httpGet,omitempty" protobuf:"bytes,2,opt,name=httpGet"`
	// HTTPPost specifies the http Post request to perform.
	// +optional
	HTTPPost *HTTPPostAction `json:"httpPost,omitempty" protobuf:"bytes,3,opt,name=httpPost"`
	// TCPSocket specifies an action involving a TCP port.
	// TCP hooks not yet supported
	// TODO: implement a realistic TCP lifecycle hook
	// +optional
	TCPSocket *core.TCPSocketAction `json:"tcpSocket,omitempty" protobuf:"bytes,4,opt,name=tcpSocket"`
	// ContainerName specifies the name of the container where to execute the commands for Exec probe
	// or where to find the port for HTTP or TCP probe
	// +optional
	ContainerName string `json:"containerName,omitempty" protobuf:"bytes,5,opt,name=containerName"`
}

// HTTPPostAction describes an action based on HTTP Post requests.
type HTTPPostAction struct {
	// Path to access on the HTTP server.
	// +optional
	Path string `json:"path,omitempty" protobuf:"bytes,1,opt,name=path"`
	// Name or number of the port to access on the container.
	// Number must be in the range 1 to 65535.
	// Name must be an IANA_SVC_NAME.
	Port intstr.IntOrString `json:"port" protobuf:"bytes,2,opt,name=port"`
	// Host name to connect to, defaults to the pod IP. You probably want to set
	// "Host" in httpHeaders instead.
	// +optional
	Host string `json:"host,omitempty" protobuf:"bytes,3,opt,name=host"`
	// Scheme to use for connecting to the host.
	// Defaults to HTTP.
	// +optional
	Scheme core.URIScheme `json:"scheme,omitempty" protobuf:"bytes,4,opt,name=scheme,casttype=k8s.io/api/core/v1.URIScheme"`
	// Custom headers to set in the request. HTTP allows repeated headers.
	// +optional
	HTTPHeaders []core.HTTPHeader `json:"httpHeaders,omitempty" protobuf:"bytes,5,rep,name=httpHeaders"`
	// Body to set in the request.
	// +optional
	Body string `json:"body,omitempty" protobuf:"bytes,6,opt,name=body"`
	// Form to set in the request body.
	// +optional
	Form []FormEntry `json:"form,omitempty" protobuf:"bytes,7,rep,name=form"`
}

type FormEntry struct {
	Key    string   `json:"key,omitempty" protobuf:"bytes,1,rep,name=key"`
	Values []string `json:"values,omitempty" protobuf:"bytes,2,rep,name=values"`
}

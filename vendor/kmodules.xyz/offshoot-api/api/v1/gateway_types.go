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
	core "k8s.io/api/core/v1"
)

type Gateway struct {
	Name      string `json:"name"`
	Namespace string `json:"namespace"`
	// +optional
	IP string `json:"ip,omitempty"`
	// +optional
	Hostname string `json:"hostname,omitempty"`
	// Services is an optional configuration for services used to expose database
	// +optional
	Services []NamedServiceStatus `json:"services,omitempty"`
	// UI is an optional list of database web uis
	// +optional
	UI []NamedURL `json:"ui,omitempty"`
}

type NamedServiceStatus struct {
	// Alias represents the identifier of the service.
	Alias string `json:"alias"`

	Ports []GatewayPort `json:"ports"`
}

type NamedURL struct {
	// Alias represents the identifier of the service.
	// This should match the db ui chart name
	Alias string `json:"alias"`

	// URL of the database ui
	URL string `json:"url"`

	Port GatewayPort `json:"port"`

	// HelmRelease is the name of the helm release used to deploy this ui
	// The name format is typically <alias>-<db-name>
	// +optional
	HelmRelease *core.LocalObjectReference `json:"helmRelease,omitempty"`
}

// GatewayPort contains information on Gateway service's port.
type GatewayPort struct {
	// The name of this port within the gateway service.
	// +optional
	Name string `json:"name,omitempty"`

	// The port that will be exposed by the gateway service.
	Port int32 `json:"port"`

	// Number of the port to access the backend service.
	// +optional
	BackendServicePort int32 `json:"backendServicePort,omitempty"`

	// The port on each node on which this gateway service is exposed when type is
	// NodePort or LoadBalancer.
	// +optional
	NodePort int32 `json:"nodePort,omitempty"`
}

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

//go:generate go-enum --mustparse --names --values
package v1

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
)

// +kubebuilder:validation:Enum=AKS;DigitalOcean;EKS;Exoscale;Generic;GKE;Linode;Packet;Rancher;Scaleway;Vultr
type HostingProvider string

const (
	HostingProviderAKS          HostingProvider = "AKS"
	HostingProviderDigitalOcean HostingProvider = "DigitalOcean"
	HostingProviderEKS          HostingProvider = "EKS"
	HostingProviderExoscale     HostingProvider = "Exoscale"
	HostingProviderGeneric      HostingProvider = "Generic"
	HostingProviderGKE          HostingProvider = "GKE"
	HostingProviderLinode       HostingProvider = "Linode"
	HostingProviderAkamai       HostingProvider = "Akamai"
	HostingProviderPacket       HostingProvider = "Packet"
	HostingProviderRancher      HostingProvider = "Rancher"
	HostingProviderScaleway     HostingProvider = "Scaleway"
	HostingProviderVultr        HostingProvider = "Vultr"
)

func (h HostingProvider) ConvertToPreferredProvider() HostingProvider {
	switch h {
	case HostingProviderLinode:
		return HostingProviderAkamai
	}
	return h
}

const (
	AceInfoConfigMapName = "ace-info"
	AceMachineProfileKey = "kubernetes.io/instance-type"

	ClusterNameKey         string = "cluster.appscode.com/name"
	ClusterDisplayNameKey  string = "cluster.appscode.com/display-name"
	ClusterProviderNameKey string = "cluster.appscode.com/provider"
	ClusterModeKey         string = "cluster.appscode.com/mode"
	ClusterProfileLabel    string = "cluster.appscode.com/profile"

	AceOrgIDKey               string = "ace.appscode.com/org-id"
	AceEnableResourceTrialKey string = "ace.appscode.com/enable-resource-trial"
	ClientOrgKey              string = "ace.appscode.com/client-org"
	ClientOrgMonitoringKey    string = "ace.appscode.com/client-org-monitoring"
	ClientKeyPrefix           string = "client.ace.appscode.com/"

	ClusterClaimKeyID       string = "id.k8s.io"
	ClusterClaimKeyInfo     string = "cluster.ace.info"
	ClusterClaimKeyFeatures string = "features.ace.info"
)

type ClusterMetadata struct {
	UID                  string          `json:"uid" protobuf:"bytes,1,opt,name=uid"`
	Name                 string          `json:"name,omitempty" protobuf:"bytes,2,opt,name=name"`
	DisplayName          string          `json:"displayName,omitempty" protobuf:"bytes,3,opt,name=displayName"`
	Provider             HostingProvider `json:"provider,omitempty" protobuf:"bytes,4,opt,name=provider,casttype=HostingProvider"`
	OwnerID              string          `json:"ownerID,omitempty" protobuf:"bytes,5,opt,name=ownerID"`
	OwnerType            string          `json:"ownerType,omitempty" protobuf:"bytes,6,opt,name=ownerType"`
	APIEndpoint          string          `json:"apiEndpoint,omitempty" protobuf:"bytes,7,opt,name=apiEndpoint"`
	CABundle             string          `json:"caBundle,omitempty" protobuf:"bytes,8,opt,name=caBundle"`
	ManagerID            string          `json:"managerID,omitempty" protobuf:"bytes,9,opt,name=managerID"`
	HubClusterID         string          `json:"hubClusterID,omitempty" protobuf:"bytes,10,opt,name=hubClusterID"`
	CloudServiceAuthMode string          `json:"cloudServiceAuthMode,omitempty" protobuf:"bytes,11,opt,name=cloudServiceAuthMode"`
	Mode                 ClusterMode     `json:"mode,omitempty" protobuf:"bytes,12,opt,name=mode,casttype=ClusterMode"`
}

func (md ClusterMetadata) Manager() string {
	if md.ManagerID != "" && md.ManagerID != "0" {
		return md.ManagerID
	}
	return md.OwnerID
}

func (md ClusterMetadata) State() string {
	hasher := hmac.New(sha256.New, []byte(md.UID))
	state := fmt.Sprintf("%s,%s", md.APIEndpoint, md.Manager())
	hasher.Write([]byte(state))
	return base64.URLEncoding.EncodeToString(hasher.Sum(nil))
}

// +kubebuilder:validation:Enum=prod;qa;staging;dev
// ENUM(prod,qa,staging,dev)
type ClusterMode string

//
//const (
//	ClusterModeProd    ClusterMode = "prod"
//	ClusterModeQA      ClusterMode = "qa"
//	ClusterModeStaging ClusterMode = "staging"
//	ClusterModeDev     ClusterMode = "dev"
//)

type CAPIClusterInfo struct {
	Provider    CAPIProvider `json:"provider" protobuf:"bytes,1,opt,name=provider,casttype=CAPIProvider"`
	Namespace   string       `json:"namespace" protobuf:"bytes,2,opt,name=namespace"`
	ClusterName string       `json:"clusterName" protobuf:"bytes,3,opt,name=clusterName"`
}

// ClusterInfo used in ace-installer
type ClusterInfo struct {
	UID             string   `json:"uid" protobuf:"bytes,1,opt,name=uid"`
	Name            string   `json:"name" protobuf:"bytes,2,opt,name=name"`
	ClusterManagers []string `json:"clusterManagers" protobuf:"bytes,3,rep,name=clusterManagers"`
	// +optional
	CAPI *CAPIClusterInfo `json:"capi" protobuf:"bytes,4,opt,name=capi"`
}

// +kubebuilder:validation:Enum=capa;capg;capz;caph;capk
type CAPIProvider string

const (
	CAPIProviderCAPA CAPIProvider = "capa"
	CAPIProviderCAPG CAPIProvider = "capg"
	CAPIProviderCAPZ CAPIProvider = "capz"
	CAPIProviderCAPH CAPIProvider = "caph"
	CAPIProviderCAPK CAPIProvider = "capk"
)

type ClusterClaimInfo struct {
	ClusterMetadata ClusterInfo `json:"clusterMetadata" protobuf:"bytes,1,opt,name=clusterMetadata"`
}

type ClusterClaimFeatures struct {
	EnabledFeatures           []string `json:"enabledFeatures,omitempty" protobuf:"bytes,1,rep,name=enabledFeatures"`
	ExternallyManagedFeatures []string `json:"externallyManagedFeatures,omitempty" protobuf:"bytes,2,rep,name=externallyManagedFeatures"`
	DisabledFeatures          []string `json:"disabledFeatures,omitempty" protobuf:"bytes,3,rep,name=disabledFeatures"`
}

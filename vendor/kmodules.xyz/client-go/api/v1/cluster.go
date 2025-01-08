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
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"strings"
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

	ClusterNameKey         string = "cluster.appscode.com/name"
	ClusterDisplayNameKey  string = "cluster.appscode.com/display-name"
	ClusterProviderNameKey string = "cluster.appscode.com/provider"
	ClusterProfileLabel    string = "cluster.appscode.com/profile"

	AceOrgIDKey     string = "ace.appscode.com/org-id"
	ClientOrgKey    string = "ace.appscode.com/client-org"
	ClientKeyPrefix string = "client.ace.appscode.com/"

	ClusterClaimKeyID       string = "id.k8s.io"
	ClusterClaimKeyInfo     string = "cluster.ace.info"
	ClusterClaimKeyFeatures string = "features.ace.info"
)

type ClusterMetadata struct {
	UID          string          `json:"uid" protobuf:"bytes,1,opt,name=uid"`
	Name         string          `json:"name,omitempty" protobuf:"bytes,2,opt,name=name"`
	DisplayName  string          `json:"displayName,omitempty" protobuf:"bytes,3,opt,name=displayName"`
	Provider     HostingProvider `json:"provider,omitempty" protobuf:"bytes,4,opt,name=provider,casttype=HostingProvider"`
	OwnerID      string          `json:"ownerID,omitempty" protobuf:"bytes,5,opt,name=ownerID"`
	OwnerType    string          `json:"ownerType,omitempty" protobuf:"bytes,6,opt,name=ownerType"`
	APIEndpoint  string          `json:"apiEndpoint,omitempty" protobuf:"bytes,7,opt,name=apiEndpoint"`
	CABundle     string          `json:"caBundle,omitempty" protobuf:"bytes,8,opt,name=caBundle"`
	ManagerID    string          `json:"managerID,omitempty" protobuf:"bytes,9,opt,name=managerID"`
	HubClusterID string          `json:"hubClusterID,omitempty" protobuf:"bytes,10,opt,name=hubClusterID"`
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

/*
ENUM(

	ACE                         = 1
	OCMHub                      = 2
	OCMMulticlusterControlplane = 4
	OCMSpoke                    = 8
	OpenShift                   = 16
	Rancher                     = 32
	VirtualCluster              = 64

)
*/
type ClusterManager int

const (
	ClusterManagerACE ClusterManager = 1 << iota
	ClusterManagerOCMHub
	ClusterManagerOCMMulticlusterControlplane
	ClusterManagerOCMSpoke
	ClusterManagerOpenShift
	ClusterManagerRancher
	ClusterManagerVirtualCluster
)

func (cm ClusterManager) ManagedByACE() bool {
	return cm&ClusterManagerACE == ClusterManagerACE
}

func (cm ClusterManager) ManagedByOCMHub() bool {
	return cm&ClusterManagerOCMHub == ClusterManagerOCMHub
}

func (cm ClusterManager) ManagedByOCMSpoke() bool {
	return cm&ClusterManagerOCMSpoke == ClusterManagerOCMSpoke
}

func (cm ClusterManager) ManagedByOCMMulticlusterControlplane() bool {
	return cm&ClusterManagerOCMMulticlusterControlplane == ClusterManagerOCMMulticlusterControlplane
}

func (cm ClusterManager) ManagedByRancher() bool {
	return cm&ClusterManagerRancher == ClusterManagerRancher
}

func (cm ClusterManager) ManagedByOpenShift() bool {
	return cm&ClusterManagerOpenShift == ClusterManagerOpenShift
}

func (cm ClusterManager) ManagedByVirtualCluster() bool {
	return cm&ClusterManagerVirtualCluster == ClusterManagerVirtualCluster
}

func (cm ClusterManager) Strings() []string {
	out := make([]string, 0, 7)
	if cm.ManagedByACE() {
		out = append(out, "ACE")
	}
	if cm.ManagedByOCMHub() {
		out = append(out, "OCMHub")
	}
	if cm.ManagedByOCMSpoke() {
		out = append(out, "OCMSpoke")
	}
	if cm.ManagedByOCMMulticlusterControlplane() {
		out = append(out, "OCMMulticlusterControlplane")
	}
	if cm.ManagedByRancher() {
		out = append(out, "Rancher")
	}
	if cm.ManagedByOpenShift() {
		out = append(out, "OpenShift")
	}
	if cm.ManagedByVirtualCluster() {
		out = append(out, "vcluster")
	}
	return out
}

func (cm ClusterManager) String() string {
	return strings.Join(cm.Strings(), ",")
}

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
	ClusterMetadata ClusterInfo `json:"clusterMetadata"`
}

type ClusterClaimFeatures struct {
	EnabledFeatures           []string `json:"enabledFeatures,omitempty"`
	ExternallyManagedFeatures []string `json:"externallyManagedFeatures,omitempty"`
	DisabledFeatures          []string `json:"disabledFeatures,omitempty"`
}

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

import "strings"

// +kubebuilder:validation:Enum=Aws;Azure;DigitalOcean;GoogleCloud;Linode;Packet;Scaleway;Vultr;BareMetal;KIND;Generic;Private
type HostingProvider string

const (
	HostingProviderAWS          HostingProvider = "Aws"
	HostingProviderAzure        HostingProvider = "Azure"
	HostingProviderDigitalOcean HostingProvider = "DigitalOcean"
	HostingProviderGoogleCloud  HostingProvider = "GoogleCloud"
	HostingProviderLinode       HostingProvider = "Linode"
	HostingProviderPacket       HostingProvider = "Packet"
	HostingProviderScaleway     HostingProvider = "Scaleway"
	HostingProviderVultr        HostingProvider = "Vultr"
	HostingProviderBareMetal    HostingProvider = "BareMetal"
	HostingProviderKIND         HostingProvider = "KIND"
	HostingProviderGeneric      HostingProvider = "Generic"
	HostingProviderPrivate      HostingProvider = "Private"
)

const (
	ClusterNameKey         string = "cluster.appscode.com/name"
	ClusterDisplayNameKey  string = "cluster.appscode.com/display-name"
	ClusterProviderNameKey string = "cluster.appscode.com/provider"
)

type ClusterMetadata struct {
	UID         string          `json:"uid" protobuf:"bytes,1,opt,name=uid"`
	Name        string          `json:"name,omitempty" protobuf:"bytes,2,opt,name=name"`
	DisplayName string          `json:"displayName,omitempty" protobuf:"bytes,3,opt,name=displayName"`
	Provider    HostingProvider `json:"provider,omitempty" protobuf:"bytes,4,opt,name=provider,casttype=HostingProvider"`
}

type ClusterManager int

const (
	ClusterManagerACE ClusterManager = 1 << iota
	ClusterManagerOCMHub
	ClusterManagerOCMSpoke
	ClusterManagerOCMMulticlusterControlplane
	ClusterManagerRancher
	ClusterManagerOpenShift
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
	Provider    string `json:"provider,omitempty"`
	Namespace   string `json:"namespace,omitempty"`
	ClusterName string `json:"clusterName,omitempty"`
}

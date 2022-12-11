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

const (
	// Deprecated: Use kmodules.xyz/constants/aws
	AWS_ACCESS_KEY_ID = "AWS_ACCESS_KEY_ID"
	// Deprecated: Use kmodules.xyz/constants/aws
	AWS_SECRET_ACCESS_KEY = "AWS_SECRET_ACCESS_KEY"
	// Deprecated: Use kmodules.xyz/constants/aws
	CA_CERT_DATA = "CA_CERT_DATA"

	// Deprecated: Use kmodules.xyz/constants/google
	GOOGLE_PROJECT_ID = "GOOGLE_PROJECT_ID"
	// Deprecated: Use kmodules.xyz/constants/google
	GOOGLE_SERVICE_ACCOUNT_JSON_KEY = "GOOGLE_SERVICE_ACCOUNT_JSON_KEY"
	// Deprecated: Use kmodules.xyz/constants/google
	GOOGLE_APPLICATION_CREDENTIALS = "GOOGLE_APPLICATION_CREDENTIALS"

	// Deprecated: Use kmodules.xyz/constants/azure
	AZURE_ACCOUNT_NAME = "AZURE_ACCOUNT_NAME"
	// Deprecated: Use kmodules.xyz/constants/azure
	AZURE_ACCOUNT_KEY = "AZURE_ACCOUNT_KEY"

	// swift
	// Deprecated: Use kmodules.xyz/constants/openstack
	OS_USERNAME = "OS_USERNAME"
	// Deprecated: Use kmodules.xyz/constants/openstack
	OS_PASSWORD = "OS_PASSWORD"
	// Deprecated: Use kmodules.xyz/constants/openstack
	OS_REGION_NAME = "OS_REGION_NAME"
	// Deprecated: Use kmodules.xyz/constants/openstack
	OS_AUTH_URL = "OS_AUTH_URL"

	// v3 specific
	// Deprecated: Use kmodules.xyz/constants/openstack
	OS_USER_DOMAIN_NAME = "OS_USER_DOMAIN_NAME"
	// Deprecated: Use kmodules.xyz/constants/openstack
	OS_PROJECT_NAME = "OS_PROJECT_NAME"
	// Deprecated: Use kmodules.xyz/constants/openstack
	OS_PROJECT_DOMAIN_NAME = "OS_PROJECT_DOMAIN_NAME"

	// v2 specific
	// Deprecated: Use kmodules.xyz/constants/openstack
	OS_TENANT_ID = "OS_TENANT_ID"
	// Deprecated: Use kmodules.xyz/constants/openstack
	OS_TENANT_NAME = "OS_TENANT_NAME"

	// v1 specific
	// Deprecated: Use kmodules.xyz/constants/openstack
	ST_AUTH = "ST_AUTH"
	// Deprecated: Use kmodules.xyz/constants/openstack
	ST_USER = "ST_USER"
	// Deprecated: Use kmodules.xyz/constants/openstack
	ST_KEY = "ST_KEY"

	// Manual authentication
	// Deprecated: Use kmodules.xyz/constants/openstack
	OS_STORAGE_URL = "OS_STORAGE_URL"
	// Deprecated: Use kmodules.xyz/constants/openstack
	OS_AUTH_TOKEN = "OS_AUTH_TOKEN"
)

type Backend struct {
	StorageSecretName string `json:"storageSecretName,omitempty" protobuf:"bytes,1,opt,name=storageSecretName"`

	Local *LocalSpec      `json:"local,omitempty" protobuf:"bytes,2,opt,name=local"`
	S3    *S3Spec         `json:"s3,omitempty" protobuf:"bytes,3,opt,name=s3"`
	GCS   *GCSSpec        `json:"gcs,omitempty" protobuf:"bytes,4,opt,name=gcs"`
	Azure *AzureSpec      `json:"azure,omitempty" protobuf:"bytes,5,opt,name=azure"`
	Swift *SwiftSpec      `json:"swift,omitempty" protobuf:"bytes,6,opt,name=swift"`
	B2    *B2Spec         `json:"b2,omitempty" protobuf:"bytes,7,opt,name=b2"`
	Rest  *RestServerSpec `json:"rest,omitempty" protobuf:"bytes,8,opt,name=rest"`
}

type LocalSpec struct {
	core.VolumeSource `json:",inline" protobuf:"bytes,1,opt,name=volumeSource"`
	MountPath         string `json:"mountPath" protobuf:"bytes,2,opt,name=mountPath"`
	SubPath           string `json:"subPath,omitempty" protobuf:"bytes,3,opt,name=subPath"`
}

type S3Spec struct {
	Endpoint string `json:"endpoint" protobuf:"bytes,1,opt,name=endpoint"`
	Bucket   string `json:"bucket" protobuf:"bytes,2,opt,name=bucket"`
	Prefix   string `json:"prefix,omitempty" protobuf:"bytes,3,opt,name=prefix"`
	Region   string `json:"region,omitempty" protobuf:"bytes,4,opt,name=region"`
}

type GCSSpec struct {
	Bucket         string `json:"bucket" protobuf:"bytes,1,opt,name=bucket"`
	Prefix         string `json:"prefix,omitempty" protobuf:"bytes,2,opt,name=prefix"`
	MaxConnections int64  `json:"maxConnections,omitempty" protobuf:"varint,3,opt,name=maxConnections"`
}

type AzureSpec struct {
	Container      string `json:"container" protobuf:"bytes,1,opt,name=container"`
	Prefix         string `json:"prefix,omitempty" protobuf:"bytes,2,opt,name=prefix"`
	MaxConnections int64  `json:"maxConnections,omitempty" protobuf:"varint,3,opt,name=maxConnections"`
}

type SwiftSpec struct {
	Container string `json:"container" protobuf:"bytes,1,opt,name=container"`
	Prefix    string `json:"prefix,omitempty" protobuf:"bytes,2,opt,name=prefix"`
}

type B2Spec struct {
	Bucket         string `json:"bucket" protobuf:"bytes,1,opt,name=bucket"`
	Prefix         string `json:"prefix,omitempty" protobuf:"bytes,2,opt,name=prefix"`
	MaxConnections int64  `json:"maxConnections,omitempty" protobuf:"varint,3,opt,name=maxConnections"`
}

type RestServerSpec struct {
	URL string `json:"url,omitempty" protobuf:"bytes,1,opt,name=url"`
}

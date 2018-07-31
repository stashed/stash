package api

import (
	core "k8s.io/api/core/v1"
)

const (
	AWS_ACCESS_KEY_ID     = "AWS_ACCESS_KEY_ID"
	AWS_SECRET_ACCESS_KEY = "AWS_SECRET_ACCESS_KEY"
	CA_CERT_DATA          = "CA_CERT_DATA"

	GOOGLE_PROJECT_ID               = "GOOGLE_PROJECT_ID"
	GOOGLE_SERVICE_ACCOUNT_JSON_KEY = "GOOGLE_SERVICE_ACCOUNT_JSON_KEY"

	AZURE_ACCOUNT_NAME = "AZURE_ACCOUNT_NAME"
	AZURE_ACCOUNT_KEY  = "AZURE_ACCOUNT_KEY"

	// swift
	OS_USERNAME    = "OS_USERNAME"
	OS_PASSWORD    = "OS_PASSWORD"
	OS_REGION_NAME = "OS_REGION_NAME"
	OS_AUTH_URL    = "OS_AUTH_URL"

	// v3 specific
	OS_USER_DOMAIN_NAME    = "OS_USER_DOMAIN_NAME"
	OS_PROJECT_NAME        = "OS_PROJECT_NAME"
	OS_PROJECT_DOMAIN_NAME = "OS_PROJECT_DOMAIN_NAME"

	// v2 specific
	OS_TENANT_ID   = "OS_TENANT_ID"
	OS_TENANT_NAME = "OS_TENANT_NAME"

	// v1 specific
	ST_AUTH = "ST_AUTH"
	ST_USER = "ST_USER"
	ST_KEY  = "ST_KEY"

	// Manual authentication
	OS_STORAGE_URL = "OS_STORAGE_URL"
	OS_AUTH_TOKEN  = "OS_AUTH_TOKEN"
)

type Backend struct {
	StorageSecretName string `json:"storageSecretName,omitempty"`

	Local *LocalSpec `json:"local,omitempty"`
	S3    *S3Spec    `json:"s3,omitempty"`
	GCS   *GCSSpec   `json:"gcs,omitempty"`
	Azure *AzureSpec `json:"azure,omitempty"`
	Swift *SwiftSpec `json:"swift,omitempty"`
	B2    *B2Spec    `json:"b2,omitempty"`
	// Rest  *RestServerSpec `json:"rest,omitempty"`
}

type LocalSpec struct {
	core.VolumeSource `json:",inline"`
	MountPath         string `json:"mountPath,omitempty"`
	SubPath           string `json:"subPath,omitempty"`
}

type S3Spec struct {
	Endpoint string `json:"endpoint,omitempty"`
	Bucket   string `json:"bucket,omitempty"`
	Prefix   string `json:"prefix,omitempty"`
}

type GCSSpec struct {
	Bucket string `json:"bucket,omitempty"`
	Prefix string `json:"prefix,omitempty"`
}

type AzureSpec struct {
	Container string `json:"container,omitempty"`
	Prefix    string `json:"prefix,omitempty"`
}

type SwiftSpec struct {
	Container string `json:"container,omitempty"`
	Prefix    string `json:"prefix,omitempty"`
}

type B2Spec struct {
	Bucket string `json:"bucket,omitempty"`
	Prefix string `json:"prefix,omitempty"`
}

type RestServerSpec struct {
	URL string `json:"url,omitempty"`
}

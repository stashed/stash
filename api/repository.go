package api

import "fmt"

func (r LocalSpec) Repository() string {
	return r.Path
}

func (r S3Spec) Repository() string {
	return fmt.Sprintf("s3:%s:%s:%s", r.Endpoint, r.Bucket, r.Prefix)
}

func (r GCSSpec) Repository() string {
	return fmt.Sprintf("gs:%s:%s:%s", r.Location, r.Bucket, r.Prefix)
}

func (r AzureSpec) Repository() string {
	return fmt.Sprintf("azure:%s:%s", r.Container, r.Prefix)
}

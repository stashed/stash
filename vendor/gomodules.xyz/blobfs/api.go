package blobfs

import (
	"context"

	"gocloud.dev/blob"
)

type Interface interface {
	WriteFile(ctx context.Context, filepath string, data []byte) error
	ReadFile(ctx context.Context, filepath string) ([]byte, error)
	DeleteFile(ctx context.Context, filepath string) error
	Exists(ctx context.Context, filepath string) (bool, error)
	SignedURL(ctx context.Context, filepath string, opts *blob.SignedURLOptions) (string, error)
}

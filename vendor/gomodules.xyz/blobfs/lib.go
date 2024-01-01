package blobfs

import (
	"bytes"
	"context"
	"io"
	"path"
	"strings"

	"gocloud.dev/blob"
	_ "gocloud.dev/blob/fileblob"
	_ "gocloud.dev/blob/memblob"
)

type BlobFS struct {
	storageURL string
	prefix     string
}

func New(storageURL string, prefix ...string) Interface {
	var bucketPrefix string
	if len(prefix) > 0 {
		bucketPrefix = prefix[0]
	}

	return &BlobFS{
		storageURL: storageURL,
		prefix:     bucketPrefix,
	}
}

func NewInMemoryFS() Interface {
	return New("mem://")
}

func NewOsFs() Interface {
	return New("file:///")
}

func (fs *BlobFS) WriteFile(ctx context.Context, filepath string, data []byte) error {
	dir, filename := path.Split(filepath)
	bucket, err := fs.openBucket(ctx, path.Join(fs.prefix, dir))
	if err != nil {
		return err
	}
	defer bucket.Close()

	w, err := bucket.NewWriter(ctx, filename, nil)
	if err != nil {
		return err
	}
	_, writeErr := w.Write(data)
	// Always check the return value of Close when writing.
	closeErr := w.Close()
	if writeErr != nil {
		return writeErr
	}
	if closeErr != nil {
		return closeErr
	}
	return nil
}

func (fs *BlobFS) ReadFile(ctx context.Context, filepath string) ([]byte, error) {
	dir, filename := path.Split(filepath)
	bucket, err := fs.openBucket(ctx, path.Join(fs.prefix, dir))
	if err != nil {
		return nil, err
	}
	defer bucket.Close()
	// Open the key "foo.txt" for reading with the default options.
	r, err := bucket.NewReader(ctx, filename, nil)
	if err != nil {
		return nil, err
	}
	defer r.Close()

	var buf bytes.Buffer
	if _, err := io.Copy(&buf, r); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

func (fs *BlobFS) DeleteFile(ctx context.Context, filepath string) error {
	dir, filename := path.Split(filepath)
	bucket, err := fs.openBucket(ctx, path.Join(fs.prefix, dir))
	if err != nil {
		return err
	}
	defer bucket.Close()

	return bucket.Delete(context.TODO(), filename)
}

func (fs *BlobFS) Exists(ctx context.Context, filepath string) (bool, error) {
	dir, filename := path.Split(filepath)
	bucket, err := fs.openBucket(ctx, path.Join(fs.prefix, dir))
	if err != nil {
		return false, err
	}
	defer bucket.Close()

	return bucket.Exists(context.TODO(), filename)
}

func (fs *BlobFS) SignedURL(ctx context.Context, filepath string, opts *blob.SignedURLOptions) (string, error) {
	dir, filename := path.Split(filepath)
	bucket, err := fs.openBucket(ctx, path.Join(fs.prefix, dir))
	if err != nil {
		return "", err
	}
	defer bucket.Close()

	return bucket.SignedURL(ctx, filename, opts)
}

func (fs *BlobFS) openBucket(ctx context.Context, dir string) (*blob.Bucket, error) {
	bucket, err := blob.OpenBucket(ctx, fs.storageURL)
	if err != nil {
		return nil, err
	}
	return blob.PrefixedBucket(bucket, strings.Trim(dir, "/")+"/"), nil
}

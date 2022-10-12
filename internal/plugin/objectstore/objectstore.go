package objectstore

//go:generate mockery --name ObjectStore --inpackage --case underscore

import (
	"context"
	"io"
)

// DownloadOptions provides options for downloading an object from the object store
type DownloadOptions struct {
	ContentRange *string
}

// ObjectStore interface
type ObjectStore interface {
	UploadObject(ctx context.Context, key string, body io.Reader) error
	DownloadObject(ctx context.Context, key string, w io.WriterAt, option *DownloadOptions) error
	GetObjectStream(ctx context.Context, key string, options *DownloadOptions) (io.ReadCloser, error)
	GetPresignedURL(ctx context.Context, key string) (string, error)
	DoesObjectExist(ctx context.Context, key string) (bool, error)
}

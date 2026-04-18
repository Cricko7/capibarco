// Package storage contains object storage adapters.
package storage

import (
	"context"
	"fmt"
	"io"
	"net/url"
	"path"
	"strings"

	"github.com/google/uuid"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"github.com/petmatch/petmatch/internal/config"
)

// Uploader stores uploaded objects and returns public URLs.
type Uploader interface {
	Upload(ctx context.Context, objectName string, reader io.Reader, size int64, contentType string) (string, error)
}

// NoopUploader rejects uploads when object storage is disabled.
type NoopUploader struct{}

func (NoopUploader) Upload(context.Context, string, io.Reader, int64, string) (string, error) {
	return "", fmt.Errorf("object storage is disabled")
}

// S3Uploader stores files in S3-compatible object storage.
type S3Uploader struct {
	client    *minio.Client
	bucket    string
	publicURL string
}

// NewS3Uploader creates an S3-compatible uploader.
func NewS3Uploader(cfg config.S3Config) (*S3Uploader, error) {
	client, err := minio.New(cfg.Endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(cfg.AccessKey, cfg.SecretKey, ""),
		Secure: cfg.UseSSL,
	})
	if err != nil {
		return nil, fmt.Errorf("create minio client: %w", err)
	}
	return &S3Uploader{client: client, bucket: cfg.Bucket, publicURL: strings.TrimRight(cfg.PublicURL, "/")}, nil
}

// Upload uploads objectName and returns its public URL.
func (u *S3Uploader) Upload(ctx context.Context, objectName string, reader io.Reader, size int64, contentType string) (string, error) {
	if objectName == "" {
		objectName = uuid.NewString()
	}
	_, err := u.client.PutObject(ctx, u.bucket, objectName, reader, size, minio.PutObjectOptions{ContentType: contentType})
	if err != nil {
		return "", fmt.Errorf("put object: %w", err)
	}
	if u.publicURL != "" {
		return u.publicURL + "/" + url.PathEscape(objectName), nil
	}
	return (&url.URL{Scheme: "s3", Host: u.bucket, Path: path.Clean("/" + objectName)}).String(), nil
}

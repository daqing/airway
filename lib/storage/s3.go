package storage

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

// S3 stores files in a cloud object storage service via the S3 protocol
// (Amazon S3, Cloudflare R2, Tencent COS). HTTPS is always used.
type S3 struct {
	client    *minio.Client
	bucket    string
	endpoint  string
	region    string
	publicURL string
}

func NewS3(cfg Config) (*S3, error) {
	if cfg.Bucket == "" {
		return nil, fmt.Errorf("STORAGE_BUCKET must be set")
	}

	if cfg.AccessKey == "" || cfg.SecretKey == "" {
		return nil, fmt.Errorf("STORAGE_ACCESS_KEY and STORAGE_SECRET_KEY must be set")
	}

	endpoint, region, err := cloudEndpoint(cfg)
	if err != nil {
		return nil, err
	}

	client, err := minio.New(endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(cfg.AccessKey, cfg.SecretKey, ""),
		Secure: true,
		Region: region,
	})
	if err != nil {
		return nil, err
	}

	return &S3{
		client:    client,
		bucket:    cfg.Bucket,
		endpoint:  endpoint,
		region:    region,
		publicURL: strings.TrimRight(cfg.PublicURL, "/"),
	}, nil
}

// cloudEndpoint resolves the endpoint and signing region for a cloud driver.
// STORAGE_ENDPOINT overrides the derived endpoint when set.
func cloudEndpoint(cfg Config) (endpoint, region string, err error) {
	region = cfg.Region

	switch cfg.Driver {
	case DriverS3:
		if region == "" {
			return "", "", fmt.Errorf("STORAGE_REGION must be set for driver %q", cfg.Driver)
		}

		endpoint = "s3." + region + ".amazonaws.com"
	case DriverCOS:
		if region == "" {
			return "", "", fmt.Errorf("STORAGE_REGION must be set for driver %q", cfg.Driver)
		}

		endpoint = "cos." + region + ".myqcloud.com"
	case DriverR2:
		if region == "" {
			region = "auto"
		}
		// R2 endpoints carry the account ID and cannot be derived.
	default:
		return "", "", fmt.Errorf("unknown storage driver: %q", cfg.Driver)
	}

	if cfg.Endpoint != "" {
		endpoint = cfg.Endpoint
	}

	if endpoint == "" {
		return "", "", fmt.Errorf("STORAGE_ENDPOINT must be set for driver %q", cfg.Driver)
	}

	return endpoint, region, nil
}

func (s *S3) Put(ctx context.Context, key string, r io.Reader, size int64, contentType string) error {
	if contentType == "" {
		contentType = "application/octet-stream"
	}

	_, err := s.client.PutObject(ctx, s.bucket, key, r, size, minio.PutObjectOptions{
		ContentType: contentType,
	})

	return err
}

func (s *S3) Get(ctx context.Context, key string) (io.ReadCloser, error) {
	return s.client.GetObject(ctx, s.bucket, key, minio.GetObjectOptions{})
}

func (s *S3) Delete(ctx context.Context, key string) error {
	return s.client.RemoveObject(ctx, s.bucket, key, minio.RemoveObjectOptions{})
}

func (s *S3) Exists(ctx context.Context, key string) (bool, error) {
	_, err := s.client.StatObject(ctx, s.bucket, key, minio.StatObjectOptions{})
	if err == nil {
		return true, nil
	}

	var resp minio.ErrorResponse
	if errors.As(err, &resp) && resp.Code == "NoSuchKey" {
		return false, nil
	}

	return false, err
}

func (s *S3) URL(ctx context.Context, key string, expires time.Duration) (string, error) {
	if s.publicURL != "" {
		return s.publicURL + "/" + key, nil
	}

	u, err := s.client.PresignedGetObject(ctx, s.bucket, key, expires, nil)
	if err != nil {
		return "", err
	}

	return u.String(), nil
}

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

// Cloud stores files in a cloud object storage service via the S3 protocol
// (Amazon S3, Cloudflare R2, Tencent COS). HTTPS is always used.
type Cloud struct {
	client    *minio.Client
	bucket    string
	endpoint  string
	region    string
	publicURL string
	lookup    minio.BucketLookupType
}

func NewCloud(cfg Config) (*Cloud, error) {
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

	lookup := minio.BucketLookupAuto
	if cfg.Driver == DriverCOS {
		// Tencent COS requires virtual-hosted style:
		// https://<bucket>.cos.<region>.myqcloud.com/<key>.
		lookup = minio.BucketLookupDNS
	}

	client, err := minio.New(endpoint, &minio.Options{
		Creds:        credentials.NewStaticV4(cfg.AccessKey, cfg.SecretKey, ""),
		Secure:       true,
		Region:       region,
		BucketLookup: lookup,
	})
	if err != nil {
		return nil, err
	}

	return &Cloud{
		client:    client,
		bucket:    cfg.Bucket,
		endpoint:  endpoint,
		region:    region,
		publicURL: strings.TrimRight(cfg.PublicURL, "/"),
		lookup:    lookup,
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
	if cfg.Driver == DriverR2 && isR2AccountID(endpoint) {
		endpoint += ".r2.cloudflarestorage.com"
	}

	if endpoint == "" {
		return "", "", fmt.Errorf("STORAGE_ENDPOINT must be set for driver %q", cfg.Driver)
	}

	return endpoint, region, nil
}

func isR2AccountID(value string) bool {
	if len(value) != 32 {
		return false
	}

	for _, char := range value {
		if (char < '0' || char > '9') && (char < 'a' || char > 'f') && (char < 'A' || char > 'F') {
			return false
		}
	}

	return true
}

func (s *Cloud) Put(ctx context.Context, key string, obj Object) error {
	if err := obj.validate(); err != nil {
		return err
	}

	_, err := s.client.PutObject(ctx, s.bucket, key, obj.Reader, obj.Size, minio.PutObjectOptions{
		ContentType: obj.ContentType,
	})

	return err
}

func (s *Cloud) Get(ctx context.Context, key string) (io.ReadCloser, error) {
	return s.client.GetObject(ctx, s.bucket, key, minio.GetObjectOptions{})
}

func (s *Cloud) Delete(ctx context.Context, key string) error {
	return s.client.RemoveObject(ctx, s.bucket, key, minio.RemoveObjectOptions{})
}

func (s *Cloud) Exists(ctx context.Context, key string) (bool, error) {
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

func (s *Cloud) URL(ctx context.Context, key string, expires time.Duration) (string, error) {
	if s.publicURL != "" {
		return s.publicURL + "/" + key, nil
	}

	u, err := s.client.PresignedGetObject(ctx, s.bucket, key, expires, nil)
	if err != nil {
		return "", err
	}

	return u.String(), nil
}

// Package storage provides a unified file storage layer.
//
// The backend is selected by configuration: local directory (driver "local")
// or any S3-compatible object storage (driver "s3"), such as Amazon S3,
// Tencent COS, Cloudflare R2 or MinIO.
package storage

import (
	"context"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"time"
)

// Driver names.
const (
	DriverLocal = "local"
	DriverS3    = "s3"
)

// DefaultRoot is used when STORAGE_ROOT is not set.
const DefaultRoot = "./data/storage"

type Storage interface {
	// Put stores the content read from r under key. size is the number of
	// bytes to store; contentType is advisory (used by the S3 backend).
	Put(ctx context.Context, key string, r io.Reader, size int64, contentType string) error
	// Get returns the content stored under key. The caller must close it.
	Get(ctx context.Context, key string) (io.ReadCloser, error)
	Delete(ctx context.Context, key string) error
	Exists(ctx context.Context, key string) (bool, error)
	// URL returns an address the file can be downloaded from. For the local
	// backend it is the app's own download path; for S3 it is either
	// PublicURL+key or a presigned URL valid for expires.
	URL(ctx context.Context, key string, expires time.Duration) (string, error)
}

type Config struct {
	Driver string // DriverLocal (default) or DriverS3
	Root   string // local only, defaults to DefaultRoot

	// s3 only
	Endpoint  string // e.g. s3.us-east-1.amazonaws.com, cos.ap-guangzhou.myqcloud.com, <account>.r2.cloudflarestorage.com
	Region    string
	Bucket    string
	AccessKey string
	SecretKey string
	UseSSL    bool
	PathStyle bool   // required by MinIO and some S3-compatible services
	PublicURL string // optional CDN base URL; disables presigning when set
}

// Open validates cfg and constructs the backend selected by cfg.Driver.
func Open(cfg Config) (Storage, error) {
	switch cfg.Driver {
	case "", DriverLocal:
		root := cfg.Root
		if root == "" {
			root = DefaultRoot
		}

		return NewLocal(root)
	case DriverS3:
		return NewS3(cfg)
	default:
		return nil, fmt.Errorf("unknown storage driver: %q", cfg.Driver)
	}
}

// FromEnv builds a Config from environment variables:
//
//	STORAGE_DRIVER, STORAGE_ROOT,
//	S3_ENDPOINT, S3_REGION, S3_BUCKET, S3_ACCESS_KEY, S3_SECRET_KEY,
//	S3_USE_SSL, S3_PATH_STYLE, S3_PUBLIC_URL
func FromEnv() Config {
	return Config{
		Driver:    envOr("STORAGE_DRIVER", DriverLocal),
		Root:      envOr("STORAGE_ROOT", DefaultRoot),
		Endpoint:  os.Getenv("S3_ENDPOINT"),
		Region:    os.Getenv("S3_REGION"),
		Bucket:    os.Getenv("S3_BUCKET"),
		AccessKey: os.Getenv("S3_ACCESS_KEY"),
		SecretKey: os.Getenv("S3_SECRET_KEY"),
		UseSSL:    envBool("S3_USE_SSL", true),
		PathStyle: envBool("S3_PATH_STYLE", false),
		PublicURL: strings.TrimRight(os.Getenv("S3_PUBLIC_URL"), "/"),
	}
}

var __storage__ Storage

// Setup constructs the backend from cfg and installs it as the default instance.
func Setup(cfg Config) (Storage, error) {
	s, err := Open(cfg)
	if err != nil {
		return nil, err
	}

	__storage__ = s
	return s, nil
}

// Current returns the default instance. It panics if Setup was never called.
func Current() Storage {
	if __storage__ == nil {
		panic("storage is not setup")
	}

	return __storage__
}

func envOr(key, fallback string) string {
	if v := strings.TrimSpace(os.Getenv(key)); v != "" {
		return v
	}

	return fallback
}

func envBool(key string, fallback bool) bool {
	v := strings.TrimSpace(os.Getenv(key))
	if v == "" {
		return fallback
	}

	parsed, err := strconv.ParseBool(v)
	if err != nil {
		return fallback
	}

	return parsed
}

// Package storage provides a unified file storage layer.
//
// The backend is selected by configuration: local directory (driver "local")
// or a cloud object storage service: Amazon S3 (driver "s3"),
// Cloudflare R2 (driver "r2") or Tencent COS (driver "cos").
package storage

import (
	"context"
	"fmt"
	"io"
	"os"
	"strings"
	"time"
)

// Driver names.
const (
	DriverLocal = "local"
	DriverS3    = "s3"
	DriverR2    = "r2"
	DriverCOS   = "cos"
)

// DefaultRoot is used when STORAGE_ROOT is not set.
const DefaultRoot = "./data/storage"

type Storage interface {
	// Put stores the content read from r under key. size is the number of
	// bytes to store; contentType is advisory (used by cloud backends).
	Put(ctx context.Context, key string, r io.Reader, size int64, contentType string) error
	// Get returns the content stored under key. The caller must close it.
	Get(ctx context.Context, key string) (io.ReadCloser, error)
	Delete(ctx context.Context, key string) error
	Exists(ctx context.Context, key string) (bool, error)
	// URL returns an address the file can be downloaded from. For the local
	// backend it is the app's own download path; for cloud backends it is
	// either PublicURL+key or a presigned URL valid for expires.
	URL(ctx context.Context, key string, expires time.Duration) (string, error)
}

type Config struct {
	Driver string // DriverLocal (default), DriverS3, DriverR2 or DriverCOS
	Root   string // local only, defaults to DefaultRoot

	// cloud drivers only
	Endpoint  string // optional override; derived from Region for s3/cos, required for r2
	Region    string // required for s3/cos; r2 defaults to "auto"
	Bucket    string
	AccessKey string
	SecretKey string
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
	case DriverS3, DriverR2, DriverCOS:
		return NewCloud(cfg)
	default:
		return nil, fmt.Errorf("unknown storage driver: %q", cfg.Driver)
	}
}

// FromEnv builds a Config from environment variables:
//
//	STORAGE_DRIVER, STORAGE_ROOT,
//	STORAGE_ENDPOINT, STORAGE_REGION, STORAGE_BUCKET,
//	STORAGE_ACCESS_KEY, STORAGE_SECRET_KEY, STORAGE_PUBLIC_URL
func FromEnv() Config {
	return Config{
		Driver:    envOr("STORAGE_DRIVER", DriverLocal),
		Root:      envOr("STORAGE_ROOT", DefaultRoot),
		Endpoint:  os.Getenv("STORAGE_ENDPOINT"),
		Region:    os.Getenv("STORAGE_REGION"),
		Bucket:    os.Getenv("STORAGE_BUCKET"),
		AccessKey: os.Getenv("STORAGE_ACCESS_KEY"),
		SecretKey: os.Getenv("STORAGE_SECRET_KEY"),
		PublicURL: strings.TrimRight(os.Getenv("STORAGE_PUBLIC_URL"), "/"),
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

package storage

import (
	"context"
	"net/url"
	"testing"
	"time"

	"github.com/minio/minio-go/v7"
)

func clearStorageEnv(t *testing.T) {
	t.Helper()

	for _, key := range []string{
		"STORAGE_DRIVER", "STORAGE_ROOT",
		"STORAGE_ENDPOINT", "STORAGE_REGION", "STORAGE_BUCKET",
		"STORAGE_ACCESS_KEY", "STORAGE_SECRET_KEY", "STORAGE_PUBLIC_URL",
	} {
		t.Setenv(key, "")
	}
}

func TestFromEnvDefaults(t *testing.T) {
	clearStorageEnv(t)

	cfg := FromEnv()

	if cfg.Driver != DriverLocal {
		t.Fatalf("expected default driver %q, got %q", DriverLocal, cfg.Driver)
	}

	if cfg.Root != DefaultRoot {
		t.Fatalf("expected default root %q, got %q", DefaultRoot, cfg.Root)
	}
}

func TestFromEnvCloud(t *testing.T) {
	clearStorageEnv(t)

	t.Setenv("STORAGE_DRIVER", "cos")
	t.Setenv("STORAGE_REGION", "ap-guangzhou")
	t.Setenv("STORAGE_BUCKET", "my-app-1234567890")
	t.Setenv("STORAGE_ACCESS_KEY", "ak")
	t.Setenv("STORAGE_SECRET_KEY", "sk")
	t.Setenv("STORAGE_PUBLIC_URL", "https://cdn.example.com/")

	cfg := FromEnv()

	if cfg.Driver != DriverCOS {
		t.Fatalf("expected driver %q, got %q", DriverCOS, cfg.Driver)
	}

	if cfg.Region != "ap-guangzhou" || cfg.Bucket != "my-app-1234567890" {
		t.Fatalf("unexpected region/bucket: %#v", cfg)
	}

	if cfg.AccessKey != "ak" || cfg.SecretKey != "sk" {
		t.Fatalf("unexpected credentials: %#v", cfg)
	}

	if cfg.PublicURL != "https://cdn.example.com" {
		t.Fatalf("expected PublicURL without trailing slash, got %q", cfg.PublicURL)
	}
}

func TestOpenUnknownDriver(t *testing.T) {
	if _, err := Open(Config{Driver: "ftp"}); err == nil {
		t.Fatalf("expected error for unknown driver")
	}
}

func cloudConfig(driver string) Config {
	return Config{
		Driver:    driver,
		Bucket:    "my-bucket",
		AccessKey: "ak",
		SecretKey: "sk",
	}
}

func TestS3EndpointDerivedFromRegion(t *testing.T) {
	cfg := cloudConfig(DriverS3)
	cfg.Region = "us-east-1"

	s, err := NewCloud(cfg)
	if err != nil {
		t.Fatalf("NewCloud: %v", err)
	}

	if s.endpoint != "s3.us-east-1.amazonaws.com" || s.region != "us-east-1" {
		t.Fatalf("unexpected endpoint/region: %q %q", s.endpoint, s.region)
	}
}

func TestCOSEndpointDerivedFromRegion(t *testing.T) {
	cfg := cloudConfig(DriverCOS)
	cfg.Region = "ap-guangzhou"

	s, err := NewCloud(cfg)
	if err != nil {
		t.Fatalf("NewCloud: %v", err)
	}

	if s.endpoint != "cos.ap-guangzhou.myqcloud.com" || s.region != "ap-guangzhou" {
		t.Fatalf("unexpected endpoint/region: %q %q", s.endpoint, s.region)
	}
	if s.lookup != minio.BucketLookupDNS {
		t.Fatalf("COS must use virtual-hosted bucket lookup, got %v", s.lookup)
	}

	presigned, err := s.URL(context.Background(), "images/foo.png", time.Hour)
	if err != nil {
		t.Fatalf("presign COS URL: %v", err)
	}
	u, err := url.Parse(presigned)
	if err != nil {
		t.Fatalf("parse COS URL: %v", err)
	}
	if u.Host != "my-bucket.cos.ap-guangzhou.myqcloud.com" {
		t.Fatalf("expected virtual-hosted COS URL, got %q", u.Host)
	}
}

func TestNonCOSDriversKeepAutomaticBucketLookup(t *testing.T) {
	for _, cfg := range []Config{
		{Driver: DriverS3, Region: "us-east-1", Bucket: "bucket", AccessKey: "ak", SecretKey: "sk"},
		{Driver: DriverR2, Endpoint: "account.r2.cloudflarestorage.com", Bucket: "bucket", AccessKey: "ak", SecretKey: "sk"},
	} {
		s, err := NewCloud(cfg)
		if err != nil {
			t.Fatalf("NewCloud(%s): %v", cfg.Driver, err)
		}
		if s.lookup != minio.BucketLookupAuto {
			t.Fatalf("driver %s should keep automatic bucket lookup, got %v", cfg.Driver, s.lookup)
		}
	}
}

func TestR2RequiresEndpointAndDefaultsRegion(t *testing.T) {
	if _, err := NewCloud(cloudConfig(DriverR2)); err == nil {
		t.Fatalf("expected error when r2 endpoint is missing")
	}

	cfg := cloudConfig(DriverR2)
	cfg.Endpoint = "abc123.r2.cloudflarestorage.com"

	s, err := NewCloud(cfg)
	if err != nil {
		t.Fatalf("NewCloud: %v", err)
	}

	if s.endpoint != "abc123.r2.cloudflarestorage.com" || s.region != "auto" {
		t.Fatalf("unexpected endpoint/region: %q %q", s.endpoint, s.region)
	}
}

func TestR2ExpandsAccountIDEndpoint(t *testing.T) {
	cfg := cloudConfig(DriverR2)
	cfg.Endpoint = "63fceb43b45f02a0fcdabd3212b7f755"

	s, err := NewCloud(cfg)
	if err != nil {
		t.Fatalf("NewCloud: %v", err)
	}
	if s.endpoint != "63fceb43b45f02a0fcdabd3212b7f755.r2.cloudflarestorage.com" {
		t.Fatalf("unexpected R2 endpoint: %q", s.endpoint)
	}
}

func TestR2PreservesCustomEndpoint(t *testing.T) {
	cfg := cloudConfig(DriverR2)
	cfg.Endpoint = "localhost:9000"

	s, err := NewCloud(cfg)
	if err != nil {
		t.Fatalf("NewCloud: %v", err)
	}
	if s.endpoint != "localhost:9000" {
		t.Fatalf("unexpected custom endpoint: %q", s.endpoint)
	}
}

func TestEndpointOverride(t *testing.T) {
	cfg := cloudConfig(DriverS3)
	cfg.Region = "us-east-1"
	cfg.Endpoint = "s3.custom.example.com"

	s, err := NewCloud(cfg)
	if err != nil {
		t.Fatalf("NewCloud: %v", err)
	}

	if s.endpoint != "s3.custom.example.com" {
		t.Fatalf("expected endpoint override, got %q", s.endpoint)
	}
}

func TestCloudRegionRequired(t *testing.T) {
	for _, driver := range []string{DriverS3, DriverCOS} {
		if _, err := NewCloud(cloudConfig(driver)); err == nil {
			t.Fatalf("expected error when region is missing for driver %q", driver)
		}
	}
}

func TestCloudCredentialsRequired(t *testing.T) {
	cases := []Config{
		{Driver: DriverS3, Region: "us-east-1"},
		{Driver: DriverS3, Region: "us-east-1", Bucket: "b"},
		{Driver: DriverS3, Region: "us-east-1", Bucket: "b", AccessKey: "ak"},
	}

	for i, cfg := range cases {
		if _, err := NewCloud(cfg); err == nil {
			t.Fatalf("case %d: expected error for incomplete cloud config", i)
		}
	}
}

func TestSetupAndCurrent(t *testing.T) {
	if _, err := Setup(Config{Driver: DriverLocal, Root: t.TempDir()}); err != nil {
		t.Fatalf("Setup: %v", err)
	}

	if Current() == nil {
		t.Fatalf("expected Current to return the setup instance")
	}
}

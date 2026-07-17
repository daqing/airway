package storage

import (
	"testing"
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

	s, err := NewS3(cfg)
	if err != nil {
		t.Fatalf("NewS3: %v", err)
	}

	if s.endpoint != "s3.us-east-1.amazonaws.com" || s.region != "us-east-1" {
		t.Fatalf("unexpected endpoint/region: %q %q", s.endpoint, s.region)
	}
}

func TestCOSEndpointDerivedFromRegion(t *testing.T) {
	cfg := cloudConfig(DriverCOS)
	cfg.Region = "ap-guangzhou"

	s, err := NewS3(cfg)
	if err != nil {
		t.Fatalf("NewS3: %v", err)
	}

	if s.endpoint != "cos.ap-guangzhou.myqcloud.com" || s.region != "ap-guangzhou" {
		t.Fatalf("unexpected endpoint/region: %q %q", s.endpoint, s.region)
	}
}

func TestR2RequiresEndpointAndDefaultsRegion(t *testing.T) {
	if _, err := NewS3(cloudConfig(DriverR2)); err == nil {
		t.Fatalf("expected error when r2 endpoint is missing")
	}

	cfg := cloudConfig(DriverR2)
	cfg.Endpoint = "abc123.r2.cloudflarestorage.com"

	s, err := NewS3(cfg)
	if err != nil {
		t.Fatalf("NewS3: %v", err)
	}

	if s.endpoint != "abc123.r2.cloudflarestorage.com" || s.region != "auto" {
		t.Fatalf("unexpected endpoint/region: %q %q", s.endpoint, s.region)
	}
}

func TestEndpointOverride(t *testing.T) {
	cfg := cloudConfig(DriverS3)
	cfg.Region = "us-east-1"
	cfg.Endpoint = "s3.custom.example.com"

	s, err := NewS3(cfg)
	if err != nil {
		t.Fatalf("NewS3: %v", err)
	}

	if s.endpoint != "s3.custom.example.com" {
		t.Fatalf("expected endpoint override, got %q", s.endpoint)
	}
}

func TestCloudRegionRequired(t *testing.T) {
	for _, driver := range []string{DriverS3, DriverCOS} {
		if _, err := NewS3(cloudConfig(driver)); err == nil {
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
		if _, err := NewS3(cfg); err == nil {
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

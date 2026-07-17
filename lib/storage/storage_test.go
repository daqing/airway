package storage

import (
	"testing"
)

func clearStorageEnv(t *testing.T) {
	t.Helper()

	for _, key := range []string{
		"STORAGE_DRIVER", "STORAGE_ROOT",
		"S3_ENDPOINT", "S3_REGION", "S3_BUCKET", "S3_ACCESS_KEY", "S3_SECRET_KEY",
		"S3_USE_SSL", "S3_PATH_STYLE", "S3_PUBLIC_URL",
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

	if !cfg.UseSSL {
		t.Fatalf("expected UseSSL to default to true")
	}

	if cfg.PathStyle {
		t.Fatalf("expected PathStyle to default to false")
	}
}

func TestFromEnvS3(t *testing.T) {
	clearStorageEnv(t)

	t.Setenv("STORAGE_DRIVER", "s3")
	t.Setenv("S3_ENDPOINT", "cos.ap-guangzhou.myqcloud.com")
	t.Setenv("S3_REGION", "ap-guangzhou")
	t.Setenv("S3_BUCKET", "my-bucket")
	t.Setenv("S3_ACCESS_KEY", "ak")
	t.Setenv("S3_SECRET_KEY", "sk")
	t.Setenv("S3_USE_SSL", "false")
	t.Setenv("S3_PATH_STYLE", "true")
	t.Setenv("S3_PUBLIC_URL", "https://cdn.example.com/")

	cfg := FromEnv()

	if cfg.Driver != DriverS3 {
		t.Fatalf("expected driver %q, got %q", DriverS3, cfg.Driver)
	}

	if cfg.Endpoint != "cos.ap-guangzhou.myqcloud.com" || cfg.Region != "ap-guangzhou" {
		t.Fatalf("unexpected endpoint/region: %q %q", cfg.Endpoint, cfg.Region)
	}

	if cfg.Bucket != "my-bucket" || cfg.AccessKey != "ak" || cfg.SecretKey != "sk" {
		t.Fatalf("unexpected bucket/credentials: %#v", cfg)
	}

	if cfg.UseSSL {
		t.Fatalf("expected UseSSL=false")
	}

	if !cfg.PathStyle {
		t.Fatalf("expected PathStyle=true")
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

func TestNewS3MissingFields(t *testing.T) {
	cases := []Config{
		{},                           // no endpoint
		{Endpoint: "e"},              // no bucket
		{Endpoint: "e", Bucket: "b"}, // no credentials
		{Endpoint: "e", Bucket: "b", AccessKey: "ak"}, // no secret
	}

	for i, cfg := range cases {
		if _, err := NewS3(cfg); err == nil {
			t.Fatalf("case %d: expected error for incomplete S3 config", i)
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

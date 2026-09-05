package utils

import (
	"testing"
)

func TestURLPrefixReadsEnv(t *testing.T) {
	t.Setenv("AIRWAY_URL_PREFIX", "")
	t.Setenv("URL_PREFIX", "")
	if got := URLPrefix(); got != "" {
		t.Fatalf("expected empty prefix when unset, got %q", got)
	}

	t.Setenv("URL_PREFIX", "/airway")
	if got := URLPrefix(); got != "/airway" {
		t.Fatalf("expected /airway, got %q", got)
	}

	t.Setenv("AIRWAY_URL_PREFIX", "/api")
	t.Setenv("URL_PREFIX", "/airway")
	if got := URLPrefix(); got != "/api" {
		t.Fatalf("expected AIRWAY_URL_PREFIX to win, got %q", got)
	}
}

func TestURLPrefixIgnoresOtherEnvVars(t *testing.T) {
	t.Setenv("URL_PREFIX", "")
	if got := URLPrefix(); got != "" {
		t.Fatalf("expected empty prefix, got %q", got)
	}
}

func TestNormalizeURLPrefix(t *testing.T) {
	cases := map[string]string{
		"":           "",
		"   ":        "",
		"/":          "",
		"//":         "",
		"airway":     "/airway",
		"/airway":    "/airway",
		"airway/":    "/airway",
		"/airway/":   "/airway",
		" airway ":   "/airway",
		"airway/api": "/airway/api",
	}

	for input, want := range cases {
		if got := NormalizeURLPrefix(input); got != want {
			t.Fatalf("NormalizeURLPrefix(%q) = %q, want %q", input, got, want)
		}
	}
}

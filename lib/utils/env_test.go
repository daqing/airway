package utils

import (
	"os"
	"testing"
)

// withShellEnv replaces the startup-environment snapshot for the duration of
// a test; the os.Setenv layer below it stands in for values loaded from .env.
func withShellEnv(t *testing.T, shell map[string]string) {
	t.Helper()

	original := shellEnv
	shellEnv = shell
	t.Cleanup(func() { shellEnv = original })
}

func TestGetEnvOr(t *testing.T) {
	withShellEnv(t, nil)
	clearEnv("AIRWAY_PORT", t)
	clearEnv("PORT", t)

	setEnv("PORT", "9527", t)

	if GetEnvOr("AIRWAY_PORT", "PORT") != "9527" {
		t.Fail()
	}

	setEnv("AIRWAY_PORT", "3180", t)
	if GetEnvOr("AIRWAY_PORT", "PORT") != "3180" {
		t.Fail()
	}
}

// A PORT passed through the environment must win over an AIRWAY_PORT that
// only exists in .env: the startup environment is consulted before any
// loaded value, per key.
func TestShellEnvWinsOverLoadedEnv(t *testing.T) {
	withShellEnv(t, map[string]string{"PORT": "1988"})
	clearEnv("PORT", t)
	t.Setenv("AIRWAY_PORT", "1900")

	if GetEnvOr("AIRWAY_PORT", "PORT") != "1988" {
		t.Fatalf("expected shell PORT to win over loaded AIRWAY_PORT, got %q", GetEnvOr("AIRWAY_PORT", "PORT"))
	}

	if GetEnvMulti("AIRWAY_PORT", "PORT") != "1988" {
		t.Fatalf("expected shell PORT to win over loaded AIRWAY_PORT, got %q", GetEnvMulti("AIRWAY_PORT", "PORT"))
	}
}

func TestShellEnvKeyOrderStillApplies(t *testing.T) {
	withShellEnv(t, map[string]string{"AIRWAY_PORT": "3180", "PORT": "1988"})
	t.Setenv("AIRWAY_PORT", "1900")

	if GetEnvOr("AIRWAY_PORT", "PORT") != "3180" {
		t.Fatalf("expected AIRWAY_PORT first within the shell layer, got %q", GetEnvOr("AIRWAY_PORT", "PORT"))
	}
}

func TestLoadedEnvFallsBackInKeyOrder(t *testing.T) {
	withShellEnv(t, nil)
	t.Setenv("AIRWAY_PORT", "3180")
	t.Setenv("PORT", "1988")

	if GetEnvOr("AIRWAY_PORT", "PORT") != "3180" {
		t.Fatalf("expected AIRWAY_PORT first within the loaded layer, got %q", GetEnvOr("AIRWAY_PORT", "PORT"))
	}
}

func TestEmptyShellValueFallsThroughToLoadedEnv(t *testing.T) {
	withShellEnv(t, map[string]string{"PORT": "   "})
	clearEnv("AIRWAY_PORT", t)
	t.Setenv("PORT", "9527")

	if GetEnvOr("AIRWAY_PORT", "PORT") != "9527" {
		t.Fatalf("expected blank shell value to fall through, got %q", GetEnvOr("AIRWAY_PORT", "PORT"))
	}
}

func TestGetEnvMultiReturnsEmptyWhenUnset(t *testing.T) {
	withShellEnv(t, nil)
	clearEnv("AIRWAY_DSN", t)
	clearEnv("DSN", t)

	if GetEnvMulti("AIRWAY_DSN", "DSN") != "" {
		t.Fatalf("expected empty value, got %q", GetEnvMulti("AIRWAY_DSN", "DSN"))
	}
}

func TestURLPrefixPrefersShellOverLoadedEnv(t *testing.T) {
	withShellEnv(t, map[string]string{"URL_PREFIX": "/shell"})
	t.Setenv("AIRWAY_URL_PREFIX", "/loaded")

	if URLPrefix() != "/shell" {
		t.Fatalf("expected shell URL_PREFIX to win, got %q", URLPrefix())
	}

	withShellEnv(t, nil)
	if URLPrefix() != "/loaded" {
		t.Fatalf("expected loaded AIRWAY_URL_PREFIX fallback, got %q", URLPrefix())
	}
}

func TestGetEnvMissingKeyErrors(t *testing.T) {
	withShellEnv(t, nil)
	clearEnv("NOT_SET_ANYWHERE", t)

	if _, err := GetEnv("NOT_SET_ANYWHERE"); err == nil {
		t.Fatal("expected an error for a missing key")
	}
}

func clearEnv(key string, t *testing.T) {
	err := os.Unsetenv(key)
	if err != nil {
		t.Fail()
	}
}

func setEnv(key string, val string, t *testing.T) {
	err := os.Setenv(key, val)
	if err != nil {
		t.Fail()
	}
}

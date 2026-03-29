package utils

import "testing"

func TestAppConfigReadsEnvironmentDynamically(t *testing.T) {
	t.Setenv("AIRWAY_ENV", "")

	config := AppConfig()
	if config.Env != "" {
		t.Fatalf("expected empty env, got %q", config.Env)
	}

	t.Setenv("AIRWAY_ENV", LOCAL_ENV)

	config = AppConfig()
	if config.Env != LOCAL_ENV {
		t.Fatalf("expected env %q, got %q", LOCAL_ENV, config.Env)
	}

	if !config.IsLocal {
		t.Fatal("expected local env to be detected dynamically")
	}
}

package utils

import (
	"os"
	"testing"
)

func TestGetEnvOr(t *testing.T) {
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

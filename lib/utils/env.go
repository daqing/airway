package utils

import (
	"fmt"
	"os"
	"strings"
)

const EMPTY_STRING = ""

// shellEnv snapshots the process environment as it was at startup, before
// any .env file is loaded into it. Multi-key lookups consult it first, so
// values passed through the environment (shell, container, process manager)
// always take priority over the same keys supplied by .env. Package
// initialization runs before main() loads .env, which keeps the snapshot
// authoritative; tests may replace it to simulate an outer environment.
var shellEnv = snapshotEnv()

func snapshotEnv() map[string]string {
	env := make(map[string]string)
	for _, entry := range os.Environ() {
		key, value, _ := strings.Cut(entry, "=")
		env[key] = value
	}
	return env
}

// getEnvFrom returns the first non-empty value among keys, consulting the
// startup environment first and the current environment (which includes
// values loaded from .env) second.
func getEnvFrom(keys []string) (string, bool) {
	for _, key := range keys {
		if value, ok := shellEnv[key]; ok {
			if trimmed := TrimFull(value); trimmed != "" {
				return trimmed, true
			}
		}
	}

	for _, key := range keys {
		if value := TrimFull(os.Getenv(key)); value != "" {
			return value, true
		}
	}

	return "", false
}

func GetEnv(key string) (string, error) {
	v := TrimFull(os.Getenv(key))
	if v == EMPTY_STRING {
		return EMPTY_STRING, fmt.Errorf("%s must be set", key)
	}

	return v, nil
}

func GetEnvMust(key string) string {
	val, err := GetEnv(key)
	if err != nil {
		panic(err)
	}

	return val
}

func GetEnvMulti(keys ...string) string {
	value, _ := getEnvFrom(keys)
	return value
}

func GetEnvOr(firstKey string, secondKey string) string {
	value, ok := getEnvFrom([]string{firstKey, secondKey})
	if !ok {
		panic(fmt.Errorf("%s must be set", secondKey))
	}

	return value
}

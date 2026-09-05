package utils

import (
	"os"
	"strings"
)

// URLPrefix returns the configured public sub-path prefix under which the app
// is served, normalized to "/airway" form, or "" when the app is served at the
// root. It reads AIRWAY_URL_PREFIX first and falls back to URL_PREFIX.
func URLPrefix() string {
	value := TrimFull(os.Getenv("AIRWAY_URL_PREFIX"))
	if value == "" {
		value = TrimFull(os.Getenv("URL_PREFIX"))
	}

	return NormalizeURLPrefix(value)
}

// NormalizeURLPrefix normalizes a sub-path prefix value to "/airway" form.
// Empty values, "/", and "//" collapse to "".
func NormalizeURLPrefix(value string) string {
	value = strings.TrimSpace(value)
	value = strings.Trim(value, "/")
	if value == "" {
		return ""
	}

	return "/" + value
}

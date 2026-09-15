package main

import (
	"os"
	"strings"
	"testing"
)

func TestVersionMatchesVERSIONFile(t *testing.T) {
	data, err := os.ReadFile("VERSION")
	if err != nil {
		t.Fatalf("read VERSION: %v", err)
	}

	if strings.TrimSpace(string(data)) != versionString() {
		t.Fatalf("embedded version %q does not match VERSION file %q", versionString(), strings.TrimSpace(string(data)))
	}
}

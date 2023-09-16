package utils

import (
	"testing"
)

func TestRandomHex(t *testing.T) {
	str := RandomHex(10)
	if len(str) != 10 {
		t.Errorf("String length mismatch, got %s, len=%d", str, len(str))
	}
}

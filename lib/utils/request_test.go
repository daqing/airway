package utils

import (
	"net/http/httptest"
	"testing"
)

func TestSameOriginRequest(t *testing.T) {
	for _, tc := range []struct {
		name   string
		origin string
		host   string
		want   bool
	}{
		{"no origin header passes", "", "127.0.0.1:5100", true},
		{"matching host passes", "http://127.0.0.1:5100", "127.0.0.1:5100", true},
		{"foreign origin rejected", "http://evil.example", "127.0.0.1:5100", false},
		{"different port rejected", "http://127.0.0.1:9999", "127.0.0.1:5100", false},
		{"unparsable origin rejected", "://", "127.0.0.1:5100", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest("POST", "/", nil)
			req.Host = tc.host
			if tc.origin != "" {
				req.Header.Set("Origin", tc.origin)
			}

			if got := SameOriginRequest(req); got != tc.want {
				t.Fatalf("SameOriginRequest(origin=%q host=%q) = %v, want %v", tc.origin, tc.host, got, tc.want)
			}
		})
	}
}

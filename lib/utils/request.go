package utils

import (
	"net/http"
	"net/url"
)

// SameOriginRequest reports whether the request carries no Origin header
// (plain navigation, curl) or one whose host equals the request Host.
// Same-origin-only middlewares and WebSocket upgraders share it.
func SameOriginRequest(r *http.Request) bool {
	origin := r.Header.Get("Origin")
	if origin == "" {
		return true
	}

	u, err := url.Parse(origin)
	if err != nil {
		return false
	}

	return u.Host == r.Host
}

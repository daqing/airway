package utils

import "net"

// DefaultListen is the address the server binds when LISTEN is not set.
const DefaultListen = ":1900"

// ListenAddress returns the TCP address the server binds: LISTEN (or
// AIRWAY_LISTEN) when set — host and port together, e.g. "0.0.0.0:1905" or
// ":1905" — otherwise DefaultListen.
func ListenAddress() string {
	if addr := GetEnvMulti("AIRWAY_LISTEN", "LISTEN"); addr != "" {
		return addr
	}

	return DefaultListen
}

// ListenHostPort splits a listen address into host and port, normalizing
// wildcard hosts ("", "0.0.0.0", "::") to "": a wildcard bind is not
// something a client can dial, so callers substitute their own loopback host.
// ok is false for addresses that do not carry a port.
func ListenHostPort(listen string) (host string, port string, ok bool) {
	host, port, err := net.SplitHostPort(listen)
	if err != nil {
		return "", "", false
	}

	switch host {
	case "0.0.0.0", "::":
		host = ""
	}

	return host, port, true
}

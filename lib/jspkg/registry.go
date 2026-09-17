package jspkg

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// Client is a minimal npm-registry HTTP client (registry.npmjs.org and its
// mirrors, e.g. registry.npmmirror.com, share the API).
type Client struct {
	BaseURL string
	HTTP    *http.Client
	Log     func(format string, args ...any)
}

func NewClient(registry string, log func(format string, args ...any)) *Client {
	if registry == "" {
		registry = DefaultRegistry
	}
	return &Client{
		BaseURL: strings.TrimRight(registry, "/"),
		HTTP:    &http.Client{Timeout: 60 * time.Second},
		Log:     log,
	}
}

func (c *Client) logf(format string, args ...any) {
	if c.Log != nil {
		c.Log(format, args...)
	}
}

// Packument is the per-package registry document.
type Packument struct {
	DistTags map[string]string      `json:"dist-tags"`
	Versions map[string]VersionMeta `json:"versions"`
}

// VersionMeta describes one published version of a package.
type VersionMeta struct {
	Version              string            `json:"version"`
	Dependencies         map[string]string `json:"dependencies"`
	OptionalDependencies map[string]string `json:"optionalDependencies"`
	Dist                 struct {
		Tarball   string `json:"tarball"`
		Integrity string `json:"integrity"`
	} `json:"dist"`
}

// pkgURL encodes a package name for the registry path; scoped names need
// their slash escaped (@scope%2Fname), which both npmjs and npmmirror accept.
func pkgURL(base, name string) string {
	return strings.TrimRight(base, "/") + "/" + url.PathEscape(name)
}

func (c *Client) getJSON(u string, into any) error {
	resp, err := c.HTTP.Get(u)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("GET %s: status %d", redactURL(u), resp.StatusCode)
	}
	return json.NewDecoder(resp.Body).Decode(into)
}

// FetchPackument returns the full package document.
func (c *Client) FetchPackument(name string) (*Packument, error) {
	var p Packument
	if err := c.getJSON(pkgURL(c.BaseURL, name), &p); err != nil {
		return nil, fmt.Errorf("fetch metadata for %s: %w", name, err)
	}
	return &p, nil
}

// FetchVersion returns the single-version document, used during install to
// locate the tarball URL on the currently configured registry.
func (c *Client) FetchVersion(name, version string) (*VersionMeta, error) {
	var m VersionMeta
	u := pkgURL(c.BaseURL, name) + "/" + version
	if err := c.getJSON(u, &m); err != nil {
		return nil, fmt.Errorf("fetch %s@%s: %w", name, version, err)
	}
	return &m, nil
}

// FetchTarball downloads a package tarball.
func (c *Client) FetchTarball(u string) ([]byte, error) {
	resp, err := c.HTTP.Get(u)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GET %s: status %d", redactURL(u), resp.StatusCode)
	}
	return io.ReadAll(resp.Body)
}

// redactURL keeps auth query parameters out of error messages.
func redactURL(u string) string {
	if parsed, err := url.Parse(u); err == nil && parsed.RawQuery != "" {
		parsed.RawQuery = "…"
		return parsed.String()
	}
	return u
}

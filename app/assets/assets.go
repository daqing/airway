// Package assets embeds the production frontend bundle (app/assets/dist,
// produced by `airway js:build`) and serves it under /assets/. The dist output
// is committed, so a fresh clone builds and serves without re-running the
// bundler. StaticHandler serves extra files from a directory on disk under a
// configurable URL prefix (default /static), keeping /assets/ limited to the
// embedded bundle.
package assets

import (
	"crypto/sha256"
	"embed"
	"encoding/hex"
	"encoding/json"
	"io/fs"
	"log"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/daqing/airway/lib/jsbuild"
)

//go:embed all:dist
var distFS embed.FS

var (
	manifestOnce sync.Once
	manifest     *jsbuild.Manifest
	manifestErr  error
)

// Manifest returns the manifest of the embedded build.
func Manifest() (*jsbuild.Manifest, error) {
	manifestOnce.Do(func() {
		data, err := distFS.ReadFile("dist/" + jsbuild.ManifestFile)
		if err != nil {
			manifestErr = err
			return
		}
		manifest = &jsbuild.Manifest{}
		manifestErr = json.Unmarshal(data, manifest)
	})
	return manifest, manifestErr
}

// EntryPath returns the cache-busted public path of the bundle entry, e.g.
// /assets/app.js?v=a80ecb1ad25190cb. Callers should prepend URL_PREFIX.
func EntryPath() string {
	m, err := Manifest()
	if err != nil || m.Hash == "" {
		return "/assets/" + jsbuild.EntryJS
	}
	return "/assets/" + m.Entry + "?v=" + m.Hash
}

// Handler serves the embedded bundle under /assets/. Requests carrying the
// ?v=<hash> query string are immutable-cacheable; bare paths fall back to
// ETag revalidation.
func Handler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		name := path.Clean(strings.TrimPrefix(r.URL.Path, "/assets/"))
		if !fs.ValidPath(name) || strings.Contains(name, `\`) {
			http.NotFound(w, r)
			return
		}
		data, err := fs.ReadFile(distFS, "dist/"+name)
		if err != nil {
			http.NotFound(w, r)
			return
		}

		if r.URL.Query().Get("v") != "" {
			w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
		} else {
			sum := sha256.Sum256(data)
			etag := `"` + hex.EncodeToString(sum[:])[:16] + `"`
			w.Header().Set("ETag", etag)
			w.Header().Set("Cache-Control", "no-cache")
			if r.Header.Get("If-None-Match") == etag {
				w.WriteHeader(http.StatusNotModified)
				return
			}
		}

		w.Header().Set("Content-Type", contentType(name))
		http.ServeContent(w, r, name, time.Time{}, strings.NewReader(string(data)))
	})
}

// StaticHandler serves files from root under urlPrefix (e.g. "static" or
// "downloads"). Directories are never listed; an unset or unusable root
// answers 404 for every request.
func StaticHandler(root, urlPrefix string) http.Handler {
	prefix := strings.Trim(urlPrefix, "/")
	if root != "" && prefix != "" {
		if info, err := os.Stat(root); err != nil || !info.IsDir() {
			log.Printf("assets: STATIC_PATH %q is not a directory; /%s is disabled", root, prefix)
			root = ""
		}
	}
	base := "/" + prefix + "/"

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if root == "" || prefix == "" {
			http.NotFound(w, r)
			return
		}
		name := path.Clean(strings.TrimPrefix(r.URL.Path, base))
		if !fs.ValidPath(name) || strings.Contains(name, `\`) {
			http.NotFound(w, r)
			return
		}
		file := filepath.Join(root, name)
		if info, err := os.Stat(file); err != nil || info.IsDir() {
			http.NotFound(w, r)
			return
		}
		http.ServeFile(w, r, file)
	})
}

func contentType(name string) string {
	switch path.Ext(name) {
	case ".js":
		return "text/javascript; charset=utf-8"
	case ".map", ".json":
		return "application/json"
	case ".css":
		return "text/css; charset=utf-8"
	default:
		return "application/octet-stream"
	}
}

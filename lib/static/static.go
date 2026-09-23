// Package static exports Airway app pages as a plain HTML directory
// deployable to any static host or CDN. Pages are templ components that
// render without a request — data is baked in at build time — so exporting
// is pure rendering: no HTTP server, no database. Interactive islands keep
// working from the static bundle, because the island runtime mounts
// client-side from the props embedded in each page (see app/assets).
//
// Output conventions — <slug>/index.html plus an assets/ directory — match
// lib/ssg, the showcase-site exporter.
package static

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/a-h/templ"

	"github.com/daqing/airway/lib/ssg"
)

// AssetDir is the output directory, relative to the export root, that
// receives the frontend bundle (app/assets/dist). Pages reference it under
// /assets/. It matches ssg.AssetDir.
const AssetDir = "assets"

// Page is one exportable app page: a URL path plus the component that
// renders its complete HTML document.
type Page struct {
	// Slug is the URL path of the page, normalized to a clean absolute
	// path ("/", "/about"). It decides the export location:
	// <out>/<slug>/index.html.
	Slug string
	// Component renders the full document. It must not depend on a request
	// or the database: export-time rendering happens offline, so page data
	// is baked in at build time.
	Component templ.Component
}

// NormalizeSlug canonicalizes a page slug exactly like ssg.NormalizeSlug,
// keeping the two exporters' output layouts identical.
func NormalizeSlug(slug string) string {
	return ssg.NormalizeSlug(slug)
}

// dirName maps the page slug to its output directory under the export root:
// "/" becomes the root itself, "/about" becomes "about".
func (p Page) dirName() string {
	return strings.Trim(p.Slug, "/")
}

// Build renders every page into outDir as a static HTML site: each page
// becomes <outDir>/<slug>/index.html. Existing files are overwritten in
// place; nothing else inside outDir is touched. The frontend bundle is not
// handled here — copy app/assets/dist into <outDir>/assets separately (the
// `airway static:build` command does both).
func Build(pages []Page, outDir string) error {
	if outDir == "" {
		return fmt.Errorf("static: export directory must not be empty")
	}

	normalized := make([]Page, len(pages))
	copy(normalized, pages)

	seen := make(map[string]bool, len(normalized))
	for _, page := range normalized {
		page.Slug = NormalizeSlug(page.Slug)

		if page.Component == nil {
			return fmt.Errorf("static: page %s has no component", page.Slug)
		}
		if seen[page.Slug] {
			return fmt.Errorf("static: duplicate page slug %q", page.Slug)
		}
		seen[page.Slug] = true

		var buf bytes.Buffer
		if err := page.Component.Render(context.Background(), &buf); err != nil {
			return fmt.Errorf("static: render %s: %w", page.Slug, err)
		}

		dir := filepath.Join(outDir, filepath.FromSlash(page.dirName()))
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return fmt.Errorf("static: create %s: %w", dir, err)
		}

		target := filepath.Join(dir, "index.html")
		if err := os.WriteFile(target, buf.Bytes(), 0o644); err != nil {
			return fmt.Errorf("static: write %s: %w", target, err)
		}
	}

	return nil
}

// Handler serves the pages dynamically — `airway static:serve` builds on it
// for previewing exactly what static:build would export.
func Handler(pages []Page) http.Handler {
	mux := http.NewServeMux()

	for _, page := range pages {
		page.Slug = NormalizeSlug(page.Slug)

		// "/" is a catch-all in ServeMux; /{$} matches the root alone so
		// unknown paths still 404.
		pattern := "GET " + page.Slug
		if page.Slug == "/" {
			pattern = "GET /{$}"
		}

		mux.HandleFunc(pattern, func(w http.ResponseWriter, r *http.Request) {
			var buf bytes.Buffer
			if err := page.Component.Render(r.Context(), &buf); err != nil {
				http.Error(w, "render failed: "+err.Error(), http.StatusInternalServerError)
				return
			}
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			_, _ = w.Write(buf.Bytes())
		})
	}

	return mux
}

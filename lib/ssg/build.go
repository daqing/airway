package ssg

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"os"
	"path/filepath"

	"github.com/daqing/airway/lib/utils"
)

// AssetDir is the output directory, relative to the export root, that
// receives the theme's static files. Pages reference them under /assets/.
const AssetDir = "assets"

// Build renders every page of s into outDir as a static HTML site: each page
// becomes <outDir>/<slug>/index.html (the root page becomes index.html) and
// the theme's assets are copied into <outDir>/assets. Existing files are
// overwritten in place; nothing else inside outDir is touched.
func Build(s *Site, outDir string) error {
	theme := s.Theme()
	if theme == nil {
		return fmt.Errorf("site: no theme selected; call Site.Use before Build")
	}

	for _, page := range s.Pages() {
		html, err := renderPage(s, page)
		if err != nil {
			return fmt.Errorf("site: render %s: %w", page.Slug, err)
		}

		dir := filepath.Join(outDir, filepath.FromSlash(page.DirName()))
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return fmt.Errorf("site: create %s: %w", dir, err)
		}

		target := filepath.Join(dir, "index.html")
		if err := os.WriteFile(target, html, 0o644); err != nil {
			return fmt.Errorf("site: write %s: %w", target, err)
		}
	}

	if theme.Assets() != nil {
		if err := utils.CopyFS(theme.Assets(), filepath.Join(outDir, AssetDir)); err != nil {
			return fmt.Errorf("site: copy theme assets: %w", err)
		}
	}

	return nil
}

// Handler serves the site dynamically: pages render per request and theme
// assets stream from the theme's embedded filesystem. `airway ssg:serve`
// builds on it for local preview; the same Site exports with Build.
func Handler(s *Site) http.Handler {
	mux := http.NewServeMux()

	if theme := s.Theme(); theme != nil && theme.Assets() != nil {
		prefix := "/" + AssetDir + "/"
		mux.Handle("GET "+prefix, http.StripPrefix(prefix, http.FileServer(http.FS(theme.Assets()))))
	}

	for _, page := range s.Pages() {
		// "/" is a catch-all in ServeMux; /{$} matches the root alone so
		// unknown paths still 404.
		pattern := "GET " + page.Slug
		if page.Slug == "/" {
			pattern = "GET /{$}"
		}

		mux.HandleFunc(pattern, func(w http.ResponseWriter, r *http.Request) {
			html, err := renderPage(s, page)
			if err != nil {
				http.Error(w, "render failed: "+err.Error(), http.StatusInternalServerError)
				return
			}
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			_, _ = w.Write(html)
		})
	}

	return mux
}

func renderPage(s *Site, page Page) ([]byte, error) {
	var buf bytes.Buffer
	comp := s.Theme().Render(&Context{Meta: s.Meta(), Page: page, Pages: s.Pages()})
	if err := comp.Render(context.Background(), &buf); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

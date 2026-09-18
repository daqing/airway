// Package jsbuild bundles the frontend TypeScript/TSX sources with esbuild
// (embedded as a Go library — no Node toolchain involved). Production builds
// write fixed-name outputs into app/assets/dist together with a manifest
// whose hash doubles as the ?v= cache buster; the dev server keeps output in
// memory and rebuilds when sources change.
package jsbuild

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/daqing/airway/lib/jspkg"
	"github.com/evanw/esbuild/pkg/api"
)

const (
	// SourceDir is the frontend source root, relative to the project root.
	SourceDir = "app/assets/js"

	// EntryPoint is the single bundle entry.
	EntryPoint = SourceDir + "/app.tsx"

	// VendorDir holds the packages installed by `airway js:install`
	// (node_modules-compatible layout, resolved through NodePaths).
	VendorDir = SourceDir + "/vendor"

	// DistDir receives production build output; it is committed so a fresh
	// clone builds without re-running js:build.
	DistDir = "app/assets/dist"

	// EntryJS is the fixed bundle filename. Cache busting uses the ?v=<hash>
	// query string from the manifest, keeping rebuild diffs small.
	EntryJS = "app.js"

	// EntryCSS is the stylesheet emitted when the entry imports CSS
	// (airway-ui styles under css/).
	EntryCSS = "app.css"

	// ManifestFile records the build hash; view helpers read it to build
	// cache-busted asset URLs.
	ManifestFile = "manifest.json"
)

// Manifest describes one build's outputs.
type Manifest struct {
	Entry string   `json:"entry"`
	Hash  string   `json:"hash"`
	Files []string `json:"files"`
}

// Alias maps react imports onto preact/compat so React-ecosystem packages
// bundle against Preact. jsx-dev-runtime is included because esbuild's
// automatic JSX mode emits dev-runtime imports when targeting development.
func Alias() map[string]string {
	return map[string]string{
		"react":                 "preact/compat",
		"react/jsx-runtime":     "preact/compat/jsx-runtime",
		"react/jsx-dev-runtime": "preact/compat/jsx-runtime",
		"react-dom":             "preact/compat",
		"react-dom/client":      "preact/compat/client",
		"react-dom/server":      "preact/compat/server",
	}
}

func options(root string, write bool) api.BuildOptions {
	return api.BuildOptions{
		EntryPoints: []string{EntryPoint},
		Outdir:      DistDir,
		Bundle:      true,
		Write:       write,
		Format:      api.FormatESModule,
		Target:      api.ES2020,
		Platform:    api.PlatformBrowser,
		JSX:         api.JSXAutomatic,
		// jsx runtime resolves through the react alias onto preact/compat,
		// so the whole island tree runs with React semantics (ref
		// forwarding included — required by react-hook-form's register).
		JSXImportSource:   "react",
		Alias:             Alias(),
		NodePaths:         []string{filepath.Join(root, VendorDir)},
		Plugins:           []api.Plugin{islandsPlugin(filepath.Join(root, SourceDir, "islands"))},
		Define:            map[string]string{"process.env.NODE_ENV": `"production"`},
		MinifyWhitespace:  true,
		MinifySyntax:      true,
		MinifyIdentifiers: true,
		Sourcemap:         api.SourceMapExternal,
		AbsWorkingDir:     root,
		LogLevel:          api.LogLevelWarning,
	}
}

// ErrVendorMissing is returned by Build and StartDev when js.pkg.json
// declares dependencies but the vendor directory is absent. Server boots
// check for it to fail fast instead of continuing without a frontend.
var ErrVendorMissing = fmt.Errorf("%s is missing; run `airway js:install` to fetch the %s dependencies", VendorDir, jspkg.ManifestFile)

// checkVendor fails fast when js.pkg.json declares dependencies but the
// vendor directory they install into is absent — the freshly scaffolded
// project case, where a build would otherwise drown in esbuild "Could not
// resolve" errors. Everything else is left to esbuild's own diagnostics, so
// projects without npm dependencies still build without a vendor directory.
func checkVendor(root string) error {
	if info, err := os.Stat(filepath.Join(root, VendorDir)); err == nil && info.IsDir() {
		return nil
	}
	m, err := jspkg.ReadManifest(root)
	if err != nil || len(m.Deps) == 0 {
		return nil
	}
	return ErrVendorMissing
}

// Build runs a production build into DistDir and writes ManifestFile.
// root may be relative; it is resolved against the current directory.
func Build(root string) (*Manifest, error) {
	root, err := filepath.Abs(root)
	if err != nil {
		return nil, err
	}
	if err := checkVendor(root); err != nil {
		return nil, err
	}
	result := api.Build(options(root, true))
	if errs := errorTexts(result); len(errs) > 0 {
		return nil, fmt.Errorf("esbuild: %s", strings.Join(errs, "; "))
	}

	m, err := manifestFrom(result)
	if err != nil {
		return nil, err
	}

	data, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return nil, err
	}
	if err := os.MkdirAll(filepath.Join(root, DistDir), 0o755); err != nil {
		return nil, err
	}
	if err := os.WriteFile(filepath.Join(root, DistDir, ManifestFile), append(data, '\n'), 0o644); err != nil {
		return nil, err
	}
	return m, nil
}

func manifestFrom(result api.BuildResult) (*Manifest, error) {
	byName := map[string][]byte{}
	for _, f := range result.OutputFiles {
		byName[filepath.Base(f.Path)] = f.Contents
	}
	if _, ok := byName[EntryJS]; !ok {
		return nil, fmt.Errorf("build produced no %s", EntryJS)
	}

	files := make([]string, 0, len(byName))
	for name := range byName {
		files = append(files, name)
	}
	sort.Strings(files)

	h := sha256.New()
	for _, name := range files {
		h.Write([]byte(name))
		h.Write(byName[name])
	}
	return &Manifest{
		Entry: EntryJS,
		Hash:  hex.EncodeToString(h.Sum(nil))[:16],
		Files: files,
	}, nil
}

func errorTexts(result api.BuildResult) []string {
	texts := make([]string, 0, len(result.Errors))
	for _, e := range result.Errors {
		if e.Location != nil {
			texts = append(texts, fmt.Sprintf("%s: %s:%d:%d", e.Text, e.Location.File, e.Location.Line, e.Location.Column))
		} else {
			texts = append(texts, e.Text)
		}
	}
	return texts
}

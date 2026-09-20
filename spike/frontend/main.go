// Phase 0 spike for the frontend plan (see PLAN.md): validates the four
// core assumptions — esbuild Go API bundling Preact TSX, preact/compat
// compatibility for the TanStack/RHF stack, Watch/Rebuild, and in-memory
// serving with ETag. Throwaway code, not part of the framework.
//
// Usage (from this directory):
//
//	go run . --fetch-deps           # download npm deps into vendor/ (no Node)
//	go run .                        # build in memory and serve the demo page
//	go run . --watch                # same, plus rebuild on source change
package main

import (
	"archive/tar"
	"compress/gzip"
	"crypto/sha256"
	_ "embed"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"mime"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/evanw/esbuild/pkg/api"
	"github.com/gin-gonic/gin"
)

var (
	registry  = flag.String("registry", "https://registry.npmmirror.com", "npm registry base URL")
	addr      = flag.String("addr", "127.0.0.1:18080", "listen address")
	watchMode = flag.Bool("watch", false, "rebuild on source change")
	fetchOnly = flag.Bool("fetch-deps", false, "download npm deps into vendor/ and exit")
)

// Downloaded at their dist-tags latest; transitive deps come from each
// package's "dependencies" (peer deps are satisfied at build time by the
// react -> preact/compat alias).
var rootDeps = []string{
	"preact",
	"@tanstack/preact-query",
	"@tanstack/react-table",
	"@tanstack/react-virtual",
	"react-hook-form",
}

const (
	vendorDir = "vendor"
	entryJS   = "js/app.tsx"
)

// --- npm dependency fetcher (pure Go, prototype for Phase 1) ---

type versionMeta struct {
	Version      string            `json:"version"`
	Dependencies map[string]string `json:"dependencies"`
	Dist         struct {
		Tarball string `json:"tarball"`
	} `json:"dist"`
}

type packageMeta struct {
	DistTags map[string]string      `json:"dist-tags"`
	Versions map[string]versionMeta `json:"versions"`
}

func fetchDeps(reg string) error {
	client := &http.Client{Timeout: 60 * time.Second}
	visited := map[string]bool{}
	queue := append([]string(nil), rootDeps...)

	for len(queue) > 0 {
		name := queue[0]
		queue = queue[1:]
		if visited[name] {
			continue
		}
		visited[name] = true

		pkgURL := strings.TrimRight(reg, "/") + "/" + url.PathEscape(name)
		resp, err := client.Get(pkgURL)
		if err != nil {
			return fmt.Errorf("fetch metadata %s: %w", name, err)
		}
		var meta packageMeta
		err = json.NewDecoder(resp.Body).Decode(&meta)
		resp.Body.Close()
		if err != nil {
			return fmt.Errorf("decode metadata %s: %w", name, err)
		}

		ver := meta.DistTags["latest"]
		vm, ok := meta.Versions[ver]
		if !ok {
			return fmt.Errorf("%s: latest version %q not found", name, ver)
		}

		dest := filepath.Join(vendorDir, filepath.FromSlash(name))
		if v, err := readInstalledVersion(dest); err != nil || v != ver {
			log.Printf("downloading %s@%s", name, ver)
			if err := downloadTarball(client, vm.Dist.Tarball, dest); err != nil {
				return fmt.Errorf("download %s: %w", name, err)
			}
		} else {
			log.Printf("cached %s@%s", name, ver)
		}

		// deterministic queue order keeps re-runs reproducible
		deps := make([]string, 0, len(vm.Dependencies))
		for dep := range vm.Dependencies {
			deps = append(deps, dep)
		}
		sort.Strings(deps)
		queue = append(queue, deps...)
	}
	return nil
}

func readInstalledVersion(dir string) (string, error) {
	data, err := os.ReadFile(filepath.Join(dir, "package.json"))
	if err != nil {
		return "", err
	}
	var pkg struct {
		Version string `json:"version"`
	}
	if err := json.Unmarshal(data, &pkg); err != nil {
		return "", err
	}
	return pkg.Version, nil
}

func downloadTarball(client *http.Client, tarballURL, dest string) error {
	resp, err := client.Get(tarballURL)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("tarball status %d", resp.StatusCode)
	}

	gz, err := gzip.NewReader(resp.Body)
	if err != nil {
		return err
	}
	tr := tar.NewReader(gz)
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}
		name := filepath.Clean(hdr.Name)
		if name == "package" || !strings.HasPrefix(name, "package"+string(os.PathSeparator)) {
			continue // npm tarballs are rooted at package/
		}
		target := filepath.Join(dest, strings.TrimPrefix(name, "package"))
		switch hdr.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(target, 0o755); err != nil {
				return err
			}
		case tar.TypeReg:
			if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
				return err
			}
			f, err := os.OpenFile(target, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o644)
			if err != nil {
				return err
			}
			if _, err := io.Copy(f, tr); err != nil {
				f.Close()
				return err
			}
			f.Close()
		case tar.TypeSymlink:
			_ = os.Remove(target)
			if err := os.Symlink(hdr.Linkname, target); err != nil {
				return err
			}
		}
	}
}

// --- esbuild in-memory bundle + watch ---

func buildOptions() api.BuildOptions {
	return api.BuildOptions{
		EntryPoints:     []string{entryJS},
		Outdir:          "dist",
		Bundle:          true,
		Write:           false, // keep output in memory
		Format:          api.FormatESModule,
		Target:          api.ES2020,
		Platform:        api.PlatformBrowser,
		JSX:             api.JSXAutomatic,
		JSXImportSource: "preact",
		Alias: map[string]string{
			"react":                 "preact/compat",
			"react/jsx-runtime":     "preact/compat/jsx-runtime",
			"react/jsx-dev-runtime": "preact/compat/jsx-runtime",
			"react-dom":             "preact/compat",
			"react-dom/client":      "preact/compat/client",
			"react-dom/server":      "preact/compat/server",
		},
		NodePaths:         []string{vendorDir},
		Define:            map[string]string{"process.env.NODE_ENV": `"production"`},
		MinifyWhitespace:  true,
		MinifySyntax:      true,
		MinifyIdentifiers: true,
		Sourcemap:         api.SourceMapExternal,
		LogLevel:          api.LogLevelWarning,
	}
}

type outputs struct {
	mu    sync.RWMutex
	files map[string][]byte
	etag  string
}

// buildCtx is non-nil in --watch mode; Rebuild() then returns the freshest
// incremental BuildResult at any time.
var buildCtx api.BuildContext

func (o *outputs) store(res api.BuildResult) {
	for _, e := range res.Errors {
		log.Printf("esbuild error: %s (%v)", e.Text, e.Location)
	}
	if len(res.Errors) > 0 {
		return
	}
	h := sha256.New()
	files := map[string][]byte{}
	for _, f := range res.OutputFiles {
		base := filepath.Base(f.Path)
		files[base] = f.Contents
		h.Write([]byte(base))
		h.Write(f.Contents)
	}
	o.mu.Lock()
	o.files = files
	o.etag = `"` + hex.EncodeToString(h.Sum(nil))[:16] + `"`
	o.mu.Unlock()
	log.Printf("built %d file(s), etag %s", len(files), o.etag)
}

func (o *outputs) snapshot() (map[string][]byte, string) {
	o.mu.RLock()
	defer o.mu.RUnlock()
	return o.files, o.etag
}

// --- gin demo server ---

var apiHits int64

func main() {
	flag.Parse()

	if _, err := os.Stat(filepath.Join(vendorDir, "preact", "package.json")); err != nil || *fetchOnly {
		if err := fetchDeps(*registry); err != nil {
			log.Fatal(err)
		}
		if *fetchOnly {
			return
		}
	}

	out := &outputs{}
	if *watchMode {
		// esbuild v0.28 has no OnRebuild callback: Watch() rebuilds in the
		// background (polling-based), and Rebuild() returns the freshest
		// BuildResult at any time. The dev flow is therefore: watch for
		// incremental rebuilds + call Rebuild() when a request comes in.
		var cerr *api.ContextError
		buildCtx, cerr = api.Context(buildOptions())
		if cerr != nil {
			log.Fatal(cerr)
		}
		if err := buildCtx.Watch(api.WatchOptions{Delay: 200}); err != nil {
			log.Fatalf("watch: %v", err)
		}
		out.store(buildCtx.Rebuild())
	} else {
		out.store(api.Build(buildOptions()))
	}
	if _, etag := out.snapshot(); etag == "" {
		log.Fatal("initial build produced no output")
	}

	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	r.Use(gin.Logger())

	r.GET("/", func(c *gin.Context) {
		c.Data(http.StatusOK, "text/html; charset=utf-8", indexHTML)
	})

	r.GET("/assets/:file", func(c *gin.Context) {
		if buildCtx != nil {
			// pick up background watch rebuilds; no-op rebuild when unchanged
			start := time.Now()
			out.store(buildCtx.Rebuild())
			log.Printf("rebuild took %s", time.Since(start).Round(time.Millisecond))
		}
		files, etag := out.snapshot()
		data, ok := files[c.Param("file")]
		if !ok {
			c.String(http.StatusNotFound, "not built: %s", c.Param("file"))
			return
		}
		c.Header("ETag", etag)
		c.Header("Cache-Control", "no-cache")
		if c.Request.Header.Get("If-None-Match") == etag {
			c.Status(http.StatusNotModified)
			return
		}
		ct := mime.TypeByExtension(filepath.Ext(c.Param("file")))
		if filepath.Ext(c.Param("file")) == ".map" {
			ct = "application/json" // not in the system mime table on macOS
		}
		if ct == "" {
			ct = "application/octet-stream"
		}
		c.Data(http.StatusOK, ct, data)
	})

	r.GET("/api/items", func(c *gin.Context) {
		n := atomic.AddInt64(&apiHits, 1)
		items := make([]gin.H, 0, 8)
		for i := 1; i <= 8; i++ {
			items = append(items, gin.H{"id": i, "name": fmt.Sprintf("item %d", i)})
		}
		c.JSON(http.StatusOK, gin.H{
			"ok":   true,
			"data": gin.H{"items": items, "requestCount": n},
		})
	})

	log.Printf("serving on http://%s", *addr)
	log.Fatal(r.Run(*addr))
}

//go:embed index.html
var indexHTML []byte

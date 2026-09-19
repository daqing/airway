package jsbuild

import (
	"crypto/sha256"
	"encoding/hex"
	"io/fs"
	"log"
	"net/http"
	"path/filepath"
	"strings"
	"sync/atomic"
	"time"

	"github.com/evanw/esbuild/pkg/api"
)

// DevServer serves build output from memory and rebuilds when frontend
// sources change. esbuild v0.28's Watch() rebuilds in the background but
// exposes no rebuild event (Phase 0 finding), so a lightweight mtime poller
// triggers an explicit Rebuild() — whose BuildResult carries the output
// bytes — and then broadcasts a livereload message.
type DevServer struct {
	ctx       api.BuildContext
	out       atomic.Pointer[buildOutput]
	watcher   *sourceWatcher
	broadcast func(string)
}

type buildOutput struct {
	files map[string][]byte
	etag  string
}

var defaultDev atomic.Pointer[DevServer]

// StartDefault starts the dev server and registers it as the package
// default; route wiring picks it up when AIRWAY_ENV=local.
func StartDefault(root string, broadcast func(string)) (*DevServer, error) {
	dev, err := StartDev(root, broadcast)
	if err != nil {
		return nil, err
	}
	defaultDev.Store(dev)
	return dev, nil
}

// Default returns the dev server registered by StartDefault, or nil.
func Default() *DevServer { return defaultDev.Load() }

// StartDev builds once, then watches frontend sources for changes.
// broadcast (optional) receives a JSON message after every rebuild — the
// livereload client reloads the page on it.
func StartDev(root string, broadcast func(string)) (*DevServer, error) {
	root, err := filepath.Abs(root)
	if err != nil {
		return nil, err
	}
	if err := checkVendor(root); err != nil {
		return nil, err
	}
	ctx, cerr := api.Context(options(root, false))
	if cerr != nil {
		return nil, cerr
	}
	if err := ctx.Watch(api.WatchOptions{Delay: 200}); err != nil {
		return nil, err
	}

	dev := &DevServer{ctx: ctx, broadcast: broadcast}
	dev.store(ctx.Rebuild())
	dev.watcher = newSourceWatcher(filepath.Join(root, SourceDir), 300*time.Millisecond, dev.rebuild)
	go dev.watcher.run()
	return dev, nil
}

// Close stops watching and releases the esbuild context.
func (d *DevServer) Close() {
	d.watcher.Stop()
	d.ctx.Dispose()
}

func (d *DevServer) rebuild() {
	d.store(d.ctx.Rebuild())
	if d.broadcast != nil {
		d.broadcast(`{"type":"js-rebuild"}`)
	}
}

func (d *DevServer) store(res api.BuildResult) {
	if errs := errorTexts(res); len(errs) > 0 {
		log.Printf("jsbuild: %s", strings.Join(errs, "; "))
		return
	}
	out := memoryOutput(res)
	if out == nil {
		return
	}
	d.out.Store(out)
	log.Printf("jsbuild: rebuilt, etag %s", out.etag)
}

// memoryOutput captures a BuildResult's output files in memory; nil when
// the entry JS file is missing.
func memoryOutput(res api.BuildResult) *buildOutput {
	files := map[string][]byte{}
	h := sha256.New()
	for _, f := range res.OutputFiles {
		name := filepath.Base(f.Path)
		files[name] = f.Contents
		h.Write([]byte(name))
		h.Write(f.Contents)
	}
	if _, ok := files[EntryJS]; !ok {
		return nil
	}
	return &buildOutput{
		files: files,
		etag:  `"` + hex.EncodeToString(h.Sum(nil))[:16] + `"`,
	}
}

// Handler serves /assets/app.js, /assets/app.js.map and
// /assets/livereload.js straight from memory.
func (d *DevServer) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/assets/livereload.js", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/javascript; charset=utf-8")
		_, _ = w.Write([]byte(livereloadJS))
	})
	mux.HandleFunc("/assets/", func(w http.ResponseWriter, r *http.Request) {
		out := d.out.Load()
		if out == nil {
			http.Error(w, "frontend build failed — see server log", http.StatusInternalServerError)
			return
		}
		name := strings.TrimPrefix(r.URL.Path, "/assets/")
		data, ok := out.files[name]
		if !ok {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("ETag", out.etag)
		w.Header().Set("Cache-Control", "no-cache")
		if r.Header.Get("If-None-Match") == out.etag {
			w.WriteHeader(http.StatusNotModified)
			return
		}
		w.Header().Set("Content-Type", contentType(name))
		_, _ = w.Write(data)
	})
	return mux
}

func contentType(name string) string {
	switch filepath.Ext(name) {
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

// sourceWatcher polls source mtimes and calls onChange after any change.
// The baseline snapshot is taken synchronously at construction: deferring
// it to the goroutine would swallow changes made between StartDev returning
// and the first poll. The vendor/ subtree is skipped: packages there change
// only through `airway js:install`, and polling thousands of vendored files
// is wasteful.
type sourceWatcher struct {
	dir      string
	interval time.Duration
	onChange func()
	prev     map[string]int64
	stop     chan struct{}
	done     chan struct{}
}

func newSourceWatcher(dir string, interval time.Duration, onChange func()) *sourceWatcher {
	return &sourceWatcher{
		dir:      dir,
		interval: interval,
		onChange: onChange,
		prev:     scanMtimes(dir),
		stop:     make(chan struct{}),
		done:     make(chan struct{}),
	}
}

func (w *sourceWatcher) run() {
	defer close(w.done)
	ticker := time.NewTicker(w.interval)
	defer ticker.Stop()
	for {
		select {
		case <-w.stop:
			return
		case <-ticker.C:
			next := scanMtimes(w.dir)
			if !mtimesEqual(w.prev, next) {
				w.prev = next
				w.onChange()
			}
		}
	}
}

func (w *sourceWatcher) Stop() {
	close(w.stop)
	<-w.done
}

func scanMtimes(dir string) map[string]int64 {
	mtimes := map[string]int64{}
	_ = filepath.WalkDir(dir, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return nil // unreadable entries are skipped, not fatal
		}
		if entry.IsDir() {
			if entry.Name() == "vendor" && path != dir {
				return filepath.SkipDir
			}
			return nil
		}
		switch filepath.Ext(entry.Name()) {
		case ".ts", ".tsx", ".js", ".jsx", ".css":
		default:
			return nil
		}
		info, err := entry.Info()
		if err != nil {
			return nil
		}
		rel, err := filepath.Rel(dir, path)
		if err != nil {
			return nil
		}
		mtimes[rel] = info.ModTime().UnixNano()
		return nil
	})
	return mtimes
}

func mtimesEqual(a, b map[string]int64) bool {
	if len(a) != len(b) {
		return false
	}
	for k, v := range a {
		if b[k] != v {
			return false
		}
	}
	return true
}

// livereloadJS connects to the app WebSocket and reloads the page when a
// rebuild is broadcast. The ws path is passed via data-ws on the script tag
// so URL_PREFIX deployments work unchanged.
const livereloadJS = `(() => {
  const el = document.currentScript;
  const wsPath = (el && el.dataset.ws) || "/ws";
  const proto = location.protocol === "https:" ? "wss:" : "ws:";
  const connect = () => {
    const ws = new WebSocket(proto + "//" + location.host + wsPath);
    ws.onmessage = (e) => {
      try {
        const msg = JSON.parse(e.data);
        if (msg && msg.type === "js-rebuild") location.reload();
      } catch {}
    };
    ws.onclose = () => setTimeout(connect, 1000);
  };
  connect();
})();
`

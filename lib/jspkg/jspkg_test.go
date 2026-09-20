package jspkg

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

// --- semver ---

func TestResolveVersion(t *testing.T) {
	available := []string{
		"0.2.9", "1.0.0", "1.2.0", "1.2.3", "1.2.9", "1.9.0", "2.0.0",
		"2.1.0-beta.1", "0.0.3",
	}

	cases := []struct {
		rng  string
		want string
	}{
		{"1.2.3", "1.2.3"},          // exact
		{"^1.2.3", "1.9.0"},         // caret, major>0
		{"^0.2.3", "0.2.9"},         // caret, major==0
		{"^0.0.3", "0.0.3"},         // caret, major==0 minor==0
		{"~1.2.3", "1.2.9"},         // tilde patch
		{"~1.2", "1.2.9"},           // tilde minor
		{"~1", "1.9.0"},             // tilde major
		{">=1.2.0 <2.0.0", "1.9.0"}, // AND comparators
		{"1.x", "1.9.0"},            // x-range
		{"1.2.x", "1.2.9"},
		{"*", "2.0.0"},
		{"", "2.0.0"},
		{"1.2.3 || 2.1.0-beta.1", "2.1.0-beta.1"}, // explicit prerelease is selectable
		{"^2.0.0", "2.0.0"},
		{">1.2.3 <=1.9.0", "1.9.0"},
	}

	for _, tc := range cases {
		got, err := ResolveVersion(available, tc.rng)
		if err != nil {
			t.Errorf("ResolveVersion(%q): %v", tc.rng, err)
			continue
		}
		if got != tc.want {
			t.Errorf("ResolveVersion(%q) = %q, want %q", tc.rng, got, tc.want)
		}
	}

	// prereleases stay hidden unless named
	if got, err := ResolveVersion(available, "*"); err != nil || got != "2.0.0" {
		t.Errorf("prerelease leaked into wildcard match: %q %v", got, err)
	}
	if _, err := ResolveVersion(available, "^3.0.0"); err == nil {
		t.Errorf("expected error for unsatisfiable range")
	}
}

func TestSatisfyingAll(t *testing.T) {
	available := []string{"1.2.0", "1.2.5", "1.2.9", "1.9.0"}

	got, err := satisfyingAll(available, []string{"^1.2.0", "~1.2.3"})
	if err != nil || got != "1.2.9" {
		t.Errorf("satisfyingAll(^1.2.0, ~1.2.3) = %q, %v; want 1.2.9", got, err)
	}

	if got, err := satisfyingAll(available, []string{"^1.2.0", ">=1.9.0"}); err != nil || got != "1.9.0" {
		t.Errorf("satisfyingAll should keep versions satisfying every range, got %q %v", got, err)
	}

	if _, err := satisfyingAll([]string{"1.2.0", "2.0.0"}, []string{"^1.2.0", "^2.0.0"}); err == nil {
		t.Errorf("expected conflict for incompatible ranges")
	}
}

func TestParseSpec(t *testing.T) {
	cases := []struct {
		in            string
		name, version string
	}{
		{"preact", "preact", ""},
		{"preact@10.29.8", "preact", "10.29.8"},
		{"@tanstack/react-table", "@tanstack/react-table", ""},
		{"@tanstack/react-table@9.2.4", "@tanstack/react-table", "9.2.4"},
	}
	for _, tc := range cases {
		name, version, err := ParseSpec(tc.in)
		if err != nil || name != tc.name || version != tc.version {
			t.Errorf("ParseSpec(%q) = %q, %q, %v; want %q, %q", tc.in, name, version, err, tc.name, tc.version)
		}
	}
	if _, _, err := ParseSpec(""); err == nil {
		t.Errorf("expected error for empty spec")
	}
}

// --- tarball & integrity ---

func makeTarball(t *testing.T, name, version string) []byte {
	t.Helper()
	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	gz.ModTime = time.Time{} // deterministic bytes: integrity is precomputed
	tw := tar.NewWriter(gz)

	pkgJSON, _ := json.Marshal(map[string]string{"name": name, "version": version})
	files := map[string][]byte{
		"package/package.json": pkgJSON,
		"package/index.js":     []byte(fmt.Sprintf("export const name = %q;", name)),
		"package/dist/":        nil,
		"package/dist/x.js":    []byte("// dist"),
	}
	paths := make([]string, 0, len(files))
	for path := range files {
		paths = append(paths, path)
	}
	sort.Strings(paths) // deterministic byte stream: integrity is precomputed
	for _, path := range paths {
		content := files[path]
		if strings.HasSuffix(path, "/") {
			_ = tw.WriteHeader(&tar.Header{Name: path, Typeflag: tar.TypeDir, Mode: 0o755})
			continue
		}
		_ = tw.WriteHeader(&tar.Header{Name: path, Typeflag: tar.TypeReg, Mode: 0o644, Size: int64(len(content))})
		_, _ = tw.Write(content)
	}
	if err := tw.Close(); err != nil {
		t.Fatal(err)
	}
	if err := gz.Close(); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

func TestExtractTarballAndIntegrity(t *testing.T) {
	dest := t.TempDir()
	data := makeTarball(t, "demo", "1.0.0")
	if err := ExtractTarballBytes(data, dest); err != nil {
		t.Fatal(err)
	}

	pkg, err := os.ReadFile(filepath.Join(dest, "package.json"))
	if err != nil || !strings.Contains(string(pkg), `"version":"1.0.0"`) {
		t.Fatalf("package.json not extracted correctly: %q %v", pkg, err)
	}
	if _, err := os.Stat(filepath.Join(dest, "dist", "x.js")); err != nil {
		t.Fatalf("nested file missing: %v", err)
	}

	integrity := SRIFor(data)
	if err := CheckIntegrity(data, integrity); err != nil {
		t.Errorf("integrity check failed: %v", err)
	}
	if err := CheckIntegrity(append(data, 'x'), integrity); err == nil {
		t.Errorf("expected integrity mismatch")
	}
	if err := CheckIntegrity(data, "md5-abc"); err == nil {
		t.Errorf("expected unsupported-algorithm error")
	}
	if err := CheckIntegrity(data, ""); err != nil {
		t.Errorf("empty integrity should pass: %v", err)
	}
}

// --- registry fixture ---

// testRegistry serves an npm-like registry over httptest with a tiny package
// universe and a tarball download counter. Tarballs are deterministic per
// package version, so integrity values are stable across requests.
type testRegistry struct {
	*httptest.Server
	tarballHits int64 // downloads served
}

func newTestRegistry(t *testing.T, packages map[string]map[string]map[string]string) *testRegistry {
	t.Helper()
	reg := &testRegistry{}
	mux := http.NewServeMux()

	writeJSON := func(w http.ResponseWriter, v any) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(v)
	}
	// findPackage resolves a possibly-scope-escaped name; ok when the
	// package and version exist.
	findPackage := func(name, version string) (map[string]string, bool) {
		vers, ok := packages[name]
		if !ok {
			return nil, false
		}
		deps, ok := vers[version]
		return deps, ok
	}
	distFor := func(name, version string) map[string]string {
		data := makeTarball(t, name, version)
		return map[string]string{
			"tarball":   reg.URL + "/tarball/" + name + "/" + version,
			"integrity": SRIFor(data),
		}
	}

	mux.HandleFunc("/tarball/", func(w http.ResponseWriter, r *http.Request) {
		parts := strings.Split(strings.TrimPrefix(r.URL.Path, "/tarball/"), "/")
		if len(parts) < 2 {
			http.NotFound(w, r)
			return
		}
		version := parts[len(parts)-1]
		name := strings.Join(parts[:len(parts)-1], "/")
		if _, ok := findPackage(name, version); !ok {
			http.NotFound(w, r)
			return
		}
		atomic.AddInt64(&reg.tarballHits, 1)
		w.Header().Set("Content-Type", "application/octet-stream")
		_, _ = w.Write(makeTarball(t, name, version))
	})

	// EscapedPath keeps %2F in scoped names so {name}/{version} splitting is
	// unambiguous: exactly one unencoded slash separates name from version.
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		segs := strings.Split(strings.Trim(r.URL.EscapedPath(), "/"), "/")
		if len(segs) == 1 {
			name := strings.ReplaceAll(segs[0], "%2F", "/")
			vers, ok := packages[name]
			if !ok {
				http.NotFound(w, r)
				return
			}
			keys := make([]string, 0, len(vers))
			for v := range vers {
				keys = append(keys, v)
			}
			latest, _ := ResolveVersion(keys, "*")
			versions := map[string]any{}
			for version, deps := range vers {
				versions[version] = map[string]any{
					"version":      version,
					"dependencies": deps,
					"dist":         distFor(name, version),
				}
			}
			writeJSON(w, map[string]any{
				"dist-tags": map[string]string{"latest": latest},
				"versions":  versions,
			})
			return
		}

		name := strings.ReplaceAll(strings.Join(segs[:len(segs)-1], "/"), "%2F", "/")
		version := segs[len(segs)-1]
		deps, ok := findPackage(name, version)
		if !ok {
			http.NotFound(w, r)
			return
		}
		writeJSON(w, map[string]any{
			"version":      version,
			"dependencies": deps,
			"dist":         distFor(name, version),
		})
	})

	reg.Server = httptest.NewServer(mux)
	t.Cleanup(reg.Close)
	return reg
}

func TestAddResolvesTreeAndInstallIdempotent(t *testing.T) {
	reg := newTestRegistry(t, map[string]map[string]map[string]string{
		"alpha": {
			"1.0.0": {"beta": "^2.0.0"},
			"1.1.0": {"beta": "^2.1.0"},
		},
		"beta": {
			"2.0.0": {"@scope/gamma": "1.x"},
			"2.1.0": {"@scope/gamma": "~1.4.0"},
			"2.1.3": {"@scope/gamma": "~1.4.0"},
		},
		"@scope/gamma": {
			"1.4.2": {},
			"1.4.9": {},
			"1.9.0": {},
		},
	})

	root := t.TempDir()
	opt := Options{Registry: reg.URL, VendorDir: "vendor"}

	if err := Add(root, "alpha", opt); err != nil {
		t.Fatalf("Add: %v", err)
	}

	m, err := ReadManifest(root)
	if err != nil {
		t.Fatal(err)
	}
	if m.Deps["alpha"] != "1.1.0" {
		t.Errorf("alpha pinned to %q, want 1.1.0 (latest)", m.Deps["alpha"])
	}
	// beta ^2.1.0 → 2.1.3; gamma ~1.4.0 → 1.4.9
	if m.Lock["beta"].Version != "2.1.3" {
		t.Errorf("beta = %q, want 2.1.3", m.Lock["beta"].Version)
	}
	if m.Lock["@scope/gamma"].Version != "1.4.9" {
		t.Errorf("@scope/gamma = %q, want 1.4.9", m.Lock["@scope/gamma"].Version)
	}
	if m.Lock["alpha"].Integrity == "" {
		t.Errorf("lock entries must carry integrity")
	}

	for _, name := range []string{"alpha", "beta", "@scope/gamma"} {
		dir := filepath.Join(root, "vendor", name)
		if v := installedVersion(dir); v != m.Lock[name].Version {
			t.Errorf("vendor %s installed %q, want %q", name, v, m.Lock[name].Version)
		}
	}

	// manifest is valid JSON with stable sections
	raw, _ := os.ReadFile(filepath.Join(root, ManifestFile))
	if !json.Valid(raw) {
		t.Errorf("manifest is not valid JSON")
	}

	// second install: everything cached, zero new downloads
	hits := atomic.LoadInt64(&reg.tarballHits)
	if err := Install(root, opt); err != nil {
		t.Fatalf("Install (idempotent run): %v", err)
	}
	if got := atomic.LoadInt64(&reg.tarballHits); got != hits {
		t.Errorf("re-install downloaded %d tarballs; expected 0", got-hits)
	}

	// add a scoped package with an explicit version; existing installs stay cached
	hits = atomic.LoadInt64(&reg.tarballHits)
	if err := Add(root, "@scope/gamma@1.4.2", opt); err != nil {
		t.Fatalf("Add scoped: %v", err)
	}
	m, _ = ReadManifest(root)
	if m.Deps["@scope/gamma"] != "1.4.2" {
		t.Errorf("scoped dep = %q, want 1.4.2", m.Deps["@scope/gamma"])
	}
	// only gamma 1.4.2 re-downloaded (3 packages were cached)
	if got := atomic.LoadInt64(&reg.tarballHits); got != hits+1 {
		t.Errorf("add re-downloaded %d tarballs, want 1", got-hits)
	}
}

func TestInstallRejectsEmptyDeps(t *testing.T) {
	root := t.TempDir()
	if err := Install(root, Options{Registry: "http://unused.example"}); err == nil {
		t.Errorf("expected error for empty deps")
	}
}

func TestPkgURL(t *testing.T) {
	got := pkgURL("https://registry.npmmirror.com", "@tanstack/react-table")
	if got != "https://registry.npmmirror.com/@tanstack%2Freact-table" {
		t.Errorf("pkgURL = %q", got)
	}
	got = pkgURL("https://registry.npmjs.org/", "preact")
	if got != "https://registry.npmjs.org/preact" {
		t.Errorf("pkgURL = %q", got)
	}
}

func TestManifestRoundTrip(t *testing.T) {
	root := t.TempDir()
	m := &Manifest{
		Deps: map[string]string{"preact": "10.29.8"},
		Lock: map[string]LockedPkg{"preact": {Version: "10.29.8", Integrity: "sha512-abc"}},
	}
	if err := WriteManifest(root, m); err != nil {
		t.Fatal(err)
	}
	got, err := ReadManifest(root)
	if err != nil {
		t.Fatal(err)
	}
	if got.Deps["preact"] != "10.29.8" || got.Lock["preact"].Integrity != "sha512-abc" {
		t.Errorf("round trip mismatch: %+v", got)
	}
}

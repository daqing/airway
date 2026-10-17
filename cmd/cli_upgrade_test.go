package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/daqing/airway/cmd/clitemplate"
)

func TestEveryTemplateFileIsClassified(t *testing.T) {
	paths, err := clitemplate.TemplatePaths()
	if err != nil {
		t.Fatalf("list template paths: %v", err)
	}
	if len(paths) == 0 {
		t.Fatal("template has no files")
	}

	for _, rel := range paths {
		if _, ok := classifySkeletonPath(rel); !ok {
			t.Errorf("template file %q has no upgrade rule; add it to skeletonRules", rel)
		}
	}
}

func TestMergeSkeletonFile(t *testing.T) {
	tests := []struct {
		name   string
		base   string
		local  string
		target string
		want   string
		ok     bool
	}{
		{
			name:   "no local edits",
			base:   "a\nb\n",
			local:  "a\nb\n",
			target: "a\nB\n",
			want:   "a\nB\n",
			ok:     true,
		},
		{
			name:   "non-overlapping edits",
			base:   "a\nb\nc\n",
			local:  "A\nb\nc\n",
			target: "a\nb\nC\n",
			want:   "A\nb\nC\n",
			ok:     true,
		},
		{
			name:   "same edit on both sides",
			base:   "a\nb\n",
			local:  "A\nb\n",
			target: "A\nb\n",
			want:   "A\nb\n",
			ok:     true,
		},
		{
			name:   "user appends after an upstream append",
			base:   "a\nb\n",
			local:  "a\nb\nuser\n",
			target: "a\nb\nupstream\n",
			want:   "a\nb\nuser\nupstream\n",
			ok:     true,
		},
		{
			name:   "user insertion at a replacement boundary",
			base:   "a\nb\n",
			local:  "a\nL\nb\n",
			target: "a\nB\n",
			want:   "a\nL\nB\n",
			ok:     true,
		},
		{
			name:   "conflicting edits",
			base:   "a\nb\nc\n",
			local:  "a\nX\nc\n",
			target: "a\nY\nc\n",
			ok:     false,
		},
		{
			name:   "insertion inside an upstream replacement",
			base:   "a\nb\nc\nd\n",
			local:  "a\nb\nL\nc\nd\n",
			target: "a\nX\nd\n",
			ok:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			merged, ok := mergeSkeletonFile([]byte(tt.local), []byte(tt.base), []byte(tt.target))
			if ok != tt.ok {
				t.Fatalf("merge ok = %v, want %v (merged: %q)", ok, tt.ok, merged)
			}
			if ok && string(merged) != tt.want {
				t.Fatalf("merged = %q, want %q", merged, tt.want)
			}
		})
	}
}

func TestMergeSkeletonFileIsIdempotent(t *testing.T) {
	base := []byte("package routes\n\nfunc Routes() {\n\told()\n}\n")
	local := []byte("package routes\n\nfunc Routes() {\n\told()\n\tcustom()\n}\n\nfunc custom() {}\n")
	target := []byte("package routes\n\nfunc Routes() {\n\told()\n\tnew()\n}\n\nfunc added() {}\n")

	merged, ok := mergeSkeletonFile(local, base, target)
	if !ok {
		t.Fatal("first merge conflicted")
	}
	again, ok := mergeSkeletonFile(merged, base, target)
	if !ok {
		t.Fatal("second merge conflicted")
	}
	if string(again) != string(merged) {
		t.Fatalf("merge is not idempotent:\nfirst:\n%s\nsecond:\n%s", merged, again)
	}
}

func TestUpgradeManifestRoundTrip(t *testing.T) {
	dir := t.TempDir()
	target := map[string][]byte{
		"config/routes.go":             []byte("package config\n"),
		"app/views/home/index.templ":   []byte("ignored\n"),
		"app/assets/dist/app.js":       []byte("ignored\n"),
		"app/assets/js/islands/x.tsx":  []byte("ignored\n"),
		"app/api/home_api/routes.go":   []byte("ignored\n"),
		"app/views/layouts/base.templ": []byte("package layouts\n"),
	}

	if err := writeUpgradeManifest(dir, "example.com/app", "v1.2.3", target); err != nil {
		t.Fatalf("write manifest: %v", err)
	}

	manifest, err := readUpgradeManifest(dir)
	if err != nil {
		t.Fatalf("read manifest: %v", err)
	}
	if manifest == nil {
		t.Fatal("manifest not written")
	}
	if manifest.Schema != upgradeManifestSchema {
		t.Errorf("schema = %d, want %d", manifest.Schema, upgradeManifestSchema)
	}
	if manifest.SkeletonVersion != "v1.2.3" {
		t.Errorf("version = %q", manifest.SkeletonVersion)
	}
	if manifest.Module != "example.com/app" {
		t.Errorf("module = %q", manifest.Module)
	}
	if manifest.Files["config/routes.go"] != skeletonHash(target["config/routes.go"]) {
		t.Errorf("routes hash = %q", manifest.Files["config/routes.go"])
	}
	if manifest.Files["app/views/layouts/base.templ"] == "" {
		t.Error("merge-mode templ file missing from manifest")
	}
	for _, rel := range []string{
		"app/views/home/index.templ",
		"app/assets/dist/app.js",
		"app/assets/js/islands/x.tsx",
		"app/api/home_api/routes.go",
	} {
		if _, ok := manifest.Files[rel]; ok {
			t.Errorf("never/regenerate file %q should not be recorded", rel)
		}
	}
}

func TestMissingEnvKeys(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, ".env.example"), strings.Join([]string{
		"# comment",
		"DSN=\"sqlite://./tmp/app.db\"",
		"LISTEN=\":1905\"",
		"STATIC_PATH=\"./static\"",
		"STATIC_URL_PREFIX=\"static\"",
		"",
	}, "\n"))
	writeFile(t, filepath.Join(dir, ".env"), "DSN=\"sqlite://./tmp/app.db\"\n# LISTEN is set in the shell\n")

	t.Setenv("LISTEN", ":1905")
	t.Setenv("STATIC_URL_PREFIX", "downloads")

	missing, err := missingEnvKeys(dir)
	if err != nil {
		t.Fatalf("missing env keys: %v", err)
	}

	if len(missing) != 1 || missing[0] != "STATIC_PATH" {
		t.Fatalf("missing = %#v, want [STATIC_PATH]", missing)
	}
}

func TestNewProjectWritesSkeletonManifest(t *testing.T) {
	dir := useTempWorkingDir(t)

	previous := Version
	Version = "v0.20.0"
	t.Cleanup(func() { Version = previous })

	if err := newProject("example.com/manifest-demo", false, false, ""); err != nil {
		t.Fatalf("new project: %v", err)
	}

	projectDir := filepath.Join(dir, "manifest-demo")
	manifest, err := readUpgradeManifest(projectDir)
	if err != nil {
		t.Fatalf("read manifest: %v", err)
	}
	if manifest == nil {
		t.Fatal("manifest not written")
	}
	if manifest.SkeletonVersion != "v0.20.0" {
		t.Errorf("version = %q, want v0.20.0", manifest.SkeletonVersion)
	}
	if manifest.Files["config/routes.go"] == "" {
		t.Error("manifest does not record config/routes.go")
	}
}

// TestUpgradeLegacyProjectFromModuleCache reproduces the original bug report:
// a project generated by an older release has no manifest and none of the
// newer skeleton files, so the baseline must come from the module cache at the
// version pinned in go.mod. It is opt-in because it downloads that module:
//
//	AIRWAY_UPGRADE_TEST_FROM_VERSION=v0.19.2 go test ./cmd -run TestUpgradeLegacyProjectFromModuleCache
func TestUpgradeLegacyProjectFromModuleCache(t *testing.T) {
	fromVersion := os.Getenv("AIRWAY_UPGRADE_TEST_FROM_VERSION")
	if fromVersion == "" {
		t.Skip("set AIRWAY_UPGRADE_TEST_FROM_VERSION to run this integration test")
	}

	dir := useTempWorkingDir(t)
	module := "example.com/legacy-demo"

	previous := Version
	Version = "v0.20.0"
	t.Cleanup(func() { Version = previous })

	moduleDir, err := downloadAirwayModule(dir, fromVersion)
	if err != nil {
		t.Fatalf("download %s: %v", fromVersion, err)
	}
	old, err := renderSkeletonFromDir(moduleDir, module)
	if err != nil {
		t.Fatalf("render %s: %v", fromVersion, err)
	}
	for rel, data := range old {
		path := filepath.Join(dir, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatalf("mkdir for %s: %v", rel, err)
		}
		if err := os.WriteFile(path, data, 0o644); err != nil {
			t.Fatalf("write %s: %v", rel, err)
		}
	}

	writeFile(t, filepath.Join(dir, "go.mod"), "module "+module+"\n\ngo 1.27.1\n\nrequire github.com/daqing/airway "+fromVersion+"\n")
	writeFile(t, filepath.Join(dir, ".env"), "DSN=\"sqlite://./tmp/airway.db\"\n")

	var upgradeErr error
	out := captureStdout(t, func() {
		upgradeErr = runUpgrade(upgradeOptions{Dir: dir, Force: true, NoDeps: true})
	})
	if upgradeErr != nil {
		t.Fatalf("upgrade: %v\noutput:\n%s", upgradeErr, out)
	}

	routes := readFile(t, filepath.Join(dir, "config", "routes.go"))
	if !strings.Contains(routes, "func staticRoutes(") {
		t.Errorf("routes.go was not upgraded:\n%s", routes)
	}
	assets := readFile(t, filepath.Join(dir, "app", "assets", "assets.go"))
	if !strings.Contains(assets, "func StaticHandler(") {
		t.Error("assets.go was not upgraded")
	}
	if _, err := os.Stat(filepath.Join(dir, upgradeManifestPath)); err != nil {
		t.Errorf("manifest not written: %v", err)
	}
}

func TestUpgradeSyncsLegacySkeleton(t *testing.T) {
	fixture, err := filepath.Abs(filepath.Join("testdata", "upgrade-v0.19.2", "template"))
	if err != nil {
		t.Fatalf("resolve fixture: %v", err)
	}

	dir := useTempWorkingDir(t)
	module := "example.com/upgrade-demo"

	previous := Version
	Version = "v0.20.0"
	t.Cleanup(func() { Version = previous })

	if err := clitemplate.Scaffold(dir, module); err != nil {
		t.Fatalf("scaffold: %v", err)
	}
	if err := rewriteScaffoldPort(dir); err != nil {
		t.Fatalf("rewrite port: %v", err)
	}

	old, err := renderSkeletonFromDir(fixture, module)
	if err != nil {
		t.Fatalf("render fixture: %v", err)
	}
	for rel, data := range old {
		path := filepath.Join(dir, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatalf("mkdir for %s: %v", rel, err)
		}
		if err := os.WriteFile(path, data, 0o644); err != nil {
			t.Fatalf("write fixture %s: %v", rel, err)
		}
	}

	routesPath := filepath.Join(dir, "config", "routes.go")
	routes := readFile(t, routesPath)
	routes = strings.Replace(routes,
		"\tplugin.MountAll(r)\n",
		"\tplugin.MountAll(r)\n\tr.GET(\"/custom\", customHandler)\n", 1)
	routes += "\nfunc customHandler(c *gin.Context) {}\n"
	writeFile(t, routesPath, routes)

	goMod := "module " + module + "\n\ngo 1.27.1\n\nrequire github.com/daqing/airway v0.19.2\n"
	writeFile(t, filepath.Join(dir, "go.mod"), goMod)
	env := "DSN=\"sqlite://./tmp/airway.db\"\nLISTEN=\":1905\"\n"
	writeFile(t, filepath.Join(dir, ".env"), env)

	var upgradeErr error
	out := captureStdout(t, func() {
		upgradeErr = runUpgrade(upgradeOptions{
			Dir:     dir,
			FromDir: fixture,
			Force:   true,
			NoDeps:  true,
		})
	})
	if upgradeErr != nil {
		t.Fatalf("upgrade: %v\noutput:\n%s", upgradeErr, out)
	}

	routes = readFile(t, routesPath)
	for _, want := range []string{
		"staticRoutes(r)",
		"func staticRoutes(",
		"STATIC_PATH",
		"r.GET(\"/custom\", customHandler)",
		"func customHandler(c *gin.Context) {}",
	} {
		if !strings.Contains(routes, want) {
			t.Errorf("routes.go missing %q:\n%s", want, routes)
		}
	}

	assets := readFile(t, filepath.Join(dir, "app", "assets", "assets.go"))
	for _, want := range []string{"func StaticHandler(", "fs.ValidPath(name)"} {
		if !strings.Contains(assets, want) {
			t.Errorf("assets.go missing %q", want)
		}
	}

	gitignore := readFile(t, filepath.Join(dir, ".gitignore"))
	if !strings.Contains(gitignore, "/static/*") {
		t.Errorf(".gitignore missing /static/*:\n%s", gitignore)
	}

	if _, err := os.Stat(filepath.Join(dir, "static", ".keep")); err != nil {
		t.Errorf("static/.keep not created: %v", err)
	}

	if got := readFile(t, filepath.Join(dir, "go.mod")); got != goMod {
		t.Errorf("--no-deps changed go.mod:\n%s", got)
	}
	if got := readFile(t, filepath.Join(dir, ".env")); got != env {
		t.Errorf("upgrade changed .env:\n%s", got)
	}
	for _, want := range []string{"STATIC_PATH", "STATIC_URL_PREFIX"} {
		if !strings.Contains(out, want) {
			t.Errorf("output does not mention missing key %s:\n%s", want, out)
		}
	}

	manifest, err := readUpgradeManifest(dir)
	if err != nil {
		t.Fatalf("read manifest: %v", err)
	}
	if manifest == nil {
		t.Fatal("manifest not written")
	}
	if manifest.SkeletonVersion != "v0.20.0" {
		t.Errorf("manifest version = %q, want v0.20.0", manifest.SkeletonVersion)
	}
	if manifest.Module != module {
		t.Errorf("manifest module = %q, want %q", manifest.Module, module)
	}
	if manifest.Files["config/routes.go"] == "" {
		t.Error("manifest does not record config/routes.go")
	}
}

func TestUpgradeCheckReportsDrift(t *testing.T) {
	fixture, err := filepath.Abs(filepath.Join("testdata", "upgrade-v0.19.2", "template"))
	if err != nil {
		t.Fatalf("resolve fixture: %v", err)
	}

	dir := useTempWorkingDir(t)
	module := "example.com/upgrade-check"

	previous := Version
	Version = "v0.20.0"
	t.Cleanup(func() { Version = previous })

	if err := clitemplate.Scaffold(dir, module); err != nil {
		t.Fatalf("scaffold: %v", err)
	}
	if err := rewriteScaffoldPort(dir); err != nil {
		t.Fatalf("rewrite port: %v", err)
	}

	old, err := renderSkeletonFromDir(fixture, module)
	if err != nil {
		t.Fatalf("render fixture: %v", err)
	}
	for rel, data := range old {
		path := filepath.Join(dir, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatalf("mkdir for %s: %v", rel, err)
		}
		if err := os.WriteFile(path, data, 0o644); err != nil {
			t.Fatalf("write fixture %s: %v", rel, err)
		}
	}

	writeFile(t, filepath.Join(dir, "go.mod"),
		"module "+module+"\n\ngo 1.27.1\n\nrequire github.com/daqing/airway v0.19.2\n")

	assetsPath := filepath.Join(dir, "app", "assets", "assets.go")
	before := readFile(t, assetsPath)

	var checkErr error
	_ = captureStdout(t, func() {
		checkErr = runUpgrade(upgradeOptions{
			Dir:     dir,
			FromDir: fixture,
			Check:   true,
			Force:   true,
		})
	})
	if checkErr == nil {
		t.Fatal("--check should report drift")
	}

	if after := readFile(t, assetsPath); after != before {
		t.Error("--check modified assets.go")
	}
	if _, err := os.Stat(filepath.Join(dir, upgradeManifestPath)); err == nil {
		t.Fatal("--check must not write the manifest")
	}
}

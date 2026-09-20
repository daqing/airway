package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/evanw/esbuild/pkg/api"
)

func TestBuildOptionsBundlePreactAliases(t *testing.T) {
	opts := buildOptions()

	if len(opts.EntryPoints) != 1 || opts.EntryPoints[0] != entryJS {
		t.Errorf("entry points %v", opts.EntryPoints)
	}

	if !opts.Bundle || opts.Write {
		t.Error("expected in-memory bundling")
	}

	if opts.Alias["react"] != "preact/compat" {
		t.Errorf("react alias %q", opts.Alias["react"])
	}

	if len(opts.NodePaths) != 1 || opts.NodePaths[0] != vendorDir {
		t.Errorf("node paths %v", opts.NodePaths)
	}
}

func TestReadInstalledVersion(t *testing.T) {
	dir := t.TempDir()

	if _, err := readInstalledVersion(dir); err == nil {
		t.Fatal("expected missing package.json to error")
	}

	if err := os.WriteFile(filepath.Join(dir, "package.json"), []byte(`{"version":"10.5.2"}`), 0o644); err != nil {
		t.Fatal(err)
	}

	version, err := readInstalledVersion(dir)
	if err != nil {
		t.Fatalf("readInstalledVersion: %v", err)
	}

	if version != "10.5.2" {
		t.Errorf("version %q", version)
	}
}

func TestOutputsStoreAndSnapshot(t *testing.T) {
	var o outputs
	o.store(api.BuildResult{OutputFiles: []api.OutputFile{
		{Path: "/build/dist/app.js", Contents: []byte("bundle")},
	}})

	files, etag := o.snapshot()
	if len(files) != 1 || string(files["app.js"]) != "bundle" {
		t.Errorf("files %v", files)
	}

	if etag == "" {
		t.Error("expected non-empty etag")
	}
}

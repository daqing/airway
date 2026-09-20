package jsbuild

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func writeIslandFixture(t *testing.T, files map[string]string) string {
	t.Helper()
	root := writeFixture(t, `import { registry } from "`+RegistryModule+`"; console.log(Object.keys(registry));`)
	islands := filepath.Join(root, SourceDir, "islands")
	for rel, content := range files {
		path := filepath.Join(islands, rel)
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return root
}

func TestGenerateRegistry(t *testing.T) {
	dir := t.TempDir()
	write := func(rel, content string) {
		path := filepath.Join(dir, rel)
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write("counter.tsx", "export default 1")
	write("admin/orders.ts", "export default 2")
	write("_partial.tsx", "export default 3")     // skipped: partial
	write("notes.md", "ignored by extension")     // skipped: extension
	write("_nested/skip.tsx", "export default 4") // skipped: _ dir

	code, err := generateRegistry(dir)
	if err != nil {
		t.Fatalf("generateRegistry: %v", err)
	}
	for _, want := range []string{
		`import Island0 from "./islands/admin/orders";`,
		`import Island1 from "./islands/counter";`,
		`"admin/orders": Island0,`,
		`"counter": Island1,`,
	} {
		if !strings.Contains(code, want) {
			t.Errorf("registry missing %q in:\n%s", want, code)
		}
	}
	if strings.Contains(code, "_partial") || strings.Contains(code, "notes") || strings.Contains(code, "_nested") {
		t.Errorf("registry includes skipped files:\n%s", code)
	}

	empty, err := generateRegistry(filepath.Join(dir, "does-not-exist"))
	if err != nil {
		t.Fatalf("missing dir should be fine: %v", err)
	}
	if !strings.Contains(empty, "registry: Record<string, (props: any) => any> = {}") {
		t.Errorf("empty registry wrong:\n%s", empty)
	}
}

func TestBuildBundlesIslandsViaRegistry(t *testing.T) {
	root := writeIslandFixture(t, map[string]string{
		"counter.tsx": "export default function Counter() { return \"phase3-marker\"; }\n",
	})

	if _, err := Build(root); err != nil {
		t.Fatalf("Build: %v", err)
	}

	bundle, err := os.ReadFile(filepath.Join(root, DistDir, EntryJS))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(bundle), "phase3-marker") {
		t.Errorf("bundle does not contain the island component code")
	}
}

func TestDevServerRebuildPicksUpNewIsland(t *testing.T) {
	root := writeIslandFixture(t, map[string]string{})

	broadcasts := make(chan string, 4)
	dev, err := StartDev(root, func(msg string) { broadcasts <- msg })
	if err != nil {
		t.Fatalf("StartDev: %v", err)
	}
	defer dev.Close()

	// adding an island file is a source-tree change: the mtime watcher
	// triggers a rebuild and the plugin rescans the islands directory
	islands := filepath.Join(root, SourceDir, "islands")
	if err := os.MkdirAll(islands, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(islands, "fresh.tsx"), []byte(`export default () => "fresh-island-marker";`), 0o644); err != nil {
		t.Fatal(err)
	}

	select {
	case <-broadcasts:
	case <-time.After(5 * time.Second):
		t.Fatal("no rebuild broadcast after adding an island file")
	}

	// the dev rebuild must have picked it up; verify via a disk build too
	if _, err := Build(root); err != nil {
		t.Fatalf("Build: %v", err)
	}
	bundle, err := os.ReadFile(filepath.Join(root, DistDir, EntryJS))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(bundle), "fresh-island-marker") {
		t.Errorf("new island did not reach the bundle after rebuild")
	}
}

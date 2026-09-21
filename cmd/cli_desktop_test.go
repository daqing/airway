package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

var stubbedAddDesktopDependency = addDesktopDependency

func setupDesktopProject(t *testing.T) string {
	t.Helper()

	addDesktopDependency = func() error { return nil }
	t.Cleanup(func() { addDesktopDependency = stubbedAddDesktopDependency })

	wd := useTempWorkingDir(t)
	makeDirs(t, filepath.Join(wd, "db", "migrate"))

	writeFile(t, filepath.Join(wd, "go.mod"), `module github.com/example/demo

go 1.27
`)
	writeFile(t, filepath.Join(wd, "plugins.go"), `package main

import (
	_ "github.com/daqing/airway/plugins/im"
	_ "github.com/example/demo/plugins/custom"
)

// Plugin support is enabled via blank imports.
`)

	writeFile(t, filepath.Join(wd, "db", "migrate", "20260327120000_create_posts.up.sql"), "CREATE TABLE posts (id INTEGER PRIMARY KEY);")
	writeFile(t, filepath.Join(wd, "db", "migrate", "20260327120000_create_posts.down.sql"), "DROP TABLE posts;")

	return wd
}

func TestDesktopInitGeneratesWrapper(t *testing.T) {
	wd := setupDesktopProject(t)

	if err := run([]string{"desktop:init"}); err != nil {
		t.Fatalf("desktop:init: %v", err)
	}

	for _, rel := range []string{
		"desktop/main.go",
		"desktop/migrations.go",
		"desktop/migrations/20260327120000_create_posts.up.sql",
		"desktop/migrations/20260327120000_create_posts.down.sql",
		"desktop/Taskfile.yml",
		"desktop/build/Taskfile.yml",
		"desktop/build/config.yml",
		"desktop/build/darwin/Info.plist",
		"desktop/build/windows/Taskfile.yml",
		"desktop/build/linux/nfpm/nfpm.yaml",
		"desktop/build/appicon.png",
	} {
		if _, err := os.Stat(filepath.Join(wd, rel)); err != nil {
			t.Fatalf("expected generated file %s: %v", rel, err)
		}
	}

	main := readFile(t, filepath.Join(wd, "desktop", "main.go"))
	if !strings.Contains(main, `"github.com/daqing/airway/lib/boot"`) {
		t.Fatalf("expected boot import in desktop main, got:\n%s", main)
	}
	if !strings.Contains(main, `"demo"`) {
		t.Fatalf("expected app name in desktop main, got:\n%s", main)
	}

	migrations := readFile(t, filepath.Join(wd, "desktop", "migrations.go"))
	if !strings.Contains(migrations, `_ "github.com/daqing/airway/plugins/im"`) {
		t.Fatalf("expected framework plugin mirror in migrations.go, got:\n%s", migrations)
	}
	if !strings.Contains(migrations, `_ "github.com/example/demo/plugins/custom"`) {
		t.Fatalf("expected project plugin mirror in migrations.go, got:\n%s", migrations)
	}

	plist := readFile(t, filepath.Join(wd, "desktop", "build", "darwin", "Info.plist"))
	if !strings.Contains(plist, "<string>com.example.demo</string>") {
		t.Fatalf("expected derived bundle id in Info.plist, got:\n%s", plist)
	}
	if !strings.Contains(plist, "NSAllowsLocalNetworking") {
		t.Fatalf("expected ATS local-networking exemption in Info.plist")
	}

	config := readFile(t, filepath.Join(wd, "desktop", "build", "config.yml"))
	if !strings.Contains(config, "productName: \"demo\"") {
		t.Fatalf("expected app name in desktop config, got:\n%s", config)
	}
}

func TestDesktopInitReRunProtectsMainGoAndSyncsMigrations(t *testing.T) {
	wd := setupDesktopProject(t)

	if err := run([]string{"desktop:init"}); err != nil {
		t.Fatalf("first desktop:init: %v", err)
	}

	writeFile(t, filepath.Join(wd, "db", "migrate", "20260401120000_add_notes.up.sql"), "CREATE TABLE notes (id INTEGER PRIMARY KEY);")
	writeFile(t, filepath.Join(wd, "db", "migrate", "20260401120000_add_notes.down.sql"), "DROP TABLE notes;")

	mainPath := filepath.Join(wd, "desktop", "main.go")
	writeFile(t, mainPath, "// user customization marker\n"+readFile(t, mainPath))

	if err := run([]string{"desktop:init"}); err != nil {
		t.Fatalf("second desktop:init: %v", err)
	}

	if !strings.Contains(readFile(t, mainPath), "user customization marker") {
		t.Fatalf("expected desktop/main.go to be preserved on re-run")
	}

	for _, name := range []string{
		"20260401120000_add_notes.up.sql",
		"20260327120000_create_posts.up.sql",
	} {
		if _, err := os.Stat(filepath.Join(wd, "desktop", "migrations", name)); err != nil {
			t.Fatalf("expected synced migration %s: %v", name, err)
		}
	}

	forced := readFile(t, filepath.Join(wd, "desktop", "migrations.go"))
	if !strings.Contains(forced, `_ "github.com/example/demo/plugins/custom"`) {
		t.Fatalf("expected plugin mirror to stay in sync, got:\n%s", forced)
	}

	if err := run([]string{"desktop:init", "--force"}); err != nil {
		t.Fatalf("forced desktop:init: %v", err)
	}

	if strings.Contains(readFile(t, mainPath), "user customization marker") {
		t.Fatalf("expected --force to regenerate desktop/main.go")
	}
}

func TestDesktopAppNameAndBundleID(t *testing.T) {
	for _, tc := range []struct {
		module   string
		wantName string
		wantID   string
	}{
		{"github.com/daqing/demo", "demo", "com.daqing.demo"},
		{"myapp", "myapp", "com.myapp"},
		{"gitee.com/some_one/my_app.v1", "my_appv1", "com.some-one.my-appv1"},
	} {
		if got := desktopAppName(tc.module); got != tc.wantName {
			t.Fatalf("desktopAppName(%q) = %q, want %q", tc.module, got, tc.wantName)
		}
		if got := desktopBundleID(tc.module); got != tc.wantID {
			t.Fatalf("desktopBundleID(%q) = %q, want %q", tc.module, got, tc.wantID)
		}
	}
}

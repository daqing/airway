package cmd

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestHostProjectAt(t *testing.T) {
	cases := []struct {
		name   string
		goMod  string
		mainGo bool
		want   bool
	}{
		{
			name: "host with require block",
			goMod: "module example.com/hostapp\n\n" +
				"go 1.26\n\n" +
				"require (\n\tgithub.com/daqing/airway v0.9.2\n\tgithub.com/gin-gonic/gin v1.10.0\n)\n",
			mainGo: true,
			want:   true,
		},
		{
			name:   "host with single-line require",
			goMod:  "module example.com/hostapp\n\nrequire github.com/daqing/airway v0.9.2\n",
			mainGo: true,
			want:   true,
		},
		{
			name: "indirect require with comment",
			goMod: "module example.com/hostapp\n\n" +
				"require (\n\tgithub.com/daqing/airway v0.9.2 // indirect\n)\n",
			mainGo: true,
			want:   true,
		},
		{
			name:   "plugin module without main.go",
			goMod:  "module example.com/airway-im-plugin\n\nrequire github.com/daqing/airway v0.9.2\n",
			mainGo: false,
			want:   false,
		},
		{
			name:   "framework repository itself",
			goMod:  "module github.com/daqing/airway\n\nrequire github.com/gin-gonic/gin v1.10.0\n",
			mainGo: true,
			want:   false,
		},
		{
			name:   "project requiring something else",
			goMod:  "module example.com/other\n\nrequire github.com/gin-gonic/gin v1.10.0\n",
			mainGo: true,
			want:   false,
		},
		{
			name:   "replace without require",
			goMod:  "module example.com/hostapp\n\nreplace github.com/daqing/airway => ../airway\n",
			mainGo: true,
			want:   false,
		},
		{
			name:   "no go.mod at all",
			goMod:  "",
			mainGo: true,
			want:   false,
		},
	}

	for _, tt := range cases {
		dir := t.TempDir()
		if tt.goMod != "" {
			writeFile(t, filepath.Join(dir, "go.mod"), tt.goMod)
		}
		if tt.mainGo {
			writeFile(t, filepath.Join(dir, "main.go"), "package main\n\nfunc main() {}\n")
		}

		if got := hostProjectAt(dir); got != tt.want {
			t.Fatalf("%s: hostProjectAt = %v, want %v", tt.name, got, tt.want)
		}
	}
}

func TestProxyableCommand(t *testing.T) {
	cases := []struct {
		command string
		want    bool
	}{
		{"", false},
		{"new", false},
		{"version", false},
		{"-v", false},
		{"--version", false},
		{"help", false},
		{"-h", false},
		{"--help", false},
		{"templates:compile", false},
		{"server", true},
		{"repl", true},
		{"plugin:list", true},
		{"plugin:install", true},
		{"db:migrate", true},
		{"generate", true},
		{"cli", true},
	}

	for _, tt := range cases {
		if got := proxyableCommand(tt.command); got != tt.want {
			t.Fatalf("proxyableCommand(%q) = %v, want %v", tt.command, got, tt.want)
		}
	}
}

func TestCheckHostAirwayVersion(t *testing.T) {
	oldVersion := Version
	t.Cleanup(func() { Version = oldVersion })

	cases := []struct {
		name        string
		version     string
		goMod       string
		checkoutVer string
		wantErr     string
	}{
		{
			name:    "matching pin",
			version: "v0.9.3",
			goMod:   "module example.com/hostapp\n\nrequire github.com/daqing/airway v0.9.3\n",
		},
		{
			name:    "matching pin without v prefix on the CLI side",
			version: "0.9.3",
			goMod:   "module example.com/hostapp\n\nrequire github.com/daqing/airway v0.9.3\n",
		},
		{
			name:    "matching pin inside require block",
			version: "v0.9.3",
			goMod: "module example.com/hostapp\n\n" +
				"require (\n\tgithub.com/daqing/airway v0.9.3 // indirect\n\tgithub.com/gin-gonic/gin v1.10.0\n)\n",
		},
		{
			name:    "mismatched pin",
			version: "v0.9.3",
			goMod:   "module example.com/hostapp\n\nrequire github.com/daqing/airway v0.9.2\n",
			wantErr: "version mismatch",
		},
		{
			name:        "replace onto matching local checkout",
			version:     "v0.9.3",
			goMod:       "module example.com/hostapp\n\nrequire github.com/daqing/airway v0.9.1\n\nreplace github.com/daqing/airway => ../airway\n",
			checkoutVer: "v0.9.3\n",
		},
		{
			name:        "replace onto drifted local checkout",
			version:     "v0.9.3",
			goMod:       "module example.com/hostapp\n\nrequire github.com/daqing/airway v0.9.1\n\nreplace github.com/daqing/airway => ../airway\n",
			checkoutVer: "v0.9.2\n",
			wantErr:     "version mismatch",
		},
		{
			name:    "replace onto local checkout without VERSION file",
			version: "v0.9.3",
			goMod:   "module example.com/hostapp\n\nrequire github.com/daqing/airway v0.9.1\n\nreplace github.com/daqing/airway => ../airway\n",
		},
		{
			name:    "replace onto matching module version",
			version: "v0.9.3",
			goMod:   "module example.com/hostapp\n\nrequire github.com/daqing/airway v0.9.1\n\nreplace github.com/daqing/airway => github.com/fork/airway v0.9.3\n",
		},
		{
			name:    "replace onto mismatched module version",
			version: "v0.9.3",
			goMod:   "module example.com/hostapp\n\nrequire github.com/daqing/airway v0.9.1\n\nreplace github.com/daqing/airway => github.com/fork/airway v0.9.2\n",
			wantErr: "version mismatch",
		},
		{
			name:    "replace block form",
			version: "v0.9.3",
			goMod: "module example.com/hostapp\n\n" +
				"require github.com/daqing/airway v0.9.1\n\n" +
				"replace (\n\tgithub.com/daqing/airway => github.com/fork/airway v0.9.3\n)\n",
		},
		{
			name:    "dev build skips the check",
			version: "dev",
			goMod:   "module example.com/hostapp\n\nrequire github.com/daqing/airway v0.9.2\n",
		},
	}

	for _, tt := range cases {
		Version = tt.version

		root := t.TempDir()
		host := filepath.Join(root, "host")
		makeDirs(t, host)
		writeFile(t, filepath.Join(host, "go.mod"), tt.goMod)
		if tt.checkoutVer != "" {
			checkout := filepath.Join(root, "airway")
			makeDirs(t, checkout)
			writeFile(t, filepath.Join(checkout, "VERSION"), tt.checkoutVer)
		}

		err := checkHostAirwayVersion(host)
		if tt.wantErr == "" {
			if err != nil {
				t.Fatalf("%s: checkHostAirwayVersion = %v, want nil", tt.name, err)
			}
			continue
		}
		if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
			t.Fatalf("%s: checkHostAirwayVersion = %v, want error containing %q", tt.name, err, tt.wantErr)
		}
	}
}

// A version mismatch aborts the proxy before `go run .` ever runs, so no
// project command executes with stale framework logic.
func TestProxyHostProjectAbortsOnVersionMismatch(t *testing.T) {
	oldVersion := Version
	t.Cleanup(func() { Version = oldVersion })
	Version = "v0.9.3"

	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "go.mod"), "module example.com/hostapp\n\nrequire github.com/daqing/airway v0.9.2\n")
	writeFile(t, filepath.Join(dir, "main.go"), "package main\n\nfunc main() {}\n")

	t.Chdir(dir)

	proxied, err := ProxyHostProject([]string{"db:migrate"})
	if err == nil || !strings.Contains(err.Error(), "version mismatch") {
		t.Fatalf("ProxyHostProject error = %v, want a version mismatch error", err)
	}
	if proxied {
		t.Fatal("expected the command not to be proxied on a version mismatch")
	}
}

func TestProxyHostProjectSkipsTemplatesCompile(t *testing.T) {
	// templates:compile must run locally even inside a host project: the
	// project may not compile yet — regenerating the templ views is how it
	// gets fixed — and go run . would abort on the build.
	oldVersion := Version
	t.Cleanup(func() { Version = oldVersion })
	Version = "v0.9.3"

	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "go.mod"), "module example.com/hostapp\n\nrequire github.com/daqing/airway v0.9.2\n")
	writeFile(t, filepath.Join(dir, "main.go"), "package main\n\nfunc main() {}\n")

	t.Chdir(dir)

	proxied, err := ProxyHostProject([]string{"templates:compile"})
	if err != nil {
		t.Fatalf("ProxyHostProject error = %v, want none", err)
	}
	if proxied {
		t.Fatal("expected templates:compile not to be proxied")
	}
}

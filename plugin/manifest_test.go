package plugin_test

import (
	"strings"
	"testing"

	pluginsdk "github.com/daqing/airway/plugin"
)

type noopPlugin struct{ manifest pluginsdk.Manifest }

func (p noopPlugin) Manifest() pluginsdk.Manifest   { return p.manifest }
func (noopPlugin) Register(pluginsdk.Context) error { return nil }

func validManifest(name string) pluginsdk.Manifest {
	return pluginsdk.Manifest{APIVersion: pluginsdk.APIVersion, Name: name, Version: "1.2.3", Core: ">=0.1.0 <0.2.0", Entry: "example.Register", Permissions: []pluginsdk.Permission{{Code: strings.ReplaceAll(name, "-", "_") + ":read", Name: "Read example"}}, Dependencies: []pluginsdk.Dependency{{Name: "foundation", Version: ">=1.0.0 <2.0.0"}}, Migrations: []string{"20260101000000_create_example"}}
}

func TestVersionedManifestValidation(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*pluginsdk.Manifest)
		want   string
	}{{name: "valid"}, {name: "api version", mutate: func(m *pluginsdk.Manifest) { m.APIVersion = "airway.dev/v2" }, want: "api_version"}, {name: "plugin version", mutate: func(m *pluginsdk.Manifest) { m.Version = "latest" }, want: "plugin version"}, {name: "core compatibility", mutate: func(m *pluginsdk.Manifest) { m.Core = ">=0.2.0" }, want: "incompatible"}, {name: "unsafe name", mutate: func(m *pluginsdk.Manifest) { m.Name = "Bad Plugin" }, want: "plugin name"}, {name: "duplicate permission", mutate: func(m *pluginsdk.Manifest) { m.Permissions = append(m.Permissions, m.Permissions[0]) }, want: "duplicate permission"}, {name: "self dependency", mutate: func(m *pluginsdk.Manifest) {
		m.Dependencies = []pluginsdk.Dependency{{Name: m.Name, Version: ">=1.0.0"}}
	}, want: "depend on itself"}}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			manifest := validManifest("example")
			if tc.mutate != nil {
				tc.mutate(&manifest)
			}
			err := manifest.Validate(pluginsdk.CoreVersion)
			if tc.want == "" && err != nil {
				t.Fatalf("valid manifest: %v", err)
			}
			if tc.want != "" && (err == nil || !strings.Contains(err.Error(), tc.want)) {
				t.Fatalf("expected error containing %q, got %v", tc.want, err)
			}
		})
	}
}

func TestStaticRegistrationRejectsDuplicateName(t *testing.T) {
	manifest := validManifest("registry-test")
	if err := pluginsdk.Register(noopPlugin{manifest}); err != nil {
		t.Fatal(err)
	}
	if err := pluginsdk.Register(noopPlugin{manifest}); err == nil || !strings.Contains(err.Error(), "already registered") {
		t.Fatalf("expected duplicate registration error, got %v", err)
	}
}

func TestParseVersionedYAMLManifest(t *testing.T) {
	manifest, err := pluginsdk.ParseManifest([]byte(`
api_version: airway.dev/v1
name: yaml-example
version: 1.0.0
core: ">=0.1.0 <0.2.0"
entry: yamlexample.Register
permissions:
  - code: yaml_example:read
    name: Read YAML example
dependencies: []
migrations:
  - 20260101000000_create_yaml_example
`))
	if err != nil {
		t.Fatal(err)
	}
	if manifest.Name != "yaml-example" || len(manifest.Permissions) != 1 || len(manifest.Migrations) != 1 {
		t.Fatalf("unexpected manifest: %#v", manifest)
	}
}

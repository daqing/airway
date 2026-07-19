// Package plugin defines the stable public contract between Airway and trusted,
// statically linked backend plugins.
package plugin

import (
	"context"
	"database/sql"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"go.yaml.in/yaml/v3"
)

const (
	APIVersion  = "airway.dev/v1"
	CoreVersion = "0.1.0"
)

type Manifest struct {
	APIVersion   string       `json:"api_version" yaml:"api_version"`
	Name         string       `json:"name" yaml:"name"`
	Version      string       `json:"version" yaml:"version"`
	Core         string       `json:"core" yaml:"core"`
	Entry        string       `json:"entry" yaml:"entry"`
	Permissions  []Permission `json:"permissions" yaml:"permissions"`
	Dependencies []Dependency `json:"dependencies" yaml:"dependencies"`
	Migrations   []string     `json:"migrations" yaml:"migrations"`
}

type Permission struct {
	Code string `json:"code" yaml:"code"`
	Name string `json:"name" yaml:"name"`
}
type Dependency struct {
	Name    string `json:"name" yaml:"name"`
	Version string `json:"version" yaml:"version"`
}
type Menu struct {
	Code       string `json:"code"`
	Label      string `json:"label"`
	Path       string `json:"path"`
	Permission string `json:"permission,omitempty"`
}
type Executor interface {
	ExecContext(context.Context, string, ...any) (sql.Result, error)
}
type Migration struct {
	Name string
	Up   func(context.Context, Executor) error
}

// ParseManifest decodes JSON or YAML and validates it against the current
// versioned plugin contract and core compatibility range.
func ParseManifest(data []byte) (Manifest, error) {
	var manifest Manifest
	if err := yaml.Unmarshal(data, &manifest); err != nil {
		return manifest, fmt.Errorf("decode plugin manifest: %w", err)
	}
	if err := manifest.Validate(CoreVersion); err != nil {
		return manifest, err
	}
	return manifest, nil
}

// Plugin is implemented by a trusted, statically linked plugin entry point.
type Plugin interface {
	Manifest() Manifest
	Register(Context) error
}

// Context exposes only stable, controlled contributions to plugin code.
type Context interface {
	Handle(method, path, permission string, handler gin.HandlerFunc) error
	AddPermission(permission Permission) error
	AddMenu(menu Menu) error
	AddMigration(migration Migration) error
}

var namePattern = regexp.MustCompile(`^[a-z][a-z0-9-]*$`)
var codePattern = regexp.MustCompile(`^[a-z][a-z0-9_.]*:[a-z][a-z0-9_]*$`)
var semverPattern = regexp.MustCompile(`^(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)(?:-[0-9A-Za-z.-]+)?$`)

func (m Manifest) Validate(coreVersion string) error {
	if err := m.ValidateContract(); err != nil {
		return err
	}
	compatible, err := Satisfies(coreVersion, m.Core)
	if err != nil {
		return fmt.Errorf("invalid core version constraint: %w", err)
	}
	if !compatible {
		return fmt.Errorf("plugin %s %s is incompatible with core %s", m.Name, m.Version, coreVersion)
	}
	return nil
}

// ValidateContract checks the manifest structure without enforcing the running
// core version, allowing incompatible plugins to be registered and reported.
func (m Manifest) ValidateContract() error {
	if m.APIVersion != APIVersion {
		return fmt.Errorf("unsupported manifest api_version %q", m.APIVersion)
	}
	if !namePattern.MatchString(m.Name) {
		return fmt.Errorf("invalid plugin name %q", m.Name)
	}
	if !semverPattern.MatchString(m.Version) {
		return fmt.Errorf("invalid plugin version %q", m.Version)
	}
	if strings.TrimSpace(m.Entry) == "" {
		return fmt.Errorf("plugin entry is required")
	}
	if _, err := Satisfies("0.0.0", m.Core); err != nil {
		return fmt.Errorf("invalid core version constraint: %w", err)
	}
	seenPermissions := map[string]bool{}
	for _, permission := range m.Permissions {
		if !codePattern.MatchString(permission.Code) {
			return fmt.Errorf("invalid permission code %q", permission.Code)
		}
		if strings.TrimSpace(permission.Name) == "" {
			return fmt.Errorf("permission %s name is required", permission.Code)
		}
		if seenPermissions[permission.Code] {
			return fmt.Errorf("duplicate permission %q", permission.Code)
		}
		seenPermissions[permission.Code] = true
	}
	seenDependencies := map[string]bool{}
	for _, dependency := range m.Dependencies {
		if !namePattern.MatchString(dependency.Name) {
			return fmt.Errorf("invalid dependency name %q", dependency.Name)
		}
		if dependency.Name == m.Name {
			return fmt.Errorf("plugin cannot depend on itself")
		}
		if seenDependencies[dependency.Name] {
			return fmt.Errorf("duplicate dependency %q", dependency.Name)
		}
		seenDependencies[dependency.Name] = true
		if _, err := Satisfies("0.0.0", dependency.Version); err != nil {
			return fmt.Errorf("invalid dependency %s constraint: %w", dependency.Name, err)
		}
	}
	return nil
}

type version [3]int

func parseVersion(value string) (version, error) {
	match := semverPattern.FindStringSubmatch(strings.TrimSpace(value))
	if match == nil {
		return version{}, fmt.Errorf("invalid semantic version %q", value)
	}
	var result version
	for i := 0; i < 3; i++ {
		result[i], _ = strconv.Atoi(match[i+1])
	}
	return result, nil
}
func compare(a, b version) int {
	for i := 0; i < 3; i++ {
		if a[i] < b[i] {
			return -1
		}
		if a[i] > b[i] {
			return 1
		}
	}
	return 0
}
func Satisfies(current, constraint string) (bool, error) {
	currentVersion, err := parseVersion(current)
	if err != nil {
		return false, err
	}
	tokens := strings.Fields(strings.TrimSpace(constraint))
	if len(tokens) == 0 {
		return false, fmt.Errorf("constraint is required")
	}
	for _, token := range tokens {
		operator := "="
		raw := token
		for _, candidate := range []string{">=", "<=", ">", "<", "="} {
			if strings.HasPrefix(token, candidate) {
				operator = candidate
				raw = strings.TrimSpace(strings.TrimPrefix(token, candidate))
				break
			}
		}
		required, err := parseVersion(raw)
		if err != nil {
			return false, err
		}
		comparison := compare(currentVersion, required)
		matched := map[string]bool{"=": comparison == 0, ">=": comparison >= 0, "<=": comparison <= 0, ">": comparison > 0, "<": comparison < 0}[operator]
		if !matched {
			return false, nil
		}
	}
	return true, nil
}

package cmd

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"

	"github.com/daqing/airway/cmd/clitemplate"
)

// skeletonMode says how `airway upgrade` treats one scaffolded file. The
// template mixes framework code with files users are expected to own, so
// every path is classified explicitly and a test keeps the rules in sync with
// the template.
type skeletonMode int

const (
	skeletonUnknown    skeletonMode = iota
	skeletonMerge                   // framework-owned; update through a 3-way merge
	skeletonCreate                  // create if missing, never overwrite
	skeletonRegenerate              // generated output; rebuilt by templates:compile / js:build
	skeletonNever                   // user-owned; never touched
)

type skeletonRule struct {
	pattern string
	mode    skeletonMode
}

// skeletonRules are matched in order; the first hit wins. Patterns are exact
// paths or a "dir/*" prefix.
var skeletonRules = []skeletonRule{
	{".dockerignore", skeletonMerge},
	{".env.example", skeletonMerge},
	{".gitignore", skeletonMerge},

	{"app/api/.keep", skeletonCreate},
	{"app/api/health_api/*", skeletonMerge},
	{"app/api/home_api/*", skeletonNever},
	{"app/api/openapi_api/*", skeletonMerge},
	{"app/api/storage_api/*", skeletonMerge},

	{"app/assets/assets.go", skeletonMerge},
	{"app/assets/assets_test.go", skeletonMerge},
	{"app/assets/css/airway.css", skeletonMerge},
	{"app/assets/dist/*", skeletonRegenerate},
	{"app/assets/js/app.tsx", skeletonMerge},
	{"app/assets/js/islands/*", skeletonNever},
	{"app/assets/js/ui/*", skeletonMerge},

	{"app/middlewares/.keep", skeletonCreate},
	{"app/models/.keep", skeletonCreate},
	{"app/services/.keep", skeletonCreate},

	{"app/views/home/*", skeletonNever},
	{"app/views/layouts/base.templ", skeletonMerge},
	{"app/views/layouts/base_templ.go", skeletonRegenerate},

	{"config/routes.go", skeletonMerge},
	{"config/routes_test.go", skeletonMerge},

	{"data/storage/.keep", skeletonCreate},
	{"db/migrate/doc.go", skeletonMerge},
	{"db/migrate/migrations_test.go", skeletonMerge},
	{"deps/.keep", skeletonCreate},

	{"Dockerfile", skeletonMerge},
	{"export.go", skeletonMerge},
	{"generate.go", skeletonMerge},
	{"go.mod", skeletonNever},
	{"js.pkg.json", skeletonMerge},
	{"justfile", skeletonMerge},
	{"main.go", skeletonMerge},
	{"plugins.go", skeletonMerge},
	{"Procfile.dev", skeletonMerge},

	{"README.md", skeletonNever},
	{"README.zh-CN.md", skeletonNever},

	{"static/.keep", skeletonCreate},
	{"tmp/.keep", skeletonCreate},
	{"version.go", skeletonMerge},
	{"VERSION", skeletonNever},
}

// classifySkeletonPath returns the upgrade mode for a scaffolded path (as
// produced by clitemplate.Render). ok is false when no rule covers the path.
func classifySkeletonPath(rel string) (mode skeletonMode, ok bool) {
	rel = strings.TrimPrefix(filepath.ToSlash(rel), "./")
	for _, rule := range skeletonRules {
		if skeletonPatternMatches(rule.pattern, rel) {
			return rule.mode, true
		}
	}
	return skeletonUnknown, false
}

func skeletonPatternMatches(pattern, rel string) bool {
	if prefix, ok := strings.CutSuffix(pattern, "/*"); ok {
		return strings.HasPrefix(rel, prefix+"/") && len(rel) > len(prefix)+1
	}
	return pattern == rel
}

// rewriteSkeletonPorts applies the same 1900 -> 1905 rewrite `airway new`
// performs, so a legacy project's files compare equal to their baseline.
func rewriteSkeletonPorts(files map[string][]byte) {
	for _, rel := range scaffoldPortFiles {
		if data, ok := files[rel]; ok {
			files[rel] = bytes.ReplaceAll(data, []byte("1900"), []byte("1905"))
		}
	}
}

// renderSkeletonFromDir renders a project template tree from a checkout or a
// bare template directory. The template root is cmd/clitemplate/template when
// present, otherwise dir itself.
func renderSkeletonFromDir(dir, module string) (map[string][]byte, error) {
	templateRoot := filepath.Join(dir, "cmd", "clitemplate", "template")
	if info, err := os.Stat(templateRoot); err != nil || !info.IsDir() {
		templateRoot = dir
	}

	files, err := clitemplate.RenderFS(os.DirFS(templateRoot), ".", module)
	if err != nil {
		return nil, fmt.Errorf("render template in %s: %w", templateRoot, err)
	}
	rewriteSkeletonPorts(files)
	return files, nil
}

// downloadAirwayModule resolves an Airway version to its module directory on
// disk, downloading it first when the module cache does not have it yet.
func downloadAirwayModule(projectDir, version string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), scaffoldCommandTimeout)
	defer cancel()

	mod := airwayModulePath + "@" + version
	cmd := exec.CommandContext(ctx, "go", "mod", "download", "-json", mod)
	cmd.Dir = projectDir
	cmd.Env = append(os.Environ(), "GIT_TERMINAL_PROMPT=0")

	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	out, err := cmd.Output()
	if err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			return "", fmt.Errorf("go mod download %s timed out after %s", mod, scaffoldCommandTimeout)
		}
		if msg := strings.TrimSpace(stderr.String()); msg != "" {
			return "", fmt.Errorf("go mod download %s: %v: %s", mod, err, msg)
		}
		return "", fmt.Errorf("go mod download %s: %w", mod, err)
	}

	var info struct {
		Dir   string
		Error string
	}
	if err := json.NewDecoder(bytes.NewReader(out)).Decode(&info); err != nil {
		return "", fmt.Errorf("go mod download %s: %w", mod, err)
	}
	if info.Error != "" {
		return "", fmt.Errorf("go mod download %s: %s", mod, info.Error)
	}
	if info.Dir == "" {
		return "", fmt.Errorf("go mod download %s: no module directory returned", mod)
	}

	return info.Dir, nil
}

// loadSkeletonBase renders the baseline template the project's local files
// are compared against: an explicit --from-dir, an explicit --from version, or
// the version recorded in the manifest / pinned in go.mod.
func loadSkeletonBase(projectDir, module string, opts upgradeOptions, manifestVersion, pinnedVersion string) (map[string][]byte, string, error) {
	version := ""
	switch {
	case opts.From != "":
		version = canonicalVersion(opts.From)
	case manifestVersion != "":
		version = canonicalVersion(manifestVersion)
	case pinnedVersion != "":
		version = canonicalVersion(pinnedVersion)
	}

	if opts.FromDir != "" {
		files, err := renderSkeletonFromDir(opts.FromDir, module)
		if err != nil {
			return nil, "", err
		}
		if version == "" {
			version = "unknown"
		}
		return files, version, nil
	}

	if version == "" {
		return nil, "", fmt.Errorf("cannot determine the skeleton version this project was generated with; pass --from <version> (for example --from v0.19.2)")
	}

	// A dev checkout has no published module to download; the caller falls
	// back to the manifest hashes, which still detects untouched files.
	if version == "dev" || version == "unknown" {
		return nil, version, nil
	}

	dir, err := downloadAirwayModule(projectDir, version)
	if err != nil {
		return nil, "", fmt.Errorf("load skeleton %s: %w (pass --from <version> or --from-dir <checkout> if the version is unavailable)", version, err)
	}

	files, err := renderSkeletonFromDir(dir, module)
	if err != nil {
		return nil, "", err
	}
	return files, version, nil
}

// missingEnvKeys lists keys defined in .env.example but neither set in .env
// nor present in the process environment. Upgrade never edits .env; it only
// reports what the user should add.
func missingEnvKeys(projectDir string) ([]string, error) {
	example, err := os.ReadFile(filepath.Join(projectDir, ".env.example"))
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	env, err := os.ReadFile(filepath.Join(projectDir, ".env"))
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	defined := envFileKeys(string(env))
	var missing []string
	for key := range envFileKeys(string(example)) {
		if defined[key] {
			continue
		}
		if _, ok := os.LookupEnv(key); ok {
			continue
		}
		missing = append(missing, key)
	}
	sort.Strings(missing)

	return missing, nil
}

func envFileKeys(content string) map[string]bool {
	keys := map[string]bool{}
	for line := range strings.Lines(content) {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		line = strings.TrimPrefix(line, "export ")
		key, _, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		key = strings.TrimSpace(key)
		if !validEnvKey(key) {
			continue
		}
		keys[key] = true
	}
	return keys
}

func validEnvKey(key string) bool {
	if key == "" {
		return false
	}
	for i, r := range key {
		switch {
		case r == '_':
		case r >= 'A' && r <= 'Z':
		case r >= 'a' && r <= 'z':
		case r >= '0' && r <= '9' && i > 0:
		default:
			return false
		}
	}
	return true
}

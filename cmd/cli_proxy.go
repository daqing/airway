package cmd

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

const airwayModulePath = "github.com/daqing/airway"

// ProxyHostProject re-executes project-scoped commands through `go run .`
// when the working directory is a host application, so the project's own
// binary — with its plugins, REPL models and Go-code migrations — is what
// serves the command; a globally installed `airway` cannot see any of those.
// It reports whether the call was proxied, in which case the caller must
// return right after.
//
// Before proxying, the CLI's version must match the airway version the host
// builds with; a mismatch aborts instead of silently running the project
// binary's older CLI logic.
//
// Only the entry binary calls this. It must never run inside cmd.Run: host
// projects dispatch their commands through this package too, and would
// proxy again in a loop.
func ProxyHostProject(args []string) (bool, error) {
	if len(args) == 0 || !proxyableCommand(args[0]) || !hostProjectAt(".") {
		return false, nil
	}

	if err := checkHostAirwayVersion("."); err != nil {
		return false, err
	}

	fmt.Fprintf(os.Stderr, "proxying to project binary: go run . %s\n", strings.Join(args, " "))

	run := exec.Command("go", append([]string{"run", "."}, args...)...)
	run.Stdin = os.Stdin
	run.Stdout = os.Stdout
	run.Stderr = os.Stderr

	if err := run.Run(); err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			os.Exit(exitErr.ExitCode())
		}
		return true, fmt.Errorf("go run .: %w", err)
	}

	return true, nil
}

// proxyableCommand reports whether a command concerns the current project.
// `new` scaffolds fresh projects anywhere, and version/help describe the
// globally installed tool itself; everything else acts on the project at
// hand.
func proxyableCommand(command string) bool {
	switch strings.ToLower(strings.TrimSpace(command)) {
	case "", "new", "version", "-v", "--version", "help", "-h", "--help":
		return false
	default:
		return true
	}
}

// hostProjectAt reports whether dir holds a host application: a go.mod
// requiring github.com/daqing/airway, a main.go beside it (so `go run .`
// works — plugin modules require the framework too but ship no main
// package), and a module path of its own (the framework repository runs its
// commands directly).
func hostProjectAt(dir string) bool {
	if _, err := os.Stat(filepath.Join(dir, "main.go")); err != nil {
		return false
	}

	module, err := modulePathAt(dir)
	if err != nil || module == airwayModulePath {
		return false
	}

	_, ok := goModAirwayVersion(dir)
	return ok
}

// goModAirwayVersion returns the airway version dir's go.mod requires, from
// either the single-line form or a require block. A hand-rolled scan keeps
// the CLI free of a go.mod parsing dependency.
func goModAirwayVersion(dir string) (string, bool) {
	data, err := os.ReadFile(filepath.Join(dir, "go.mod"))
	if err != nil {
		return "", false
	}

	inBlock := false
	for _, line := range strings.Split(string(data), "\n") {
		if i := strings.Index(line, "//"); i >= 0 {
			line = line[:i]
		}
		line = strings.TrimSpace(line)

		var fields []string
		switch {
		case line == "":
			continue
		case strings.HasPrefix(line, "require ("):
			inBlock = true
		case inBlock && line == ")":
			inBlock = false
		case strings.HasPrefix(line, "require "):
			fields = strings.Fields(strings.TrimPrefix(line, "require "))
		case inBlock:
			fields = strings.Fields(line)
		default:
			continue
		}

		if len(fields) >= 1 && fields[0] == airwayModulePath {
			if len(fields) >= 2 {
				return fields[1], true
			}
			return "", true
		}
	}

	return "", false
}

// goModAirwayReplace reports how dir's go.mod replaces the airway module:
// with a local directory (resolved to an absolute path) or with a module
// version. ok is false when no replace targets airway.
func goModAirwayReplace(dir string) (target string, local bool, ok bool) {
	data, err := os.ReadFile(filepath.Join(dir, "go.mod"))
	if err != nil {
		return "", false, false
	}

	inBlock := false
	for _, line := range strings.Split(string(data), "\n") {
		if i := strings.Index(line, "//"); i >= 0 {
			line = line[:i]
		}
		line = strings.TrimSpace(line)

		var body string
		switch {
		case line == "":
			continue
		case strings.HasPrefix(line, "replace ("):
			inBlock = true
			continue
		case inBlock && line == ")":
			inBlock = false
			continue
		case inBlock:
			body = line
		case strings.HasPrefix(line, "replace "):
			body = strings.TrimPrefix(line, "replace ")
		default:
			continue
		}

		fields := strings.Fields(body)
		arrow := -1
		for i, f := range fields {
			if f == "=>" {
				arrow = i
				break
			}
		}
		if arrow < 0 || fields[0] != airwayModulePath || arrow+1 >= len(fields) {
			continue
		}

		t := fields[arrow+1]
		if isLocalReplacePath(t) {
			if !filepath.IsAbs(t) {
				t = filepath.Join(dir, t)
			}
			return t, true, true
		}
		if arrow+2 < len(fields) {
			return fields[arrow+2], false, true
		}
	}

	return "", false, false
}

// isLocalReplacePath reports whether a replace target is a local directory:
// go.mod accepts absolute paths and ./ or ../ relative paths.
func isLocalReplacePath(target string) bool {
	return strings.HasPrefix(target, "/") || strings.HasPrefix(target, "./") || strings.HasPrefix(target, "../")
}

// checkHostAirwayVersion fails when the running CLI's version differs from
// the airway version the host project at dir builds with. Proxied commands
// are served by the project binary, so a stale pin would silently run older
// CLI logic than the globally installed tool. A replace onto a local
// directory overrides the pin: its VERSION file decides when present, and
// without one there is nothing to compare. A dev build without a version
// skips the check.
func checkHostAirwayVersion(dir string) error {
	global := normalizeVersion(Version)
	if global == "" || global == "dev" {
		return nil
	}

	pinned, ok := goModAirwayVersion(dir)
	if !ok || pinned == "" {
		return nil
	}

	if target, local, replaced := goModAirwayReplace(dir); replaced {
		if !local {
			pinned = target
		} else if data, err := os.ReadFile(filepath.Join(target, "VERSION")); err == nil && normalizeVersion(string(data)) != "" {
			pinned = string(data)
		} else {
			// A local-directory replace without a VERSION file pins
			// nothing comparable.
			return nil
		}
	}

	pinned = normalizeVersion(pinned)
	if global == pinned {
		return nil
	}

	return fmt.Errorf(`airway version mismatch

- global airway: v%s
- this project: github.com/daqing/airway v%s

Align them before running project commands:

- go get github.com/daqing/airway@v%s (in the project)
- go install github.com/daqing/airway@v%s (for the global command)
- go mod edit -replace github.com/daqing/airway=/path/to/local/airway`,
		global, pinned, global, pinned,
	)
}

// normalizeVersion trims whitespace and the optional leading v, so VERSION
// files and go.mod requires compare equal.
func normalizeVersion(v string) string {
	return strings.TrimPrefix(strings.TrimPrefix(strings.TrimSpace(v), "v"), "V")
}

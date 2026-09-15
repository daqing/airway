package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"text/template"
	"time"
)

var timeNow = time.Now

// currentModulePath reads the module path of the project in the current
// working directory, so generated code imports the project's own packages
// (e.g. <module>/app/models) instead of the framework's. Falls back to the
// framework module path when go.mod is missing or malformed.
func currentModulePath() string {
	const fallback = "github.com/daqing/airway"

	data, err := os.ReadFile("go.mod")
	if err != nil {
		return fallback
	}

	for line := range strings.Lines(string(data)) {
		if module, ok := strings.CutPrefix(strings.TrimSpace(line), "module "); ok {
			if module = strings.TrimSpace(module); module != "" {
				return module
			}
		}
	}

	return fallback
}

func ensureDir(path string) error {
	return os.MkdirAll(path, 0o755)
}

func writeTemplateFile(tplText string, out string, data any) error {
	if _, err := os.Stat(out); err == nil {
		return os.ErrExist
	} else if !os.IsNotExist(err) {
		return err
	}

	if err := ensureDir(filepath.Dir(out)); err != nil {
		return err
	}

	tpl, err := template.New(filepath.Base(out)).Parse(tplText)
	if err != nil {
		return err
	}

	file, err := os.OpenFile(out, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	defer file.Close()

	return tpl.Execute(file, data)
}

func parsePositiveInt(value string) (int, error) {
	n, err := strconv.Atoi(value)
	if err != nil {
		return 0, err
	}
	if n < 0 {
		return 0, fmt.Errorf("must be non-negative")
	}
	return n, nil
}

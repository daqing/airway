package main

import (
	"fmt"
	"os"
	"text/template"
)

func pathExists(path string) bool {
	if _, err := os.Stat(path); err == nil {
		return true
	}

	return false
}

// Execute the data with template from path and write to out
func ExecTemplate(path string, out string, data any) error {
	if pathExists(out) {
		return fmt.Errorf("target file %s already exists", out)
	}

	if !pathExists(path) {
		return fmt.Errorf("template %s does not exist", path)
	}

	txt, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	tmpl, err := template.New("test").Parse(string(txt))
	if err != nil {
		return err
	}

	outputFile, err := os.Create(out)
	if err != nil {
		panic(err)
	}

	defer outputFile.Close()

	return tmpl.Execute(outputFile, data)
}
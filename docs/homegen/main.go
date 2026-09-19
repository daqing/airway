//go:build ignore

// Renders the templ landing page (app/views/home) to static HTML and
// overwrites the VitePress-built homepage for GitHub Pages.
//
//	go run docs/homegen/main.go [output]
//
// The default output is docs/.vitepress/dist/index.html, so run it after
// `npm run docs:build`.
package main

import (
	"context"
	"fmt"
	"os"

	"github.com/daqing/airway/app/views/home"
)

func main() {
	output := "docs/.vitepress/dist/index.html"
	if len(os.Args) > 1 {
		output = os.Args[1]
	}

	f, err := os.Create(output)
	if err != nil {
		fmt.Fprintf(os.Stderr, "create %s: %v\n", output, err)
		os.Exit(1)
	}
	defer f.Close()

	if err := home.Index(3).Render(context.Background(), f); err != nil {
		fmt.Fprintf(os.Stderr, "render home page: %v\n", err)
		os.Exit(1)
	}
}

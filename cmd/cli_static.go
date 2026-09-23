package cmd

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/daqing/airway/lib/jsbuild"
	"github.com/daqing/airway/lib/repo"
	"github.com/daqing/airway/lib/static"
	"github.com/daqing/airway/lib/utils"
)

// staticPages holds the app pages exported by `airway static:build`. Host
// projects register them through SetStaticPages from an init function in
// the project's root package (an export.go file) — the same pattern as
// plugins, REPL models, and the ssg site builder.
var staticPages []static.Page

// StaticPagesProvider returns pages discovered at export time — e.g.
// detail pages enumerated from the database, one per row. Providers run
// only when static:build or static:serve collects pages, never at process
// start, so registering one never triggers I/O in other commands.
type StaticPagesProvider func() ([]static.Page, error)

// staticProviders holds the registered providers, in registration order.
var staticProviders []StaticPagesProvider

// SetStaticPages registers the pages that `airway static:build` exports.
// Page components render without a request, so keep request-time data out
// of them: bake build-time data into the component instead of querying the
// database, which is not available during export.
func SetStaticPages(pages ...static.Page) {
	seen := make(map[string]bool, len(pages))

	for _, page := range pages {
		page.Slug = static.NormalizeSlug(page.Slug)

		if seen[page.Slug] {
			panic(fmt.Sprintf("static: duplicate page slug %q", page.Slug))
		}
		seen[page.Slug] = true

		staticPages = append(staticPages, page)
	}
}

// SetStaticPagesProvider registers a provider that contributes pages at
// export time — one call per generated resource or dynamic route family.
// Enumerating rows from the database belongs in a provider, not in
// SetStaticPages: init functions run for every command (including
// `--version`, which must not touch the environment), while providers run
// only inside static:build/static:serve.
func SetStaticPagesProvider(provider StaticPagesProvider) {
	staticProviders = append(staticProviders, provider)
}

// projectStaticPages collects the registered pages plus everything the
// providers contribute. A panicking provider — for example repo queries
// hit before the database is set up — becomes an error instead of a crash.
func projectStaticPages() ([]static.Page, error) {
	pages := staticPages

	for _, provider := range staticProviders {
		provided, err := runStaticPagesProvider(provider)
		if err != nil {
			return nil, err
		}
		pages = append(pages, provided...)
	}

	if len(pages) == 0 {
		return nil, fmt.Errorf("no static pages are defined in this project; add an export.go in the project root that calls cmd.SetStaticPages (see docs/static-export.md)")
	}

	return pages, nil
}

func runStaticPagesProvider(provider StaticPagesProvider) (provided []static.Page, err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("static pages provider failed: %v", r)
		}
	}()

	return provider()
}

// setupExportDB prepares the database for providers that enumerate rows at
// export time. Without a DSN it does nothing — database-free sites export
// anywhere; a configured but unreachable database fails the command, since
// data-driven pages cannot be rendered without it.
func setupExportDB() error {
	dsn, err := cliDSN()
	if err != nil {
		return nil
	}

	if _, err := repo.SetupDB(dsn); err != nil {
		return fmt.Errorf("database setup failed: %w", err)
	}

	return nil
}

// pinProductionEnv forces production asset URLs for export and preview:
// local development renders in-memory livereload URLs that would 404 on a
// static host, and the developer's AIRWAY_ENV (often local via .env) must
// not change what gets exported.
func pinProductionEnv() {
	os.Setenv("AIRWAY_ENV", "production")
}

func runCLIStaticBuild(args []string) error {
	out := "dist"

	for i := 0; i < len(args); i++ {
		arg := args[i]
		switch {
		case isHelpArg(arg):
			printCLIStaticBuildUsage(os.Stdout)
			return nil
		case arg == "--out" && i+1 < len(args):
			out = args[i+1]
			i++
		case strings.HasPrefix(arg, "--out="):
			out = strings.TrimPrefix(arg, "--out=")
		default:
			return fmt.Errorf("unknown argument %q; usage: airway static:build [--out dist]", arg)
		}
	}

	if out == "" {
		return fmt.Errorf("--out must not be empty")
	}

	pinProductionEnv()

	if err := setupExportDB(); err != nil {
		return err
	}

	pages, err := projectStaticPages()
	if err != nil {
		return err
	}

	entry := filepath.Join(jsbuild.DistDir, jsbuild.EntryJS)
	if _, err := os.Stat(entry); err != nil {
		return fmt.Errorf("%s is missing; run `airway js:build` first so the static site ships a frontend bundle", entry)
	}

	if err := static.Build(pages, out); err != nil {
		return err
	}

	if err := utils.CopyFS(os.DirFS(jsbuild.DistDir), filepath.Join(out, static.AssetDir)); err != nil {
		return fmt.Errorf("copy frontend bundle: %w", err)
	}

	fmt.Printf("Built %d static page(s) into %s\n", len(pages), out)
	return nil
}

func runCLIStaticServe(args []string) error {
	addr := "127.0.0.1:3000"

	for i := 0; i < len(args); i++ {
		arg := args[i]
		switch {
		case isHelpArg(arg):
			printCLIStaticServeUsage(os.Stdout)
			return nil
		case arg == "--addr" && i+1 < len(args):
			addr = args[i+1]
			i++
		case strings.HasPrefix(arg, "--addr="):
			addr = strings.TrimPrefix(arg, "--addr=")
		default:
			return fmt.Errorf("unknown argument %q; usage: airway static:serve [--addr 127.0.0.1:3000]", arg)
		}
	}

	pinProductionEnv()

	if err := setupExportDB(); err != nil {
		return err
	}

	pages, err := projectStaticPages()
	if err != nil {
		return err
	}

	fmt.Printf("Serving %d static page(s) on http://%s (press Ctrl-C to stop)\n", len(pages), addr)

	mux := http.NewServeMux()
	// Serve the bundle that static:build would export, so the preview is
	// fully interactive without a prior export.
	prefix := "/" + static.AssetDir + "/"
	mux.Handle("GET "+prefix, http.StripPrefix(prefix, http.FileServer(http.Dir(jsbuild.DistDir))))
	mux.Handle("/", static.Handler(pages))

	return http.ListenAndServe(addr, mux)
}

func printCLIStaticBuildUsage(w *os.File) {
	_, _ = fmt.Fprintln(w, "usage:")
	_, _ = fmt.Fprintln(w, "  airway static:build [--out dist]    export the pages registered via cmd.SetStaticPages as static HTML")
}

func printCLIStaticServeUsage(w *os.File) {
	_, _ = fmt.Fprintln(w, "usage:")
	_, _ = fmt.Fprintln(w, "  airway static:serve [--addr 127.0.0.1:3000]    preview the static pages with a local server")
}

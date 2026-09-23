package cmd

import (
	"errors"
	"fmt"
	"io/fs"
	"net/http"
	"os"
	"os/exec"
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

// rebuildFrontendBundle refreshes app/assets/dist from app/assets/js before
// exporting, so frontend edits are always reflected. Projects without
// frontend sources keep using the committed bundle as-is; a missing vendor
// directory degrades to the committed bundle with a warning, while a real
// build failure aborts the export instead of shipping stale assets.
func rebuildFrontendBundle() error {
	if _, err := os.Stat(filepath.Join(jsbuild.SourceDir, "app.tsx")); err != nil {
		return nil
	}

	if _, err := jsbuild.Build("."); err != nil {
		if errors.Is(err, jsbuild.ErrVendorMissing) {
			fmt.Fprintf(os.Stderr, "warning: %v; exporting the committed %s instead\n", err, jsbuild.DistDir)
			return nil
		}
		return fmt.Errorf("frontend bundle: %w", err)
	}

	return nil
}

// staticReexecEnv marks an already re-executed static:build/static:serve, so
// the templ refresh never loops.
const staticReexecEnv = "AIRWAY_STATIC_REEXEC"

// ensureFreshTemplViews makes edited .templ sources usable for the export: it
// regenerates them (the templates:compile step) and then re-executes the
// command, because the running process can only carry the previously compiled
// views — regenerating alone would still export stale pages. It reports
// whether the caller should stop, the re-executed run having taken over.
func ensureFreshTemplViews(command string, args []string) (bool, error) {
	stale, err := countStaleTemplViews(filepath.Join("app", "views"))
	if err != nil || stale == 0 {
		return false, nil
	}

	if os.Getenv(staticReexecEnv) != "" {
		// Regenerating did not clear the staleness (for example generated
		// files templ does not rewrite); export what the binary has instead
		// of looping.
		fmt.Fprintf(os.Stderr, "warning: %d templ view(s) still look stale after regenerating\n", stale)
		return false, nil
	}

	fmt.Fprintf(os.Stderr, "%d templ view(s) changed; regenerating them first\n", stale)

	if err := generateTemplViews(); err != nil {
		return false, fmt.Errorf("regenerate templ views: %w", err)
	}

	if err := reexecCommand(command, args); err != nil {
		return false, fmt.Errorf("re-run %s with the fresh views: %w", command, err)
	}

	return true, nil
}

// reexecCommand runs `go run . <command> <args>` in the project, so the
// freshly generated views are compiled in before the command runs again. The
// environment marker keeps the child from re-executing in turn.
func reexecCommand(command string, args []string) error {
	run := exec.Command("go", append([]string{"run", ".", command}, args...)...)
	run.Env = append(os.Environ(), staticReexecEnv+"=1")
	run.Stdin = os.Stdin
	run.Stdout = os.Stdout
	run.Stderr = os.Stderr
	return run.Run()
}

func countStaleTemplViews(dir string) (int, error) {
	stale := 0

	err := filepath.WalkDir(dir, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() || !strings.HasSuffix(path, ".templ") {
			return nil
		}

		info, err := entry.Info()
		if err != nil {
			return err
		}

		generated := strings.TrimSuffix(path, ".templ") + "_templ.go"
		genInfo, err := os.Stat(generated)
		if err != nil {
			if os.IsNotExist(err) {
				stale++
				return nil
			}
			return err
		}

		if info.ModTime().After(genInfo.ModTime()) {
			stale++
		}
		return nil
	})

	return stale, err
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

	reexecuted, err := ensureFreshTemplViews("static:build", args)
	if err != nil {
		return err
	}
	if reexecuted {
		return nil
	}

	if err := rebuildFrontendBundle(); err != nil {
		return err
	}

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

	reexecuted, err := ensureFreshTemplViews("static:serve", args)
	if err != nil {
		return err
	}
	if reexecuted {
		return nil
	}

	if err := rebuildFrontendBundle(); err != nil {
		return err
	}

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

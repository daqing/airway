package cmd

import (
	"fmt"
	"net/http"
	"os"
	"strings"

	"github.com/daqing/airway/lib/ssg"
)

// ssgBuilder holds the project's site definition. Host projects register it
// through SetSSGBuilder from an init function in the project's root package
// (a ssg.go file) — the same pattern plugins and REPL models use, so the
// project binary carries its own site code.
var ssgBuilder ssg.Builder

// SetSSGBuilder registers the builder that constructs the project's site
// definition for `airway ssg:build` and `airway ssg:serve`.
func SetSSGBuilder(builder ssg.Builder) {
	ssgBuilder = builder
}

// projectSSG invokes the registered builder.
func projectSSG() (*ssg.Site, error) {
	if ssgBuilder == nil {
		return nil, fmt.Errorf("no site is defined in this project; add a ssg.go in the project root that calls cmd.SetSSGBuilder (see docs/ssg.md)")
	}

	return ssgBuilder()
}

func runCLISSGBuild(args []string) error {
	out := "dist"

	for i := 0; i < len(args); i++ {
		arg := args[i]
		switch {
		case isHelpArg(arg):
			printCLISSGBuildUsage(os.Stdout)
			return nil
		case arg == "--out" && i+1 < len(args):
			out = args[i+1]
			i++
		case strings.HasPrefix(arg, "--out="):
			out = strings.TrimPrefix(arg, "--out=")
		default:
			return fmt.Errorf("unknown argument %q; usage: airway ssg:build [--out dist]", arg)
		}
	}

	if out == "" {
		return fmt.Errorf("--out must not be empty")
	}

	s, err := projectSSG()
	if err != nil {
		return err
	}

	if err := ssg.Build(s, out); err != nil {
		return err
	}

	fmt.Printf("Built %d page(s) into %s\n", len(s.Pages()), out)
	return nil
}

func runCLISSGServe(args []string) error {
	addr := "127.0.0.1:3000"

	for i := 0; i < len(args); i++ {
		arg := args[i]
		switch {
		case isHelpArg(arg):
			printCLISSGServeUsage(os.Stdout)
			return nil
		case arg == "--addr" && i+1 < len(args):
			addr = args[i+1]
			i++
		case strings.HasPrefix(arg, "--addr="):
			addr = strings.TrimPrefix(arg, "--addr=")
		default:
			return fmt.Errorf("unknown argument %q; usage: airway ssg:serve [--addr 127.0.0.1:3000]", arg)
		}
	}

	s, err := projectSSG()
	if err != nil {
		return err
	}

	fmt.Printf("Serving %d page(s) on http://%s (press Ctrl-C to stop)\n", len(s.Pages()), addr)
	return http.ListenAndServe(addr, ssg.Handler(s))
}

func printCLISSGBuildUsage(w *os.File) {
	_, _ = fmt.Fprintln(w, "usage:")
	_, _ = fmt.Fprintln(w, "  airway ssg:build [--out dist]    export the site defined in ssg.go as static HTML")
}

func printCLISSGServeUsage(w *os.File) {
	_, _ = fmt.Fprintln(w, "usage:")
	_, _ = fmt.Fprintln(w, "  airway ssg:serve [--addr 127.0.0.1:3000]    preview the site with a local server")
}

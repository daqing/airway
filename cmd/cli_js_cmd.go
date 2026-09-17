package cmd

import (
	"fmt"
	"io"
	"log"
	"os"

	"github.com/daqing/airway/lib/jsbuild"
	"github.com/daqing/airway/lib/jspkg"
	"github.com/daqing/airway/lib/utils"
)

func runCLIJsAdd(args []string) error {
	if len(args) != 1 || isHelpArg(args[0]) {
		printCLIJsAddUsage(os.Stdout)
		return nil
	}

	return jspkg.Add(".", args[0], jspkg.Options{
		Registry: jsRegistry(),
		Log:      log.Printf,
	})
}

func runCLIJsInstall(args []string) error {
	if len(args) > 0 && isHelpArg(args[0]) {
		printCLIJsInstallUsage(os.Stdout)
		return nil
	}

	return jspkg.Install(".", jspkg.Options{
		Registry: jsRegistry(),
		Log:      log.Printf,
	})
}

func runCLIJsBuild(args []string) error {
	if len(args) > 0 && isHelpArg(args[0]) {
		printCLIJsBuildUsage(os.Stdout)
		return nil
	}

	m, err := jsbuild.Build(".")
	if err != nil {
		return err
	}
	log.Printf("built %s/%s: %d file(s), hash %s", jsbuild.DistDir, jsbuild.EntryJS, len(m.Files), m.Hash)
	return nil
}

// jsRegistry resolves the registry URL: AIRWAY_JS_REGISTRY (or short
// JS_REGISTRY) first, then the npm default.
func jsRegistry() string {
	if v := utils.GetEnvMulti("AIRWAY_JS_REGISTRY", "JS_REGISTRY"); v != "" {
		return v
	}
	return jspkg.DefaultRegistry
}

func printCLIJsAddUsage(w io.Writer) {
	_, _ = fmt.Fprintln(w, "usage:")
	_, _ = fmt.Fprintln(w, "  airway js:add <pkg>[@version]   add an npm dependency at an exact version")
	_, _ = fmt.Fprintln(w, "")
	_, _ = fmt.Fprintln(w, "examples:")
	_, _ = fmt.Fprintln(w, "  airway js:add preact")
	_, _ = fmt.Fprintln(w, "  airway js:add @tanstack/react-table@9.2.4")
}

func printCLIJsInstallUsage(w io.Writer) {
	_, _ = fmt.Fprintln(w, "usage:")
	_, _ = fmt.Fprintln(w, "  airway js:install   install js.pkg.json dependencies into app/assets/js/vendor/")
}

func printCLIJsBuildUsage(w io.Writer) {
	_, _ = fmt.Fprintln(w, "usage:")
	_, _ = fmt.Fprintln(w, "  airway js:build   bundle app/assets/js/app.tsx into app/assets/dist (esbuild, no Node)")
}

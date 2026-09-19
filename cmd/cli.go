package cmd

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/daqing/airway/lib/utils"
)

func runCLI(args []string) error {
	if len(args) == 0 {
		printUsage(os.Stdout)
		return nil
	}

	command := strings.ToLower(strings.TrimSpace(args[0]))
	xargs := args[1:]

	switch command {
	case "new":
		return runCLINew(xargs)
	case "generate", "g":
		return runCLIGenerate(xargs)
	case "db:migrate":
		return runCLIMigrate(xargs)
	case "db:rollback":
		return runCLIRollback(xargs)
	case "db:status":
		return runCLIStatus(xargs)
	case "schema:show":
		return runCLISchemaShow(xargs)
	case "schema:dump":
		return runCLISchemaDump(xargs)
	case "schema":
		return runCLISchema(xargs)
	case "openapi:generate":
		return runCLIOpenAPIGenerate(xargs)
	case "db:drop":
		return runCLIDBDrop(xargs)
	case "db:create":
		return runCLIDBCreate(xargs)
	case "plugin":
		return runCLIPlugin(xargs)
	case "plugin:new":
		return runCLIPluginNew(xargs)
	case "plugin:list":
		return runCLIPluginList()
	case "plugin:install":
		return runCLIPluginInstall(xargs)
	case "plugin:lint":
		return runCLIPluginLint(xargs)
	case "js:add":
		return runCLIJsAdd(xargs)
	case "js:install":
		return runCLIJsInstall(xargs)
	case "js:build":
		return runCLIJsBuild(xargs)
	case "upload":
		return runUpload(xargs)
	case "help", "-h", "--help":
		printUsage(os.Stdout)
		return nil
	default:
		return fmt.Errorf("unknown command: %s", command)
	}
}

func cliDSN() (string, error) {
	// Prefer the same scheme as the server: AIRWAY_DSN, then the short DSN.
	// AIRWAY_DB_DSN and AIRWAY_PG remain supported for backward compatibility.
	for _, key := range []string{"AIRWAY_DSN", "DSN", "AIRWAY_DB_DSN", "AIRWAY_PG"} {
		if dsn, err := utils.GetEnv(key); err == nil {
			return dsn, nil
		}
	}

	return "", fmt.Errorf("database dsn is not configured; set DSN (or AIRWAY_DSN)")
}

func printUsage(w io.Writer) {
	_, _ = fmt.Fprintln(w, "usage:")
	_, _ = fmt.Fprintln(w, "  airway new <module-path>                 create a new Airway project")
	_, _ = fmt.Fprintln(w, "  airway server                            start the HTTP server")
	_, _ = fmt.Fprintln(w, "  airway generate [action|api|model|migration|service|cmd] [params]")
	_, _ = fmt.Fprintln(w, "  airway db:create")
	_, _ = fmt.Fprintln(w, "  airway db:drop")
	_, _ = fmt.Fprintln(w, "  airway db:migrate [version]")
	_, _ = fmt.Fprintln(w, "  airway db:rollback [step]")
	_, _ = fmt.Fprintln(w, "  airway db:status")
	_, _ = fmt.Fprintln(w, "  airway schema:dump")
	_, _ = fmt.Fprintln(w, "  airway schema:show")
	_, _ = fmt.Fprintln(w, "  airway openapi:generate [--out path]     write the OpenAPI 3.2 document (default ./openapi.json)")
	_, _ = fmt.Fprintln(w, "  airway plugin:new <module-path>")
	_, _ = fmt.Fprintln(w, "  airway plugin:list")
	_, _ = fmt.Fprintln(w, "  airway plugin:install <module>")
	_, _ = fmt.Fprintln(w, "  airway plugin:lint")
	_, _ = fmt.Fprintln(w, "  airway js:add <pkg>[@version]       add a frontend dependency (no Node required)")
	_, _ = fmt.Fprintln(w, "  airway js:install                   install frontend dependencies from js.pkg.json")
	_, _ = fmt.Fprintln(w, "  airway upload [key] /path/to/file")
	_, _ = fmt.Fprintln(w, "  airway repl                              interactive repo REPL (runs through go run . inside a project)")
	_, _ = fmt.Fprintln(w, "  airway version                             print version (also -v, --version)")
	_, _ = fmt.Fprintln(w, "")
	_, _ = fmt.Fprintln(w, "`airway <command>` remains accepted as an alias for `airway <command>`.")
}

func printCLIGenerateUsage(w io.Writer) {
	_, _ = fmt.Fprintln(w, "usage:")
	_, _ = fmt.Fprintln(w, "  airway generate [action|api|model|migration|service|cmd] [params]")
}

func printCLIGenerateMigrationUsage(w io.Writer) {
	_, _ = fmt.Fprintln(w, "usage:")
	_, _ = fmt.Fprintln(w, "  airway generate migration [name]")
}

func printCLIGenerateActionUsage(w io.Writer) {
	_, _ = fmt.Fprintln(w, "usage:")
	_, _ = fmt.Fprintln(w, "  airway generate action [api] [action]")
}

func printCLIGenerateAPIUsage(w io.Writer) {
	_, _ = fmt.Fprintln(w, "usage:")
	_, _ = fmt.Fprintln(w, "  airway generate api [name]")
}

func printCLIGenerateModelUsage(w io.Writer) {
	_, _ = fmt.Fprintln(w, "usage:")
	_, _ = fmt.Fprintln(w, "  airway generate model [name] [field:type]...")
}

func printCLIGenerateServiceUsage(w io.Writer) {
	_, _ = fmt.Fprintln(w, "usage:")
	_, _ = fmt.Fprintln(w, "  airway generate service <name> <field:type> <field:type>...")
}

func printCLIGenerateCmdUsage(w io.Writer) {
	_, _ = fmt.Fprintln(w, "usage:")
	_, _ = fmt.Fprintln(w, "  airway generate cmd <name> <field> <field>...")
}

func isHelpArg(value string) bool {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "help", "-h", "--help":
		return true
	default:
		return false
	}
}

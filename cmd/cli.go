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
	case "admin:generate":
		return generateAdmin(xargs)
	case "admin:root":
		return runAdminRoot(xargs)
	case "admin:member":
		return runAdminMember(xargs)
	case "desktop:init":
		return runCLIDesktopInit(xargs)
	case "ssg:new":
		return runCLISSGNew(xargs)
	case "ssg:build":
		return runCLISSGBuild(xargs)
	case "ssg:serve":
		return runCLISSGServe(xargs)
	case "theme:install":
		return runCLIThemeInstall(xargs)
	case "theme:new":
		return runCLIThemeNew(xargs)
	case "templates:compile":
		return runTemplatesCompile(xargs)
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
	_, _ = fmt.Fprintln(w, "  airway generate [action|api|model|migration|service|island|scaffold|cmd] [params]")
	_, _ = fmt.Fprintln(w, "  airway db:create")
	_, _ = fmt.Fprintln(w, "  airway db:drop")
	_, _ = fmt.Fprintln(w, "  airway db:migrate [version]")
	_, _ = fmt.Fprintln(w, "  airway db:rollback [step]")
	_, _ = fmt.Fprintln(w, "  airway db:status")
	_, _ = fmt.Fprintln(w, "  airway schema:dump")
	_, _ = fmt.Fprintln(w, "  airway schema:show")
	_, _ = fmt.Fprintln(w, "  airway openapi:generate [--out path]     write the OpenAPI 3.2 document (default ./openapi.json)")
	_, _ = fmt.Fprintln(w, "  airway admin:generate [config/admin.toml]  generate the admin backend from a TOML table spec")
	_, _ = fmt.Fprintln(w, "  airway admin:root <username> <password>   create the administrator account (role admin)")
	_, _ = fmt.Fprintln(w, "  airway admin:member <username> <password> [--role=editor|viewer]")
	_, _ = fmt.Fprintln(w, "                                           create a non-admin panel account")
	_, _ = fmt.Fprintln(w, "  airway desktop:init [--force]            generate the Wails v3 desktop target in ./desktop")
	_, _ = fmt.Fprintln(w, "  airway ssg:new [--local[=path]] <name>   scaffold a static showcase site project")
	_, _ = fmt.Fprintln(w, "  airway ssg:build [--out dist]           export the site as static HTML (ssg.go)")
	_, _ = fmt.Fprintln(w, "  airway ssg:serve [--addr 127.0.0.1:3000]  preview the site with a local server")
	_, _ = fmt.Fprintln(w, "  airway theme:install <module | /path>    install a site theme into the host project")
	_, _ = fmt.Fprintln(w, "  airway theme:new [--local[=path]] <name> scaffold a new site theme module")
	_, _ = fmt.Fprintln(w, "  airway templates:compile                 regenerate the templ views (shorthand for `go generate ./...`)")
	_, _ = fmt.Fprintln(w, "  airway plugin:new <module-path>")
	_, _ = fmt.Fprintln(w, "  airway plugin:list")
	_, _ = fmt.Fprintln(w, "  airway plugin:install <module>")
	_, _ = fmt.Fprintln(w, "  airway plugin:lint")
	_, _ = fmt.Fprintln(w, "  airway js:add <pkg>[@version]       add a frontend dependency (no Node required)")
	_, _ = fmt.Fprintln(w, "  airway js:install                   install frontend dependencies from js.pkg.json")
	_, _ = fmt.Fprintln(w, "  airway js:build                     bundle app/assets/js into app/assets/dist (esbuild)")
	_, _ = fmt.Fprintln(w, "  airway upload [key] /path/to/file")
	_, _ = fmt.Fprintln(w, "  airway repl                              interactive repo REPL (runs through go run . inside a project)")
	_, _ = fmt.Fprintln(w, "  airway version                             print version (also -v, --version)")
	_, _ = fmt.Fprintln(w, "")
	_, _ = fmt.Fprintln(w, "`airway cli <command>` remains accepted as an alias for `airway <command>`.")
}

func printCLIGenerateUsage(w io.Writer) {
	_, _ = fmt.Fprintln(w, "usage:")
	_, _ = fmt.Fprintln(w, "  airway generate [action|api|model|migration|service|cmd|island|scaffold] [params]")
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

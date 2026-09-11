package cmd

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
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
	case "db:drop":
		return runCLIDBDrop(xargs)
	case "db:create":
		return runCLIDBCreate(xargs)
	case "engine":
		return runCLIEngine(xargs)
	case "engine:list":
		return runCLIEngineList()
	case "engine:install":
		return runCLIEngineInstall(xargs)
	case "plugin", "plugin:install":
		return runCLIPlugin(command, xargs)
	case "upload":
		return runUpload(xargs)
	case "help", "-h", "--help":
		printUsage(os.Stdout)
		return nil
	default:
		return fmt.Errorf("unknown command: %s", command)
	}
}

func runCLIPlugin(command string, args []string) error {
	if command == "plugin" {
		if len(args) == 0 {
			return fmt.Errorf("usage: airway plugin install /path/to/project")
		}

		subcommand := strings.ToLower(strings.TrimSpace(args[0]))
		if subcommand != "install" {
			return fmt.Errorf("unknown plugin command: %s", subcommand)
		}

		args = args[1:]
	}

	if len(args) != 1 {
		return fmt.Errorf("usage: airway plugin install /path/to/project")
	}

	fmt.Println("WARNING: `cli plugin install` is deprecated and will be removed in a future release.")
	fmt.Println("Use engines instead: ship the module as a Go module and enable it with a blank import in engines.go (see docs/engine.md).")
	fmt.Println("Install current plugin to", args[0])
	return installPlugin(args[0], timeNow().Format("20060102150405"))
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
	_, _ = fmt.Fprintln(w, "  airway engine new <module-path>")
	_, _ = fmt.Fprintln(w, "  airway engine:list")
	_, _ = fmt.Fprintln(w, "  airway engine:install [name]")
	_, _ = fmt.Fprintln(w, "  airway upload [key] /path/to/file")
	_, _ = fmt.Fprintln(w, "  airway repl                              interactive repo REPL (project binary only)")
	_, _ = fmt.Fprintln(w, "  airway version")
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

func installPlugin(projectPath string, timestamp string) error {
	projectPath = strings.TrimSpace(projectPath)
	if projectPath == "" {
		return fmt.Errorf("project path must not be empty")
	}

	absProjectPath, err := filepath.Abs(projectPath)
	if err != nil {
		return err
	}

	for _, relativeDir := range []string{"app", "cmd"} {
		srcDir := filepath.Join(".", relativeDir)
		dstDir := filepath.Join(absProjectPath, relativeDir)
		if err := copyDirContents(srcDir, dstDir); err != nil {
			return fmt.Errorf("copy %s to %s: %w", srcDir, dstDir, err)
		}
	}

	srcMigrateDir := filepath.Join(".", "db", "migrate")
	dstMigrateDir := filepath.Join(absProjectPath, "db", "migrate")
	if err := copyMigrationFiles(srcMigrateDir, dstMigrateDir, timestamp); err != nil {
		return fmt.Errorf("copy migrations to %s: %w", dstMigrateDir, err)
	}

	return nil
}

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
		printCLIUsage(os.Stdout)
		return nil
	}

	command := strings.ToLower(strings.TrimSpace(args[0]))
	xargs := args[1:]

	switch command {
	case "generate", "g":
		return runCLIGenerate(xargs)
	case "migrate":
		return runCLIMigrate(xargs)
	case "rollback":
		return runCLIRollback(xargs)
	case "status":
		return runCLIStatus(xargs)
	case "schema:show":
		return runCLISchemaShow(xargs)
	case "schema:dump":
		return runCLISchemaDump(xargs)
	case "schema":
		return runCLISchema(xargs)
	case "plugin", "plugin:install":
		return runCLIPlugin(command, xargs)
	case "help", "-h", "--help":
		printCLIUsage(os.Stdout)
		return nil
	default:
		return fmt.Errorf("unknown cli command: %s", command)
	}
}

func runCLIPlugin(command string, args []string) error {
	if command == "plugin" {
		if len(args) == 0 {
			return fmt.Errorf("usage: airway cli plugin install /path/to/project")
		}

		subcommand := strings.ToLower(strings.TrimSpace(args[0]))
		if subcommand != "install" {
			return fmt.Errorf("unknown plugin command: %s", subcommand)
		}

		args = args[1:]
	}

	if len(args) != 1 {
		return fmt.Errorf("usage: airway cli plugin install /path/to/project")
	}

	fmt.Println("Install current plugin to", args[0])
	return installPlugin(args[0], timeNow().Format("20060102150405"))
}

func cliDSN() (string, error) {
	if dsn, err := utils.GetEnv("AIRWAY_DB_DSN"); err == nil {
		return dsn, nil
	}

	if dsn, err := utils.GetEnv("AIRWAY_PG"); err == nil {
		return dsn, nil
	}

	return "", fmt.Errorf("database dsn is not configured; set AIRWAY_DB_DSN (preferred) or AIRWAY_PG")
}

func printCLIUsage(w io.Writer) {
	_, _ = fmt.Fprintln(w, "usage:")
	_, _ = fmt.Fprintln(w, "  airway cli generate [action|api|model|migration|service|cmd] [params]")
	_, _ = fmt.Fprintln(w, "  airway cli migrate [version]")
	_, _ = fmt.Fprintln(w, "  airway cli rollback [step]")
	_, _ = fmt.Fprintln(w, "  airway cli status")
	_, _ = fmt.Fprintln(w, "  airway cli schema:show")
	_, _ = fmt.Fprintln(w, "  airway cli schema:dump")
	_, _ = fmt.Fprintln(w, "  airway cli plugin install /path/to/project")
}

func printCLIGenerateUsage(w io.Writer) {
	_, _ = fmt.Fprintln(w, "usage:")
	_, _ = fmt.Fprintln(w, "  airway cli generate [action|api|model|migration|service|cmd] [params]")
}

func printCLIGenerateMigrationUsage(w io.Writer) {
	_, _ = fmt.Fprintln(w, "usage:")
	_, _ = fmt.Fprintln(w, "  airway cli generate migration [name]")
}

func printCLIGenerateActionUsage(w io.Writer) {
	_, _ = fmt.Fprintln(w, "usage:")
	_, _ = fmt.Fprintln(w, "  airway cli generate action [api] [action]")
}

func printCLIGenerateAPIUsage(w io.Writer) {
	_, _ = fmt.Fprintln(w, "usage:")
	_, _ = fmt.Fprintln(w, "  airway cli generate api [name]")
}

func printCLIGenerateModelUsage(w io.Writer) {
	_, _ = fmt.Fprintln(w, "usage:")
	_, _ = fmt.Fprintln(w, "  airway cli generate model [name] [field:type]...")
}

func printCLIGenerateServiceUsage(w io.Writer) {
	_, _ = fmt.Fprintln(w, "usage:")
	_, _ = fmt.Fprintln(w, "  airway cli generate service <name> <field:type> <field:type>...")
}

func printCLIGenerateCmdUsage(w io.Writer) {
	_, _ = fmt.Fprintln(w, "usage:")
	_, _ = fmt.Fprintln(w, "  airway cli generate cmd <name> <field> <field>...")
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

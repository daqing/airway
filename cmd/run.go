package cmd

import (
	"log"
	"os"
	"strconv"
)

func Run(args []string) {
	if err := run(args); err != nil {
		log.Fatal(err)
	}
}

func run(args []string) error {
	if len(args) == 0 {
		printUsage(os.Stdout)
		return nil
	}

	command := args[0]

	switch command {
	case "repl":
		runRepoREPL(args[1:])
		return nil
	case "version":
		showVersion(args[1:])
		return nil
	case "cli":
		// Backward-compatible alias for the pre-0.5 `airway cli ...` form.
		return runCLI(args[1:])
	case "help", "-h", "--help":
		printUsage(os.Stdout)
		return nil
	default:
		// Direct subcommands: `airway generate ...`, `airway db:migrate`, ...
		return runCLI(args)
	}
}

func parseInt(s string) int {
	i, err := strconv.Atoi(s)
	if err != nil {
		log.Fatalf("Invalid int: %s", s)
	}
	return i
}

func parseBool(s string) bool {
	if s == "true" {
		return true
	}

	if s == "false" {
		return false
	}

	i, err := strconv.Atoi(s)
	if err != nil {
		log.Fatalf("Invalid bool: %s", s)
	}

	return i != 0
}

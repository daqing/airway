package cmd

import (
	"fmt"
	"log"
)

// len(args) >= 1
func Run(args []string) {
	if err := run(args); err != nil {
		log.Fatal(err)
	}
}

func run(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("no args")
	}

	command := args[0]

	switch command {
	case "repl":
		runRepoREPL(args[1:])
	case "version":
		showVersion(args[1:])
	case "cli":
		return runCLI(args[1:])
	default:
		return fmt.Errorf("unknown command: %s", command)
	}

	return nil
}

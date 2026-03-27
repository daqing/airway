package cmd

import (
	"log"
)

// len(args) >= 1
func Run(args []string) {
	if len(args) == 0 {
		log.Fatal("No args")
	}

	command := args[0]

	switch command {
	case "repl":
		runRepoREPL(args[1:])
	case "version":
		showVersion(args[1:])
	default:
		log.Fatalf("Unknown command: %s", command)
	}
}

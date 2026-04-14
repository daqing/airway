package cmd

import (
	"fmt"
	"log"
	"strconv"
)

// args 最少会有一个参数
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

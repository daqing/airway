package main

import (
	"fmt"
	"os"
)

func main() {
	args := os.Args[1:]

	if len(args) == 0 {
		showHelp()
		return
	}

	switch args[0] {
	case "g":
		if len(args) == 1 {
			fmt.Println("cli g [action]")
			return
		}

		thing := args[1]
		switch thing {
		case "action":
			actionArgs := args[2:]
			if len(actionArgs) != 2 {
				fmt.Println("cli g action [api] [action]")
				return
			}

			GenerateAPIAction(actionArgs[0], actionArgs[1])
		case "migration":
			migrationArgs := args[2:]
			if len(migrationArgs) == 0 {
				fmt.Println("cli g migration [name]")
				return
			}

			GenerateMigration(migrationArgs[0])
		case "api":
			apiArgs := args[2:]
			if len(apiArgs) == 0 {
				fmt.Println("cli g api [name]")
				return
			}

			GenerateAPI(apiArgs[0])
		case "page":
			pageArgs := args[2:]
			if len(pageArgs) == 0 {
				fmt.Println("cli g page [name] [action]")
				return
			}

			var action = "index"
			if len(pageArgs) == 2 {
				action = pageArgs[1]
			}

			GeneratePage(pageArgs[0], action)
		}
	}
}

func showHelp() {
	fmt.Println("cli g [what] [params]")
}

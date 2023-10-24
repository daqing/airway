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
				fmt.Println("cli g action [mod] [action]")
				return
			}

			GenerateAction(actionArgs[0], actionArgs[1])
		case "migration":
			migrationArgs := args[2:]
			if len(migrationArgs) == 0 {
				fmt.Println("cli g migration [name]")
				return
			}

			GenerateMigration(migrationArgs[0])
		case "plugin":
			pluginArgs := args[2:]
			if len(pluginArgs) == 0 {
				fmt.Println("cli g plugin [name]")
				return
			}

			GeneratePlugin(pluginArgs[0])
		}
	}
}

func showHelp() {
	fmt.Println("cli g [what] [params]")
}

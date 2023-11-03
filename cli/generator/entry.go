package generator

import "github.com/daqing/airway/cli/helper"

func Generate(args []string) {
	if len(args) == 0 {
		helper.Help("cli g [what] [params]")
	}

	thing := args[1]
	xargs := args[2:]

	switch thing {
	case "action":
		GenAction(xargs)
	case "migration":
		GenMigration(xargs)
	case "api":
		GenAPI(xargs)
	case "page":
		GenPage(xargs)
	case "js":
		GenJS(xargs)
	}
}

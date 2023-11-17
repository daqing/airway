package generator

import (
	"fmt"

	"github.com/daqing/airway/cli/helper"
)

func Generate(args []string) {
	if len(args) == 0 {
		helper.Help("cli g [what] [params]")
	}

	thing := args[0]
	xargs := args[1:]

	switch thing {
	case "action":
		GenAction(xargs)
	case "page_action":
		GenPageAction(xargs)
	case "migration":
		GenMigration(xargs)
	case "api":
		GenAPI(xargs)
	case "page":
		GenPage(xargs)
	case "js":
		GenJS(xargs)
	default:
		panic("unknown generator")
	}

	fmt.Println("done.")
}

package generator

import (
	"fmt"
)

func Generate(args []string) {
	if len(args) == 0 {
		fmt.Println("cli g [what] [params]")
		return
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

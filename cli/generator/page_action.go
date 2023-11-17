package generator

import (
	"github.com/daqing/airway/cli/helper"
)

type PageActionGenerator struct {
	Mod     string
	Name    string
	APIName string
}

func GenPageAction(xargs []string) {
	if len(xargs) != 3 {
		helper.Help("cli g page_action [top-dir] [page] [action]")
	}

	GeneratePageAction(xargs[0], xargs[1], xargs[2])
	GeneratePageActionTemplate(xargs[0], xargs[1], xargs[2])
}

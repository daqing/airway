package generator

import (
	"strings"

	"github.com/daqing/airway/cli/helper"
	"github.com/daqing/airway/lib/utils"
)

type ActionGenerator struct {
	Mod  string
	Name string
}

func GenAction(xargs []string) {
	if len(xargs) != 2 {
		helper.Help("cli g action [api] [action]")
	}

	GenerateAPIAction(xargs[0], xargs[1])
}

func GenerateAPIAction(mod string, name string) {
	targetFileName := strings.Join(
		[]string{
			"./api",
			mod + "_api",
			name + "_action.go",
		},
		"/",
	)

	err := helper.ExecTemplate(
		"./cli/template/api/action.txt",
		targetFileName,
		ActionGenerator{Mod: mod, Name: utils.ToCamel(name)},
	)

	if err != nil {
		panic(err)
	}

}

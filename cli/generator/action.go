package generator

import (
	"strings"

	"github.com/daqing/airway/cli/helper"
	"github.com/daqing/airway/lib/repo"
)

type ActionGenerator struct {
	Mod  string
	Name string
}

func GenAction(xargs []string) {
	if len(xargs) != 3 {
		helper.Help("cli g action [top-dir] [api] [action]")
	}

	GenerateAPIAction(xargs[0], xargs[1], xargs[2])
}

func GenerateAPIAction(topDir, mod string, name string) {
	targetFileName := strings.Join(
		[]string{
			".",
			topDir,
			"api",
			mod + "_api",
			name + "_action.go",
		},
		"/",
	)

	err := helper.ExecTemplate(
		"./cli/template/api/action.txt",
		targetFileName,
		ActionGenerator{Mod: mod, Name: repo.ToCamel(name)},
	)

	if err != nil {
		panic(err)
	}

}

package generator

import (
	"fmt"
	"strings"

	"github.com/daqing/airway/cli/helper"
	"github.com/daqing/airway/lib/pg_repo"
)

type ActionGenerator struct {
	Mod     string
	Name    string
	APIName string
}

func GenAction(xargs []string) {
	if len(xargs) != 3 {
		helper.Help("cli g action [top-dir] [api] [action]")
	}

	GenerateAPIAction(xargs[0], xargs[1], xargs[2])
}

func GenerateAPIAction(topDir, mod string, name string) {
	apiName := apiDirName(topDir, mod)

	targetFileName := strings.Join(
		[]string{
			".",
			topDir,
			"api",
			apiName,
			name + "_action.go",
		},
		"/",
	)

	err := helper.ExecTemplate(
		"./cli/template/api/action.txt",
		targetFileName,
		ActionGenerator{Mod: mod, Name: pg_repo.ToCamel(name), APIName: apiName},
	)

	if err != nil {
		fmt.Println(err)
	}

}

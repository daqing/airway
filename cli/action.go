package main

import (
	"strings"

	"github.com/daqing/airway/lib/utils"
)

type ActionGenerator struct {
	Mod  string
	Name string
}

func GenerateAction(mod string, name string) {
	targetFileName := strings.Join(
		[]string{
			"./api",
			mod + "_api",
			name + "_action.go",
		},
		"/",
	)

	err := ExecTemplate(
		"./cli/template/action.txt",
		targetFileName,
		ActionGenerator{Mod: mod, Name: utils.ToCamel(name)},
	)

	if err != nil {
		panic(err)
	}

}

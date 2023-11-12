package generator

import (
	"strings"

	"github.com/daqing/airway/cli/helper"
	"github.com/daqing/airway/lib/repo"
)

func GenJS(args []string) {
	if len(args) < 2 {
		helper.Help("cli g js [prefix] [api] [action]")
	}

	if len(args) == 2 {
		GeneratePageReactJS(DEFAULT_PREFIX_FOLDER, args[0], args[1])
	} else {
		GeneratePageReactJS(args[0], args[1], args[2])
	}

}

func GeneratePageReactJS(prefixFolder string, page string, action string) {
	filename := page + "_" + action + ".jsx"

	// TODO: add const definition for default value "."
	if prefixFolder != DEFAULT_PREFIX_FOLDER {
		filename = prefixFolder + "_" + filename
	}

	targetFileName := strings.Join(
		[]string{
			"./frontend/javascripts/src",
			filename,
		},
		"/",
	)

	err := helper.ExecTemplate(
		"./cli/template/js/react.txt",
		targetFileName,
		PageGenerator{Page: page, Name: action, Action: repo.ToCamel(action)},
	)

	if err != nil {
		panic(err)
	}
}

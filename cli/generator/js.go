package generator

import (
	"strings"

	"github.com/daqing/airway/cli/helper"
	"github.com/daqing/airway/lib/utils"
)

func GenJS(args []string) {
	if len(args) < 2 {
		helper.Help("cli g js [api] [action]")
	}

	GeneratePageReactJS(args[0], args[1])
}

func GeneratePageReactJS(page string, action string) {
	targetFileName := strings.Join(
		[]string{
			"./app/javascripts/src",
			page + "_" + action + ".jsx",
		},
		"/",
	)

	err := helper.ExecTemplate(
		"./cli/template/js/react.txt",
		targetFileName,
		PageGenerator{Page: page, Name: action, Action: utils.ToCamel(action)},
	)

	if err != nil {
		panic(err)
	}
}

package generator

import (
	"fmt"
	"strings"

	"github.com/daqing/airway/cli/helper"
	"github.com/daqing/airway/lib/repo"
	"github.com/daqing/airway/lib/utils"
)

func GenJS(args []string) {
	if len(args) < 3 {
		helper.Help("cli g js [top-dir] [page] [action]")
	}

	GeneratePageReactJS(args[0], args[1], args[2])
}

func GeneratePageReactJS(topDir, page, action string) {
	filename := topDir + "_" + utils.NormalizePage(page) + "_" + action + ".jsx"

	targetFileName := strings.Join(
		[]string{
			"./app/frontend/javascripts/src",
			filename,
		},
		"/",
	)

	err := helper.ExecTemplate(
		"./cli/template/js/react.txt",
		targetFileName,
		PageGenerator{Page: utils.NormalizePage(page), Name: action, Action: repo.ToCamel(action)},
	)

	if err != nil {
		fmt.Println(err)
	}
}

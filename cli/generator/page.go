package generator

import (
	"fmt"
	"os"
	"strings"

	"github.com/daqing/airway/cli/helper"
	"github.com/daqing/airway/lib/utils"
)

func GenPage(args []string) {
	if len(args) < 2 {
		helper.Help("cli g page [api] [action]")
	}

	GeneratePage(args[0], args[1])
}

func GeneratePage(name string, action string) {
	dir := fmt.Sprintf("%s_page", name)

	if _, err := os.Stat(dir); err == nil {
		panic("Page already exists")
	}

	dirPath := fmt.Sprintf("./pages/%s", dir)

	if err := os.Mkdir(dirPath, 0755); err != nil {
		panic(err)
	}

	GeneratePageAction(name, action)

	GeneratePageActionTemplate(name, action)
	GeneratePageLayout(name)
	GeneratePageRoutes(name, action)

	GeneratePageReactJS(name, action)
}

func GeneratePageAction(page string, action string) {
	targetFileName := strings.Join(
		[]string{
			"./pages",
			page + "_page",
			action + "_action.go",
		},
		"/",
	)

	err := helper.ExecTemplate(
		"./cli/template/page/action.txt",
		targetFileName,
		PageGenerator{Page: page, Name: action, Action: utils.ToCamel(action)},
	)

	if err != nil {
		panic(err)
	}

}

type PageGenerator struct {
	Page   string
	Name   string
	Action string
}

func GeneratePageActionTemplate(page string, action string) {
	targetFileName := strings.Join(
		[]string{
			"./pages",
			page + "_page",
			action + ".amber",
		},
		"/",
	)

	err := helper.ExecTemplate(
		"./cli/template/page/action.amber",
		targetFileName,
		PageGenerator{Page: page, Action: utils.ToCamel(action)},
	)

	if err != nil {
		panic(err)
	}
}

func GeneratePageLayout(page string) {
	targetFileName := strings.Join(
		[]string{
			"./pages",
			page + "_page",
			"layout.amber",
		},
		"/",
	)

	err := helper.ExecTemplate(
		"./cli/template/page/layout.amber",
		targetFileName,
		PageGenerator{Page: page},
	)

	if err != nil {
		panic(err)
	}
}

func GeneratePageRoutes(page string, action string) {
	targetFileName := strings.Join(
		[]string{
			"./pages",
			page + "_page",
			"routes.go",
		},
		"/",
	)

	err := helper.ExecTemplate(
		"./cli/template/page/routes.txt",
		targetFileName,
		PageGenerator{Page: page, Name: action, Action: utils.ToCamel(action)},
	)

	if err != nil {
		panic(err)
	}

}

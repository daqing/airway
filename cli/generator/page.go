package generator

import (
	"fmt"
	"os"
	"strings"

	"github.com/daqing/airway/cli/helper"
	"github.com/daqing/airway/lib/repo"
)

const DEFAULT_PREFIX_FOLDER = "."

func GenPage(args []string) {
	if len(args) < 2 {
		helper.Help("cli g page [api] [action]")
	}

	fmt.Printf("gen page for %s, %s\n", args[0], args[1])

	GeneratePage(args[0], args[1])
}

func GeneratePage(name string, action string) {
	var prefixFolder = DEFAULT_PREFIX_FOLDER

	if strings.Contains(name, ".") {
		parts := strings.Split(name, ".")

		prefixFolder = parts[0]
		name = parts[1]
	}

	dir := fmt.Sprintf("%s/%s_page", prefixFolder, name)

	dirPath := fmt.Sprintf("./pages/%s", dir)
	if err := os.Mkdir(dirPath, 0755); err != nil {
		// page directory exists, generate new action
		GeneratePageAction(prefixFolder, name, action)
		GeneratePageActionTemplate(prefixFolder, name, action)
		GeneratePageReactJS(prefixFolder, name, action)

		os.Exit(0)
	}

	GeneratePageAction(prefixFolder, name, action)

	GeneratePageActionTemplate(prefixFolder, name, action)

	GeneratePageLayout(prefixFolder, name)
	GeneratePageRoutes(prefixFolder, name, action)

	GeneratePageReactJS(prefixFolder, name, action)
}

func GeneratePageAction(prefixFolder string, page string, action string) {
	targetFileName := strings.Join(
		[]string{
			"./pages",
			prefixFolder,
			page + "_page",
			action + "_action.go",
		},
		"/",
	)

	err := helper.ExecTemplate(
		"./cli/template/page/action.txt",
		targetFileName,
		PageGenerator{Page: page, Name: action, Action: repo.ToCamel(action)},
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

func GeneratePageActionTemplate(prefixFolder string, page string, action string) {
	targetFileName := strings.Join(
		[]string{
			"./pages",
			prefixFolder,
			page + "_page",
			action + ".amber",
		},
		"/",
	)

	err := helper.ExecTemplate(
		"./cli/template/page/action.amber",
		targetFileName,
		PageGenerator{Page: page, Action: repo.ToCamel(action)},
	)

	if err != nil {
		panic(err)
	}
}

func GeneratePageLayout(prefixFolder, page string) {
	targetFileName := strings.Join(
		[]string{
			"./pages",
			prefixFolder,
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

func GeneratePageRoutes(prefixFolder, page string, action string) {
	targetFileName := strings.Join(
		[]string{
			"./pages",
			prefixFolder,
			page + "_page",
			"routes.go",
		},
		"/",
	)

	err := helper.ExecTemplate(
		"./cli/template/page/routes.txt",
		targetFileName,
		PageGenerator{Page: page, Name: action, Action: repo.ToCamel(action)},
	)

	if err != nil {
		panic(err)
	}

}

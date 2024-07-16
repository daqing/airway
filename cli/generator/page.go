package generator

import (
	"fmt"
	"os"
	"strings"

	"github.com/daqing/airway/cli/helper"
	"github.com/daqing/airway/lib/sql_orm"
	"github.com/daqing/airway/lib/utils"
)

func GenPage(args []string) {
	if len(args) < 3 {
		helper.Help("cli g page [top-dir] [page] [action]")
	}

	fmt.Printf("gen page for %s, %s, %s\n", args[0], args[1], args[2])

	GeneratePage(args[0], args[1], args[2])
}

func GeneratePage(topDir, page, action string) {
	dirName := utils.PageDirPath(topDir, page)

	if err := os.MkdirAll(dirName, 0755); err != nil {
		// page directory exists, generate new action
		GeneratePageAction(topDir, page, action)
		GeneratePageActionTemplate(topDir, page, action)
		GeneratePageReactJS(topDir, page, action)

		os.Exit(0)
	}

	GeneratePageAction(topDir, page, action)
	GeneratePageActionTemplate(topDir, page, action)

	GeneratePageRoutes(topDir, page, action)

	GeneratePageReactJS(topDir, page, action)
}

func GeneratePageAction(topDir, page, action string) {
	dirName := utils.PageDirPath(topDir, page)

	if err := os.MkdirAll(dirName, 0755); err != nil {
		fmt.Println(err)
	}

	targetFileName := strings.Join(
		[]string{
			dirName,
			action + "_action.go",
		},
		"/",
	)

	err := helper.ExecTemplate(
		"./cli/template/page/action.txt",
		targetFileName,
		PageGenerator{
			Page:    page,
			Name:    action,
			Action:  sql_orm.ToCamel(action),
			TopDir:  topDir,
			PkgName: utils.PagePkgName(topDir, page),
		},
	)

	if err != nil {
		fmt.Println(err)
	}

}

type PageGenerator struct {
	Page   string
	Name   string
	Action string

	TopDir  string
	PkgName string
}

func GeneratePageActionTemplate(topDir, page, action string) {
	dirName := utils.PageDirPath(topDir, page)

	targetFileName := strings.Join(
		[]string{
			dirName,
			action + ".amber",
		},
		"/",
	)

	err := helper.ExecTemplate(
		"./cli/template/page/action.amber",
		targetFileName,
		PageGenerator{Page: utils.NormalizePage(page), Action: sql_orm.ToCamel(action)},
	)

	if err != nil {
		fmt.Println(err)
	}
}

func GeneratePageRoutes(topDir, page, action string) {
	dirName := utils.PageDirPath(topDir, page)

	targetFileName := strings.Join(
		[]string{
			dirName,
			"routes.go",
		},
		"/",
	)

	err := helper.ExecTemplate(
		"./cli/template/page/routes.txt",
		targetFileName,
		PageGenerator{
			Page:    utils.NormalizePage(page),
			Name:    action,
			Action:  sql_orm.ToCamel(action),
			PkgName: utils.PagePkgName(topDir, page),
		},
	)

	if err != nil {
		fmt.Println(err)
	}

}

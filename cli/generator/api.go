package generator

import (
	"fmt"
	"os"
	"strings"

	"github.com/daqing/airway/cli/helper"
	"github.com/daqing/airway/lib/repo"
)

func GenAPI(xargs []string) {
	if len(xargs) != 2 {
		helper.Help("cli g api [top-dir] [name]")
	}

	GenerateAPI(xargs[0], xargs[1])
}

type APIGenerator struct {
	Mod       string
	ModelName string
}

func GenerateAPI(topDir, name string) {
	dir := fmt.Sprintf("%s_api", name)
	dirPath := fmt.Sprintf("./%s/api/%s", topDir, dir)

	if err := os.Mkdir(dirPath, 0755); err != nil {
		panic(err)
	}

	GenerateAPIAction(topDir, name, "index")
	GenerateAPIAction(topDir, name, "show")
	GenerateAPIAction(topDir, name, "create")

	generateAPIRoutes(name)
	generateAPIModels(name)
	generateAPIServices(name)
	generateResp(name)

}

func generateAPIRoutes(name string) {
	targetPath := strings.Join([]string{
		"./api",
		fmt.Sprintf("%s_api", name),
		"routes.go",
	}, "/")

	helper.ExecTemplate(
		"./cli/template/api/routes.txt",
		targetPath,
		APIGenerator{name, repo.ToCamel(name)},
	)
}

func generateAPIModels(name string) {
	targetPath := strings.Join([]string{
		"./api",
		fmt.Sprintf("%s_api", name),
		"models.go",
	}, "/")

	helper.ExecTemplate(
		"./cli/template/api/models.txt",
		targetPath,
		APIGenerator{name, repo.ToCamel(name)},
	)
}

func generateAPIServices(name string) {
	targetPath := strings.Join([]string{
		"./api",
		fmt.Sprintf("%s_api", name),
		"services.go",
	}, "/")

	helper.ExecTemplate(
		"./cli/template/api/services.txt",
		targetPath,
		APIGenerator{name, repo.ToCamel(name)},
	)
}

func generateResp(name string) {
	targetPath := strings.Join([]string{
		"./api",
		fmt.Sprintf("%s_api", name),
		"resp.go",
	}, "/")

	helper.ExecTemplate(
		"./cli/template/api/resp.txt",
		targetPath,
		APIGenerator{name, repo.ToCamel(name)},
	)
}

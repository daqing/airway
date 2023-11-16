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
	APIName   string
}

func GenerateAPI(topDir, name string) {
	dir := apiDirName(topDir, name)
	dirPath := fmt.Sprintf("./%s/api/%s", topDir, dir)

	if err := os.Mkdir(dirPath, 0755); err != nil {
		panic(err)
	}

	GenerateAPIAction(topDir, name, "index")
	GenerateAPIAction(topDir, name, "show")
	GenerateAPIAction(topDir, name, "create")

	generateAPIRoutes(topDir, name)
	generateAPIModels(topDir, name)
	generateAPIServices(topDir, name)
	generateResp(topDir, name)

}

func generateAPIRoutes(topDir, name string) {
	apiName := apiDirName(topDir, name)
	targetPath := strings.Join([]string{
		".",
		topDir,
		"api",
		apiName,
		"routes.go",
	}, "/")

	helper.ExecTemplate(
		"./cli/template/api/routes.txt",
		targetPath,
		APIGenerator{name, repo.ToCamel(name), apiName},
	)
}

func generateAPIModels(topDir, name string) {
	apiName := apiDirName(topDir, name)

	targetPath := strings.Join([]string{
		".",
		topDir,
		"api",
		apiName,
		"models.go",
	}, "/")

	helper.ExecTemplate(
		"./cli/template/api/models.txt",
		targetPath,
		APIGenerator{name, repo.ToCamel(name), apiName},
	)
}

func generateAPIServices(topDir, name string) {
	apiName := apiDirName(topDir, name)

	targetPath := strings.Join([]string{
		".",
		topDir,
		"api",
		apiName,
		"services.go",
	}, "/")

	helper.ExecTemplate(
		"./cli/template/api/services.txt",
		targetPath,
		APIGenerator{name, repo.ToCamel(name), apiName},
	)
}

func generateResp(topDir, name string) {
	apiName := apiDirName(topDir, name)

	targetPath := strings.Join([]string{
		".",
		topDir,
		"api",
		apiName,
		"resp.go",
	}, "/")

	helper.ExecTemplate(
		"./cli/template/api/resp.txt",
		targetPath,
		APIGenerator{name, repo.ToCamel(name), apiName},
	)
}

func apiDirName(topDir, name string) string {
	if topDir == "core" {
		return fmt.Sprintf("%s_api", name)
	}

	return fmt.Sprintf("%s_%s_api", topDir, name)
}

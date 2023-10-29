package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/daqing/airway/lib/utils"
)

type APIGenerator struct {
	Mod       string
	ModelName string
}

func GenerateAPI(name string) {
	dir := fmt.Sprintf("%s_api", name)

	if _, err := os.Stat(dir); err == nil {
		panic("API already exists")
	}

	dirPath := fmt.Sprintf("./api/%s", dir)

	if err := os.Mkdir(dirPath, 0755); err != nil {
		panic(err)
	}

	GenerateAPIAction(name, "index")
	GenerateAPIAction(name, "show")
	GenerateAPIAction(name, "create")

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

	ExecTemplate(
		"./cli/template/api/routes.txt",
		targetPath,
		APIGenerator{name, utils.ToCamel(name)},
	)
}

func generateAPIModels(name string) {
	targetPath := strings.Join([]string{
		"./api",
		fmt.Sprintf("%s_api", name),
		"models.go",
	}, "/")

	ExecTemplate(
		"./cli/template/api/models.txt",
		targetPath,
		APIGenerator{name, utils.ToCamel(name)},
	)
}

func generateAPIServices(name string) {
	targetPath := strings.Join([]string{
		"./api",
		fmt.Sprintf("%s_api", name),
		"services.go",
	}, "/")

	ExecTemplate(
		"./cli/template/api/services.txt",
		targetPath,
		APIGenerator{name, utils.ToCamel(name)},
	)
}

func generateResp(name string) {
	targetPath := strings.Join([]string{
		"./api",
		fmt.Sprintf("%s_api", name),
		"resp.go",
	}, "/")

	ExecTemplate(
		"./cli/template/api/resp.txt",
		targetPath,
		APIGenerator{name, utils.ToCamel(name)},
	)
}

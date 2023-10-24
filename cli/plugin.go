package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/daqing/airway/lib/utils"
)

type PluginGenerator struct {
	Mod       string
	ModelName string
}

func GeneratePlugin(name string) {
	dir := fmt.Sprintf("%s_plugin", name)

	if _, err := os.Stat(dir); err == nil {
		panic("Plugin already exists")
	}

	dirPath := fmt.Sprintf("./plugins/%s", dir)

	if err := os.Mkdir(dirPath, 0755); err != nil {
		panic(err)
	}

	GenerateAction(name, "index")
	GenerateAction(name, "show")
	GenerateAction(name, "create")

	generateRoutes(name)
	generateModels(name)
	generateServices(name)
	generateResp(name)

}

func generateRoutes(name string) {
	targetPath := strings.Join([]string{
		"./plugins",
		fmt.Sprintf("%s_plugin", name),
		"routes.go",
	}, "/")

	ExecTemplate(
		"./cli/template/plugin/routes.txt",
		targetPath,
		PluginGenerator{name, utils.ToCamel(name)},
	)
}

func generateModels(name string) {
	targetPath := strings.Join([]string{
		"./plugins",
		fmt.Sprintf("%s_plugin", name),
		"models.go",
	}, "/")

	ExecTemplate(
		"./cli/template/plugin/models.txt",
		targetPath,
		PluginGenerator{name, utils.ToCamel(name)},
	)
}

func generateServices(name string) {
	targetPath := strings.Join([]string{
		"./plugins",
		fmt.Sprintf("%s_plugin", name),
		"services.go",
	}, "/")

	ExecTemplate(
		"./cli/template/plugin/services.txt",
		targetPath,
		PluginGenerator{name, utils.ToCamel(name)},
	)
}

func generateResp(name string) {
	targetPath := strings.Join([]string{
		"./plugins",
		fmt.Sprintf("%s_plugin", name),
		"resp.go",
	}, "/")

	ExecTemplate(
		"./cli/template/plugin/resp.txt",
		targetPath,
		PluginGenerator{name, utils.ToCamel(name)},
	)
}

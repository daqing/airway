package scaffold

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/daqing/airway/cli/generator"
	"github.com/daqing/airway/cli/helper"
	"github.com/daqing/airway/lib/repo"
	"github.com/daqing/airway/lib/utils"
)

type Scaffold struct {
	TopDir     string
	Page       string
	IsAdmin    bool
	PkgName    string
	Model      string
	Lower      string
	FieldPairs []FieldType
}

type FieldType struct {
	Name string
	Type string
}

func (ft FieldType) SQLType() string {
	switch ft.Type {
	case "string":
		return "VARCHAR(255) NOT NULL"
	case "int":
		return "INT NOT NULL"
	case "int64":
		return "BIGINT NOT NULL"
	default:
		return "<unknown>"
	}
}

func LogError(prefix string, err error) {
	fmt.Println(prefix, ":", err)
	panic(err)
}

func (ft FieldType) SkipTrim() bool {
	return ft.Type == "int64"
}

func (sf *Scaffold) FieldTypes() []FieldType {
	return sf.FieldPairs
}

func (sf *Scaffold) RedirectURL() string {
	if sf.IsAdmin {
		return "/admin/" + sf.Lower
	}

	return "/" + sf.Lower
}

func (sf *Scaffold) LayoutName() string {
	if sf.IsAdmin {
		return "/views/admin/layout"
	}

	return "/views/layout"
}

func (f FieldType) NameCamel() string {
	return repo.ToCamel(f.Name)
}

func Generate(xargs []string) {
	if len(xargs) < 3 {
		fmt.Println("Usage: cli sf [top-dir] [page] [attr:type] [attr:value]")
		os.Exit(1)
	}

	var sf Scaffold

	sf.TopDir = xargs[0]
	sf.Page = xargs[1]

	if strings.Contains(sf.Page, ".") {
		parts := strings.Split(sf.Page, ".")
		sf.IsAdmin = parts[0] == "admin"
		sf.Model = repo.ToCamel(parts[1])
		sf.Lower = parts[1]
	} else {
		sf.IsAdmin = false
		sf.Model = repo.ToCamel(sf.Page)
		sf.Lower = sf.Page
	}

	sf.PkgName = utils.PagePkgName(sf.TopDir, sf.Page)

	for _, pair := range xargs[2:] {
		var name string
		var typ string

		if strings.Contains(pair, ":") {
			parts := strings.Split(pair, ":")
			name = parts[0]
			typ = parts[1]

			if typ == "bigint" {
				typ = "int64"
			}

		} else {
			name = pair
			typ = "string"
		}

		sf.FieldPairs = append(sf.FieldPairs, FieldType{name, typ})
	}

	sf.generate()

	path := generator.GenerateMigration("create_" + sf.Lower + "_table")

	sf.genTableSQL(path)
}

func (sf *Scaffold) genTableSQL(path string) {
	err := helper.ExecTemplateForce(
		"./cli/template/scaffold/sql.txt",
		path,
		sf,
	)

	if err != nil && !errors.Is(err, os.ErrExist) {
		LogError("genTableSQL", err)
	}
}

func (sf *Scaffold) generate() {
	sf.genAction("index", true)
	sf.genAction("new", true)
	sf.genAction("edit", true)

	sf.genAction("create", false)
	sf.genAction("update", false)
	sf.genAction("delete", false)

	sf.genRoutes()
	sf.genModel()
}

func (sf *Scaffold) genAction(action string, hasView bool) {
	dirName := utils.PageDirPath(sf.TopDir, sf.Page)

	if err := os.MkdirAll(dirName, 0755); err != nil {
		fmt.Println(err)
	}

	targetActionFile := strings.Join(
		[]string{
			dirName,
			action + "_action.go",
		},
		"/",
	)

	err := helper.ExecTemplate(
		"./cli/template/scaffold/"+action+"_action.txt",
		targetActionFile,
		sf,
	)

	if err != nil && !errors.Is(err, os.ErrExist) {
		LogError(fmt.Sprintf("generate action file: %s", action), err)
	}

	if !hasView {
		return
	}

	targetViewFile := strings.Join(
		[]string{
			dirName,
			action + ".amber",
		},
		"/",
	)

	err = helper.ExecTemplate(
		"./cli/template/scaffold/"+action+"_view.txt",
		targetViewFile,
		sf,
	)

	if err != nil && !errors.Is(err, os.ErrExist) {
		LogError(fmt.Sprintf("generate view file: %s", action), err)
	}

}

func (sf *Scaffold) genRoutes() {
	dirName := utils.PageDirPath(sf.TopDir, sf.Page)

	if err := os.MkdirAll(dirName, 0755); err != nil {
		fmt.Println(err)
	}

	targetRoutesFile := strings.Join(
		[]string{
			dirName,
			"routes.go",
		},
		"/",
	)

	err := helper.ExecTemplate(
		"./cli/template/scaffold/routes.txt",
		targetRoutesFile,
		sf,
	)

	if err != nil && !errors.Is(err, os.ErrExist) {
		LogError("generate routes", err)
	}
}

func (sf *Scaffold) genModel() {
	dirName := utils.PageDirPath(sf.TopDir, sf.Page)

	if err := os.MkdirAll(dirName, 0755); err != nil {
		fmt.Println(err)
	}

	targetFile := strings.Join(
		[]string{
			dirName,
			"models.go",
		},
		"/",
	)

	err := helper.ExecTemplate(
		"./cli/template/scaffold/models.txt",
		targetFile,
		sf,
	)

	if err != nil && !errors.Is(err, os.ErrExist) {
		LogError("generate models", err)
	}
}

func (sf *Scaffold) Fields() string {
	var result []string

	for _, f := range sf.FieldPairs {
		result = append(result, fmt.Sprintf(`"%s"`, f.Name))
	}

	return fmt.Sprintf("[]string{%s}", strings.Join(result, ", "))
}

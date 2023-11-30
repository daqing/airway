package scaffold

import (
	"fmt"
	"os"
	"strings"

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
	FieldTypes []FieldType
}

type FieldType struct {
	Name string
	Type string
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
	} else {
		sf.IsAdmin = false
		sf.Model = repo.ToCamel(sf.Page)
	}

	sf.PkgName = utils.PagePkgName(sf.TopDir, sf.Page)

	sf.FieldTypes = append(sf.FieldTypes, FieldType{"id", "int64"})

	for _, pair := range xargs[2:] {
		var name string
		var typ string

		if strings.Contains(pair, ":") {
			parts := strings.Split(pair, ":")
			name = parts[0]
			typ = parts[1]
		} else {
			name = pair
			typ = "string"
		}

		sf.FieldTypes = append(sf.FieldTypes, FieldType{name, typ})
	}

	fmt.Println("top dir:", sf.TopDir)
	fmt.Println("page:", sf.Page)
	fmt.Println("is admin?:", sf.IsAdmin)
	fmt.Println("modal name:", sf.Model)
	fmt.Println("fields:", sf.Fields())

	sf.generate()
}

func (sf *Scaffold) generate() {
	fmt.Println("generate scaffold...")
	sf.genAction("index")
}

func (sf *Scaffold) genAction(action string) {
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
		"./cli/template/scaffold/"+action+".txt",
		targetActionFile,
		sf,
	)

	if err != nil {
		fmt.Println(err)
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

	if err != nil {
		fmt.Println(err)
	}

}

func (sf *Scaffold) Fields() string {
	var result []string

	for _, f := range sf.FieldTypes {
		result = append(result, fmt.Sprintf(`"%s"`, f.Name))
	}

	return fmt.Sprintf("[]string{%s}", strings.Join(result, ", "))
}

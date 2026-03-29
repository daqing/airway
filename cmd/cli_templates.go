package cmd

const actionTemplate = `package {{.APIName}}

import (
	"github.com/gin-gonic/gin"

	"github.com/daqing/airway/lib/render"
)

type {{.Name}}Params struct {
}

func {{.Name}}Action(c *gin.Context) {
	var p {{.Name}}Params

	if err := c.BindJSON(&p); err != nil {
		render.Error(c, err)
		return
	}

	render.Empty(c)
}
`

type actionTemplateData struct {
	Name    string
	APIName string
}

const routesTemplate = `package {{.APIName}}

import "github.com/gin-gonic/gin"

func Routes(r *gin.RouterGroup) {
	g := r.Group("/{{.Mod}}")
	{
		g.GET("/index", IndexAction)
	}
}
`

type routesTemplateData struct {
	Mod     string
	APIName string
}

const modelTemplate = `package models

import (
	"time"

	airwaysql "github.com/daqing/airway/lib/sql"
)

type {{.Name}} struct {
	ID        airwaysql.IdType ` + "`db:\"id\" json:\"id\"`" + `
	CreatedAt time.Time        ` + "`db:\"created_at\" json:\"created_at\"`" + `
	UpdatedAt time.Time        ` + "`db:\"updated_at\" json:\"updated_at\"`" + `
}

func ({{.Name}}) TableName() string {
	return "{{.TableName}}"
}

func init() {
	registerREPLModel("{{.Name}}", {{.Name}}{})
}
`

type modelTemplateData struct {
	Name      string
	TableName string
}

const serviceTemplate = `package services

import (
	"github.com/daqing/airway/app/models"
	"github.com/daqing/airway/lib/repo"
	sql "github.com/daqing/airway/lib/sql"
)

func Find{{.Name}}(id sql.IdType) (*models.{{.Name}}, error) {
	return repo.FindOne[models.{{.Name}}](repo.CurrentDB(), sql.Select("*").From((models.{{.Name}}{}).TableName()).Where(sql.Eq("id", id)))
}

func Create{{.Name}}({{.Fields}}) (*models.{{.Name}}, error) {
	return repo.Insert[models.{{.Name}}](repo.CurrentDB(), sql.InsertInto((models.{{.Name}}{}).TableName(), {{.SQLH}}))
}

func Update{{.Name}}(id sql.IdType, {{.Fields}}) error {
	return repo.Update(repo.CurrentDB(), sql.Update((models.{{.Name}}{}).TableName(), {{.SQLH}}).Where(sql.Eq("id", id)))
}

func Delete{{.Name}}(id sql.IdType) error {
	return repo.Delete(repo.CurrentDB(), sql.DeleteFrom((models.{{.Name}}{}).TableName()).Where(sql.Eq("id", id)))
}
`

type serviceTemplateData struct {
	Name   string
	Fields string
	SQLH   string
}

const cmdTemplate = `package cmd

import (
	"fmt"
	"log"
	"strconv"

	"github.com/daqing/airway/app/models"
	"github.com/daqing/airway/app/services"
	sql "github.com/daqing/airway/lib/sql"
)

func create{{.Name}}(args []string) {
	if len(args) != {{.CreateArgCount}} {
		fmt.Println("Usage: create_{{.LowerName}} {{.Fields}}")
		return
	}

	_, _ = services.Create{{.Name}}({{.Args}})
}

func delete{{.Name}}(args []string) {
	if len(args) != 1 {
		log.Fatal("Usage: delete_{{.LowerName}} <id>")
	}

	id, err := strconv.ParseInt(args[0], 10, 64)
	if err != nil {
		log.Fatalf("Invalid id: %s", args[0])
	}

	_ = services.Delete{{.Name}}(sql.IdType(id))
}

func update{{.Name}}(args []string) {
	if len(args) != {{.UpdateArgCount}} {
		log.Fatal("Usage: update_{{.LowerName}} <id> {{.Fields}}")
	}

	id, err := strconv.ParseInt(args[0], 10, 64)
	if err != nil {
		log.Fatalf("Invalid id: %s", args[0])
	}

	_ = services.Update{{.Name}}(sql.IdType(id), {{.Args1}})
}

func find{{.Name}}(args []string) (*models.{{.Name}}, error) {
	if len(args) != 1 {
		log.Fatal("Usage: find_{{.LowerName}} <id>")
	}

	id, err := strconv.ParseInt(args[0], 10, 64)
	if err != nil {
		log.Fatalf("Invalid id: %s", args[0])
	}

	return services.Find{{.Name}}(sql.IdType(id))
}
`

type cmdTemplateData struct {
	Name           string
	LowerName      string
	Fields         string
	Args           string
	Args1          string
	CreateArgCount int
	UpdateArgCount int
}

package cmd

const scaffoldModelTemplate = `package models

import (
	"time"

	airwaysql "github.com/daqing/airway/lib/sql"
)

type {{.Name}} struct {
	ID        airwaysql.IdType ` + "`db:\"id\" json:\"id\"`" + `
{{range .Fields}}	{{.Name}} {{.GoType}} ` + "`db:\"{{.JSON}}\" json:\"{{.JSON}}\"`" + `
{{end}}	CreatedAt time.Time ` + "`db:\"created_at\" json:\"created_at\"`" + `
	UpdatedAt time.Time ` + "`db:\"updated_at\" json:\"updated_at\"`" + `
}

func ({{.Name}}) TableName() string {
	return "{{.SlugPlural}}"
}

func init() {
	registerREPLModel("{{.Name}}", {{.Name}}{})
}
`

const scaffoldUpTemplate = `CREATE TABLE {{.SlugPlural}} (
	{{.IDColumn}},
{{range .Fields}}	{{.SQLCol}},
{{end}}	created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
	updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);
`

const scaffoldDownTemplate = `DROP TABLE IF EXISTS {{.SlugPlural}};
`

const scaffoldRoutesTemplate = `package {{.APIName}}

import "github.com/gin-gonic/gin"

func Routes(r *gin.Engine) {
	g := r.Group("/api/v1/{{.SlugPlural}}")
	{
		g.GET("", IndexAction)
		g.POST("", CreateAction)
		g.PUT("/:id", UpdateAction)
		g.DELETE("/:id", DestroyAction)
	}

	r.GET("/{{.SlugPlural}}", PageAction)
	r.GET("/{{.SlugPlural}}/:id", ShowAction)
}
`

const scaffoldOpenAPITemplate = `package {{.APIName}}

import (
	"github.com/daqing/airway/lib/openapi"

	"{{.Module}}/app/models"
)

func init() {
	openapi.Get("/api/v1/{{.SlugPlural}}", func(o *openapi.Operation) {
		o.Summary("List {{.NamePlural}}").Tag("{{.SlugPlural}}").
			OK(openapi.List[models.{{.Name}}]())
	})

	openapi.Post("/api/v1/{{.SlugPlural}}", func(o *openapi.Operation) {
		o.Summary("Create a {{.Slug}}").Tag("{{.SlugPlural}}").
			Body(openapi.Item[{{.Slug}}Params]()).
			OK(openapi.Item[models.{{.Name}}]())
	})

	openapi.Put("/api/v1/{{.SlugPlural}}/{id}", func(o *openapi.Operation) {
		o.Summary("Update a {{.Slug}}").Tag("{{.SlugPlural}}").
			Path("id", openapi.Int(), "{{.Name}} id").
			Body(openapi.Item[{{.Slug}}Params]()).
			OK()
	})

	openapi.Delete("/api/v1/{{.SlugPlural}}/{id}", func(o *openapi.Operation) {
		o.Summary("Delete a {{.Slug}}").Tag("{{.SlugPlural}}").
			Path("id", openapi.Int(), "{{.Name}} id").
			OK()
	})

	openapi.Get("/{{.SlugPlural}}", func(o *openapi.Operation) {
		o.Summary("{{.NamePlural}} page").Tag("{{.SlugPlural}}")
		o.Respond(200, "text/html", openapi.Str()).
			Description("Server-rendered page hosting the CRUD island")
	})

	openapi.Get("/{{.SlugPlural}}/{id}", func(o *openapi.Operation) {
		o.Summary("{{.Name}} page").Tag("{{.SlugPlural}}")
		o.Path("id", openapi.Int(), "{{.Name}} id")
		o.Respond(200, "text/html", openapi.Str()).
			Description("Server-rendered {{.Slug}} detail page")
	})
}
`

const scaffoldActionsTemplate = `package {{.APIName}}

import (
	"strconv"

	"github.com/gin-gonic/gin"

	"{{.Module}}/app/models"
	"{{.Module}}/app/views/{{.SlugPlural}}"
	"github.com/daqing/airway/lib/render"
	"github.com/daqing/airway/lib/repo"
	"github.com/daqing/airway/lib/sql"
)

// PageAction renders the interactive {{.SlugPlural}} page.
func PageAction(c *gin.Context) {
	render.HTML(c, {{.SlugPlural}}.Index())
}

// ShowAction renders one {{.Slug}} as a full server-rendered page.
func ShowAction(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		render.ErrorMessage(c, "invalid id")
		return
	}

	item, err := repo.FindByID[models.{{.Name}}](sql.IdType(id))
	if err != nil {
		render.Error(c, err)
		return
	}
	render.HTML(c, {{.SlugPlural}}.Show(item))
}

// IndexAction lists every {{.Slug}} as JSON.
func IndexAction(c *gin.Context) {
	items, err := repo.FindAll[models.{{.Name}}]()
	if err != nil {
		render.Error(c, err)
		return
	}
	render.OK(c, items)
}

type {{.Slug}}Params struct {
{{range .Fields}}	{{.Name}} {{.GoType}} ` + "`json:\"{{.JSON}}\"`" + `
{{end}}}

// CreateAction inserts a {{.Slug}}.
func CreateAction(c *gin.Context) {
	var p {{.Slug}}Params
	if err := c.BindJSON(&p); err != nil {
		render.Error(c, err)
		return
	}

	item, err := repo.CreateFrom[models.{{.Name}}](sql.H{
{{range .Fields}}		"{{.JSON}}": p.{{.Name}},
{{end}}	})
	if err != nil {
		render.Error(c, err)
		return
	}
	render.OK(c, item)
}

// UpdateAction modifies a {{.Slug}} by id.
func UpdateAction(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		render.ErrorMessage(c, "invalid id")
		return
	}

	var p {{.Slug}}Params
	if err := c.BindJSON(&p); err != nil {
		render.Error(c, err)
		return
	}

	if err := repo.UpdateByID[models.{{.Name}}](sql.IdType(id), sql.H{
{{range .Fields}}		"{{.JSON}}": p.{{.Name}},
{{end}}	}); err != nil {
		render.Error(c, err)
		return
	}
	render.Empty(c)
}

// DestroyAction deletes a {{.Slug}} by id.
func DestroyAction(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		render.ErrorMessage(c, "invalid id")
		return
	}

	if err := repo.DeleteByID[models.{{.Name}}](sql.IdType(id)); err != nil {
		render.Error(c, err)
		return
	}
	render.Empty(c)
}
`

const scaffoldViewTemplate = `package {{.SlugPlural}}

import (
	"github.com/daqing/airway/app/assets"
	"github.com/daqing/airway/app/views/layouts"
)

templ Index() {
	@layouts.Base("{{.NamePlural}}") {
		<div class="aw-showcase" style="max-width: 920px; margin: 0 auto; padding: 32px 22px 72px;">
			<h1 style="font-size: 26px; letter-spacing: -0.5px; margin: 0;">{{.NamePlural}}</h1>
			<p style="color: var(--aw-muted, #6b7280); margin: 6px 0 0; font-size: 14px;">
				Scaffolded CRUD — table, modal form and toasts are airway-ui islands talking to /api/v1/{{.SlugPlural}}.
			</p>
			@assets.Island("{{.SlugPlural}}-crud", map[string]any{})
		</div>
	}
}
`

const scaffoldShowTemplate = `package {{.SlugPlural}}

import (
	"fmt"

	"github.com/daqing/airway/app/views/layouts"

	"{{.Module}}/app/models"
)

// Show renders one {{.Slug}} as a complete server-rendered document: the
// content is baked into the HTML, so the page works offline and exports
// to a CDN as-is (see export_{{.SlugPlural}}.go).
templ Show(item *models.{{.Name}}) {
	@layouts.Base(fmt.Sprintf("{{.Name}} #%d", item.ID)) {
		<div class="aw-showcase" style="max-width: 720px; margin: 0 auto; padding: 32px 22px 72px;">
			<h1 style="font-size: 24px; letter-spacing: -0.5px; margin: 0;">{{.Name}} #{ item.ID }</h1>
			<dl style="margin-top: 20px; display: grid; grid-template-columns: 140px 1fr; gap: 8px 16px; font-size: 14px;">
			{{range .Fields}}				<dt style="color: var(--aw-muted, #6b7280);">{{.Name}}</dt>
				<dd style="margin: 0;">{ item.{{.Name}} }</dd>
			{{end}}		</dl>
			<p style="margin-top: 28px;">
				<a href="/{{.SlugPlural}}" style="color: var(--aw-accent, #2563eb);">← All {{.NamePlural}}</a>
			</p>
		</div>
	}
}
`

// scaffoldExportTemplate writes export_<plural>.go in the project root:
// one file per scaffolded resource, so repeated scaffolds never need to
// merge into a shared file. The list page is static; detail pages are
// enumerated from the database at export time through a provider.
const scaffoldExportTemplate = `package main

import (
	"fmt"

	"github.com/daqing/airway/cmd"
	"github.com/daqing/airway/lib/repo"
	"github.com/daqing/airway/lib/static"

	"{{.Module}}/app/models"
	"{{.Module}}/app/views/{{.SlugPlural}}"
)

// Static export for the {{.SlugPlural}} pages (see docs/static-export.md):
// ` + "`airway static:build`" + ` renders them next to the frontend bundle.
func init() {
	cmd.SetStaticPages(
		static.Page{Slug: "/{{.SlugPlural}}", Component: {{.SlugPlural}}.Index()},
	)

	// Detail pages are enumerated at export time — one per {{.Slug}}.
	// AIRWAY_DSN must be configured when running static:build/static:serve.
	cmd.SetStaticPagesProvider(func() ([]static.Page, error) {
		items, err := repo.FindAll[models.{{.Name}}]()
		if err != nil {
			return nil, fmt.Errorf("enumerate {{.SlugPlural}}: %w", err)
		}

		pages := make([]static.Page, 0, len(items))
		for _, item := range items {
			pages = append(pages, static.Page{
				Slug:      fmt.Sprintf("/{{.SlugPlural}}/%d", item.ID),
				Component: {{.SlugPlural}}.Show(item),
			})
		}
		return pages, nil
	})
}
`

const scaffoldIslandTemplate = `import { useCallback, useEffect, useState } from "react";
import { useForm } from "react-hook-form";
import type { ColumnDef } from "@tanstack/react-table";

import {
  apiFetch,
  Button,
  DataTable,
  Field,
  Form,
  Input,
  Modal,
  useToast,
} from "../ui";

type Row = {
  id: number;
{{range .Fields}}  {{.JSON}}: {{if eq .GoType "string"}}string{{else if eq .GoType "int64"}}number{{else if eq .GoType "float64"}}number{{else}}boolean{{end}};
{{end}}  created_at?: string;
};

type FormValues = {
{{range .Fields}}  {{.JSON}}: string;
{{end}}};

export default function {{.NamePlural}}Crud() {
  const toast = useToast();
  const [rows, setRows] = useState<Row[]>([]);
  const [loading, setLoading] = useState(true);
  const [editing, setEditing] = useState<Row | "new" | null>(null);
  const form = useForm<FormValues>();

  const load = useCallback(async () => {
    setLoading(true);
    try {
      setRows(await apiFetch<Row[]>("/api/v1/{{.SlugPlural}}"));
    } catch (e) {
      toast.error((e as Error).message);
    } finally {
      setLoading(false);
    }
  }, [toast]);

  useEffect(() => { load(); }, [load]);

  const openNew = () => {
    form.reset({{"{"}}{{range $i, $f := .Fields}}{{if $i}}, {{end}}{{$f.JSON}}: ""{{end}}{{"}"}});
    setEditing("new");
  };

  const openEdit = (row: Row) => {
    form.reset({{"{"}}{{range $i, $f := .Fields}}{{if $i}}, {{end}}{{$f.JSON}}: String(row.{{$f.JSON}}){{end}}{{"}"}});
    setEditing(row);
  };

  const save = async (values: FormValues) => {
    try {
      if (editing === "new") {
        await apiFetch("/api/v1/{{.SlugPlural}}", { method: "POST", body: JSON.stringify(values) });
      } else if (editing) {
        await apiFetch("/api/v1/{{.SlugPlural}}/" + editing.id, { method: "PUT", body: JSON.stringify(values) });
      }
      toast.success(editing === "new" ? "Created" : "Updated");
      setEditing(null);
      await load();
    } catch (e) {
      toast.error((e as Error).message);
    }
  };

  const remove = async (row: Row) => {
    try {
      await apiFetch("/api/v1/{{.SlugPlural}}/" + row.id, { method: "DELETE" });
      toast.show("Deleted " + row.id, "error");
      await load();
    } catch (e) {
      toast.error((e as Error).message);
    }
  };

  const columns: ColumnDef<Row, any>[] = [
{{range .Fields}}    { accessorKey: "{{.JSON}}", header: "{{.Name}}" },
{{end}}    {
      id: "actions",
      header: "",
      cell: ({ row }) => (
        <span style="display: inline-flex; gap: 6px;">
          <Button size="sm" onClick={() => openEdit(row.original)}>Edit</Button>
          <Button size="sm" variant="danger" onClick={() => remove(row.original)}>Delete</Button>
        </span>
      ),
    },
  ];

  const editTitle = editing === "new" ? "New {{.Slug}}" : editing ? "Edit #" + editing.id : "";

  return (
    <div style="margin-top: 22px;">
      <div style="margin-bottom: 10px;">
        <Button variant="primary" onClick={openNew}>New {{.Slug}}</Button>
      </div>
      <DataTable columns={columns} data={rows} pageSize={10} loading={loading} />

      <Modal open={editing !== null} onClose={() => setEditing(null)} title={editTitle}>
        <Form
          form={form}
          submitLabel={editing === "new" ? "Create" : "Save"}
          onSubmit={save}
          secondary={<Button onClick={() => setEditing(null)}>Cancel</Button>}
        >
{{range .Fields}}          <Field label="{{.Name}}" required error={form.formState.errors.{{.JSON}}?.message}>
            {(id: string) => (
              <Input id={id} invalid={!!form.formState.errors.{{.JSON}}} {...form.register("{{.JSON}}", { required: "{{.Name}} is required" })} />
            )}
          </Field>
{{end}}        </Form>
      </Modal>
    </div>
  );
}
`

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

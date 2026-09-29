package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// --- generate island ---

func generateIsland(args []string) error {
	if len(args) == 1 && isHelpArg(args[0]) {
		fmt.Println("usage: airway generate island <name>")
		return nil
	}
	if len(args) != 1 {
		return fmt.Errorf("usage: airway generate island <name>")
	}

	name := strings.ToLower(strings.TrimSpace(args[0]))
	target := filepath.Join(".", "app", "assets", "js", "islands", name+".tsx")
	data := islandTemplateData{Name: toCamelName(name), Slug: name}

	fmt.Println("Next steps:")
	fmt.Printf("  1. run `go run . js:build` (or restart `just dev`)\n")
	fmt.Printf("  2. embed it in a view: @assets.Island(%q, props)\n", name)
	return writeTemplateFile(islandTemplate, target, data)
}

const islandTemplate = `import type { ComponentChild } from "react";

// {{.Slug}} island — initial props arrive as JSON from the server-rendered
// page. Build the UI from the airway-ui components under ../ui.
export default function {{.Name}}(props: Record<string, unknown>): ComponentChild {
  const message = typeof props.message === "string" ? props.message : "{{.Slug}} island";
  return <div>{message}</div>;
}
`

type islandTemplateData struct {
	Name string
	Slug string
}

// --- generate scaffold ---

func generateScaffold(args []string) error {
	if len(args) == 1 && isHelpArg(args[0]) {
		fmt.Println("usage: airway generate scaffold <name> <field:type> <field:type>...")
		return nil
	}
	if len(args) < 2 {
		return fmt.Errorf("usage: airway generate scaffold <name> <field:type> <field:type>...")
	}

	name := strings.ToLower(strings.TrimSpace(args[0]))
	plural := pluralize(name)
	apiName := plural + "_api"
	camel := toCamelName(name)
	camelPlural := toCamelName(plural)

	fields, err := parseScaffoldFields(args[1:])
	if err != nil {
		return err
	}

	data := scaffoldData{
		Name:       camel,
		NamePlural: camelPlural,
		Slug:       name,
		SlugPlural: plural,
		APIName:    apiName,
		Module:     currentModulePath(),
		Fields:     fields,
	}

	// model with the declared fields
	if err := writeTemplateFile(scaffoldModelTemplate,
		filepath.Join(".", "app", "models", name+".go"), data); err != nil {
		return err
	}

	// Go DSL migration, portable across PostgreSQL, MySQL and SQLite
	version := nextMigrationVersion("create_" + plural)
	mig := scaffoldMigrationData{Version: version, scaffoldData: data}
	if err := writeTemplateFile(scaffoldMigrationTemplate,
		filepath.Join(".", "db", "migrate", version+"_create_"+plural+".go"), mig); err != nil {
		return err
	}
	noteMigrationImport(data.Module)

	// service layer (same as `generate service`)
	if err := generateService(args); err != nil {
		return err
	}

	// JSON API + HTML page actions
	apiDir := filepath.Join(".", "app", "api", apiName)
	if err := ensureDir(apiDir); err != nil {
		return err
	}
	if err := writeTemplateFile(scaffoldRoutesTemplate,
		filepath.Join(apiDir, "routes.go"), data); err != nil {
		return err
	}
	if err := writeTemplateFile(scaffoldActionsTemplate,
		filepath.Join(apiDir, plural+"_action.go"), data); err != nil {
		return err
	}

	// OpenAPI declarations so the new resource is documented from birth
	if err := writeTemplateFile(scaffoldOpenAPITemplate,
		filepath.Join(apiDir, "openapi.go"), data); err != nil {
		return err
	}

	// server-rendered pages: the interactive list (CRUD island) and the
	// detail page with the row's content baked into the HTML
	if err := writeTemplateFile(scaffoldViewTemplate,
		filepath.Join(".", "app", "views", plural, "index.templ"), data); err != nil {
		return err
	}
	if err := writeTemplateFile(scaffoldShowTemplate,
		filepath.Join(".", "app", "views", plural, "show.templ"), data); err != nil {
		return err
	}

	// the interactive island (DataTable + modal form, talking to the JSON API)
	if err := writeTemplateFile(scaffoldIslandTemplate,
		filepath.Join(".", "app", "assets", "js", "islands", plural+"-crud.tsx"), data); err != nil {
		return err
	}

	// static export registration (one file per resource, so repeated
	// scaffolds never need to merge into a shared file)
	if err := writeTemplateFile(scaffoldExportTemplate,
		filepath.Join(".", fmt.Sprintf("export_%s.go", plural)), data); err != nil {
		return err
	}

	// keep the project compilable: the new views only have .templ files, and
	// the root package imports them through export_<resource>.go
	compileViewsOrNote()

	registerScaffoldRoutes(data)

	fmt.Println("\nNext steps:")
	fmt.Println("  airway templates:compile   # compile the .templ views")
	fmt.Println("  airway js:build            # bundle the new island")
	fmt.Println("  airway db:migrate          # create the table")
	fmt.Println("  airway static:build        # export the pages + detail pages as static HTML (needs DSN; see docs/static-export.md)")
	fmt.Println("  airway server              # visit /" + plural)
	return nil
}

type scaffoldField struct {
	Name   string // Title
	JSON   string // title
	GoType string // string
	DSL    string // t.String("title")
}

type scaffoldData struct {
	Name       string
	NamePlural string
	Slug       string
	SlugPlural string
	APIName    string
	Module     string
	Fields     []scaffoldField
}

type scaffoldMigrationData struct {
	scaffoldData
	Version string
}

func parseScaffoldFields(args []string) ([]scaffoldField, error) {
	var fields []scaffoldField
	for _, arg := range args {
		fieldName, fieldType, err := parseFieldArg(arg)
		if err != nil {
			return nil, err
		}
		var goType, dsl string
		switch strings.ToLower(fieldType) {
		case "string":
			goType, dsl = "string", fmt.Sprintf("t.String(%q)", fieldName)
		case "text":
			goType, dsl = "string", fmt.Sprintf("t.Text(%q)", fieldName)
		case "integer", "int":
			goType, dsl = "int64", fmt.Sprintf("t.BigInt(%q)", fieldName)
		case "float":
			goType, dsl = "float64", fmt.Sprintf("t.Float(%q)", fieldName)
		case "boolean", "bool":
			goType, dsl = "bool", fmt.Sprintf("t.Boolean(%q)", fieldName)
		default:
			return nil, fmt.Errorf("unsupported field type %q (use string, text, integer, float or boolean)", fieldType)
		}
		fields = append(fields, scaffoldField{
			Name:   toCamelName(fieldName),
			JSON:   fieldName,
			GoType: goType,
			DSL:    dsl,
		})
	}
	return fields, nil
}

func pluralize(name string) string {
	if strings.HasSuffix(name, "y") && len(name) > 1 && !isVowel(name[len(name)-2]) {
		return name[:len(name)-1] + "ies"
	}
	if strings.HasSuffix(name, "s") || strings.HasSuffix(name, "x") || strings.HasSuffix(name, "ch") || strings.HasSuffix(name, "sh") {
		return name + "es"
	}
	return name + "s"
}

func isVowel(b byte) bool {
	switch b {
	case 'a', 'e', 'i', 'o', 'u':
		return true
	}
	return false
}

// registerScaffoldRoutes wires the new API module into config/routes.go.
// It edits the file in place; on any surprise it falls back to printing the
// manual snippet instead of failing the scaffold.
func registerScaffoldRoutes(data scaffoldData) {
	path := filepath.Join(".", "config", "routes.go")
	raw, err := os.ReadFile(path)
	if err != nil {
		printRouteSnippet(data)
		return
	}
	content := string(raw)

	importLine := fmt.Sprintf("\t\"%s/app/api/%s\"", data.Module, data.APIName)
	mountLine := fmt.Sprintf("\t%s.Routes(r)", data.APIName)

	if strings.Contains(content, mountLine) {
		return // already registered
	}

	anchor := "plugin.MountAll(r)"
	if !strings.Contains(content, anchor) {
		printRouteSnippet(data)
		return
	}
	content = strings.Replace(content, anchor, mountLine+"\n\n\t"+anchor, 1)

	// place the import with the other app/api imports
	importAnchor := "app/api/health_api\""
	if !strings.Contains(content, importAnchor) {
		printRouteSnippet(data)
		return
	}
	content = strings.Replace(content,
		"import (\n",
		"import (\n"+importLine+"\n", 1)

	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		printRouteSnippet(data)
		return
	}
	fmt.Printf("Updated config/routes.go (registered %s)\n", data.APIName)
}

func printRouteSnippet(data scaffoldData) {
	fmt.Printf("\nAdd to config/routes.go PublicRoutes:\n")
	fmt.Printf("  import \"%s/app/api/%s\"\n", data.Module, data.APIName)
	fmt.Printf("  %s.Routes(r)\n", data.APIName)
}

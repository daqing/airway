package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
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
		IDColumn:   idColumnForDSN(),
	}

	// model with the declared fields
	if err := writeTemplateFile(scaffoldModelTemplate,
		filepath.Join(".", "app", "models", name+".go"), data); err != nil {
		return err
	}

	// migration pair, tailored to the configured database dialect
	version := time.Now().Format("20060102150405")
	mig := scaffoldMigrationData{Version: version, scaffoldData: data}
	mig.Version = version
	migName := fmt.Sprintf("%s_create_%s", version, plural)
	if err := writeTemplateFile(scaffoldUpTemplate,
		filepath.Join(".", "db", "migrate", migName+".up.sql"), mig); err != nil {
		return err
	}
	if err := writeTemplateFile(scaffoldDownTemplate,
		filepath.Join(".", "db", "migrate", migName+".down.sql"), mig); err != nil {
		return err
	}

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
	Name    string // Title
	JSON    string // title
	GoType  string // string
	SQLCol  string // title VARCHAR(255)
	SQLName string // title
}

type scaffoldData struct {
	Name       string
	NamePlural string
	Slug       string
	SlugPlural string
	APIName    string
	Module     string
	IDColumn   string
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
		var goType, sqlType string
		switch strings.ToLower(fieldType) {
		case "string":
			goType, sqlType = "string", "VARCHAR(255)"
		case "text":
			goType, sqlType = "string", "TEXT"
		case "integer", "int":
			goType, sqlType = "int64", "BIGINT"
		case "float":
			goType, sqlType = "float64", "DOUBLE PRECISION"
		case "boolean", "bool":
			goType, sqlType = "bool", "BOOLEAN"
		default:
			return nil, fmt.Errorf("unsupported field type %q (use string, text, integer, float or boolean)", fieldType)
		}
		fields = append(fields, scaffoldField{
			Name:    toCamelName(fieldName),
			JSON:    fieldName,
			GoType:  goType,
			SQLCol:  fieldName + " " + sqlType,
			SQLName: fieldName,
		})
	}
	return fields, nil
}

// idColumnForDSN renders the auto-increment primary key for the database
// the project is currently configured for (scaffold migrations are
// dialect-specific).
func idColumnForDSN() string {
	dsn, err := cliDSN()
	if err != nil {
		return "id INTEGER PRIMARY KEY AUTOINCREMENT"
	}
	lower := strings.ToLower(dsn)
	switch {
	case strings.HasPrefix(lower, "postgres://"), strings.HasPrefix(lower, "postgresql://"):
		return "id BIGSERIAL PRIMARY KEY"
	case strings.HasPrefix(lower, "mysql://"), strings.Contains(lower, "@tcp("):
		return "id BIGINT NOT NULL AUTO_INCREMENT PRIMARY KEY"
	default:
		return "id INTEGER PRIMARY KEY AUTOINCREMENT"
	}
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

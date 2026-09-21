package cmd

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/pelletier/go-toml/v2"
)

// --- admin:generate: config model ---

// Supported field kinds in config/admin.toml.
const (
	adminKindString     = "string"
	adminKindText       = "text"
	adminKindInteger    = "integer"
	adminKindFloat      = "float"
	adminKindBoolean    = "boolean"
	adminKindDatetime   = "datetime"
	adminKindEnum       = "enum"
	adminKindReferences = "references"
	adminKindAttachment = "attachment"
)

type adminField struct {
	Name        string   // Go field name: CategoryID
	JSON        string   // snake_case column / JSON name: category_id
	DisplayName string   // form / table label: Category
	Kind        string   // one of the adminKind* constants
	Options     []string // enum values
	RefTarget   string   // referenced table slug (singular)
	RefGoName   string   // referenced model name, for Relations()
	RefPlural   string   // referenced SQL table, for the island's remote select
	GoType      string   // model / params field type
	SQLCol      string   // column DDL fragment (table-level FK lines are added separately)
}

type adminTable struct {
	Slug       string // singular: post
	Name       string // CamelCase singular: Post
	SlugPlural string // SQL table: posts
	NamePlural string // CamelCase plural: Posts
	Fields     []adminField
	HasRefs    bool
	// HasDatetime is set when any field is a datetime: the generated
	// resource file needs the time import for the *time.Time params.
	HasDatetime bool
}

type adminConfig struct {
	Tables []*adminTable // slug-alphabetical
}

var adminReservedFields = map[string]bool{
	"id":         true,
	"created_at": true,
	"updated_at": true,
}

func parseAdminConfig(data []byte) (*adminConfig, error) {
	var raw map[string]map[string]string
	if err := toml.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("invalid admin config: %w", err)
	}

	slugs := make([]string, 0, len(raw))
	for slug := range raw {
		slugs = append(slugs, slug)
	}
	sort.Strings(slugs)

	cfg := &adminConfig{}
	seenPlurals := map[string]string{}
	for _, slug := range slugs {
		if !validAdminIdentifier(slug) {
			return nil, fmt.Errorf("invalid table name %q: use lowercase snake_case identifiers", slug)
		}
		rawFields := raw[slug]
		if len(rawFields) == 0 {
			return nil, fmt.Errorf("table [%s] declares no fields", slug)
		}

		table := &adminTable{
			Slug:       slug,
			Name:       toCamelName(slug),
			SlugPlural: pluralize(slug),
			NamePlural: toCamelName(pluralize(slug)),
		}
		if other, dup := seenPlurals[table.SlugPlural]; dup {
			return nil, fmt.Errorf("tables [%s] and [%s] both map to SQL table %q", other, slug, table.SlugPlural)
		}
		seenPlurals[table.SlugPlural] = slug

		fieldNames := make([]string, 0, len(rawFields))
		for name := range rawFields {
			fieldNames = append(fieldNames, name)
		}
		sort.Strings(fieldNames)

		for _, name := range fieldNames {
			field, err := parseAdminField(name, rawFields[name])
			if err != nil {
				return nil, fmt.Errorf("table [%s]: %w", slug, err)
			}
			table.Fields = append(table.Fields, field)
		}

		cfg.Tables = append(cfg.Tables, table)
	}

	if len(cfg.Tables) == 0 {
		return nil, fmt.Errorf("admin config declares no tables")
	}

	bySlug := map[string]*adminTable{}
	for _, table := range cfg.Tables {
		bySlug[table.Slug] = table
	}

	// Resolve references targets after all tables are known, so forward
	// references ([post] before [category]) just work.
	for _, table := range cfg.Tables {
		for i := range table.Fields {
			field := &table.Fields[i]
			switch field.Kind {
			case adminKindReferences:
				target := field.RefTarget
				if target == "" {
					target = strings.TrimSuffix(field.JSON, "_id")
				}
				ref, ok := bySlug[target]
				if !ok {
					return nil, fmt.Errorf("%s.%s references unknown table %q", table.Slug, field.JSON, target)
				}
				field.RefTarget = target
				field.RefGoName = ref.Name
				field.RefPlural = ref.SlugPlural
				table.HasRefs = true
			case adminKindDatetime:
				table.HasDatetime = true
			}
		}
	}

	return cfg, nil
}

func parseAdminField(name, spec string) (adminField, error) {
	if !validAdminIdentifier(name) {
		return adminField{}, fmt.Errorf("invalid field name %q: use lowercase snake_case identifiers", name)
	}
	if adminReservedFields[name] {
		return adminField{}, fmt.Errorf("field %q is reserved (id, created_at and updated_at are generated automatically)", name)
	}

	spec = strings.TrimSpace(spec)
	head, arg, hasArg := strings.Cut(spec, ":")
	head = strings.ToLower(strings.TrimSpace(head))
	arg = strings.TrimSpace(arg)

	field := adminField{
		JSON:        name,
		Name:        toCamelName(name),
		DisplayName: toCamelName(name),
	}

	switch head {
	case adminKindString:
		field.Kind, field.GoType, field.SQLCol = adminKindString, "string", name+" VARCHAR(255)"
	case adminKindText:
		field.Kind, field.GoType, field.SQLCol = adminKindText, "string", name+" TEXT"
	case adminKindInteger, "int":
		field.Kind, field.GoType, field.SQLCol = adminKindInteger, "int64", name+" BIGINT"
	case adminKindFloat:
		field.Kind, field.GoType, field.SQLCol = adminKindFloat, "float64", name+" DOUBLE PRECISION"
	case adminKindBoolean, "bool":
		field.Kind, field.GoType, field.SQLCol = adminKindBoolean, "bool", name+" BOOLEAN"
	case adminKindDatetime:
		field.Kind, field.GoType, field.SQLCol = adminKindDatetime, "*time.Time", name+" TIMESTAMP"
	case adminKindEnum:
		if !hasArg {
			return adminField{}, fmt.Errorf("field %q: enum requires options (e.g. %q)", name, "status = \"enum:draft,published\"")
		}
		options, err := parseAdminEnumOptions(arg)
		if err != nil {
			return adminField{}, fmt.Errorf("field %q: %w", name, err)
		}
		quoted := make([]string, 0, len(options))
		for _, option := range options {
			quoted = append(quoted, "'"+option+"'")
		}
		// *string so a cleared select round-trips as SQL NULL (the column
		// carries a CHECK constraint that empty strings would violate).
		field.Kind = adminKindEnum
		field.GoType = "*string"
		field.Options = options
		field.SQLCol = name + " VARCHAR(255) CHECK (" + name + " IN (" + strings.Join(quoted, ", ") + "))"
	case adminKindReferences, "ref":
		field.Kind = adminKindReferences
		field.GoType = "*int64"
		field.SQLCol = name + " BIGINT"
		field.DisplayName = toCamelName(strings.TrimSuffix(name, "_id"))
		if hasArg {
			if !validAdminIdentifier(arg) {
				return adminField{}, fmt.Errorf("field %q: invalid references target %q", name, arg)
			}
			field.RefTarget = arg
		}
	case adminKindAttachment:
		field.Kind, field.GoType, field.SQLCol = adminKindAttachment, "string", name+" VARCHAR(255)"
	default:
		return adminField{}, fmt.Errorf(
			"field %q has unknown type %q (use string, text, integer, float, boolean, datetime, enum:<a,b,c>, references[:table] or attachment)",
			name, head)
	}

	return field, nil
}

func parseAdminEnumOptions(arg string) ([]string, error) {
	seen := map[string]bool{}
	var options []string
	for _, raw := range strings.Split(arg, ",") {
		option := strings.TrimSpace(raw)
		if option == "" {
			return nil, fmt.Errorf("enum options must not be empty")
		}
		if !validAdminIdentifier(option) {
			return nil, fmt.Errorf("invalid enum option %q: use letters, digits, dashes and underscores", option)
		}
		if seen[option] {
			return nil, fmt.Errorf("duplicate enum option %q", option)
		}
		seen[option] = true
		options = append(options, option)
	}
	if len(options) == 0 {
		return nil, fmt.Errorf("enum requires at least one option")
	}
	return options, nil
}

func validAdminIdentifier(value string) bool {
	if value == "" {
		return false
	}
	for i := 0; i < len(value); i++ {
		c := value[i]
		switch {
		case c >= 'a' && c <= 'z', c >= '0' && c <= '9':
		case c == '_':
			if i == 0 {
				return false
			}
		case c >= 'A' && c <= 'Z':
			return false
		default:
			return false
		}
	}
	return true
}

// topoOrder returns the tables so that referenced tables come before the
// tables that reference them — the order the CREATE TABLE statements need.
// Self-references are allowed; cycles between tables are a hard error.
func (cfg *adminConfig) topoOrder() ([]*adminTable, error) {
	bySlug := map[string]*adminTable{}
	deps := map[string][]string{}
	for _, table := range cfg.Tables {
		bySlug[table.Slug] = table
		set := map[string]bool{}
		for _, field := range table.Fields {
			if field.Kind == adminKindReferences && field.RefTarget != table.Slug {
				set[field.RefTarget] = true
			}
		}
		sorted := make([]string, 0, len(set))
		for dep := range set {
			sorted = append(sorted, dep)
		}
		sort.Strings(sorted)
		deps[table.Slug] = sorted
	}

	var ordered []*adminTable
	state := map[string]int{} // 1 = visiting, 2 = done

	var visit func(slug string) error
	visit = func(slug string) error {
		switch state[slug] {
		case 1:
			return fmt.Errorf("circular references involving table %q", slug)
		case 2:
			return nil
		}
		state[slug] = 1
		for _, dep := range deps[slug] {
			if err := visit(dep); err != nil {
				return err
			}
		}
		state[slug] = 2
		ordered = append(ordered, bySlug[slug])
		return nil
	}

	for _, table := range cfg.Tables {
		if err := visit(table.Slug); err != nil {
			return nil, err
		}
	}
	return ordered, nil
}

// --- admin:generate: entry point ---

func generateAdmin(args []string) error {
	if len(args) == 1 && isHelpArg(args[0]) {
		fmt.Println("usage: airway admin:generate [config/admin.toml]")
		return nil
	}
	if len(args) > 1 {
		return fmt.Errorf("usage: airway admin:generate [config/admin.toml]")
	}

	path := filepath.Join("config", "admin.toml")
	if len(args) == 1 {
		path = args[0]
	}

	raw, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	cfg, err := parseAdminConfig(raw)
	if err != nil {
		return err
	}
	ordered, err := cfg.topoOrder()
	if err != nil {
		return err
	}

	module := currentModulePath()
	idColumn := idColumnForDSN()

	var created []string
	write := func(tpl string, rel string, data any) error {
		if err := writeTemplateFile(tpl, rel, data); err != nil {
			if errors.Is(err, os.ErrExist) {
				fmt.Println("skipped (exists): " + rel)
				return nil
			}
			return err
		}
		fmt.Println("created: " + rel)
		created = append(created, rel)
		return nil
	}

	// Models (per table; a table whose model already exists is considered
	// already generated and skipped entirely, migrations included).
	authExists := adminPathExists("app", "models", "admin_user.go")
	freshSet := map[string]bool{}
	for _, table := range cfg.Tables {
		if adminPathExists("app", "models", table.Slug+".go") {
			fmt.Printf("skipped (table %q already generated)\n", table.Slug)
			continue
		}
		freshSet[table.Slug] = true

		if err := write(adminModelTemplate,
			filepath.Join("app", "models", table.Slug+".go"),
			adminModelData{Module: module, adminTable: table}); err != nil {
			return err
		}
	}

	// Fresh tables in reference dependency order — the order the migration
	// needs to create them in.
	var fresh []*adminTable
	for _, table := range ordered {
		if freshSet[table.Slug] {
			fresh = append(fresh, table)
		}
	}

	// Auth models, middleware and the admin_api module shell — all once.
	for _, file := range []struct {
		tpl  string
		rel  string
		data any
	}{
		{adminAuthUserModelTemplate, filepath.Join("app", "models", "admin_user.go"), nil},
		{adminAuthSessionModelTemplate, filepath.Join("app", "models", "admin_session.go"), nil},
		{adminMiddlewareTemplate, filepath.Join("app", "middlewares", "admin_auth.go"),
			adminModuleData{Module: module}},
		{adminRoutesTemplate, filepath.Join("app", "api", "admin_api", "routes.go"),
			adminModuleData{Module: module}},
		{adminRegistryTemplate, filepath.Join("app", "api", "admin_api", "registry.go"),
			adminModuleData{Module: module}},
		{adminAuthActionsTemplate, filepath.Join("app", "api", "admin_api", "auth_action.go"),
			adminModuleData{Module: module}},
		{adminUploadsTemplate, filepath.Join("app", "api", "admin_api", "uploads_action.go"), nil},
		{adminOpenAPITemplate, filepath.Join("app", "api", "admin_api", "openapi.go"), nil},
		{adminTypesTemplate, filepath.Join("app", "views", "admin", "types.go"), nil},
		{adminLayoutTemplate, filepath.Join("app", "views", "admin", "admin.templ"), nil},
		{adminDashboardTemplate, filepath.Join("app", "views", "admin", "index.templ"), nil},
		{adminLoginTemplate, filepath.Join("app", "views", "admin", "login.templ"), nil},
	} {
		if file.data == nil {
			file.data = adminModuleData{Module: module}
		}
		if err := write(file.tpl, file.rel, file.data); err != nil {
			return err
		}
	}

	// Per-table admin_api resource, admin page and CRUD island.
	for _, table := range fresh {
		if err := write(adminResourceTemplate,
			filepath.Join("app", "api", "admin_api", table.Slug+"_resource.go"),
			adminResourceData{Module: module, adminTable: table}); err != nil {
			return err
		}

		if err := write(adminPageTemplate,
			filepath.Join("app", "views", "admin", table.SlugPlural+"_page.templ"),
			adminPageData{adminTable: table}); err != nil {
			return err
		}

		if err := write(adminIslandTemplate,
			filepath.Join("app", "assets", "js", "islands", "admin-"+table.SlugPlural+"-crud.tsx"),
			islandData(table)); err != nil {
			return err
		}
	}

	// One migration pair per run: auth tables on the first run, then every
	// newly generated table in reference dependency order.
	if authExists && len(fresh) == 0 {
		fmt.Println("skipped migration: no new tables")
	} else {
		reversed := make([]*adminTable, len(fresh))
		for i, table := range fresh {
			reversed[len(fresh)-1-i] = table
		}
		mig := adminMigrationData{
			Module:   module,
			IDColumn: idColumn,
			Auth:     !authExists,
			Tables:   fresh,
			Reversed: reversed,
		}
		// Migration filenames order db:migrate, so on a same-second re-run
		// bump the timestamp forward instead of colliding with the existing
		// pair.
		stamp := timeNow()
		name := stamp.Format("20060102150405") + "_create_admin_tables"
		for adminPathExists("db", "migrate", name+".up.sql") {
			stamp = stamp.Add(time.Second)
			name = stamp.Format("20060102150405") + "_create_admin_tables"
		}
		if err := write(adminMigrationUpTemplate,
			filepath.Join("db", "migrate", name+".up.sql"), mig); err != nil {
			return err
		}
		if err := write(adminMigrationDownTemplate,
			filepath.Join("db", "migrate", name+".down.sql"), mig); err != nil {
			return err
		}
	}

	registerAdminRoutes(module)

	fmt.Println("\nNext steps:")
	fmt.Println("  airway templates:compile   # compile the .templ views")
	fmt.Println("  airway js:build            # bundle the admin islands")
	fmt.Println("  airway db:migrate          # create the admin tables")
	fmt.Println("  airway admin:user <email> <password>   # create the first admin")
	fmt.Println("  airway server              # visit /admin")
	return nil
}

type adminModuleData struct {
	Module string
}

type adminModelData struct {
	Module string
	*adminTable
}

type adminResourceData struct {
	Module string
	*adminTable
}

type adminPageData struct {
	*adminTable
}

type adminMigrationData struct {
	Module   string
	IDColumn string
	Auth     bool
	Tables   []*adminTable // topo order for the up migration
	Reversed []*adminTable // children-first for the down migration
}

func adminPathExists(parts ...string) bool {
	_, err := os.Stat(filepath.Join(parts...))
	return err == nil
}

// registerAdminRoutes wires admin_api.Routes into config/routes.go, once.
// Like the scaffold's wiring helper it falls back to printing the manual
// snippet instead of failing the whole run.
func registerAdminRoutes(module string) {
	path := filepath.Join(".", "config", "routes.go")
	raw, err := os.ReadFile(path)
	if err != nil {
		printAdminRouteSnippet(module)
		return
	}
	content := string(raw)

	mountLine := "\tadmin_api.Routes(r)"
	if strings.Contains(content, mountLine) {
		return
	}

	anchor := "plugin.MountAll(r)"
	if !strings.Contains(content, anchor) {
		printAdminRouteSnippet(module)
		return
	}
	// The anchor's own leading tab stays, so the mount line needs none.
	content = strings.Replace(content, anchor, "admin_api.Routes(r)\n\t"+anchor, 1)

	adminImport := fmt.Sprintf("\t\"%s/app/api/admin_api\"\n", module)
	switch {
	case strings.Contains(content, "import (\n"):
		content = strings.Replace(content, "import (\n", "import (\n"+adminImport, 1)
	case strings.Contains(content, "import \""):
		idx := strings.Index(content, "import \"")
		lineEnd := strings.Index(content[idx:], "\n")
		if lineEnd < 0 {
			printAdminRouteSnippet(module)
			return
		}
		line := content[idx : idx+lineEnd]
		quoted := strings.TrimSpace(strings.TrimPrefix(line, "import "))
		block := "import (\n" + adminImport + "\n\t" + quoted + "\n)"
		content = content[:idx] + block + content[idx+lineEnd:]
	default:
		printAdminRouteSnippet(module)
		return
	}

	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		printAdminRouteSnippet(module)
		return
	}
	fmt.Println("Updated config/routes.go (registered admin_api)")
}

func printAdminRouteSnippet(module string) {
	fmt.Println("\nAdd to config/routes.go PublicRoutes:")
	fmt.Printf("  import %q\n", module+"/app/api/admin_api")
	fmt.Println("  admin_api.Routes(r)")
}

// --- island template data ---

type adminIslandField struct {
	adminField
	TSRowType string
	TSPayload string
	TSPrefill string
	TSColumn  string
	TSControl string
}

type adminRefTarget struct {
	Singular string
	Plural   string
}

type adminIslandData struct {
	*adminTable
	// Fields carries precomputed TypeScript fragments per field, so the
	// template stays a straight-line file instead of branching per kind.
	Fields        []adminIslandField
	RefTargets    []adminRefTarget
	HasDatetime   bool
	HasAttachment bool
	// UploadsReset is the statement restoring attachment state when opening
	// the edit dialog, e.g. `setUploads({ cover: row.cover ?? "" });`.
	UploadsReset string
}

func islandFields(table *adminTable) []adminIslandField {
	fields := make([]adminIslandField, 0, len(table.Fields))
	for _, field := range table.Fields {
		f := adminIslandField{adminField: field}
		f.TSRowType = islandRowType(field)
		f.TSPayload = islandPayload(field)
		f.TSPrefill = islandPrefill(field)
		f.TSColumn = islandColumn(field)
		f.TSControl = islandControl(field)
		fields = append(fields, f)
	}
	return fields
}

func refTargets(table *adminTable) []adminRefTarget {
	var targets []adminRefTarget
	seen := map[string]bool{}
	for _, field := range table.Fields {
		if field.Kind != adminKindReferences || seen[field.RefPlural] {
			continue
		}
		seen[field.RefPlural] = true
		targets = append(targets, adminRefTarget{Singular: field.RefTarget, Plural: field.RefPlural})
	}
	return targets
}

// islandData assembles the complete template payload for a CRUD island.
// HasRefs comes promoted from the table (set while resolving references).
func islandData(table *adminTable) adminIslandData {
	data := adminIslandData{
		adminTable: table,
		Fields:     islandFields(table),
		RefTargets: refTargets(table),
	}
	for _, field := range table.Fields {
		switch field.Kind {
		case adminKindDatetime:
			data.HasDatetime = true
		case adminKindAttachment:
			data.HasAttachment = true
		}
	}
	if data.HasAttachment {
		var parts []string
		for _, field := range table.Fields {
			if field.Kind == adminKindAttachment {
				parts = append(parts, fmt.Sprintf("%s: row.%s ?? \"\"", field.JSON, field.JSON))
			}
		}
		data.UploadsReset = "setUploads({ " + strings.Join(parts, ", ") + " });"
	}
	return data
}

func islandRowType(field adminField) string {
	switch field.Kind {
	case adminKindInteger, adminKindFloat:
		return "number"
	case adminKindBoolean:
		return "boolean"
	case adminKindDatetime, adminKindEnum:
		return "string | null"
	case adminKindReferences:
		return "number | null"
	default:
		return "string"
	}
}

func islandPayload(field adminField) string {
	switch field.Kind {
	case adminKindEnum:
		return fmt.Sprintf("values.%s === \"\" ? null : values.%s", field.JSON, field.JSON)
	case adminKindInteger, adminKindFloat:
		return fmt.Sprintf("Number(values.%s)", field.JSON)
	case adminKindBoolean:
		return fmt.Sprintf("values.%s === \"true\"", field.JSON)
	case adminKindDatetime:
		return fmt.Sprintf("values.%s === \"\" ? null : new Date(values.%s).toISOString()", field.JSON, field.JSON)
	case adminKindReferences:
		return fmt.Sprintf("values.%s === \"\" ? null : Number(values.%s)", field.JSON, field.JSON)
	default:
		return fmt.Sprintf("values.%s", field.JSON)
	}
}

func islandPrefill(field adminField) string {
	switch field.Kind {
	case adminKindBoolean:
		return fmt.Sprintf("row.%s ? \"true\" : \"false\"", field.JSON)
	case adminKindDatetime:
		return fmt.Sprintf("toLocalInput(row.%s)", field.JSON)
	case adminKindReferences:
		return fmt.Sprintf("row.%s == null ? \"\" : String(row.%s)", field.JSON, field.JSON)
	default:
		return fmt.Sprintf("String(row.%s ?? \"\")", field.JSON)
	}
}

func islandColumn(field adminField) string {
	base := fmt.Sprintf("{ accessorKey: %q, header: %q", field.JSON, field.DisplayName)
	switch field.Kind {
	case adminKindBoolean:
		return base + fmt.Sprintf(", cell: ({ row }) => (row.original.%s ? \"✓\" : \"—\") },", field.JSON)
	case adminKindDatetime:
		return base + fmt.Sprintf(", cell: ({ row }) => (row.original.%s ? new Date(row.original.%s).toLocaleString() : \"—\") },", field.JSON, field.JSON)
	case adminKindReferences:
		return base + fmt.Sprintf(", cell: ({ row }) => refLabel(refOptions[\"%s\"], row.original.%s) },", field.RefPlural, field.JSON)
	case adminKindAttachment:
		return base + fmt.Sprintf(", cell: ({ row }) => (row.original.%s ? <a href={row.original.%s} target=\"_blank\" rel=\"noreferrer\">file</a> : \"—\") },", field.JSON, field.JSON)
	default:
		return base + " },"
	}
}

func islandControl(field adminField) string {
	switch field.Kind {
	case adminKindText:
		return fmt.Sprintf(`          <Field label=%q>
            {(id: string) => (
              <Textarea id={id} rows={4} invalid={!!form.formState.errors.%s} {...form.register("%s")} />
            )}
          </Field>`, field.DisplayName, field.JSON, field.JSON)
	case adminKindBoolean:
		return fmt.Sprintf(`          <Field label=%q>
            <Checkbox value="true" {...form.register("%s")} />
          </Field>`, field.DisplayName, field.JSON)
	case adminKindDatetime:
		return fmt.Sprintf(`          <Field label=%q>
            {(id: string) => (
              <Input id={id} type="datetime-local" invalid={!!form.formState.errors.%s} {...form.register("%s")} />
            )}
          </Field>`, field.DisplayName, field.JSON, field.JSON)
	case adminKindEnum:
		var options strings.Builder
		for _, option := range field.Options {
			options.WriteString(fmt.Sprintf("                <option value=\"%s\">%s</option>\n", option, option))
		}
		return fmt.Sprintf(`          <Field label=%q>
            {(id: string) => (
              <Select id={id} invalid={!!form.formState.errors.%s} {...form.register("%s")}>
                <option value="">—</option>
%s              </Select>
            )}
          </Field>`, field.DisplayName, field.JSON, field.JSON, options.String())
	case adminKindReferences:
		return fmt.Sprintf(`          <Field label=%q>
            {(id: string) => (
              <Select id={id} invalid={!!form.formState.errors.%s} {...form.register("%s")}>
                <option value="">—</option>
                {(refOptions["%s"] ?? []).map((o) => (
                  <option key={o.id} value={String(o.id)}>{o.label}</option>
                ))}
              </Select>
            )}
          </Field>`, field.DisplayName, field.JSON, field.JSON, field.RefPlural)
	case adminKindAttachment:
		return fmt.Sprintf(`          <Field label=%q hint={uploads.%s || ""}>
            {(id: string) => (
              <span class="aw-admin-upload">
                <input id={id} type="file" class="aw-input" onChange={(e) => uploadFile(e, "%s")} />
                <input type="hidden" {...form.register("%s")} />
                {uploads.%s && <a href={uploads.%s} target="_blank" rel="noreferrer">view</a>}
              </span>
            )}
          </Field>`, field.DisplayName, field.JSON, field.JSON, field.JSON, field.JSON, field.JSON)
	case adminKindInteger, adminKindFloat:
		return fmt.Sprintf(`          <Field label=%q>
            {(id: string) => (
              <Input id={id} type="number" invalid={!!form.formState.errors.%s} {...form.register("%s")} />
            )}
          </Field>`, field.DisplayName, field.JSON, field.JSON)
	default:
		return fmt.Sprintf(`          <Field label=%q>
            {(id: string) => (
              <Input id={id} invalid={!!form.formState.errors.%s} {...form.register("%s")} />
            )}
          </Field>`, field.DisplayName, field.JSON, field.JSON)
	}
}

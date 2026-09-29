package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// migrationDSLTemplate is the generic shape produced by `generate
// migration`: one registered change whose down migration is derived by
// reversing the ops. Raw SQL is not reversible, so it is pointed at
// schema.Register with an explicit down migration instead.
const migrationDSLTemplate = `package migrations

import "github.com/daqing/airway/lib/migrate/schema"

func init() {
	schema.RegisterChange("{{.Version}}", "{{.Slug}}", func(m *schema.Migrator) {
		// The down migration is derived by reversing the change below.
		// Examples:
		//   m.AddColumn("posts", schema.Column{Name: "slug", Type: schema.Type{Kind: schema.TypeString}})
		//   m.AddIndex("posts", "slug").Unique()
		//   m.DropTable("legacy_posts")
		//
		// Raw SQL is not reversible; register it with
		// schema.Register(version, name, up, down) and an explicit down
		// migration, or make it reversible with m.Reversible(up, down).
	})
}
`

// migrationDSLCreateTemplate pre-fills a CreateTable skeleton for
// "create_<table>" names.
const migrationDSLCreateTemplate = `package migrations

import "github.com/daqing/airway/lib/migrate/schema"

func init() {
	schema.RegisterChange("{{.Version}}", "{{.Slug}}", func(m *schema.Migrator) {
		m.CreateTable("{{.Table}}", func(t *schema.Table) {
			t.ID()
			// t.String("title", 255).Null(false)
			// t.References("category")
			t.Timestamps()
		})
	})
}
`

type migrationTemplateData struct {
	Version string
	Slug    string
	Table   string
}

func generateMigrationFiles(args []string) error {
	if len(args) == 1 && isHelpArg(args[0]) {
		printCLIGenerateMigrationUsage(os.Stdout)
		return nil
	}

	if len(args) != 1 {
		return fmt.Errorf("usage: airway generate migration [name]")
	}

	name := strings.TrimSpace(args[0])
	if name == "" {
		return fmt.Errorf("migration name must not be empty")
	}

	if err := ensureDir(migrationDir); err != nil {
		return err
	}

	version := nextMigrationVersion(name)
	data := migrationTemplateData{
		Version: version,
		Slug:    name,
	}

	tpl := migrationDSLTemplate
	if table, ok := strings.CutPrefix(name, "create_"); ok && table != "" {
		data.Table = table
		tpl = migrationDSLCreateTemplate
	}

	path := filepath.Join(migrationDir, fmt.Sprintf("%s_%s.go", version, name))
	if err := writeTemplateFile(tpl, path, data); err != nil {
		return err
	}

	fmt.Printf("Created migration %s\n", path)
	noteMigrationImport(currentModulePath())
	return nil
}

// nextMigrationVersion returns a timestamp version for a migration named
// name, bumped forward one second at a time while the matching db/migrate
// file exists, so two runs within the same second never collide.
func nextMigrationVersion(name string) string {
	stamp := timeNow()
	version := stamp.Format("20060102150405")
	for adminPathExists("db", "migrate", version+"_"+name+".go") {
		stamp = stamp.Add(time.Second)
		version = stamp.Format("20060102150405")
	}
	return version
}

// noteMigrationImport makes sure the generated db/migrate package is compiled
// into the project binary: without the blank import its init registrations
// never run and `airway db:migrate` sees no migrations. Projects scaffolded
// by `airway new` already carry the import; anything else gets the import
// spliced into main.go when the layout is recognized, or a printed snippet.
func noteMigrationImport(module string) {
	quoted := "\"" + module + "/db/migrate\""
	if migrationPackageImported(module) {
		return
	}

	if raw, err := os.ReadFile("main.go"); err == nil {
		content := string(raw)
		anchor := "\"" + module + "/config\""
		if strings.Contains(content, anchor) && !strings.Contains(content, quoted) {
			updated := strings.Replace(content, anchor, anchor+"\n\n\t_ "+quoted, 1)
			if err := os.WriteFile("main.go", []byte(updated), 0o644); err == nil {
				fmt.Println("Updated main.go (registered db/migrate)")
				return
			}
		}
	}

	fmt.Println("\nNOTE: compile the migrations into your binary — add to main.go:")
	fmt.Printf("  _ %s\n", quoted)
}

// migrationPackageImported reports whether any root-level Go file
// blank-imports the project's db/migrate package.
func migrationPackageImported(module string) bool {
	needle := "\"" + module + "/db/migrate\""
	for _, name := range []string{"main.go", "plugins.go"} {
		raw, err := os.ReadFile(name)
		if err != nil {
			continue
		}
		if strings.Contains(string(raw), needle) {
			return true
		}
	}
	return false
}

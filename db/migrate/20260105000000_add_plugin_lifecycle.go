package migrate

import "github.com/daqing/airway/lib/migrate/schema"

func init() {
	schema.RegisterChange("20260105000000", "add_plugin_lifecycle", func(m *schema.Migrator) {
		m.AddColumn("plugins", schema.Column{Name: "error_message", Type: schema.Type{Kind: schema.TypeText}})
		m.AddColumn("plugins", schema.Column{Name: "enabled_at", Type: schema.Type{Kind: schema.TypeDateTime}})
		m.AddColumn("plugins", schema.Column{Name: "disabled_at", Type: schema.Type{Kind: schema.TypeDateTime}})
		m.CreateTable("plugin_migrations", func(t *schema.Table) {
			t.ID()
			t.String("plugin_name", 100).Null(false)
			t.String("migration", 150).Null(false)
			t.DateTime("applied_at").Null(false)
			t.UniqueIndex("plugin_name", "migration")
		})
	})
}

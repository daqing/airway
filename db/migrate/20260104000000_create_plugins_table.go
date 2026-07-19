package migrate

import "github.com/daqing/airway/lib/migrate/schema"

func init() {
	schema.RegisterChange("20260104000000", "create_plugins_table", func(m *schema.Migrator) {
		m.CreateTable("plugins", func(t *schema.Table) {
			t.ID()
			t.String("name", 100).Null(false).Unique()
			t.String("version", 50).Null(false)
			t.String("api_version", 50).Null(false)
			t.String("status", 20).Null(false).Default("registered")
			t.JSON("manifest_json").Null(false)
			t.DateTime("installed_at").Null(false)
			t.Timestamps()
		})
	})
}

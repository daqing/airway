package migrate

import "github.com/daqing/airway/lib/migrate/schema"

func init() {
	schema.RegisterChange("20260103000000", "create_resource_definition_tables", func(m *schema.Migrator) {
		m.CreateTable("resource_definitions", func(t *schema.Table) {
			t.ID()
			t.String("code", 100).Null(false).Unique()
			t.String("name", 150).Null(false)
			t.String("table_name", 100).Null(false).Unique()
			t.String("status", 20).Null(false).Default("draft")
			t.BigInt("active_version")
			t.JSON("draft_schema_json").Null(false)
			t.Timestamps()
		})
		m.CreateTable("resource_versions", func(t *schema.Table) {
			t.ID()
			t.References("resource_definition").Null(false).Index().ForeignKey().OnDelete("CASCADE")
			t.BigInt("version").Null(false)
			t.JSON("schema_json").Null(false)
			t.String("checksum", 64).Null(false)
			t.DateTime("published_at").Null(false)
			t.BigInt("published_by").Null(false)
			t.UniqueIndex("resource_definition_id", "version")
		})
	})
}

package migrate

import "github.com/daqing/airway/lib/migrate/schema"

func init() {
	schema.RegisterChange("20260102000000", "create_rbac_tables", func(m *schema.Migrator) {
		m.CreateTable("roles", func(t *schema.Table) {
			t.ID()
			t.String("code", 100).Null(false).Unique()
			t.String("name", 100).Null(false)
			t.Boolean("system").Null(false).Default(false)
			t.BigInt("version").Null(false).Default(1)
			t.Timestamps()
		})
		m.CreateTable("permissions", func(t *schema.Table) {
			t.ID()
			t.String("code", 150).Null(false).Unique()
			t.String("name", 150).Null(false)
			t.String("source", 100).Null(false).Default("custom")
			t.Timestamps()
		})
		m.CreateTable("admin_roles", func(t *schema.Table) {
			t.References("admin").Null(false).ForeignKey().OnDelete("CASCADE")
			t.References("role").Null(false).ForeignKey().OnDelete("CASCADE")
			t.UniqueIndex("admin_id", "role_id")
		})
		m.CreateTable("role_permissions", func(t *schema.Table) {
			t.References("role").Null(false).ForeignKey().OnDelete("CASCADE")
			t.References("permission").Null(false).ForeignKey().OnDelete("CASCADE")
			t.UniqueIndex("role_id", "permission_id")
		})
	})
}

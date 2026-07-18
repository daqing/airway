package migrate

import "github.com/daqing/airway/lib/migrate/schema"

// init registers the migration for administrator, session, audit log, and setup state tables.
func init() {
	schema.RegisterChange("20260101000000", "create_identity_tables", func(m *schema.Migrator) {
		m.CreateTable("admins", func(t *schema.Table) {
			t.ID()
			t.String("login", 100).Null(false).Unique()
			t.String("email", 255).Null(false).Unique()
			t.String("password_digest", 255).Null(false)
			t.String("status", 20).Null(false).Default("active")
			t.Boolean("super_admin").Null(false).Default(false)
			t.BigInt("auth_version").Null(false).Default(1)
			t.Timestamps()
		})

		m.CreateTable("sessions", func(t *schema.Table) {
			t.ID()
			t.References("admin").Null(false).Index().ForeignKey().OnDelete("CASCADE")
			t.String("token_digest", 64).Null(false).Unique()
			t.BigInt("auth_version").Null(false)
			t.DateTime("expires_at").Null(false)
			t.DateTime("last_seen_at").Null(false)
			t.DateTime("revoked_at")
			t.Timestamps()
		})

		m.CreateTable("audit_logs", func(t *schema.Table) {
			t.ID()
			t.BigInt("actor_id")
			t.String("action", 100).Null(false)
			t.String("target_type", 100).Null(false)
			t.String("target_id", 100)
			t.String("result", 20).Null(false)
			t.String("request_id", 100)
			t.String("ip_address", 64)
			t.Text("metadata_json")
			t.DateTime("created_at").Null(false).Default(schema.CurrentTimestamp)
			t.Index("actor_id")
			t.Index("action")
			t.Index("created_at")
		})

		m.CreateTable("setup_state", func(t *schema.Table) {
			t.String("key", 100).Null(false).Unique()
			t.DateTime("created_at").Null(false).Default(schema.CurrentTimestamp)
		})
	})
}

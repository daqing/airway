package migrate

import (
	"testing"

	"github.com/daqing/airway/lib/migrate/schema"
)

// DSL migrations in this package register themselves via init(). Their
// versions must be unique, otherwise the migrator cannot order them.
func TestRegisteredMigrationsHaveUniqueVersions(t *testing.T) {
	seen := map[string]string{}

	for _, def := range schema.Definitions() {
		if def.Version == "" {
			t.Fatalf("migration %q has an empty version", def.Name)
		}

		if name, dup := seen[def.Version]; dup {
			t.Fatalf("duplicate migration version %s used by %q and %q", def.Version, name, def.Name)
		}
		seen[def.Version] = def.Name
	}
}

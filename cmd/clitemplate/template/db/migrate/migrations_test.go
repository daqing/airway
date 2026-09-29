package migrations

import (
	"testing"

	"github.com/daqing/airway/lib/migrate/schema"
)

// Every migration file in this package registers itself in init() through
// schema.Register / schema.RegisterChange. An empty version, a duplicate
// version or a non-reversible RegisterChange panics at init time; running
// the package under `go test` surfaces that long before the server boots or
// a migration runs against a real database.
func TestRegisteredMigrationsAreWellFormed(t *testing.T) {
	names := map[string]string{}
	for _, def := range schema.Definitions() {
		if def.Version == "" {
			t.Fatalf("migration %q registered with an empty version", def.Name)
		}
		if prev, dup := names[def.Version]; dup {
			t.Fatalf("migrations %q and %q share version %s", prev, def.Name, def.Version)
		}
		names[def.Version] = def.Name
	}
}

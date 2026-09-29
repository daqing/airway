// Package migrations holds the project's Go DSL migrations: each migration
// lives in its own <timestamp>_<name>.go file and registers itself with
// lib/migrate/schema in init() (see `airway generate migration`). The down
// migration is derived by reversing the change, so a file is all it takes.
//
// The package is blank-imported from main.go so the registrations are
// compiled into the project binary; without that import `airway db:migrate`
// cannot see the migrations defined here.
package migrations

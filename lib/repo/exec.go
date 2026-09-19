package repo

import (
	"context"
	"database/sql"

	buildersql "github.com/daqing/airway/lib/sql"
)

// Querier is the minimal execution surface shared by *sql.DB and *sql.Tx.
type Querier interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
}

// Executor runs builder-compiled statements on a single connection or
// transaction while applying the dialect of the bound driver.
type Executor struct {
	q      Querier
	driver Driver
}

func (db *DB) executor() *Executor {
	return &Executor{q: db.conn, driver: db.driver}
}

func (ex *Executor) prepareBuilder(b buildersql.Stmt) (string, []any, error) {
	return ex.driver.prepareBuilder(b)
}

func (ex *Executor) prepareInsertBuilder(b buildersql.Stmt) (string, []any, error) {
	return ex.driver.prepareInsertBuilder(b)
}

func (ex *Executor) prepareQuery(query string, vals buildersql.NamedArgs) (string, []any, error) {
	return ex.driver.prepareQuery(query, vals)
}

package pg

import (
	"context"
	"time"

	"github.com/daqing/airway/lib/sql"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type DB struct {
	pool *pgxpool.Pool
}

func NewDB(dsn string) (*DB, error) {
	// Create configuration
	config, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		return nil, err
	}

	// Configure pool settings
	config.MaxConns = 20                      // Maximum number of connections
	config.MinConns = 5                       // Minimum number of connections
	config.MaxConnLifetime = time.Hour        // Maximum connection lifetime
	config.MaxConnIdleTime = time.Minute * 30 // Maximum idle time

	// Create connection pool
	pool, err := pgxpool.NewWithConfig(context.Background(), config)
	if err != nil {
		return nil, err
	}

	return &DB{pool: pool}, nil
}

func (db *DB) Count(b *sql.Builder) (n int64, err error) {
	sql, vals := b.ToSQL()

	db.pool.QueryRow(context.Background(), sql, pgx.NamedArgs(vals)).Scan(&n)

	return n, nil
}

func (db *DB) Delete(b *sql.Builder) error {
	sql, vals := b.ToSQL()
	_, err := db.pool.Exec(context.Background(), sql, pgx.NamedArgs(vals))

	return err
}

func (db *DB) Exists(b *sql.Builder) (bool, error) {
	n, err := db.Count(b)

	return n > 0, err
}

func (db *DB) Update(b *sql.Builder) error {
	sql, vals := b.ToSQL()

	_, err := db.pool.Exec(context.Background(), sql, pgx.NamedArgs(vals))

	return err
}

func (db *DB) Tx(fn func(tx pgx.Tx) error) error {
	ctx := context.Background()

	tx, err := db.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	err = fn(tx)
	if err != nil {
		return err
	}

	return tx.Commit(ctx)
}

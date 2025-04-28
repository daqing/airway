package orm

import (
	"context"
	"errors"
	"time"

	"github.com/daqing/airway/lib/utils"
	"github.com/jackc/pgx/v5/pgxpool"
)

type DB struct {
	pool *pgxpool.Pool
}

var __DB__ *DB

var ErrNotSetup = errors.New("database is not setup yet")

func Setup() error {
	// don't setup if no AIRWAY_PG_URL is set
	connString, err := utils.GetEnv("AIRWAY_PG_URL")
	if err != nil {
		return nil
	}

	// Create configuration
	config, err := pgxpool.ParseConfig(connString)
	if err != nil {
		return err
	}

	// Configure pool settings
	config.MaxConns = 20                      // Maximum number of connections
	config.MinConns = 5                       // Minimum number of connections
	config.MaxConnLifetime = time.Hour        // Maximum connection lifetime
	config.MaxConnIdleTime = time.Minute * 30 // Maximum idle time

	// Create connection pool
	pool, err := pgxpool.NewWithConfig(context.Background(), config)
	if err != nil {
		return err
	}

	__DB__ = &DB{
		pool: pool,
	}

	return nil
}

func Database() *DB {
	if __DB__ == nil {
		panic(ErrNotSetup)
	}

	return __DB__
}

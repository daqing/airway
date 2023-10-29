package repo

import (
	"os"

	"github.com/daqing/airway/lib/clients"
	"github.com/jackc/pgx/v5/pgxpool"
)

var Pool *pgxpool.Pool

func Setup() {
	pgUrl := os.Getenv("AIRWAY_PG_URL")
	if pgUrl == "" {
		panic("AIRWAY_PG_URL not set")
	}

	Pool = clients.NewPGPool(pgUrl)
}

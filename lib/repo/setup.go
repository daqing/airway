package repo

import (
	"log"
	"os"

	"github.com/daqing/airway/lib/clients"
	"github.com/jackc/pgx/v5/pgxpool"
)

var Pool *pgxpool.Pool

func Setup() {
	pgUrl := os.Getenv("AIRWAY_PG_URL")
	if pgUrl == "" {
		log.Fatalf("No PG_URL environment variable set")
	}

	Pool = clients.NewPGPool(pgUrl)
}

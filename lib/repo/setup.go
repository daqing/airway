package repo

import (
	"log"
	"os"

	"github.com/daqing/airway/lib/clients"
	"github.com/jackc/pgx/v5"
)

var Conn *pgx.Conn

func Setup() {
	pgUrl := os.Getenv("PG_URL")
	if pgUrl == "" {
		log.Fatalf("No PG_URL environment variable set")
	}

	Conn = clients.NewPG(pgUrl)
}

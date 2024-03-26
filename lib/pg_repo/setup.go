package pg_repo

import (
	"github.com/daqing/airway/lib/clients"
	"github.com/daqing/airway/lib/utils"
	"github.com/jackc/pgx/v5/pgxpool"
)

var Pool *pgxpool.Pool

func Setup() {
	Pool = clients.NewPGPool(utils.GetEnvMust("AIRWAY_PG_URL"))
}

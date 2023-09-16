package clients

import (
	"context"

	"github.com/jackc/pgx/v5"
)

// postgres://username:password@localhost:5432/database_name
func NewPG(url string) *pgx.Conn {
	conn, err := pgx.Connect(context.Background(), url)
	if err != nil {
		panic(err)
	}

	return conn
}

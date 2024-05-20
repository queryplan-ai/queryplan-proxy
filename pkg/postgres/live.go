package postgres

import (
	"context"
	"database/sql"

	"github.com/jackc/pgx/v5"
)

var (
	DB *sql.DB
)

func GetPostgresConnection(uri string) (*pgx.Conn, error) {
	conn, err := pgx.Connect(context.Background(), uri)
	if err != nil {
		panic("failed to connect to Postgres: " + err.Error())
	}

	return conn, nil
}

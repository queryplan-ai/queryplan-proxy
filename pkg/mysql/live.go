package mysql

import (
	"database/sql"
	"fmt"

	_ "github.com/go-sql-driver/mysql"
)

var (
	DB *sql.DB
)

func GetMysqlConnection(uri string) (*sql.DB, error) {
	var db *sql.DB
	var err error
	db, err = sql.Open("mysql", uri)
	if err != nil {
		return nil, fmt.Errorf("failed to open mysql connection: %v", err)
	}

	DB = db

	return db, nil
}

package mysql

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	daemontypes "github.com/queryplan-ai/queryplan-proxy/pkg/daemon/types"
	"github.com/queryplan-ai/queryplan-proxy/pkg/mysql/types"
)

var (
	Interval = 30 * time.Minute
)

func ProcessSchema(ctx context.Context, opts daemontypes.DaemonOpts) {
	for {
		if err := collectAndSendSchema(ctx, opts); err != nil {
			log.Printf("Error in schema collection: %v", err)
			time.Sleep(Interval)
			continue
		}

		time.Sleep(Interval)
	}
}

func collectAndSendSchema(ctx context.Context, opts daemontypes.DaemonOpts) error {
	tables, err := listTables(opts.LiveConnectionURI, opts.DatabaseName)
	if err != nil {
		return fmt.Errorf("list tables: %v", err)
	}

	primaryKeys, err := listPrimaryKeys(opts.LiveConnectionURI, opts.DatabaseName)
	if err != nil {
		return fmt.Errorf("list primary keys: %v", err)
	}

	for i, table := range tables {
		if _, ok := primaryKeys[table.TableName]; !ok {
			primaryKeys[table.TableName] = []string{}
		}

		tables[i].PrimaryKeys = primaryKeys[table.TableName]
	}

	payload := types.QueryPlanTablesPayload{
		Tables: tables,
	}

	marshaled, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal payload: %v", err)
	}

	req, err := http.NewRequest("PUT", fmt.Sprintf("%s/v1/schema", opts.APIURL), bytes.NewBuffer(marshaled))
	if err != nil {
		return fmt.Errorf("create request: %v", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", opts.Token))

	if opts.Environment != "" {
		req.Header.Set("X-QueryPlan-Environment", opts.Environment)
	}

	req.Header.Set("X-QueryPlan-DBMS", string(daemontypes.Mysql))
	req.Header.Set("X-QueryPlan-Database", opts.DatabaseName)

	client := http.DefaultClient
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("send request: %v", err)
	}

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	return nil
}

func listTables(uri string, dbName string) ([]types.MysqlTable, error) {
	// read the schema from mysql
	db, err := GetMysqlConnection(uri)
	if err != nil {
		return nil, fmt.Errorf("get mysql connection: %v", err)
	}

	rows, err := db.Query(`SELECT
c.TABLE_NAME, c.COLUMN_NAME, c.DATA_TYPE, c.COLUMN_TYPE, c.IS_NULLABLE, c.COLUMN_KEY, c.COLUMN_DEFAULT, c.EXTRA,
t.TABLE_ROWS
FROM INFORMATION_SCHEMA.COLUMNS c
INNER JOIN INFORMATION_SCHEMA.TABLES t ON t.TABLE_NAME = c.TABLE_NAME AND t.TABLE_SCHEMA = c.TABLE_SCHEMA
WHERE c.TABLE_SCHEMA = ?`, dbName)
	if err != nil {
		return nil, fmt.Errorf("query schema: %v", err)
	}

	defer rows.Close()

	tables := []types.MysqlTable{}
	for rows.Next() {
		column := types.MysqlColumn{}

		tableName := ""
		estimatedRowCount := int64(0)
		isNullable := ""
		columnDefault := sql.NullString{}
		if err := rows.Scan(&tableName, &column.ColumnName, &column.DataType, &column.ColumnType, &isNullable, &column.ColumnKey, &columnDefault, &column.Extra, &estimatedRowCount); err != nil {
			return nil, fmt.Errorf("scan: %v", err)
		}

		if isNullable == "YES" {
			column.IsNullable = true
		}

		if columnDefault.Valid {
			column.ColumnDefault = &columnDefault.String
		}

		found := false
		for i, table := range tables {
			if table.TableName == tableName {
				tables[i].Columns = append(table.Columns, column)
				found = true
				continue
			}
		}

		if !found {
			tables = append(tables, types.MysqlTable{
				TableName:         tableName,
				Columns:           []types.MysqlColumn{column},
				EstimatedRowCount: estimatedRowCount,
			})
		}
	}

	return tables, nil
}

func listPrimaryKeys(uri string, dbName string) (map[string][]string, error) {
	db, err := GetMysqlConnection(uri)
	if err != nil {
		return nil, fmt.Errorf("get mysql connection: %v", err)
	}

	rows, err := db.Query("SELECT TABLE_SCHEMA, TABLE_NAME, COLUMN_NAME FROM  INFORMATION_SCHEMA.KEY_COLUMN_USAGE  WHERE  CONSTRAINT_NAME = 'PRIMARY' AND TABLE_SCHEMA = ? ORDER BY TABLE_NAME, ORDINAL_POSITION", dbName)
	if err != nil {
		return nil, fmt.Errorf("query primary keys: %v", err)
	}

	defer rows.Close()

	primaryKeys := map[string][]string{}
	for rows.Next() {
		tableName := ""
		columnName := ""
		if err := rows.Scan(&tableName, &tableName, &columnName); err != nil {
			return nil, fmt.Errorf("scan: %v", err)
		}

		if _, ok := primaryKeys[tableName]; !ok {
			primaryKeys[tableName] = []string{}
		}

		primaryKeys[tableName] = append(primaryKeys[tableName], columnName)
	}

	return primaryKeys, nil
}

package postgres

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"regexp"
	"time"

	daemontypes "github.com/queryplan-ai/queryplan-proxy/pkg/daemon/types"
	"github.com/queryplan-ai/queryplan-proxy/pkg/postgres/types"
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

	url := fmt.Sprintf("%s/v1/schema", opts.APIURL)
	fmt.Printf("Sending schema to %s\n", url)
	req, err := http.NewRequest("PUT", url, bytes.NewBuffer(marshaled))
	if err != nil {
		return fmt.Errorf("create request: %v", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", opts.Token))

	if opts.Environment != "" {
		req.Header.Set("X-QueryPlan-Environment", opts.Environment)
	}

	req.Header.Set("X-QueryPlan-DBMS", string(daemontypes.Postgres))
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

func listTables(uri string, dbName string) ([]types.PostgresTable, error) {
	// read the schema from postgres
	db, err := GetPostgresConnection(uri)
	if err != nil {
		return nil, fmt.Errorf("get postgres connection: %v", err)
	}

	rows, err := db.Query(context.TODO(), `select table_name from information_schema.tables where table_catalog = $1 and table_schema = $2`, dbName, "public")
	if err != nil {
		return nil, fmt.Errorf("query schema: %v", err)
	}
	defer rows.Close()

	tableNames := []string{}
	for rows.Next() {
		tableName := ""
		if err := rows.Scan(&tableName); err != nil {
			return nil, fmt.Errorf("scan: %v", err)
		}

		tableNames = append(tableNames, tableName)
	}

	rows.Close()

	tables := []types.PostgresTable{}
	for _, tableName := range tableNames {
		rows, err = db.Query(context.TODO(), `select column_name, data_type, character_maximum_length, column_default, is_nullable from information_schema.columns where table_name = $1 and table_catalog = $2`, tableName, dbName)
		if err != nil {
			return nil, fmt.Errorf("query columns: %v", err)
		}

		defer rows.Close()
		for rows.Next() {
			column := types.PostgresColumn{}

			var maxLength sql.NullInt64
			var isNullable string
			var columnDefault sql.NullString

			if err := rows.Scan(&column.ColumnName, &column.DataType, &maxLength, &columnDefault, &isNullable); err != nil {
				return nil, err
			}

			if isNullable == "NO" {
				column.IsNullable = false
			} else {
				column.IsNullable = true
			}

			if columnDefault.Valid {
				value := stripOIDClass(columnDefault.String)
				column.ColumnDefault = &value
			}

			if maxLength.Valid {
				column.DataType = fmt.Sprintf("%s (%d)", column.DataType, maxLength.Int64)
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
				tables = append(tables, types.PostgresTable{
					TableName: tableName,
					Columns:   []types.PostgresColumn{column},
				})
			}
		}
	}

	return tables, nil
}

var oidClassRegexp = regexp.MustCompile(`'(.*)'::.+`)

func stripOIDClass(value string) string {
	matches := oidClassRegexp.FindStringSubmatch(value)
	if len(matches) == 2 {
		return matches[1]
	}
	return value
}

func listPrimaryKeys(uri string, dbName string) (map[string][]string, error) {
	db, err := GetPostgresConnection(uri)
	if err != nil {
		return nil, fmt.Errorf("get postgres connection: %v", err)
	}

	rows, err := db.Query(context.TODO(), `select table_name, column_name from information_schema.key_column_usage where constraint_name = 'PRIMARY' and table_catalog = $1`, dbName)
	if err != nil {
		return nil, fmt.Errorf("query primary keys: %v", err)
	}

	defer rows.Close()

	primaryKeys := map[string][]string{}
	for rows.Next() {
		tableName := ""
		columnName := ""
		if err := rows.Scan(&tableName, &columnName); err != nil {
			return nil, fmt.Errorf("scan: %v", err)
		}

		if _, ok := primaryKeys[tableName]; !ok {
			primaryKeys[tableName] = []string{}
		}

		primaryKeys[tableName] = append(primaryKeys[tableName], columnName)
	}

	return primaryKeys, nil
}

package types

type Table struct {
	TableName         string   `json:"table_name"`
	Columns           []Column `json:"columns"`
	PrimaryKeys       []string `json:"primary_keys"`
	EstimatedRowCount int64    `json:"estimated_row_count"`
}

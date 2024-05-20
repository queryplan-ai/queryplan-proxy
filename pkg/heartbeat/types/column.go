package types

type Column struct {
	ColumnName    string  `json:"column_name"`
	DataType      string  `json:"data_type"`
	ColumnType    string  `json:"column_type"`
	IsNullable    bool    `json:"is_nullable"`
	ColumnKey     string  `json:"column_key"`
	ColumnDefault *string `json:"column_default,omitempty"`
	Extra         string  `json:"extra"`
}

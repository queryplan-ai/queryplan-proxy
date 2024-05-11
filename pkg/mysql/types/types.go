package types

import (
	"github.com/pubnative/mysqlproto-go"
)

type COM_Query_ResponsePacket struct {
	ERR_Packet          *mysqlproto.ERRPacket
	OK_Packet           *mysqlproto.OKPacket
	TextResultsetPacket *COM_Query_TextResultsetPacket
}

type COM_Query_TextResultsetPacket struct {
}

type MysqlTable struct {
	TableName         string        `json:"table_name"`
	Columns           []MysqlColumn `json:"columns"`
	PrimaryKeys       []string      `json:"primary_keys"`
	EstimatedRowCount int64         `json:"estimated_row_count"`
}

type MysqlColumn struct {
	ColumnName    string  `json:"column_name"`
	DataType      string  `json:"data_type"`
	ColumnType    string  `json:"column_type"`
	IsNullable    bool    `json:"is_nullable"`
	ColumnKey     string  `json:"column_key"`
	ColumnDefault *string `json:"column_default,omitempty"`
	Extra         string  `json:"extra"`
}

type QueryPlanTablesPayload struct {
	Tables []MysqlTable `json:"tables"`
}

type QueryPlanTablesResponse struct {
	Token string `json:"token"`
}

type QueryPlanQuery struct {
	ExecutedAt int64  `json:"executed_at"`
	Duration   int64  `json:"duration"`
	Query      string `json:"query"`

	// Callstack?
}

type QueryPlanQueriesPayload struct {
	Queries []QueryPlanQuery `json:"queries"`
	// Transactions []QueryPlanTransaction `json:"transactions"`
}

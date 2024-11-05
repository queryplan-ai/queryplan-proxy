package types

import (
	"github.com/pubnative/mysqlproto-go"
	"github.com/tuvistavie/securerandom"
)

type COM_Query_ResponsePacket struct {
	ERR_Packet          *mysqlproto.ERRPacket
	OK_Packet           *mysqlproto.OKPacket
	TextResultsetPacket *COM_Query_TextResultsetPacket
}

type COM_Query_TextResultsetPacket struct {
	ColumnCount int
}

type PreparedStatement struct {
	ID               int
	Query            string
	IsExecuted       bool
	CountEOFReceived int
}

type ConnectionState struct {
	ID                string
	RowCount          int64
	PreparedStatement *PreparedStatement
}

func NewConnectionState() (*ConnectionState, error) {
	connectionID, err := securerandom.Hex(4)
	if err != nil {
		return nil, err
	}

	return &ConnectionState{
		ID:                connectionID,
		RowCount:          0,
		PreparedStatement: nil,
	}, nil
}

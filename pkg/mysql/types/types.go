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
	ColumnCount int
}

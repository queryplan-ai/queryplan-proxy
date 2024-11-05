package mysql

import (
	"bytes"
	"io"
	"log"
	"net"

	"github.com/queryplan-ai/queryplan-proxy/pkg/heartbeat"
	"github.com/queryplan-ai/queryplan-proxy/pkg/mysql/types"
)

type MysqlPacketType byte

const (
	MysqlPacketTypeUnknown           = 0xFF
	MysqlPacketTypeOKPacket          = 0x00
	MysqlPacketTypeEOFPacket         = 0xFE
	MysqlPacketTypeHandshake         = 0x0A
	MysqlPacketTypeHandshakeResponse = 0x01
	MysqlPacketTypeColumnDefinition  = 0x03
	MysqlPacketTypeComFieldList      = 0x04
)

// copyAndInspectResponse copies data from src to dst and parses the MySQL response
// mysql keeps the connection alive though, so the scope of this function
// is likely > 1 query
func copyAndInspectResponses(src, dst net.Conn, connectionState *types.ConnectionState, inspect bool) error {
	var accum bytes.Buffer
	buf := make([]byte, 8192)

	for {
		n, err := src.Read(buf)
		if err != nil {
			if err != io.EOF {
				log.Printf("Error reading from upstream: %v", err)
			}
			return err
		}

		accum.Write(buf[:n])

		// Continue to read every possible complete packet from accum
		for {
			data := accum.Bytes()
			count, ok := parseNextPacket(data)
			if !ok {
				break
			}

			dataToForward := data[:count]

			_, err = dst.Write(dataToForward)
			if err != nil {
				log.Printf("Error writing to client: %v", err)
				return err
			}

			// process this packet
			if err := parseFullResponsePacket(dataToForward, connectionState); err != nil {
				return err
			}

			// Remove the processed packet from the buffer
			accum.Next(count)
		}
	}
}

func parseNextPacket(data []byte) (length int, ok bool) {
	// Check if we have enough data to determine the length
	if len(data) < 4 {
		return 0, false
	}
	length = int(data[0]) | int(data[1])<<8 | int(data[2])<<16

	// Check if we have a full packet
	if len(data) >= length+4 {
		return length + 4, true
	}

	return 0, false
}

func parseFullResponsePacket(data []byte, connectionState *types.ConnectionState) error {
	payloadStartIndex := 4
	packetType := data[payloadStartIndex]

	switch packetType {
	case MysqlPacketTypeOKPacket:
		if len(data) >= 9 {
			if connectionState.PreparedStatement != nil {
				if !connectionState.PreparedStatement.IsExecuted {
					stmtID := uint32(data[5]) | uint32(data[6])<<8 | uint32(data[7])<<16 | uint32(data[8])<<24
					connectionState.PreparedStatement.ID = int(stmtID)
				} else if connectionState.PreparedStatement.IsExecuted {
					connectionState.RowCount++
					return nil
				}
			}
		}

	case MysqlPacketTypeEOFPacket:
		connectionState.EOFCount++

		if connectionState.PreparedStatement != nil {
			if connectionState.PreparedStatement.IsExecuted {
				connectionState.PreparedStatement.CountEOFReceived++
				if connectionState.PreparedStatement.CountEOFReceived == 2 {
					heartbeat.CompleteCurrentQuery(connectionState.CurrentQuery, connectionState.RowCount)
				}
			}
		}

		if connectionState.ReceivedSimpleQuery && connectionState.EOFCount == 3 {
			heartbeat.CompleteCurrentQuery(connectionState.CurrentQuery, connectionState.RowCount)
		}
		connectionState.RowCount = 0

	case MysqlPacketTypeColumnDefinition, MysqlPacketTypeHandshake, MysqlPacketTypeHandshakeResponse, MysqlPacketTypeComFieldList:
		break

	default:
		if connectionState.ReceivedSimpleQuery {
			connectionState.RowCount++
		}

	}

	return nil
}

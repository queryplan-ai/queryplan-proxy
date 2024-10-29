package mysql

import (
	"bytes"
	"io"
	"log"
	"net"

	"github.com/queryplan-ai/queryplan-proxy/pkg/heartbeat"
)

type MysqlPacketType byte

const (
	MysqlPacketTypeOKPacket          = 0x00
	MysqlPacketTypeEOFPacket         = 0xFE
	MysqlPacketTypeHandshake         = 0x0A
	MysqlPacketTypeHandshakeResponse = 0x01
	MysqlPacketTypeColumnDefinition  = 0x03
	MysqlPacketTypeComFieldList      = 0x04
	// MysqlPacketTypeAuthSwitchRequest  = 0xFE
	// MysqlPacketTypeAuthSwitchResponse = 0x01
)

type MysqlResponsePhase int

const (
	MysqlResponsePhaseColumnDefinition MysqlResponsePhase = 1
	MysqlResponsePhaseRowData          MysqlResponsePhase = 2
)

var ignoreQuery bool
var isResultSet bool
var rowCount = int64(0)
var phase = MysqlResponsePhaseColumnDefinition

// copyAndInspectResponse copies data from src to dst and parses the MySQL response
// mysql keeps the connection alive though, so the scope of this function
// is likely > 1 query
func copyAndInspectResponse(src, dst net.Conn, inspect bool) error {
	var accum bytes.Buffer
	buf := make([]byte, 8192)

	// default to ignoring the next query
	ignoreQuery = true

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
			if err := parseFullResponsePacket(dataToForward); err != nil {
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

func parseFullResponsePacket(data []byte) error {
	if ignoreQuery {
		return nil
	}

	payloadStartIndex := 4
	packetType := data[payloadStartIndex]

	switch packetType {
	case MysqlPacketTypeOKPacket:
		ignoreQuery = false
		if len(data) >= 9 {
			// This could be a prepared statement response
			stmtID := uint32(data[5]) | uint32(data[6])<<8 | uint32(data[7])<<16 | uint32(data[8])<<24

			// hmm, for now, we find the prepared statement with an id of -1 and assign to this
			for _, ps := range preparedStatements.GetAll() {
				if ps.ID == -1 {
					ps.ID = int(stmtID)
					break
				}
			}
		}
	case MysqlPacketTypeColumnDefinition:
		break

	case MysqlPacketTypeEOFPacket:
		switch phase {
		case MysqlResponsePhaseColumnDefinition:
			phase = MysqlResponsePhaseRowData
			isResultSet = true
			rowCount = 0
		case MysqlResponsePhaseRowData:
			heartbeat.CompleteCurrentQuery(rowCount)
			isResultSet = false
			rowCount = 0
		}

	case MysqlPacketTypeHandshake, MysqlPacketTypeHandshakeResponse:
		break

	case MysqlPacketTypeComFieldList:
		break

	default:
		rowCount++
	}
	return nil
}

func resetState() {
	isResultSet = false
	rowCount = 0
	phase = MysqlResponsePhaseColumnDefinition
}

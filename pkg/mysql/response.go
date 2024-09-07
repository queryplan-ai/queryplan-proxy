package mysql

import (
	"bytes"
	"fmt"
	"io"
	"net"

	"github.com/queryplan-ai/queryplan-proxy/pkg/mysql/types"
)

func copyAndInspectResponse(src, dst net.Conn, inspect bool) error {
	var buffer bytes.Buffer
	for {
		data := make([]byte, 4096)
		n, err := src.Read(data)
		if err != nil {
			if err != io.EOF {
				return err
			}
			break
		}

		if _, err := dst.Write(data[:n]); err != nil {
			return err
		}

		if inspect {
			buffer.Write(data[:n])

			packetsReceived := 0
			resultSetPacket := types.COM_Query_TextResultsetPacket{}

			for {
				full, length := isFullPacket(&buffer)
				if !full {
					break
				}
				packetData := buffer.Next(length + 4)
				err := parseFullResponsePacket(packetData, packetsReceived, &resultSetPacket)
				if err != nil {
					return err
				}

				packetsReceived++
			}

			// log.Printf("Packet: %#v", resultSetPacket)
		}
	}
	return nil
}

func isFullPacket(buffer *bytes.Buffer) (bool, int) {
	if buffer.Len() < 4 {
		return false, 0 // Not enough data to determine the length
	}

	data := buffer.Bytes()
	length := int(data[0]) | int(data[1])<<8 | int(data[2])<<16

	if buffer.Len() >= length+4 {
		return true, length // Full packet is available, return its length
	}
	return false, 0 // Full packet is not available
}

func parseFullResponsePacket(packetData []byte, packetsReceived int, resultSetPacket *types.COM_Query_TextResultsetPacket) error {
	if len(packetData) < 5 { // 4 bytes header + at least 1 byte payload
		return fmt.Errorf("packet is too short to determine type")
	}

	// The payload starts at index 4
	payloadStartIndex := 4
	packetType := packetData[payloadStartIndex]

	switch packetType {
	case 0x00:
		if packetsReceived == 0 && len(packetData) >= 9 { // 4 byte header + 5 bytes minimum for stmt ID
			// This could be a prepared statement response
			stmtID := uint32(packetData[5]) | uint32(packetData[6])<<8 | uint32(packetData[7])<<16 | uint32(packetData[8])<<24

			// hmm, for now, we find the prepared statement with an id of -1 and assign to this
			for _, ps := range preparedStatements.GetAll() {
				if ps.ID == -1 {
					ps.ID = int(stmtID)
					break
				}
			}
		}
	case 0xFB:
		fmt.Println("LOCAL INFILE packet received")
	case 0xFF:
		fmt.Println("Error packet received")
	default:
		// Likely the start of a result set
		if packetsReceived == 0 {
			colCount, _, _ := lenDecInt(packetData[payloadStartIndex:])
			resultSetPacket.ColumnCount = int(colCount)
			// fmt.Printf("Result set with %d columns\n", colCount)
		}
	}

	return nil
}

func lenDecInt(b []byte) (uint64, uint64, bool) { // int, offset, is null
	if len(b) == 0 { // MySQL may return 0 bytes for NULL value
		return 0, 0, true
	}

	switch b[0] {
	case 0xfb:
		return 0, 1, true
	case 0xfc:
		return uint64(b[1]) | uint64(b[2])<<8, 3, false
	case 0xfd:
		return uint64(b[1]) | uint64(b[2])<<8 | uint64(b[3])<<16, 4, false
	case 0xfe:
		return uint64(b[1]) | uint64(b[2])<<8 | uint64(b[3])<<16 |
			uint64(b[4])<<24 | uint64(b[5])<<32 | uint64(b[6])<<40 |
			uint64(b[7])<<48 | uint64(b[8])<<56, 9, false
	default:
		return uint64(b[0]), 1, false
	}
}

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
				return err // Handle read error
			}
			break // EOF reached, stop reading
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
	if len(packetData) < 1 {
		return fmt.Errorf("packet is too short to determine type")
	}

	if packetsReceived == 0 {
		if packetData[0] == 0x00 {
			// This is the first response packet, indicating success and containing the statement ID.
			if len(packetData) >= 5 {
				// Extract the statement ID (bytes 1-4, little-endian).
				stmtID := uint32(packetData[1]) | uint32(packetData[2])<<8 | uint32(packetData[3])<<16 | uint32(packetData[4])<<24
				fmt.Printf("Prepared statement ID: %d\n", stmtID)

				// Here you would store the statement ID in your prepared statement tracking map.
				// preparedStatements[stmtID] = <Your stored prepared query>
			} else {
				return fmt.Errorf("COM_STMT_PREPARE response too short to contain statement ID")
			}
		}
	}

	switch packetsReceived {
	case 0:
		colCount, _, _ := lenDecInt(packetData)
		resultSetPacket.ColumnCount = int(colCount)
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

package mysql

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"net"

	"github.com/queryplan-ai/queryplan-proxy/pkg/heartbeat"
)

func copyAndInspectResponse(src, dst net.Conn, inspect bool) error {
	var accum bytes.Buffer
	buf := make([]byte, 8192)

	var isResultSet bool
	var rowCount int64

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
			if err := parseFullResponsePacket(dataToForward, &isResultSet, &rowCount); err != nil {
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

func parseFullResponsePacket(data []byte, isResultSet *bool, rowCount *int64) error {
	payloadStartIndex := 4
	packetType := data[payloadStartIndex]

	switch packetType {
	case 0x00:
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

	case 0xFE: // EOF Packet
		fmt.Printf("EOF Packet\n")
		if *isResultSet {
			// End of rows in result set
			fmt.Printf("EOF packet received, ending result set with %d rows\n", *rowCount)
			heartbeat.CompleteCurrentQuery(*rowCount)
			*isResultSet = false
		} else {
			// End of column definitions, starting row counting
			fmt.Println("EOF packet received, preparing for row counting")
			*isResultSet = true
			*rowCount = 0
		}

	case 0xFF:
		fmt.Println("Error packet received")

	default:
		// Possible data row if in a result set
		if *isResultSet {
			*rowCount++
			// fmt.Printf("Data Row Packet - Row %d\n", *rowCount)
			// fmt.Printf("data: %s\n", string(data[payloadStartIndex:]))

		} else {
			colCount, _, _ := lenDecInt(data[payloadStartIndex:])
			fmt.Printf("Result set with %d columns\n", colCount)
			fmt.Printf("ignoring data: %s\n", string(data[payloadStartIndex:]))
			*isResultSet = true
			*rowCount = 1 // because something is wrong, we are getting the 1st row in this packet
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

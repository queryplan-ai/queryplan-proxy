package mysql

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"net"
	"strings"
	"time"

	"github.com/DataDog/datadog-agent/pkg/obfuscate"
	"github.com/queryplan-ai/queryplan-proxy/pkg/mysql/types"
)

var (
	ErrNonQueryData                   = fmt.Errorf("non-query data")
	ErrNonQueryDataOrIncompletePacket = fmt.Errorf("non-query data or incomplete packet")
)

const (
	COM_QUERY = 0x03
)

func copyAndInspectCommand(src, dst net.Conn, inspect bool) error {
	buffer := make([]byte, 4096)
	for {
		n, err := src.Read(buffer)
		if err != nil {
			if err != io.EOF {
				return err
			}
			break
		}

		data := buffer[:n]
		if inspect {
			query, err := extractQuery(data)
			if err == nil {
				cleanedQuery, err := cleanQuery(query)
				if err != nil {
					log.Printf("Error cleaning query: %v", err)
				} else {
					qpq := types.QueryPlanQuery{
						Query:      cleanedQuery,
						ExecutedAt: time.Now().UnixNano(),
					}
					pendingQueries.Add(qpq)
				}
			}
		}

		if _, err := dst.Write(data); err != nil {
			return err
		}
	}
	return nil
}

func cleanQuery(query string) (string, error) {
	cleanedQuery, err := obfuscate.NewObfuscator(obfuscate.Config{
		SQL: obfuscate.SQLConfig{
			KeepSQLAlias: true,
		},
	}).ObfuscateSQLString(query)
	if err != nil {
		return "", err
	}

	return cleanedQuery.Query, nil
}

// extractQuery returns the query and an id we can use to map it later
// the id is deterministic
func extractQuery(data []byte) (string, error) {
	if len(data) < 5 {
		return "", ErrNonQueryDataOrIncompletePacket
	}
	if data[4] == COM_QUERY {
		fmt.Printf("data: %v\n", data)
		return strings.TrimSpace(string(data[5:])), nil
	}
	return "", ErrNonQueryData
}

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

		// Write to destination connection
		if _, err := dst.Write(data[:n]); err != nil {
			return err // Handle write error
		}

		// Inspect the data if required
		if inspect {
			buffer.Write(data[:n]) // Accumulate data in the buffer

			// Process all complete packets in the buffer
			for {
				full, length := isFullPacket(&buffer)
				if !full {
					break // No more full packets in the buffer
				}
				// Extract the full packet data using the correct length
				packetData := buffer.Next(length + 4)
				packet, err := parseFullResponsePacket(packetData)
				if err != nil {
					return err // Handle parse error
				}

				// Log the parsed packet
				log.Printf("Packet: %v", packet)
			}
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

func parseFullResponsePacket(packetData []byte) (interface{}, error) {
	// Check the first byte to determine the type of packet
	if len(packetData) < 1 {
		return nil, fmt.Errorf("packet is too short to determine type")
	}

	switch packetData[0] {
	case 0x00:
		return parseOKPacket(packetData)
	case 0xFF:
		return parseErrorPacket(packetData)
	case 0xFE:
		if len(packetData) < 9 {
			return parseEOFPacket(packetData)
		}
		// Fall through to default if not an EOF packet
	default:
		return parseResultSetPacket(packetData)
	}

	return nil, fmt.Errorf("unknown packet type")
}

func parseOKPacket(data []byte) (interface{}, error) {
	fmt.Printf("OK packet: %v\n", data)
	return nil, nil
}

func parseErrorPacket(data []byte) (interface{}, error) {
	fmt.Printf("Error packet: %v\n", data)
	return nil, nil
}

func parseEOFPacket(data []byte) (interface{}, error) {
	fmt.Printf("EOF packet: %v\n", data)
	return nil, nil
}

func parseResultSetPacket(packetData []byte) (interface{}, error) {
	fmt.Printf("ResultSet packet: %v\n", packetData)
	return nil, nil
}

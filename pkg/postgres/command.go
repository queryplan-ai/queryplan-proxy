package postgres

import (
	"fmt"
	"io"
	"log"
	"net"
	"time"

	"github.com/pkg/errors"
	heartbeattypes "github.com/queryplan-ai/queryplan-proxy/pkg/heartbeat/types"
	"github.com/queryplan-ai/queryplan-proxy/pkg/postgres/types"
)

var (
	ErrNonQueryData = fmt.Errorf("non-query data")
)

func copyAndInspectCommand(src net.Conn, dst net.Conn, connectionState *types.ConnectionState, inspect bool) error {
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
			query, isPreparedStatement, err := extractQuery(data)
			if err == nil {
				cleanedQuery, err := cleanQuery(query)
				if err != nil {
					log.Printf("Error cleaning query: %v", err)
				} else {
					connectionState.CurrentQuery = &heartbeattypes.CurrentQuery{
						ExecutionStartedAt:  time.Now().UnixNano(),
						Query:               cleanedQuery,
						IsPreparedStatement: isPreparedStatement,
					}
				}
			} else {
				if errors.Cause(err) != ErrNonQueryData {
					log.Printf("Error extracting query: %v", err)
				}
			}
		}

		if _, err = dst.Write(data); err != nil {
			return err
		}
	}

	return nil
}

func extractQuery(data []byte) (string, bool, error) {
	// Check if we have at least a message type (1 byte) and length (4 bytes)
	if len(data) < 5 {
		return "", false, fmt.Errorf("data too short to be a valid message")
	}

	// The first byte is the message type
	messageType := data[0]

	// The next 4 bytes are the length of the message (big-endian)
	messageLength := int(data[1])<<24 | int(data[2])<<16 | int(data[3])<<8 | int(data[4])

	// Check if we have enough data for the full message
	if len(data) < messageLength {
		return "", false, fmt.Errorf("incomplete message")
	}

	// Handle specific message types
	switch messageType {
	case 'Q': // Simple query
		query := string(data[5:messageLength])
		return query, false, nil

	case 'P': // Parse (for prepared statements)
		// Prepared statement message format:
		// 'P' (1 byte), length (4 bytes), name (null-terminated), query (null-terminated), etc.
		// For simplicity, we skip extracting the name here
		query := string(data[5:messageLength])
		fmt.Printf("Prepared statement: %s\n", query)
		return query, true, nil

	default:
		// Non-query message type
		return "", false, ErrNonQueryData
	}
}

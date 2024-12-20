package mysql

import (
	"encoding/binary"
	"fmt"
	"io"
	"log"
	"net"
	"strings"
	"time"

	"github.com/pkg/errors"
	heartbeattypes "github.com/queryplan-ai/queryplan-proxy/pkg/heartbeat/types"
	"github.com/queryplan-ai/queryplan-proxy/pkg/mysql/types"
)

var (
	ErrNonQueryData                   = fmt.Errorf("non-query data")
	ErrNonQueryDataOrIncompletePacket = fmt.Errorf("non-query data or incomplete packet")
	ErrUnknownPreparedStatement       = fmt.Errorf("unknown prepared statement")
)

const (
	COM_QUIT             = 0x01
	COM_INIT_DB          = 0x02
	COM_QUERY            = 0x03
	COM_FIELD_LIST       = 0x04
	COM_CREATE_DB        = 0x05
	COM_DROP_DB          = 0x06
	COM_REFRESH          = 0x07
	COM_STATISTICS       = 0x09
	COM_PROCESS_INFO     = 0x0a
	COM_CONNECT          = 0x0b
	COM_PROCESS_KILL     = 0x0c
	COM_DEBUG            = 0x0d
	COM_PING             = 0x0e
	COM_CHANGE_USER      = 0x11
	COM_RESET_CONNECTION = 0x1f
	COM_STMT_PREPARE     = 0x16
	COM_STMT_EXECUTE     = 0x17
	COM_STMT_CLOSE       = 0x19
)

func copyAndInspectCommands(src, dst net.Conn, connectionState *types.ConnectionState) error {
	buffer := make([]byte, 0, 8192)
	tempBuffer := make([]byte, 4096)

	for {
		n, err := src.Read(tempBuffer)
		if err != nil {
			if err != io.EOF {
				return err
			}
			break
		}

		buffer = append(buffer, tempBuffer[:n]...)

		for {
			query, isPreparedStatement, bytesRead, err := extractQuery(buffer, connectionState)
			if err != nil {
				if errors.Cause(err) == ErrNonQueryDataOrIncompletePacket {
					break
				} else if errors.Cause(err) != ErrNonQueryData {
					log.Printf("Error extracting query: %v", err)
				}
				buffer = buffer[bytesRead:]
				continue
			}

			buffer = buffer[bytesRead:]

			if err == nil {
				cleanedQuery, err := cleanQuery(query)
				if err != nil {
					log.Printf("Error cleaning query: %v", err)
				} else {
					if strings.ToLower(cleanedQuery) == "select connection_id ( ) as pid" {
						// Ignore this query
					}

					connectionState.CurrentQuery = &heartbeattypes.CurrentQuery{
						ExecutionStartedAt:  time.Now().UnixNano(),
						Query:               cleanedQuery,
						IsPreparedStatement: isPreparedStatement,
					}
				}
			}
		}

		if _, err := dst.Write(tempBuffer[:n]); err != nil {
			return err
		}
	}

	return nil
}

// extractQuery returns the query and an id we can use to map it later
// the id is deterministic
// the bool indicates if the query is a prepared statement
func extractQuery(data []byte, connectionState *types.ConnectionState) (string, bool, int, error) {
	if len(data) < 4 {
		return "", false, 0, ErrNonQueryDataOrIncompletePacket // Not enough data for the header
	}

	payloadLength := int(data[0]) | int(data[1])<<8 | int(data[2])<<16
	totalPacketLength := 4 + payloadLength

	if len(data) < totalPacketLength {
		return "", false, 0, ErrNonQueryDataOrIncompletePacket
	}

	switch data[4] {
	case COM_QUERY:
		connectionState.ReceivedSimpleQuery = true
		connectionState.RowCount = 0
		query := strings.TrimSpace(string(data[5:]))
		return query, false, totalPacketLength, nil
	case COM_STMT_PREPARE:
		query := strings.TrimSpace(string(data[5:]))
		connectionState.PreparedStatement = &types.PreparedStatement{
			Query: query,
			ID:    -1,
		}
		connectionState.RowCount = 0
		return "", false, totalPacketLength, ErrNonQueryData // really this should be after we find the statement in COM_STMT_EXECUTE
	case COM_STMT_EXECUTE:
		// find the prepared statement with the same id, log that this query was executed
		if len(data) < 9 {
			return "", false, totalPacketLength, ErrNonQueryDataOrIncompletePacket
		}
		stmtID := binary.LittleEndian.Uint32(data[5:9])

		if connectionState.PreparedStatement == nil {
			return "", false, totalPacketLength, ErrUnknownPreparedStatement
		}
		if connectionState.PreparedStatement != nil && connectionState.PreparedStatement.ID == int(stmtID) {
			connectionState.PreparedStatement.IsExecuted = true
			return connectionState.PreparedStatement.Query, true, totalPacketLength, nil
		}

		return "", false, totalPacketLength, ErrUnknownPreparedStatement

	case COM_STMT_CLOSE:
		connectionState.PreparedStatement = nil
		return "", false, totalPacketLength, ErrNonQueryData

	case COM_QUIT, COM_INIT_DB, COM_FIELD_LIST, COM_CREATE_DB,
		COM_DROP_DB, COM_REFRESH, COM_STATISTICS, COM_PROCESS_INFO,
		COM_CONNECT, COM_PROCESS_KILL, COM_DEBUG, COM_PING,
		COM_CHANGE_USER, COM_RESET_CONNECTION:
		return "", false, totalPacketLength, ErrNonQueryData
	}

	return "", false, totalPacketLength, ErrNonQueryData
}

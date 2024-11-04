package mysql

import (
	"encoding/binary"
	"fmt"
	"io"
	"log"
	"net"
	"strings"

	"github.com/pkg/errors"
	"github.com/queryplan-ai/queryplan-proxy/pkg/heartbeat"
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
			query, isPreparedStatement, err := extractQuery(data)
			if err == nil {
				cleanedQuery, err := cleanQuery(query)
				if err != nil {
					log.Printf("Error cleaning query: %v", err)
				} else {
					if strings.ToLower(cleanedQuery) == "select connection_id ( ) as pid" {
						ignoreQuery = true
					} else {
						ignoreQuery = false
						resetState()
					}
					heartbeat.SetCurrentQuery(cleanedQuery, isPreparedStatement)
				}
			} else {
				if errors.Cause(err) != ErrNonQueryData {
					log.Printf("Error extracting query: %v", err)
				}
			}
		}

		if _, err := dst.Write(data); err != nil {
			return err
		}
	}

	return nil
}

// extractQuery returns the query and an id we can use to map it later
// the id is deterministic
// the bool indicates if the query is a prepared statement
func extractQuery(data []byte) (string, bool, error) {
	if len(data) < 5 {
		return "", false, ErrNonQueryDataOrIncompletePacket
	}

	switch data[4] {
	case COM_QUERY:
		return strings.TrimSpace(string(data[5:])), false, nil
	case COM_STMT_PREPARE:
		query := strings.TrimSpace(string(data[5:]))
		preparedStatement = &PreparedStatement{
			ID:          -1,
			Query:       query,
			PrepareSent: true,
			ExecuteSent: false,
		}
		return "", false, ErrNonQueryData // really this should be after we find the statement in COM_STMT_EXECUTE
	case COM_STMT_EXECUTE:
		// find the prepared statement with the same id, log that this query was executed
		if len(data) < 9 {
			return "", false, ErrNonQueryDataOrIncompletePacket
		}
		stmtID := binary.LittleEndian.Uint32(data[5:9])
		if preparedStatement != nil && preparedStatement.ID == int(stmtID) {
			preparedStatement.ExecuteSent = true
			return preparedStatement.Query, true, nil
		}

		fmt.Printf("Unknown prepared statement ID: %v\n", stmtID)
		return "", false, ErrUnknownPreparedStatement

	case COM_QUIT, COM_INIT_DB, COM_FIELD_LIST, COM_CREATE_DB,
		COM_DROP_DB, COM_REFRESH, COM_STATISTICS, COM_PROCESS_INFO,
		COM_CONNECT, COM_PROCESS_KILL, COM_DEBUG, COM_PING,
		COM_CHANGE_USER, COM_RESET_CONNECTION, COM_STMT_CLOSE:
		return "", false, ErrNonQueryData
	}

	// fmt.Printf("Unknown command: %v\n", data[4])
	return "", false, ErrNonQueryData
}

package mysql

import (
	"fmt"
	"io"
	"log"
	"net"
	"strings"

	"github.com/queryplan-ai/queryplan-proxy/pkg/heartbeat"
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
					heartbeat.AddPendingQuery(cleanedQuery)
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

package postgres

import (
	"fmt"
	"net"
)

var (
	ErrNonQueryData = fmt.Errorf("non-query data")
)

func copyAndInspectCommand(src, dst net.Conn, inspect bool) error {

	return nil
}

func extractQuery(data []byte) (string, bool, error) {

	return "", false, ErrNonQueryData
}

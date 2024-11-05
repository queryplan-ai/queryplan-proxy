package types

import (
	heartbeattypes "github.com/queryplan-ai/queryplan-proxy/pkg/heartbeat/types"
	"github.com/tuvistavie/securerandom"
)

type ConnectionState struct {
	ID           string
	RowCount     int64
	CurrentQuery *heartbeattypes.CurrentQuery
}

func NewConnectionState() (*ConnectionState, error) {
	connectionID, err := securerandom.Hex(4)
	if err != nil {
		return nil, err
	}

	return &ConnectionState{
		ID:       connectionID,
		RowCount: 0,
	}, nil
}

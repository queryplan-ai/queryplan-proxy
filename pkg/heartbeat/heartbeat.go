package heartbeat

import (
	"strings"
	"time"

	"github.com/queryplan-ai/queryplan-proxy/pkg/heartbeat/types"
)

func AddPendingQuery(query string, isPreparedStatement bool) {
	// some queries we filter here
	if isFilteredQuery(query) {
		return
	}

	qpq := types.QueryPlanQuery{
		Query:               query,
		ExecutedAt:          time.Now().UnixNano(),
		IsPreparedStatement: isPreparedStatement,
	}

	pendingQueries.Add(qpq)
}

func isFilteredQuery(query string) bool {
	if strings.ToLower(query) == "select ?" {
		return true
	}

	if strings.ToLower(query) == "start transaction" {
		return true
	}

	if strings.ToLower(query) == "commit" {
		return true
	}

	if strings.ToLower(query) == "rollback" {
		return true
	}

	if strings.ToLower(query) == "SELECT @@max_allowed_packet" {
		return true
	}

	return false
}

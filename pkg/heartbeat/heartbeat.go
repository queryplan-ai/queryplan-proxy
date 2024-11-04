package heartbeat

import (
	"strings"
	"time"

	"github.com/queryplan-ai/queryplan-proxy/pkg/heartbeat/types"
)

func CompleteCurrentQuery(rowCount int64) {
	if currentQuery == nil {
		return
	}

	duration := time.Now().UnixNano() - currentQuery.ExecutionStartedAt

	AddPendingQuery(*currentQuery, duration, rowCount)
	currentQuery = nil
}

func SetCurrentQuery(query string, isPreparedStatement bool) {
	currentQuery = &types.QueryPlanCurrentQuery{
		Query:               query,
		ExecutionStartedAt:  time.Now().UnixNano(),
		IsPreparedStatement: isPreparedStatement,
	}
}

func AddPendingQuery(currentQuery types.QueryPlanCurrentQuery, duration int64, rowCount int64) {
	// some queries we filter here
	if isFilteredQuery(currentQuery.Query) {
		return
	}

	qpq := types.QueryPlanQuery{
		Query:               currentQuery.Query,
		ExecutedAt:          time.Now().UnixNano(),
		RowCount:            rowCount,
		Duration:            duration,
		IsPreparedStatement: currentQuery.IsPreparedStatement,
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

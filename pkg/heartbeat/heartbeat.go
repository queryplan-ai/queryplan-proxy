package heartbeat

import (
	"time"

	"github.com/queryplan-ai/queryplan-proxy/pkg/heartbeat/types"
)

func AddPendingQuery(query string) {
	qpq := types.QueryPlanQuery{
		Query:      query,
		ExecutedAt: time.Now().UnixNano(),
	}

	pendingQueries.Add(qpq)
}

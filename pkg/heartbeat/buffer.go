package heartbeat

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	daemontypes "github.com/queryplan-ai/queryplan-proxy/pkg/daemon/types"
	heartbeattypes "github.com/queryplan-ai/queryplan-proxy/pkg/heartbeat/types"
	"github.com/queryplan-ai/queryplan-proxy/pkg/ringbuffer"
)

const (
	defaultMaxPendingQueriesSize = 10000
)

var (
	// pendingQueries is the ring buffer that are pending to send to the API
	pendingQueries = ringbuffer.New[heartbeattypes.QueryPlanQuery](defaultMaxPendingQueriesSize)
)

func SendPendingQueries(ctx context.Context, opts daemontypes.DaemonOpts) error {
	queries := pendingQueries.GetAll()
	if len(queries) == 0 {
		return nil
	}

	payload := heartbeattypes.QueryPlanQueriesPayload{
		Queries: queries,
	}

	marshaled, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal payload: %v", err)
	}

	url := fmt.Sprintf("%s/v1/queries", opts.APIURL)
	req, err := http.NewRequest("PUT", url, bytes.NewBuffer(marshaled))
	if err != nil {
		return fmt.Errorf("create request: %v", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", opts.Token))

	if opts.Environment != "" {
		req.Header.Set("X-QueryPlan-Environment", opts.Environment)
	}

	req.Header.Set("X-QueryPlan-DBMS", string(opts.DBMS))
	req.Header.Set("X-QueryPlan-Database", opts.DatabaseName)

	client := http.DefaultClient
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("send request: %v", err)
	}

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	pendingQueries.Clear()

	return nil
}

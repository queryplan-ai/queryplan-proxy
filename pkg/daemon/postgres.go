package daemon

import (
	"context"

	"github.com/queryplan-ai/queryplan-proxy/pkg/daemon/types"
)

func runPostgres(ctx context.Context, opts types.DaemonOpts) {
	<-ctx.Done()
}

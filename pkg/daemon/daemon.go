package daemon

import (
	"context"
	"fmt"
	"os"

	"github.com/queryplan-ai/queryplan-proxy/pkg/daemon/types"
)

func Run(ctx context.Context, opts types.DaemonOpts) {
	switch opts.DBMS {
	case types.Postgres:
		runPostgres(ctx, opts)
	case types.Mysql:
		runMysql(ctx, opts)
	default:
		fmt.Printf("Unsupported DBMS: %s\n", opts.DBMS)
		os.Exit(1)
	}
}

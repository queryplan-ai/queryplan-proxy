package daemon

import (
	"context"
	"fmt"
	"os"
	"sync"

	"github.com/queryplan-ai/queryplan-proxy/pkg/daemon/types"
	"github.com/queryplan-ai/queryplan-proxy/pkg/mysql"
	"github.com/queryplan-ai/queryplan-proxy/pkg/postgres"
)

func Run(ctx context.Context, opts types.DaemonOpts) {
	switch opts.DBMS {
	case types.Postgres:
		var wg sync.WaitGroup
		wg.Add(2)

		go func() {
			defer wg.Done()
			postgres.ProcessSchema(ctx, opts)
		}()
		go func() {
			defer wg.Done()
			postgres.RunProxy(ctx, opts)
		}()

		wg.Wait()
	case types.Mysql:
		var wg sync.WaitGroup
		wg.Add(2)

		go func() {
			defer wg.Done()
			mysql.ProcessSchema(ctx, opts)
		}()
		go func() {
			defer wg.Done()
			mysql.RunProxy(ctx, opts)
		}()

		wg.Wait()
	default:
		fmt.Printf("Unsupported DBMS: %s\n", opts.DBMS)
		os.Exit(1)
	}
}

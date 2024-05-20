package daemon

import (
	"context"
	"fmt"
	"log"
	"os"
	"sync"
	"time"

	"github.com/queryplan-ai/queryplan-proxy/pkg/daemon/types"
	"github.com/queryplan-ai/queryplan-proxy/pkg/heartbeat"
	"github.com/queryplan-ai/queryplan-proxy/pkg/mysql"
	"github.com/queryplan-ai/queryplan-proxy/pkg/postgres"
)

const (
	sendInterval = 10 * time.Second
)

func Run(ctx context.Context, opts types.DaemonOpts) {
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case <-time.After(sendInterval):
				if err := heartbeat.SendPendingQueries(ctx, opts); err != nil {
					log.Printf("Error sending pending queries: %v", err)
				}
			}
		}
	}()

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

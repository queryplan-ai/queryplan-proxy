package main

import (
	"fmt"
	"time"

	"github.com/queryplan-ai/queryplan-proxy/test/integration/pkg/proxy"
	"github.com/queryplan-ai/queryplan-proxy/test/integration/pkg/server"
	"github.com/queryplan-ai/queryplan-proxy/test/integration/pkg/upstream"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"golang.org/x/exp/rand"
)

func PostgresCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:          "postgres",
		SilenceUsage: true,
		PersistentPreRun: func(cmd *cobra.Command, args []string) {
			viper.BindPFlags(cmd.Flags())
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			// pick a random upstream port
			fmt.Printf("Starting postgres\n")
			upstreamProcess, err := upstream.StartPostgres(false, false)
			if err != nil {
				return err
			}
			defer upstreamProcess.Stop()

			// pick a random port
			fmt.Printf("Starting mock api server\n")
			bindPort := 3000 + rand.Intn(1000)
			mockServerPort, err := server.StartMockServer()
			if err != nil {
				return err
			}

			fmt.Printf("Starting proxy\n")
			v := viper.GetViper()
			proxyProcess, err := proxy.StartProxy(
				v.GetString("proxy-binary"),
				"start",
				"--bind-address=0.0.0.0",
				fmt.Sprintf("--bind-port=%d", bindPort),
				"--upstream-address=localhost",
				fmt.Sprintf("--upstream-port=%d", upstreamProcess.Port()),
				"--dbms=postgres",
				"--live-connection-uri=postgres://postgres:password@localhost:5432/postgres",
				fmt.Sprintf("--api-url=http://localhost:%d", mockServerPort),
				"--token=a-token",
				"--env=dev",
			)

			if err != nil {
				return err
			}

			time.Sleep(time.Second * 10)
			defer proxyProcess.Stop()

			return nil
		},
	}

	return cmd
}

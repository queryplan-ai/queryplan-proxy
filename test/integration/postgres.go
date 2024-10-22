package main

import (
	"context"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
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
			queryReceivedCh := make(chan string)
			mockServerPort, err := server.StartMockServer(server.MockServerOpts{
				QueryReceivedCh: queryReceivedCh,
			})
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
			defer proxyProcess.Stop()

			queriesReceived := []string{}
			go func() {
				for query := range queryReceivedCh {
					queriesReceived = append(queriesReceived, query)
				}
			}()

			if err := SendPostgresQueries("localhost", bindPort, "postgres", "password"); err != nil {
				return err
			}

			if err := AssertQueriesReceived(queriesReceived); err != nil {
				return err
			}
			return nil
		},
	}

	return cmd
}

func SendPostgresQueries(postgresHost string, postgresPort int, username string, password string) error {
	connectionURI := fmt.Sprintf("postgres://%s:%s@%s:%d/postgres", username, password, postgresHost, postgresPort)
	for _, query := range queriesToSend {
		conn, err := pgx.Connect(context.Background(), connectionURI)
		if err != nil {
			return err
		}
		defer conn.Close(context.Background())

		_, err = conn.Exec(context.Background(), query)
		if err != nil {
			return err
		}

		conn.Close(context.Background())
	}
	return nil
}

func AssertQueriesReceived(queriesReceived []string) error {
	receivedSet := make(map[string]bool)
	expectedSet := make(map[string]bool)

	// Populate sets
	for _, query := range queriesReceived {
		receivedSet[strings.TrimSpace(query)] = true
	}
	for _, query := range queriesToExpect {
		expectedSet[strings.TrimSpace(query)] = true
	}

	var unexpectedQueries []string
	var extraQueries []string

	// Find unexpected queries
	for query := range receivedSet {
		if !expectedSet[query] {
			unexpectedQueries = append(unexpectedQueries, query)
		}
	}

	// Find extra queries
	for query := range expectedSet {
		if !receivedSet[query] {
			extraQueries = append(extraQueries, query)
		}
	}

	if len(unexpectedQueries) > 0 || len(extraQueries) > 0 {
		errorMsg := ""
		if len(unexpectedQueries) > 0 {
			errorMsg += fmt.Sprintf("Unexpected queries received: %v\n", unexpectedQueries)
		}
		if len(extraQueries) > 0 {
			errorMsg += fmt.Sprintf("Expected queries not received: %v\n", extraQueries)
		}
		return fmt.Errorf(errorMsg)
	}

	return nil
}

var (
	queriesToSend = []string{
		"select * from users",
		"select * from users where id = 1",
	}

	queriesToExpect = []string{
		"select * from users",
		"select * from users where id = $1",
	}
)

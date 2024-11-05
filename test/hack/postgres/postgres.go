package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"
	"sync"

	"github.com/jackc/pgx/v5"
)

func main() {
	dsn := os.Getenv("QUERYPLAN_LIVE_CONNECTION_URI")
	if dsn == "" {
		log.Fatal("Environment variable QUERYPLAN_LIVE_CONNECTION_URI is not set")
	}

	// replace the upstream port with the bind port, this is really hacky
	upstreamPort := os.Getenv("QUERYPLAN_UPSTREAM_PORT")
	bindPort := os.Getenv("QUERYPLAN_BIND_PORT")

	fmt.Printf("Replacing upstream port %s with bind port %s\n", upstreamPort, bindPort)
	dsn = strings.Replace(dsn, upstreamPort, bindPort, 1)

	fmt.Printf("Connecting to database: %s\n", dsn)
	conn, err := pgx.Connect(context.Background(), dsn)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Unable to connect to database: %v\n", err)
		os.Exit(1)
	}
	defer conn.Close(context.Background())

	executeSimpleQuery := func(conn *pgx.Conn, wg *sync.WaitGroup) {
		defer wg.Done()
		query := "select query_id from query_execution limit 8"
		rows, err := conn.Query(context.Background(), query)
		if err != nil {
			log.Fatalf("Failed to execute query: %v", err)
		}
		defer rows.Close()

		for rows.Next() {
			id := ""
			if err := rows.Scan(&id); err != nil {
				log.Fatalf("Failed to scan row: %v", err)
			}

			fmt.Printf("Query executed successfully, result: %s\n", id)
		}
	}

	// start 5 goroutines to execute queries concurrently, waiting for all to finish
	simpleWg := sync.WaitGroup{}
	for i := 0; i < 1; i++ {
		simpleWg.Add(1)
		go executeSimpleQuery(conn, &simpleWg)
	}

	simpleWg.Wait()

}

package main

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"strings"
	"sync"
	"time"

	_ "github.com/go-sql-driver/mysql"
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
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		log.Fatalf("Failed to ping database: %v", err)
	}

	executeSimpleQuery := func(db *sql.DB, wg *sync.WaitGroup) {
		defer wg.Done()
		query := "select id from cluster_history limit 8"
		rows, err := db.Query(query)
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
	for i := 0; i < 10; i++ {
		simpleWg.Add(1)
		go executeSimpleQuery(db, &simpleWg)
	}

	simpleWg.Wait()

	executePreparedQuery := func(db *sql.DB, wg *sync.WaitGroup) {
		defer wg.Done()
		query := `select id from cluster_history where created_at < ? limit 8`
		when, _ := time.Parse(time.RFC3339, "2023-08-09T00:00:00Z")
		// query := "select id from cluster_history where id <> ?"
		rows, err := db.Query(query, when)
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
	preparedWg := sync.WaitGroup{}
	for i := 0; i < 10; i++ {
		preparedWg.Add(1)
		go executePreparedQuery(db, &preparedWg)
	}

	preparedWg.Wait()
}

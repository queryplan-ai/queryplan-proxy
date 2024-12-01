package mysql

import (
	"database/sql"
	"fmt"
	"math/rand"
	"os"
	"reflect"
	"syscall"
	"time"

	"github.com/queryplan-ai/queryplan-proxy/cmd/queryplan-proxy/cli"
	"github.com/queryplan-ai/queryplan-proxy/test/integration/pkg/mocks"
	"github.com/queryplan-ai/queryplan-proxy/test/integration/pkg/upstream"
)

func Execute() ([]*upstream.UpsreamProcess, error) {
	upstreamProcesses := []*upstream.UpsreamProcess{}

	standardUpstream, err := executeStandardQueries()
	if err != nil {
		return nil, err
	}
	upstreamProcesses = append(upstreamProcesses, standardUpstream)

	preparedUpstream, err := executePreparedQueries()
	if err != nil {
		return nil, err
	}
	upstreamProcesses = append(upstreamProcesses, preparedUpstream)

	return upstreamProcesses, nil
}

func executeStandardQueries() (*upstream.UpsreamProcess, error) {
	schema := []string{
		"CREATE TABLE users (id int, name varchar(255))",
		"CREATE TABLE posts (id int, title varchar(255))",
		"CREATE TABLE comments (id int, post_id int, content varchar(255))",
		"CREATE TABLE tags (id int, name varchar(255))",
	}

	fixtures := []string{
		"INSERT INTO users (id, name) VALUES (1, 'John')",
		"INSERT INTO posts (id, title) VALUES (1, 'Hello World')",
		"INSERT INTO comments (id, post_id, content) VALUES (1, 1, 'Hello World')",
		"INSERT INTO tags (id, name) VALUES (1, 'hello')",
	}

	u, err := upstream.StartMysql(false, false, schema, fixtures)
	if err != nil {
		return nil, err
	}

	mockServerPort := 3000 + rand.Intn(1000)
	mockServer, err := mocks.StartMockServer(mockServerPort)
	if err != nil {
		return nil, err
	}

	proxyBindPort := 3000 + rand.Intn(1000)
	fmt.Printf("Starting proxy on port %d\n", proxyBindPort)
	signalChan := make(chan os.Signal, 1)
	rootCmd := cli.RootCmd(&signalChan)

	rootCmd.SetArgs([]string{
		"start",
		"--bind-address=0.0.0.0",
		fmt.Sprintf("--bind-port=%d", proxyBindPort),
		"--upstream-address=localhost",
		fmt.Sprintf("--upstream-port=%d", u.Port()),
		"--dbms=mysql",
		fmt.Sprintf("--live-connection-uri=testuser:%s@tcp(localhost:%d)/testdb", u.Password(), u.Port()),
		fmt.Sprintf("--api-url=http://localhost:%d", mockServerPort),
		"--token=a-token",
		"--env=dev",
	})

	done := make(chan error, 1)
	go func() {
		done <- rootCmd.Execute()
	}()

	// wait for the proxy to start
	time.Sleep(time.Second * 5)

	queriesToSend := []Query{
		{
			Query: "select id from users",
			ScanArgs: []interface{}{
				&sql.NullInt64{},
			},
		},
		{
			Query: "select id from posts",
			ScanArgs: []interface{}{
				&sql.NullInt64{},
			},
		},
		{
			Query: "select id from comments",
			ScanArgs: []interface{}{
				&sql.NullInt64{},
			},
		},
		{
			Query: "select name from tags",
			ScanArgs: []interface{}{
				&sql.NullString{},
			},
		},
	}
	if err := sendMysqlQueries(fmt.Sprintf("testuser:%s@tcp(localhost:%d)/testdb", u.Password(), proxyBindPort), queriesToSend); err != nil {
		return nil, err
	}

	// wait 10 seconds for the queries to be received
	time.Sleep(10 * time.Second)

	signalChan <- syscall.SIGTERM

	// stop the mock server
	if err := mockServer.Stop(); err != nil {
		return nil, err
	}

	expectedQueries := []string{}
	receivedQueries := []string{}
	for _, receivedQuery := range mockServer.ReceivedQueries {
		receivedQueries = append(receivedQueries, receivedQuery.Query)
	}
	for _, expectedQuery := range queriesToSend {
		expectedQueries = append(expectedQueries, expectedQuery.Query)
	}

	// ensure that all match
	if !reflect.DeepEqual(expectedQueries, receivedQueries) {
		return nil, fmt.Errorf("expected queries %v, received queries %v", expectedQueries, receivedQueries)
	}

	<-done

	return u, nil
}

func executePreparedQueries() (*upstream.UpsreamProcess, error) {
	schema := []string{
		"CREATE TABLE users (id int, name varchar(255))",
		"CREATE TABLE posts (id int, title varchar(255))",
		"CREATE TABLE comments (id int, post_id int, content varchar(255))",
		"CREATE TABLE tags (id int, name varchar(255))",
	}

	fixtures := []string{
		"INSERT INTO users (id, name) VALUES (1, 'John')",
		"INSERT INTO posts (id, title) VALUES (1, 'Hello World')",
		"INSERT INTO comments (id, post_id, content) VALUES (1, 1, 'Hello World')",
		"INSERT INTO tags (id, name) VALUES (1, 'hello')",
	}

	u, err := upstream.StartMysql(false, false, schema, fixtures)
	if err != nil {
		return nil, err
	}

	mockServerPort := 3000 + rand.Intn(1000)
	mockServer, err := mocks.StartMockServer(mockServerPort)
	if err != nil {
		return nil, err
	}

	proxyBindPort := 3000 + rand.Intn(1000)
	fmt.Printf("Starting proxy on port %d\n", proxyBindPort)
	signalChan := make(chan os.Signal, 1)
	rootCmd := cli.RootCmd(&signalChan)

	rootCmd.SetArgs([]string{
		"start",
		"--bind-address=0.0.0.0",
		fmt.Sprintf("--bind-port=%d", proxyBindPort),
		"--upstream-address=localhost",
		fmt.Sprintf("--upstream-port=%d", u.Port()),
		"--dbms=mysql",
		fmt.Sprintf("--live-connection-uri=testuser:%s@tcp(localhost:%d)/testdb", u.Password(), u.Port()),
		fmt.Sprintf("--api-url=http://localhost:%d", mockServerPort),
		"--token=a-token",
		"--env=dev",
	})

	done := make(chan error, 1)
	go func() {
		done <- rootCmd.Execute()
	}()

	// wait for the proxy to start
	time.Sleep(time.Second * 5)

	queriesToSend := []Query{
		{
			Query: "select id from users where id = ?",
			Args: []interface{}{
				1,
			},
			ScanArgs: []interface{}{
				&sql.NullInt64{},
			},
		},
		{
			Query: "select id from posts where id = ?",
			Args: []interface{}{
				1,
			},
			ScanArgs: []interface{}{
				&sql.NullInt64{},
			},
		},
		{
			Query: "select id, post_id, content from comments where id = ?",
			Args: []interface{}{
				1,
			},
			ScanArgs: []interface{}{
				&sql.NullInt64{},
				&sql.NullInt64{},
				&sql.NullString{},
			},
		},
		{
			Query: "select name from tags where id = ?",
			Args: []interface{}{
				1,
			},
			ScanArgs: []interface{}{
				&sql.NullString{},
			},
		},
	}
	if err := sendMysqlQueries(fmt.Sprintf("testuser:%s@tcp(localhost:%d)/testdb", u.Password(), proxyBindPort), queriesToSend); err != nil {
		return nil, err
	}

	// wait 10 seconds for the queries to be received
	time.Sleep(10 * time.Second)

	signalChan <- syscall.SIGTERM

	// stop the mock server
	if err := mockServer.Stop(); err != nil {
		return nil, err
	}

	expectedQueries := []string{}
	receivedQueries := []string{}
	for _, receivedQuery := range mockServer.ReceivedQueries {
		receivedQueries = append(receivedQueries, receivedQuery.Query)
	}
	for _, expectedQuery := range queriesToSend {
		expectedQueries = append(expectedQueries, expectedQuery.Query)
	}

	// ensure that all match
	if !reflect.DeepEqual(expectedQueries, receivedQueries) {
		return nil, fmt.Errorf("expected queries %v, received queries %v", expectedQueries, receivedQueries)
	}

	<-done

	return u, nil
}

type Query struct {
	Query    string
	Args     []interface{}
	ScanArgs []interface{}
}

func sendMysqlQueries(dsn string, queries []Query) error {
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return err
	}
	defer db.Close()

	for _, query := range queries {
		rows, err := db.Query(query.Query, query.Args...)
		if err != nil {
			fmt.Printf("Error executing query: %s\n", query.Query)
			return err
		}
		defer rows.Close()

		for rows.Next() {
			if err := rows.Scan(query.ScanArgs...); err != nil {
				return err
			}
		}
	}

	return nil
}

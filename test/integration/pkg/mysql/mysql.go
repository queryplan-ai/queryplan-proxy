package mysql

import (
	"fmt"
	"math/rand"
	"os"
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

	return upstreamProcesses, nil
}

func executeStandardQueries() (*upstream.UpsreamProcess, error) {
	u, err := upstream.StartMysql(false, false)
	if err != nil {
		return nil, err
	}

	mockServerPort := 3000 + rand.Intn(1000)
	mockServer, err := mocks.StartMockServer(mockServerPort)
	if err != nil {
		return nil, err
	}

	proxyBindPort := 3000 + rand.Intn(1000)

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

	time.Sleep(10 * time.Second)

	signalChan <- syscall.SIGTERM

	// stop the mock server
	if err := mockServer.Stop(); err != nil {
		return nil, err
	}

	fmt.Printf("Received queries: %s\n", mockServer.ReceivedQueries)
	<-done

	return u, nil
}

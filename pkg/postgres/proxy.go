package postgres

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"sync"

	daemontypes "github.com/queryplan-ai/queryplan-proxy/pkg/daemon/types"
	"github.com/queryplan-ai/queryplan-proxy/pkg/postgres/types"
)

func RunProxy(ctx context.Context, opts daemontypes.DaemonOpts) {
	address := fmt.Sprintf("%s:%v", opts.BindAddress, opts.BindPort)
	upstreamAddress := fmt.Sprintf("%s:%v", opts.UpstreamAddress, opts.UpstreamPort)

	fmt.Printf("Listening on %s, proxying to %s\n", address, upstreamAddress)

	listener, err := net.Listen("tcp", address)
	if err != nil {
		panic(err)
	}
	defer listener.Close()

	for {
		select {
		case <-ctx.Done():
			return
		default:
			localConn, err := listener.Accept()
			if err != nil {
				panic(err)
			}
			go handlePostgresConnection(localConn, upstreamAddress)
		}
	}
}

func handlePostgresConnection(localConn net.Conn, targetAddress string) {
	targetConn, err := net.Dial("tcp", targetAddress)
	if err != nil {
		log.Printf("Failed to connect to target address %s: %v", targetAddress, err)
		localConn.Close()
		return
	}

	var wg sync.WaitGroup
	wg.Add(2)

	connectionState, err := types.NewConnectionState()
	if err != nil {
		log.Printf("Error creating connection state: %v", err)
		localConn.Close()
		return
	}

	go func() {
		defer wg.Done()
		if err := copyAndInspectCommand(localConn, targetConn, connectionState, true); err != nil {
			log.Printf("Error in data transfer from local to target: %v", err)
		}
	}()

	go func() {
		defer wg.Done()
		if err := copyAndInspectResponse(targetConn, localConn, connectionState, true); err != nil {
			if errors.Is(err, io.EOF) {
				// safe to ignore, the client went away
				return
			}

			log.Printf("Error in data transfer from target to local: %v", err)
		}
	}()

	wg.Wait()
	localConn.Close()
	targetConn.Close()
}

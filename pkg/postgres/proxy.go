package postgres

import (
	"context"
	"fmt"
	"log"
	"net"
	"sync"

	daemontypes "github.com/queryplan-ai/queryplan-proxy/pkg/daemon/types"
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

	go func() {
		defer wg.Done()
		if err := copyAndInspectCommand(localConn, targetConn, true); err != nil {
			log.Printf("Error in data transfer from local to target: %v", err)
		}
	}()

	go func() {
		defer wg.Done()
		if err := copyAndInspectResponse(targetConn, localConn, true); err != nil {
			log.Printf("Error in data transfer from target to local: %v", err)
		}
	}()

	wg.Wait()
	localConn.Close()
	targetConn.Close()
}

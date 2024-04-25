package daemon

import (
	"context"
	"fmt"
	"io"
	"log"
	"net"
	"sync"

	"github.com/queryplan-ai/queryplan-proxy/pkg/daemon/types"
)

func runMysql(ctx context.Context, opts types.DaemonOpts) {
	address := fmt.Sprintf("%s:%d", opts.BindAddress, opts.BindPort)

	listener, err := net.Listen("tcp", address)
	if err != nil {
		panic(err)
	}
	defer listener.Close()

	upstreamAddress := fmt.Sprintf("%s:%d", opts.UpstreamAddress, opts.UpstreamPort)

	fmt.Printf("Listening on %s, proxying to %s\n", address, upstreamAddress)

	for {
		select {
		case <-ctx.Done():
			return
		default:
			localConn, err := listener.Accept()
			if err != nil {
				panic(err)
			}
			go handleMysqlConnection(localConn, upstreamAddress)
		}
	}
}

func handleMysqlConnection(localConn net.Conn, targetAddress string) {
	fmt.Printf("Accepted connection from %s\n", localConn.RemoteAddr())
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
		if err := copyAndInspect(localConn, targetConn, true); err != nil {
			log.Printf("Error in data transfer from local to target: %v", err)
		}
	}()

	go func() {
		defer wg.Done()
		if err := copyAndInspect(targetConn, localConn, false); err != nil {
			log.Printf("Error in data transfer from target to local: %v", err)
		}
	}()

	wg.Wait()
	localConn.Close()
	targetConn.Close()
}

func copyAndInspect(src, dst net.Conn, inspect bool) error {
	buffer := make([]byte, 4096)
	for {
		n, err := src.Read(buffer)
		if err != nil {
			if err != io.EOF {
				return err
			}
			break
		}

		data := buffer[:n]
		if inspect {
			// Here we can check for SQL queries. This is a simplification.
			log.Printf("Inspecting data: %s", extractQuery(data))
		}

		if _, err := dst.Write(data); err != nil {
			return err
		}
	}
	return nil
}

=func extractQuery(data []byte) string {
	if len(data) < 5 {
		return "Non-query data or incomplete packet"
	}
	if data[4] == 0x03 { // COM_QUERY
		return string(data[5:])
	}
	return "Non-query data"
}

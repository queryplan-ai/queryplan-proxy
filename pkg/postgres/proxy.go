package postgres

import (
	"context"
	"fmt"
	"io"
	"log"
	"net"

	daemontypes "github.com/queryplan-ai/queryplan-proxy/pkg/daemon/types"
	"github.com/queryplan-ai/queryplan-proxy/pkg/heartbeat"
)

func RunProxy(ctx context.Context, opts daemontypes.DaemonOpts) {
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
			go handlePostgresConnection(localConn, upstreamAddress)
		}
	}
}

func handlePostgresConnection(localConn net.Conn, upstreamAddress string) {
	upstreamConn, err := net.Dial("tcp", upstreamAddress)
	if err != nil {
		panic(err)
	}
	defer upstreamConn.Close()

	// Create TeeReaders to log the data while forwarding it
	localReader := io.TeeReader(localConn, newLoggingWriter("Client -> Server"))
	upstreamReader := io.TeeReader(upstreamConn, newLoggingWriter("Server -> Client"))

	go func() {
		defer localConn.Close()
		io.Copy(upstreamConn, localReader)
	}()

	io.Copy(localConn, upstreamReader)
}

// loggingWriter is an io.Writer that logs the data written to it.
type loggingWriter struct {
	prefix string
}

func newLoggingWriter(prefix string) *loggingWriter {
	return &loggingWriter{prefix: prefix}
}

func (w *loggingWriter) Write(p []byte) (n int, err error) {
	// copy into a buffer
	buf := make([]byte, len(p))
	copy(buf, p)

	// look at the first byte
	if len(buf) > 0 {
		// if the first char is a Q it's a query
		if buf[0] == 'Q' {
			// print the query, which starts at the 3rd byte
			// this is far too niaive, but it's a start
			// and cannot by shipped because it's
			// very incomplete
			query := string(buf[5:])
			// remove the last null byte
			query = query[:len(query)-1]

			cleanedQuery, err := cleanQuery(query)
			if err != nil {
				log.Printf("Error cleaning query: %v", err)
			} else {
				heartbeat.AddPendingQuery(cleanedQuery, false)
			}
		}
	}

	return len(p), nil
}

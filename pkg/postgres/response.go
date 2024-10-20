package postgres

import (
	"io"
	"log"
	"net"
)

func copyAndInspectResponse(src, dst net.Conn, inspect bool) error {
	buf := make([]byte, 4096)
	for {
		n, err := src.Read(buf)
		if err != nil {
			if err != io.EOF {
				log.Printf("Error reading from upstream: %v", err)
			}
			return err
		}

		if inspect {
			// Inspect the data here if needed (currently just logging)
			// log.Printf("Response: %s", string(buf[:n]))
		}

		_, err = dst.Write(buf[:n])
		if err != nil {
			log.Printf("Error writing to client: %v", err)
			return err
		}
	}
}

package postgres

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"net"
	"strconv"

	"github.com/queryplan-ai/queryplan-proxy/pkg/heartbeat"
	"github.com/queryplan-ai/queryplan-proxy/pkg/postgres/types"
)

type PostgresResponseType byte

const (
	PostgresResponseTypeRowDescription  = 'T'
	PostgresResponseTypeDataRow         = 'D'
	PostgresResponseTypeCommandComplete = 'C'
	PostgresResponseTypeErrorResponse   = 'E'
	PostgresResponseTypeAuthentication  = 'R'
)

func copyAndInspectResponse(src net.Conn, dst net.Conn, connectionState *types.ConnectionState, inspect bool) error {
	var accum bytes.Buffer
	buf := make([]byte, 8192)

	for {
		n, err := src.Read(buf)
		if err != nil {
			if err != io.EOF {
				log.Printf("Error reading from upstream: %v", err)
			}
			return err
		}

		accum.Write(buf[:n])

		// continue to read every possible complete packet from accum
		for {
			if len(accum.Bytes()) < 5 {
				break
			}

			data := accum.Bytes()
			messageType := PostgresResponseType(data[0])
			messageLength := int(data[1])<<24 | int(data[2])<<16 | int(data[3])<<8 | int(data[4])

			fmt.Printf("read %d bytes, this message is %d bytes long and type is %c\n", len(data), messageLength, messageType)

			if len(data) < messageLength {
				break
			}

			// send the data
			dataToForward := data[:messageLength+1]
			_, err = dst.Write(dataToForward)
			if err != nil {
				log.Printf("Error writing to client: %v", err)
				return err
			}

			// parse this message
			switch messageType {
			case PostgresResponseTypeRowDescription:
				if len(data) < 7 {
					return fmt.Errorf("incomplete row description message")
				}
			case PostgresResponseTypeDataRow:
			case PostgresResponseTypeCommandComplete:
				commandTag := string(data[5:messageLength])
				rowCount := int64(-1)
				if len(commandTag) > 6 && commandTag[6] == ' ' {
					rowCountPart := commandTag[7:] // "SELECT"
					rowCountInt, err := strconv.Atoi(rowCountPart)
					if err != nil {
						log.Printf("Error parsing row count: %v", err)
					}
					rowCount = int64(rowCountInt)
				}
				heartbeat.CompleteCurrentQuery(nil, rowCount)
			case PostgresResponseTypeErrorResponse:
				log.Printf("Error in Response: %s", string(data[5:messageLength]))
			case PostgresResponseTypeAuthentication:
			default:
				log.Printf("Unhandled response type: %c", messageType)
			}

			// remove this message from the buffer
			accum.Next(messageLength + 1)

		}

	}
}

package mysql

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"net"
	"strings"

	"github.com/pubnative/mysqlproto-go"
	"github.com/queryplan-ai/queryplan-proxy/pkg/mysql/types"
)

var (
	ErrNonQueryData                   = fmt.Errorf("non-query data")
	ErrNonQueryDataOrIncompletePacket = fmt.Errorf("non-query data or incomplete packet")
)

const (
	COM_QUERY = 0x03
)

func copyAndInspectCommand(src, dst net.Conn, inspect bool) error {
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
			query, err := extractQuery(data)
			if err == nil {
				log.Printf("Query: %s", query)
			}
		}

		if _, err := dst.Write(data); err != nil {
			return err
		}
	}
	return nil
}

func extractQuery(data []byte) (string, error) {
	if len(data) < 5 {
		return "", ErrNonQueryDataOrIncompletePacket
	}
	if data[4] == COM_QUERY {
		return strings.TrimSpace(string(data[5:])), nil
	}
	return "", ErrNonQueryData
}

func copyAndInspectResponse(src, dst net.Conn, inspect bool) error {
	var buffer bytes.Buffer
	for {
		data := make([]byte, 4096)
		n, err := src.Read(data)
		if err != nil {
			if err != io.EOF {
				return err
			}
			break
		}

		if _, err := dst.Write(data[:n]); err != nil {
			return err
		}

		if inspect {
			buffer.Write(data[:n])     // Accumulate data for inspection
			if isFullPacket(&buffer) { // Simplified check for a full packet
				packet, err := parseFullResponsePacket(buffer.Bytes())
				if err != nil {
					return err
				}

				log.Printf("Packet: %v", packet)
			}
		}
	}
	return nil
}

func isFullPacket(buffer *bytes.Buffer) bool {
	if buffer.Len() < 4 {
		return false
	}

	data := buffer.Bytes()
	length := int(data[0]) | int(data[1])<<8 | int(data[2])<<16

	return buffer.Len() >= length+4 // Total length including the header
}

func parseFullResponsePacket(data []byte) (*types.COM_Query_ResponsePacket, error) {
	if len(data) < 4 {
		return nil, ErrNonQueryDataOrIncompletePacket
	}

	// attempt to use the go-mysqlproto library to parse the packet
	// if it's a OK or ERR packet
	okPacket, err := mysqlproto.ParseOKPacket(data, 0)
	if err == nil {
		return &types.COM_Query_ResponsePacket{OK_Packet: &okPacket}, nil
	}

	errPacket, err := mysqlproto.ParseERRPacket(data, 0)
	if err == nil {
		return &types.COM_Query_ResponsePacket{ERR_Packet: &errPacket}, nil
	}

	// if it's a text resultset packet, we need to parse it differently
	// as the go-mysqlproto library doesn't support it
	return &types.COM_Query_ResponsePacket{TextResultsetPacket: &types.COM_Query_TextResultsetPacket{}}, nil
}

package mocks

import (
	"fmt"
	"io"
	"net/http"
)

type MockServer struct {
	Port            int
	Server          *http.Server
	ReceivedQueries []string
}

func (m *MockServer) Stop() error {
	if m.Server != nil {
		return m.Server.Close()
	}

	return nil
}

func StartMockServer(port int) (*MockServer, error) {
	// start an http server with 2 methods to
	// receive and log the requests

	mockServer := &MockServer{
		Port:            port,
		ReceivedQueries: []string{},
	}

	handleRequest := func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/v1/schema" {
			w.WriteHeader(http.StatusOK)
			return
		} else if r.URL.Path == "/v1/queries" {
			// parse the body
			body, err := io.ReadAll(r.Body)
			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				return
			}

			fmt.Printf("Query: %s\n", string(body))
			mockServer.ReceivedQueries = append(mockServer.ReceivedQueries, "query")
			w.WriteHeader(http.StatusOK)
			return
		}

		fmt.Printf("Request: %s %s\n", r.Method, r.URL.Path)
		w.WriteHeader(http.StatusNotFound)
	}

	mockServer.Server = &http.Server{
		Addr:    fmt.Sprintf(":%d", mockServer.Port),
		Handler: http.HandlerFunc(handleRequest),
	}
	go mockServer.Server.ListenAndServe()

	return mockServer, nil
}

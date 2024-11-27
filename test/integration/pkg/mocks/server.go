package mocks

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

type MockServer struct {
	Port            int
	Server          *http.Server
	ReceivedQueries []Query
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
		ReceivedQueries: []Query{},
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

			putBody := PutQueriesParametersBody{}
			if err := json.Unmarshal(body, &putBody); err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				return
			}

			for _, query := range putBody.Queries {
				mockServer.ReceivedQueries = append(mockServer.ReceivedQueries, query)
			}

			w.WriteHeader(http.StatusOK)
			return
		}

		w.WriteHeader(http.StatusNotFound)
	}

	mockServer.Server = &http.Server{
		Addr:    fmt.Sprintf(":%d", mockServer.Port),
		Handler: http.HandlerFunc(handleRequest),
	}
	go mockServer.Server.ListenAndServe()

	return mockServer, nil
}

// the following types are copied from the queryplan api code

type PutQueriesParametersBody struct {
	Queries []Query `json:"queries"`
}

type Query struct {
	ExecutedAt          int64  `json:"executed_at"`
	Duration            int64  `json:"duration"`
	RowCount            int64  `json:"row_count"`
	Query               string `json:"query"`
	IsPreparedStatement bool   `json:"is_prepared_statement"`
}

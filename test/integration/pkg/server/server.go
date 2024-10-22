package server

import (
	"fmt"
	"io"
	"log"
	"net/http"

	"golang.org/x/exp/rand"
)

type MockServerOpts struct {
	QueryReceivedCh chan string
}

func StartMockServer(opts MockServerOpts) (int, error) {
	// start an http server with 2 methods to
	// receive and log the requests

	listenPort := 3000 + rand.Intn(1000)

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
			opts.QueryReceivedCh <- "query"
			w.WriteHeader(http.StatusOK)
			return
		}

		fmt.Printf("Request: %s %s\n", r.Method, r.URL.Path)
		w.WriteHeader(http.StatusNotFound)
	}

	server := http.Server{
		Addr:    fmt.Sprintf(":%d", listenPort),
		Handler: http.HandlerFunc(handleRequest),
	}

	go func() {
		if err := server.ListenAndServe(); err != nil {
			log.Fatal(err)
		}
	}()

	return listenPort, nil
}

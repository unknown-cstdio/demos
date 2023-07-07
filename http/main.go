package main

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

const serverPort = 3333

func main() {
	go func() {
		mux := http.NewServeMux()
		mux.HandleFunc("/migration", func(w http.ResponseWriter, r *http.Request) {
			fmt.Printf("server: %s /\n", r.Method)
			//fmt.Printf("server: query id: %s\n", r.URL.Query().Get("id"))
			fmt.Printf("server: content-type: %s\n", r.Header.Get("content-type"))
			fmt.Printf("server: headers:\n")
			for headerName, headerValue := range r.Header {
				fmt.Printf("\t%s = %s\n", headerName, strings.Join(headerValue, ", "))
			}

			reqBody, err := io.ReadAll(r.Body)
			if err != nil {
				fmt.Printf("server: could not read request body: %s\n", err)
			}
			fmt.Printf("server: request body: %s\n", reqBody)
			fmt.Fprintf(w, `{"message": "hello!"}`)
		})
		server := http.Server{
			Addr:    fmt.Sprintf(":%d", serverPort),
			Handler: mux,
		}
		if err := server.ListenAndServe(); err != nil {
			if !errors.Is(err, http.ErrServerClosed) {
				fmt.Printf("error running http server: %s\n", err)
			}
		}
	}()

	time.Sleep(100 * time.Millisecond)

	jsonBody := []byte(`{"client_message": "hello, server!"}`)
	bodyReader := bytes.NewReader(jsonBody)

	requestURL := fmt.Sprintf("http://localhost:%d/migration", serverPort)
	req, err := http.NewRequest(http.MethodPost, requestURL, bodyReader)
	if err != nil {
		fmt.Printf("client: could not create request: %s\n", err)
		os.Exit(1)
	}

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		fmt.Printf("error making http request: %s\n", err)
		os.Exit(1)
	}

	fmt.Printf("client: got response!\n")
	fmt.Printf("client: status code: %d\n", res.StatusCode)

	resBody, err := io.ReadAll(res.Body)
	if err != nil {
		fmt.Printf("client: could not read response body: %s\n", err)
		os.Exit(1)
	}
	fmt.Printf("client: response body: %s\n", resBody)

}

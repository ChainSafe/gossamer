// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package pprof

import (
	"context"
	"io"
	"net/http"
	"testing"
	"time"

	"github.com/ChainSafe/gossamer/internal/httpserver"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_Server(t *testing.T) {
	t.Parallel()
	ctrl := gomock.NewController(t)

	const address = "127.0.0.1:0"
	logger := NewMockLogger(ctrl)

	logger.EXPECT().Info(newRegexMatcher("^pprof http server listening on 127.0.0.1:[1-9][0-9]{0,4}$"))
	logger.EXPECT().Warn("pprof http server shutting down: context canceled")

	const httpServerShutdownTimeout = 10 * time.Second // 10s in case test worker is slow
	server := NewServer(address, logger,
		httpserver.ShutdownTimeout(httpServerShutdownTimeout))
	require.NotNil(t, server)

	ctx, cancel := context.WithCancel(context.Background())
	ready := make(chan struct{})
	done := make(chan error)

	go server.Run(ctx, ready, done)

	select {
	case <-ready:
	case err := <-done:
		t.Fatalf("server crashed before being ready: %s", err)
	}

	serverAddress := server.GetAddress()

	const clientTimeout = 2 * time.Second
	httpClient := &http.Client{Timeout: clientTimeout}

	pathsToCheck := []string{
		"debug/pprof/",
		"debug/pprof/cmdline",
		"debug/pprof/profile?seconds=1",
		"debug/pprof/symbol",
		"debug/pprof/trace?seconds=0",
		"debug/pprof/block",
		"debug/pprof/goroutine",
		"debug/pprof/heap",
		"debug/pprof/threadcreate",
	}

	type httpResult struct {
		url    string
		status int
		body   []byte
		err    error
	}
	results := make(chan httpResult)

	for _, pathToCheck := range pathsToCheck {
		url := "http://" + serverAddress + "/" + pathToCheck

		request, err := http.NewRequestWithContext(ctx, http.MethodGet, url, http.NoBody)
		require.NoError(t, err)

		go func(client *http.Client, request *http.Request, results chan<- httpResult) {
			result := httpResult{
				url: request.URL.String(),
			}
			var response *http.Response
			response, result.err = client.Do(request)
			if result.err != nil {
				results <- result
				return
			}

			result.status = response.StatusCode
			result.body, result.err = io.ReadAll(response.Body)
			if result.err != nil {
				_ = response.Body.Close()
				results <- result
				return
			}

			result.err = response.Body.Close()
			results <- result
		}(httpClient, request, results)
	}

	for range pathsToCheck {
		httpResult := <-results

		require.NoErrorf(t, httpResult.err, "unexpected error for URL %s: %s", httpResult.url, httpResult.err)
		assert.Equalf(t, http.StatusOK, httpResult.status,
			"unexpected status code for URL %s: %s", httpResult.url, http.StatusText(httpResult.status))

		assert.NotEmptyf(t, httpResult.body, "response body is empty for URL %s", httpResult.url)
	}

	cancel()
	<-done
}

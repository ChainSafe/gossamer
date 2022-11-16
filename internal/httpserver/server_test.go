// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package httpserver

import (
	"net/http"
	"regexp"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_New(t *testing.T) {
	t.Parallel()

	const address = "test"
	handler := http.NewServeMux()

	expectedServer := &Server{
		settings: optionalSettings{
			handler:           handler,
			address:           address,
			logger:            &noopLogger{},
			readHeaderTimeout: time.Minute,
			readTimeout:       time.Hour,
			shutdownTimeout:   time.Second,
		},
	}

	server := New(
		Handler(handler),
		Address(address),
		ShutdownTimeout(time.Second),
		ReadTimeout(time.Hour),
		ReadHeaderTimeout(time.Minute),
	)

	assert.NotNil(t, server.addressSet)
	server.addressSet = nil
	assert.NotNil(t, server.stopping)
	server.stopping = nil

	assert.Equal(t, expectedServer, server)
}

func Test_Server_success(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)

	logger := NewMockInfoer(ctrl)
	logger.EXPECT().Info(newRegexMatcher("^test HTTP server listening on 127.0.0.1:[1-9][0-9]{0,4}$"))

	server := &Server{
		settings: optionalSettings{
			address:         "127.0.0.1:0",
			serverName:      "test",
			shutdownTimeout: 10 * time.Second,
			logger:          logger,
		},
		addressSet: make(chan struct{}),
		stopping:   make(chan struct{}),
	}

	runtimeError, err := server.Start()
	require.NoError(t, err)

	addressRegex := regexp.MustCompile(`^127.0.0.1:[1-9][0-9]{0,4}$`)
	address := server.GetAddress()
	assert.Regexp(t, addressRegex, address)

	select {
	case err := <-runtimeError:
		require.NoError(t, err)
	default:
	}

	err = server.Stop()
	require.NoError(t, err)

	_, ok := <-runtimeError
	assert.False(t, ok)
}

func Test_Server_startError(t *testing.T) {
	t.Parallel()

	server := &Server{
		settings: optionalSettings{
			address:         "127.0.0.1:-1",
			shutdownTimeout: 10 * time.Second,
		},
	}

	runtimeError, err := server.Start()

	require.EqualError(t, err, "listen tcp: address -1: invalid port")
	assert.Nil(t, runtimeError)
}

// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package httpserver

import (
	"context"
	"regexp"
	"testing"

	gomock "github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
)

func Test_Server_Run_success(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)

	logger := NewMockLogger(ctrl)
	logger.EXPECT().Info(newRegexMatcher("^test http server listening on 127.0.0.1:[1-9][0-9]{0,4}$"))
	logger.EXPECT().Warn("test http server shutting down: context canceled")

	server := &Server{
		name:       "test",
		address:    "127.0.0.1:0",
		addressSet: make(chan struct{}),
		logger:     logger,
	}

	ctx, cancel := context.WithCancel(context.Background())
	ready := make(chan struct{})
	done := make(chan error)

	go server.Run(ctx, ready, done)

	addressRegex := regexp.MustCompile(`^127.0.0.1:[1-9][0-9]{0,4}$`)
	address := server.GetAddress()
	assert.Regexp(t, addressRegex, address)
	address = server.GetAddress()
	assert.Regexp(t, addressRegex, address)

	<-ready

	cancel()
	err := <-done
	assert.NoError(t, err)
}

func Test_Server_Run_failure(t *testing.T) {
	t.Parallel()
	ctrl := gomock.NewController(t)

	logger := NewMockLogger(ctrl)

	server := &Server{
		name:       "test",
		address:    "127.0.0.1:-1",
		addressSet: make(chan struct{}),
		logger:     logger,
	}

	ready := make(chan struct{})
	done := make(chan error)

	go server.Run(context.Background(), ready, done)

	select {
	case <-ready:
		t.Fatal("server should not be ready")
	case err := <-done:
		assert.EqualError(t, err, "listen tcp: address -1: invalid port")
	}
}

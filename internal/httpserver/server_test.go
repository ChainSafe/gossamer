// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package httpserver

import (
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

func Test_New(t *testing.T) {
	t.Parallel()
	ctrl := gomock.NewController(t)

	const name = "name"
	const address = "test"
	handler := http.NewServeMux()
	logger := NewMockLogger(ctrl)

	expectedServer := &Server{
		name:    name,
		address: address,
		handler: handler,
		logger:  logger,
		optional: optionalSettings{
			readHeaderTimeout: time.Minute,
			readTimeout:       time.Hour,
			shutdownTimeout:   time.Second,
		},
	}

	server := New(name, address, handler, logger,
		ShutdownTimeout(time.Second),
		ReadTimeout(time.Hour),
		ReadHeaderTimeout(time.Minute),
	)

	assert.NotNil(t, server.addressSet)
	server.addressSet = nil

	assert.Equal(t, expectedServer, server)
}

// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package httpserver

import (
	"net/http"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
)

//go:generate mockgen -destination=logger_mock_test.go -package $GOPACKAGE . Logger

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
	}

	server := New(name, address, handler, logger)

	assert.NotNil(t, server.addressSet)
	server.addressSet = nil

	assert.Equal(t, expectedServer, server)
}

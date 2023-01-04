// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

// Package httpserver implements an HTTP server.
package httpserver

import (
	"net/http"
)

// Server is an HTTP server implementation, which uses
// the HTTP handler provided.
type Server struct {
	name       string
	address    string
	addressSet chan struct{}
	handler    http.Handler
	logger     Logger
	optional   optionalSettings
}

// New creates a new HTTP server with a name, listening on
// the address specified and using the HTTP handler provided.
func New(name, address string, handler http.Handler,
	logger Logger, options ...Option) *Server {
	return &Server{
		name:       name,
		address:    address,
		addressSet: make(chan struct{}),
		handler:    handler,
		logger:     logger,
		optional:   newOptionalSettings(options),
	}
}

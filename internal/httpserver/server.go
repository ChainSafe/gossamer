// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

// Package httpserver implements an HTTP server.
package httpserver

import (
	"context"
	"net/http"
)

var _ Interface = (*Server)(nil)

// Interface is the HTTP server interface
type Interface interface {
	Runner
	AddressGetter
}

// Runner is the interface for an HTTP server with a Run method.
type Runner interface {
	Run(ctx context.Context, ready chan<- struct{}, done chan<- error)
}

// AddressGetter obtains the address the HTTP server is listening on.
type AddressGetter interface {
	GetAddress() (address string)
}

// Server is an HTTP server implementation, which uses
// the HTTP handler provided.
type Server struct {
	name       string
	address    string
	addressSet chan struct{}
	handler    http.Handler
	logger     Logger
}

// New creates a new HTTP server with a name, listening on
// the address specified and using the HTTP handler provided.
func New(name, address string, handler http.Handler,
	logger Logger) *Server {
	return &Server{
		name:       name,
		address:    address,
		addressSet: make(chan struct{}),
		handler:    handler,
		logger:     logger,
	}
}

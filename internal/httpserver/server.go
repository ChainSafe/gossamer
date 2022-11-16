// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

// Package httpserver implements an HTTP server.
package httpserver

import (
	"context"
	"net"
	"net/http"
)

// Server is an HTTP server implementation, which uses
// the HTTP handler provided.
type Server struct {
	// server is initialised by the Start method. It is shared through
	// the Server struct so the `Start`` and `Stop` methods can access it.
	server     http.Server
	settings   optionalSettings
	addressSet chan struct{}
	stopping   chan struct{}
}

// New creates a new HTTP server with a name, listening on
// the address specified and using the HTTP handler provided.
func New(options ...Option) *Server {
	settings := newOptionalSettings(options)

	return &Server{
		settings:   settings,
		addressSet: make(chan struct{}),
		stopping:   make(chan struct{}),
	}
}

// Start starts the HTTP server and returns a read error channel
// and an eventual error.
// It is not necessary for the caller to call Stop on the server
// if an error is received in the error channel.
func (s *Server) Start() (runtimeError <-chan error, err error) {
	runtimeErrorCh := make(chan error)

	listener, err := net.Listen("tcp", s.settings.address)
	if err != nil {
		return nil, err
	}

	s.server = http.Server{
		Addr:              listener.Addr().String(),
		Handler:           s.settings.handler,
		ReadHeaderTimeout: s.settings.readHeaderTimeout,
		ReadTimeout:       s.settings.readTimeout,
	}
	close(s.addressSet)

	s.settings.logger.Info(s.settings.serverName + " HTTP server listening on " + s.server.Addr)

	go func() {
		err := s.server.Serve(listener)

		select {
		case <-s.stopping:
			close(runtimeErrorCh)
			return
		default:
			runtimeErrorCh <- err
		}
	}()

	return runtimeErrorCh, nil
}

// Stop stops the server within the given shutdown timeout.
func (s *Server) Stop() (err error) {
	shutdownCtx, cancel := context.WithTimeout(context.Background(),
		s.settings.shutdownTimeout)
	defer cancel()
	close(s.stopping)
	return s.server.Shutdown(shutdownCtx)
}

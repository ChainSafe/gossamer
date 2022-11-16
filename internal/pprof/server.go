// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package pprof

import (
	"runtime"

	"github.com/ChainSafe/gossamer/internal/httpserver"
)

// Server is a pprof http server service compatible with the
// dot/service.go interface.
type Server struct {
	settings Settings
	server   *httpserver.Server
}

// New creates a pprof server.
func New(settings Settings) *Server {
	settings.setDefaults()
	return &Server{
		settings: settings,
	}
}

// Start starts the pprof server.
// TODO return a runtimeError channel once services can read runtime
// errors.
func (s *Server) Start() (err error) {
	runtime.SetBlockProfileRate(s.settings.BlockProfileRate)
	runtime.SetMutexProfileFraction(s.settings.MutexProfileRate)

	handler := newHandler()
	s.server = httpserver.New(
		httpserver.Address(s.settings.ListeningAddress),
		httpserver.Handler(handler),
		httpserver.Logger("pprof", s.settings.Logger),
	)

	_, err = s.server.Start()
	return err
}

// Stop stops the pprof server.
func (s *Server) Stop() (err error) {
	runtime.SetBlockProfileRate(0)
	runtime.SetMutexProfileFraction(0)
	return s.server.Stop()
}

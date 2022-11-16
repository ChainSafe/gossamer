// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package httpserver

import (
	"net/http"
	"time"
)

// Option is a functional option for the HTTP server.
type Option func(s *optionalSettings)

type optionalSettings struct {
	handler           http.Handler
	address           string
	serverName        string
	logger            Infoer
	readTimeout       time.Duration
	readHeaderTimeout time.Duration
	shutdownTimeout   time.Duration
}

func newOptionalSettings(options []Option) (settings optionalSettings) {
	for _, option := range options {
		option(&settings)
	}

	if settings.handler == nil {
		settings.handler = http.NewServeMux()
	}

	if settings.logger == nil {
		settings.logger = &noopLogger{}
	}

	if settings.readTimeout == 0 {
		const defaultReadTimeout = 10 * time.Second
		settings.readTimeout = defaultReadTimeout
	}

	if settings.readHeaderTimeout == 0 {
		const defaultReadHeaderTimeout = time.Second
		settings.readHeaderTimeout = defaultReadHeaderTimeout
	}

	if settings.shutdownTimeout == 0 {
		const defaultShutdownTimeout = 3 * time.Second
		settings.shutdownTimeout = defaultShutdownTimeout
	}

	return settings
}

// Handler sets the http handler to use for the HTTP server.
// It defaults to an empty mux created with `http.NewServeMux()`.
func Handler(handler http.Handler) Option {
	return func(s *optionalSettings) {
		s.handler = handler
	}
}

// Address sets the listening address for the HTTP server.
// The default is the empty address which means any available
// address is assigned by the OS.
func Address(address string) Option {
	return func(s *optionalSettings) {
		s.address = address
	}
}

// Infoer logs information messages at the info level.
type Infoer interface {
	Info(message string)
}

// Logger sets the logger to use for the HTTP server,
// together with a server name to use in the logs.
// It defaults to a no-op logger.
func Logger(serverName string, logger Infoer) Option {
	return func(s *optionalSettings) {
		s.serverName = serverName
		s.logger = logger
	}
}

// ReadTimeout sets the read timeout for the HTTP server.
// The default timeout is 10 seconds.
func ReadTimeout(timeout time.Duration) Option {
	return func(s *optionalSettings) {
		s.readTimeout = timeout
	}
}

// ReadHeaderTimeout sets the header read timeout
// for the HTTP server. The default timeout is 1 second.
func ReadHeaderTimeout(timeout time.Duration) Option {
	return func(s *optionalSettings) {
		s.readHeaderTimeout = timeout
	}
}

// ShutdownTimeout sets an optional timeout for the HTTP server
// to shutdown. The default shutdown is 3 seconds.
func ShutdownTimeout(timeout time.Duration) Option {
	return func(s *optionalSettings) {
		s.shutdownTimeout = timeout
	}
}

type noopLogger struct{}

func (noopLogger) Info(_ string) {}

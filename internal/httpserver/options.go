// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package httpserver

import "time"

// Option is a functional option for the HTTP server.
type Option func(s *optionalSettings)

type optionalSettings struct {
	readTimeout       time.Duration
	readHeaderTimeout time.Duration
	shutdownTimeout   time.Duration
}

func newOptionalSettings(options []Option) (settings optionalSettings) {
	for _, option := range options {
		option(&settings)
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

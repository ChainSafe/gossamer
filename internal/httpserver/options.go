// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package httpserver

import "time"

// Option is a functional option for the HTTP server.
type Option func(s *optionalSettings)

type optionalSettings struct {
	shutdownTimeout time.Duration
}

func newOptionalSettings(options []Option) (settings optionalSettings) {
	for _, option := range options {
		option(&settings)
	}

	if settings.shutdownTimeout == 0 {
		const defaultShutdownTimeout = 3 * time.Second
		settings.shutdownTimeout = defaultShutdownTimeout
	}

	return settings
}

// ShutdownTimeout sets an optional timeout for the HTTP server
// to shutdown. The default shutdown is 3 seconds.
func ShutdownTimeout(timeout time.Duration) Option {
	return func(s *optionalSettings) {
		s.shutdownTimeout = timeout
	}
}

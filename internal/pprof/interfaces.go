// Copyright 2022 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package pprof

import "context"

// Logger for the pprof http server.
type Logger interface {
	Info(msg string)
	Warn(msg string)
	Error(msg string)
}

// Runner runs in a blocking manner.
type Runner interface {
	Run(ctx context.Context, ready chan<- struct{}, done chan<- error)
}

// Copyright 2022 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package pprof

// Logger for the pprof http server.
type Logger interface {
	Info(msg string)
	Warn(msg string)
	Error(msg string)
}

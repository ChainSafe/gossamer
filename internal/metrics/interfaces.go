// Copyright 2022 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package metrics

// Logger logs formatted strings at the different log levels.
type Logger interface {
	Info(s string)
	Warn(s string)
	Error(s string)
	Infof(format string, args ...interface{})
}

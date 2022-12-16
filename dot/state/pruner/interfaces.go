// Copyright 2022 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package pruner

// Logger logs formatted strings at the different log levels.
type Logger interface {
	Debug(s string)
	Debugf(format string, args ...interface{})
	Warnf(format string, args ...interface{})
}

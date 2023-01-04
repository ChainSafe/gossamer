// Copyright 2022 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package main

// Logger is the logger to log messages.
type Logger interface {
	Trace(s string)
	Debug(s string)
	Info(s string)
	Warn(s string)
	Critical(s string)
	Debugf(format string, args ...interface{})
	Infof(format string, args ...interface{})
	Errorf(format string, args ...interface{})
}

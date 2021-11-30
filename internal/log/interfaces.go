// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package log

// ChildConstructor is the interface to create child loggers.
type ChildConstructor interface {
	New(options ...Option) Logger
}

// LoggerPatcher is the interface to update the current logger.
type LoggerPatcher interface {
	Patch(options ...Option)
}

// LeveledLogger is the interface to log at different levels.
type LeveledLogger interface {
	Trace(s string)
	Debug(s string)
	Info(s string)
	Warn(s string)
	Error(s string)
	Critical(s string)
	Tracef(format string, args ...interface{})
	Debugf(format string, args ...interface{})
	Infof(format string, args ...interface{})
	Warnf(format string, args ...interface{})
	Errorf(format string, args ...interface{})
	Criticalf(format string, args ...interface{})
}

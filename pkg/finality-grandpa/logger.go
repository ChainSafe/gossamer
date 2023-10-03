// Copyright 2023 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package grandpa

type logLevel int

const (
	debug logLevel = iota
	trace
)

type Logger interface {
	Warn(l string)
	Warnf(format string, values ...any)

	Info(l string)
	Infof(format string, values ...any)

	Debug(l string)
	Debugf(format string, values ...any)

	Trace(l string)
	Tracef(format string, values ...any)
}

type noopLogger struct{}

func (nl noopLogger) Warn(l string) {
}

func (nl noopLogger) Warnf(format string, values ...any) {
}

func (nl noopLogger) Info(l string) {
}

func (nl noopLogger) Infof(format string, values ...any) {
}

func (nl noopLogger) Debug(l string) {
}

func (nl noopLogger) Debugf(format string, values ...any) {
}

func (nl noopLogger) Trace(l string) {
}

func (nl noopLogger) Tracef(format string, values ...any) {
}

var log Logger

func init() {
	log = noopLogger{}
}

func SetLogger(l Logger) {
	log = l
}

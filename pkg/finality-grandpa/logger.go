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

func (noopLogger) Warn(_ string) {
}

func (noopLogger) Warnf(_ string, _ ...any) {
}

func (noopLogger) Info(_ string) {
}

func (noopLogger) Infof(_ string, _ ...any) {
}

func (noopLogger) Debug(_ string) {
}

func (noopLogger) Debugf(_ string, _ ...any) {
}

func (noopLogger) Trace(_ string) {
}

func (noopLogger) Tracef(_ string, _ ...any) {
}

var log Logger

func init() {
	log = noopLogger{}
}

func SetLogger(l Logger) {
	log = l
}

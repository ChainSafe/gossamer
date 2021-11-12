// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package log

import (
	"io"
)

// Option is the type to specify settings modifier
// for the logger operation.
type Option func(s *settings)

// SetLevel sets the level for the logger.
// The level defaults to the lowest level, trce.
func SetLevel(level Level) Option {
	return func(s *settings) {
		s.level = &level
	}
}

// SetCallerFile enables or disables logging the caller file.
// The default is disabled.
func SetCallerFile(enabled bool) Option {
	return func(s *settings) {
		s.caller.file = &enabled
	}
}

// SetCallerLine enables or disables logging the caller line number.
// The default is disabled.
func SetCallerLine(enabled bool) Option {
	return func(s *settings) {
		s.caller.line = &enabled
	}
}

// SetCallerFunc enables or disables logging the caller function.
// The default is disabled.
func SetCallerFunc(enabled bool) Option {
	return func(s *settings) {
		s.caller.funC = &enabled
	}
}

// SetFormat set the format for the logger.
// The format defaults to FormatConsole.
func SetFormat(format Format) Option {
	return func(s *settings) {
		s.format = &format
	}
}

// SetWriter set the writer for the logger.
// The writer defaults to os.Stdout.
func SetWriter(writer io.Writer) Option {
	return func(s *settings) {
		s.writer = writer
	}
}

// AddContext adds the context for the logger as a key values pair.
// It adds them in order. If a key already exists, the value is added to the
// existing values.
func AddContext(key, value string) Option {
	return func(s *settings) {
		for i := range s.context {
			if s.context[i].key == key {
				s.context[i].values = append(s.context[i].values, value)
				return
			}
		}
		newKV := contextKeyValues{key: key, values: []string{value}}
		s.context = append(s.context, newKV)
	}
}

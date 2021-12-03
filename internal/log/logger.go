// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package log

import (
	"sync"
)

// Logger is the logger implementation structure.
// It is thread safe to use.
type Logger struct {
	settings settings
	mutex    *sync.Mutex // pointer for child loggers
	childs   []*Logger   // TODO-1946 remove this field
}

// New creates a new logger.
// It can only be called once per writer.
// If you want to create more loggers with different settings for the
// same writer, child loggers can be created using the New(options) method,
// to ensure thread safety on the same writer.
func New(options ...Option) *Logger {
	s := newSettings(options)
	s.setDefaults()

	return &Logger{
		settings: s,
		mutex:    new(sync.Mutex),
	}
}

// New creates a new thread safe child logger.
// It can use a different writer, but it is expected to use the
// same writer since it is thread safe.
func (l *Logger) New(options ...Option) *Logger {
	var childSettings settings
	childSettings.mergeWith(l.settings)
	childSettings.mergeWith(newSettings(options))
	// defaults are already set in parent

	newLogger := &Logger{
		settings: childSettings,
		mutex:    l.mutex,
	}

	l.childs = append(l.childs, newLogger)

	return newLogger
}

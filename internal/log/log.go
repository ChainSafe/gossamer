// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package log

import (
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/fatih/color" //nolint:misspell
)

func (l *Logger) log(logLevel Level, s string, args ...interface{}) {
	l.mutex.Lock()
	defer l.mutex.Unlock()

	if *l.settings.level < logLevel {
		return
	}

	if len(args) > 0 {
		s = fmt.Sprintf(s, args...)
	}

	line := time.Now().Format(time.RFC3339) + " " + logLevel.ColouredString() + " " + s

	callerString := getCallerString(l.settings.caller)
	if callerString != "" {
		line += "\t" + color.HiWhiteString(callerString)
	}

	if len(l.settings.context) > 0 {
		keyValues := make([]string, 0, len(l.settings.context))
		for _, kvs := range l.settings.context {
			valuesString := strings.Join(kvs.values, ",")
			keyValue := color.CyanString(kvs.key) + "=" + valuesString
			keyValues = append(keyValues, keyValue)
		}
		line += "\t" + strings.Join(keyValues, " ")
	}

	line += "\n"

	_, _ = io.WriteString(l.settings.writer, line)
}

// Trace logs with the trce level.
func (l *Logger) Trace(s string) { l.log(Trace, s) }

// Debug logs with the dbug level.
func (l *Logger) Debug(s string) { l.log(Debug, s) }

// Info logs with the info level.
func (l *Logger) Info(s string) { l.log(Info, s) }

// Warn logs with the warn level.
func (l *Logger) Warn(s string) { l.log(Warn, s) }

// Error logs with the eror level.
func (l *Logger) Error(s string) { l.log(Error, s) }

// Critical logs with the crit level.
func (l *Logger) Critical(s string) { l.log(Critical, s) }

// Tracef formats and logs at the trce level.
func (l *Logger) Tracef(format string, args ...interface{}) {
	l.log(Trace, format, args...)
}

// Debugf formats and logs at the dbug level.
func (l *Logger) Debugf(format string, args ...interface{}) {
	l.log(Debug, format, args...)
}

// Infof formats and logs at the info level.
func (l *Logger) Infof(format string, args ...interface{}) {
	l.log(Info, format, args...)
}

// Warnf formats and logs at the warn level.
func (l *Logger) Warnf(format string, args ...interface{}) {
	l.log(Warn, format, args...)
}

// Errorf formats and logs at the eror level.
func (l *Logger) Errorf(format string, args ...interface{}) {
	l.log(Error, format, args...)
}

// Criticalf formats and logs at the crit level.
func (l *Logger) Criticalf(format string, args ...interface{}) {
	l.log(Critical, format, args...)
}

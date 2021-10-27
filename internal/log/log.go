package log

import (
	"io"
	"strings"
	"time"
)

func (l *Logger) log(logLevel Level, s string) {
	l.mutex.Lock()
	defer l.mutex.Unlock()

	if *l.settings.level > logLevel {
		return
	}

	line := time.Now().Format(time.RFC3339) + " " + logLevel.String() + " " + s

	callerString := getCallerString(l.settings.caller)
	if callerString != "" {
		line += "\t" + callerString
	}

	if len(l.settings.context) > 0 {
		keyValues := make([]string, 0, len(l.settings.context))
		for _, kvs := range l.settings.context {
			valuesString := strings.Join(kvs.values, ",")
			keyValue := kvs.key + "=" + valuesString
			keyValues = append(keyValues, keyValue)
		}
		line += "\t" + strings.Join(keyValues, " ")
	}

	line += "\n"

	_, _ = io.WriteString(l.settings.writer, line)
}

// Trace logs with the trce level.
func (l *Logger) Trace(s string) { l.log(LevelTrace, s) }

// Debug logs with the dbug level.
func (l *Logger) Debug(s string) { l.log(LevelDebug, s) }

// Info logs with the info level.
func (l *Logger) Info(s string) { l.log(LevelInfo, s) }

// Warn logs with the warn level.
func (l *Logger) Warn(s string) { l.log(LevelWarn, s) }

// Error logs with the eror level.
func (l *Logger) Error(s string) { l.log(LevelError, s) }

// Critical logs with the crit level.
func (l *Logger) Critical(s string) { l.log(LevelCritical, s) }

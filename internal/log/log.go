package log

import "strings"

func (l *Logger) log(logLevel Level, s string, options []LogOption) {
	l.mutex.Lock()
	defer l.mutex.Unlock()

	if l.settings.level > logLevel {
		return
	}

	logSettings := newLogSettings(options)

	line := logLevel.String() + " " + s
	if len(logSettings.context) > 0 {
		keyValues := make([]string, 0, len(logSettings.context))
		for k, v := range logSettings.context {
			keyValue := k + "=" + v
			keyValues = append(keyValues, keyValue)
		}
		line += "\t" + strings.Join(keyValues, " ")
	}

	const callDepth = 3
	_ = l.stdLogger.Output(callDepth, line)
}

// Trace logs with the trce level.
func (l *Logger) Trace(s string, options ...LogOption) { l.log(LevelTrace, s, options) }

// Debug logs with the dbug level.
func (l *Logger) Debug(s string, options ...LogOption) { l.log(LevelDebug, s, options) }

// Info logs with the info level.
func (l *Logger) Info(s string, options ...LogOption) { l.log(LevelInfo, s, options) }

// Warn logs with the warn level.
func (l *Logger) Warn(s string, options ...LogOption) { l.log(LevelWarn, s, options) }

// Error logs with the eror level.
func (l *Logger) Error(s string, options ...LogOption) { l.log(LevelError, s, options) }

// Critical logs with the crit level.
func (l *Logger) Critical(s string, options ...LogOption) { l.log(LevelCritical, s, options) }

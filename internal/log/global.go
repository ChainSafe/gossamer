package log

// TODO do not use a global logger.
var globalLogger = New(SetCaller(CallerShort))

// NewFromGlobal creates a child logger from the global logger.
func NewFromGlobal(options ...Option) *Logger {
	return globalLogger.New(options...)
}

// Patch patches the global package logger.
func Patch(options ...Option) {
	globalLogger.Patch(options...)
}

// PatchLevel patches the global package logger level.
func PatchLevel(level Level) {
	globalLogger.PatchLevel(level)
}

// Trace using the global logger.
func Trace(s string) {
	globalLogger.Trace(s)
}

// Debug using the global logger.
func Debug(s string) {
	globalLogger.Debug(s)
}

// Info using the global logger.
func Info(s string) {
	globalLogger.Info(s)
}

// Warn using the global logger.
func Warn(s string) {
	globalLogger.Warn(s)
}

// Error using the global logger.
func Error(s string) {
	globalLogger.Error(s)
}

// Critical using the global logger.
func Critical(s string) {
	globalLogger.Critical(s)
}

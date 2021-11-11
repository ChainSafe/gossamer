package log

// TODO do not use a global logger.
var globalLogger = New()

// NewFromGlobal creates a child logger from the global logger.
func NewFromGlobal(options ...Option) *Logger {
	return globalLogger.New(options...)
}

// Patch patches the global logger.
func Patch(options ...Option) {
	globalLogger.Patch(options...)
}

// Errorf using the global logger, only used in test
// main runners initialisation error.
func Errorf(s string, args ...interface{}) {
	globalLogger.Errorf(s, args...)
}

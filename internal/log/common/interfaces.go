package common

// Logger is the interface encompassing all the methods of the logger.
type Logger interface {
	ChildConstructor
	LoggerPatcher
	LeveledLogger
	LeveledFormatterLogger
}

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
}

// LeveledFormatterLogger is the interface to format and log at different levels.
type LeveledFormatterLogger interface {
	Tracef(format string, args ...interface{})
	Debugf(format string, args ...interface{})
	Infof(format string, args ...interface{})
	Warnf(format string, args ...interface{})
	Errorf(format string, args ...interface{})
	Criticalf(format string, args ...interface{})
}

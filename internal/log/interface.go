package log

var _ Interface = (*Logger)(nil)

// Interface is the interface encompassing all the methods of the logger.
type Interface interface {
	ChildConstructor
	LoggerUpdater
	LeveledLogger
}

// ChildConstructor is the interface to create child loggers.
type ChildConstructor interface {
	New(options ...Option) *Logger
}

// LoggerUpdater is the interface to update the current logger.
type LoggerUpdater interface {
	Patch(options ...Option)
	PatchLevel(level Level)
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

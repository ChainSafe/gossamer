package log

import (
	"log"
	"sync"
)

// Logger is the logger implementation structure.
// It is thread safe to use.
type Logger struct {
	stdLogger *log.Logger
	settings  settings
	mutex     *sync.Mutex // pointer for child loggers
}

// New creates a new logger based on the standard library logger.
// It should only be called once at most per writer (settings.Writer).
// If you want to create more loggers with different settings for the
// same writer, child loggers can be created using the NewChild method,
// to ensure thread safety on the same writer.
func New(options ...Option) *Logger {
	s := newSettings(options)
	s.setDefaults()

	flags := log.Ldate | log.Ltime
	if s.caller == CallerShort {
		flags |= log.Lshortfile
	}

	stdLogger := log.New(s.writer, "", flags)

	return &Logger{
		stdLogger: stdLogger,
		settings:  s,
		mutex:     new(sync.Mutex),
	}
}

// New creates a new thread safe child logger.
// It can use a different writer, but it is expected to use the
// same writer since it is thread safe.
func (l *Logger) New(options ...Option) *Logger {
	s := newSettings(options)
	s.mergeWith(l.settings)
	s.setDefaults()

	flags := log.Ldate | log.Ltime
	if s.caller == CallerShort {
		flags |= log.Lshortfile
	}

	stdLogger := log.New(l.stdLogger.Writer(), "", flags)

	return &Logger{
		stdLogger: stdLogger,
		settings:  s,
		mutex:     l.mutex,
	}
}

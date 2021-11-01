package production

import (
	"sync"

	"github.com/ChainSafe/gossamer/internal/log/common"
)

// Logger is the logger implementation structure.
// It is thread safe to use.
type Logger struct {
	settings settings
	mutex    *sync.Mutex // pointer for child loggers
}

// New creates a new logger.
// It can only be called once per writer.
// If you want to create more loggers with different settings for the
// same writer, child loggers can be created using the New(options) method,
// to ensure thread safety on the same writer.
func New(options ...common.Option) *Logger {
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
func (l *Logger) New(options ...common.Option) common.Logger {
	var childSettings settings
	childSettings.mergeWith(l.settings)
	childSettings.mergeWith(newSettings(options))
	// defaults are already set in parent

	return &Logger{
		settings: childSettings,
		mutex:    l.mutex,
	}
}

package log

import "io"

// Option is the option type to set or modify internal settings
// of the logger.
type Option func(s *settings)

// SetLevel sets the level for the logger.
// The level defaults to the lowest level, trce.
func SetLevel(level Level) Option {
	return func(s *settings) {
		s.level = level
	}
}

// SetCaller set the caller for the logger.
// The caller defaults to not show the caller (CallerHidden).
func SetCaller(caller Caller) Option {
	return func(s *settings) {
		s.caller = caller
	}
}

// SetFormat set the format for the logger.
// The format defaults to FormatConsole.
func SetFormat(format Format) Option {
	return func(s *settings) {
		s.format = format
	}
}

// SetWriter set the writer for the logger.
// The writer defaults to os.Stdout.
func SetWriter(writer io.Writer) Option {
	return func(s *settings) {
		s.writer = writer
	}
}

// AddContext adds the context for the logger as a key values pair.
// It adds them in order. If a key already exists, the value is added to the
// existing values.
func AddContext(key, value string) Option {
	return func(s *settings) {
		for i := range s.context {
			if s.context[i].key == key {
				s.context[i].values = append(s.context[i].values, value)
				return
			}
		}
		newKV := contextKeyValues{key: key, values: []string{value}}
		s.context = append(s.context, newKV)
	}
}

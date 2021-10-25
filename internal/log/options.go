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

// SetContext set the context for the logger as key value pairs.
func SetContext(kv map[string]string) Option {
	return func(s *settings) {
		for k, v := range kv {
			s.context[k] = v
		}
	}
}

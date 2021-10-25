package log

import (
	"io"
	"os"
)

type settings struct {
	level   Level
	caller  Caller
	format  Format
	context map[string]string
	writer  io.Writer
}

// newSettings returns settings using the options given.
func newSettings(options []Option) (settings settings) {
	settings.context = make(map[string]string)
	for _, option := range options {
		option(&settings)
	}
	return settings
}

func (s *settings) setDefaults() {
	if s.writer == nil {
		s.writer = os.Stdout
	}
}

// mergeWith sets empty values of s with the values from other.
// It also merges contexts together but does not override existing keys.
func (s *settings) mergeWith(other settings) {
	if s.level == 0 {
		s.level = other.level
	}
	if s.caller == 0 {
		s.caller = other.caller
	}
	if s.format == 0 {
		s.format = other.format
	}
	if s.writer == nil {
		s.writer = other.writer
	}
	if s.context == nil {
		s.context = other.context
	} else {
		for k, v := range other.context {
			if _, ok := s.context[k]; !ok {
				s.context[k] = v
			}
		}
	}
}

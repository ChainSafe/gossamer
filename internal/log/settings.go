package log

import (
	"io"
	"os"
)

type settings struct {
	level   Level
	caller  Caller
	format  Format
	context []contextKeyValues
	writer  io.Writer
}

type contextKeyValues struct {
	key    string
	values []string
}

// newSettings returns settings using the options given.
func newSettings(options []Option) (settings settings) {
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

	existingKeyToIndex := make(map[string]int, len(s.context))
	for i, kvs := range s.context {
		existingKeyToIndex[kvs.key] = i
	}
	for _, kvs := range other.context {
		i, ok := existingKeyToIndex[kvs.key]
		if ok {
			s.context[i].values = append(s.context[i].values, kvs.values...)
			continue
		}
		kvsCopy := contextKeyValues{
			key:    kvs.key,
			values: make([]string, len(kvs.values)),
		}
		copy(kvsCopy.values, kvs.values)
		s.context = append(s.context, kvsCopy)
	}
}

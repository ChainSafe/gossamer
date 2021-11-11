package log

import (
	"io"
	"os"
)

type settings struct {
	writer  io.Writer
	level   *Level
	format  *Format
	caller  callerSettings
	context []contextKeyValues
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

	if s.level == nil {
		value := Info
		s.level = &value
	}

	if s.format == nil {
		value := FormatConsole
		s.format = &value
	}

	s.caller.setDefaults()
}

// mergeWith sets values to s for all values that are set in other.
// It also merges contexts together without overriding existing keys.
func (s *settings) mergeWith(other settings) {
	if other.writer != nil { // use other's writer
		s.writer = other.writer
	}

	if other.level != nil {
		value := *other.level
		s.level = &value
	}

	if other.format != nil {
		value := *other.format
		s.format = &value
	}

	s.caller.mergeWith(other.caller)

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

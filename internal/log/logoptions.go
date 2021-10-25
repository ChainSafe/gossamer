package log

// LogOption is the option type to set or modify log settings
// for one particular log action.
type LogOption func(s *logSettings) //nolint:revive

// WithKeyValue sets an additional context key value for the log operation.
func WithKeyValue(key, value string) LogOption {
	return func(s *logSettings) {
		if s.context == nil {
			s.context = map[string]string{key: value}
		} else {
			s.context[key] = value
		}
	}
}

type logSettings struct {
	context map[string]string
}

func newLogSettings(options []LogOption) *logSettings {
	settings := &logSettings{
		context: make(map[string]string),
	}
	for _, option := range options {
		option(settings)
	}
	return settings
}

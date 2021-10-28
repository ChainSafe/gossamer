package production

import (
	"io"

	"github.com/ChainSafe/gossamer/internal/log/common"
)

// SetLevel sets the level for the logger.
// The level defaults to the lowest level, trce.
func SetLevel(level Level) common.Option {
	return func(sIntf interface{}) {
		s := sIntf.(*settings)
		s.level = &level
	}
}

// SetCallerFile enables or disables logging the caller file.
// The default is disabled.
func SetCallerFile(enabled bool) common.Option {
	return func(sIntf interface{}) {
		s := sIntf.(*settings)
		s.caller.file = &enabled
	}
}

// SetCallerLine enables or disables logging the caller line number.
// The default is disabled.
func SetCallerLine(enabled bool) common.Option {
	return func(sIntf interface{}) {
		s := sIntf.(*settings)
		s.caller.line = &enabled
	}
}

// SetCallerFunc enables or disables logging the caller function.
// The default is disabled.
func SetCallerFunc(enabled bool) common.Option {
	return func(sIntf interface{}) {
		s := sIntf.(*settings)
		s.caller.funC = &enabled
	}
}

// SetFormat set the format for the logger.
// The format defaults to FormatConsole.
func SetFormat(format Format) common.Option {
	return func(sIntf interface{}) {
		s := sIntf.(*settings)
		s.format = &format
	}
}

// SetWriter set the writer for the logger.
// The writer defaults to os.Stdout.
func SetWriter(writer io.Writer) common.Option {
	return func(sIntf interface{}) {
		s := sIntf.(*settings)
		s.writer = writer
	}
}

// AddContext adds the context for the logger as a key values pair.
// It adds them in order. If a key already exists, the value is added to the
// existing values.
func AddContext(key, value string) common.Option {
	return func(sIntf interface{}) {
		s := sIntf.(*settings)
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

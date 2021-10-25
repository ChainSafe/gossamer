package log

import (
	"errors"
	"fmt"
	"strings"
)

// Level is the level of the logger.
type Level uint8

const (
	// LevelTrace is the trace (trce) level.
	LevelTrace Level = iota
	// LevelDebug is the debug (dbug) level.
	LevelDebug
	// LevelInfo is the info level.
	LevelInfo
	// LevelWarn is the warn level.
	LevelWarn
	// LevelError is the error (eror) level.
	LevelError
	// LevelCritical is the cirtical (crit) level.
	LevelCritical
)

func (level Level) String() (s string) {
	switch level {
	case LevelTrace:
		return "TRCE"
	case LevelDebug:
		return "DBUG"
	case LevelInfo:
		return "INFO"
	case LevelWarn:
		return "WARN"
	case LevelError:
		return "EROR"
	case LevelCritical:
		return "CRIT"
	}
	return string(level)
}

// ErrLevelNotRecognised is an error returned if the level string is
// not recognised by the ParseLevel function.
var ErrLevelNotRecognised = errors.New("level is not recognised")

// ParseLevel parses a string into a level, and returns an
// error if it fails.
func ParseLevel(s string) (level Level, err error) {
	switch strings.ToUpper(s) {
	case LevelTrace.String():
		return LevelTrace, nil
	case LevelDebug.String():
		return LevelDebug, nil
	case LevelInfo.String():
		return LevelInfo, nil
	case LevelWarn.String():
		return LevelWarn, nil
	case LevelError.String():
		return LevelError, nil
	case LevelCritical.String():
		return LevelCritical, nil
	}
	return 0, fmt.Errorf("%w: %s", ErrLevelNotRecognised, s)
}

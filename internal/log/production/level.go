package production

import (
	"errors"
	"fmt"
	"strings"

	"github.com/fatih/color" //nolint:misspell
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
	// LevelDoNotChange indicates the level of the logger should be
	// left as is.
	LevelDoNotChange Level = Level(^uint8(0))
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
	default:
		return "???"
	}
}

// ColouredString returns the corresponding coloured
// string for the level.
func (level Level) ColouredString() (s string) {
	attribute := color.Reset

	switch level {
	case LevelTrace:
		attribute = color.FgHiCyan
	case LevelDebug:
		attribute = color.FgHiBlue
	case LevelInfo:
		attribute = color.FgCyan
	case LevelWarn:
		attribute = color.FgYellow
	case LevelError:
		attribute = color.FgHiRed
	case LevelCritical:
		attribute = color.FgRed
	}

	c := color.New(attribute)
	return c.Sprint(level.String())
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

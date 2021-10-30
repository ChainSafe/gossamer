package log

import (
	"errors"
	"fmt"
	"strings"

	"github.com/fatih/color" //nolint:misspell
)

// Level is the level of the logger.
type Level uint8

const (
	// Trace is the trace (trce) level.
	Trace Level = iota
	// Debug is the debug (dbug) level.
	Debug
	// Info is the info level.
	Info
	// Warn is the warn level.
	Warn
	// Error is the error (eror) level.
	Error
	// Critical is the cirtical (crit) level.
	Critical
	// DoNotChange indicates the level of the logger should be
	// left as is.
	DoNotChange Level = Level(^uint8(0))
)

func (level Level) String() (s string) {
	switch level {
	case Trace:
		return "TRCE"
	case Debug:
		return "DBUG"
	case Info:
		return "INFO"
	case Warn:
		return "WARN"
	case Error:
		return "EROR"
	case Critical:
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
	case Trace:
		attribute = color.FgHiCyan
	case Debug:
		attribute = color.FgHiBlue
	case Info:
		attribute = color.FgCyan
	case Warn:
		attribute = color.FgYellow
	case Error:
		attribute = color.FgHiRed
	case Critical:
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
	case Trace.String():
		return Trace, nil
	case Debug.String():
		return Debug, nil
	case Info.String():
		return Info, nil
	case Warn.String():
		return Warn, nil
	case Error.String():
		return Error, nil
	case Critical.String():
		return Critical, nil
	}
	return 0, fmt.Errorf("%w: %s", ErrLevelNotRecognised, s)
}

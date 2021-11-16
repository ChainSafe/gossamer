// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package log

import (
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/fatih/color" //nolint:misspell
)

// Level is the level of the logger.
type Level uint8

const (
	// Critical is the cirtical (crit) level.
	Critical Level = iota
	// Error is the error (eror) level.
	Error
	// Warn is the warn level.
	Warn
	// Info is the info level.
	Info
	// Debug is the debug (dbug) level.
	Debug
	// Trace is the trace (trce) level.
	Trace
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

var (
	ErrLevelNotRecognised     = errors.New("level is not recognised")
	ErrLevelIntegerOutOfRange = errors.New("level integer can only be between 0 and 5 included")
)

// ParseLevel parses a string into a level, and returns an
// error if it fails. It accepts integers between 0 (critical)
// and 5 (trace) as well as strings such as 'trace' or 'dbug'.
func ParseLevel(s string) (level Level, err error) {
	n, err := strconv.Atoi(s)
	if err == nil { // level given as an integer
		if n < 0 || n > 5 {
			return 0, fmt.Errorf("%w: %d", ErrLevelIntegerOutOfRange, n)
		}
		return Level(n), nil
	}

	switch strings.ToUpper(s) {
	case Trace.String(), "TRACE":
		return Trace, nil
	case Debug.String(), "DEBUG":
		return Debug, nil
	case Info.String():
		return Info, nil
	case Warn.String():
		return Warn, nil
	case Error.String(), "ERROR":
		return Error, nil
	case Critical.String(), "CRITICAL":
		return Critical, nil
	}
	return 0, fmt.Errorf("%w: %s", ErrLevelNotRecognised, s)
}

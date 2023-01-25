// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package log

import (
	"bytes"
	"regexp"
	"strings"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_Logger_log(t *testing.T) {
	t.Parallel()

	testCases := map[string]struct {
		logger      *Logger
		level       Level
		s           string
		args        []interface{}
		outputRegex string
	}{
		"log_at_trace": {
			logger: &Logger{
				settings: settings{
					level:  levelPtr(Trace),
					caller: newCallerSettings(false, false, false),
				},
				mutex: new(sync.Mutex),
			},
			level:       Trace,
			s:           "some words",
			outputRegex: timePrefixRegex + "TRACE    some words\n$",
		},
		"do_not_log_at_trace": {
			logger: &Logger{
				settings: settings{
					level:  levelPtr(Debug),
					caller: newCallerSettings(false, false, false),
				},
				mutex: new(sync.Mutex),
			},
			level:       Trace,
			s:           "some words",
			outputRegex: "^$",
		},
		"log_at_debug_with_trace_set": {
			logger: &Logger{
				settings: settings{
					level:  levelPtr(Trace),
					caller: newCallerSettings(false, false, false),
				},
				mutex: new(sync.Mutex),
			},
			level:       Debug,
			s:           "some words",
			outputRegex: timePrefixRegex + "DEBUG    some words\n$",
		},
		"format_string": {
			logger: &Logger{
				settings: settings{
					level:  levelPtr(Trace),
					caller: newCallerSettings(false, false, false),
				},
				mutex: new(sync.Mutex),
			},
			level:       Trace,
			s:           "some %s",
			args:        []interface{}{"words"},
			outputRegex: timePrefixRegex + "TRACE    some words\n$",
		},
		"show_caller": {
			logger: &Logger{
				settings: settings{
					level:  levelPtr(Trace),
					caller: newCallerSettings(true, true, true),
				},
				mutex: new(sync.Mutex),
			},
			level:       Trace,
			s:           "some words",
			outputRegex: timePrefixRegex + "TRACE    some words\tlog_test.go:L[0-9]+:func[0-9]+\n$",
		},
		"context": {
			logger: &Logger{
				settings: settings{
					level:  levelPtr(Trace),
					caller: newCallerSettings(false, false, false),
					context: []contextKeyValues{
						{key: "key1", values: []string{"a", "b"}},
						{key: "key2", values: []string{"c", "d"}},
					},
				},
				mutex: new(sync.Mutex),
			},
			level:       Trace,
			s:           "some words",
			outputRegex: timePrefixRegex + "TRACE    some words\tkey1=a,b key2=c,d\n$",
		},
	}

	for name, testCase := range testCases {
		testCase := testCase
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			buffer := bytes.NewBuffer(nil)
			testCase.logger.settings.writer = buffer

			logWrapper := func() { // wrap for caller depth of 3
				testCase.logger.log(testCase.level, testCase.s, testCase.args...)
			}

			logWrapper()

			line := buffer.String()
			buffer.Reset()

			regex, err := regexp.Compile(testCase.outputRegex)
			require.NoError(t, err)

			assert.True(t, regex.MatchString(line),
				"line %q does not match regex %q", line, regex.String())
		})
	}
}

func Test_Logger_LevelsLog(t *testing.T) {
	t.Parallel()

	buffer := bytes.NewBuffer(nil)

	logger := New(SetLevel(Trace), SetWriter(buffer))
	logger.Trace("some trace")
	logger.Debug("some debug")
	logger.Info("some info")
	logger.Warn("some warn")
	logger.Error("some error")
	logger.Critical("some critical")
	logger.Tracef("some %dnd trace", 2)
	logger.Debugf("some %dnd debug", 2)
	logger.Infof("some %dnd info", 2)
	logger.Warnf("some %dnd warn", 2)
	logger.Errorf("some %dnd error", 2)
	logger.Criticalf("some %dnd critical", 2)

	lines := strings.Split(buffer.String(), "\n")
	buffer.Reset()

	// Check for trailing newline
	require.NotEmpty(t, lines)
	assert.Equal(t, "", lines[len(lines)-1])
	lines = lines[:len(lines)-1]

	expectedRegexes := []string{
		timePrefixRegex + "TRACE    some trace$",
		timePrefixRegex + "DEBUG    some debug$",
		timePrefixRegex + "INFO     some info$",
		timePrefixRegex + "WARN     some warn$",
		timePrefixRegex + "ERROR    some error$",
		timePrefixRegex + "CRITICAL some critical$",
		timePrefixRegex + "TRACE    some 2nd trace$",
		timePrefixRegex + "DEBUG    some 2nd debug$",
		timePrefixRegex + "INFO     some 2nd info$",
		timePrefixRegex + "WARN     some 2nd warn$",
		timePrefixRegex + "ERROR    some 2nd error$",
		timePrefixRegex + "CRITICAL some 2nd critical$",
	}

	require.Equal(t, len(expectedRegexes), len(lines))

	for i := range lines {
		regex, err := regexp.Compile(expectedRegexes[i])
		require.NoError(t, err)

		assert.True(t, regex.MatchString(lines[i]),
			"line %q does not match regex %q", lines[i], expectedRegexes[i])
	}
}

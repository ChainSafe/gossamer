// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package log

import (
	"io"
	"os"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_New(t *testing.T) {
	t.Parallel()

	testCases := map[string]struct {
		options        []Option
		expectedLogger *Logger
	}{
		"no option": {
			expectedLogger: &Logger{
				settings: settings{
					writer: os.Stdout,
					level:  levelPtr(Info),
					format: formatPtr(FormatConsole),
					caller: newCallerSettings(false, false, false),
				},
				mutex: new(sync.Mutex),
			},
		},
		"all options": {
			options: []Option{
				SetLevel(Trace),
				SetCallerFile(true),
				SetCallerLine(true),
				SetCallerFunc(true),
				SetFormat(FormatConsole),
				SetWriter(io.Discard),
				AddContext("key1", "value1"),
				AddContext("key1", "value2"),
			},
			expectedLogger: &Logger{
				settings: settings{
					writer: io.Discard,
					level:  levelPtr(Trace),
					format: formatPtr(FormatConsole),
					caller: newCallerSettings(true, true, true),
					context: []contextKeyValues{
						{key: "key1", values: []string{"value1", "value2"}},
					},
				},
				mutex: new(sync.Mutex),
			},
		},
	}

	for name, testCase := range testCases {
		testCase := testCase
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			logger := New(testCase.options...)

			assert.Equal(t, testCase.expectedLogger, logger)
		})
	}
}

func Test_Logger_New(t *testing.T) {
	t.Parallel()

	testCases := map[string]struct {
		initialLogger  *Logger
		options        []Option
		expectedLogger *Logger
	}{
		"no option": {
			initialLogger: &Logger{
				settings: settings{
					writer: os.Stdout,
					level:  levelPtr(Info),
					format: formatPtr(FormatConsole),
					caller: newCallerSettings(false, false, false),
				},
				mutex: new(sync.Mutex),
			},
			expectedLogger: &Logger{
				settings: settings{
					writer: os.Stdout,
					level:  levelPtr(Info),
					format: formatPtr(FormatConsole),
					caller: newCallerSettings(false, false, false),
				},
				mutex: new(sync.Mutex),
			},
		},
		"some options": {
			initialLogger: &Logger{
				settings: settings{
					writer: os.Stdout,
					level:  levelPtr(Info),
					format: formatPtr(FormatConsole),
					caller: newCallerSettings(true, true, true),
					context: []contextKeyValues{
						{key: "key1", values: []string{"value1"}},
					},
				},
				mutex: new(sync.Mutex),
			},
			options: []Option{
				SetLevel(Trace),
				SetCallerFunc(false),
				SetFormat(FormatConsole),
				SetWriter(io.Discard),
				AddContext("key1", "value1.2"),
				AddContext("key2", "value2"),
			},
			expectedLogger: &Logger{
				settings: settings{
					writer: io.Discard,
					level:  levelPtr(Trace),
					format: formatPtr(FormatConsole),
					caller: newCallerSettings(true, true, false),
					context: []contextKeyValues{
						{key: "key1", values: []string{"value1", "value1.2"}},
						{key: "key2", values: []string{"value2"}},
					},
				},
				mutex: new(sync.Mutex),
			},
		},
	}

	for name, testCase := range testCases {
		testCase := testCase
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			logger := testCase.initialLogger.New(testCase.options...)

			assert.Equal(t, testCase.expectedLogger, logger)
		})
	}
}

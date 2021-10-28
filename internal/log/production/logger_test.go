package production

import (
	"io/ioutil"
	"os"
	"sync"
	"testing"

	"github.com/ChainSafe/gossamer/internal/log/common"
	"github.com/stretchr/testify/assert"
)

func Test_New(t *testing.T) {
	t.Parallel()

	testCases := map[string]struct {
		options        []common.Option
		expectedLogger *Logger
	}{
		"no option": {
			expectedLogger: &Logger{
				settings: settings{
					writer: os.Stdout,
					level:  levelPtr(LevelInfo),
					format: formatPtr(FormatConsole),
					caller: newCallerSettings(false, false, false),
				},
				mutex: new(sync.Mutex),
			},
		},
		"all options": {
			options: []common.Option{
				SetLevel(LevelTrace),
				SetCallerFile(true),
				SetCallerLine(true),
				SetCallerFunc(true),
				SetFormat(FormatConsole),
				SetWriter(ioutil.Discard),
				AddContext("key1", "value1"),
				AddContext("key1", "value2"),
			},
			expectedLogger: &Logger{
				settings: settings{
					writer: ioutil.Discard,
					level:  levelPtr(LevelTrace),
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
		options        []common.Option
		expectedLogger *Logger
	}{
		"no option": {
			initialLogger: &Logger{
				settings: settings{
					writer: os.Stdout,
					level:  levelPtr(LevelInfo),
					format: formatPtr(FormatConsole),
					caller: newCallerSettings(false, false, false),
				},
				mutex: new(sync.Mutex),
			},
			expectedLogger: &Logger{
				settings: settings{
					writer: os.Stdout,
					level:  levelPtr(LevelInfo),
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
					level:  levelPtr(LevelInfo),
					format: formatPtr(FormatConsole),
					caller: newCallerSettings(true, true, true),
					context: []contextKeyValues{
						{key: "key1", values: []string{"value1"}},
					},
				},
				mutex: new(sync.Mutex),
			},
			options: []common.Option{
				SetLevel(LevelTrace),
				SetCallerFunc(false),
				SetFormat(FormatConsole),
				SetWriter(ioutil.Discard),
				AddContext("key1", "value1.2"),
				AddContext("key2", "value2"),
			},
			expectedLogger: &Logger{
				settings: settings{
					writer: ioutil.Discard,
					level:  levelPtr(LevelTrace),
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

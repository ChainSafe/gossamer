package log

import (
	"io/ioutil"
	"os"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_Logger_Patch(t *testing.T) {
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
		"with options": {
			initialLogger: &Logger{
				settings: settings{
					writer: ioutil.Discard,
					level:  levelPtr(LevelInfo),
					format: formatPtr(FormatConsole),
					caller: newCallerSettings(false, false, false),
				},
				mutex: new(sync.Mutex),
			},
			options: []Option{
				SetLevel(LevelWarn),
				SetCallerFile(true),
			},
			expectedLogger: &Logger{
				settings: settings{
					writer: ioutil.Discard,
					level:  levelPtr(LevelWarn),
					format: formatPtr(FormatConsole),
					caller: newCallerSettings(true, false, false),
				},
				mutex: new(sync.Mutex),
			},
		},
	}

	for name, testCase := range testCases {
		testCase := testCase
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			logger := testCase.initialLogger

			logger.Patch(testCase.options...)

			assert.Equal(t, testCase.expectedLogger, logger)
		})
	}
}

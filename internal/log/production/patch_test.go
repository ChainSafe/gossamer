package production

import (
	"io/ioutil"
	"os"
	"sync"
	"testing"

	"github.com/ChainSafe/gossamer/internal/log/common"
	"github.com/stretchr/testify/assert"
)

func Test_Logger_Patch(t *testing.T) {
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
		"with options": {
			initialLogger: &Logger{
				settings: settings{
					writer: ioutil.Discard,
					level:  levelPtr(Info),
					format: formatPtr(FormatConsole),
					caller: newCallerSettings(false, false, false),
				},
				mutex: new(sync.Mutex),
			},
			options: []common.Option{
				SetLevel(Warn),
				SetCallerFile(true),
			},
			expectedLogger: &Logger{
				settings: settings{
					writer: ioutil.Discard,
					level:  levelPtr(Warn),
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

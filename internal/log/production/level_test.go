package production

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_Level_ColouredString(t *testing.T) {
	t.Parallel()

	testCases := map[string]struct {
		level Level
		s     string
	}{
		"trace": {
			level: LevelTrace,
			s:     "TRCE",
		},
		"debug": {
			level: LevelDebug,
			s:     "DBUG",
		},
		"info": {
			level: LevelInfo,
			s:     "INFO",
		},
		"warn": {
			level: LevelWarn,
			s:     "WARN",
		},
		"error": {
			level: LevelError,
			s:     "EROR",
		},
		"critical": {
			level: LevelCritical,
			s:     "CRIT",
		},
		"unknown": {
			level: 178,
			s:     "???",
		},
	}

	for name, testCase := range testCases {
		testCase := testCase
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			s := testCase.level.ColouredString()
			// Note: fatih/colour is clever enough to not add colours
			// when running tests, so the string is effectively without
			// colour here.

			assert.Equal(t, testCase.s, s)
		})
	}
}

func Test_ParseLevel(t *testing.T) {
	t.Parallel()

	testCases := map[string]struct {
		s     string
		level Level
		err   error
	}{
		"trace": {
			s:     "TRCE",
			level: LevelTrace,
		},
		"debug": {
			s:     "DBUG",
			level: LevelDebug,
		},
		"info": {
			s:     "INFO",
			level: LevelInfo,
		},
		"warn": {
			s:     "WARN",
			level: LevelWarn,
		},
		"error": {
			s:     "EROR",
			level: LevelError,
		},
		"critical": {
			s:     "CRIT",
			level: LevelCritical,
		},
		"invalid": {
			s:   "someinvalid",
			err: errors.New("level is not recognised: someinvalid"),
		},
	}

	for name, testCase := range testCases {
		testCase := testCase
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			level, err := ParseLevel(testCase.s)

			if testCase.err != nil {
				require.EqualError(t, err, testCase.err.Error())
			} else {
				assert.NoError(t, err)
			}

			assert.Equal(t, testCase.level, level)
		})
	}
}

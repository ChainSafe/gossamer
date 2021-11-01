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
			level: Trace,
			s:     "TRCE",
		},
		"debug": {
			level: Debug,
			s:     "DBUG",
		},
		"info": {
			level: Info,
			s:     "INFO",
		},
		"warn": {
			level: Warn,
			s:     "WARN",
		},
		"error": {
			level: Error,
			s:     "EROR",
		},
		"critical": {
			level: Critical,
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
			level: Trace,
		},
		"debug": {
			s:     "DBUG",
			level: Debug,
		},
		"info": {
			s:     "INFO",
			level: Info,
		},
		"warn": {
			s:     "WARN",
			level: Warn,
		},
		"error": {
			s:     "EROR",
			level: Error,
		},
		"critical": {
			s:     "CRIT",
			level: Critical,
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

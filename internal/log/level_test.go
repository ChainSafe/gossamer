// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package log

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_Level_Format(t *testing.T) {
	t.Parallel()

	testCases := map[string]struct {
		level Level
		s     string
	}{
		"trace": {
			level: Trace,
			s:     "TRACE   ",
		},
		"debug": {
			level: Debug,
			s:     "DEBUG   ",
		},
		"info": {
			level: Info,
			s:     "INFO    ",
		},
		"warn": {
			level: Warn,
			s:     "WARN    ",
		},
		"error": {
			level: Error,
			s:     "ERROR   ",
		},
		"critical": {
			level: Critical,
			s:     "CRITICAL",
		},
		"unknown": {
			level: 178,
			s:     "???     ",
		},
	}

	for name, testCase := range testCases {
		testCase := testCase
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			s := testCase.level.format()
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
		"-1": {
			s:   "-1",
			err: errors.New("level integer can only be between 0 and 5 included: -1"),
		},
		"0": {
			s:     "0",
			level: Critical,
		},
		"5": {
			s:     "5",
			level: Trace,
		},
		"6": {
			s:   "6",
			err: errors.New("level integer can only be between 0 and 5 included: 6"),
		},
		"trace": {
			s:     "TRACE",
			level: Trace,
		},
		"debug": {
			s:     "DEBUG",
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
			s:     "ERROR",
			level: Error,
		},
		"critical": {
			s:     "CRITICAL",
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

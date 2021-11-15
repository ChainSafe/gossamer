// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package log

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_getCallerString(t *testing.T) {
	t.Parallel()

	boolPtr := func(b bool) *bool { return &b }

	testCases := map[string]struct {
		settings callerSettings
		s        string
	}{
		"no show": {
			settings: callerSettings{
				file: boolPtr(false),
				line: boolPtr(false),
				funC: boolPtr(false),
			},
		},
		"show file line": {
			settings: callerSettings{
				file: boolPtr(true),
				line: boolPtr(true),
				funC: boolPtr(false),
			},
			s: "caller_test.go:L58",
		},
		"show all": {
			settings: callerSettings{
				file: boolPtr(true),
				line: boolPtr(true),
				funC: boolPtr(true),
			},
			s: "caller_test.go:L58:func2",
		},
	}

	for name, testCase := range testCases {
		testCase := testCase
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			var s string
			wrapFunc1 := func() { // Debug/Info calls
				func() { // log function
					s = getCallerString(testCase.settings)
				}()
			}

			wrapFunc1()

			assert.Equal(t, testCase.s, s)
		})
	}
}

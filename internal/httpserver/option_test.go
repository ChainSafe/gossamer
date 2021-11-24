// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package httpserver

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func Test_newOptionalSettings(t *testing.T) {
	t.Parallel()

	testCases := map[string]struct {
		options  []Option
		settings optionalSettings
	}{
		"no option": {
			settings: optionalSettings{
				shutdownTimeout: 3 * time.Second,
			},
		},
		"shutdown option": {
			options: []Option{
				ShutdownTimeout(time.Second),
			},
			settings: optionalSettings{
				shutdownTimeout: time.Second,
			},
		},
	}

	for name, testCase := range testCases {
		testCase := testCase
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			settings := newOptionalSettings(testCase.options)

			assert.Equal(t, testCase.settings, settings)
		})
	}
}

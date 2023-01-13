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
		"no_option": {
			settings: optionalSettings{
				shutdownTimeout:   3 * time.Second,
				readTimeout:       10 * time.Second,
				readHeaderTimeout: time.Second,
			},
		},
		"all_options_set": {
			options: []Option{
				ShutdownTimeout(3 * time.Second),
				ReadTimeout(time.Second),
				ReadHeaderTimeout(2 * time.Second),
			},
			settings: optionalSettings{
				readTimeout:       time.Second,
				readHeaderTimeout: 2 * time.Second,
				shutdownTimeout:   3 * time.Second,
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

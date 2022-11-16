// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package httpserver

import (
	"net/http"
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
				handler:           http.NewServeMux(),
				logger:            &noopLogger{},
				shutdownTimeout:   3 * time.Second,
				readTimeout:       10 * time.Second,
				readHeaderTimeout: time.Second,
			},
		},
		"all options set": {
			options: []Option{
				Handler(http.NewServeMux()),
				Address("test"),
				Logger("testname", NewMockInfoer(nil)),
				ShutdownTimeout(3 * time.Second),
				ReadTimeout(time.Second),
				ReadHeaderTimeout(2 * time.Second),
			},
			settings: optionalSettings{
				handler:           http.NewServeMux(),
				address:           "test",
				serverName:        "testname",
				logger:            NewMockInfoer(nil),
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

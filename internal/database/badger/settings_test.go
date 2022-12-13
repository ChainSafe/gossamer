// Copyright 2022 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package badger

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func ptrTo[T any](x T) *T { return &x }

func Test_Settings_SetDefaults(t *testing.T) {
	t.Parallel()

	testCases := map[string]struct {
		originalSettings Settings
		expectedSettings Settings
	}{
		"empty settings": {
			expectedSettings: Settings{
				Path: new(string),
			},
		},
		"non-empty settings": {
			originalSettings: Settings{
				Path: ptrTo("x"),
			},
			expectedSettings: Settings{
				Path: ptrTo("x"),
			},
		},
	}

	for name, testCase := range testCases {
		testCase := testCase

		t.Run(name, func(t *testing.T) {
			t.Parallel()

			testCase.originalSettings.SetDefaults()
			assert.Equal(t, testCase.expectedSettings, testCase.originalSettings)
		})
	}
}

func Test_Settings_Validate(t *testing.T) {
	t.Parallel()

	testCases := map[string]struct {
		settings   Settings
		errMessage string
	}{
		// Note we cannot test for a bad path since we would
		// need os.Getcwd() to fail.
		"valid settings": {
			settings: Settings{
				Path: ptrTo("."),
			},
		},
	}

	for name, testCase := range testCases {
		testCase := testCase

		t.Run(name, func(t *testing.T) {
			t.Parallel()

			err := testCase.settings.Validate()

			if testCase.errMessage == "" {
				assert.NoError(t, err)
			} else {
				assert.EqualError(t, err, testCase.errMessage)
			}
		})
	}
}

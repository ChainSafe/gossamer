// Copyright 2022 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package trie

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_Version_String(t *testing.T) {
	t.Parallel()

	testCases := map[string]struct {
		version       TrieLayout
		versionString string
		panicMessage  string
	}{
		"v0": {
			version:       V0,
			versionString: "v0",
		},
		"invalid": {
			version:      TrieLayout(99),
			panicMessage: "unknown version 99",
		},
	}

	for name, testCase := range testCases {
		testCase := testCase
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			if testCase.panicMessage != "" {
				assert.PanicsWithValue(t, testCase.panicMessage, func() {
					_ = testCase.version.String()
				})
				return
			}

			versionString := testCase.version.String()
			assert.Equal(t, testCase.versionString, versionString)
		})
	}
}

func Test_ParseVersion(t *testing.T) {
	t.Parallel()

	testCases := map[string]struct {
		v          any
		version    TrieLayout
		errWrapped error
		errMessage string
	}{
		"v0": {
			v:       "v0",
			version: V0,
		},
		"V0": {
			v:       "V0",
			version: V0,
		},
		"0": {
			v:       uint8(0),
			version: V0,
		},
		"v1": {
			v:       "v1",
			version: V1,
		},
		"V1": {
			v:       "V1",
			version: V1,
		},
		"1": {
			v:       uint8(1),
			version: V1,
		},
		"invalid": {
			v:          "xyz",
			errWrapped: ErrParseVersion,
			errMessage: "parsing version failed: \"xyz\" must be one of [v0, v1]",
		},
		"invalid_uint8": {
			v:          uint8(99),
			errWrapped: ErrParseVersion,
			errMessage: "parsing version failed: \"V99\" must be one of [v0, v1]",
		},
	}

	for name, testCase := range testCases {
		testCase := testCase
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			var version TrieLayout

			var err error
			switch typed := testCase.v.(type) {
			case string:
				version, err = ParseVersion(typed)
			case uint8:
				version, err = ParseVersion(typed)
			default:
				t.Fail()
			}

			assert.Equal(t, testCase.version, version)
			assert.ErrorIs(t, err, testCase.errWrapped)
			if testCase.errWrapped != nil {
				assert.EqualError(t, err, testCase.errMessage)
			}
		})
	}
}

func Test_Version_MaxInlineValue(t *testing.T) {
	t.Parallel()

	testCases := map[string]struct {
		version      TrieLayout
		max          int
		panicMessage string
	}{
		"v0": {
			version: V0,
			max:     NoMaxInlineValueSize,
		},
		"v1": {
			version: V1,
			max:     V1MaxInlineValueSize,
		},
		"invalid": {
			version:      TrieLayout(99),
			max:          0,
			panicMessage: "unknown version 99",
		},
	}

	for name, testCase := range testCases {
		testCase := testCase
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			if testCase.panicMessage != "" {
				assert.PanicsWithValue(t, testCase.panicMessage, func() {
					_ = testCase.version.MaxInlineValue()
				})
				return
			}

			maxInline := testCase.version.MaxInlineValue()
			assert.Equal(t, testCase.max, maxInline)
		})
	}
}

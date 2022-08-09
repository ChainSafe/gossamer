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
		version       Version
		versionString string
		panicMessage  string
	}{
		"v0": {
			version:       V0,
			versionString: "v0",
		},
		"invalid": {
			version:      Version(99),
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
		s          string
		version    Version
		errWrapped error
		errMessage string
	}{
		"v0": {
			s:       "v0",
			version: V0,
		},
		"V0": {
			s:       "V0",
			version: V0,
		},
		"invalid": {
			s:          "xyz",
			errWrapped: ErrParseVersion,
			errMessage: "parsing version failed: \"xyz\" must be v0",
		},
	}

	for name, testCase := range testCases {
		testCase := testCase
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			version, err := ParseVersion(testCase.s)

			assert.Equal(t, testCase.version, version)
			assert.ErrorIs(t, err, testCase.errWrapped)
			if testCase.errWrapped != nil {
				assert.EqualError(t, err, testCase.errMessage)
			}
		})
	}
}

func Test_orStrings(t *testing.T) {
	t.Parallel()

	testCases := map[string]struct {
		strings []string
		s       string
	}{
		"nil": {},
		"v0": {
			strings: []string{"v0"},
			s:       "v0",
		},
		"v0 or v1": {
			strings: []string{"v0", "v1"},
			s:       "v0 or v1",
		},
		"v0, v1 or v2": {
			strings: []string{"v0", "v1", "v2"},
			s:       "v0, v1 or v2",
		},
		"v0, v1, v2 or v3": {
			strings: []string{"v0", "v1", "v2", "v3"},
			s:       "v0, v1, v2 or v3",
		},
	}

	for name, testCase := range testCases {
		testCase := testCase
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			s := orStrings(testCase.strings)

			assert.Equal(t, testCase.s, s)
		})
	}
}

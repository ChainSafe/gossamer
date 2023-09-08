// Copyright 2022 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package trie

import (
	"fmt"
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
		v          any
		version    Version
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
			v:       uint32(0),
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
			v:       uint32(1),
			version: V1,
		},
		"invalid": {
			v:          "xyz",
			errWrapped: ErrParseVersion,
			errMessage: "parsing version failed: \"xyz\" must be one of [v0, v1]",
		},
		"invalid_uint32": {
			v:          uint32(999),
			errWrapped: ErrParseVersion,
			errMessage: "parsing version failed: \"V999\" must be one of [v0, v1]",
		},
	}

	for name, testCase := range testCases {
		testCase := testCase
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			var version Version

			var err error
			switch typed := testCase.v.(type) {
			case string:
				version, err = ParseVersion(typed)
			case uint32:
				version, err = ParseVersion(typed)
			default:
				panic(fmt.Sprintf("unsupported type %T", testCase.v))
			}

			assert.Equal(t, testCase.version, version)
			assert.ErrorIs(t, err, testCase.errWrapped)
			if testCase.errWrapped != nil {
				assert.EqualError(t, err, testCase.errMessage)
			}
		})
	}
}

func Test_ShouldHashValue(t *testing.T) {
	t.Parallel()

	testCases := map[string]struct {
		version      Version
		value        []byte
		shouldHash   bool
		panicMessage string
	}{
		"v0_small_value": {
			version:    V0,
			value:      []byte("smallvalue"),
			shouldHash: false,
		},
		"v0_large_value": {
			version:    V0,
			value:      []byte("newvaluewithmorethan32byteslength"),
			shouldHash: false,
		},
		"v1_small_value": {
			version:    V1,
			value:      []byte("smallvalue"),
			shouldHash: false,
		},
		"v1_large_value": {
			version:    V1,
			value:      []byte("newvaluewithmorethan32byteslength"),
			shouldHash: true,
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
					_ = testCase.version.ShouldHashValue(testCase.value)
				})
				return
			}

			shouldHash := testCase.version.ShouldHashValue(testCase.value)
			assert.Equal(t, testCase.shouldHash, shouldHash)
		})
	}
}

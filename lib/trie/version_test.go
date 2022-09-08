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
		x          any
		version    Version
		errWrapped error
		errMessage string
	}{
		"v0": {
			x:       "v0",
			version: V0,
		},
		"V0": {
			x:       "V0",
			version: V0,
		},
		"invalid string": {
			x:          "xyz",
			errWrapped: ErrVersionNotValid,
			errMessage: "version not valid: xyz",
		},
		"0 uint32": {
			x:       uint32(0),
			version: V0,
		},
		"invalid uint32": {
			x:          uint32(100),
			errWrapped: ErrVersionNotValid,
			errMessage: "version not valid: V100",
		},
	}

	for name, testCase := range testCases {
		testCase := testCase
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			var version Version
			var err error
			switch typedX := testCase.x.(type) {
			case string:
				version, err = ParseVersion(typedX)
			case uint32:
				version, err = ParseVersion(typedX)
			default:
				panic(fmt.Sprintf("unsupported type %T", testCase.x))
			}

			assert.Equal(t, testCase.version, version)
			assert.ErrorIs(t, err, testCase.errWrapped)
			if testCase.errWrapped != nil {
				assert.EqualError(t, err, testCase.errMessage)
			}
		})
	}
}

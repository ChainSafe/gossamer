// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package runtime

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_Version_Encode(t *testing.T) {
	t.Parallel()

	someVersion := Version{
		SpecName:         []byte{1},
		ImplName:         []byte{2},
		AuthoringVersion: 3,
		SpecVersion:      4,
		ImplVersion:      5,
		APIItems: []APIItem{{
			Name: [8]byte{1, 2, 3, 4, 5, 6, 7, 8},
			Ver:  6,
		}},
		TransactionVersion: 7,
	}

	testCases := map[string]struct {
		version    Version
		encoding   []byte
		errWrapped error
		errMessage string
	}{
		"not legacy": {
			version: someVersion,
			encoding: []byte{
				0x4, 0x1, 0x4, 0x2, 0x3, 0x0, 0x0, 0x0, 0x4, 0x0, 0x0, 0x0,
				0x5, 0x0, 0x0, 0x0, 0x4, 0x1, 0x2, 0x3, 0x4, 0x5, 0x6, 0x7,
				0x8, 0x6, 0x0, 0x0, 0x0, 0x7, 0x0, 0x0, 0x0},
		},
		"legacy": {
			version: someVersion.WithLegacy(),
			encoding: []byte{
				0x4, 0x1, 0x4, 0x2, 0x3, 0x0, 0x0, 0x0, 0x4, 0x0, 0x0, 0x0,
				0x5, 0x0, 0x0, 0x0, 0x4, 0x1, 0x2, 0x3, 0x4, 0x5, 0x6, 0x7,
				0x8, 0x6, 0x0, 0x0, 0x0},
		},
	}

	for name, testCase := range testCases {
		testCase := testCase
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			encoded, err := testCase.version.Encode()

			assert.ErrorIs(t, err, testCase.errWrapped)
			if testCase.errWrapped != nil {
				require.EqualError(t, err, testCase.errMessage)
			}
			assert.Equal(t, testCase.encoding, encoded)
		})
	}
}

func Test_DecodeVersion(t *testing.T) {
	t.Parallel()

	testCases := map[string]struct {
		encoded    []byte
		version    Version
		errWrapped error
		errMessage string
	}{
		"unmarshal success": {
			encoded: []byte{
				0x4, 0x1, 0x4, 0x2, 0x3, 0x0, 0x0, 0x0, 0x4, 0x0, 0x0, 0x0,
				0x5, 0x0, 0x0, 0x0, 0x4, 0x1, 0x2, 0x3, 0x4, 0x5, 0x6, 0x7,
				0x8, 0x6, 0x0, 0x0, 0x0, 0x7, 0x0, 0x0, 0x0},
			version: Version{
				SpecName:         []byte{1},
				ImplName:         []byte{2},
				AuthoringVersion: 3,
				SpecVersion:      4,
				ImplVersion:      5,
				APIItems: []APIItem{{
					Name: [8]byte{1, 2, 3, 4, 5, 6, 7, 8},
					Ver:  6,
				}},
				TransactionVersion: 7,
			},
		},
		"unmarshal error": {
			encoded:    []byte{255, 255},
			errWrapped: ErrDecodingVersion,
			errMessage: "decoding version: could not decode invalid integer, field: []",
		},
		"legacy unmarshal error": {
			encoded:    []byte{0},
			errWrapped: ErrDecodingLegacyVersion,
			errMessage: "decoding legacy version: EOF, field: []",
		},
		"legacy unmarshal success": {
			encoded: []byte{
				0x4, 0x1, 0x4, 0x2, 0x3, 0x0, 0x0, 0x0, 0x4, 0x0, 0x0, 0x0,
				0x5, 0x0, 0x0, 0x0, 0x4, 0x1, 0x2, 0x3, 0x4, 0x5, 0x6, 0x7,
				0x8, 0x6, 0x0, 0x0, 0x0},
			version: Version{
				SpecName:         []byte{1},
				ImplName:         []byte{2},
				AuthoringVersion: 3,
				SpecVersion:      4,
				ImplVersion:      5,
				APIItems: []APIItem{{
					Name: [8]byte{1, 2, 3, 4, 5, 6, 7, 8},
					Ver:  6,
				}},
				legacy: true,
			},
		},
	}

	for name, testCase := range testCases {
		testCase := testCase
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			version, err := DecodeVersion(testCase.encoded)

			assert.ErrorIs(t, err, testCase.errWrapped)
			if testCase.errWrapped != nil {
				require.EqualError(t, err, testCase.errMessage)
			}
			assert.Equal(t, testCase.version, version)
		})
	}
}

func Test_Version_Scale(t *testing.T) {
	t.Parallel()

	testCases := map[string]struct {
		version  Version
		encoding []byte
		decoded  Version
	}{
		"current version": {
			version: Version{
				SpecName:         []byte{1},
				ImplName:         []byte{2},
				AuthoringVersion: 3,
				SpecVersion:      4,
				ImplVersion:      5,
				APIItems: []APIItem{{
					Name: [8]byte{1, 2, 3, 4, 5, 6, 7, 8},
					Ver:  6,
				}},
				TransactionVersion: 7,
			},
			encoding: []byte{
				0x4, 0x1, 0x4, 0x2, 0x3, 0x0, 0x0, 0x0, 0x4, 0x0, 0x0, 0x0,
				0x5, 0x0, 0x0, 0x0, 0x4, 0x1, 0x2, 0x3, 0x4, 0x5, 0x6, 0x7,
				0x8, 0x6, 0x0, 0x0, 0x0, 0x7, 0x0, 0x0, 0x0},
			decoded: Version{
				SpecName:         []byte{1},
				ImplName:         []byte{2},
				AuthoringVersion: 3,
				SpecVersion:      4,
				ImplVersion:      5,
				APIItems: []APIItem{{
					Name: [8]byte{1, 2, 3, 4, 5, 6, 7, 8},
					Ver:  6,
				}},
				TransactionVersion: 7,
			},
		},
		"legacy version": {
			version: Version{
				SpecName:         []byte{1},
				ImplName:         []byte{2},
				AuthoringVersion: 3,
				SpecVersion:      4,
				ImplVersion:      5,
				APIItems: []APIItem{{
					Name: [8]byte{1, 2, 3, 4, 5, 6, 7, 8},
					Ver:  6,
				}},
				TransactionVersion: 7,
				legacy:             true,
			},
			encoding: []byte{
				0x4, 0x1, 0x4, 0x2, 0x3, 0x0, 0x0, 0x0, 0x4, 0x0, 0x0, 0x0,
				0x5, 0x0, 0x0, 0x0, 0x4, 0x1, 0x2, 0x3, 0x4, 0x5, 0x6, 0x7,
				0x8, 0x6, 0x0, 0x0, 0x0},
			decoded: Version{
				SpecName:         []byte{1},
				ImplName:         []byte{2},
				AuthoringVersion: 3,
				SpecVersion:      4,
				ImplVersion:      5,
				APIItems: []APIItem{{
					Name: [8]byte{1, 2, 3, 4, 5, 6, 7, 8},
					Ver:  6,
				}},
				legacy: true,
			},
		},
	}

	for name, testCase := range testCases {
		testCase := testCase
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			encoded, err := testCase.version.Encode()

			require.NoError(t, err)
			require.Equal(t, testCase.encoding, encoded)

			decoded, err := DecodeVersion(encoded)
			require.NoError(t, err)

			assert.Equal(t, testCase.decoded, decoded)
		})
	}
}

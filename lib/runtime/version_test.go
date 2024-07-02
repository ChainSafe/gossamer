// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package runtime

import (
	"testing"

	"github.com/ChainSafe/gossamer/pkg/scale"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func scaleEncode(t *testing.T, x any) []byte {
	encoded, err := scale.Marshal(x)
	require.NoError(t, err)
	return encoded
}

func concatBytes(slices [][]byte) (concatenated []byte) {
	for _, slice := range slices {
		concatenated = append(concatenated, slice...)
	}
	return concatenated
}

func Test_DecodeVersion(t *testing.T) {
	t.Parallel()

	testCases := map[string]struct {
		encoded    []byte
		version    Version
		errWrapped error
		errMessage string
	}{
		"required_field_decode_error": {
			encoded: concatBytes([][]byte{
				scaleEncode(t, []byte{1, 2}),
				{255, 255}, // error
			}),
			errWrapped: ErrDecodingVersionField,
			errMessage: "decoding version field impl name: decoding uint: unknown prefix for compact uint: 255",
		},
		// TODO add transaction version decode error once
		// https://github.com/ChainSafe/gossamer/pull/2683
		// is merged.
		// "transaction version decode error": {
		// 	encoded: concatBytes([][]byte{
		// 		scaleEncode(t, []byte("a")),   // spec name
		// 		scaleEncode(t, []byte("b")),   // impl name
		// 		scaleEncode(t, uint32(1)),     // authoring version
		// 		scaleEncode(t, uint32(2)),     // spec version
		// 		scaleEncode(t, uint32(3)),     // impl version
		// 		scaleEncode(t, []APIItem{{}}), // api items
		// 		{1, 2, 3},                     // transaction version
		// 	}),
		// 	errWrapped: ErrDecoding,
		// 	errMessage: "decoding transaction version: could not decode invalid integer",
		// },
		// TODO add state version decode error once
		// https://github.com/ChainSafe/gossamer/pull/2683
		// is merged.
		// "state version decode error": {
		// 	encoded: concatBytes([][]byte{
		// 		scaleEncode(t, []byte("a")),   // spec name
		// 		scaleEncode(t, []byte("b")),   // impl name
		// 		scaleEncode(t, uint32(1)),     // authoring version
		// 		scaleEncode(t, uint32(2)),     // spec version
		// 		scaleEncode(t, uint32(3)),     // impl version
		// 		scaleEncode(t, []APIItem{{}}), // api items
		// 		scaleEncode(t, uint32(4)),     // transaction version
		// 		{1, 2, 3},                     // state version
		// 	}),
		// 	errWrapped: ErrDecoding,
		// 	errMessage: "decoding state version: could not decode invalid integer",
		// },
		"no_optional_field_set": {
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
			},
		},
		"transaction_version_set": {
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
		"transaction_and_state_versions_set": {
			encoded: []byte{
				0x4, 0x1, 0x4, 0x2, 0x3, 0x0, 0x0, 0x0, 0x4, 0x0, 0x0, 0x0,
				0x5, 0x0, 0x0, 0x0, 0x4, 0x1, 0x2, 0x3, 0x4, 0x5, 0x6, 0x7,
				0x8, 0x6, 0x0, 0x0, 0x0, 0x7, 0x0, 0x0, 0x0, 0x4, 0x0, 0x0, 0x0},
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
				StateVersion:       4,
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
		"all_fields_set": {
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
				StateVersion:       4,
			},
			encoding: []byte{
				0x4, 0x1, 0x4, 0x2, 0x3, 0x0, 0x0, 0x0, 0x4, 0x0, 0x0, 0x0,
				0x5, 0x0, 0x0, 0x0, 0x4, 0x1, 0x2, 0x3, 0x4, 0x5, 0x6, 0x7,
				0x8, 0x6, 0x0, 0x0, 0x0, 0x7, 0x0, 0x0, 0x0, 0x4},
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
				StateVersion:       4,
			},
		},
	}

	for name, testCase := range testCases {
		testCase := testCase
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			encoded, err := scale.Marshal(testCase.version)

			require.NoError(t, err)
			require.Equal(t, testCase.encoding, encoded)

			decoded, err := DecodeVersion(encoded)
			require.NoError(t, err)

			assert.Equal(t, testCase.decoded, decoded)
		})
	}
}

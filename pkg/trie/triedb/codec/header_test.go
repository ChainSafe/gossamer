// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package codec

import (
	"errors"
	"sort"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

var errTest = errors.New("test error")

func Test_decodeHeader(t *testing.T) {
	t.Parallel()

	testCases := map[string]struct {
		reads            []readCall
		nodeVariant      variant
		partialKeyLength uint16
		errWrapped       error
		errMessage       string
	}{
		"first_byte_read_error": {
			reads: []readCall{
				{buffArgCap: 1, err: errTest},
			},
			errWrapped: errTest,
			errMessage: "reading header byte: test error",
		},
		"header_byte_decoding_error": {
			reads: []readCall{
				{buffArgCap: 1, read: []byte{0b0000_1000}},
			},
			errWrapped: ErrVariantUnknown,
			errMessage: "decoding header byte: node variant is unknown: for header byte 00001000",
		},
		"partial_key_length_contained_in_first_byte": {
			reads: []readCall{
				{buffArgCap: 1, read: []byte{leafVariant.bits | 0b0011_1110}},
			},
			nodeVariant:      leafVariant,
			partialKeyLength: uint16(0b0011_1110),
		},
		"long_partial_key_length_and_second_byte_read_error": {
			reads: []readCall{
				{buffArgCap: 1, read: []byte{leafVariant.bits | 0b0011_1111}},
				{buffArgCap: 1, err: errTest},
			},
			errWrapped: errTest,
			errMessage: "reading key length: test error",
		},
		"partial_key_length_spread_on_multiple_bytes": {
			reads: []readCall{
				{buffArgCap: 1, read: []byte{leafVariant.bits | 0b0011_1111}},
				{buffArgCap: 1, read: []byte{0b1111_1111}},
				{buffArgCap: 1, read: []byte{0b1111_0000}},
			},
			nodeVariant:      leafVariant,
			partialKeyLength: uint16(0b0011_1111 + 0b1111_1111 + 0b1111_0000),
		},
		"partial_key_length_too_long": {
			reads: repeatReadCall(readCall{
				buffArgCap: 1,
				read:       []byte{0b1111_1111},
			}, 258),
			errWrapped: ErrPartialKeyTooBig,
			errMessage: "partial key length cannot be larger than 2^16: overflowed by 254",
		},
	}

	for name, testCase := range testCases {
		testCase := testCase
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)

			reader := NewMockReader(ctrl)
			var previousCall *gomock.Call
			for _, readCall := range testCase.reads {
				readCall := readCall // required variable pinning
				byteSliceCapMatcher := newByteSliceCapMatcher(readCall.buffArgCap)
				call := reader.EXPECT().Read(byteSliceCapMatcher).
					DoAndReturn(func(b []byte) (n int, err error) {
						copy(b, readCall.read)
						return readCall.n, readCall.err
					})
				if previousCall != nil {
					call.After(previousCall)
				}
				previousCall = call
			}

			nodeVariant, partialKeyLength, err := decodeHeader(reader)

			assert.Equal(t, testCase.nodeVariant, nodeVariant)
			assert.Equal(t, int(testCase.partialKeyLength), int(partialKeyLength))
			assert.ErrorIs(t, err, testCase.errWrapped)
			if testCase.errWrapped != nil {
				assert.EqualError(t, err, testCase.errMessage)
			}
		})
	}
}

func Test_decodeHeaderByte(t *testing.T) {
	t.Parallel()

	testCases := map[string]struct {
		header                 byte
		nodeVariant            variant
		partialKeyLengthHeader byte
		errWrapped             error
		errMessage             string
	}{
		"empty_variant_header": {
			header:                 0b0000_0000,
			nodeVariant:            emptyVariant,
			partialKeyLengthHeader: 0b0000_0000,
		},
		"branch_with_value_header": {
			header:                 0b1110_1001,
			nodeVariant:            branchWithValueVariant,
			partialKeyLengthHeader: 0b0010_1001,
		},
		"branch_header": {
			header:                 0b1010_1001,
			nodeVariant:            branchVariant,
			partialKeyLengthHeader: 0b0010_1001,
		},
		"leaf_header": {
			header:                 0b0110_1001,
			nodeVariant:            leafVariant,
			partialKeyLengthHeader: 0b0010_1001,
		},
		"leaf_containing_hashes_header": {
			header:                 0b0011_1001,
			nodeVariant:            leafWithHashedValueVariant,
			partialKeyLengthHeader: 0b0001_1001,
		},
		"branch_containing_hashes_header": {
			header:                 0b0001_1001,
			nodeVariant:            branchWithHashedValueVariant,
			partialKeyLengthHeader: 0b0000_1001,
		},
		"compact_encoding_header": {
			header:                 0b0000_0001,
			nodeVariant:            compactEncodingVariant,
			partialKeyLengthHeader: 0b0000_0000,
		},
		"unknown_variant_header": {
			header:     0b0000_1000,
			errWrapped: ErrVariantUnknown,
			errMessage: "node variant is unknown: for header byte 00001000",
		},
	}

	for name, testCase := range testCases {
		testCase := testCase
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			nodeVariant, partialKeyLengthHeader,
				err := decodeHeaderByte(testCase.header)

			assert.Equal(t, testCase.nodeVariant, nodeVariant)
			assert.Equal(t, testCase.partialKeyLengthHeader, partialKeyLengthHeader)
			assert.ErrorIs(t, err, testCase.errWrapped)
			if testCase.errWrapped != nil {
				assert.EqualError(t, err, testCase.errMessage)
			}
		})
	}
}

func Test_variantsOrderedByBitMask(t *testing.T) {
	t.Parallel()

	slice := make([]variant, len(variantsOrderedByBitMask))
	sortedSlice := make([]variant, len(variantsOrderedByBitMask))
	copy(slice, variantsOrderedByBitMask[:])
	copy(sortedSlice, variantsOrderedByBitMask[:])

	sort.Slice(slice, func(i, j int) bool {
		return slice[i].mask < slice[j].mask
	})

	assert.Equal(t, sortedSlice, slice)
}

func Benchmark_decodeHeaderByte(b *testing.B) {
	// For 7 variants defined in the variants array:
	// With global scoped variants slice:
	// 2.987 ns/op	       0 B/op	       0 allocs/op
	// With locally scoped variants slice:
	// 3.873 ns/op	       0 B/op	       0 allocs/op
	header := leafVariant.bits | 0b0000_0001
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _, _ = decodeHeaderByte(header)
	}
}

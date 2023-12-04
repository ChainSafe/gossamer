// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package node

import (
	"bytes"
	"io"
	"math"
	"sort"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func Test_encodeHeader(t *testing.T) {
	t.Parallel()

	largeValue := []byte("newvaluewithmorethan32byteslength")

	testCases := map[string]struct {
		node               *Node
		writes             []writeCall
		maxInlineValueSize int
		errWrapped         error
		errMessage         string
	}{
		"branch_with_no_key": {
			node: &Node{
				Children: make([]*Node, ChildrenCapacity),
			},
			writes: []writeCall{
				{written: []byte{branchVariant.bits}},
			},
		},
		"branch_with_value": {
			node: &Node{
				StorageValue: []byte{},
				Children:     make([]*Node, ChildrenCapacity),
			},
			writes: []writeCall{
				{written: []byte{branchWithValueVariant.bits}},
			},
		},
		"branch_with_hashed_value": {
			node: &Node{
				StorageValue: largeValue,
				Children:     make([]*Node, ChildrenCapacity),
			},
			maxInlineValueSize: 32,
			writes: []writeCall{
				{written: []byte{branchWithHashedValueVariant.bits}},
			},
		},
		"branch_with_key_of_length_30": {
			node: &Node{
				PartialKey: make([]byte, 30),
				Children:   make([]*Node, ChildrenCapacity),
			},
			writes: []writeCall{
				{written: []byte{branchVariant.bits | 30}},
			},
		},
		"branch_with_key_of_length_62": {
			node: &Node{
				PartialKey: make([]byte, 62),
				Children:   make([]*Node, ChildrenCapacity),
			},
			writes: []writeCall{
				{written: []byte{branchVariant.bits | 62}},
			},
		},
		"branch_with_key_of_length_63": {
			node: &Node{
				PartialKey: make([]byte, 63),
				Children:   make([]*Node, ChildrenCapacity),
			},
			writes: []writeCall{
				{written: []byte{branchVariant.bits | 63}},
				{written: []byte{0x00}}, // trailing 0 to indicate the partial
				// key length is done here.
			},
		},
		"branch_with_key_of_length_64": {
			node: &Node{
				PartialKey: make([]byte, 64),
				Children:   make([]*Node, ChildrenCapacity),
			},
			writes: []writeCall{
				{written: []byte{branchVariant.bits | 63}},
				{written: []byte{0x01}},
			},
		},
		"branch_with_small_key_length_write_error": {
			node: &Node{
				Children: make([]*Node, ChildrenCapacity),
			},
			writes: []writeCall{
				{
					written: []byte{branchVariant.bits},
					err:     errTest,
				},
			},
			errWrapped: errTest,
			errMessage: "test error",
		},
		"branch_with_long_key_length_write_error": {
			node: &Node{
				PartialKey: make([]byte, int(^branchVariant.mask)+1),
				Children:   make([]*Node, ChildrenCapacity),
			},
			writes: []writeCall{
				{
					written: []byte{branchVariant.bits | ^branchVariant.mask},
				},
				{
					written: []byte{0x01},
					err:     errTest,
				},
			},
			errWrapped: errTest,
			errMessage: "test error",
		},
		"leaf_with_hashed_value": {
			node: &Node{
				StorageValue: largeValue,
			},
			maxInlineValueSize: 32,
			writes: []writeCall{
				{written: []byte{leafWithHashedValueVariant.bits}},
			},
		},
		"leaf_with_no_key": {
			node: &Node{StorageValue: []byte{1}},
			writes: []writeCall{
				{written: []byte{leafVariant.bits}},
			},
			maxInlineValueSize: 32,
		},
		"leaf_with_key_of_length_30": {
			node: &Node{
				PartialKey: make([]byte, 30),
			},
			writes: []writeCall{
				{written: []byte{leafVariant.bits | 30}},
			},
		},
		"leaf_with_short_key_write_error": {
			node: &Node{
				PartialKey: make([]byte, 30),
			},
			writes: []writeCall{
				{
					written: []byte{leafVariant.bits | 30},
					err:     errTest,
				},
			},
			errWrapped: errTest,
			errMessage: "test error",
		},
		"leaf_with_key_of_length_62": {
			node: &Node{
				PartialKey: make([]byte, 62),
			},
			writes: []writeCall{
				{written: []byte{leafVariant.bits | 62}},
			},
		},
		"leaf_with_key_of_length_63": {
			node: &Node{
				PartialKey: make([]byte, 63),
			},
			writes: []writeCall{
				{written: []byte{leafVariant.bits | 63}},
				{written: []byte{0x0}},
			},
		},
		"leaf_with_key_of_length_64": {
			node: &Node{
				PartialKey: make([]byte, 64),
			},
			writes: []writeCall{
				{written: []byte{leafVariant.bits | 63}},
				{written: []byte{0x1}},
			},
		},
		"leaf_with_long_key_first_byte_write_error": {
			node: &Node{
				PartialKey: make([]byte, 63),
			},
			writes: []writeCall{
				{
					written: []byte{leafVariant.bits | 63},
					err:     errTest,
				},
			},
			errWrapped: errTest,
			errMessage: "test error",
		},
		"leaf_with_key_length_over_3_bytes": {
			node: &Node{
				PartialKey: make([]byte, int(^leafVariant.mask)+0b1111_1111+0b0000_0001),
			},
			writes: []writeCall{
				{written: []byte{leafVariant.bits | ^leafVariant.mask}},
				{written: []byte{0b1111_1111}},
				{written: []byte{0b0000_0001}},
			},
		},
		"leaf_with_key_length_over_3_bytes_and_last_byte_zero": {
			node: &Node{
				PartialKey: make([]byte, int(^leafVariant.mask)+0b1111_1111),
			},
			writes: []writeCall{
				{written: []byte{leafVariant.bits | ^leafVariant.mask}},
				{written: []byte{0b1111_1111}},
				{written: []byte{0x00}},
			},
		},
	}

	for name, testCase := range testCases {
		testCase := testCase
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			ctrl := gomock.NewController(t)

			writer := NewMockWriter(ctrl)
			var previousCall *gomock.Call
			for _, write := range testCase.writes {
				call := writer.EXPECT().
					Write(write.written).
					Return(write.n, write.err)

				if previousCall != nil {
					call.After(previousCall)
				}
				previousCall = call
			}

			err := encodeHeader(testCase.node, testCase.maxInlineValueSize, writer)

			assert.ErrorIs(t, err, testCase.errWrapped)
			if testCase.errWrapped != nil {
				assert.EqualError(t, err, testCase.errMessage)
			}
		})
	}

	t.Run("partial_key_length_is_too_big", func(t *testing.T) {
		t.Parallel()

		const keyLength = uint(maxPartialKeyLength) + 1
		node := &Node{
			PartialKey: make([]byte, keyLength),
		}

		assert.PanicsWithValue(t, "partial key length is too big: 65536", func() {
			_ = encodeHeader(node, 0, io.Discard)
		})
	})
}

func Test_encodeHeader_At_Maximum(t *testing.T) {
	t.Parallel()

	// Note: this test case cannot run with the
	// mock writer since it's too slow, so we use
	// an actual buffer.

	variant := leafVariant.bits
	const partialKeyLengthHeaderMask = 0b0011_1111
	const keyLength = uint(maxPartialKeyLength)
	extraKeyBytesNeeded := math.Ceil(float64(maxPartialKeyLength-partialKeyLengthHeaderMask) / 255.0)
	expectedEncodingLength := 1 + int(extraKeyBytesNeeded)

	lengthLeft := maxPartialKeyLength
	expectedBytes := make([]byte, expectedEncodingLength)
	expectedBytes[0] = variant | partialKeyLengthHeaderMask
	lengthLeft -= partialKeyLengthHeaderMask
	for i := 1; i < len(expectedBytes)-1; i++ {
		expectedBytes[i] = 255
		lengthLeft -= 255
	}
	expectedBytes[len(expectedBytes)-1] = byte(lengthLeft)

	buffer := bytes.NewBuffer(nil)
	buffer.Grow(expectedEncodingLength)

	node := &Node{
		PartialKey: make([]byte, keyLength),
	}

	err := encodeHeader(node, NoMaxInlineValueSize, buffer)

	require.NoError(t, err)
	assert.Equal(t, expectedBytes, buffer.Bytes())
}

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

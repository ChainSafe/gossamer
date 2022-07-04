// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package node

import (
	"bytes"
	"io"
	"math"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_encodeHeader(t *testing.T) {
	testCases := map[string]struct {
		node       *Node
		writes     []writeCall
		errWrapped error
		errMessage string
	}{
		"branch with no key": {
			node: &Node{
				Children: make([]*Node, ChildrenCapacity),
			},
			writes: []writeCall{
				{written: []byte{branchVariant.bits}},
			},
		},
		"branch with value": {
			node: &Node{
				Value:    []byte{},
				Children: make([]*Node, ChildrenCapacity),
			},
			writes: []writeCall{
				{written: []byte{branchWithValueVariant.bits}},
			},
		},
		"branch with key of length 30": {
			node: &Node{
				Key:      make([]byte, 30),
				Children: make([]*Node, ChildrenCapacity),
			},
			writes: []writeCall{
				{written: []byte{branchVariant.bits | 30}},
			},
		},
		"branch with key of length 62": {
			node: &Node{
				Key:      make([]byte, 62),
				Children: make([]*Node, ChildrenCapacity),
			},
			writes: []writeCall{
				{written: []byte{branchVariant.bits | 62}},
			},
		},
		"branch with key of length 63": {
			node: &Node{
				Key:      make([]byte, 63),
				Children: make([]*Node, ChildrenCapacity),
			},
			writes: []writeCall{
				{written: []byte{branchVariant.bits | 63}},
				{written: []byte{0x00}}, // trailing 0 to indicate the partial
				// key length is done here.
			},
		},
		"branch with key of length 64": {
			node: &Node{
				Key:      make([]byte, 64),
				Children: make([]*Node, ChildrenCapacity),
			},
			writes: []writeCall{
				{written: []byte{branchVariant.bits | 63}},
				{written: []byte{0x01}},
			},
		},
		"branch with small key length write error": {
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
		"branch with long key length write error": {
			node: &Node{
				Key:      make([]byte, int(^branchVariant.mask)+1),
				Children: make([]*Node, ChildrenCapacity),
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
		"leaf with no key": {
			node: &Node{Value: []byte{1}},
			writes: []writeCall{
				{written: []byte{leafVariant.bits}},
			},
		},
		"leaf with key of length 30": {
			node: &Node{
				Key: make([]byte, 30),
			},
			writes: []writeCall{
				{written: []byte{leafVariant.bits | 30}},
			},
		},
		"leaf with short key write error": {
			node: &Node{
				Key: make([]byte, 30),
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
		"leaf with key of length 62": {
			node: &Node{
				Key: make([]byte, 62),
			},
			writes: []writeCall{
				{written: []byte{leafVariant.bits | 62}},
			},
		},
		"leaf with key of length 63": {
			node: &Node{
				Key: make([]byte, 63),
			},
			writes: []writeCall{
				{written: []byte{leafVariant.bits | 63}},
				{written: []byte{0x0}},
			},
		},
		"leaf with key of length 64": {
			node: &Node{
				Key: make([]byte, 64),
			},
			writes: []writeCall{
				{written: []byte{leafVariant.bits | 63}},
				{written: []byte{0x1}},
			},
		},
		"leaf with long key first byte write error": {
			node: &Node{
				Key: make([]byte, 63),
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
		"leaf with key length over 3 bytes": {
			node: &Node{
				Key: make([]byte, int(^leafVariant.mask)+0b1111_1111+0b0000_0001),
			},
			writes: []writeCall{
				{written: []byte{leafVariant.bits | ^leafVariant.mask}},
				{written: []byte{0b1111_1111}},
				{written: []byte{0b0000_0001}},
			},
		},
		"leaf with key length over 3 bytes and last byte zero": {
			node: &Node{
				Key: make([]byte, int(^leafVariant.mask)+0b1111_1111),
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

			err := encodeHeader(testCase.node, writer)

			assert.ErrorIs(t, err, testCase.errWrapped)
			if testCase.errWrapped != nil {
				assert.EqualError(t, err, testCase.errMessage)
			}
		})
	}

	t.Run("partial key length is too big", func(t *testing.T) {
		t.Parallel()

		const keyLength = uint(maxPartialKeyLength) + 1
		node := &Node{
			Key: make([]byte, keyLength),
		}

		assert.PanicsWithValue(t, "partial key length is too big: 65536", func() {
			_ = encodeHeader(node, io.Discard)
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
		Key: make([]byte, keyLength),
	}

	err := encodeHeader(node, buffer)

	require.NoError(t, err)
	assert.Equal(t, expectedBytes, buffer.Bytes())
}

func Test_decodeHeader(t *testing.T) {
	testCases := map[string]struct {
		reads            []readCall
		nodeVariant      variant
		partialKeyLength uint16
		errWrapped       error
		errMessage       string
	}{
		"first byte read error": {
			reads: []readCall{
				{buffArgCap: 1, err: errTest},
			},
			errWrapped: errTest,
			errMessage: "reading header byte: test error",
		},
		"header byte decoding error": {
			reads: []readCall{
				{buffArgCap: 1, read: []byte{0b0011_1110}},
			},
			errWrapped: ErrVariantUnknown,
			errMessage: "decoding header byte: node variant is unknown: for header byte 00111110",
		},
		"partial key length contained in first byte": {
			reads: []readCall{
				{buffArgCap: 1, read: []byte{leafVariant.bits | 0b0011_1110}},
			},
			nodeVariant:      leafVariant,
			partialKeyLength: uint16(0b0011_1110),
		},
		"long partial key length and second byte read error": {
			reads: []readCall{
				{buffArgCap: 1, read: []byte{leafVariant.bits | 0b0011_1111}},
				{buffArgCap: 1, err: errTest},
			},
			errWrapped: errTest,
			errMessage: "reading key length: test error",
		},
		"partial key length spread on multiple bytes": {
			reads: []readCall{
				{buffArgCap: 1, read: []byte{leafVariant.bits | 0b0011_1111}},
				{buffArgCap: 1, read: []byte{0b1111_1111}},
				{buffArgCap: 1, read: []byte{0b1111_0000}},
			},
			nodeVariant:      leafVariant,
			partialKeyLength: uint16(0b0011_1111 + 0b1111_1111 + 0b1111_0000),
		},
		"partial key length too long": {
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
		"branch with value header": {
			header:                 0b1110_1001,
			nodeVariant:            branchWithValueVariant,
			partialKeyLengthHeader: 0b0010_1001,
		},
		"branch header": {
			header:                 0b1010_1001,
			nodeVariant:            branchVariant,
			partialKeyLengthHeader: 0b0010_1001,
		},
		"leaf header": {
			header:                 0b0110_1001,
			nodeVariant:            leafVariant,
			partialKeyLengthHeader: 0b0010_1001,
		},
		"unknown variant header": {
			header:     0b0000_0000,
			errWrapped: ErrVariantUnknown,
			errMessage: "node variant is unknown: for header byte 00000000",
		},
	}

	for name, testCase := range testCases {
		testCase := testCase
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			variant, partialKeyLengthHeader,
				err := decodeHeaderByte(testCase.header)

			assert.Equal(t, testCase.nodeVariant, variant)
			assert.Equal(t, testCase.partialKeyLengthHeader, partialKeyLengthHeader)
			assert.ErrorIs(t, err, testCase.errWrapped)
			if testCase.errWrapped != nil {
				assert.EqualError(t, err, testCase.errMessage)
			}
		})
	}
}

func Benchmark_decodeHeaderByte(b *testing.B) {
	// With global scoped variants slice:
	// 3.453 ns/op	       0 B/op	       0 allocs/op
	// With locally scoped variants slice:
	// 3.441 ns/op	       0 B/op	       0 allocs/op
	header := leafVariant.bits | 0b0000_0001
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _, _ = decodeHeaderByte(header)
	}
}

// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package node

import (
	"bytes"
	"fmt"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_Branch_GetKey(t *testing.T) {
	t.Parallel()

	branch := &Branch{
		Key: []byte{2},
	}
	key := branch.GetKey()
	assert.Equal(t, []byte{2}, key)
}

func Test_Leaf_GetKey(t *testing.T) {
	t.Parallel()

	leaf := &Leaf{
		Key: []byte{2},
	}
	key := leaf.GetKey()
	assert.Equal(t, []byte{2}, key)
}

func Test_Branch_SetKey(t *testing.T) {
	t.Parallel()

	branch := &Branch{
		Key: []byte{2},
	}
	branch.SetKey([]byte{3})
	assert.Equal(t, &Branch{Key: []byte{3}}, branch)
}

func Test_Leaf_SetKey(t *testing.T) {
	t.Parallel()

	leaf := &Leaf{
		Key: []byte{2},
	}
	leaf.SetKey([]byte{3})
	assert.Equal(t, &Leaf{Key: []byte{3}}, leaf)
}

func repeatBytes(n int, b byte) (slice []byte) {
	slice = make([]byte, n)
	for i := range slice {
		slice[i] = b
	}
	return slice
}

func Test_encodeKeyLength(t *testing.T) {
	t.Parallel()

	testCases := map[string]struct {
		keyLength  int
		writes     []writeCall
		errWrapped error
		errMessage string
	}{
		"length equal to maximum": {
			keyLength:  int(maxPartialKeySize) + 63,
			errWrapped: ErrPartialKeyTooBig,
			errMessage: "partial key length cannot be " +
				"larger than or equal to 2^16: 65535",
		},
		"zero length": {
			writes: []writeCall{
				{
					written: []byte{0xc1},
				},
			},
		},
		"one length": {
			keyLength: 1,
			writes: []writeCall{
				{
					written: []byte{0xc2},
				},
			},
		},
		"error at single byte write": {
			keyLength: 1,
			writes: []writeCall{
				{
					written: []byte{0xc2},
					err:     errTest,
				},
			},
			errWrapped: errTest,
			errMessage: errTest.Error(),
		},
		"error at first byte write": {
			keyLength: 255 + 100 + 63,
			writes: []writeCall{
				{
					written: []byte{255},
					err:     errTest,
				},
			},
			errWrapped: errTest,
			errMessage: errTest.Error(),
		},
		"error at last byte write": {
			keyLength: 255 + 100 + 63,
			writes: []writeCall{
				{
					written: []byte{255},
				},
				{
					written: []byte{100},
					err:     errTest,
				},
			},
			errWrapped: errTest,
			errMessage: errTest.Error(),
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

				if write.err != nil {
					break
				} else if previousCall != nil {
					call.After(previousCall)
				}
				previousCall = call
			}

			err := encodeKeyLength(testCase.keyLength, writer)

			assert.ErrorIs(t, err, testCase.errWrapped)
			if testCase.errWrapped != nil {
				assert.EqualError(t, err, testCase.errMessage)
			}
		})
	}

	t.Run("length at maximum", func(t *testing.T) {
		t.Parallel()

		// Note: this test case cannot run with the
		// mock writer since it's too slow, so we use
		// an actual buffer.

		const keyLength = int(maxPartialKeySize) + 62
		const expectedEncodingLength = 257
		expectedBytes := make([]byte, expectedEncodingLength)
		for i := 0; i < len(expectedBytes)-1; i++ {
			expectedBytes[i] = 255
		}
		expectedBytes[len(expectedBytes)-1] = 254

		buffer := bytes.NewBuffer(nil)
		buffer.Grow(expectedEncodingLength)

		err := encodeKeyLength(keyLength, buffer)

		require.NoError(t, err)
		assert.Equal(t, expectedBytes, buffer.Bytes())
	})
}

//go:generate mockgen -destination=reader_mock_test.go -package $GOPACKAGE io Reader

type readCall struct {
	buffArgCap int
	read       []byte
	n          int // number of bytes read
	err        error
}

func repeatReadCalls(rc readCall, length int) (readCalls []readCall) {
	readCalls = make([]readCall, length)
	for i := range readCalls {
		readCalls[i] = readCall{
			buffArgCap: rc.buffArgCap,
			n:          rc.n,
			err:        rc.err,
		}
		if rc.read != nil {
			readCalls[i].read = make([]byte, len(rc.read))
			copy(readCalls[i].read, rc.read)
		}
	}
	return readCalls
}

var _ gomock.Matcher = (*byteSliceCapMatcher)(nil)

type byteSliceCapMatcher struct {
	capacity int
}

func (b *byteSliceCapMatcher) Matches(x interface{}) bool {
	slice, ok := x.([]byte)
	if !ok {
		return false
	}
	return cap(slice) == b.capacity
}

func (b *byteSliceCapMatcher) String() string {
	return fmt.Sprintf("capacity of slice is not the expected capacity %d", b.capacity)
}

func newByteSliceCapMatcher(capacity int) *byteSliceCapMatcher {
	return &byteSliceCapMatcher{
		capacity: capacity,
	}
}

func Test_decodeKey(t *testing.T) {
	t.Parallel()

	testCases := map[string]struct {
		reads      []readCall
		keyLength  byte
		b          []byte
		errWrapped error
		errMessage string
	}{
		"zero key length": {
			b: []byte{},
		},
		"short key length": {
			reads: []readCall{
				{buffArgCap: 3, read: []byte{1, 2, 3}, n: 3},
			},
			keyLength: 5,
			b:         []byte{0x1, 0x0, 0x2, 0x0, 0x3},
		},
		"key read error": {
			reads: []readCall{
				{buffArgCap: 3, err: errTest},
			},
			keyLength:  5,
			errWrapped: ErrReadKeyData,
			errMessage: "cannot read key data: test error",
		},

		"key read bytes count mismatch": {
			reads: []readCall{
				{buffArgCap: 3, n: 2},
			},
			keyLength:  5,
			errWrapped: ErrReadKeyData,
			errMessage: "cannot read key data: read 2 bytes instead of 3",
		},
		"long key length": {
			reads: []readCall{
				{buffArgCap: 1, read: []byte{6}, n: 1},            // key length
				{buffArgCap: 35, read: repeatBytes(35, 7), n: 35}, // key data
			},
			keyLength: 0x3f,
			b: []byte{
				0x0, 0x7, 0x0, 0x7, 0x0, 0x7, 0x0, 0x7, 0x0, 0x7,
				0x0, 0x7, 0x0, 0x7, 0x0, 0x7, 0x0, 0x7, 0x0, 0x7,
				0x0, 0x7, 0x0, 0x7, 0x0, 0x7, 0x0, 0x7, 0x0, 0x7,
				0x0, 0x7, 0x0, 0x7, 0x0, 0x7, 0x0, 0x7, 0x0, 0x7,
				0x0, 0x7, 0x0, 0x7, 0x0, 0x7, 0x0, 0x7, 0x0, 0x7,
				0x0, 0x7, 0x0, 0x7, 0x0, 0x7, 0x0, 0x7, 0x0, 0x7,
				0x0, 0x7, 0x0, 0x7, 0x0, 0x7, 0x0, 0x7, 0x0, 0x7},
		},
		"key length read error": {
			reads: []readCall{
				{buffArgCap: 1, err: errTest},
			},
			keyLength:  0x3f,
			errWrapped: ErrReadKeyLength,
			errMessage: "cannot read key length: test error",
		},
		"key length too big": {
			reads:      repeatReadCalls(readCall{buffArgCap: 1, read: []byte{0xff}, n: 1}, 257),
			keyLength:  0x3f,
			errWrapped: ErrPartialKeyTooBig,
			errMessage: "partial key length cannot be larger than or equal to 2^16: 65598",
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

			b, err := decodeKey(reader, testCase.keyLength)

			assert.ErrorIs(t, err, testCase.errWrapped)
			if err != nil {
				assert.EqualError(t, err, testCase.errMessage)
			}
			assert.Equal(t, testCase.b, b)
		})
	}
}

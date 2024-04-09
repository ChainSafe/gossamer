// Copyright 2024 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package codec

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

func repeatBytes(b byte, n int) (slice []byte) {
	slice = make([]byte, n)
	for i := range slice {
		slice[i] = b
	}
	return slice
}

type readCall struct {
	buffArgCap int
	read       []byte
	n          int // number of bytes read
	err        error
}

func repeatReadCall(base readCall, n int) (calls []readCall) {
	calls = make([]readCall, n)
	for i := range calls {
		calls[i] = base
	}
	return calls
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
	return fmt.Sprintf("slice with capacity %d", b.capacity)
}

func newByteSliceCapMatcher(capacity int) *byteSliceCapMatcher {
	return &byteSliceCapMatcher{
		capacity: capacity,
	}
}

func Test_decodeKey(t *testing.T) {
	t.Parallel()

	testCases := map[string]struct {
		reads            []readCall
		partialKeyLength uint16
		b                []byte
		errWrapped       error
		errMessage       string
	}{
		"zero_key_length": {
			partialKeyLength: 0,
			b:                []byte{},
		},
		"short_key_length": {
			reads: []readCall{
				{buffArgCap: 3, read: []byte{1, 2, 3}, n: 3},
			},
			partialKeyLength: 5,
			b:                []byte{0x1, 0x0, 0x2, 0x0, 0x3},
		},
		"key_read_error": {
			reads: []readCall{
				{buffArgCap: 3, err: errTest},
			},
			partialKeyLength: 5,
			errWrapped:       errTest,
			errMessage:       "reading from reader: test error",
		},

		"key_read_bytes_count_mismatch": {
			reads: []readCall{
				{buffArgCap: 3, n: 2},
			},
			partialKeyLength: 5,
			errWrapped:       ErrReaderMismatchCount,
			errMessage:       "read unexpected number of bytes from reader: read 2 bytes instead of expected 3 bytes",
		},
		"long_key_length": {
			reads: []readCall{
				{buffArgCap: 35, read: repeatBytes(7, 35), n: 35}, // key data
			},
			partialKeyLength: 70,
			b: []byte{
				0x0, 0x7, 0x0, 0x7, 0x0, 0x7, 0x0, 0x7, 0x0, 0x7,
				0x0, 0x7, 0x0, 0x7, 0x0, 0x7, 0x0, 0x7, 0x0, 0x7,
				0x0, 0x7, 0x0, 0x7, 0x0, 0x7, 0x0, 0x7, 0x0, 0x7,
				0x0, 0x7, 0x0, 0x7, 0x0, 0x7, 0x0, 0x7, 0x0, 0x7,
				0x0, 0x7, 0x0, 0x7, 0x0, 0x7, 0x0, 0x7, 0x0, 0x7,
				0x0, 0x7, 0x0, 0x7, 0x0, 0x7, 0x0, 0x7, 0x0, 0x7,
				0x0, 0x7, 0x0, 0x7, 0x0, 0x7, 0x0, 0x7, 0x0, 0x7},
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

			b, err := decodeKey(reader, testCase.partialKeyLength)

			assert.ErrorIs(t, err, testCase.errWrapped)
			if err != nil {
				assert.EqualError(t, err, testCase.errMessage)
			}
			assert.Equal(t, testCase.b, b)
		})
	}
}

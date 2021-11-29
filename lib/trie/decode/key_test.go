// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package decode

import (
	"bytes"
	"io"
	"testing"

	"github.com/stretchr/testify/assert"
)

func repeatBytes(n int, b byte) (slice []byte) {
	slice = make([]byte, n)
	for i := range slice {
		slice[i] = b
	}
	return slice
}

func Test_Key(t *testing.T) {
	t.Parallel()

	testCases := map[string]struct {
		reader     io.Reader
		keyLength  byte
		b          []byte
		errWrapped error
		errMessage string
	}{
		"zero key length": {
			b: []byte{},
		},
		"short key length": {
			reader:    bytes.NewBuffer([]byte{1, 2, 3}),
			keyLength: 5,
			b:         []byte{0x1, 0x0, 0x2, 0x0, 0x3},
		},
		"key read error": {
			reader:     bytes.NewBuffer(nil),
			keyLength:  5,
			errWrapped: ErrReadKeyData,
			errMessage: "cannot read key data: EOF",
		},
		"long key length": {
			reader: bytes.NewBuffer(
				append(
					[]byte{
						6, // key length
					},
					repeatBytes(64, 7)..., // key data
				)),
			keyLength: 0x3f,
			b: []byte{
				0x7, 0x0, 0x7, 0x0, 0x7, 0x0, 0x7, 0x0, 0x7, 0x0,
				0x7, 0x0, 0x7, 0x0, 0x7, 0x0, 0x7, 0x0, 0x7, 0x0,
				0x7, 0x0, 0x7, 0x0, 0x7, 0x0, 0x7, 0x0, 0x7, 0x0,
				0x7, 0x0, 0x7, 0x0, 0x7, 0x0, 0x7, 0x0, 0x7, 0x0,
				0x7, 0x0, 0x7, 0x0, 0x7, 0x0, 0x7, 0x0, 0x7, 0x0,
				0x7, 0x0, 0x7, 0x0, 0x7, 0x0, 0x7, 0x0, 0x7, 0x0,
				0x7, 0x0, 0x7, 0x0, 0x7, 0x0, 0x7, 0x0, 0x7},
		},
		"key length read error": {
			reader:     bytes.NewBuffer(nil),
			keyLength:  0x3f,
			errWrapped: ErrReadKeyLength,
			errMessage: "cannot read key length: EOF",
		},
		"key length too big": {
			reader:     bytes.NewBuffer(repeatBytes(257, 0xff)),
			keyLength:  0x3f,
			errWrapped: ErrPartialKeyTooBig,
			errMessage: "partial key length cannot be larger than or equal to 2^16: 65598",
		},
	}

	for name, testCase := range testCases {
		testCase := testCase
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			b, err := Key(testCase.reader, testCase.keyLength)

			assert.ErrorIs(t, err, testCase.errWrapped)
			if err != nil {
				assert.EqualError(t, err, testCase.errMessage)
			}
			assert.Equal(t, testCase.b, b)
		})
	}
}

func Test_KeyToNibbles(t *testing.T) {
	t.Parallel()

	testCases := map[string]struct {
		in      []byte
		nibbles []byte
	}{
		"nil input": {
			nibbles: []byte{},
		},
		"empty input": {
			in:      []byte{},
			nibbles: []byte{},
		},
		"0x0": {
			in:      []byte{0x0},
			nibbles: []byte{0, 0}},
		"0xFF": {
			in:      []byte{0xFF},
			nibbles: []byte{0xF, 0xF}},
		"0x3a 0x05": {
			in:      []byte{0x3a, 0x05},
			nibbles: []byte{0x3, 0xa, 0x0, 0x5}},
		"0xAA 0xFF 0x01": {
			in:      []byte{0xAA, 0xFF, 0x01},
			nibbles: []byte{0xa, 0xa, 0xf, 0xf, 0x0, 0x1}},
		"0xAA 0xFF 0x01 0xc2": {
			in:      []byte{0xAA, 0xFF, 0x01, 0xc2},
			nibbles: []byte{0xa, 0xa, 0xf, 0xf, 0x0, 0x1, 0xc, 0x2}},
		"0xAA 0xFF 0x01 0xc0": {
			in:      []byte{0xAA, 0xFF, 0x01, 0xc0},
			nibbles: []byte{0xa, 0xa, 0xf, 0xf, 0x0, 0x1, 0xc, 0x0}},
	}

	for name, testCase := range testCases {
		testCase := testCase
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			nibbles := KeyLEToNibbles(testCase.in)

			assert.Equal(t, testCase.nibbles, nibbles)
		})
	}
}

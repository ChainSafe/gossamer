// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package encode

import (
	"bytes"
	"errors"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type writeCall struct {
	written []byte
	n       int
	err     error
}

var errTest = errors.New("test error")

//go:generate mockgen -destination=writer_mock_test.go -package $GOPACKAGE io Writer

func Test_KeyLength(t *testing.T) {
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

			err := KeyLength(testCase.keyLength, writer)

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

		err := KeyLength(keyLength, buffer)

		require.NoError(t, err)
		assert.Equal(t, expectedBytes, buffer.Bytes())
	})
}

func Test_NibblesToKeyLE(t *testing.T) {
	t.Parallel()

	testCases := map[string]struct {
		nibbles []byte
		keyLE   []byte
	}{
		"nil nibbles": {
			keyLE: []byte{},
		},
		"empty nibbles": {
			nibbles: []byte{},
			keyLE:   []byte{},
		},
		"0xF 0xF": {
			nibbles: []byte{0xF, 0xF},
			keyLE:   []byte{0xFF},
		},
		"0x3 0xa 0x0 0x5": {
			nibbles: []byte{0x3, 0xa, 0x0, 0x5},
			keyLE:   []byte{0x3a, 0x05},
		},
		"0xa 0xa 0xf 0xf 0x0 0x1": {
			nibbles: []byte{0xa, 0xa, 0xf, 0xf, 0x0, 0x1},
			keyLE:   []byte{0xaa, 0xff, 0x01},
		},
		"0xa 0xa 0xf 0xf 0x0 0x1 0xc 0x2": {
			nibbles: []byte{0xa, 0xa, 0xf, 0xf, 0x0, 0x1, 0xc, 0x2},
			keyLE:   []byte{0xaa, 0xff, 0x01, 0xc2},
		},
		"0xa 0xa 0xf 0xf 0x0 0x1 0xc": {
			nibbles: []byte{0xa, 0xa, 0xf, 0xf, 0x0, 0x1, 0xc},
			keyLE:   []byte{0xa, 0xaf, 0xf0, 0x1c},
		},
	}

	for name, testCase := range testCases {
		testCase := testCase
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			keyLE := NibblesToKeyLE(testCase.nibbles)

			assert.Equal(t, testCase.keyLE, keyLE)
		})
	}
}

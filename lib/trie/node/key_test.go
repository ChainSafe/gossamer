package node

import (
	"bytes"
	"io"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

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

func Test_decodeKey(t *testing.T) {
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

			b, err := decodeKey(testCase.reader, testCase.keyLength)

			assert.ErrorIs(t, err, testCase.errWrapped)
			if err != nil {
				assert.EqualError(t, err, testCase.errMessage)
			}
			assert.Equal(t, testCase.b, b)
		})
	}
}

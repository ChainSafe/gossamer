// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package leaf

import (
	"errors"
	"testing"

	"github.com/ChainSafe/gossamer/lib/trie/encode"
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

//go:generate mockgen -destination=buffer_mock_test.go -package $GOPACKAGE github.com/ChainSafe/gossamer/lib/trie/encode Buffer

func Test_Leaf_Encode(t *testing.T) {
	t.Parallel()

	testCases := map[string]struct {
		leaf             *Leaf
		writes           []writeCall
		bufferLenCall    bool
		bufferBytesCall  bool
		bufferBytes      []byte
		expectedEncoding []byte
		wrappedErr       error
		errMessage       string
	}{
		"clean leaf with encoding": {
			leaf: &Leaf{
				Encoding: []byte{1, 2, 3},
			},
			writes: []writeCall{
				{
					written: []byte{1, 2, 3},
				},
			},
			expectedEncoding: []byte{1, 2, 3},
		},
		"write error for clean leaf with encoding": {
			leaf: &Leaf{
				Encoding: []byte{1, 2, 3},
			},
			writes: []writeCall{
				{
					written: []byte{1, 2, 3},
					err:     errTest,
				},
			},
			expectedEncoding: []byte{1, 2, 3},
			wrappedErr:       errTest,
			errMessage:       "cannot write stored encoding to buffer: test error",
		},
		"header encoding error": {
			leaf: &Leaf{
				Key: make([]byte, 63+(1<<16)),
			},
			wrappedErr: encode.ErrPartialKeyTooBig,
			errMessage: "cannot encode header: partial key length cannot be larger than or equal to 2^16: 65536",
		},
		"buffer write error for encoded header": {
			leaf: &Leaf{
				Key: []byte{1, 2, 3},
			},
			writes: []writeCall{
				{
					written: []byte{67},
					err:     errTest,
				},
			},
			wrappedErr: errTest,
			errMessage: "cannot write encoded header to buffer: test error",
		},
		"buffer write error for encoded key": {
			leaf: &Leaf{
				Key: []byte{1, 2, 3},
			},
			writes: []writeCall{
				{
					written: []byte{67},
				},
				{
					written: []byte{1, 35},
					err:     errTest,
				},
			},
			wrappedErr: errTest,
			errMessage: "cannot write LE key to buffer: test error",
		},
		"buffer write error for encoded value": {
			leaf: &Leaf{
				Key:   []byte{1, 2, 3},
				Value: []byte{4, 5, 6},
			},
			writes: []writeCall{
				{
					written: []byte{67},
				},
				{
					written: []byte{1, 35},
				},
				{
					written: []byte{12, 4, 5, 6},
					err:     errTest,
				},
			},
			wrappedErr: errTest,
			errMessage: "cannot write scale encoded value to buffer: test error",
		},
		"success": {
			leaf: &Leaf{
				Key:   []byte{1, 2, 3},
				Value: []byte{4, 5, 6},
			},
			writes: []writeCall{
				{
					written: []byte{67},
				},
				{
					written: []byte{1, 35},
				},
				{
					written: []byte{12, 4, 5, 6},
				},
			},
			bufferLenCall:    true,
			bufferBytesCall:  true,
			bufferBytes:      []byte{1, 2, 3},
			expectedEncoding: []byte{1, 2, 3},
		},
	}

	for name, testCase := range testCases {
		testCase := testCase
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			ctrl := gomock.NewController(t)

			buffer := NewMockBuffer(ctrl)
			var previousCall *gomock.Call
			for _, write := range testCase.writes {
				call := buffer.EXPECT().
					Write(write.written).
					Return(write.n, write.err)

				if previousCall != nil {
					call.After(previousCall)
				}
				previousCall = call
			}
			if testCase.bufferLenCall {
				buffer.EXPECT().Len().Return(len(testCase.bufferBytes))
			}
			if testCase.bufferBytesCall {
				buffer.EXPECT().Bytes().Return(testCase.bufferBytes)
			}

			err := testCase.leaf.Encode(buffer)

			if testCase.wrappedErr != nil {
				assert.ErrorIs(t, err, testCase.wrappedErr)
				assert.EqualError(t, err, testCase.errMessage)
			} else {
				require.NoError(t, err)
			}
			assert.Equal(t, testCase.expectedEncoding, testCase.leaf.Encoding)
		})
	}
}

func Test_Leaf_ScaleEncodeHash(t *testing.T) {
	t.Parallel()

	testCases := map[string]struct {
		leaf       *Leaf
		b          []byte
		wrappedErr error
		errMessage string
	}{
		"leaf": {
			leaf: &Leaf{},
			b:    []byte{0x8, 0x40, 0},
		},
	}

	for name, testCase := range testCases {
		testCase := testCase
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			b, err := testCase.leaf.ScaleEncodeHash()

			if testCase.wrappedErr != nil {
				assert.ErrorIs(t, err, testCase.wrappedErr)
				assert.EqualError(t, err, testCase.errMessage)
			} else {
				require.NoError(t, err)
			}

			assert.Equal(t, testCase.b, b)
		})
	}
}

//go:generate mockgen -destination=writer_mock_test.go -package $GOPACKAGE io Writer

func Test_Leaf_hash(t *testing.T) {
	t.Parallel()

	testCases := map[string]struct {
		leaf       *Leaf
		writeCall  bool
		write      writeCall
		wrappedErr error
		errMessage string
	}{
		"small leaf buffer write error": {
			leaf: &Leaf{
				Encoding: []byte{1, 2, 3},
			},
			writeCall: true,
			write: writeCall{
				written: []byte{1, 2, 3},
				err:     errTest,
			},
			wrappedErr: errTest,
			errMessage: "cannot write encoded leaf to buffer: " +
				"test error",
		},
		"small leaf success": {
			leaf: &Leaf{
				Encoding: []byte{1, 2, 3},
			},
			writeCall: true,
			write: writeCall{
				written: []byte{1, 2, 3},
			},
		},
		"leaf hash sum buffer write error": {
			leaf: &Leaf{
				Encoding: []byte{
					1, 2, 3, 4, 5, 6, 7, 8,
					1, 2, 3, 4, 5, 6, 7, 8,
					1, 2, 3, 4, 5, 6, 7, 8,
					1, 2, 3, 4, 5, 6, 7, 8,
					1, 2, 3, 4, 5, 6, 7, 8,
				},
			},
			writeCall: true,
			write: writeCall{
				written: []byte{
					107, 105, 154, 175, 253, 170, 232,
					135, 240, 21, 207, 148, 82, 117,
					249, 230, 80, 197, 254, 17, 149,
					108, 50, 7, 80, 56, 114, 176,
					84, 114, 125, 234},
				err: errTest,
			},
			wrappedErr: errTest,
			errMessage: "cannot write hash sum of leaf to buffer: " +
				"test error",
		},
		"leaf hash sum success": {
			leaf: &Leaf{
				Encoding: []byte{
					1, 2, 3, 4, 5, 6, 7, 8,
					1, 2, 3, 4, 5, 6, 7, 8,
					1, 2, 3, 4, 5, 6, 7, 8,
					1, 2, 3, 4, 5, 6, 7, 8,
					1, 2, 3, 4, 5, 6, 7, 8,
				},
			},
			writeCall: true,
			write: writeCall{
				written: []byte{
					107, 105, 154, 175, 253, 170, 232,
					135, 240, 21, 207, 148, 82, 117,
					249, 230, 80, 197, 254, 17, 149,
					108, 50, 7, 80, 56, 114, 176,
					84, 114, 125, 234},
			},
		},
	}

	for name, testCase := range testCases {
		testCase := testCase
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			ctrl := gomock.NewController(t)

			writer := NewMockWriter(ctrl)
			if testCase.writeCall {
				writer.EXPECT().
					Write(testCase.write.written).
					Return(testCase.write.n, testCase.write.err)
			}

			err := testCase.leaf.hash(writer)

			if testCase.wrappedErr != nil {
				assert.ErrorIs(t, err, testCase.wrappedErr)
				assert.EqualError(t, err, testCase.errMessage)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

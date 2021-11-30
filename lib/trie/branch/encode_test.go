// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package branch

import (
	"errors"
	"testing"

	"github.com/ChainSafe/gossamer/lib/trie/encode"
	"github.com/ChainSafe/gossamer/lib/trie/leaf"
	"github.com/ChainSafe/gossamer/lib/trie/node"
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

func Test_Branch_Encode(t *testing.T) {
	t.Parallel()

	testCases := map[string]struct {
		branch     *Branch
		writes     []writeCall
		wrappedErr error
		errMessage string
	}{
		"clean branch with encoding": {
			branch: &Branch{
				Encoding: []byte{1, 2, 3},
			},
			writes: []writeCall{
				{ // stored encoding
					written: []byte{1, 2, 3},
				},
			},
		},
		"write error for clean branch with encoding": {
			branch: &Branch{
				Encoding: []byte{1, 2, 3},
			},
			writes: []writeCall{
				{ // stored encoding
					written: []byte{1, 2, 3},
					err:     errTest,
				},
			},
			wrappedErr: errTest,
			errMessage: "cannot write stored encoding to buffer: test error",
		},
		"header encoding error": {
			branch: &Branch{
				Key: make([]byte, 63+(1<<16)),
			},
			wrappedErr: encode.ErrPartialKeyTooBig,
			errMessage: "cannot encode header: partial key length cannot be larger than or equal to 2^16: 65536",
		},
		"buffer write error for encoded header": {
			branch: &Branch{
				Key:   []byte{1, 2, 3},
				Value: []byte{100},
			},
			writes: []writeCall{
				{ // header
					written: []byte{195},
					err:     errTest,
				},
			},
			wrappedErr: errTest,
			errMessage: "cannot write encoded header to buffer: test error",
		},
		"buffer write error for encoded key": {
			branch: &Branch{
				Key:   []byte{1, 2, 3},
				Value: []byte{100},
			},
			writes: []writeCall{
				{ // header
					written: []byte{195},
				},
				{ // key LE
					written: []byte{1, 35},
					err:     errTest,
				},
			},
			wrappedErr: errTest,
			errMessage: "cannot write encoded key to buffer: test error",
		},
		"buffer write error for children bitmap": {
			branch: &Branch{
				Key:   []byte{1, 2, 3},
				Value: []byte{100},
				Children: [16]node.Node{
					nil, nil, nil, &leaf.Leaf{Key: []byte{9}},
					nil, nil, nil, &leaf.Leaf{Key: []byte{11}},
				},
			},
			writes: []writeCall{
				{ // header
					written: []byte{195},
				},
				{ // key LE
					written: []byte{1, 35},
				},
				{ // children bitmap
					written: []byte{136, 0},
					err:     errTest,
				},
			},
			wrappedErr: errTest,
			errMessage: "cannot write children bitmap to buffer: test error",
		},
		"buffer write error for value": {
			branch: &Branch{
				Key:   []byte{1, 2, 3},
				Value: []byte{100},
				Children: [16]node.Node{
					nil, nil, nil, &leaf.Leaf{Key: []byte{9}},
					nil, nil, nil, &leaf.Leaf{Key: []byte{11}},
				},
			},
			writes: []writeCall{
				{ // header
					written: []byte{195},
				},
				{ // key LE
					written: []byte{1, 35},
				},
				{ // children bitmap
					written: []byte{136, 0},
				},
				{ // value
					written: []byte{4, 100},
					err:     errTest,
				},
			},
			wrappedErr: errTest,
			errMessage: "cannot write encoded value to buffer: test error",
		},
		"buffer write error for children encoded sequentially": {
			branch: &Branch{
				Key:   []byte{1, 2, 3},
				Value: []byte{100},
				Children: [16]node.Node{
					nil, nil, nil, &leaf.Leaf{Key: []byte{9}},
					nil, nil, nil, &leaf.Leaf{Key: []byte{11}},
				},
			},
			writes: []writeCall{
				{ // header
					written: []byte{195},
				},
				{ // key LE
					written: []byte{1, 35},
				},
				{ // children bitmap
					written: []byte{136, 0},
				},
				{ // value
					written: []byte{4, 100},
				},
				{ // children
					written: []byte{12, 65, 9, 0},
					err:     errTest,
				},
			},
			wrappedErr: errTest,
			errMessage: "cannot encode children of branch: " +
				"cannot encode child at index 3: " +
				"failed to write child to buffer: test error",
		},
		"success with sequential children encoding": {
			branch: &Branch{
				Key:   []byte{1, 2, 3},
				Value: []byte{100},
				Children: [16]node.Node{
					nil, nil, nil, &leaf.Leaf{Key: []byte{9}},
					nil, nil, nil, &leaf.Leaf{Key: []byte{11}},
				},
			},
			writes: []writeCall{
				{ // header
					written: []byte{195},
				},
				{ // key LE
					written: []byte{1, 35},
				},
				{ // children bitmap
					written: []byte{136, 0},
				},
				{ // value
					written: []byte{4, 100},
				},
				{ // first children
					written: []byte{12, 65, 9, 0},
				},
				{ // second children
					written: []byte{12, 65, 11, 0},
				},
			},
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

			err := testCase.branch.Encode(buffer)

			if testCase.wrappedErr != nil {
				assert.ErrorIs(t, err, testCase.wrappedErr)
				assert.EqualError(t, err, testCase.errMessage)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func Test_encodeChildrenInParallel(t *testing.T) {
	t.Parallel()

	testCases := map[string]struct {
		children   [16]node.Node
		writes     []writeCall
		wrappedErr error
		errMessage string
	}{
		"no children": {},
		"first child not nil": {
			children: [16]node.Node{
				&leaf.Leaf{Key: []byte{1}},
			},
			writes: []writeCall{
				{
					written: []byte{12, 65, 1, 0},
				},
			},
		},
		"last child not nil": {
			children: [16]node.Node{
				nil, nil, nil, nil, nil,
				nil, nil, nil, nil, nil,
				nil, nil, nil, nil, nil,
				&leaf.Leaf{Key: []byte{1}},
			},
			writes: []writeCall{
				{
					written: []byte{12, 65, 1, 0},
				},
			},
		},
		"first two children not nil": {
			children: [16]node.Node{
				&leaf.Leaf{Key: []byte{1}},
				&leaf.Leaf{Key: []byte{2}},
			},
			writes: []writeCall{
				{
					written: []byte{12, 65, 1, 0},
				},
				{
					written: []byte{12, 65, 2, 0},
				},
			},
		},
		"encoding error": {
			children: [16]node.Node{
				nil, nil, nil, nil,
				nil, nil, nil, nil,
				nil, nil, nil,
				&leaf.Leaf{
					Key: []byte{1},
				},
				nil, nil, nil, nil,
			},
			writes: []writeCall{
				{
					written: []byte{12, 65, 1, 0},
					err:     errTest,
				},
			},
			wrappedErr: errTest,
			errMessage: "cannot write encoding of child at index 11: " +
				"test error",
		},
	}

	for name, testCase := range testCases {
		testCase := testCase
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			ctrl := gomock.NewController(t)

			buffer := NewMockWriter(ctrl)
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

			err := encodeChildrenInParallel(testCase.children, buffer)

			if testCase.wrappedErr != nil {
				assert.ErrorIs(t, err, testCase.wrappedErr)
				assert.EqualError(t, err, testCase.errMessage)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func Test_encodeChildrenSequentially(t *testing.T) {
	t.Parallel()

	testCases := map[string]struct {
		children   [16]node.Node
		writes     []writeCall
		wrappedErr error
		errMessage string
	}{
		"no children": {},
		"first child not nil": {
			children: [16]node.Node{
				&leaf.Leaf{Key: []byte{1}},
			},
			writes: []writeCall{
				{
					written: []byte{12, 65, 1, 0},
				},
			},
		},
		"last child not nil": {
			children: [16]node.Node{
				nil, nil, nil, nil, nil,
				nil, nil, nil, nil, nil,
				nil, nil, nil, nil, nil,
				&leaf.Leaf{Key: []byte{1}},
			},
			writes: []writeCall{
				{
					written: []byte{12, 65, 1, 0},
				},
			},
		},
		"first two children not nil": {
			children: [16]node.Node{
				&leaf.Leaf{Key: []byte{1}},
				&leaf.Leaf{Key: []byte{2}},
			},
			writes: []writeCall{
				{
					written: []byte{12, 65, 1, 0},
				},
				{
					written: []byte{12, 65, 2, 0},
				},
			},
		},
		"encoding error": {
			children: [16]node.Node{
				nil, nil, nil, nil,
				nil, nil, nil, nil,
				nil, nil, nil,
				&leaf.Leaf{
					Key: []byte{1},
				},
				nil, nil, nil, nil,
			},
			writes: []writeCall{
				{
					written: []byte{12, 65, 1, 0},
					err:     errTest,
				},
			},
			wrappedErr: errTest,
			errMessage: "cannot encode child at index 11: " +
				"failed to write child to buffer: test error",
		},
	}

	for name, testCase := range testCases {
		testCase := testCase
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			ctrl := gomock.NewController(t)

			buffer := NewMockWriter(ctrl)
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

			err := encodeChildrenSequentially(testCase.children, buffer)

			if testCase.wrappedErr != nil {
				assert.ErrorIs(t, err, testCase.wrappedErr)
				assert.EqualError(t, err, testCase.errMessage)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

//go:generate mockgen -destination=writer_mock_test.go -package $GOPACKAGE io Writer

func Test_encodeChild(t *testing.T) {
	t.Parallel()

	testCases := map[string]struct {
		child      node.Node
		writeCall  bool
		write      writeCall
		wrappedErr error
		errMessage string
	}{
		"nil node": {},
		"nil leaf": {
			child: (*leaf.Leaf)(nil),
		},
		"nil branch": {
			child: (*Branch)(nil),
		},
		"empty leaf child": {
			child:     &leaf.Leaf{},
			writeCall: true,
			write: writeCall{
				written: []byte{8, 64, 0},
			},
		},
		"empty branch child": {
			child:     &Branch{},
			writeCall: true,
			write: writeCall{
				written: []byte{12, 128, 0, 0},
			},
		},
		"buffer write error": {
			child:     &Branch{},
			writeCall: true,
			write: writeCall{
				written: []byte{12, 128, 0, 0},
				err:     errTest,
			},
			wrappedErr: errTest,
			errMessage: "failed to write child to buffer: test error",
		},
		"leaf child": {
			child: &leaf.Leaf{
				Key:   []byte{1},
				Value: []byte{2},
			},
			writeCall: true,
			write: writeCall{
				written: []byte{16, 65, 1, 4, 2},
			},
		},
		"branch child": {
			child: &Branch{
				Key:   []byte{1},
				Value: []byte{2},
				Children: [16]node.Node{
					nil, nil, &leaf.Leaf{
						Key:   []byte{5},
						Value: []byte{6},
					},
				},
			},
			writeCall: true,
			write: writeCall{
				written: []byte{44, 193, 1, 4, 0, 4, 2, 16, 65, 5, 4, 6},
			},
		},
	}

	for name, testCase := range testCases {
		testCase := testCase
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			ctrl := gomock.NewController(t)

			buffer := NewMockWriter(ctrl)

			if testCase.writeCall {
				buffer.EXPECT().
					Write(testCase.write.written).
					Return(testCase.write.n, testCase.write.err)
			}

			err := encodeChild(testCase.child, buffer)

			if testCase.wrappedErr != nil {
				assert.ErrorIs(t, err, testCase.wrappedErr)
				assert.EqualError(t, err, testCase.errMessage)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package trie

import (
	"bytes"
	"errors"
	"math/rand"
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

func generateRandBytes(size int) []byte {
	buf := make([]byte, rand.Intn(size)+1)
	rand.Read(buf)
	return buf
}

func generateRand(size int) [][]byte {
	rt := make([][]byte, size)
	for i := range rt {
		buf := make([]byte, rand.Intn(379)+1)
		rand.Read(buf)
		rt[i] = buf
	}
	return rt
}

func TestHashLeaf(t *testing.T) {
	n := &leaf{key: generateRandBytes(380), value: generateRandBytes(64)}

	buffer := bytes.NewBuffer(nil)
	const parallel = false
	err := encodeNode(n, buffer, parallel)

	if err != nil {
		t.Errorf("did not hash leaf node: %s", err)
	} else if buffer.Len() == 0 {
		t.Errorf("did not hash leaf node: nil")
	}
}

func TestHashBranch(t *testing.T) {
	n := &branch{key: generateRandBytes(380), value: generateRandBytes(380)}
	n.children[3] = &leaf{key: generateRandBytes(380), value: generateRandBytes(380)}

	buffer := bytes.NewBuffer(nil)
	const parallel = false
	err := encodeNode(n, buffer, parallel)

	if err != nil {
		t.Errorf("did not hash branch node: %s", err)
	} else if buffer.Len() == 0 {
		t.Errorf("did not hash branch node: nil")
	}
}

func TestHashShort(t *testing.T) {
	n := &leaf{
		key:   generateRandBytes(2),
		value: generateRandBytes(3),
	}

	encodingBuffer := bytes.NewBuffer(nil)
	const parallel = false
	err := encodeNode(n, encodingBuffer, parallel)
	require.NoError(t, err)

	digestBuffer := bytes.NewBuffer(nil)
	err = hashNode(n, digestBuffer)
	require.NoError(t, err)
	assert.Equal(t, encodingBuffer.Bytes(), digestBuffer.Bytes())
}

var errTest = errors.New("test error")

//go:generate mockgen -destination=readwriter_mock_test.go -package $GOPACKAGE io ReadWriter

func Test_encodeChildrenInParallel(t *testing.T) {
	t.Parallel()

	testCases := map[string]struct {
		children   [16]node
		writes     []writeCall
		wrappedErr error
		errMessage string
	}{
		"no children": {},
		"first child not nil": {
			children: [16]node{
				&leaf{key: []byte{1}},
			},
			writes: []writeCall{
				{
					written: []byte{12, 65, 1, 0},
				},
			},
		},
		"last child not nil": {
			children: [16]node{
				nil, nil, nil, nil, nil,
				nil, nil, nil, nil, nil,
				nil, nil, nil, nil, nil,
				&leaf{key: []byte{1}},
			},
			writes: []writeCall{
				{
					written: []byte{12, 65, 1, 0},
				},
			},
		},
		"first two children not nil": {
			children: [16]node{
				&leaf{key: []byte{1}},
				&leaf{key: []byte{2}},
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
			children: [16]node{
				nil, nil, nil, nil,
				nil, nil, nil, nil,
				nil, nil, nil,
				&leaf{
					key: []byte{1},
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

			buffer := NewMockReadWriter(ctrl)
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
		children   [16]node
		writes     []writeCall
		wrappedErr error
		errMessage string
	}{
		"no children": {},
		"first child not nil": {
			children: [16]node{
				&leaf{key: []byte{1}},
			},
			writes: []writeCall{
				{
					written: []byte{12, 65, 1, 0},
				},
			},
		},
		"last child not nil": {
			children: [16]node{
				nil, nil, nil, nil, nil,
				nil, nil, nil, nil, nil,
				nil, nil, nil, nil, nil,
				&leaf{key: []byte{1}},
			},
			writes: []writeCall{
				{
					written: []byte{12, 65, 1, 0},
				},
			},
		},
		"first two children not nil": {
			children: [16]node{
				&leaf{key: []byte{1}},
				&leaf{key: []byte{2}},
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
			children: [16]node{
				nil, nil, nil, nil,
				nil, nil, nil, nil,
				nil, nil, nil,
				&leaf{
					key: []byte{1},
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

			buffer := NewMockReadWriter(ctrl)
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
		child      node
		writeCall  bool
		write      writeCall
		wrappedErr error
		errMessage string
	}{
		"nil node": {},
		"nil leaf": {
			child: (*leaf)(nil),
		},
		"nil branch": {
			child: (*branch)(nil),
		},
		"empty leaf child": {
			child:     &leaf{},
			writeCall: true,
			write: writeCall{
				written: []byte{8, 64, 0},
			},
		},
		"empty branch child": {
			child:     &branch{},
			writeCall: true,
			write: writeCall{
				written: []byte{12, 128, 0, 0},
			},
		},
		"buffer write error": {
			child:     &branch{},
			writeCall: true,
			write: writeCall{
				written: []byte{12, 128, 0, 0},
				err:     errTest,
			},
			wrappedErr: errTest,
			errMessage: "failed to write child to buffer: test error",
		},
		"leaf child": {
			child: &leaf{
				key:   []byte{1},
				value: []byte{2},
			},
			writeCall: true,
			write: writeCall{
				written: []byte{16, 65, 1, 4, 2},
			},
		},
		"branch child": {
			child: &branch{
				key:   []byte{1},
				value: []byte{2},
				children: [16]node{
					nil, nil, &leaf{
						key:   []byte{5},
						value: []byte{6},
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

func Test_encodeLeaf(t *testing.T) {
	t.Parallel()

	testCases := map[string]struct {
		leaf       *leaf
		writes     []writeCall
		wrappedErr error
		errMessage string
	}{
		"clean leaf with encoding": {
			leaf: &leaf{
				encoding: []byte{1, 2, 3},
			},
			writes: []writeCall{
				{
					written: []byte{1, 2, 3},
				},
			},
		},
		"write error for clean leaf with encoding": {
			leaf: &leaf{
				encoding: []byte{1, 2, 3},
			},
			writes: []writeCall{
				{
					written: []byte{1, 2, 3},
					err:     errTest,
				},
			},
			wrappedErr: errTest,
			errMessage: "cannot write stored encoding to buffer: test error",
		},
		"header encoding error": {
			leaf: &leaf{
				key: make([]byte, 63+(1<<16)),
			},
			wrappedErr: ErrPartialKeyTooBig,
			errMessage: "cannot encode header: partial key length greater than or equal to 2^16",
		},
		"buffer write error for encoded header": {
			leaf: &leaf{
				key: []byte{1, 2, 3},
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
			leaf: &leaf{
				key: []byte{1, 2, 3},
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
			leaf: &leaf{
				key:   []byte{1, 2, 3},
				value: []byte{4, 5, 6},
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
			leaf: &leaf{
				key:   []byte{1, 2, 3},
				value: []byte{4, 5, 6},
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
		},
	}

	for name, testCase := range testCases {
		testCase := testCase
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			ctrl := gomock.NewController(t)

			buffer := NewMockReadWriter(ctrl)
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

			err := encodeLeaf(testCase.leaf, buffer)

			if testCase.wrappedErr != nil {
				assert.ErrorIs(t, err, testCase.wrappedErr)
				assert.EqualError(t, err, testCase.errMessage)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func Test_encodeBranch(t *testing.T) {
	t.Parallel()

	testCases := map[string]struct {
		branch     *branch
		writes     []writeCall
		parallel   bool
		wrappedErr error
		errMessage string
	}{
		"clean branch with encoding": {
			branch: &branch{
				encoding: []byte{1, 2, 3},
			},
			writes: []writeCall{
				{ // stored encoding
					written: []byte{1, 2, 3},
				},
			},
		},
		"write error for clean branch with encoding": {
			branch: &branch{
				encoding: []byte{1, 2, 3},
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
			branch: &branch{
				key: make([]byte, 63+(1<<16)),
			},
			wrappedErr: ErrPartialKeyTooBig,
			errMessage: "cannot encode header: partial key length greater than or equal to 2^16",
		},
		"buffer write error for encoded header": {
			branch: &branch{
				key:   []byte{1, 2, 3},
				value: []byte{100},
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
			branch: &branch{
				key:   []byte{1, 2, 3},
				value: []byte{100},
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
			branch: &branch{
				key:   []byte{1, 2, 3},
				value: []byte{100},
				children: [16]node{
					nil, nil, nil, &leaf{key: []byte{9}},
					nil, nil, nil, &leaf{key: []byte{11}},
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
			branch: &branch{
				key:   []byte{1, 2, 3},
				value: []byte{100},
				children: [16]node{
					nil, nil, nil, &leaf{key: []byte{9}},
					nil, nil, nil, &leaf{key: []byte{11}},
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
			branch: &branch{
				key:   []byte{1, 2, 3},
				value: []byte{100},
				children: [16]node{
					nil, nil, nil, &leaf{key: []byte{9}},
					nil, nil, nil, &leaf{key: []byte{11}},
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
		"buffer write error for children encoded in parallel": {
			branch: &branch{
				key:   []byte{1, 2, 3},
				value: []byte{100},
				children: [16]node{
					nil, nil, nil, &leaf{key: []byte{9}},
					nil, nil, nil, &leaf{key: []byte{11}},
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
					err:     errTest,
				},
				{ // second children
					written: []byte{12, 65, 11, 0},
				},
			},
			parallel:   true,
			wrappedErr: errTest,
			errMessage: "cannot encode children of branch: " +
				"cannot write encoding of child at index 3: " +
				"test error",
		},
		"success with parallel children encoding": {
			branch: &branch{
				key:   []byte{1, 2, 3},
				value: []byte{100},
				children: [16]node{
					nil, nil, nil, &leaf{key: []byte{9}},
					nil, nil, nil, &leaf{key: []byte{11}},
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
			parallel: true,
		},
		"success with sequential children encoding": {
			branch: &branch{
				key:   []byte{1, 2, 3},
				value: []byte{100},
				children: [16]node{
					nil, nil, nil, &leaf{key: []byte{9}},
					nil, nil, nil, &leaf{key: []byte{11}},
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

			buffer := NewMockReadWriter(ctrl)
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

			err := encodeBranch(testCase.branch, buffer, testCase.parallel)

			if testCase.wrappedErr != nil {
				assert.ErrorIs(t, err, testCase.wrappedErr)
				assert.EqualError(t, err, testCase.errMessage)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

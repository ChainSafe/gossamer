// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package node

import (
	"bytes"
	"io"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_Branch_ScaleEncodeHash(t *testing.T) {
	t.Parallel()

	testCases := map[string]struct {
		branch     *Branch
		encoding   []byte
		wrappedErr error
		errMessage string
	}{
		"empty branch": {
			branch:   &Branch{},
			encoding: []byte{0xc, 0x80, 0x0, 0x0},
		},
		"non empty branch": {
			branch: &Branch{
				Key:   []byte{1, 2},
				Value: []byte{3, 4},
				Children: [16]Node{
					nil, nil, &Leaf{Key: []byte{9}},
				},
			},
			encoding: []byte{0x2c, 0xc2, 0x12, 0x4, 0x0, 0x8, 0x3, 0x4, 0xc, 0x41, 0x9, 0x0},
		},
	}

	for name, testCase := range testCases {
		testCase := testCase
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			encoding, err := testCase.branch.ScaleEncodeHash()

			if testCase.wrappedErr != nil {
				assert.ErrorIs(t, err, testCase.wrappedErr)
				assert.EqualError(t, err, testCase.errMessage)
			} else {
				require.NoError(t, err)
			}
			assert.Equal(t, testCase.encoding, encoding)
		})
	}
}

func Test_Branch_hash(t *testing.T) {
	t.Parallel()

	testCases := map[string]struct {
		branch     *Branch
		write      writeCall
		errWrapped error
		errMessage string
	}{
		"empty branch": {
			branch: &Branch{},
			write: writeCall{
				written: []byte{128, 0, 0},
			},
		},
		"less than 32 bytes encoding": {
			branch: &Branch{
				Key: []byte{1, 2},
			},
			write: writeCall{
				written: []byte{130, 18, 0, 0},
			},
		},
		"less than 32 bytes encoding write error": {
			branch: &Branch{
				Key: []byte{1, 2},
			},
			write: writeCall{
				written: []byte{130, 18, 0, 0},
				err:     errTest,
			},
			errWrapped: errTest,
			errMessage: "cannot write encoded branch to buffer: test error",
		},
		"more than 32 bytes encoding": {
			branch: &Branch{
				Key: repeatBytes(100, 1),
			},
			write: writeCall{
				written: []byte{
					70, 102, 188, 24, 31, 68, 86, 114,
					95, 156, 225, 138, 175, 254, 176, 251,
					81, 84, 193, 40, 11, 234, 142, 233,
					69, 250, 158, 86, 72, 228, 66, 46},
			},
		},
		"more than 32 bytes encoding write error": {
			branch: &Branch{
				Key: repeatBytes(100, 1),
			},
			write: writeCall{
				written: []byte{
					70, 102, 188, 24, 31, 68, 86, 114,
					95, 156, 225, 138, 175, 254, 176, 251,
					81, 84, 193, 40, 11, 234, 142, 233,
					69, 250, 158, 86, 72, 228, 66, 46},
				err: errTest,
			},
			errWrapped: errTest,
			errMessage: "cannot write hash sum of branch to buffer: test error",
		},
	}

	for name, testCase := range testCases {
		testCase := testCase
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			ctrl := gomock.NewController(t)

			digestBuffer := NewMockWriter(ctrl)
			digestBuffer.EXPECT().Write(testCase.write.written).
				Return(testCase.write.n, testCase.write.err)

			err := testCase.branch.hash(digestBuffer)

			if testCase.errWrapped != nil {
				assert.ErrorIs(t, err, testCase.errWrapped)
				assert.EqualError(t, err, testCase.errMessage)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

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
				encoding: []byte{1, 2, 3},
			},
			writes: []writeCall{
				{ // stored encoding
					written: []byte{1, 2, 3},
				},
			},
		},
		"write error for clean branch with encoding": {
			branch: &Branch{
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
			branch: &Branch{
				Key: make([]byte, 63+(1<<16)),
			},
			writes: []writeCall{
				{ // header
					written: []byte{191},
				},
			},
			wrappedErr: ErrPartialKeyTooBig,
			errMessage: "cannot encode header: partial key length cannot be larger than or equal to 2^16: 65536",
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
				Children: [16]Node{
					nil, nil, nil, &Leaf{Key: []byte{9}},
					nil, nil, nil, &Leaf{Key: []byte{11}},
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
				Children: [16]Node{
					nil, nil, nil, &Leaf{Key: []byte{9}},
					nil, nil, nil, &Leaf{Key: []byte{11}},
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
		"buffer write error for children encoding": {
			branch: &Branch{
				Key:   []byte{1, 2, 3},
				Value: []byte{100},
				Children: [16]Node{
					nil, nil, nil, &Leaf{Key: []byte{9}},
					nil, nil, nil, &Leaf{Key: []byte{11}},
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
				"cannot write encoding of child at index 3: " +
				"test error",
		},
		"success with children encoding": {
			branch: &Branch{
				Key:   []byte{1, 2, 3},
				Value: []byte{100},
				Children: [16]Node{
					nil, nil, nil, &Leaf{Key: []byte{9}},
					nil, nil, nil, &Leaf{Key: []byte{11}},
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

// Opportunistic parallel:	13781602 ns/op	14419488 B/op	  323575 allocs/op
// Sequentially:			24269268 ns/op	20126525 B/op	  327668 allocs/op
func Benchmark_encodeChildrenOpportunisticParallel(b *testing.B) {
	const valueBytesSize = 10
	const depth = 3 // do not raise above 4

	children := populateChildren(valueBytesSize, depth)

	b.Run("", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_ = encodeChildrenOpportunisticParallel(children, io.Discard)
		}
	})
}

func populateChildren(valueSize, depth int) (children [16]Node) {
	someValue := make([]byte, valueSize)

	if depth == 0 {
		for i := range children {
			children[i] = &Leaf{
				Key:   someValue,
				Value: someValue,
			}
		}
		return children
	}

	for i := range children {
		children[i] = &Branch{
			Key:      someValue,
			Value:    someValue,
			Children: populateChildren(valueSize, depth-1),
		}
	}

	return children
}

func Test_encodeChildrenOpportunisticParallel(t *testing.T) {
	t.Parallel()

	testCases := map[string]struct {
		children   [16]Node
		writes     []writeCall
		wrappedErr error
		errMessage string
	}{
		"no children": {},
		"first child not nil": {
			children: [16]Node{
				&Leaf{Key: []byte{1}},
			},
			writes: []writeCall{
				{
					written: []byte{12, 65, 1, 0},
				},
			},
		},
		"last child not nil": {
			children: [16]Node{
				nil, nil, nil, nil, nil,
				nil, nil, nil, nil, nil,
				nil, nil, nil, nil, nil,
				&Leaf{Key: []byte{1}},
			},
			writes: []writeCall{
				{
					written: []byte{12, 65, 1, 0},
				},
			},
		},
		"first two children not nil": {
			children: [16]Node{
				&Leaf{Key: []byte{1}},
				&Leaf{Key: []byte{2}},
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
		"leaf encoding error": {
			children: [16]Node{
				nil, nil, nil, nil,
				nil, nil, nil, nil,
				nil, nil, nil,
				&Leaf{
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
		"branch encoding": {
			// Note this may run in parallel or not depending on other tests
			// running in parallel.
			children: [16]Node{
				&Branch{
					Key: []byte{1},
					Children: [16]Node{
						&Leaf{Key: []byte{1}},
					},
				},
			},
			writes: []writeCall{
				{
					written: []byte{32, 129, 1, 1, 0, 12, 65, 1, 0},
				},
			},
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

			err := encodeChildrenOpportunisticParallel(testCase.children, buffer)

			if testCase.wrappedErr != nil {
				assert.ErrorIs(t, err, testCase.wrappedErr)
				assert.EqualError(t, err, testCase.errMessage)
			} else {
				require.NoError(t, err)
			}
		})
	}

	t.Run("opportunist parallel branch encoding", func(t *testing.T) {
		t.Parallel()

		var children [16]Node
		for i := range children {
			children[i] = &Branch{}
		}

		buffer := bytes.NewBuffer(nil)

		err := encodeChildrenOpportunisticParallel(children, buffer)

		require.NoError(t, err)
		expectedBytes := []byte{
			0xc, 0x80, 0x0, 0x0, 0xc, 0x80, 0x0, 0x0,
			0xc, 0x80, 0x0, 0x0, 0xc, 0x80, 0x0, 0x0,
			0xc, 0x80, 0x0, 0x0, 0xc, 0x80, 0x0, 0x0,
			0xc, 0x80, 0x0, 0x0, 0xc, 0x80, 0x0, 0x0,
			0xc, 0x80, 0x0, 0x0, 0xc, 0x80, 0x0, 0x0,
			0xc, 0x80, 0x0, 0x0, 0xc, 0x80, 0x0, 0x0,
			0xc, 0x80, 0x0, 0x0, 0xc, 0x80, 0x0, 0x0,
			0xc, 0x80, 0x0, 0x0, 0xc, 0x80, 0x0, 0x0}
		assert.Equal(t, expectedBytes, buffer.Bytes())
	})
}

func Test_encodeChildrenSequentially(t *testing.T) {
	t.Parallel()

	testCases := map[string]struct {
		children   [16]Node
		writes     []writeCall
		wrappedErr error
		errMessage string
	}{
		"no children": {},
		"first child not nil": {
			children: [16]Node{
				&Leaf{Key: []byte{1}},
			},
			writes: []writeCall{
				{
					written: []byte{12, 65, 1, 0},
				},
			},
		},
		"last child not nil": {
			children: [16]Node{
				nil, nil, nil, nil, nil,
				nil, nil, nil, nil, nil,
				nil, nil, nil, nil, nil,
				&Leaf{Key: []byte{1}},
			},
			writes: []writeCall{
				{
					written: []byte{12, 65, 1, 0},
				},
			},
		},
		"first two children not nil": {
			children: [16]Node{
				&Leaf{Key: []byte{1}},
				&Leaf{Key: []byte{2}},
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
			children: [16]Node{
				nil, nil, nil, nil,
				nil, nil, nil, nil,
				nil, nil, nil,
				&Leaf{
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

func Test_encodeChild(t *testing.T) {
	t.Parallel()

	testCases := map[string]struct {
		child      Node
		writeCall  bool
		write      writeCall
		wrappedErr error
		errMessage string
	}{
		"nil node": {},
		"nil leaf": {
			child: (*Leaf)(nil),
		},
		"nil branch": {
			child: (*Branch)(nil),
		},
		"empty leaf child": {
			child:     &Leaf{},
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
			child: &Leaf{
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
				Children: [16]Node{
					nil, nil, &Leaf{
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

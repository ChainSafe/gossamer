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

func populateChildren(valueSize, depth int) (children []*Node) {
	someValue := make([]byte, valueSize)
	children = make([]*Node, ChildrenCapacity)

	if depth == 0 {
		for i := range children {
			children[i] = &Node{
				Key:      someValue,
				SubValue: someValue,
			}
		}
		return children
	}

	for i := range children {
		children[i] = &Node{
			Key:      someValue,
			SubValue: someValue,
			Children: populateChildren(valueSize, depth-1),
		}
	}

	return children
}

func Test_encodeChildrenOpportunisticParallel(t *testing.T) {
	t.Parallel()

	testCases := map[string]struct {
		children   []*Node
		writes     []writeCall
		wrappedErr error
		errMessage string
	}{
		"no children": {},
		"first child not nil": {
			children: []*Node{
				{Key: []byte{1}, SubValue: []byte{2}},
			},
			writes: []writeCall{
				{
					written: []byte{16, 65, 1, 4, 2},
				},
			},
		},
		"last child not nil": {
			children: []*Node{
				nil, nil, nil, nil, nil,
				nil, nil, nil, nil, nil,
				nil, nil, nil, nil, nil,
				{Key: []byte{1}, SubValue: []byte{2}},
			},
			writes: []writeCall{
				{
					written: []byte{16, 65, 1, 4, 2},
				},
			},
		},
		"first two children not nil": {
			children: []*Node{
				{Key: []byte{1}, SubValue: []byte{2}},
				{Key: []byte{3}, SubValue: []byte{4}},
			},
			writes: []writeCall{
				{
					written: []byte{16, 65, 1, 4, 2},
				},
				{
					written: []byte{16, 65, 3, 4, 4},
				},
			},
		},
		"leaf encoding error": {
			children: []*Node{
				nil, nil, nil, nil,
				nil, nil, nil, nil,
				nil, nil, nil,
				{Key: []byte{1}, SubValue: []byte{2}},
				nil, nil, nil, nil,
			},
			writes: []writeCall{
				{
					written: []byte{16, 65, 1, 4, 2},
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
			children: []*Node{
				{
					Key: []byte{1},
					Children: []*Node{
						{Key: []byte{1}, SubValue: []byte{2}},
					},
				},
			},
			writes: []writeCall{
				{
					written: []byte{36, 129, 1, 1, 0, 16, 65, 1, 4, 2},
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

		children := make([]*Node, ChildrenCapacity)
		for i := range children {
			children[i] = &Node{
				Children: make([]*Node, ChildrenCapacity),
			}
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
		children   []*Node
		writes     []writeCall
		wrappedErr error
		errMessage string
	}{
		"no children": {},
		"first child not nil": {
			children: []*Node{
				{Key: []byte{1}, SubValue: []byte{2}},
			},
			writes: []writeCall{
				{
					written: []byte{16, 65, 1, 4, 2},
				},
			},
		},
		"last child not nil": {
			children: []*Node{
				nil, nil, nil, nil, nil,
				nil, nil, nil, nil, nil,
				nil, nil, nil, nil, nil,
				{Key: []byte{1}, SubValue: []byte{2}},
			},
			writes: []writeCall{
				{
					written: []byte{16, 65, 1, 4, 2},
				},
			},
		},
		"first two children not nil": {
			children: []*Node{
				{Key: []byte{1}, SubValue: []byte{2}},
				{Key: []byte{3}, SubValue: []byte{4}},
			},
			writes: []writeCall{
				{
					written: []byte{16, 65, 1, 4, 2},
				},
				{
					written: []byte{16, 65, 3, 4, 4},
				},
			},
		},
		"encoding error": {
			children: []*Node{
				nil, nil, nil, nil,
				nil, nil, nil, nil,
				nil, nil, nil,
				{Key: []byte{1}, SubValue: []byte{2}},
				nil, nil, nil, nil,
			},
			writes: []writeCall{
				{
					written: []byte{16, 65, 1, 4, 2},
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
		child      *Node
		writeCall  bool
		write      writeCall
		wrappedErr error
		errMessage string
	}{
		"nil node": {},
		"empty branch child": {
			child: &Node{
				Children: make([]*Node, ChildrenCapacity),
			},
			writeCall: true,
			write: writeCall{
				written: []byte{12, 128, 0, 0},
			},
		},
		"buffer write error": {
			child: &Node{
				Children: make([]*Node, ChildrenCapacity),
			},
			writeCall: true,
			write: writeCall{
				written: []byte{12, 128, 0, 0},
				err:     errTest,
			},
			wrappedErr: errTest,
			errMessage: "failed to write child to buffer: test error",
		},
		"leaf child": {
			child: &Node{
				Key:      []byte{1},
				SubValue: []byte{2},
			},
			writeCall: true,
			write: writeCall{
				written: []byte{16, 65, 1, 4, 2},
			},
		},
		"branch child": {
			child: &Node{
				Key:      []byte{1},
				SubValue: []byte{2},
				Children: []*Node{
					nil, nil, {Key: []byte{5},
						SubValue: []byte{6},
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

func Test_scaleEncodeHash(t *testing.T) {
	t.Parallel()

	testCases := map[string]struct {
		node       *Node
		encoding   []byte
		wrappedErr error
		errMessage string
	}{
		"branch": {
			node: &Node{
				Key:      []byte{1, 2},
				SubValue: []byte{3, 4},
				Children: []*Node{
					nil, nil, {Key: []byte{9}, SubValue: []byte{1}},
				},
			},
			encoding: []byte{0x30, 0xc2, 0x12, 0x4, 0x0, 0x8, 0x3, 0x4, 0x10, 0x41, 0x9, 0x4, 0x1},
		},
	}

	for name, testCase := range testCases {
		testCase := testCase
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			encoding, err := scaleEncodeHash(testCase.node)

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

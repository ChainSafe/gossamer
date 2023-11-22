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
			_ = encodeChildrenOpportunisticParallel(children, NoMaxInlineValueSize, io.Discard)
		}
	})
}

func populateChildren(valueSize, depth int) (children []*Node) {
	someValue := make([]byte, valueSize)
	children = make([]*Node, ChildrenCapacity)

	if depth == 0 {
		for i := range children {
			children[i] = &Node{
				PartialKey:   someValue,
				StorageValue: someValue,
			}
		}
		return children
	}

	for i := range children {
		children[i] = &Node{
			PartialKey:   someValue,
			StorageValue: someValue,
			Children:     populateChildren(valueSize, depth-1),
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
		"no_children": {},
		"first_child_not_nil": {
			children: []*Node{
				{PartialKey: []byte{1}, StorageValue: []byte{2}},
			},
			writes: []writeCall{
				{
					written: []byte{16, 65, 1, 4, 2},
				},
			},
		},
		"last_child_not_nil": {
			children: []*Node{
				nil, nil, nil, nil, nil,
				nil, nil, nil, nil, nil,
				nil, nil, nil, nil, nil,
				{PartialKey: []byte{1}, StorageValue: []byte{2}},
			},
			writes: []writeCall{
				{
					written: []byte{16, 65, 1, 4, 2},
				},
			},
		},
		"first_two_children_not_nil": {
			children: []*Node{
				{PartialKey: []byte{1}, StorageValue: []byte{2}},
				{PartialKey: []byte{3}, StorageValue: []byte{4}},
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
		"leaf_encoding_error": {
			children: []*Node{
				nil, nil, nil, nil,
				nil, nil, nil, nil,
				nil, nil, nil,
				{PartialKey: []byte{1}, StorageValue: []byte{2}},
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

			err := encodeChildrenOpportunisticParallel(testCase.children, NoMaxInlineValueSize, buffer)

			if testCase.wrappedErr != nil {
				assert.ErrorIs(t, err, testCase.wrappedErr)
				assert.EqualError(t, err, testCase.errMessage)
			} else {
				require.NoError(t, err)
			}
		})
	}

	t.Run("opportunist_parallel_branch_encoding", func(t *testing.T) {
		t.Parallel()

		children := make([]*Node, ChildrenCapacity)
		for i := range children {
			children[i] = &Node{
				Children: make([]*Node, ChildrenCapacity),
			}
		}

		buffer := bytes.NewBuffer(nil)

		// Note this may run in parallel or not depending on other tests
		// running in parallel.
		err := encodeChildrenOpportunisticParallel(children, NoMaxInlineValueSize, buffer)

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

func Test_encodeChild(t *testing.T) {
	t.Parallel()

	testCases := map[string]struct {
		child      *Node
		writes     []writeCall
		wrappedErr error
		errMessage string
	}{
		"empty_branch_child": {
			child: &Node{
				Children: make([]*Node, ChildrenCapacity),
			},
			writes: []writeCall{
				{written: []byte{12}},
				{written: []byte{128, 0, 0}},
			},
		},
		"scale_encoding_error": {
			child: &Node{
				Children: make([]*Node, ChildrenCapacity),
			},
			writes: []writeCall{{
				written: []byte{12},
				err:     errTest,
			}},
			wrappedErr: errTest,
			errMessage: "scale encoding Merkle value: test error",
		},
		"leaf_child": {
			child: &Node{
				PartialKey:   []byte{1},
				StorageValue: []byte{2},
			},
			writes: []writeCall{
				{written: []byte{16}},
				{written: []byte{65, 1, 4, 2}},
			},
		},
		"branch_child": {
			child: &Node{
				PartialKey:   []byte{1},
				StorageValue: []byte{2},
				Children: []*Node{
					nil, nil, {PartialKey: []byte{5},
						StorageValue: []byte{6},
					},
				},
			},
			writes: []writeCall{
				{written: []byte{44}},
				{written: []byte{193, 1, 4, 0, 4, 2, 16, 65, 5, 4, 6}},
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

			err := encodeChild(testCase.child, NoMaxInlineValueSize, buffer)

			if testCase.wrappedErr != nil {
				assert.ErrorIs(t, err, testCase.wrappedErr)
				assert.EqualError(t, err, testCase.errMessage)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package node

import (
	"errors"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type writeCall struct {
	written []byte
	n       int // number of bytes
	err     error
}

var errTest = errors.New("test error")

func Test_Node_Encode(t *testing.T) {
	t.Parallel()

	testCases := map[string]struct {
		node             *Node
		writes           []writeCall
		bufferLenCall    bool
		bufferBytesCall  bool
		expectedEncoding []byte
		wrappedErr       error
		errMessage       string
	}{
		"clean leaf with encoding": {
			node: &Node{
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
			node: &Node{
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
		"leaf header encoding error": {
			node: &Node{
				Key: make([]byte, 1),
			},
			writes: []writeCall{
				{
					written: []byte{leafVariant.bits | 1},
					err:     errTest,
				},
			},
			wrappedErr: errTest,
			errMessage: "cannot encode header: test error",
		},
		"leaf buffer write error for encoded key": {
			node: &Node{
				Key: []byte{1, 2, 3},
			},
			writes: []writeCall{
				{
					written: []byte{leafVariant.bits | 3}, // partial key length 3
				},
				{
					written: []byte{0x01, 0x23},
					err:     errTest,
				},
			},
			wrappedErr: errTest,
			errMessage: "cannot write LE key to buffer: test error",
		},
		"leaf buffer write error for encoded value": {
			node: &Node{
				Key:   []byte{1, 2, 3},
				Value: []byte{4, 5, 6},
			},
			writes: []writeCall{
				{
					written: []byte{leafVariant.bits | 3}, // partial key length 3
				},
				{
					written: []byte{0x01, 0x23},
				},
				{
					written: []byte{12, 4, 5, 6},
					err:     errTest,
				},
			},
			wrappedErr: errTest,
			errMessage: "cannot write scale encoded value to buffer: test error",
		},
		"leaf success": {
			node: &Node{
				Key:   []byte{1, 2, 3},
				Value: []byte{4, 5, 6},
			},
			writes: []writeCall{
				{
					written: []byte{leafVariant.bits | 3}, // partial key length 3
				},
				{
					written: []byte{0x01, 0x23},
				},
				{
					written: []byte{12, 4, 5, 6},
				},
			},
			bufferLenCall:    true,
			bufferBytesCall:  true,
			expectedEncoding: []byte{1, 2, 3},
		},
		"clean branch with encoding": {
			node: &Node{
				Children: make([]*Node, ChildrenCapacity),
				Encoding: []byte{1, 2, 3},
			},
			writes: []writeCall{
				{ // stored encoding
					written: []byte{1, 2, 3},
				},
			},
		},
		"write error for clean branch with encoding": {
			node: &Node{
				Children: make([]*Node, ChildrenCapacity),
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
		"branch header encoding error": {
			node: &Node{
				Children: make([]*Node, ChildrenCapacity),
				Key:      make([]byte, 1),
			},
			writes: []writeCall{
				{ // header
					written: []byte{branchVariant.bits | 1}, // partial key length 1
					err:     errTest,
				},
			},
			wrappedErr: errTest,
			errMessage: "cannot encode header: test error",
		},
		"buffer write error for encoded key": {
			node: &Node{
				Children: make([]*Node, ChildrenCapacity),
				Key:      []byte{1, 2, 3},
				Value:    []byte{100},
			},
			writes: []writeCall{
				{ // header
					written: []byte{branchWithValueVariant.bits | 3}, // partial key length 3
				},
				{ // key LE
					written: []byte{0x01, 0x23},
					err:     errTest,
				},
			},
			wrappedErr: errTest,
			errMessage: "cannot write LE key to buffer: test error",
		},
		"buffer write error for children bitmap": {
			node: &Node{
				Key:   []byte{1, 2, 3},
				Value: []byte{100},
				Children: []*Node{
					nil, nil, nil, {Key: []byte{9}},
					nil, nil, nil, {Key: []byte{11}},
				},
			},
			writes: []writeCall{
				{ // header
					written: []byte{branchWithValueVariant.bits | 3}, // partial key length 3
				},
				{ // key LE
					written: []byte{0x01, 0x23},
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
			node: &Node{
				Key:   []byte{1, 2, 3},
				Value: []byte{100},
				Children: []*Node{
					nil, nil, nil, {Key: []byte{9}},
					nil, nil, nil, {Key: []byte{11}},
				},
			},
			writes: []writeCall{
				{ // header
					written: []byte{branchWithValueVariant.bits | 3}, // partial key length 3
				},
				{ // key LE
					written: []byte{0x01, 0x23},
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
			errMessage: "cannot write scale encoded value to buffer: test error",
		},
		"buffer write error for children encoding": {
			node: &Node{
				Key:   []byte{1, 2, 3},
				Value: []byte{100},
				Children: []*Node{
					nil, nil, nil, {Key: []byte{9}},
					nil, nil, nil, {Key: []byte{11}},
				},
			},
			writes: []writeCall{
				{ // header
					written: []byte{branchWithValueVariant.bits | 3}, // partial key length 3
				},
				{ // key LE
					written: []byte{0x01, 0x23},
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
			node: &Node{
				Key:   []byte{1, 2, 3},
				Value: []byte{100},
				Children: []*Node{
					nil, nil, nil, {Key: []byte{9}},
					nil, nil, nil, {Key: []byte{11}},
				},
			},
			writes: []writeCall{
				{ // header
					written: []byte{branchWithValueVariant.bits | 3}, // partial key length 3
				},
				{ // key LE
					written: []byte{0x01, 0x23},
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
			if testCase.bufferLenCall {
				buffer.EXPECT().Len().Return(len(testCase.expectedEncoding))
			}
			if testCase.bufferBytesCall {
				buffer.EXPECT().Bytes().Return(testCase.expectedEncoding)
			}

			err := testCase.node.Encode(buffer)

			if testCase.wrappedErr != nil {
				assert.ErrorIs(t, err, testCase.wrappedErr)
				assert.EqualError(t, err, testCase.errMessage)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

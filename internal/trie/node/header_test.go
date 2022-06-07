// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package node

import (
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
)

func Test_encodeHeader(t *testing.T) {
	testCases := map[string]struct {
		node       *Node
		writes     []writeCall
		errWrapped error
		errMessage string
	}{
		"branch with no key": {
			node: &Node{
				Children: make([]*Node, ChildrenCapacity),
			},
			writes: []writeCall{
				{written: []byte{0x80}},
			},
		},
		"branch with value": {
			node: &Node{
				Value:    []byte{},
				Children: make([]*Node, ChildrenCapacity),
			},
			writes: []writeCall{
				{written: []byte{0xc0}},
			},
		},
		"branch with key of length 30": {
			node: &Node{
				Key:      make([]byte, 30),
				Children: make([]*Node, ChildrenCapacity),
			},
			writes: []writeCall{
				{written: []byte{0x9e}},
			},
		},
		"branch with key of length 62": {
			node: &Node{
				Key:      make([]byte, 62),
				Children: make([]*Node, ChildrenCapacity),
			},
			writes: []writeCall{
				{written: []byte{0xbe}},
			},
		},
		"branch with key of length 63": {
			node: &Node{
				Key:      make([]byte, 63),
				Children: make([]*Node, ChildrenCapacity),
			},
			writes: []writeCall{
				{written: []byte{0xbf}},
				{written: []byte{0x0}},
			},
		},
		"branch with key of length 64": {
			node: &Node{
				Key:      make([]byte, 64),
				Children: make([]*Node, ChildrenCapacity),
			},
			writes: []writeCall{
				{written: []byte{0xbf}},
				{written: []byte{0x1}},
			},
		},
		"branch with key too big": {
			node: &Node{
				Key:      make([]byte, 65535+63),
				Children: make([]*Node, ChildrenCapacity),
			},
			writes: []writeCall{
				{written: []byte{0xbf}},
			},
			errWrapped: ErrPartialKeyTooBig,
			errMessage: "partial key length cannot be larger than or equal to 2^16: 65535",
		},
		"branch with small key length write error": {
			node: &Node{
				Children: make([]*Node, ChildrenCapacity),
			},
			writes: []writeCall{
				{
					written: []byte{0x80},
					err:     errTest,
				},
			},
			errWrapped: errTest,
			errMessage: "test error",
		},
		"branch with long key length write error": {
			node: &Node{
				Key:      make([]byte, 64),
				Children: make([]*Node, ChildrenCapacity),
			},
			writes: []writeCall{
				{
					written: []byte{0xbf},
					err:     errTest,
				},
			},
			errWrapped: errTest,
			errMessage: "test error",
		},
		"leaf with no key": {
			node: &Node{},
			writes: []writeCall{
				{written: []byte{0x40}},
			},
		},
		"leaf with key of length 30": {
			node: &Node{
				Key: make([]byte, 30),
			},
			writes: []writeCall{
				{written: []byte{0x5e}},
			},
		},
		"leaf with short key write error": {
			node: &Node{
				Key: make([]byte, 30),
			},
			writes: []writeCall{
				{
					written: []byte{0x5e},
					err:     errTest,
				},
			},
			errWrapped: errTest,
			errMessage: errTest.Error(),
		},
		"leaf with key of length 62": {
			node: &Node{
				Key: make([]byte, 62),
			},
			writes: []writeCall{
				{written: []byte{0x7e}},
			},
		},
		"leaf with key of length 63": {
			node: &Node{
				Key: make([]byte, 63),
			},
			writes: []writeCall{
				{written: []byte{0x7f}},
				{written: []byte{0x0}},
			},
		},
		"leaf with key of length 64": {
			node: &Node{
				Key: make([]byte, 64),
			},
			writes: []writeCall{
				{written: []byte{0x7f}},
				{written: []byte{0x1}},
			},
		},
		"leaf with long key first byte write error": {
			node: &Node{
				Key: make([]byte, 63),
			},
			writes: []writeCall{
				{
					written: []byte{0x7f},
					err:     errTest,
				},
			},
			errWrapped: errTest,
			errMessage: errTest.Error(),
		},
		"leaf with key too big": {
			node: &Node{
				Key: make([]byte, 65535+63),
			},
			writes: []writeCall{
				{written: []byte{0x7f}},
			},
			errWrapped: ErrPartialKeyTooBig,
			errMessage: "partial key length cannot be larger than or equal to 2^16: 65535",
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

				if previousCall != nil {
					call.After(previousCall)
				}
				previousCall = call
			}

			err := encodeHeader(testCase.node, writer)

			assert.ErrorIs(t, err, testCase.errWrapped)
			if testCase.errWrapped != nil {
				assert.EqualError(t, err, testCase.errMessage)
			}
		})
	}
}

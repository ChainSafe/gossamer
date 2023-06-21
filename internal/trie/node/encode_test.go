// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package node

import (
	"errors"
	"testing"

	"github.com/ChainSafe/gossamer/lib/common"
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

	hashedValue, err := common.Blake2bHash([]byte("test"))
	assert.NoError(t, err)

	testCases := map[string]struct {
		node       *Node
		writes     []writeCall
		wrappedErr error
		errMessage string
	}{
		"nil_node": {
			node: nil,
			writes: []writeCall{
				{
					written: []byte{emptyVariant.bits},
				},
			},
		},
		"leaf_header_encoding_error": {
			node: &Node{
				PartialKey: make([]byte, 1),
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
		"leaf_buffer_write_error_for_encoded_key": {
			node: &Node{
				PartialKey:   []byte{1, 2, 3},
				StorageValue: []byte{1},
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
		"leaf_buffer_write_error_for_encoded_storage_value": {
			node: &Node{
				PartialKey:   []byte{1, 2, 3},
				StorageValue: []byte{4, 5, 6},
			},
			writes: []writeCall{
				{
					written: []byte{leafVariant.bits | 3}, // partial key length 3
				},
				{
					written: []byte{0x01, 0x23},
				},
				{
					written: []byte{12},
					err:     errTest,
				},
			},
			wrappedErr: errTest,
			errMessage: "scale encoding storage value: test error",
		},
		"leaf_success": {
			node: &Node{
				PartialKey:   []byte{1, 2, 3},
				StorageValue: []byte{4, 5, 6},
			},
			writes: []writeCall{
				{
					written: []byte{leafVariant.bits | 3}, // partial key length 3
				},
				{written: []byte{0x01, 0x23}},
				{written: []byte{12}},
				{written: []byte{4, 5, 6}},
			},
		},
		"leaf_with_empty_storage_value_success": {
			node: &Node{
				PartialKey:   []byte{1, 2, 3},
				StorageValue: []byte{},
			},
			writes: []writeCall{
				{written: []byte{leafVariant.bits | 3}}, // partial key length 3
				{written: []byte{0x01, 0x23}},           // partial key
				{written: []byte{0}},                    // node storage value encoded length
				{written: []byte{}},                     // node storage value
			},
		},
		"leaf_with_hashed_value_success": {
			node: &Node{
				PartialKey:   []byte{1, 2, 3},
				StorageValue: hashedValue.ToBytes(),
				HashedValue:  true,
			},
			writes: []writeCall{
				{
					written: []byte{leafWithHashedValueVariant.bits | 3},
				},
				{written: []byte{0x01, 0x23}},
				{written: hashedValue.ToBytes()},
			},
		},
		"leaf_with_hashed_value_fail": {
			node: &Node{
				PartialKey:   []byte{1, 2, 3},
				StorageValue: hashedValue.ToBytes(),
				HashedValue:  true,
			},
			writes: []writeCall{
				{
					written: []byte{leafWithHashedValueVariant.bits | 3},
				},
				{
					written: []byte{0x01, 0x23},
				},
				{
					written: hashedValue.ToBytes(),
					err:     errTest,
				},
			},
			wrappedErr: errTest,
			errMessage: "encoding hashed storage value: test error",
		},
		"leaf_with_hashed_value_fail_too_short": {
			node: &Node{
				PartialKey:   []byte{1, 2, 3},
				StorageValue: []byte("tooshort"),
				HashedValue:  true,
			},
			writes: []writeCall{
				{
					written: []byte{leafWithHashedValueVariant.bits | 3},
				},
				{written: []byte{0x01, 0x23}},
			},
			wrappedErr: ErrEncodeHashedValueTooShort,
			errMessage: "hashed storage value too short: expected 32, got: 8",
		},
		"branch_header_encoding_error": {
			node: &Node{
				Children:   make([]*Node, ChildrenCapacity),
				PartialKey: make([]byte, 1),
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
		"buffer_write_error_for_encoded_key": {
			node: &Node{
				Children:     make([]*Node, ChildrenCapacity),
				PartialKey:   []byte{1, 2, 3},
				StorageValue: []byte{100},
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
		"buffer_write_error_for_children_bitmap": {
			node: &Node{
				PartialKey:   []byte{1, 2, 3},
				StorageValue: []byte{100},
				Children: []*Node{
					nil, nil, nil, {PartialKey: []byte{9}, StorageValue: []byte{1}},
					nil, nil, nil, {PartialKey: []byte{11}, StorageValue: []byte{1}},
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
		"buffer_write_error_for_storage_value": {
			node: &Node{
				PartialKey:   []byte{1, 2, 3},
				StorageValue: []byte{100},
				Children: []*Node{
					nil, nil, nil, {PartialKey: []byte{9}, StorageValue: []byte{1}},
					nil, nil, nil, {PartialKey: []byte{11}, StorageValue: []byte{1}},
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
				{ // storage value
					written: []byte{4},
					err:     errTest,
				},
			},
			wrappedErr: errTest,
			errMessage: "scale encoding storage value: test error",
		},
		"buffer_write_error_for_children_encoding": {
			node: &Node{
				PartialKey:   []byte{1, 2, 3},
				StorageValue: []byte{100},
				Children: []*Node{
					nil, nil, nil, {PartialKey: []byte{9}, StorageValue: []byte{1}},
					nil, nil, nil, {PartialKey: []byte{11}, StorageValue: []byte{1}},
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
				// storage value
				{written: []byte{4}},
				{written: []byte{100}},
				{ // children
					written: []byte{16, 65, 9, 4, 1},
					err:     errTest,
				},
			},
			wrappedErr: errTest,
			errMessage: "cannot encode children of branch: " +
				"cannot write encoding of child at index 3: " +
				"test error",
		},
		"success_with_children_encoding": {
			node: &Node{
				PartialKey:   []byte{1, 2, 3},
				StorageValue: []byte{100},
				Children: []*Node{
					nil, nil, nil, {PartialKey: []byte{9}, StorageValue: []byte{1}},
					nil, nil, nil, {PartialKey: []byte{11}, StorageValue: []byte{1}},
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
				// storage value
				{written: []byte{4}},
				{written: []byte{100}},
				{ // first children
					written: []byte{16, 65, 9, 4, 1},
				},
				{ // second children
					written: []byte{16, 65, 11, 4, 1},
				},
			},
		},
		"branch_without_value_and_with_children_success": {
			node: &Node{
				PartialKey: []byte{1, 2, 3},
				Children: []*Node{
					nil, nil, nil, {PartialKey: []byte{9}, StorageValue: []byte{1}},
					nil, nil, nil, {PartialKey: []byte{11}, StorageValue: []byte{1}},
				},
			},
			writes: []writeCall{
				{ // header
					written: []byte{branchVariant.bits | 3}, // partial key length 3
				},
				{ // key LE
					written: []byte{0x01, 0x23},
				},
				{ // children bitmap
					written: []byte{136, 0},
				},
				{ // first children
					written: []byte{16, 65, 9, 4, 1},
				},
				{ // second children
					written: []byte{16, 65, 11, 4, 1},
				},
			},
		},
		"branch_with_hashed_value_success": {
			node: &Node{
				PartialKey:   []byte{1, 2, 3},
				StorageValue: hashedValue.ToBytes(),
				HashedValue:  true,
				Children: []*Node{
					nil, nil, nil, {PartialKey: []byte{9}, StorageValue: []byte{1}},
					nil, nil, nil, {PartialKey: []byte{11}, StorageValue: []byte{1}},
				},
			},
			writes: []writeCall{
				{ // header
					written: []byte{branchWithHashedValueVariant.bits | 3}, // partial key length 3
				},
				{ // key LE
					written: []byte{0x01, 0x23},
				},
				{ // children bitmap
					written: []byte{136, 0},
				},
				{
					written: hashedValue.ToBytes(),
				},
				{ // first children
					written: []byte{16, 65, 9, 4, 1},
				},
				{ // second children
					written: []byte{16, 65, 11, 4, 1},
				},
			},
		},
		"branch_with_hashed_value_fail_too_short": {
			node: &Node{
				PartialKey:   []byte{1, 2, 3},
				StorageValue: []byte("tooshort"),
				HashedValue:  true,
				Children: []*Node{
					nil, nil, nil, {PartialKey: []byte{9}, StorageValue: []byte{1}},
					nil, nil, nil, {PartialKey: []byte{11}, StorageValue: []byte{1}},
				},
			},
			writes: []writeCall{
				{ // header
					written: []byte{branchWithHashedValueVariant.bits | 3}, // partial key length 3
				},
				{ // key LE
					written: []byte{0x01, 0x23},
				},
				{ // children bitmap
					written: []byte{136, 0},
				},
			},
			wrappedErr: ErrEncodeHashedValueTooShort,
			errMessage: "hashed storage value too short: expected 32, got: 8",
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

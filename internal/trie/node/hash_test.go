// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package node

import (
	"io"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
)

func Test_MerkleValue(t *testing.T) {
	t.Parallel()

	testCases := map[string]struct {
		encoding      []byte
		writerBuilder func(ctrl *gomock.Controller) io.Writer
		errWrapped    error
		errMessage    string
	}{
		"small encoding": {
			encoding: []byte{1},
			writerBuilder: func(ctrl *gomock.Controller) io.Writer {
				writer := NewMockWriter(ctrl)
				writer.EXPECT().Write([]byte{1}).Return(1, nil)
				return writer
			},
		},
		"encoding write error": {
			encoding: []byte{1},
			writerBuilder: func(ctrl *gomock.Controller) io.Writer {
				writer := NewMockWriter(ctrl)
				writer.EXPECT().Write([]byte{1}).Return(0, errTest)
				return writer
			},
			errWrapped: errTest,
			errMessage: "writing encoding: test error",
		},
		"long encoding": {
			encoding: []byte{
				1, 2, 3, 4, 5, 6, 7, 8,
				9, 10, 11, 12, 13, 14, 15, 16,
				17, 18, 19, 20, 21, 22, 23, 24,
				25, 26, 27, 28, 29, 30, 31, 32, 33},
			writerBuilder: func(ctrl *gomock.Controller) io.Writer {
				writer := NewMockWriter(ctrl)
				writer.EXPECT().Write([]byte{
					0xfc, 0xd2, 0xd9, 0xac, 0xe8, 0x70, 0x52, 0x81,
					0x1d, 0x9f, 0x34, 0x27, 0xb5, 0x8f, 0xf3, 0x98,
					0xd2, 0xe9, 0xed, 0x83, 0xf3, 0x1, 0xbc, 0x7e,
					0xc1, 0xbe, 0x8b, 0x59, 0x39, 0x62, 0xf1, 0x7d,
				}).Return(32, nil)
				return writer
			},
		},
		"digest write error": {
			encoding: []byte{
				1, 2, 3, 4, 5, 6, 7, 8,
				9, 10, 11, 12, 13, 14, 15, 16,
				17, 18, 19, 20, 21, 22, 23, 24,
				25, 26, 27, 28, 29, 30, 31, 32, 33},
			writerBuilder: func(ctrl *gomock.Controller) io.Writer {
				writer := NewMockWriter(ctrl)
				writer.EXPECT().Write([]byte{
					0xfc, 0xd2, 0xd9, 0xac, 0xe8, 0x70, 0x52, 0x81,
					0x1d, 0x9f, 0x34, 0x27, 0xb5, 0x8f, 0xf3, 0x98,
					0xd2, 0xe9, 0xed, 0x83, 0xf3, 0x1, 0xbc, 0x7e,
					0xc1, 0xbe, 0x8b, 0x59, 0x39, 0x62, 0xf1, 0x7d,
				}).Return(0, errTest)
				return writer
			},
			errWrapped: errTest,
			errMessage: "writing digest: test error",
		},
	}

	for name, testCase := range testCases {
		testCase := testCase
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			ctrl := gomock.NewController(t)

			writer := testCase.writerBuilder(ctrl)

			err := MerkleValue(testCase.encoding, writer)

			assert.ErrorIs(t, err, testCase.errWrapped)
			if testCase.errWrapped != nil {
				assert.EqualError(t, err, testCase.errMessage)
			}
		})
	}
}

func Test_MerkleValueRoot(t *testing.T) {
	t.Parallel()

	testCases := map[string]struct {
		encoding      []byte
		writerBuilder func(ctrl *gomock.Controller) io.Writer
		errWrapped    error
		errMessage    string
	}{
		"digest write error": {
			encoding: []byte{1},
			writerBuilder: func(ctrl *gomock.Controller) io.Writer {
				writer := NewMockWriter(ctrl)
				writer.EXPECT().Write([]byte{
					0xee, 0x15, 0x5a, 0xce, 0x9c, 0x40, 0x29, 0x20,
					0x74, 0xcb, 0x6a, 0xff, 0x8c, 0x9c, 0xcd, 0xd2,
					0x73, 0xc8, 0x16, 0x48, 0xff, 0x11, 0x49, 0xef,
					0x36, 0xbc, 0xea, 0x6e, 0xbb, 0x8a, 0x3e, 0x25,
				}).Return(0, errTest)
				return writer
			},
			errWrapped: errTest,
			errMessage: "writing digest: test error",
		},
		"small encoding": {
			encoding: []byte{1},
			writerBuilder: func(ctrl *gomock.Controller) io.Writer {
				writer := NewMockWriter(ctrl)
				writer.EXPECT().Write([]byte{
					0xee, 0x15, 0x5a, 0xce, 0x9c, 0x40, 0x29, 0x20,
					0x74, 0xcb, 0x6a, 0xff, 0x8c, 0x9c, 0xcd, 0xd2,
					0x73, 0xc8, 0x16, 0x48, 0xff, 0x11, 0x49, 0xef,
					0x36, 0xbc, 0xea, 0x6e, 0xbb, 0x8a, 0x3e, 0x25,
				}).Return(32, nil)
				return writer
			},
		},
		"long encoding": {
			encoding: []byte{
				1, 2, 3, 4, 5, 6, 7, 8,
				9, 10, 11, 12, 13, 14, 15, 16,
				17, 18, 19, 20, 21, 22, 23, 24,
				25, 26, 27, 28, 29, 30, 31, 32, 33},
			writerBuilder: func(ctrl *gomock.Controller) io.Writer {
				writer := NewMockWriter(ctrl)
				writer.EXPECT().Write([]byte{
					0xfc, 0xd2, 0xd9, 0xac, 0xe8, 0x70, 0x52, 0x81,
					0x1d, 0x9f, 0x34, 0x27, 0xb5, 0x8f, 0xf3, 0x98,
					0xd2, 0xe9, 0xed, 0x83, 0xf3, 0x1, 0xbc, 0x7e,
					0xc1, 0xbe, 0x8b, 0x59, 0x39, 0x62, 0xf1, 0x7d,
				}).Return(32, nil)
				return writer
			},
		},
	}

	for name, testCase := range testCases {
		testCase := testCase
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			ctrl := gomock.NewController(t)

			writer := testCase.writerBuilder(ctrl)

			err := MerkleValueRoot(testCase.encoding, writer)

			assert.ErrorIs(t, err, testCase.errWrapped)
			if testCase.errWrapped != nil {
				assert.EqualError(t, err, testCase.errMessage)
			}
		})
	}
}

func Test_Node_CalculateMerkleValue(t *testing.T) {
	t.Parallel()

	testCases := map[string]struct {
		node        Node
		isRoot      bool
		merkleValue []byte
		errWrapped  error
		errMessage  string
	}{
		"cached merkle value": {
			node: Node{
				MerkleValue: []byte{1},
			},
			merkleValue: []byte{1},
		},
		"non root small encoding": {
			node: Node{
				Encoding: []byte{1},
			},
			merkleValue: []byte{1},
		},
		"root small encoding": {
			node: Node{
				Encoding: []byte{1},
			},
			isRoot: true,
			merkleValue: []byte{
				0xee, 0x15, 0x5a, 0xce, 0x9c, 0x40, 0x29, 0x20,
				0x74, 0xcb, 0x6a, 0xff, 0x8c, 0x9c, 0xcd, 0xd2,
				0x73, 0xc8, 0x16, 0x48, 0xff, 0x11, 0x49, 0xef,
				0x36, 0xbc, 0xea, 0x6e, 0xbb, 0x8a, 0x3e, 0x25},
		},
	}

	for name, testCase := range testCases {
		testCase := testCase
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			merkleValue, err := testCase.node.CalculateMerkleValue(testCase.isRoot)

			assert.ErrorIs(t, err, testCase.errWrapped)
			if testCase.errWrapped != nil {
				assert.EqualError(t, err, testCase.errMessage)
			}
			assert.Equal(t, testCase.merkleValue, merkleValue)
		})
	}
}

func Test_Node_EncodeAndHash(t *testing.T) {
	t.Parallel()

	testCases := map[string]struct {
		node         Node
		expectedNode Node
		encoding     []byte
		hash         []byte
		errWrapped   error
		errMessage   string
	}{
		"small leaf encoding": {
			node: Node{
				Key:      []byte{1},
				SubValue: []byte{2},
			},
			expectedNode: Node{
				Encoding:    []byte{0x41, 0x1, 0x4, 0x2},
				MerkleValue: []byte{0x41, 0x1, 0x4, 0x2},
			},
			encoding: []byte{0x41, 0x1, 0x4, 0x2},
			hash:     []byte{0x41, 0x1, 0x4, 0x2},
		},
		"leaf dirty with precomputed encoding and hash": {
			node: Node{
				Key:         []byte{1},
				SubValue:    []byte{2},
				Dirty:       true,
				Encoding:    []byte{3},
				MerkleValue: []byte{4},
			},
			expectedNode: Node{
				Encoding:    []byte{0x41, 0x1, 0x4, 0x2},
				MerkleValue: []byte{0x41, 0x1, 0x4, 0x2},
			},
			encoding: []byte{0x41, 0x1, 0x4, 0x2},
			hash:     []byte{0x41, 0x1, 0x4, 0x2},
		},
		"leaf not dirty with precomputed encoding and hash": {
			node: Node{
				Key:         []byte{1},
				SubValue:    []byte{2},
				Dirty:       false,
				Encoding:    []byte{3},
				MerkleValue: []byte{4},
			},
			expectedNode: Node{
				Key:         []byte{1},
				SubValue:    []byte{2},
				Encoding:    []byte{3},
				MerkleValue: []byte{4},
			},
			encoding: []byte{3},
			hash:     []byte{4},
		},
		"large leaf encoding": {
			node: Node{
				Key:      repeatBytes(65, 7),
				SubValue: []byte{0x01},
			},
			expectedNode: Node{
				Encoding:    []byte{0x7f, 0x2, 0x7, 0x77, 0x77, 0x77, 0x77, 0x77, 0x77, 0x77, 0x77, 0x77, 0x77, 0x77, 0x77, 0x77, 0x77, 0x77, 0x77, 0x77, 0x77, 0x77, 0x77, 0x77, 0x77, 0x77, 0x77, 0x77, 0x77, 0x77, 0x77, 0x77, 0x77, 0x77, 0x77, 0x4, 0x1}, //nolint:lll
				MerkleValue: []byte{0xd2, 0x1d, 0x43, 0x7, 0x18, 0x17, 0x1b, 0xf1, 0x45, 0x9c, 0xe5, 0x8f, 0xd7, 0x79, 0x82, 0xb, 0xc8, 0x5c, 0x8, 0x47, 0xfe, 0x6c, 0x99, 0xc5, 0xe9, 0x57, 0x87, 0x7, 0x1d, 0x2e, 0x24, 0x5d},                               //nolint:lll
			},
			encoding: []byte{0x7f, 0x2, 0x7, 0x77, 0x77, 0x77, 0x77, 0x77, 0x77, 0x77, 0x77, 0x77, 0x77, 0x77, 0x77, 0x77, 0x77, 0x77, 0x77, 0x77, 0x77, 0x77, 0x77, 0x77, 0x77, 0x77, 0x77, 0x77, 0x77, 0x77, 0x77, 0x77, 0x77, 0x77, 0x77, 0x4, 0x1}, //nolint:lll
			hash:     []byte{0xd2, 0x1d, 0x43, 0x7, 0x18, 0x17, 0x1b, 0xf1, 0x45, 0x9c, 0xe5, 0x8f, 0xd7, 0x79, 0x82, 0xb, 0xc8, 0x5c, 0x8, 0x47, 0xfe, 0x6c, 0x99, 0xc5, 0xe9, 0x57, 0x87, 0x7, 0x1d, 0x2e, 0x24, 0x5d},                               //nolint:lll
		},
		"empty branch": {
			node: Node{
				Children: make([]*Node, ChildrenCapacity),
			},
			expectedNode: Node{
				Children:    make([]*Node, ChildrenCapacity),
				Encoding:    []byte{0x80, 0x0, 0x0},
				MerkleValue: []byte{0x80, 0x0, 0x0},
			},
			encoding: []byte{0x80, 0x0, 0x0},
			hash:     []byte{0x80, 0x0, 0x0},
		},
		"small branch encoding": {
			node: Node{
				Children: make([]*Node, ChildrenCapacity),
				Key:      []byte{1},
				SubValue: []byte{2},
			},
			expectedNode: Node{
				Children:    make([]*Node, ChildrenCapacity),
				Encoding:    []byte{0xc1, 0x1, 0x0, 0x0, 0x4, 0x2},
				MerkleValue: []byte{0xc1, 0x1, 0x0, 0x0, 0x4, 0x2},
			},
			encoding: []byte{0xc1, 0x1, 0x0, 0x0, 0x4, 0x2},
			hash:     []byte{0xc1, 0x1, 0x0, 0x0, 0x4, 0x2},
		},
		"branch dirty with precomputed encoding and hash": {
			node: Node{
				Children:    make([]*Node, ChildrenCapacity),
				Key:         []byte{1},
				SubValue:    []byte{2},
				Dirty:       true,
				Encoding:    []byte{3},
				MerkleValue: []byte{4},
			},
			expectedNode: Node{
				Children:    make([]*Node, ChildrenCapacity),
				Encoding:    []byte{0xc1, 0x1, 0x0, 0x0, 0x4, 0x2},
				MerkleValue: []byte{0xc1, 0x1, 0x0, 0x0, 0x4, 0x2},
			},
			encoding: []byte{0xc1, 0x1, 0x0, 0x0, 0x4, 0x2},
			hash:     []byte{0xc1, 0x1, 0x0, 0x0, 0x4, 0x2},
		},
		"branch not dirty with precomputed encoding and hash": {
			node: Node{
				Children:    make([]*Node, ChildrenCapacity),
				Key:         []byte{1},
				SubValue:    []byte{2},
				Dirty:       false,
				Encoding:    []byte{3},
				MerkleValue: []byte{4},
			},
			expectedNode: Node{
				Children:    make([]*Node, ChildrenCapacity),
				Key:         []byte{1},
				SubValue:    []byte{2},
				Encoding:    []byte{3},
				MerkleValue: []byte{4},
			},
			encoding: []byte{3},
			hash:     []byte{4},
		},
		"large branch encoding": {
			node: Node{
				Children: make([]*Node, ChildrenCapacity),
				Key:      repeatBytes(65, 7),
			},
			expectedNode: Node{
				Children:    make([]*Node, ChildrenCapacity),
				Encoding:    []byte{0xbf, 0x2, 0x7, 0x77, 0x77, 0x77, 0x77, 0x77, 0x77, 0x77, 0x77, 0x77, 0x77, 0x77, 0x77, 0x77, 0x77, 0x77, 0x77, 0x77, 0x77, 0x77, 0x77, 0x77, 0x77, 0x77, 0x77, 0x77, 0x77, 0x77, 0x77, 0x77, 0x77, 0x77, 0x77, 0x0, 0x0}, //nolint:lll
				MerkleValue: []byte{0x6b, 0xd8, 0xcc, 0xac, 0x71, 0x77, 0x44, 0x17, 0xfe, 0xe0, 0xde, 0xda, 0xd5, 0x97, 0x6e, 0x69, 0xeb, 0xe9, 0xdd, 0x80, 0x1d, 0x4b, 0x51, 0xf1, 0x5b, 0xf3, 0x4a, 0x93, 0x27, 0x32, 0x2c, 0xb0},                           //nolint:lll
			},
			encoding: []byte{0xbf, 0x2, 0x7, 0x77, 0x77, 0x77, 0x77, 0x77, 0x77, 0x77, 0x77, 0x77, 0x77, 0x77, 0x77, 0x77, 0x77, 0x77, 0x77, 0x77, 0x77, 0x77, 0x77, 0x77, 0x77, 0x77, 0x77, 0x77, 0x77, 0x77, 0x77, 0x77, 0x77, 0x77, 0x77, 0x0, 0x0}, //nolint:lll
			hash:     []byte{0x6b, 0xd8, 0xcc, 0xac, 0x71, 0x77, 0x44, 0x17, 0xfe, 0xe0, 0xde, 0xda, 0xd5, 0x97, 0x6e, 0x69, 0xeb, 0xe9, 0xdd, 0x80, 0x1d, 0x4b, 0x51, 0xf1, 0x5b, 0xf3, 0x4a, 0x93, 0x27, 0x32, 0x2c, 0xb0},                           //nolint:lll
		},
	}

	for name, testCase := range testCases {
		testCase := testCase
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			encoding, hash, err := testCase.node.EncodeAndHash()

			assert.ErrorIs(t, err, testCase.errWrapped)
			if testCase.errWrapped != nil {
				assert.EqualError(t, err, testCase.errMessage)
			}
			assert.Equal(t, testCase.encoding, encoding)
			assert.Equal(t, testCase.hash, hash)
		})
	}
}

func Test_Node_EncodeAndHashRoot(t *testing.T) {
	t.Parallel()

	testCases := map[string]struct {
		node         Node
		expectedNode Node
		encoding     []byte
		hash         []byte
		errWrapped   error
		errMessage   string
	}{
		"leaf not dirty with precomputed encoding and hash": {
			node: Node{
				Key:         []byte{1},
				SubValue:    []byte{2},
				Dirty:       false,
				Encoding:    []byte{3},
				MerkleValue: []byte{4},
			},
			expectedNode: Node{
				Key:         []byte{1},
				SubValue:    []byte{2},
				Encoding:    []byte{3},
				MerkleValue: []byte{4},
			},
			encoding: []byte{3},
			hash:     []byte{4},
		},
		"small leaf encoding": {
			node: Node{
				Key:      []byte{1},
				SubValue: []byte{2},
			},
			expectedNode: Node{
				Encoding:    []byte{0x41, 0x1, 0x4, 0x2},
				MerkleValue: []byte{0x60, 0x51, 0x6d, 0xb, 0xb6, 0xe1, 0xbb, 0xfb, 0x12, 0x93, 0xf1, 0xb2, 0x76, 0xea, 0x95, 0x5, 0xe9, 0xf4, 0xa4, 0xe7, 0xd9, 0x8f, 0x62, 0xd, 0x5, 0x11, 0x5e, 0xb, 0x85, 0x27, 0x4a, 0xe1}, //nolint: lll
			},
			encoding: []byte{0x41, 0x1, 0x4, 0x2},
			hash:     []byte{0x60, 0x51, 0x6d, 0xb, 0xb6, 0xe1, 0xbb, 0xfb, 0x12, 0x93, 0xf1, 0xb2, 0x76, 0xea, 0x95, 0x5, 0xe9, 0xf4, 0xa4, 0xe7, 0xd9, 0x8f, 0x62, 0xd, 0x5, 0x11, 0x5e, 0xb, 0x85, 0x27, 0x4a, 0xe1}, // nolint: lll
		},
		"small branch encoding": {
			node: Node{
				Children: make([]*Node, ChildrenCapacity),
				Key:      []byte{1},
				SubValue: []byte{2},
			},
			expectedNode: Node{
				Children:    make([]*Node, ChildrenCapacity),
				Encoding:    []byte{0xc1, 0x1, 0x0, 0x0, 0x4, 0x2},
				MerkleValue: []byte{0x48, 0x3c, 0xf6, 0x87, 0xcc, 0x5a, 0x60, 0x42, 0xd3, 0xcf, 0xa6, 0x91, 0xe6, 0x88, 0xfb, 0xdc, 0x1b, 0x38, 0x39, 0x5d, 0x6, 0x0, 0xbf, 0xc3, 0xb, 0x4b, 0x5d, 0x6a, 0x37, 0xd9, 0xc5, 0x1c}, // nolint: lll
			},
			encoding: []byte{0xc1, 0x1, 0x0, 0x0, 0x4, 0x2},
			hash:     []byte{0x48, 0x3c, 0xf6, 0x87, 0xcc, 0x5a, 0x60, 0x42, 0xd3, 0xcf, 0xa6, 0x91, 0xe6, 0x88, 0xfb, 0xdc, 0x1b, 0x38, 0x39, 0x5d, 0x6, 0x0, 0xbf, 0xc3, 0xb, 0x4b, 0x5d, 0x6a, 0x37, 0xd9, 0xc5, 0x1c}, // nolint: lll
		},
	}

	for name, testCase := range testCases {
		testCase := testCase
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			encoding, hash, err := testCase.node.EncodeAndHashRoot()

			assert.ErrorIs(t, err, testCase.errWrapped)
			if testCase.errWrapped != nil {
				assert.EqualError(t, err, testCase.errMessage)
			}
			assert.Equal(t, testCase.encoding, encoding)
			assert.Equal(t, testCase.hash, hash)
		})
	}
}

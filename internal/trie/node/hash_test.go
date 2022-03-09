// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package node

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_Branch_SetEncodingAndHash(t *testing.T) {
	t.Parallel()

	branch := &Branch{
		Encoding:   []byte{2},
		HashDigest: []byte{3},
	}
	branch.SetEncodingAndHash([]byte{4}, []byte{5})

	expectedBranch := &Branch{
		Encoding:   []byte{4},
		HashDigest: []byte{5},
	}
	assert.Equal(t, expectedBranch, branch)
}

func Test_Branch_GetHash(t *testing.T) {
	t.Parallel()

	branch := &Branch{
		HashDigest: []byte{3},
	}
	hash := branch.GetHash()

	expectedHash := []byte{3}
	assert.Equal(t, expectedHash, hash)
}

func Test_Branch_EncodeAndHash(t *testing.T) {
	t.Parallel()

	testCases := map[string]struct {
		branch         *Branch
		expectedBranch *Branch
		encoding       []byte
		hash           []byte
		errWrapped     error
		errMessage     string
	}{
		"empty branch": {
			branch: &Branch{},
			expectedBranch: &Branch{
				Encoding:   []byte{0x80, 0x0, 0x0},
				HashDigest: []byte{0x80, 0x0, 0x0},
			},
			encoding: []byte{0x80, 0x0, 0x0},
			hash:     []byte{0x80, 0x0, 0x0},
		},
		"small branch encoding": {
			branch: &Branch{
				Key:   []byte{1},
				Value: []byte{2},
			},
			expectedBranch: &Branch{
				Encoding:   []byte{0xc1, 0x1, 0x0, 0x0, 0x4, 0x2},
				HashDigest: []byte{0xc1, 0x1, 0x0, 0x0, 0x4, 0x2},
			},
			encoding: []byte{0xc1, 0x1, 0x0, 0x0, 0x4, 0x2},
			hash:     []byte{0xc1, 0x1, 0x0, 0x0, 0x4, 0x2},
		},
		"branch dirty with precomputed encoding and hash": {
			branch: &Branch{
				Key:        []byte{1},
				Value:      []byte{2},
				Dirty:      true,
				Encoding:   []byte{3},
				HashDigest: []byte{4},
			},
			expectedBranch: &Branch{
				Encoding:   []byte{0xc1, 0x1, 0x0, 0x0, 0x4, 0x2},
				HashDigest: []byte{0xc1, 0x1, 0x0, 0x0, 0x4, 0x2},
			},
			encoding: []byte{0xc1, 0x1, 0x0, 0x0, 0x4, 0x2},
			hash:     []byte{0xc1, 0x1, 0x0, 0x0, 0x4, 0x2},
		},
		"branch not dirty with precomputed encoding and hash": {
			branch: &Branch{
				Key:        []byte{1},
				Value:      []byte{2},
				Dirty:      false,
				Encoding:   []byte{3},
				HashDigest: []byte{4},
			},
			expectedBranch: &Branch{
				Key:        []byte{1},
				Value:      []byte{2},
				Encoding:   []byte{3},
				HashDigest: []byte{4},
			},
			encoding: []byte{3},
			hash:     []byte{4},
		},
		"large branch encoding": {
			branch: &Branch{
				Key: repeatBytes(65, 7),
			},
			expectedBranch: &Branch{
				Encoding:   []byte{0xbf, 0x2, 0x7, 0x77, 0x77, 0x77, 0x77, 0x77, 0x77, 0x77, 0x77, 0x77, 0x77, 0x77, 0x77, 0x77, 0x77, 0x77, 0x77, 0x77, 0x77, 0x77, 0x77, 0x77, 0x77, 0x77, 0x77, 0x77, 0x77, 0x77, 0x77, 0x77, 0x77, 0x77, 0x77, 0x0, 0x0}, //nolint:lll
				HashDigest: []byte{0x6b, 0xd8, 0xcc, 0xac, 0x71, 0x77, 0x44, 0x17, 0xfe, 0xe0, 0xde, 0xda, 0xd5, 0x97, 0x6e, 0x69, 0xeb, 0xe9, 0xdd, 0x80, 0x1d, 0x4b, 0x51, 0xf1, 0x5b, 0xf3, 0x4a, 0x93, 0x27, 0x32, 0x2c, 0xb0},                           //nolint:lll
			},
			encoding: []byte{0xbf, 0x2, 0x7, 0x77, 0x77, 0x77, 0x77, 0x77, 0x77, 0x77, 0x77, 0x77, 0x77, 0x77, 0x77, 0x77, 0x77, 0x77, 0x77, 0x77, 0x77, 0x77, 0x77, 0x77, 0x77, 0x77, 0x77, 0x77, 0x77, 0x77, 0x77, 0x77, 0x77, 0x77, 0x77, 0x0, 0x0}, //nolint:lll
			hash:     []byte{0x6b, 0xd8, 0xcc, 0xac, 0x71, 0x77, 0x44, 0x17, 0xfe, 0xe0, 0xde, 0xda, 0xd5, 0x97, 0x6e, 0x69, 0xeb, 0xe9, 0xdd, 0x80, 0x1d, 0x4b, 0x51, 0xf1, 0x5b, 0xf3, 0x4a, 0x93, 0x27, 0x32, 0x2c, 0xb0},                           //nolint:lll
		},
	}

	for name, testCase := range testCases {
		testCase := testCase
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			encoding, hash, err := testCase.branch.EncodeAndHash(false)

			assert.ErrorIs(t, err, testCase.errWrapped)
			if testCase.errWrapped != nil {
				assert.EqualError(t, err, testCase.errMessage)
			}
			assert.Equal(t, testCase.encoding, encoding)
			assert.Equal(t, testCase.hash, hash)
		})
	}
}

func Test_Leaf_SetEncodingAndHash(t *testing.T) {
	t.Parallel()

	leaf := &Leaf{
		Encoding:   []byte{2},
		HashDigest: []byte{3},
	}
	leaf.SetEncodingAndHash([]byte{4}, []byte{5})

	expectedLeaf := &Leaf{
		Encoding:   []byte{4},
		HashDigest: []byte{5},
	}
	assert.Equal(t, expectedLeaf, leaf)
}

func Test_Leaf_GetHash(t *testing.T) {
	t.Parallel()

	leaf := &Leaf{
		HashDigest: []byte{3},
	}
	hash := leaf.GetHash()

	expectedHash := []byte{3}
	assert.Equal(t, expectedHash, hash)
}

func Test_Leaf_EncodeAndHash(t *testing.T) {
	t.Parallel()

	testCases := map[string]struct {
		leaf         *Leaf
		expectedLeaf *Leaf
		encoding     []byte
		hash         []byte
		errWrapped   error
		errMessage   string
	}{
		"empty leaf": {
			leaf: &Leaf{},
			expectedLeaf: &Leaf{
				Encoding:   []byte{0x40, 0x0},
				HashDigest: []byte{0x40, 0x0},
			},
			encoding: []byte{0x40, 0x0},
			hash:     []byte{0x40, 0x0},
		},
		"small leaf encoding": {
			leaf: &Leaf{
				Key:   []byte{1},
				Value: []byte{2},
			},
			expectedLeaf: &Leaf{
				Encoding:   []byte{0x41, 0x1, 0x4, 0x2},
				HashDigest: []byte{0x41, 0x1, 0x4, 0x2},
			},
			encoding: []byte{0x41, 0x1, 0x4, 0x2},
			hash:     []byte{0x41, 0x1, 0x4, 0x2},
		},
		"leaf dirty with precomputed encoding and hash": {
			leaf: &Leaf{
				Key:        []byte{1},
				Value:      []byte{2},
				Dirty:      true,
				Encoding:   []byte{3},
				HashDigest: []byte{4},
			},
			expectedLeaf: &Leaf{
				Encoding:   []byte{0x41, 0x1, 0x4, 0x2},
				HashDigest: []byte{0x41, 0x1, 0x4, 0x2},
			},
			encoding: []byte{0x41, 0x1, 0x4, 0x2},
			hash:     []byte{0x41, 0x1, 0x4, 0x2},
		},
		"leaf not dirty with precomputed encoding and hash": {
			leaf: &Leaf{
				Key:        []byte{1},
				Value:      []byte{2},
				Dirty:      false,
				Encoding:   []byte{3},
				HashDigest: []byte{4},
			},
			expectedLeaf: &Leaf{
				Key:        []byte{1},
				Value:      []byte{2},
				Encoding:   []byte{3},
				HashDigest: []byte{4},
			},
			encoding: []byte{3},
			hash:     []byte{4},
		},
		"large leaf encoding": {
			leaf: &Leaf{
				Key: repeatBytes(65, 7),
			},
			expectedLeaf: &Leaf{
				Encoding:   []byte{0x7f, 0x2, 0x7, 0x77, 0x77, 0x77, 0x77, 0x77, 0x77, 0x77, 0x77, 0x77, 0x77, 0x77, 0x77, 0x77, 0x77, 0x77, 0x77, 0x77, 0x77, 0x77, 0x77, 0x77, 0x77, 0x77, 0x77, 0x77, 0x77, 0x77, 0x77, 0x77, 0x77, 0x77, 0x77, 0x0}, //nolint:lll
				HashDigest: []byte{0xfb, 0xae, 0x31, 0x4b, 0xef, 0x31, 0x9, 0xc7, 0x62, 0x99, 0x9d, 0x40, 0x9b, 0xd4, 0xdc, 0x64, 0xe7, 0x39, 0x46, 0x8b, 0xd3, 0xaf, 0xe8, 0x63, 0x9d, 0xf9, 0x41, 0x40, 0x76, 0x40, 0x10, 0xa3},                       //nolint:lll
			},
			encoding: []byte{0x7f, 0x2, 0x7, 0x77, 0x77, 0x77, 0x77, 0x77, 0x77, 0x77, 0x77, 0x77, 0x77, 0x77, 0x77, 0x77, 0x77, 0x77, 0x77, 0x77, 0x77, 0x77, 0x77, 0x77, 0x77, 0x77, 0x77, 0x77, 0x77, 0x77, 0x77, 0x77, 0x77, 0x77, 0x77, 0x0}, //nolint:lll
			hash:     []byte{0xfb, 0xae, 0x31, 0x4b, 0xef, 0x31, 0x9, 0xc7, 0x62, 0x99, 0x9d, 0x40, 0x9b, 0xd4, 0xdc, 0x64, 0xe7, 0x39, 0x46, 0x8b, 0xd3, 0xaf, 0xe8, 0x63, 0x9d, 0xf9, 0x41, 0x40, 0x76, 0x40, 0x10, 0xa3},                       //nolint:lll
		},
	}

	for name, testCase := range testCases {
		testCase := testCase
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			encoding, hash, err := testCase.leaf.EncodeAndHash(false)

			assert.ErrorIs(t, err, testCase.errWrapped)
			if testCase.errWrapped != nil {
				assert.EqualError(t, err, testCase.errMessage)
			}
			assert.Equal(t, testCase.encoding, encoding)
			assert.Equal(t, testCase.hash, hash)
		})
	}
}

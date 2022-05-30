// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package node

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_Branch_IsDirty(t *testing.T) {
	t.Parallel()

	testCases := map[string]struct {
		branch *Branch
		dirty  bool
	}{
		"not dirty": {
			branch: &Branch{},
		},
		"dirty": {
			branch: &Branch{
				Dirty: true,
			},
			dirty: true,
		},
	}

	for name, testCase := range testCases {
		testCase := testCase
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			dirty := testCase.branch.IsDirty()

			assert.Equal(t, testCase.dirty, dirty)
		})
	}
}

func Test_Branch_SetDirty(t *testing.T) {
	t.Parallel()

	testCases := map[string]struct {
		branch   *Branch
		dirty    bool
		expected *Branch
	}{
		"not dirty to not dirty": {
			branch: &Branch{
				Encoding:   []byte{1},
				HashDigest: []byte{1},
			},
			expected: &Branch{
				Encoding:   []byte{1},
				HashDigest: []byte{1},
			},
		},
		"not dirty to dirty": {
			branch: &Branch{
				Encoding:   []byte{1},
				HashDigest: []byte{1},
			},
			dirty:    true,
			expected: &Branch{Dirty: true},
		},
		"dirty to not dirty": {
			branch: &Branch{
				Encoding:   []byte{1},
				HashDigest: []byte{1},
				Dirty:      true,
			},
			expected: &Branch{
				Encoding:   []byte{1},
				HashDigest: []byte{1},
			},
		},
		"dirty to dirty": {
			branch: &Branch{
				Encoding:   []byte{1},
				HashDigest: []byte{1},
				Dirty:      true,
			},
			dirty:    true,
			expected: &Branch{Dirty: true},
		},
	}

	for name, testCase := range testCases {
		testCase := testCase
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			testCase.branch.SetDirty(testCase.dirty)

			assert.Equal(t, testCase.expected, testCase.branch)
		})
	}
}

func Test_Leaf_IsDirty(t *testing.T) {
	t.Parallel()

	testCases := map[string]struct {
		leaf  *Leaf
		dirty bool
	}{
		"not dirty": {
			leaf: &Leaf{},
		},
		"dirty": {
			leaf: &Leaf{
				Dirty: true,
			},
			dirty: true,
		},
	}

	for name, testCase := range testCases {
		testCase := testCase
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			dirty := testCase.leaf.IsDirty()

			assert.Equal(t, testCase.dirty, dirty)
		})
	}
}

func Test_Leaf_SetDirty(t *testing.T) {
	t.Parallel()

	testCases := map[string]struct {
		leaf     *Leaf
		dirty    bool
		expected *Leaf
	}{
		"not dirty to not dirty": {
			leaf: &Leaf{
				Encoding:   []byte{1},
				HashDigest: []byte{1},
			},
			expected: &Leaf{
				Encoding:   []byte{1},
				HashDigest: []byte{1},
			},
		},
		"not dirty to dirty": {
			leaf: &Leaf{
				Encoding:   []byte{1},
				HashDigest: []byte{1},
			},
			dirty:    true,
			expected: &Leaf{Dirty: true},
		},
		"dirty to not dirty": {
			leaf: &Leaf{
				Encoding:   []byte{1},
				HashDigest: []byte{1},
				Dirty:      true,
			},
			expected: &Leaf{
				Encoding:   []byte{1},
				HashDigest: []byte{1},
			},
		},
		"dirty to dirty": {
			leaf: &Leaf{
				Encoding:   []byte{1},
				HashDigest: []byte{1},
				Dirty:      true,
			},
			dirty:    true,
			expected: &Leaf{Dirty: true},
		},
	}

	for name, testCase := range testCases {
		testCase := testCase
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			testCase.leaf.SetDirty(testCase.dirty)

			assert.Equal(t, testCase.expected, testCase.leaf)
		})
	}
}

// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package node

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_NewLeaf(t *testing.T) {
	t.Parallel()

	key := []byte{1, 2}
	value := []byte{3, 4}
	const dirty = true
	const generation = 9

	leaf := NewLeaf(key, value, dirty, generation)

	expectedLeaf := &Leaf{
		Key:        key,
		Value:      value,
		Dirty:      dirty,
		Generation: generation,
	}
	assert.Equal(t, expectedLeaf, leaf)

	// Check modifying passed slice modifies leaf slices
	key[0] = 11
	value[0] = 13
	assert.Equal(t, expectedLeaf, leaf)
}

func Test_Leaf_Type(t *testing.T) {
	t.Parallel()

	leaf := new(Leaf)

	Type := leaf.Type()

	assert.Equal(t, LeafType, Type)
}

func Test_Leaf_String(t *testing.T) {
	t.Parallel()

	testCases := map[string]struct {
		leaf *Leaf
		s    string
	}{
		"empty leaf": {
			leaf: &Leaf{},
			s: `Leaf
├── Generation: 0
├── Dirty: false
├── Key: nil
├── Value: nil
├── Calculated encoding: nil
└── Calculated digest: nil`,
		},
		"leaf with value smaller than 1024": {
			leaf: &Leaf{
				Key:   []byte{1, 2},
				Value: []byte{3, 4},
				Dirty: true,
			},
			s: `Leaf
├── Generation: 0
├── Dirty: true
├── Key: 0x0102
├── Value: 0x0304
├── Calculated encoding: nil
└── Calculated digest: nil`,
		},
		"leaf with value higher than 1024": {
			leaf: &Leaf{
				Key:   []byte{1, 2},
				Value: make([]byte, 1025),
				Dirty: true,
			},
			s: `Leaf
├── Generation: 0
├── Dirty: true
├── Key: 0x0102
├── Value: 0x0000000000000000...0000000000000000
├── Calculated encoding: nil
└── Calculated digest: nil`,
		},
	}

	for name, testCase := range testCases {
		testCase := testCase
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			s := testCase.leaf.String()

			assert.Equal(t, testCase.s, s)
		})
	}
}

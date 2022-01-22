// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package node

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_NewBranch(t *testing.T) {
	t.Parallel()

	key := []byte{1, 2}
	value := []byte{3, 4}
	const dirty = true
	const generation = 9

	branch := NewBranch(key, value, dirty, generation)

	expectedBranch := &Branch{
		Key:        key,
		Value:      value,
		Dirty:      dirty,
		Generation: generation,
	}
	assert.Equal(t, expectedBranch, branch)

	// Check modifying passed slice modifies branch slices
	key[0] = 11
	value[0] = 13
	assert.Equal(t, expectedBranch, branch)
}

func Test_Branch_Type(t *testing.T) {
	testCases := map[string]struct {
		branch *Branch
		Type   Type
	}{
		"nil value": {
			branch: &Branch{},
			Type:   BranchType,
		},
		"empty value": {
			branch: &Branch{
				Value: []byte{},
			},
			Type: BranchWithValueType,
		},
		"non empty value": {
			branch: &Branch{
				Value: []byte{1},
			},
			Type: BranchWithValueType,
		},
	}

	for name, testCase := range testCases {
		testCase := testCase
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			Type := testCase.branch.Type()

			assert.Equal(t, testCase.Type, Type)
		})
	}
}

func Test_Branch_String(t *testing.T) {
	t.Parallel()

	testCases := map[string]struct {
		branch *Branch
		s      string
	}{
		"empty branch": {
			branch: &Branch{},
			s:      "branch key=0x childrenBitmap=0 value=0x dirty=false",
		},
		"branch with value smaller than 1024": {
			branch: &Branch{
				Key:   []byte{1, 2},
				Value: []byte{3, 4},
				Dirty: true,
				Children: [16]Node{
					nil, nil, nil,
					&Leaf{},
					nil, nil, nil,
					&Branch{},
					nil, nil, nil,
					&Leaf{},
					nil, nil, nil, nil,
				},
			},
			s: "branch key=0x0102 childrenBitmap=100010001000 value=0x0304 dirty=true",
		},
		"branch with value higher than 1024": {
			branch: &Branch{
				Key:   []byte{1, 2},
				Value: make([]byte, 1025),
				Dirty: true,
				Children: [16]Node{
					nil, nil, nil,
					&Leaf{},
					nil, nil, nil,
					&Branch{},
					nil, nil, nil,
					&Leaf{},
					nil, nil, nil, nil,
				},
			},
			s: "branch key=0x0102 childrenBitmap=100010001000 " +
				"value (hashed)=0x307861663233363133353361303538646238383034626337353735323831663131663735313265326331346336373032393864306232336630396538386266333066 " + //nolint:lll
				"dirty=true",
		},
	}

	for name, testCase := range testCases {
		testCase := testCase
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			s := testCase.branch.String()

			assert.Equal(t, testCase.s, s)
		})
	}
}

// Copyright 2023 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package scale

import (
	"testing"

	"github.com/stretchr/testify/require"
)

type dummy struct {
	Field1 uint32
	Field2 [32]byte
}

func TestBTree(t *testing.T) {
	comparator := func(a, b interface{}) bool {
		v1 := a.(dummy)
		v2 := b.(dummy)
		return v1.Field1 < v2.Field1
	}

	// Create a BTree with 3 dummy items
	tree := NewBTree[dummy](comparator)
	tree.Set(dummy{Field1: 1})
	tree.Set(dummy{Field1: 2})
	tree.Set(dummy{Field1: 3})

	encoded, err := Marshal(tree)
	require.NoError(t, err)

	//let mut btree = BTreeMap::<u32, Hash>::new();
	//btree.insert(1, Hash::zero());
	//btree.insert(2, Hash::zero());
	//btree.insert(3, Hash::zero());
	//let encoded = btree.encode();
	//println!("encoded: {:?}", encoded);
	expectedEncoded := []byte{12,
		1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
		2, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
		3, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
	}
	require.Equal(t, expectedEncoded, encoded)

	expected := NewBTree[dummy](comparator)
	err = Unmarshal(encoded, expected)
	require.NoError(t, err)

	// Check that the expected BTree has the same items as the original
	actualTree := tree.GetTree()
	expectedTree := expected.GetTree()
	require.Equal(t, actualTree.Len(), expectedTree.Len())
	require.Equal(t, actualTree.ItemType, expectedTree.ItemType)
	require.Equal(t, actualTree.Min(), expectedTree.Min())
	require.Equal(t, actualTree.Max(), expectedTree.Max())
	require.Equal(t, actualTree.Get(dummy{Field1: 1}), expectedTree.Get(dummy{Field1: 1}))
	require.Equal(t, actualTree.Get(dummy{Field1: 2}), expectedTree.Get(dummy{Field1: 2}))
	require.Equal(t, actualTree.Get(dummy{Field1: 3}), expectedTree.Get(dummy{Field1: 3}))
}

func TestBTreeMap_Codec(t *testing.T) {
	btreeMap := NewBTreeMap[uint32, dummy](32)
	btreeMap.Set(uint32(1), dummy{Field1: 1})
	btreeMap.Set(uint32(2), dummy{Field1: 2})
	btreeMap.Set(uint32(3), dummy{Field1: 3})

	encoded, err := Marshal(btreeMap)
	require.NoError(t, err)

	expected := NewBTreeMap[uint32, dummy](32)
	err = Unmarshal(encoded, expected)
	require.NoError(t, err)

	require.Equal(t, btreeMap.Len(), expected.Len())
}

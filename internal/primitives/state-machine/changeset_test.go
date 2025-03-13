// Copyright 2025 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package statemachine

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/tidwall/btree"
)

type DrainedValue struct {
	string
	StorageValue
}
type Drained []DrainedValue

func TestDirtyKeysSetsPop(t *testing.T) {
	set1 := btree.Set[int]{}
	set1.Insert(1)
	set1.Insert(2)
	set1.Insert(3)

	set2 := btree.Set[int]{}
	set2.Insert(4)
	set2.Insert(5)
	set2.Insert(6)

	dirtyKeys := DirtyKeysSets[int]{
		set1,
		set2,
	}

	reverseSets := []btree.Set[int]{set2, set1}

	for i := 0; i < len(reverseSets); i++ {
		last, has := dirtyKeys.Pop()
		if !has {
			break
		}
		require.Equal(t, last, reverseSets[i])
	}
}

func TestNoTransactionWorks(t *testing.T) {
	changeSet := NewOverlayedChangeSet()

	require.Equal(t, uint(0), changeSet.TransactionDepth())

	extrinsic1 := uint32(1)
	extrinsic2 := uint32(2)
	extrinsic9 := uint32(9)

	changeSet.Set("key0", NewStorageValue([]byte("value0")), &extrinsic1)
	changeSet.Set("key1", NewStorageValue([]byte("value1")), &extrinsic2)
	changeSet.Set("key0", NewStorageValue([]byte("value0-1")), &extrinsic9)

	assertDrained(t, changeSet, Drained{
		{"key0", NewStorageValue([]byte("value0-1"))},
		{"key1", NewStorageValue([]byte("value1"))},
	})

}

func assertDrained(t *testing.T, is OverlayedChangeSet, expected Drained) {
	var drained Drained
	for k, v := range is.DrainCommited() {
		drained = append(drained, DrainedValue{k, v.value()})
	}

	require.Equal(t, expected, drained)
}

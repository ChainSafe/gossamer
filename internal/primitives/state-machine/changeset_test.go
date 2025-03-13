// Copyright 2025 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package statemachine

import (
	"maps"
	"slices"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/tidwall/btree"
)

type ChangesValue struct {
	key        string
	value      StorageValue
	extrinsics []uint32
}
type Changes []ChangesValue

type DrainedValue struct {
	string
	StorageValue
}
type Drained []DrainedValue

func extrinsic(value uint32) *uint32 {
	return &value
}

func assertChanges(t *testing.T, is OverlayedChangeSet, expected Changes) {
	var changes Changes
	for k, v := range is.Changes() {
		extrinsics := slices.Collect(maps.Keys(v.Extrinsics()))
		slices.Sort(extrinsics)
		if extrinsics == nil {
			extrinsics = []uint32{}
		}

		changes = append(changes, ChangesValue{k, v.Value().value(), extrinsics})
	}

	slices.SortFunc(changes, func(a, b ChangesValue) int {
		return strings.Compare(a.key, b.key)
	})

	require.Equal(t, expected, changes)
}

func assertDrainedChanges(t *testing.T, is OverlayedChangeSet, expected Changes) {
	var drained Drained
	for k, v := range is.DrainCommited() {
		drained = append(drained, DrainedValue{k, v.value()})
	}

	expect := Drained{}
	for _, v := range expected {
		expect = append(expect, DrainedValue{v.key, v.value})
	}

	require.Equal(t, expect, drained)
}

func assertDrained(t *testing.T, is OverlayedChangeSet, expected Drained) {
	var drained Drained
	for k, v := range is.DrainCommited() {
		drained = append(drained, DrainedValue{k, v.value()})
	}

	require.Equal(t, expected, drained)
}

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

	changeSet.Set("key0", NewStorageValue([]byte("val0")), extrinsic(1))
	changeSet.Set("key1", NewStorageValue([]byte("val1")), extrinsic(2))
	changeSet.Set("key0", NewStorageValue([]byte("val0-1")), extrinsic(9))

	assertDrained(t, changeSet, Drained{
		{"key0", NewStorageValue([]byte("val0-1"))},
		{"key1", NewStorageValue([]byte("val1"))},
	})
}

func TestTransactionWorks(t *testing.T) {
	changeSet := NewOverlayedChangeSet()
	require.Equal(t, uint(0), changeSet.TransactionDepth())

	// no transaction: commited on set
	changeSet.Set("key0", NewStorageValue([]byte("val0")), extrinsic(1))
	changeSet.Set("key1", NewStorageValue([]byte("val1")), extrinsic(1))
	changeSet.Set("key0", NewStorageValue([]byte("val0-1")), extrinsic(10))

	changeSet.StartTransaction()
	require.Equal(t, uint(1), changeSet.TransactionDepth())

	// we will commit that later
	changeSet.Set("key42", NewStorageValue([]byte("val42")), extrinsic(42))
	changeSet.Set("key99", NewStorageValue([]byte("val99")), extrinsic(99))

	changeSet.StartTransaction()
	require.Equal(t, uint(2), changeSet.TransactionDepth())

	// we will roll that back
	changeSet.Set("key42", NewStorageValue([]byte("val42-rolled")), extrinsic(421))
	changeSet.Set("key7", NewStorageValue([]byte("val7-rolled")), extrinsic(77))
	changeSet.Set("key0", NewStorageValue([]byte("val0-rolled")), extrinsic(1000))
	changeSet.Set("key5", NewStorageValue([]byte("val5-rolled")), nil)

	// allChanges contain all changes not only the committed ones.
	allChanges := Changes{
		{"key0", NewStorageValue([]byte("val0-rolled")), []uint32{1, 10, 1000}},
		{"key1", NewStorageValue([]byte("val1")), []uint32{1}},
		{"key42", NewStorageValue([]byte("val42-rolled")), []uint32{42, 421}},
		{"key5", NewStorageValue([]byte("val5-rolled")), []uint32{}},
		{"key7", NewStorageValue([]byte("val7-rolled")), []uint32{77}},
		{"key99", NewStorageValue([]byte("val99")), []uint32{99}},
	}

	assertChanges(t, changeSet, allChanges)

	// this should be no-op
	changeSet.StartTransaction()
	require.Equal(t, uint(3), changeSet.TransactionDepth())
	changeSet.StartTransaction()
	require.Equal(t, uint(4), changeSet.TransactionDepth())
	changeSet.RollbackTransaction()
	require.Equal(t, uint(3), changeSet.TransactionDepth())
	changeSet.CommitTransaction()
	require.Equal(t, uint(2), changeSet.TransactionDepth())
	assertChanges(t, changeSet, allChanges)

	// roll back our first transactions that actually contains something
	changeSet.RollbackTransaction()
	require.Equal(t, uint(1), changeSet.TransactionDepth())

	rollBack := Changes{
		{"key0", NewStorageValue([]byte("val0-1")), []uint32{1, 10}},
		{"key1", NewStorageValue([]byte("val1")), []uint32{1}},
		{"key42", NewStorageValue([]byte("val42")), []uint32{42}},
		{"key99", NewStorageValue([]byte("val99")), []uint32{99}},
	}
	assertChanges(t, changeSet, rollBack)
}

func TestTransactionCommitThenRollbackWorks(t *testing.T) {
	changeSet := NewOverlayedChangeSet()
	require.Equal(t, uint(0), changeSet.TransactionDepth())

	changeSet.Set("key0", NewStorageValue([]byte("val0")), extrinsic(1))
	changeSet.Set("key1", NewStorageValue([]byte("val1")), extrinsic(1))
	changeSet.Set("key0", NewStorageValue([]byte("val0-1")), extrinsic(10))

	changeSet.StartTransaction()
	require.Equal(t, uint(1), changeSet.TransactionDepth())

	changeSet.Set("key42", NewStorageValue([]byte("val42")), extrinsic(42))
	changeSet.Set("key99", NewStorageValue([]byte("val99")), extrinsic(99))

	changeSet.StartTransaction()
	require.Equal(t, uint(2), changeSet.TransactionDepth())

	changeSet.Set("key42", NewStorageValue([]byte("val42-rolled")), extrinsic(421))
	changeSet.Set("key7", NewStorageValue([]byte("val7-rolled")), extrinsic(77))
	changeSet.Set("key0", NewStorageValue([]byte("val0-rolled")), extrinsic(1000))
	changeSet.Set("key5", NewStorageValue([]byte("val5-rolled")), nil)

	allChanges := Changes{
		{"key0", NewStorageValue([]byte("val0-rolled")), []uint32{1, 10, 1000}},
		{"key1", NewStorageValue([]byte("val1")), []uint32{1}},
		{"key42", NewStorageValue([]byte("val42-rolled")), []uint32{42, 421}},
		{"key5", NewStorageValue([]byte("val5-rolled")), []uint32{}},
		{"key7", NewStorageValue([]byte("val7-rolled")), []uint32{77}},
		{"key99", NewStorageValue([]byte("val99")), []uint32{99}},
	}
	assertChanges(t, changeSet, allChanges)

	// this should be no-op
	changeSet.StartTransaction()
	require.Equal(t, uint(3), changeSet.TransactionDepth())
	changeSet.StartTransaction()
	require.Equal(t, uint(4), changeSet.TransactionDepth())
	changeSet.RollbackTransaction()
	require.Equal(t, uint(3), changeSet.TransactionDepth())
	changeSet.CommitTransaction()
	require.Equal(t, uint(2), changeSet.TransactionDepth())
	assertChanges(t, changeSet, allChanges)

	err := changeSet.CommitTransaction()
	require.NoError(t, err)
	require.Equal(t, uint(1), changeSet.TransactionDepth())

	assertChanges(t, changeSet, allChanges)

	changeSet.RollbackTransaction()
	require.Equal(t, uint(0), changeSet.TransactionDepth())

	rollBack := Changes{
		{"key0", NewStorageValue([]byte("val0-1")), []uint32{1, 10}},
		{"key1", NewStorageValue([]byte("val1")), []uint32{1}},
	}
	assertChanges(t, changeSet, rollBack)

	assertDrainedChanges(t, changeSet, rollBack)
}

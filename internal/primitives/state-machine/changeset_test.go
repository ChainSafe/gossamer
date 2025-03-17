// Copyright 2025 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package statemachine

import (
	"maps"
	"slices"
	"strings"
	"testing"

	"github.com/ChainSafe/gossamer/pkg/scale"
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

		changes = append(changes, ChangesValue{k, v.StorageValue(), extrinsics})
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

	changeSet.Set(StorageKey("key0"), StorageValue("val0"), extrinsic(1))
	changeSet.Set(StorageKey("key1"), StorageValue("val1"), extrinsic(2))
	changeSet.Set(StorageKey("key0"), StorageValue("val0-1"), extrinsic(9))

	assertDrained(t, changeSet, Drained{
		{"key0", StorageValue("val0-1")},
		{"key1", StorageValue("val1")},
	})
}

func TestTransactionWorks(t *testing.T) {
	changeSet := NewOverlayedChangeSet()
	require.Equal(t, uint(0), changeSet.TransactionDepth())

	// no transaction: committed on set
	changeSet.Set(StorageKey("key0"), StorageValue("val0"), extrinsic(1))
	changeSet.Set(StorageKey("key1"), StorageValue("val1"), extrinsic(1))
	changeSet.Set(StorageKey("key0"), StorageValue("val0-1"), extrinsic(10))

	changeSet.StartTransaction()
	require.Equal(t, uint(1), changeSet.TransactionDepth())

	// we will commit that later
	changeSet.Set(StorageKey("key42"), StorageValue("val42"), extrinsic(42))
	changeSet.Set(StorageKey("key99"), StorageValue("val99"), extrinsic(99))

	changeSet.StartTransaction()
	require.Equal(t, uint(2), changeSet.TransactionDepth())

	// we will roll that back
	changeSet.Set(StorageKey("key42"), StorageValue("val42-rolled"), extrinsic(421))
	changeSet.Set(StorageKey("key7"), StorageValue("val7-rolled"), extrinsic(77))
	changeSet.Set(StorageKey("key0"), StorageValue("val0-rolled"), extrinsic(1000))
	changeSet.Set(StorageKey("key5"), StorageValue("val5-rolled"), nil)

	// allChanges contain all changes not only the committed ones.
	allChanges := Changes{
		{"key0", StorageValue("val0-rolled"), []uint32{1, 10, 1000}},
		{"key1", StorageValue("val1"), []uint32{1}},
		{"key42", StorageValue("val42-rolled"), []uint32{42, 421}},
		{"key5", StorageValue("val5-rolled"), []uint32{}},
		{"key7", StorageValue("val7-rolled"), []uint32{77}},
		{"key99", StorageValue("val99"), []uint32{99}},
	}

	assertChanges(t, changeSet, allChanges)

	// this should be no-op
	changeSet.StartTransaction()
	require.Equal(t, uint(3), changeSet.TransactionDepth())
	changeSet.StartTransaction()
	require.Equal(t, uint(4), changeSet.TransactionDepth())
	changeSet.RollbackTransaction()
	require.Equal(t, uint(3), changeSet.TransactionDepth())
	require.NoError(t, changeSet.CommitTransaction())
	require.Equal(t, uint(2), changeSet.TransactionDepth())
	assertChanges(t, changeSet, allChanges)

	// roll back our first transactions that actually contains something
	require.NoError(t, changeSet.RollbackTransaction())
	require.Equal(t, uint(1), changeSet.TransactionDepth())

	rollBack := Changes{
		{"key0", StorageValue("val0-1"), []uint32{1, 10}},
		{"key1", StorageValue("val1"), []uint32{1}},
		{"key42", StorageValue("val42"), []uint32{42}},
		{"key99", StorageValue("val99"), []uint32{99}},
	}
	assertChanges(t, changeSet, rollBack)
}

func TestTransactionCommitThenRollbackWorks(t *testing.T) {
	changeSet := NewOverlayedChangeSet()
	require.Equal(t, uint(0), changeSet.TransactionDepth())

	changeSet.Set(StorageKey("key0"), StorageValue("val0"), extrinsic(1))
	changeSet.Set(StorageKey("key1"), StorageValue("val1"), extrinsic(1))
	changeSet.Set(StorageKey("key0"), StorageValue("val0-1"), extrinsic(10))

	changeSet.StartTransaction()
	require.Equal(t, uint(1), changeSet.TransactionDepth())

	changeSet.Set(StorageKey("key42"), StorageValue("val42"), extrinsic(42))
	changeSet.Set(StorageKey("key99"), StorageValue("val99"), extrinsic(99))

	changeSet.StartTransaction()
	require.Equal(t, uint(2), changeSet.TransactionDepth())

	changeSet.Set(StorageKey("key42"), StorageValue("val42-rolled"), extrinsic(421))
	changeSet.Set(StorageKey("key7"), StorageValue("val7-rolled"), extrinsic(77))
	changeSet.Set(StorageKey("key0"), StorageValue("val0-rolled"), extrinsic(1000))
	changeSet.Set(StorageKey("key5"), StorageValue("val5-rolled"), nil)

	allChanges := Changes{
		{"key0", StorageValue("val0-rolled"), []uint32{1, 10, 1000}},
		{"key1", StorageValue("val1"), []uint32{1}},
		{"key42", StorageValue("val42-rolled"), []uint32{42, 421}},
		{"key5", StorageValue("val5-rolled"), []uint32{}},
		{"key7", StorageValue("val7-rolled"), []uint32{77}},
		{"key99", StorageValue("val99"), []uint32{99}},
	}
	assertChanges(t, changeSet, allChanges)

	// this should be no-op
	changeSet.StartTransaction()
	require.Equal(t, uint(3), changeSet.TransactionDepth())
	changeSet.StartTransaction()
	require.Equal(t, uint(4), changeSet.TransactionDepth())
	changeSet.RollbackTransaction()
	require.Equal(t, uint(3), changeSet.TransactionDepth())
	require.NoError(t, changeSet.CommitTransaction())
	require.Equal(t, uint(2), changeSet.TransactionDepth())
	assertChanges(t, changeSet, allChanges)

	require.NoError(t, changeSet.CommitTransaction())
	require.Equal(t, uint(1), changeSet.TransactionDepth())

	assertChanges(t, changeSet, allChanges)

	require.NoError(t, changeSet.RollbackTransaction())
	require.Equal(t, uint(0), changeSet.TransactionDepth())

	rollBack := Changes{
		{"key0", StorageValue("val0-1"), []uint32{1, 10}},
		{"key1", StorageValue("val1"), []uint32{1}},
	}
	assertChanges(t, changeSet, rollBack)

	assertDrainedChanges(t, changeSet, rollBack)
}

func TestAppendWorks(t *testing.T) {
	changeSet := NewOverlayedChangeSet()
	require.Equal(t, uint(0), changeSet.TransactionDepth())

	init := func() StorageValue {
		val, err := scale.Marshal([][]byte{[]byte("valinit")})
		require.NoError(t, err)
		return StorageValue(val)
	}

	// committed set
	val0, err := scale.Marshal([][]byte{[]byte("val0")})
	require.NoError(t, err)

	changeSet.Set(StorageKey("key0"), StorageValue(val0), extrinsic(0))
	changeSet.Set(StorageKey("key1"), nil, extrinsic(1))
	allChanges := Changes{
		{"key0", StorageValue(val0), []uint32{0}},
		{"key1", nil, []uint32{1}},
	}

	assertChanges(t, changeSet, allChanges)

	appendValue, err := scale.Marshal([]byte("-modified"))
	require.NoError(t, err)

	changeSet.AppendStorage(StorageKey("key3"), StorageValue(appendValue), init, extrinsic(3))
	val3, err := scale.Marshal([][]byte{[]byte("valinit"), []byte("-modified")})
	require.NoError(t, err)

	allChanges = Changes{
		{"key0", StorageValue(val0), []uint32{0}},
		{"key1", nil, []uint32{1}},
		{"key3", val3, []uint32{3}},
	}

	assertChanges(t, changeSet, allChanges)

	changeSet.StartTransaction()
	require.Equal(t, uint(1), changeSet.TransactionDepth())
	changeSet.StartTransaction()
	require.Equal(t, uint(2), changeSet.TransactionDepth())

	// non existing value -> init value should be returned
	appendValue, err = scale.Marshal([]byte("-twice"))
	require.NoError(t, err)
	changeSet.AppendStorage(StorageKey("key3"), StorageValue(appendValue), init, extrinsic(15))
	// non existing value -> init value should be returned
	appendValue, err = scale.Marshal([]byte("-modified"))
	require.NoError(t, err)
	changeSet.AppendStorage(StorageKey("key2"), StorageValue(appendValue), init, extrinsic(2))
	// existing value should be reuse on append
	changeSet.AppendStorage(StorageKey("key0"), StorageValue(appendValue), init, extrinsic(10))

	// should work for deleted keys
	appendValue, err = scale.Marshal([]byte("-deleted-modified"))
	require.NoError(t, err)
	changeSet.AppendStorage(StorageKey("key1"), StorageValue(appendValue), init, extrinsic(20))

	val02, err := scale.Marshal([][]byte{[]byte("val0"), []byte("-modified")})
	require.NoError(t, err)

	val32, err := scale.Marshal([][]byte{[]byte("valinit"), []byte("-modified"), []byte("-twice")})
	require.NoError(t, err)

	val1, err := scale.Marshal([][]byte{[]byte("-deleted-modified")})
	require.NoError(t, err)

	allChanges = Changes{
		{"key0", StorageValue(val02), []uint32{0, 10}},
		{"key1", StorageValue(val1), []uint32{1, 20}},
		{"key2", StorageValue(val3), []uint32{2}},
		{"key3", StorageValue(val32), []uint32{3, 15}},
	}
	assertChanges(t, changeSet, allChanges)

	changeSet.StartTransaction()
	require.Equal(t, uint(3), changeSet.TransactionDepth())

	val33, err := scale.Marshal([][]byte{[]byte("valinit"), []byte("-modified"), []byte("-twice"), []byte("-2")})
	require.NoError(t, err)

	appendValue, err = scale.Marshal([]byte("-2"))
	require.NoError(t, err)
	changeSet.AppendStorage(StorageKey("key3"), StorageValue(appendValue), init, extrinsic(21))

	allChanges2 := Changes{
		{"key0", StorageValue(val02), []uint32{0, 10}},
		{"key1", StorageValue(val1), []uint32{1, 20}},
		{"key2", StorageValue(val3), []uint32{2}},
		{"key3", StorageValue(val33), []uint32{3, 15, 21}},
	}
	assertChanges(t, changeSet, allChanges2)

	require.NoError(t, changeSet.RollbackTransaction())
	require.Equal(t, uint(2), changeSet.TransactionDepth())

	assertChanges(t, changeSet, allChanges)

	changeSet.StartTransaction()
	require.Equal(t, uint(3), changeSet.TransactionDepth())

	val34, err := scale.Marshal([][]byte{[]byte("valinit"), []byte("-modified"), []byte("-twice"), []byte("-thrice")})
	require.NoError(t, err)

	appendValue, err = scale.Marshal([]byte("-thrice"))
	require.NoError(t, err)
	changeSet.AppendStorage(StorageKey("key3"), StorageValue(appendValue), init, extrinsic(25))

	allChanges = Changes{
		{"key0", StorageValue(val02), []uint32{0, 10}},
		{"key1", StorageValue(val1), []uint32{1, 20}},
		{"key2", StorageValue(val3), []uint32{2}},
		{"key3", StorageValue(val34), []uint32{3, 15, 25}},
	}
	assertChanges(t, changeSet, allChanges)

	require.NoError(t, changeSet.CommitTransaction())
	require.NoError(t, changeSet.CommitTransaction())

	require.Equal(t, uint(1), changeSet.TransactionDepth())
	assertChanges(t, changeSet, allChanges)

	require.NoError(t, changeSet.RollbackTransaction())
	require.Equal(t, uint(0), changeSet.TransactionDepth())

	rolledBack := Changes{
		{"key0", StorageValue(val0), []uint32{0}},
		{"key1", nil, []uint32{1}},
		{"key3", StorageValue(val3), []uint32{3}},
	}
	assertChanges(t, changeSet, rolledBack)
	assertDrainedChanges(t, changeSet, rolledBack)
}

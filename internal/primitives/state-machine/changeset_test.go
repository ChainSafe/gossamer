// Copyright 2025 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package statemachine

import (
	"bytes"
	"iter"
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

func assertChanges(t *testing.T, is *OverlayedChangeSet, expected Changes) {
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

func assertDrainedChanges(t *testing.T, is *OverlayedChangeSet, expected Changes) {
	var drained Drained
	for k, v := range is.DrainCommited() {
		drained = append(drained, DrainedValue{k, v.optionalValue()})
	}

	expect := Drained{}
	for _, v := range expected {
		expect = append(expect, DrainedValue{v.key, v.value})
	}

	require.Equal(t, expect, drained)
}

func assertDrained(t *testing.T, is *OverlayedChangeSet, expected Drained) {
	var drained Drained
	for k, v := range is.DrainCommited() {
		drained = append(drained, DrainedValue{k, v.optionalValue()})
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

	init := func() StorageValue { return StorageValue(scale.MustMarshal([][]byte{[]byte("valinit")})) }

	// committed set
	val0 := scale.MustMarshal([][]byte{[]byte("val0")})
	changeSet.Set(StorageKey("key0"), StorageValue(val0), extrinsic(0))
	changeSet.Set(StorageKey("key1"), nil, extrinsic(1))
	allChanges := Changes{
		{"key0", StorageValue(val0), []uint32{0}},
		{"key1", nil, []uint32{1}},
	}

	assertChanges(t, changeSet, allChanges)

	appendValue := scale.MustMarshal([]byte("-modified"))
	changeSet.AppendStorage(StorageKey("key3"), StorageValue(appendValue), init, extrinsic(3))
	val3 := scale.MustMarshal([][]byte{[]byte("valinit"), []byte("-modified")})

	allChanges = Changes{
		{"key0", StorageValue(val0), []uint32{0}},
		{"key1", nil, []uint32{1}},
		{"key3", val3, []uint32{3}},
	}

	assertChanges(t, changeSet, allChanges)

	changeSet.StartTransaction()
	require.Equal(t, uint(1), changeSet.TransactionDepth())
	{
		// transaction 1
		changeSet.StartTransaction()
		require.Equal(t, uint(2), changeSet.TransactionDepth())

		{
			// transaction 2
			// non existing value -> init value should be returned
			appendValue = scale.MustMarshal([]byte("-twice"))
			changeSet.AppendStorage(StorageKey("key3"), StorageValue(appendValue), init, extrinsic(15))
			// non existing value -> init value should be returned
			appendValue = scale.MustMarshal([]byte("-modified"))
			changeSet.AppendStorage(StorageKey("key2"), StorageValue(appendValue), init, extrinsic(2))
			// existing value should be reuse on append
			changeSet.AppendStorage(StorageKey("key0"), StorageValue(appendValue), init, extrinsic(10))

			// should work for deleted keys
			appendValue = scale.MustMarshal([]byte("-deleted-modified"))
			changeSet.AppendStorage(StorageKey("key1"), StorageValue(appendValue), init, extrinsic(20))

			val02 := scale.MustMarshal([][]byte{[]byte("val0"), []byte("-modified")})
			val32 := scale.MustMarshal([][]byte{[]byte("valinit"), []byte("-modified"), []byte("-twice")})
			val1 := scale.MustMarshal([][]byte{[]byte("-deleted-modified")})

			allChanges = Changes{
				{"key0", StorageValue(val02), []uint32{0, 10}},
				{"key1", StorageValue(val1), []uint32{1, 20}},
				{"key2", StorageValue(val3), []uint32{2}},
				{"key3", StorageValue(val32), []uint32{3, 15}},
			}
			assertChanges(t, changeSet, allChanges)

			changeSet.StartTransaction()
			require.Equal(t, uint(3), changeSet.TransactionDepth())
			{
				// transaction 3
				val33 := scale.MustMarshal([][]byte{[]byte("valinit"), []byte("-modified"), []byte("-twice"), []byte("-2")})
				appendValue = scale.MustMarshal([]byte("-2"))
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
			}
			changeSet.StartTransaction()
			require.Equal(t, uint(3), changeSet.TransactionDepth())
			{
				// new transaction 3
				val34 := scale.MustMarshal([][]byte{[]byte("valinit"), []byte("-modified"), []byte("-twice"), []byte("-thrice")})

				appendValue = scale.MustMarshal([]byte("-thrice"))
				changeSet.AppendStorage(StorageKey("key3"), StorageValue(appendValue), init, extrinsic(25))

				allChanges = Changes{
					{"key0", StorageValue(val02), []uint32{0, 10}},
					{"key1", StorageValue(val1), []uint32{1, 20}},
					{"key2", StorageValue(val3), []uint32{2}},
					{"key3", StorageValue(val34), []uint32{3, 15, 25}},
				}
				assertChanges(t, changeSet, allChanges)

				require.NoError(t, changeSet.CommitTransaction())
				require.Equal(t, uint(2), changeSet.TransactionDepth())
				assertChanges(t, changeSet, allChanges)
			}

			require.NoError(t, changeSet.CommitTransaction())
			require.Equal(t, uint(1), changeSet.TransactionDepth())
			assertChanges(t, changeSet, allChanges)
		}
		require.NoError(t, changeSet.RollbackTransaction())
		require.Equal(t, uint(0), changeSet.TransactionDepth())
	}

	rolledBack := Changes{
		{"key0", StorageValue(val0), []uint32{0}},
		{"key1", nil, []uint32{1}},
		{"key3", StorageValue(val3), []uint32{3}},
	}
	assertChanges(t, changeSet, rolledBack)
	assertDrainedChanges(t, changeSet, rolledBack)
}

func TestClearWorks(t *testing.T) {
	changeSet := NewOverlayedChangeSet()

	changeSet.Set(StorageKey("key0"), StorageValue("val0"), extrinsic(1))
	changeSet.Set(StorageKey("key1"), StorageValue("val1"), extrinsic(2))
	changeSet.Set(StorageKey("del1"), StorageValue("delval1"), extrinsic(3))
	changeSet.Set(StorageKey("del2"), StorageValue("delval2"), extrinsic(4))

	changeSet.StartTransaction()

	predicate := func(k []byte, ov *OverlayedValue) bool { return bytes.HasPrefix(k, []byte("del")) }
	changeSet.ClearWhere(predicate, extrinsic(5))

	allChanges := Changes{
		{"del1", nil, []uint32{3, 5}},
		{"del2", nil, []uint32{4, 5}},
		{"key0", StorageValue("val0"), []uint32{1}},
		{"key1", StorageValue("val1"), []uint32{2}},
	}
	assertChanges(t, changeSet, allChanges)

	changeSet.RollbackTransaction()

	allChanges = Changes{
		{"del1", StorageValue("delval1"), []uint32{3}},
		{"del2", StorageValue("delval2"), []uint32{4}},
		{"key0", StorageValue("val0"), []uint32{1}},
		{"key1", StorageValue("val1"), []uint32{2}},
	}
	assertChanges(t, changeSet, allChanges)
}

func TestNextChangeWorks(t *testing.T) {
	changeSet := NewOverlayedChangeSet()

	changeSet.Set(StorageKey("key0"), StorageValue("val0"), extrinsic(0))
	changeSet.Set(StorageKey("key1"), StorageValue("val1"), extrinsic(1))
	changeSet.Set(StorageKey("key2"), StorageValue("val2"), extrinsic(2))

	changeSet.StartTransaction()

	changeSet.Set(StorageKey("key3"), StorageValue("val3"), extrinsic(3))
	changeSet.Set(StorageKey("key4"), StorageValue("val4"), extrinsic(4))
	changeSet.Set(StorageKey("key11"), StorageValue("val11"), extrinsic(11))

	next, _ := iter.Pull2(changeSet.ChangesAfter(StorageKey("key0")))

	k, v, _ := next()
	require.Equal(t, k, StorageKey("key1"))
	require.Equal(t, v.StorageValue(), StorageValue("val1"))

	next, _ = iter.Pull2(changeSet.ChangesAfter(StorageKey("key1")))

	k, v, _ = next()
	require.Equal(t, k, StorageKey("key11"))
	require.Equal(t, v.StorageValue(), StorageValue("val11"))

	next, _ = iter.Pull2(changeSet.ChangesAfter(StorageKey("key11")))

	k, v, _ = next()
	require.Equal(t, k, StorageKey("key2"))
	require.Equal(t, v.StorageValue(), StorageValue("val2"))

	next, _ = iter.Pull2(changeSet.ChangesAfter(StorageKey("key2")))

	k, v, _ = next()
	require.Equal(t, k, StorageKey("key3"))
	require.Equal(t, v.StorageValue(), StorageValue("val3"))

	next, _ = iter.Pull2(changeSet.ChangesAfter(StorageKey("key3")))

	k, v, _ = next()
	require.Equal(t, k, StorageKey("key4"))
	require.Equal(t, v.StorageValue(), StorageValue("val4"))

	_, _, has := next()
	require.False(t, has)

	changeSet.RollbackTransaction()

	next, _ = iter.Pull2(changeSet.ChangesAfter(StorageKey("key0")))

	k, v, _ = next()
	require.Equal(t, k, StorageKey("key1"))
	require.Equal(t, v.StorageValue(), StorageValue("val1"))

	next, _ = iter.Pull2(changeSet.ChangesAfter(StorageKey("key1")))

	k, v, _ = next()
	require.Equal(t, k, StorageKey("key2"))
	require.Equal(t, v.StorageValue(), StorageValue("val2"))

	next, _ = iter.Pull2(changeSet.ChangesAfter(StorageKey("key11")))

	k, v, _ = next()
	require.Equal(t, k, StorageKey("key2"))
	require.Equal(t, v.StorageValue(), StorageValue("val2"))

	next, _ = iter.Pull2(changeSet.ChangesAfter(StorageKey("key2")))
	_, _, has = next()
	require.False(t, has)

	next, _ = iter.Pull2(changeSet.ChangesAfter(StorageKey("key3")))
	_, _, has = next()
	require.False(t, has)

	next, _ = iter.Pull2(changeSet.ChangesAfter(StorageKey("key4")))
	_, _, has = next()
	require.False(t, has)
}

func TestNoOpenTxCommitErrors(t *testing.T) {
	changeSet := NewOverlayedChangeSet()
	require.Equal(t, uint(0), changeSet.TransactionDepth())
	require.Error(t, changeSet.CommitTransaction(), errorNoOpenTransaction)
}

func TestNoOpenTxRollbackErrors(t *testing.T) {
	changeSet := NewOverlayedChangeSet()
	require.Equal(t, uint(0), changeSet.TransactionDepth())
	require.Error(t, changeSet.RollbackTransaction(), errorNoOpenTransaction)
}

func TestUnbalancedTransactionsError(t *testing.T) {
	changeSet := NewOverlayedChangeSet()
	changeSet.StartTransaction()
	require.NoError(t, changeSet.CommitTransaction())

	require.Error(t, changeSet.CommitTransaction(), errorNoOpenTransaction)
}

func TestDrainWithOpenTransactionPanics(t *testing.T) {
	changeSet := NewOverlayedChangeSet()
	changeSet.StartTransaction()
	require.Panics(t, func() { changeSet.DrainCommited() })
}

func TestRuntimeCannotCloseClientTx(t *testing.T) {
	changeSet := NewOverlayedChangeSet()
	changeSet.StartTransaction()
	require.NoError(t, changeSet.EnterRuntime())

	changeSet.StartTransaction()
	require.NoError(t, changeSet.CommitTransaction())

	require.Error(t, changeSet.CommitTransaction(), errorNoOpenTransaction)
	require.Error(t, changeSet.RollbackTransaction(), errorNoOpenTransaction)
}

func TestExitRuntimeClosesRuntimeTx(t *testing.T) {
	changeSet := NewOverlayedChangeSet()
	changeSet.StartTransaction()
	changeSet.Set(StorageKey("key0"), StorageValue("val0"), extrinsic(1))

	require.NoError(t, changeSet.EnterRuntime())
	changeSet.StartTransaction()
	changeSet.Set(StorageKey("key1"), StorageValue("val1"), extrinsic(2))
	changeSet.ExitRuntime()

	changeSet.CommitTransaction()
	require.Equal(t, uint(0), changeSet.TransactionDepth())

	assertDrained(t, changeSet, Drained{
		{"key0", StorageValue("val0")},
	})
}

func TestEnterExitRuntimeFailsWhenAlreadyInRequestedMode(t *testing.T) {
	changeSet := NewOverlayedChangeSet()

	require.Error(t, changeSet.ExitRuntime(), errorNotInRuntime)
	require.NoError(t, changeSet.EnterRuntime())
	require.Error(t, changeSet.EnterRuntime(), errorAlreadyInRuntime)
	require.NoError(t, changeSet.ExitRuntime())
	require.Error(t, changeSet.ExitRuntime(), errorNotInRuntime)
}

func TestRestoreAppendToParent(t *testing.T) {
	changeSet := NewOverlayedChangeSet()
	key := "akey"

	from := 50 // 1 byte len
	to := 100  // 2 byte len
	defaultInit := func() StorageValue { return StorageValue{} }
	for i := 0; i < from; i++ {
		changeSet.AppendStorage(StorageKey(key), StorageValue([]byte{byte(i)}), defaultInit, nil)
	}

	// materialised
	encoded := changeSet.Get(key).StorageValue()
	encodedFromLen := scale.MustMarshal(uint(from))
	require.Equal(t, 1, len(encodedFromLen))
	require.True(t, bytes.HasPrefix(encoded, encodedFromLen))
	encodedFrom := encoded[:]

	changeSet.StartTransaction()

	for i := from; i < to; i++ {
		changeSet.AppendStorage(StorageKey(key), StorageValue([]byte{byte(i)}), defaultInit, nil)
	}

	// materialised
	encoded = changeSet.Get(key).StorageValue()
	encodedToLen := scale.MustMarshal(uint(to))
	require.Equal(t, 2, len(encodedToLen))
	require.True(t, bytes.HasPrefix(encoded, encodedToLen))

	changeSet.RollbackTransaction()

	encoded = changeSet.Get(key).StorageValue()
	require.Equal(t, encodedFrom, encoded)
}

func TestRestoreInitialSetAfterAppendToParent(t *testing.T) {
	changeSet := NewOverlayedChangeSet()
	key := "akey"

	defaultInit := func() StorageValue { return StorageValue{} }
	initialData := StorageValue(scale.MustMarshal(bytes.Repeat([]byte{1}, 50)))

	changeSet.Set(StorageKey(key), initialData, nil)
	changeSet.StartTransaction()

	// Append until we require 2 bytes for the length prefix.
	for i := 0; i < 50; i++ {
		changeSet.AppendStorage(StorageKey(key), StorageValue([]byte{byte(i)}), defaultInit, nil)
	}

	// Materialise the value.
	encoded := changeSet.Get(key).StorageValue()
	encodedToLen := scale.MustMarshal(uint(100))

	require.Equal(t, 2, len(encodedToLen))
	require.True(t, bytes.HasPrefix(encoded, encodedToLen))

	require.NoError(t, changeSet.RollbackTransaction())

	encoded = changeSet.Get(key).StorageValue()
	require.Equal(t, initialData, encoded)
}

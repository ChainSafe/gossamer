// Copyright 2025 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package overlayedchanges

import (
	"bytes"
	"iter"
	"testing"

	"github.com/ChainSafe/gossamer/pkg/scale"
	"github.com/stretchr/testify/require"
)

type ChangesValue struct {
	key        StorageKey
	value      StorageValue
	extrinsics []uint32
}
type Changes []ChangesValue

type DrainedValue struct {
	key   StorageKey
	value StorageValue
}
type Drained []DrainedValue

func extrinsic(value uint32) *uint32 {
	return &value
}

func assertChanges(t *testing.T, is overlayedChangeSet, expected Changes) {
	var changes Changes
	for k, v := range is.Changes() {
		extrinsics := v.Extrinsics().Keys()
		changes = append(changes, ChangesValue{k, v.Value(), extrinsics})
	}

	require.Equal(t, expected, changes)
}

func assertDrainedChanges(t *testing.T, is overlayedChangeSet, expected Changes) {
	var drained Drained
	for k, v := range is.DrainCommitted() {
		drained = append(drained, DrainedValue{k, v.value()})
	}

	expect := Drained{}
	for _, v := range expected {
		expect = append(expect, DrainedValue{v.key, v.value})
	}

	require.Equal(t, expect, drained)
}

func assertDrained(t *testing.T, is overlayedChangeSet, expected Drained) {
	var drained Drained
	for k, v := range is.DrainCommitted() {
		drained = append(drained, DrainedValue{k, v.value()})
	}

	require.Equal(t, expected, drained)
}

func TestDirtyKeysSetsPop(t *testing.T) {
	set1 := make(map[int]struct{})
	set1[1] = struct{}{}
	set1[2] = struct{}{}
	set1[3] = struct{}{}

	set2 := make(map[int]struct{})
	set2[4] = struct{}{}
	set2[5] = struct{}{}
	set2[6] = struct{}{}

	dirtyKeys := dirtyKeysSets[int]{
		set1,
		set2,
	}

	reverseSets := []map[int]struct{}{set2, set1}

	for i := 0; i < len(reverseSets); i++ {
		last, has := dirtyKeys.Pop()
		if !has {
			break
		}
		require.Equal(t, last, reverseSets[i])
	}
}

func TestNoTransactionWorks(t *testing.T) {
	changeSet := newOverlayedChangeSet()
	require.Equal(t, uint(0), changeSet.TransactionDepth())

	changeSet.set(StorageKey("key0"), StorageValue("val0"), extrinsic(1))
	changeSet.set(StorageKey("key1"), StorageValue("val1"), extrinsic(2))
	changeSet.set(StorageKey("key0"), StorageValue("val0-1"), extrinsic(9))

	assertDrained(t, changeSet, Drained{
		{StorageKey("key0"), StorageValue("val0-1")},
		{StorageKey("key1"), StorageValue("val1")},
	})
}

func TestTransactionWorks(t *testing.T) {
	changeSet := newOverlayedChangeSet()
	require.Equal(t, uint(0), changeSet.TransactionDepth())

	// no transaction: committed on set
	changeSet.set(StorageKey("key0"), StorageValue("val0"), extrinsic(1))
	changeSet.set(StorageKey("key1"), StorageValue("val1"), extrinsic(1))
	changeSet.set(StorageKey("key0"), StorageValue("val0-1"), extrinsic(10))

	changeSet.StartTransaction()
	require.Equal(t, uint(1), changeSet.TransactionDepth())

	// we will commit that later
	changeSet.set(StorageKey("key42"), StorageValue("val42"), extrinsic(42))
	changeSet.set(StorageKey("key99"), StorageValue("val99"), extrinsic(99))

	changeSet.StartTransaction()
	require.Equal(t, uint(2), changeSet.TransactionDepth())

	// we will roll that back
	changeSet.set(StorageKey("key42"), StorageValue("val42-rolled"), extrinsic(421))
	changeSet.set(StorageKey("key7"), StorageValue("val7-rolled"), extrinsic(77))
	changeSet.set(StorageKey("key0"), StorageValue("val0-rolled"), extrinsic(1000))
	changeSet.set(StorageKey("key5"), StorageValue("val5-rolled"), nil)

	// allChanges contain all changes not only the committed ones.
	allChanges := Changes{
		{StorageKey("key0"), StorageValue("val0-rolled"), []uint32{1, 10, 1000}},
		{StorageKey("key1"), StorageValue("val1"), []uint32{1}},
		{StorageKey("key42"), StorageValue("val42-rolled"), []uint32{42, 421}},
		{StorageKey("key5"), StorageValue("val5-rolled"), []uint32{}},
		{StorageKey("key7"), StorageValue("val7-rolled"), []uint32{77}},
		{StorageKey("key99"), StorageValue("val99"), []uint32{99}},
	}

	assertChanges(t, changeSet, allChanges)

	// this should be no-op
	changeSet.StartTransaction()
	require.Equal(t, uint(3), changeSet.TransactionDepth())
	changeSet.StartTransaction()
	require.Equal(t, uint(4), changeSet.TransactionDepth())
	changeSet.rollbackTransaction()
	require.Equal(t, uint(3), changeSet.TransactionDepth())
	require.NoError(t, changeSet.commitTransaction())
	require.Equal(t, uint(2), changeSet.TransactionDepth())
	assertChanges(t, changeSet, allChanges)

	// roll back our first transactions that actually contains something
	require.NoError(t, changeSet.rollbackTransaction())
	require.Equal(t, uint(1), changeSet.TransactionDepth())

	rollBack := Changes{
		{StorageKey("key0"), StorageValue("val0-1"), []uint32{1, 10}},
		{StorageKey("key1"), StorageValue("val1"), []uint32{1}},
		{StorageKey("key42"), StorageValue("val42"), []uint32{42}},
		{StorageKey("key99"), StorageValue("val99"), []uint32{99}},
	}
	assertChanges(t, changeSet, rollBack)
}

func TestTransactionCommitThenRollbackWorks(t *testing.T) {
	changeSet := newOverlayedChangeSet()
	require.Equal(t, uint(0), changeSet.TransactionDepth())

	changeSet.set(StorageKey("key0"), StorageValue("val0"), extrinsic(1))
	changeSet.set(StorageKey("key1"), StorageValue("val1"), extrinsic(1))
	changeSet.set(StorageKey("key0"), StorageValue("val0-1"), extrinsic(10))

	changeSet.StartTransaction()
	require.Equal(t, uint(1), changeSet.TransactionDepth())

	changeSet.set(StorageKey("key42"), StorageValue("val42"), extrinsic(42))
	changeSet.set(StorageKey("key99"), StorageValue("val99"), extrinsic(99))

	changeSet.StartTransaction()
	require.Equal(t, uint(2), changeSet.TransactionDepth())

	changeSet.set(StorageKey("key42"), StorageValue("val42-rolled"), extrinsic(421))
	changeSet.set(StorageKey("key7"), StorageValue("val7-rolled"), extrinsic(77))
	changeSet.set(StorageKey("key0"), StorageValue("val0-rolled"), extrinsic(1000))
	changeSet.set(StorageKey("key5"), StorageValue("val5-rolled"), nil)

	allChanges := Changes{
		{StorageKey("key0"), StorageValue("val0-rolled"), []uint32{1, 10, 1000}},
		{StorageKey("key1"), StorageValue("val1"), []uint32{1}},
		{StorageKey("key42"), StorageValue("val42-rolled"), []uint32{42, 421}},
		{StorageKey("key5"), StorageValue("val5-rolled"), []uint32{}},
		{StorageKey("key7"), StorageValue("val7-rolled"), []uint32{77}},
		{StorageKey("key99"), StorageValue("val99"), []uint32{99}},
	}
	assertChanges(t, changeSet, allChanges)

	// this should be no-op
	changeSet.StartTransaction()
	require.Equal(t, uint(3), changeSet.TransactionDepth())
	changeSet.StartTransaction()
	require.Equal(t, uint(4), changeSet.TransactionDepth())
	changeSet.rollbackTransaction()
	require.Equal(t, uint(3), changeSet.TransactionDepth())
	require.NoError(t, changeSet.commitTransaction())
	require.Equal(t, uint(2), changeSet.TransactionDepth())
	assertChanges(t, changeSet, allChanges)

	require.NoError(t, changeSet.commitTransaction())
	require.Equal(t, uint(1), changeSet.TransactionDepth())

	assertChanges(t, changeSet, allChanges)

	require.NoError(t, changeSet.rollbackTransaction())
	require.Equal(t, uint(0), changeSet.TransactionDepth())

	rollBack := Changes{
		{StorageKey("key0"), StorageValue("val0-1"), []uint32{1, 10}},
		{StorageKey("key1"), StorageValue("val1"), []uint32{1}},
	}
	assertChanges(t, changeSet, rollBack)

	assertDrainedChanges(t, changeSet, rollBack)
}

func TestAppendWorks(t *testing.T) {
	changeSet := newOverlayedChangeSet()
	require.Equal(t, uint(0), changeSet.TransactionDepth())

	init := func() StorageValue {
		return StorageValue(scale.MustMarshal([][]byte{[]byte("valinit")}))
	}

	// committed set
	val0 := scale.MustMarshal([][]byte{[]byte("val0")})
	changeSet.set(StorageKey("key0"), StorageValue(val0), extrinsic(0))
	changeSet.set(StorageKey("key1"), nil, extrinsic(1))
	allChanges := Changes{
		{StorageKey("key0"), StorageValue(val0), []uint32{0}},
		{StorageKey("key1"), nil, []uint32{1}},
	}

	assertChanges(t, changeSet, allChanges)

	appendValue := scale.MustMarshal([]byte("-modified"))
	changeSet.appendStorage(StorageKey("key3"), StorageValue(appendValue), init, extrinsic(3))
	val3 := scale.MustMarshal([][]byte{[]byte("valinit"), []byte("-modified")})

	allChanges = Changes{
		{StorageKey("key0"), StorageValue(val0), []uint32{0}},
		{StorageKey("key1"), nil, []uint32{1}},
		{StorageKey("key3"), StorageValue(val3), []uint32{3}},
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
			changeSet.appendStorage(StorageKey("key3"), StorageValue(appendValue), init, extrinsic(15))
			// non existing value -> init value should be returned
			appendValue = scale.MustMarshal([]byte("-modified"))
			changeSet.appendStorage(StorageKey("key2"), StorageValue(appendValue), init, extrinsic(2))
			// existing value should be reuse on append
			changeSet.appendStorage(StorageKey("key0"), StorageValue(appendValue), init, extrinsic(10))

			// should work for deleted keys
			appendValue = scale.MustMarshal([]byte("-deleted-modified"))
			changeSet.appendStorage(StorageKey("key1"), StorageValue(appendValue), init, extrinsic(20))

			val02 := scale.MustMarshal([][]byte{[]byte("val0"), []byte("-modified")})
			val32 := scale.MustMarshal([][]byte{[]byte("valinit"), []byte("-modified"), []byte("-twice")})
			val1 := scale.MustMarshal([][]byte{[]byte("-deleted-modified")})

			allChanges = Changes{
				{StorageKey("key0"), StorageValue(val02), []uint32{0, 10}},
				{StorageKey("key1"), StorageValue(val1), []uint32{1, 20}},
				{StorageKey("key2"), StorageValue(val3), []uint32{2}},
				{StorageKey("key3"), StorageValue(val32), []uint32{3, 15}},
			}
			assertChanges(t, changeSet, allChanges)

			changeSet.StartTransaction()
			require.Equal(t, uint(3), changeSet.TransactionDepth())
			{
				// transaction 3
				val33 := scale.MustMarshal([][]byte{[]byte("valinit"), []byte("-modified"), []byte("-twice"), []byte("-2")})
				appendValue = scale.MustMarshal([]byte("-2"))
				changeSet.appendStorage(StorageKey("key3"), StorageValue(appendValue), init, extrinsic(21))

				allChanges2 := Changes{
					{StorageKey("key0"), StorageValue(val02), []uint32{0, 10}},
					{StorageKey("key1"), StorageValue(val1), []uint32{1, 20}},
					{StorageKey("key2"), StorageValue(val3), []uint32{2}},
					{StorageKey("key3"), StorageValue(val33), []uint32{3, 15, 21}},
				}
				assertChanges(t, changeSet, allChanges2)

				require.NoError(t, changeSet.rollbackTransaction())
				require.Equal(t, uint(2), changeSet.TransactionDepth())
				assertChanges(t, changeSet, allChanges)
			}
			changeSet.StartTransaction()
			require.Equal(t, uint(3), changeSet.TransactionDepth())
			{
				// new transaction 3
				val34 := scale.MustMarshal([][]byte{[]byte("valinit"), []byte("-modified"), []byte("-twice"), []byte("-thrice")})

				appendValue = scale.MustMarshal([]byte("-thrice"))
				changeSet.appendStorage(StorageKey("key3"), StorageValue(appendValue), init, extrinsic(25))

				allChanges = Changes{
					{StorageKey("key0"), StorageValue(val02), []uint32{0, 10}},
					{StorageKey("key1"), StorageValue(val1), []uint32{1, 20}},
					{StorageKey("key2"), StorageValue(val3), []uint32{2}},
					{StorageKey("key3"), StorageValue(val34), []uint32{3, 15, 25}},
				}
				assertChanges(t, changeSet, allChanges)

				require.NoError(t, changeSet.commitTransaction())
				require.Equal(t, uint(2), changeSet.TransactionDepth())
				assertChanges(t, changeSet, allChanges)
			}

			require.NoError(t, changeSet.commitTransaction())
			require.Equal(t, uint(1), changeSet.TransactionDepth())
			assertChanges(t, changeSet, allChanges)
		}
		require.NoError(t, changeSet.rollbackTransaction())
		require.Equal(t, uint(0), changeSet.TransactionDepth())
	}

	rolledBack := Changes{
		{StorageKey("key0"), StorageValue(val0), []uint32{0}},
		{StorageKey("key1"), nil, []uint32{1}},
		{StorageKey("key3"), StorageValue(val3), []uint32{3}},
	}
	assertChanges(t, changeSet, rolledBack)
	assertDrainedChanges(t, changeSet, rolledBack)
}

func TestClearWorks(t *testing.T) {
	changeSet := newOverlayedChangeSet()

	changeSet.set(StorageKey("key0"), StorageValue("val0"), extrinsic(1))
	changeSet.set(StorageKey("key1"), StorageValue("val1"), extrinsic(2))
	changeSet.set(StorageKey("del1"), StorageValue("delval1"), extrinsic(3))
	changeSet.set(StorageKey("del2"), StorageValue("delval2"), extrinsic(4))

	changeSet.StartTransaction()

	predicate := func(k []byte, ov *overlayedValue) bool { return bytes.HasPrefix(k, []byte("del")) }
	changeSet.clearWhere(predicate, extrinsic(5))

	allChanges := Changes{
		{StorageKey("del1"), nil, []uint32{3, 5}},
		{StorageKey("del2"), nil, []uint32{4, 5}},
		{StorageKey("key0"), StorageValue("val0"), []uint32{1}},
		{StorageKey("key1"), StorageValue("val1"), []uint32{2}},
	}
	assertChanges(t, changeSet, allChanges)

	changeSet.rollbackTransaction()

	allChanges = Changes{
		{StorageKey("del1"), StorageValue("delval1"), []uint32{3}},
		{StorageKey("del2"), StorageValue("delval2"), []uint32{4}},
		{StorageKey("key0"), StorageValue("val0"), []uint32{1}},
		{StorageKey("key1"), StorageValue("val1"), []uint32{2}},
	}
	assertChanges(t, changeSet, allChanges)
}

func TestNextChangeWorks(t *testing.T) {
	changeSet := newOverlayedChangeSet()

	changeSet.set(StorageKey("key0"), StorageValue("val0"), extrinsic(0))
	changeSet.set(StorageKey("key1"), StorageValue("val1"), extrinsic(1))
	changeSet.set(StorageKey("key2"), StorageValue("val2"), extrinsic(2))

	changeSet.StartTransaction()

	changeSet.set(StorageKey("key3"), StorageValue("val3"), extrinsic(3))
	changeSet.set(StorageKey("key4"), StorageValue("val4"), extrinsic(4))
	changeSet.set(StorageKey("key11"), StorageValue("val11"), extrinsic(11))

	next, _ := iter.Pull2(changeSet.changesAfter(StorageKey("key0")))

	k, v, _ := next()
	require.Equal(t, k, StorageKey("key1"))
	require.Equal(t, v.Value(), StorageValue("val1"))

	next, _ = iter.Pull2(changeSet.changesAfter(StorageKey("key1")))

	k, v, _ = next()
	require.Equal(t, k, StorageKey("key11"))
	require.Equal(t, v.Value(), StorageValue("val11"))

	next, _ = iter.Pull2(changeSet.changesAfter(StorageKey("key11")))

	k, v, _ = next()
	require.Equal(t, k, StorageKey("key2"))
	require.Equal(t, v.Value(), StorageValue("val2"))

	next, _ = iter.Pull2(changeSet.changesAfter(StorageKey("key2")))

	k, v, _ = next()
	require.Equal(t, k, StorageKey("key3"))
	require.Equal(t, v.Value(), StorageValue("val3"))

	next, _ = iter.Pull2(changeSet.changesAfter(StorageKey("key3")))

	k, v, _ = next()
	require.Equal(t, k, StorageKey("key4"))
	require.Equal(t, v.Value(), StorageValue("val4"))

	_, _, has := next()
	require.False(t, has)

	changeSet.rollbackTransaction()

	next, _ = iter.Pull2(changeSet.changesAfter(StorageKey("key0")))

	k, v, _ = next()
	require.Equal(t, k, StorageKey("key1"))
	require.Equal(t, v.Value(), StorageValue("val1"))

	next, _ = iter.Pull2(changeSet.changesAfter(StorageKey("key1")))

	k, v, _ = next()
	require.Equal(t, k, StorageKey("key2"))
	require.Equal(t, v.Value(), StorageValue("val2"))

	next, _ = iter.Pull2(changeSet.changesAfter(StorageKey("key11")))

	k, v, _ = next()
	require.Equal(t, k, StorageKey("key2"))
	require.Equal(t, v.Value(), StorageValue("val2"))

	next, _ = iter.Pull2(changeSet.changesAfter(StorageKey("key2")))
	_, _, has = next()
	require.False(t, has)

	next, _ = iter.Pull2(changeSet.changesAfter(StorageKey("key3")))
	_, _, has = next()
	require.False(t, has)

	next, _ = iter.Pull2(changeSet.changesAfter(StorageKey("key4")))
	_, _, has = next()
	require.False(t, has)
}

func TestNoOpenTxCommitErrors(t *testing.T) {
	changeSet := newOverlayedChangeSet()
	require.Equal(t, uint(0), changeSet.TransactionDepth())
	require.Error(t, changeSet.commitTransaction(), errorNoOpenTransaction)
}

func TestNoOpenTxRollbackErrors(t *testing.T) {
	changeSet := newOverlayedChangeSet()
	require.Equal(t, uint(0), changeSet.TransactionDepth())
	require.Error(t, changeSet.rollbackTransaction(), errorNoOpenTransaction)
}

func TestUnbalancedTransactionsError(t *testing.T) {
	changeSet := newOverlayedChangeSet()
	changeSet.StartTransaction()
	require.NoError(t, changeSet.commitTransaction())

	require.Error(t, changeSet.commitTransaction(), errorNoOpenTransaction)
}

func TestDrainWithOpenTransactionPanics(t *testing.T) {
	changeSet := newOverlayedChangeSet()
	changeSet.StartTransaction()
	require.Panics(t, func() { changeSet.DrainCommitted() })
}

func TestRuntimeCannotCloseClientTx(t *testing.T) {
	changeSet := newOverlayedChangeSet()
	changeSet.StartTransaction()
	require.NoError(t, changeSet.enterRuntime())

	changeSet.StartTransaction()
	require.NoError(t, changeSet.commitTransaction())

	require.Error(t, changeSet.commitTransaction(), errorNoOpenTransaction)
	require.Error(t, changeSet.rollbackTransaction(), errorNoOpenTransaction)
}

func TestExitRuntimeClosesRuntimeTx(t *testing.T) {
	changeSet := newOverlayedChangeSet()
	changeSet.StartTransaction()
	changeSet.set(StorageKey("key0"), StorageValue("val0"), extrinsic(1))

	require.NoError(t, changeSet.enterRuntime())
	changeSet.StartTransaction()
	changeSet.set(StorageKey("key1"), StorageValue("val1"), extrinsic(2))
	changeSet.exitRuntime()

	changeSet.commitTransaction()
	require.Equal(t, uint(0), changeSet.TransactionDepth())

	assertDrained(t, changeSet, Drained{
		{StorageKey("key0"), StorageValue("val0")},
	})
}

func TestEnterExitRuntimeFailsWhenAlreadyInRequestedMode(t *testing.T) {
	changeSet := newOverlayedChangeSet()

	require.Error(t, changeSet.exitRuntime(), errorNotInRuntime)
	require.NoError(t, changeSet.enterRuntime())
	require.Error(t, changeSet.enterRuntime(), errorAlreadyInRuntime)
	require.NoError(t, changeSet.exitRuntime())
	require.Error(t, changeSet.exitRuntime(), errorNotInRuntime)
}

func TestRestoreAppendToParent(t *testing.T) {
	changeSet := newOverlayedChangeSet()
	key := "akey"

	from := 50 // 1 byte len
	to := 100  // 2 byte len
	defaultInit := func() StorageValue { return StorageValue{} }
	for i := 0; i < from; i++ {
		changeSet.appendStorage(StorageKey(key), StorageValue([]byte{byte(i)}), defaultInit, nil)
	}

	// materialised
	encoded, has := changeSet.Get(key)
	require.True(t, has)
	encodedFromLen := scale.MustMarshal(uint(from))
	require.Equal(t, 1, len(encodedFromLen))
	require.True(t, bytes.HasPrefix(encoded.Value(), encodedFromLen))
	encodedFrom := encoded.Value()[:]

	changeSet.StartTransaction()

	for i := from; i < to; i++ {
		changeSet.appendStorage(StorageKey(key), StorageValue([]byte{byte(i)}), defaultInit, nil)
	}

	// materialised
	encoded, has = changeSet.Get(key)
	require.True(t, has)
	encodedToLen := scale.MustMarshal(uint(to))
	require.Equal(t, 2, len(encodedToLen))
	require.True(t, bytes.HasPrefix(encoded.Value(), encodedToLen))

	changeSet.rollbackTransaction()

	encoded, has = changeSet.Get(key)
	require.True(t, has)
	require.Equal(t, encodedFrom, encoded.Value())
}

func TestRestoreInitialSetAfterAppendToParent(t *testing.T) {
	changeSet := newOverlayedChangeSet()
	key := "akey"

	defaultInit := func() StorageValue { return StorageValue{} }
	initialData := StorageValue(scale.MustMarshal(bytes.Repeat([]byte{1}, 50)))

	changeSet.set(StorageKey(key), initialData, nil)
	changeSet.StartTransaction()

	// Append until we require 2 bytes for the length prefix.
	for i := 0; i < 50; i++ {
		changeSet.appendStorage(StorageKey(key), StorageValue([]byte{byte(i)}), defaultInit, nil)
	}

	// Materialise the value.
	encoded, has := changeSet.Get(key)
	require.True(t, has)
	encodedToLen := scale.MustMarshal(uint(100))

	require.Equal(t, 2, len(encodedToLen))
	require.True(t, bytes.HasPrefix(encoded.Value(), encodedToLen))

	require.NoError(t, changeSet.rollbackTransaction())

	encoded, has = changeSet.Get(key)
	require.True(t, has)
	require.Equal(t, initialData, encoded.Value())
}

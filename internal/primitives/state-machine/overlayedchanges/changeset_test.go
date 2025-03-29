// Copyright 2025 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package overlayedchanges

import (
	"bytes"
	"iter"
	"slices"
	"strings"
	"testing"

	"github.com/ChainSafe/gossamer/internal/primitives/state-machine/backend"
	"github.com/ChainSafe/gossamer/pkg/scale"
	"github.com/stretchr/testify/require"
)

type ChangesValue struct {
	key        string
	value      backend.StorageValue
	extrinsics []uint32
}
type Changes []ChangesValue

type DrainedValue struct {
	key   string
	value backend.StorageValue
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

	slices.SortFunc(changes, func(a, b ChangesValue) int {
		return strings.Compare(a.key, b.key)
	})

	require.Equal(t, expected, changes)
}

func assertDrainedChanges(t *testing.T, is overlayedChangeSet, expected Changes) {
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

func assertDrained(t *testing.T, is overlayedChangeSet, expected Drained) {
	var drained Drained
	for k, v := range is.DrainCommited() {
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

	changeSet.set(backend.StorageKey("key0"), backend.StorageValue("val0"), extrinsic(1))
	changeSet.set(backend.StorageKey("key1"), backend.StorageValue("val1"), extrinsic(2))
	changeSet.set(backend.StorageKey("key0"), backend.StorageValue("val0-1"), extrinsic(9))

	assertDrained(t, changeSet, Drained{
		{"key0", backend.StorageValue("val0-1")},
		{"key1", backend.StorageValue("val1")},
	})
}

func TestTransactionWorks(t *testing.T) {
	changeSet := newOverlayedChangeSet()
	require.Equal(t, uint(0), changeSet.TransactionDepth())

	// no transaction: committed on set
	changeSet.set(backend.StorageKey("key0"), backend.StorageValue("val0"), extrinsic(1))
	changeSet.set(backend.StorageKey("key1"), backend.StorageValue("val1"), extrinsic(1))
	changeSet.set(backend.StorageKey("key0"), backend.StorageValue("val0-1"), extrinsic(10))

	changeSet.StartTransaction()
	require.Equal(t, uint(1), changeSet.TransactionDepth())

	// we will commit that later
	changeSet.set(backend.StorageKey("key42"), backend.StorageValue("val42"), extrinsic(42))
	changeSet.set(backend.StorageKey("key99"), backend.StorageValue("val99"), extrinsic(99))

	changeSet.StartTransaction()
	require.Equal(t, uint(2), changeSet.TransactionDepth())

	// we will roll that back
	changeSet.set(backend.StorageKey("key42"), backend.StorageValue("val42-rolled"), extrinsic(421))
	changeSet.set(backend.StorageKey("key7"), backend.StorageValue("val7-rolled"), extrinsic(77))
	changeSet.set(backend.StorageKey("key0"), backend.StorageValue("val0-rolled"), extrinsic(1000))
	changeSet.set(backend.StorageKey("key5"), backend.StorageValue("val5-rolled"), nil)

	// allChanges contain all changes not only the committed ones.
	allChanges := Changes{
		{"key0", backend.StorageValue("val0-rolled"), []uint32{1, 10, 1000}},
		{"key1", backend.StorageValue("val1"), []uint32{1}},
		{"key42", backend.StorageValue("val42-rolled"), []uint32{42, 421}},
		{"key5", backend.StorageValue("val5-rolled"), []uint32{}},
		{"key7", backend.StorageValue("val7-rolled"), []uint32{77}},
		{"key99", backend.StorageValue("val99"), []uint32{99}},
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
		{"key0", backend.StorageValue("val0-1"), []uint32{1, 10}},
		{"key1", backend.StorageValue("val1"), []uint32{1}},
		{"key42", backend.StorageValue("val42"), []uint32{42}},
		{"key99", backend.StorageValue("val99"), []uint32{99}},
	}
	assertChanges(t, changeSet, rollBack)
}

func TestTransactionCommitThenRollbackWorks(t *testing.T) {
	changeSet := newOverlayedChangeSet()
	require.Equal(t, uint(0), changeSet.TransactionDepth())

	changeSet.set(backend.StorageKey("key0"), backend.StorageValue("val0"), extrinsic(1))
	changeSet.set(backend.StorageKey("key1"), backend.StorageValue("val1"), extrinsic(1))
	changeSet.set(backend.StorageKey("key0"), backend.StorageValue("val0-1"), extrinsic(10))

	changeSet.StartTransaction()
	require.Equal(t, uint(1), changeSet.TransactionDepth())

	changeSet.set(backend.StorageKey("key42"), backend.StorageValue("val42"), extrinsic(42))
	changeSet.set(backend.StorageKey("key99"), backend.StorageValue("val99"), extrinsic(99))

	changeSet.StartTransaction()
	require.Equal(t, uint(2), changeSet.TransactionDepth())

	changeSet.set(backend.StorageKey("key42"), backend.StorageValue("val42-rolled"), extrinsic(421))
	changeSet.set(backend.StorageKey("key7"), backend.StorageValue("val7-rolled"), extrinsic(77))
	changeSet.set(backend.StorageKey("key0"), backend.StorageValue("val0-rolled"), extrinsic(1000))
	changeSet.set(backend.StorageKey("key5"), backend.StorageValue("val5-rolled"), nil)

	allChanges := Changes{
		{"key0", backend.StorageValue("val0-rolled"), []uint32{1, 10, 1000}},
		{"key1", backend.StorageValue("val1"), []uint32{1}},
		{"key42", backend.StorageValue("val42-rolled"), []uint32{42, 421}},
		{"key5", backend.StorageValue("val5-rolled"), []uint32{}},
		{"key7", backend.StorageValue("val7-rolled"), []uint32{77}},
		{"key99", backend.StorageValue("val99"), []uint32{99}},
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
		{"key0", backend.StorageValue("val0-1"), []uint32{1, 10}},
		{"key1", backend.StorageValue("val1"), []uint32{1}},
	}
	assertChanges(t, changeSet, rollBack)

	assertDrainedChanges(t, changeSet, rollBack)
}

func TestAppendWorks(t *testing.T) {
	changeSet := newOverlayedChangeSet()
	require.Equal(t, uint(0), changeSet.TransactionDepth())

	init := func() backend.StorageValue {
		return backend.StorageValue(scale.MustMarshal([][]byte{[]byte("valinit")}))
	}

	// committed set
	val0 := scale.MustMarshal([][]byte{[]byte("val0")})
	changeSet.set(backend.StorageKey("key0"), backend.StorageValue(val0), extrinsic(0))
	changeSet.set(backend.StorageKey("key1"), nil, extrinsic(1))
	allChanges := Changes{
		{"key0", backend.StorageValue(val0), []uint32{0}},
		{"key1", nil, []uint32{1}},
	}

	assertChanges(t, changeSet, allChanges)

	appendValue := scale.MustMarshal([]byte("-modified"))
	changeSet.appendStorage(backend.StorageKey("key3"), backend.StorageValue(appendValue), init, extrinsic(3))
	val3 := scale.MustMarshal([][]byte{[]byte("valinit"), []byte("-modified")})

	allChanges = Changes{
		{"key0", backend.StorageValue(val0), []uint32{0}},
		{"key1", nil, []uint32{1}},
		{"key3", backend.StorageValue(val3), []uint32{3}},
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
			changeSet.appendStorage(backend.StorageKey("key3"), backend.StorageValue(appendValue), init, extrinsic(15))
			// non existing value -> init value should be returned
			appendValue = scale.MustMarshal([]byte("-modified"))
			changeSet.appendStorage(backend.StorageKey("key2"), backend.StorageValue(appendValue), init, extrinsic(2))
			// existing value should be reuse on append
			changeSet.appendStorage(backend.StorageKey("key0"), backend.StorageValue(appendValue), init, extrinsic(10))

			// should work for deleted keys
			appendValue = scale.MustMarshal([]byte("-deleted-modified"))
			changeSet.appendStorage(backend.StorageKey("key1"), backend.StorageValue(appendValue), init, extrinsic(20))

			val02 := scale.MustMarshal([][]byte{[]byte("val0"), []byte("-modified")})
			val32 := scale.MustMarshal([][]byte{[]byte("valinit"), []byte("-modified"), []byte("-twice")})
			val1 := scale.MustMarshal([][]byte{[]byte("-deleted-modified")})

			allChanges = Changes{
				{"key0", backend.StorageValue(val02), []uint32{0, 10}},
				{"key1", backend.StorageValue(val1), []uint32{1, 20}},
				{"key2", backend.StorageValue(val3), []uint32{2}},
				{"key3", backend.StorageValue(val32), []uint32{3, 15}},
			}
			assertChanges(t, changeSet, allChanges)

			changeSet.StartTransaction()
			require.Equal(t, uint(3), changeSet.TransactionDepth())
			{
				// transaction 3
				val33 := scale.MustMarshal([][]byte{[]byte("valinit"), []byte("-modified"), []byte("-twice"), []byte("-2")})
				appendValue = scale.MustMarshal([]byte("-2"))
				changeSet.appendStorage(backend.StorageKey("key3"), backend.StorageValue(appendValue), init, extrinsic(21))

				allChanges2 := Changes{
					{"key0", backend.StorageValue(val02), []uint32{0, 10}},
					{"key1", backend.StorageValue(val1), []uint32{1, 20}},
					{"key2", backend.StorageValue(val3), []uint32{2}},
					{"key3", backend.StorageValue(val33), []uint32{3, 15, 21}},
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
				changeSet.appendStorage(backend.StorageKey("key3"), backend.StorageValue(appendValue), init, extrinsic(25))

				allChanges = Changes{
					{"key0", backend.StorageValue(val02), []uint32{0, 10}},
					{"key1", backend.StorageValue(val1), []uint32{1, 20}},
					{"key2", backend.StorageValue(val3), []uint32{2}},
					{"key3", backend.StorageValue(val34), []uint32{3, 15, 25}},
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
		{"key0", backend.StorageValue(val0), []uint32{0}},
		{"key1", nil, []uint32{1}},
		{"key3", backend.StorageValue(val3), []uint32{3}},
	}
	assertChanges(t, changeSet, rolledBack)
	assertDrainedChanges(t, changeSet, rolledBack)
}

func TestClearWorks(t *testing.T) {
	changeSet := newOverlayedChangeSet()

	changeSet.set(backend.StorageKey("key0"), backend.StorageValue("val0"), extrinsic(1))
	changeSet.set(backend.StorageKey("key1"), backend.StorageValue("val1"), extrinsic(2))
	changeSet.set(backend.StorageKey("del1"), backend.StorageValue("delval1"), extrinsic(3))
	changeSet.set(backend.StorageKey("del2"), backend.StorageValue("delval2"), extrinsic(4))

	changeSet.StartTransaction()

	predicate := func(k []byte, ov *overlayedValue) bool { return bytes.HasPrefix(k, []byte("del")) }
	changeSet.clearWhere(predicate, extrinsic(5))

	allChanges := Changes{
		{"del1", nil, []uint32{3, 5}},
		{"del2", nil, []uint32{4, 5}},
		{"key0", backend.StorageValue("val0"), []uint32{1}},
		{"key1", backend.StorageValue("val1"), []uint32{2}},
	}
	assertChanges(t, changeSet, allChanges)

	changeSet.rollbackTransaction()

	allChanges = Changes{
		{"del1", backend.StorageValue("delval1"), []uint32{3}},
		{"del2", backend.StorageValue("delval2"), []uint32{4}},
		{"key0", backend.StorageValue("val0"), []uint32{1}},
		{"key1", backend.StorageValue("val1"), []uint32{2}},
	}
	assertChanges(t, changeSet, allChanges)
}

func TestNextChangeWorks(t *testing.T) {
	changeSet := newOverlayedChangeSet()

	changeSet.set(backend.StorageKey("key0"), backend.StorageValue("val0"), extrinsic(0))
	changeSet.set(backend.StorageKey("key1"), backend.StorageValue("val1"), extrinsic(1))
	changeSet.set(backend.StorageKey("key2"), backend.StorageValue("val2"), extrinsic(2))

	changeSet.StartTransaction()

	changeSet.set(backend.StorageKey("key3"), backend.StorageValue("val3"), extrinsic(3))
	changeSet.set(backend.StorageKey("key4"), backend.StorageValue("val4"), extrinsic(4))
	changeSet.set(backend.StorageKey("key11"), backend.StorageValue("val11"), extrinsic(11))

	next, _ := iter.Pull2(changeSet.changesAfter(backend.StorageKey("key0")))

	k, v, _ := next()
	require.Equal(t, k, backend.StorageKey("key1"))
	require.Equal(t, v.Value(), backend.StorageValue("val1"))

	next, _ = iter.Pull2(changeSet.changesAfter(backend.StorageKey("key1")))

	k, v, _ = next()
	require.Equal(t, k, backend.StorageKey("key11"))
	require.Equal(t, v.Value(), backend.StorageValue("val11"))

	next, _ = iter.Pull2(changeSet.changesAfter(backend.StorageKey("key11")))

	k, v, _ = next()
	require.Equal(t, k, backend.StorageKey("key2"))
	require.Equal(t, v.Value(), backend.StorageValue("val2"))

	next, _ = iter.Pull2(changeSet.changesAfter(backend.StorageKey("key2")))

	k, v, _ = next()
	require.Equal(t, k, backend.StorageKey("key3"))
	require.Equal(t, v.Value(), backend.StorageValue("val3"))

	next, _ = iter.Pull2(changeSet.changesAfter(backend.StorageKey("key3")))

	k, v, _ = next()
	require.Equal(t, k, backend.StorageKey("key4"))
	require.Equal(t, v.Value(), backend.StorageValue("val4"))

	_, _, has := next()
	require.False(t, has)

	changeSet.rollbackTransaction()

	next, _ = iter.Pull2(changeSet.changesAfter(backend.StorageKey("key0")))

	k, v, _ = next()
	require.Equal(t, k, backend.StorageKey("key1"))
	require.Equal(t, v.Value(), backend.StorageValue("val1"))

	next, _ = iter.Pull2(changeSet.changesAfter(backend.StorageKey("key1")))

	k, v, _ = next()
	require.Equal(t, k, backend.StorageKey("key2"))
	require.Equal(t, v.Value(), backend.StorageValue("val2"))

	next, _ = iter.Pull2(changeSet.changesAfter(backend.StorageKey("key11")))

	k, v, _ = next()
	require.Equal(t, k, backend.StorageKey("key2"))
	require.Equal(t, v.Value(), backend.StorageValue("val2"))

	next, _ = iter.Pull2(changeSet.changesAfter(backend.StorageKey("key2")))
	_, _, has = next()
	require.False(t, has)

	next, _ = iter.Pull2(changeSet.changesAfter(backend.StorageKey("key3")))
	_, _, has = next()
	require.False(t, has)

	next, _ = iter.Pull2(changeSet.changesAfter(backend.StorageKey("key4")))
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
	require.Panics(t, func() { changeSet.DrainCommited() })
}

func TestRuntimeCannotCloseClientTx(t *testing.T) {
	changeSet := newOverlayedChangeSet()
	changeSet.StartTransaction()
	require.NoError(t, changeSet.EnterRuntime())

	changeSet.StartTransaction()
	require.NoError(t, changeSet.commitTransaction())

	require.Error(t, changeSet.commitTransaction(), errorNoOpenTransaction)
	require.Error(t, changeSet.rollbackTransaction(), errorNoOpenTransaction)
}

func TestExitRuntimeClosesRuntimeTx(t *testing.T) {
	changeSet := newOverlayedChangeSet()
	changeSet.StartTransaction()
	changeSet.set(backend.StorageKey("key0"), backend.StorageValue("val0"), extrinsic(1))

	require.NoError(t, changeSet.EnterRuntime())
	changeSet.StartTransaction()
	changeSet.set(backend.StorageKey("key1"), backend.StorageValue("val1"), extrinsic(2))
	changeSet.exitRuntime()

	changeSet.commitTransaction()
	require.Equal(t, uint(0), changeSet.TransactionDepth())

	assertDrained(t, changeSet, Drained{
		{"key0", backend.StorageValue("val0")},
	})
}

func TestEnterExitRuntimeFailsWhenAlreadyInRequestedMode(t *testing.T) {
	changeSet := newOverlayedChangeSet()

	require.Error(t, changeSet.exitRuntime(), errorNotInRuntime)
	require.NoError(t, changeSet.EnterRuntime())
	require.Error(t, changeSet.EnterRuntime(), errorAlreadyInRuntime)
	require.NoError(t, changeSet.exitRuntime())
	require.Error(t, changeSet.exitRuntime(), errorNotInRuntime)
}

func TestRestoreAppendToParent(t *testing.T) {
	changeSet := newOverlayedChangeSet()
	key := "akey"

	from := 50 // 1 byte len
	to := 100  // 2 byte len
	defaultInit := func() backend.StorageValue { return backend.StorageValue{} }
	for i := 0; i < from; i++ {
		changeSet.appendStorage(backend.StorageKey(key), backend.StorageValue([]byte{byte(i)}), defaultInit, nil)
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
		changeSet.appendStorage(backend.StorageKey(key), backend.StorageValue([]byte{byte(i)}), defaultInit, nil)
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

	defaultInit := func() backend.StorageValue { return backend.StorageValue{} }
	initialData := backend.StorageValue(scale.MustMarshal(bytes.Repeat([]byte{1}, 50)))

	changeSet.set(backend.StorageKey(key), initialData, nil)
	changeSet.StartTransaction()

	// Append until we require 2 bytes for the length prefix.
	for i := 0; i < 50; i++ {
		changeSet.appendStorage(backend.StorageKey(key), backend.StorageValue([]byte{byte(i)}), defaultInit, nil)
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

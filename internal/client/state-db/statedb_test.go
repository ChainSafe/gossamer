// Copyright 2024 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package statedb

import (
	"testing"

	"github.com/ChainSafe/gossamer/internal/primitives/core/hash"
	"github.com/stretchr/testify/assert"
)

type TestDB struct {
	Data map[hash.H256]DBValue
	Meta map[string]DBValue
}

func NewTestDB(inserted []uint64) TestDB {
	data := make(map[hash.H256]DBValue)
	for _, v := range inserted {
		data[hash.NewH256FromLowUint64BigEndian(v)] = DBValue(hash.NewH256FromLowUint64BigEndian(v))
	}
	return TestDB{
		Data: data,
		Meta: make(map[string]DBValue),
	}
}

func (tdb TestDB) GetMeta(key []byte) (*DBValue, error) {
	val, ok := tdb.Meta[string(key)]
	if !ok {
		return nil, nil
	}
	return &val, nil
}

func (tdb *TestDB) Commit(commitSet CommitSet[hash.H256]) {
	for _, insert := range commitSet.Data.Inserted {
		tdb.Data[insert.Hash] = insert.DBValue
	}
	for _, insert := range commitSet.Meta.Inserted {
		tdb.Meta[string(insert.Hash)] = insert.DBValue
	}
	for _, k := range commitSet.Data.Deleted {
		delete(tdb.Data, k)
	}
	for _, k := range commitSet.Meta.Deleted {
		delete(tdb.Meta, string(k))
	}
}

func NewCommit(inserted []uint64, deleted []uint64) CommitSet[hash.H256] {
	return CommitSet[hash.H256]{
		Data: NewChangeset(inserted, deleted),
	}
}

func NewChangeset(inserted []uint64, deleted []uint64) ChangeSet[hash.H256] {
	var insertedHDBVs []HashDBValue[hash.H256]
	for _, v := range inserted {
		insertedHDBVs = append(insertedHDBVs, HashDBValue[hash.H256]{
			Hash:    hash.NewH256FromLowUint64BigEndian(v),
			DBValue: DBValue(hash.NewH256FromLowUint64BigEndian(v)),
		})
	}
	var deletedHashes []hash.H256
	for _, v := range deleted {
		deletedHashes = append(deletedHashes, hash.NewH256FromLowUint64BigEndian(v))
	}
	return ChangeSet[hash.H256]{
		Inserted: insertedHDBVs,
		Deleted:  deletedHashes,
	}
}

func newTestDB(t *testing.T, settings PruningMode) (TestDB, *StateDB[hash.H256, hash.H256]) {
	t.Helper()

	db := NewTestDB([]uint64{91, 921, 922, 93, 94})
	stateDBInit, stateDB, err := NewStateDB[hash.H256, hash.H256](db, settings, true)
	assert.NoError(t, err)
	db.Commit(stateDBInit)

	commit, err := stateDB.InsertBlock(
		hash.NewH256FromLowUint64BigEndian(1),
		1,
		hash.NewH256FromLowUint64BigEndian(0),
		NewChangeset([]uint64{1}, []uint64{91}),
	)
	assert.NoError(t, err)
	db.Commit(commit)
	commit, err = stateDB.InsertBlock(
		hash.NewH256FromLowUint64BigEndian(21),
		2,
		hash.NewH256FromLowUint64BigEndian(1),
		NewChangeset([]uint64{21}, []uint64{921, 1}),
	)
	assert.NoError(t, err)
	db.Commit(commit)
	commit, err = stateDB.InsertBlock(
		hash.NewH256FromLowUint64BigEndian(22),
		2,
		hash.NewH256FromLowUint64BigEndian(1),
		NewChangeset([]uint64{22}, []uint64{922}),
	)
	assert.NoError(t, err)
	db.Commit(commit)
	commit, err = stateDB.InsertBlock(
		hash.NewH256FromLowUint64BigEndian(3),
		3,
		hash.NewH256FromLowUint64BigEndian(21),
		NewChangeset([]uint64{3}, []uint64{93}),
	)
	assert.NoError(t, err)
	db.Commit(commit)
	commit, err = stateDB.CanonicalizeBlock(hash.NewH256FromLowUint64BigEndian(1))
	assert.NoError(t, err)
	db.Commit(commit)
	commit, err = stateDB.InsertBlock(
		hash.NewH256FromLowUint64BigEndian(4),
		4,
		hash.NewH256FromLowUint64BigEndian(3),
		NewChangeset([]uint64{4}, []uint64{94}),
	)
	assert.NoError(t, err)
	db.Commit(commit)
	commit, err = stateDB.CanonicalizeBlock(hash.NewH256FromLowUint64BigEndian(21))
	assert.NoError(t, err)
	db.Commit(commit)
	commit, err = stateDB.CanonicalizeBlock(hash.NewH256FromLowUint64BigEndian(3))
	assert.NoError(t, err)
	db.Commit(commit)

	return db, stateDB
}

func TestStateDB_FullArchiveKeepsEverything(t *testing.T) {
	db, stateDB := newTestDB(t, PruningModeArchiveAll{})
	assert.Equal(t, NewTestDB([]uint64{1, 21, 22, 3, 4, 91, 921, 922, 93, 94}).Data, db.Data)
	assert.Equal(t, IsPrunedNotPruned, stateDB.IsPruned(hash.NewH256FromLowUint64BigEndian(0), 0))
}

func TestStateDB_CanonicalArchiveKeepsCanonical(t *testing.T) {
	db, _ := newTestDB(t, PruningModeArchiveCanonical{})
	assert.Equal(t, NewTestDB([]uint64{1, 21, 3, 91, 921, 922, 93, 94}).Data, db.Data)
}

func TestStateDB_BlockRecordUnavailable(t *testing.T) {
	t.Skipf("this test is for the DB backed pruning where we do not count references")
	maxBlocks := uint32(1)
	db, stateDB := newTestDB(t, PruningModeConstrained{MaxBlocks: &maxBlocks})
	// import 2 blocks
	for _, i := range []uint64{5, 6} {
		commit, err := stateDB.InsertBlock(
			hash.NewH256FromLowUint64BigEndian(i), i,
			hash.NewH256FromLowUint64BigEndian(i-1),
			NewChangeset(nil, nil))
		assert.NoError(t, err)
		db.Commit(commit)
	}
	// canonicalize block 4 but not commit it to db
	c1, err := stateDB.CanonicalizeBlock(hash.NewH256FromLowUint64BigEndian(4))
	assert.NoError(t, err)
	assert.Equal(t, IsPrunedPruned, stateDB.IsPruned(hash.NewH256FromLowUint64BigEndian(3), 3))

	// canonicalize block 5 but not commit it to db, block 4 is not pruned due to it is not
	// commit to db yet (unavailable), return `MaybePruned` here because `apply_pending` is not
	// called and block 3 is still in cache
	c2, err := stateDB.CanonicalizeBlock(hash.NewH256FromLowUint64BigEndian(5))
	assert.NoError(t, err)
	assert.Equal(t, IsPrunedMaybePruned, stateDB.IsPruned(hash.NewH256FromLowUint64BigEndian(4), 4))
	// commit block 4 and 5 to db, and import a new block will prune both block 4 and 5
	db.Commit(c1)
	db.Commit(c2)
	commit, err := stateDB.CanonicalizeBlock(hash.NewH256FromLowUint64BigEndian(6))
	assert.NoError(t, err)
	db.Commit(commit)
	assert.Equal(t, IsPrunedPruned, stateDB.IsPruned(hash.NewH256FromLowUint64BigEndian(4), 4))
	assert.Equal(t, IsPrunedPruned, stateDB.IsPruned(hash.NewH256FromLowUint64BigEndian(4), 4))
}

func TestStateDB_PruneWindow0(t *testing.T) {
	var maxBlocks uint32
	db, _ := newTestDB(t, PruningModeConstrained{MaxBlocks: &maxBlocks})
	assert.Equal(t, NewTestDB([]uint64{21, 3, 922, 94}).Data, db.Data)
}

func TestStateDB_PruneWindow1(t *testing.T) {
	var maxBlocks uint32 = 1
	db, stateDB := newTestDB(t, PruningModeConstrained{MaxBlocks: &maxBlocks})
	assert.Equal(t, IsPrunedPruned, stateDB.IsPruned(hash.NewH256FromLowUint64BigEndian(0), 0))
	assert.Equal(t, IsPrunedPruned, stateDB.IsPruned(hash.NewH256FromLowUint64BigEndian(1), 1))
	assert.Equal(t, IsPrunedPruned, stateDB.IsPruned(hash.NewH256FromLowUint64BigEndian(21), 2))
	assert.Equal(t, IsPrunedPruned, stateDB.IsPruned(hash.NewH256FromLowUint64BigEndian(22), 2))
	assert.Equal(t, NewTestDB([]uint64{21, 3, 922, 93, 94}).Data, db.Data)
}

func TestStateDB_PruneWindow2(t *testing.T) {
	var maxBlocks uint32 = 2
	db, stateDB := newTestDB(t, PruningModeConstrained{MaxBlocks: &maxBlocks})
	assert.Equal(t, IsPrunedPruned, stateDB.IsPruned(hash.NewH256FromLowUint64BigEndian(0), 0))
	assert.Equal(t, IsPrunedPruned, stateDB.IsPruned(hash.NewH256FromLowUint64BigEndian(1), 1))
	assert.Equal(t, IsPrunedNotPruned, stateDB.IsPruned(hash.NewH256FromLowUint64BigEndian(21), 2))
	assert.Equal(t, IsPrunedPruned, stateDB.IsPruned(hash.NewH256FromLowUint64BigEndian(22), 2))
	assert.Equal(t, NewTestDB([]uint64{1, 21, 3, 921, 922, 93, 94}).Data, db.Data)
}

func TestStateDB_DetectsIncompatibleMode(t *testing.T) {
	db := NewTestDB(nil)
	init, stateDB, err := NewStateDB[hash.H256, hash.H256](db, PruningModeArchiveAll{}, true)
	assert.NoError(t, err)
	db.Commit(init)
	commit, err := stateDB.InsertBlock(
		hash.NewH256FromLowUint64BigEndian(0), 0, hash.NewH256FromLowUint64BigEndian(0), NewChangeset(nil, nil),
	)
	assert.NoError(t, err)
	db.Commit(commit)
	var maxBlocks uint32 = 2
	newMode := PruningModeConstrained{MaxBlocks: &maxBlocks}

	_, _, err = NewStateDB[hash.H256, hash.H256](db, newMode, false)
	assert.ErrorIs(t, err, ErrIncompatiblePruningModes)
}

func checkStoredAndRequestedModeCompatibility(
	t *testing.T,
	created PruningMode, reopened PruningMode, expectedMode PruningMode, expectedErr error) {
	db := NewTestDB(nil)
	init, _, err := NewStateDB[hash.H256, hash.H256](db, created, true)
	assert.NoError(t, err)
	db.Commit(init)

	init, stateDB, err := NewStateDB[hash.H256, hash.H256](db, reopened, false)

	if expectedErr == nil {
		assert.NoError(t, err)
		db.Commit(init)
		assert.Equal(t, expectedMode, stateDB.PruningMode())
	} else {
		assert.Error(t, err)
		assert.ErrorIs(t, err, expectedErr)
	}
}

func maxBlocks(m uint32) *uint32 {
	return &m
}

func TestStateDB_PruningModeCompatibility(t *testing.T) {
	for _, test := range []struct {
		created      PruningMode
		reopened     PruningMode
		expectedMode PruningMode
		expectedErr  error
	}{
		{nil, nil, PruningModeConstrained{maxBlocks(256)}, nil},
		{nil, PruningModeConstrained{maxBlocks(256)}, PruningModeConstrained{maxBlocks(256)}, nil},
		{nil, PruningModeConstrained{maxBlocks(128)}, PruningModeConstrained{maxBlocks(128)}, nil},
		{nil, PruningModeConstrained{maxBlocks(512)}, PruningModeConstrained{maxBlocks(512)}, nil},
		{nil, PruningModeArchiveAll{}, nil, ErrIncompatiblePruningModes},
		{nil, PruningModeArchiveCanonical{}, nil, ErrIncompatiblePruningModes},
		{PruningModeConstrained{maxBlocks(256)}, nil, PruningModeConstrained{maxBlocks(256)}, nil},
		{PruningModeConstrained{maxBlocks(256)}, PruningModeConstrained{maxBlocks(256)},
			PruningModeConstrained{maxBlocks(256)}, nil},
		{PruningModeConstrained{maxBlocks(256)}, PruningModeConstrained{maxBlocks(128)},
			PruningModeConstrained{maxBlocks(128)}, nil},
		{PruningModeConstrained{maxBlocks(256)}, PruningModeConstrained{maxBlocks(512)},
			PruningModeConstrained{maxBlocks(512)}, nil},
		{PruningModeConstrained{maxBlocks(256)}, PruningModeArchiveAll{}, nil, ErrIncompatiblePruningModes},
		{PruningModeConstrained{maxBlocks(256)}, PruningModeArchiveCanonical{}, nil, ErrIncompatiblePruningModes},
		{PruningModeArchiveAll{}, nil, PruningModeArchiveAll{}, nil},
		{PruningModeArchiveAll{}, PruningModeConstrained{maxBlocks(256)}, nil, ErrIncompatiblePruningModes},
		{PruningModeArchiveAll{}, PruningModeConstrained{maxBlocks(128)}, nil, ErrIncompatiblePruningModes},
		{PruningModeArchiveAll{}, PruningModeConstrained{maxBlocks(512)}, nil, ErrIncompatiblePruningModes},
		{PruningModeArchiveAll{}, PruningModeArchiveAll{}, PruningModeArchiveAll{}, nil},
		{PruningModeArchiveAll{}, PruningModeArchiveCanonical{}, nil, ErrIncompatiblePruningModes},
		{PruningModeArchiveCanonical{}, nil, PruningModeArchiveCanonical{}, nil},
		{PruningModeArchiveCanonical{}, PruningModeConstrained{maxBlocks(256)}, nil, ErrIncompatiblePruningModes},
		{PruningModeArchiveCanonical{}, PruningModeConstrained{maxBlocks(128)}, nil, ErrIncompatiblePruningModes},
		{PruningModeArchiveCanonical{}, PruningModeConstrained{maxBlocks(512)}, nil, ErrIncompatiblePruningModes},
		{PruningModeArchiveCanonical{}, PruningModeArchiveAll{}, nil, ErrIncompatiblePruningModes},
		{PruningModeArchiveCanonical{}, PruningModeArchiveCanonical{}, PruningModeArchiveCanonical{}, nil},
	} {
		t.Run("", func(t *testing.T) {
			checkStoredAndRequestedModeCompatibility(t, test.created, test.reopened, test.expectedMode, test.expectedErr)
		})
	}
}

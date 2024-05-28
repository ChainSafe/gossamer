// Copyright 2024 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package statedb

import (
	"testing"

	"github.com/ChainSafe/gossamer/internal/primitives/core/hash"
	"github.com/stretchr/testify/assert"
)

func TestSplitFirst(t *testing.T) {
	slice := []int{1, 2, 3, 4, 5, 6, 7, 8}

	first, remainder, err := splitFirst[int](&slice)
	assert.NoError(t, err)
	assert.Equal(t, 1, *first)
	assert.Equal(t, []int{2, 3, 4, 5, 6, 7, 8}, *remainder)

	*first = 10
	assert.Equal(t, []int{10, 2, 3, 4, 5, 6, 7, 8}, slice)

	first, remainder, err = splitFirst(remainder)
	assert.NoError(t, err)
	assert.Equal(t, 2, *first)
	assert.Equal(t, []int{3, 4, 5, 6, 7, 8}, *remainder)

	*first = 20
	assert.Equal(t, []int{10, 20, 3, 4, 5, 6, 7, 8}, slice)
}

func TestCreatedFromEmptyDB(t *testing.T) {
	db := TestDB{}
	overlay, err := newNonCanonicalOverlay[hash.H256, hash.H256](db)
	assert.NoError(t, err)
	assert.True(t, overlay.levels.Len() == 0)
	assert.True(t, len(overlay.parents) == 0)
}

func TestCanonicalizeEmptyError(t *testing.T) {
	db := TestDB{}
	overlay, err := newNonCanonicalOverlay[hash.H256, hash.H256](db)
	assert.NoError(t, err)
	commit := CommitSet[hash.H256]{}
	_, err = overlay.Canonicalize(hash.H256(make([]byte, 32)), &commit)
	assert.ErrorIs(t, err, ErrInvalidBlock)
}

func TestInsertAheadError(t *testing.T) {
	h1 := hash.NewRandomH256()
	h2 := hash.NewRandomH256()
	db := TestDB{}
	overlay, err := newNonCanonicalOverlay[hash.H256, hash.H256](db)
	assert.NoError(t, err)
	_, err = overlay.Insert(h1, 2, hash.H256(make([]byte, 32)), ChangeSet[hash.H256]{})
	assert.NoError(t, err)
	_, err = overlay.Insert(h2, 1, h1, ChangeSet[hash.H256]{})
	assert.ErrorIs(t, err, ErrInvalidBlockNumber)
}

func TestInsertBehindError(t *testing.T) {
	h1 := hash.NewRandomH256()
	h2 := hash.NewRandomH256()
	db := TestDB{}
	overlay, err := newNonCanonicalOverlay[hash.H256, hash.H256](db)
	assert.NoError(t, err)
	_, err = overlay.Insert(h1, 1, hash.H256(make([]byte, 32)), ChangeSet[hash.H256]{})
	assert.NoError(t, err)
	_, err = overlay.Insert(h2, 3, h1, ChangeSet[hash.H256]{})
	assert.ErrorIs(t, err, ErrInvalidBlockNumber)
}

func TestInsertUnknownParentError(t *testing.T) {
	h1 := hash.NewRandomH256()
	h2 := hash.NewRandomH256()
	db := TestDB{}
	overlay, err := newNonCanonicalOverlay[hash.H256, hash.H256](db)
	assert.NoError(t, err)
	_, err = overlay.Insert(h1, 1, hash.H256(make([]byte, 32)), ChangeSet[hash.H256]{})
	assert.NoError(t, err)
	_, err = overlay.Insert(h2, 2, hash.H256(make([]byte, 32)), ChangeSet[hash.H256]{})
	assert.ErrorIs(t, err, ErrInvalidParent)
}

func TestInsertExistingError(t *testing.T) {
	h1 := hash.NewRandomH256()
	db := TestDB{}
	overlay, err := newNonCanonicalOverlay[hash.H256, hash.H256](db)
	assert.NoError(t, err)
	_, err = overlay.Insert(h1, 2, hash.H256(make([]byte, 32)), ChangeSet[hash.H256]{})
	assert.NoError(t, err)
	_, err = overlay.Insert(h1, 2, hash.H256(make([]byte, 32)), ChangeSet[hash.H256]{})
	assert.ErrorIs(t, err, ErrBlockAlreadyExists)
}

func TestCanonicalizeUnknownError(t *testing.T) {
	h1 := hash.NewRandomH256()
	h2 := hash.NewRandomH256()
	db := TestDB{}
	overlay, err := newNonCanonicalOverlay[hash.H256, hash.H256](db)
	assert.NoError(t, err)
	_, err = overlay.Insert(h1, 1, hash.H256(make([]byte, 32)), ChangeSet[hash.H256]{})
	assert.NoError(t, err)
	commit := CommitSet[hash.H256]{}
	_, err = overlay.Canonicalize(h2, &commit)
	assert.ErrorIs(t, err, ErrInvalidBlock)
}

func TestInsertCanonicalizeOne(t *testing.T) {
	h1 := hash.NewRandomH256()
	db := NewTestDB([]uint64{1, 2})
	overlay, err := newNonCanonicalOverlay[hash.H256, hash.H256](db)
	assert.NoError(t, err)
	changeset := NewChangeset([]uint64{3, 4}, []uint64{2})
	insertion, err := overlay.Insert(h1, 1, hash.H256(make([]byte, 32)), changeset)
	assert.NoError(t, err)
	assert.Equal(t, 0, len(insertion.Data.Inserted))
	assert.Equal(t, 0, len(insertion.Data.Deleted))
	assert.Equal(t, 2, len(insertion.Meta.Inserted))
	assert.Equal(t, 0, len(insertion.Data.Deleted))
	db.Commit(insertion)
	finalisation := CommitSet[hash.H256]{}
	_, err = overlay.Canonicalize(h1, &finalisation)
	assert.NoError(t, err)
	assert.Equal(t, len(changeset.Inserted), len(finalisation.Data.Inserted))
	assert.Equal(t, len(changeset.Deleted), len(finalisation.Data.Deleted))
	assert.Equal(t, 1, len(finalisation.Meta.Inserted))
	assert.Equal(t, 1, len(finalisation.Meta.Deleted))
	db.Commit(finalisation)
	assert.Equal(t, NewTestDB([]uint64{1, 3, 4}).Data, db.Data)
}
func TestRestoreFromJournal(t *testing.T) {
	h1 := hash.NewRandomH256()
	h2 := hash.NewRandomH256()
	db := NewTestDB([]uint64{1, 2})
	overlay, err := newNonCanonicalOverlay[hash.H256, hash.H256](db)
	assert.NoError(t, err)
	insertion, err := overlay.Insert(h1, 10, hash.H256(make([]byte, 32)), NewChangeset([]uint64{3, 4}, []uint64{2}))
	assert.NoError(t, err)
	db.Commit(insertion)
	insertion, err = overlay.Insert(h2, 11, h1, NewChangeset([]uint64{5}, []uint64{3}))
	assert.NoError(t, err)
	db.Commit(insertion)
	assert.Equal(t, 3, len(db.Meta))

	overlay2, err := newNonCanonicalOverlay[hash.H256, hash.H256](db)
	assert.NoError(t, err)
	assert.Equal(t, overlay.levels.Len(), overlay2.levels.Len())
	actual := make([]overlayLevel[hash.H256, hash.H256], overlay.levels.Len())
	expected := make([]overlayLevel[hash.H256, hash.H256], overlay.levels.Len())
	for i := 0; i < overlay.levels.Len(); i++ {
		actual[i] = overlay2.levels.At(i)
		expected[i] = overlay.levels.At(i)
	}
	assert.Equal(t, expected, actual)
}

func TestRestoreFromJournalAfterCanonicalize(t *testing.T) {
	h1 := hash.NewRandomH256()
	h2 := hash.NewRandomH256()
	db := NewTestDB([]uint64{1, 2})
	overlay, err := newNonCanonicalOverlay[hash.H256, hash.H256](db)
	assert.NoError(t, err)
	insertion, err := overlay.Insert(h1, 10, hash.H256(make([]byte, 32)), NewChangeset([]uint64{3, 4}, []uint64{2}))
	assert.NoError(t, err)
	db.Commit(insertion)
	insertion, err = overlay.Insert(h2, 11, h1, NewChangeset([]uint64{5}, []uint64{3}))
	assert.NoError(t, err)
	db.Commit(insertion)
	assert.Equal(t, 3, len(db.Meta))

	commit := CommitSet[hash.H256]{}
	_, err = overlay.Canonicalize(h1, &commit)
	assert.NoError(t, err)
	overlay.Unpin(h1)
	db.Commit(commit)
	assert.Equal(t, 1, overlay.levels.Len())

	overlay2, err := newNonCanonicalOverlay[hash.H256, hash.H256](db)
	assert.NoError(t, err)
	assert.Equal(t, overlay.levels.Len(), overlay2.levels.Len())
	actual := make([]overlayLevel[hash.H256, hash.H256], overlay.levels.Len())
	expected := make([]overlayLevel[hash.H256, hash.H256], overlay.levels.Len())
	for i := 0; i < overlay.levels.Len(); i++ {
		actual[i] = overlay2.levels.At(i)
		expected[i] = overlay.levels.At(i)
	}
	assert.Equal(t, expected, actual)
	assert.Equal(t, overlay.parents, overlay2.parents)
	assert.Equal(t, overlay.lastCanonicalized, overlay2.lastCanonicalized)
}

func contains(overlay nonCanonicalOverlay[hash.H256, hash.H256], key uint64) bool {
	val := overlay.Get(hash.NewH256FromLowUint64BigEndian(key))
	return val != nil && string(*val) == string(hash.NewH256FromLowUint64BigEndian(key))
}

func TestInsertCanonicalizeTwo(t *testing.T) {
	h1 := hash.NewRandomH256()
	h2 := hash.NewRandomH256()
	db := NewTestDB([]uint64{1, 2, 3, 4})
	overlay, err := newNonCanonicalOverlay[hash.H256, hash.H256](db)
	assert.NoError(t, err)
	changeset1 := NewChangeset([]uint64{5, 6}, []uint64{2})
	changeset2 := NewChangeset([]uint64{7, 8}, []uint64{5, 3})
	insertion, err := overlay.Insert(h1, 1, hash.H256(make([]byte, 32)), changeset1)
	assert.NoError(t, err)
	db.Commit(insertion)
	assert.True(t, contains(overlay, 5))
	insertion, err = overlay.Insert(h2, 2, h1, changeset2)
	assert.NoError(t, err)
	db.Commit(insertion)
	assert.True(t, contains(overlay, 7))
	assert.True(t, contains(overlay, 5))
	assert.Equal(t, 2, overlay.levels.Len())
	assert.Equal(t, 2, len(overlay.parents))
	commit := CommitSet[hash.H256]{}
	_, err = overlay.Canonicalize(h1, &commit)
	assert.NoError(t, err)
	db.Commit(commit)
	overlay.Sync()
	assert.False(t, contains(overlay, 5))
	assert.True(t, contains(overlay, 7))
	assert.Equal(t, 1, overlay.levels.Len())
	assert.Equal(t, 1, len(overlay.parents))
	commit = CommitSet[hash.H256]{}
	overlay.Canonicalize(h2, &commit)
	db.Commit(commit)
	overlay.Sync()
	assert.Equal(t, 0, overlay.levels.Len())
	assert.Equal(t, 0, len(overlay.parents))
	assert.Equal(t, NewTestDB([]uint64{1, 4, 6, 7, 8}).Data, db.Data)
}

func TestInsertSameKey(t *testing.T) {
	db := NewTestDB([]uint64{})
	h1 := hash.NewRandomH256()
	c1 := NewChangeset([]uint64{1}, []uint64{})
	h2 := hash.NewRandomH256()
	c2 := NewChangeset([]uint64{1}, []uint64{})

	overlay, err := newNonCanonicalOverlay[hash.H256, hash.H256](db)
	assert.NoError(t, err)
	insertion, err := overlay.Insert(h1, 1, hash.H256(make([]byte, 32)), c1)
	assert.NoError(t, err)
	db.Commit(insertion)
	insertion, err = overlay.Insert(h2, 1, hash.H256(make([]byte, 32)), c2)
	assert.NoError(t, err)
	db.Commit(insertion)
	assert.True(t, contains(overlay, 1))
	commit := CommitSet[hash.H256]{}
	_, err = overlay.Canonicalize(h1, &commit)
	assert.NoError(t, err)
	db.Commit(commit)
	overlay.Sync()
	assert.False(t, contains(overlay, 1))
}

func TestInsertAndCanonicalize(t *testing.T) {
	h1 := hash.NewRandomH256()
	h2 := hash.NewRandomH256()
	h3 := hash.NewRandomH256()
	db := NewTestDB([]uint64{})
	overlay, err := newNonCanonicalOverlay[hash.H256, hash.H256](db)
	assert.NoError(t, err)
	changeset := NewChangeset([]uint64{}, []uint64{})
	insertion, err := overlay.Insert(h1, 1, hash.H256(make([]byte, 32)), changeset)
	assert.NoError(t, err)
	db.Commit(insertion)
	insertion, err = overlay.Insert(h2, 2, h1, changeset)
	assert.NoError(t, err)
	db.Commit(insertion)
	commit := CommitSet[hash.H256]{}
	_, err = overlay.Canonicalize(h1, &commit)
	assert.NoError(t, err)
	_, err = overlay.Canonicalize(h2, &commit)
	assert.NoError(t, err)
	db.Commit(commit)
	insertion, err = overlay.Insert(h3, 3, h2, changeset)
	assert.NoError(t, err)
	db.Commit(insertion)
	assert.Equal(t, 1, overlay.levels.Len())
}

func TestComplexTree(t *testing.T) {
	db := NewTestDB([]uint64{})

	// - 1 - 1_1 - 1_1_1
	//     \ 1_2 - 1_2_1
	//           \ 1_2_2
	//           \ 1_2_3
	//
	// - 2 - 2_1 - 2_1_1
	//     \ 2_2
	//
	// 1_2_2 is the winner

	h1 := hash.NewRandomH256()
	c1 := NewChangeset([]uint64{1}, []uint64{})
	h2 := hash.NewRandomH256()
	c2 := NewChangeset([]uint64{2}, []uint64{})

	h11 := hash.NewRandomH256()
	c11 := NewChangeset([]uint64{11}, []uint64{})
	h12 := hash.NewRandomH256()
	c12 := NewChangeset([]uint64{12}, []uint64{})
	h21 := hash.NewRandomH256()
	c21 := NewChangeset([]uint64{21}, []uint64{})
	h22 := hash.NewRandomH256()
	c22 := NewChangeset([]uint64{22}, []uint64{})

	h111 := hash.NewRandomH256()
	c111 := NewChangeset([]uint64{111}, []uint64{})
	h121 := hash.NewRandomH256()
	c121 := NewChangeset([]uint64{121}, []uint64{})
	h122 := hash.NewRandomH256()
	c122 := NewChangeset([]uint64{122}, []uint64{})
	h123 := hash.NewRandomH256()
	c123 := NewChangeset([]uint64{123}, []uint64{})
	h211 := hash.NewRandomH256()
	c211 := NewChangeset([]uint64{211}, []uint64{})

	overlay, err := newNonCanonicalOverlay[hash.H256, hash.H256](db)
	assert.NoError(t, err)
	insertion, err := overlay.Insert(h1, 1, hash.H256(make([]byte, 32)), c1)
	assert.NoError(t, err)
	db.Commit(insertion)

	insertion, err = overlay.Insert(h11, 2, h1, c11)
	assert.NoError(t, err)
	db.Commit(insertion)
	insertion, err = overlay.Insert(h12, 2, h1, c12)
	assert.NoError(t, err)
	db.Commit(insertion)

	insertion, err = overlay.Insert(h2, 1, hash.H256(make([]byte, 32)), c2)
	assert.NoError(t, err)
	db.Commit(insertion)

	insertion, err = overlay.Insert(h21, 2, h2, c21)
	assert.NoError(t, err)
	db.Commit(insertion)
	insertion, err = overlay.Insert(h22, 2, h2, c22)
	assert.NoError(t, err)
	db.Commit(insertion)

	insertion, err = overlay.Insert(h111, 3, h11, c111)
	assert.NoError(t, err)
	db.Commit(insertion)
	insertion, err = overlay.Insert(h121, 3, h12, c121)
	assert.NoError(t, err)
	db.Commit(insertion)
	insertion, err = overlay.Insert(h122, 3, h12, c122)
	assert.NoError(t, err)
	db.Commit(insertion)
	insertion, err = overlay.Insert(h123, 3, h12, c123)
	assert.NoError(t, err)
	db.Commit(insertion)
	insertion, err = overlay.Insert(h211, 3, h21, c211)
	assert.NoError(t, err)
	db.Commit(insertion)

	assert.True(t, contains(overlay, 2))
	assert.True(t, contains(overlay, 11))
	assert.True(t, contains(overlay, 21))
	assert.True(t, contains(overlay, 111))
	assert.True(t, contains(overlay, 122))
	assert.True(t, contains(overlay, 211))
	assert.Equal(t, 3, overlay.levels.Len())
	assert.Equal(t, 11, len(overlay.parents))
	assert.Equal(t, overlay.lastCanonicalized, &hashBlock[hash.H256]{hash.H256(make([]byte, 32)), 0})

	// check if restoration from journal results in the same tree
	overlay2, err := newNonCanonicalOverlay[hash.H256, hash.H256](db)
	assert.NoError(t, err)
	assert.Equal(t, overlay.levels.Len(), overlay2.levels.Len())
	actual := make([]overlayLevel[hash.H256, hash.H256], overlay.levels.Len())
	expected := make([]overlayLevel[hash.H256, hash.H256], overlay.levels.Len())
	for i := 0; i < overlay.levels.Len(); i++ {
		actual[i] = overlay2.levels.At(i)
		expected[i] = overlay.levels.At(i)
	}
	assert.Equal(t, expected, actual)
	assert.Equal(t, overlay.parents, overlay2.parents)
	assert.Equal(t, overlay.lastCanonicalized, overlay2.lastCanonicalized)

	// canonicalize 1. 2 and all its children should be discarded
	commit := CommitSet[hash.H256]{}
	_, err = overlay.Canonicalize(h1, &commit)
	assert.NoError(t, err)
	db.Commit(commit)
	overlay.Sync()
	assert.Equal(t, 2, overlay.levels.Len())
	assert.Equal(t, 6, len(overlay.parents))
	assert.False(t, contains(overlay, 1))
	assert.False(t, contains(overlay, 2))
	assert.False(t, contains(overlay, 21))
	assert.False(t, contains(overlay, 22))
	assert.False(t, contains(overlay, 211))
	assert.True(t, contains(overlay, 111))
	// check that journals are deleted
	val, err := db.GetMeta(toJournalKey(1, 0))
	assert.NoError(t, err)
	assert.Nil(t, val)
	val, err = db.GetMeta(toJournalKey(1, 1))
	assert.NoError(t, err)
	assert.Nil(t, val)
	val, err = db.GetMeta(toJournalKey(2, 1))
	assert.NoError(t, err)
	assert.NotNil(t, val)
	val, err = db.GetMeta(toJournalKey(2, 2))
	assert.NoError(t, err)
	assert.Nil(t, val)
	val, err = db.GetMeta(toJournalKey(2, 3))
	assert.NoError(t, err)
	assert.Nil(t, val)

	// canonicalize 1_2. 1_1 and all its children should be discarded
	commit = CommitSet[hash.H256]{}
	_, err = overlay.Canonicalize(h12, &commit)
	assert.NoError(t, err)
	db.Commit(commit)
	overlay.Sync()
	assert.Equal(t, 1, overlay.levels.Len())
	assert.Equal(t, 3, len(overlay.parents))
	assert.False(t, contains(overlay, 11))
	assert.False(t, contains(overlay, 111))
	assert.True(t, contains(overlay, 121))
	assert.True(t, contains(overlay, 122))
	assert.True(t, contains(overlay, 123))
	assert.True(t, overlay.HaveBlock(h121))
	assert.False(t, overlay.HaveBlock(h12))
	assert.False(t, overlay.HaveBlock(h11))
	assert.False(t, overlay.HaveBlock(h111))

	// canonicalize 1_2_2
	commit = CommitSet[hash.H256]{}
	_, err = overlay.Canonicalize(h122, &commit)
	assert.NoError(t, err)
	db.Commit(commit)
	overlay.Sync()
	assert.Equal(t, 0, overlay.levels.Len())
	assert.Equal(t, 0, len(overlay.parents))
	assert.Equal(t, NewTestDB([]uint64{1, 12, 122}).Data, db.Data)
	assert.Equal(t, overlay.lastCanonicalized, &hashBlock[hash.H256]{h122, 3})
}

func TestInsertRevert(t *testing.T) {
	h1 := hash.NewRandomH256()
	h2 := hash.NewRandomH256()
	db := NewTestDB([]uint64{1, 2, 3, 4})
	overlay, err := newNonCanonicalOverlay[hash.H256, hash.H256](db)
	assert.NoError(t, err)
	assert.Nil(t, overlay.RevertOne())
	changeset1 := NewChangeset([]uint64{5, 6}, []uint64{2})
	changeset2 := NewChangeset([]uint64{7, 8}, []uint64{5, 3})
	insertion, err := overlay.Insert(h1, 1, hash.H256(make([]byte, 32)), changeset1)
	assert.NoError(t, err)
	db.Commit(insertion)
	assert.True(t, contains(overlay, 5))
	insertion, err = overlay.Insert(h2, 2, h1, changeset2)
	assert.NoError(t, err)
	db.Commit(insertion)
	assert.True(t, contains(overlay, 7))
	revert := overlay.RevertOne()
	assert.NotNil(t, revert)
	db.Commit(*revert)
	assert.Equal(t, 1, len(overlay.parents))
	assert.True(t, contains(overlay, 5))
	assert.False(t, contains(overlay, 7))
	revert = overlay.RevertOne()
	assert.NotNil(t, revert)
	db.Commit(*revert)
	assert.Equal(t, 0, overlay.levels.Len())
	assert.Equal(t, 0, len(overlay.parents))
	assert.Nil(t, overlay.RevertOne())
}

func TestKeepsPinned(t *testing.T) {
	db := NewTestDB([]uint64{})

	// - 0 - 1_1
	//     \ 1_2

	h1 := hash.NewRandomH256()
	c1 := NewChangeset([]uint64{1}, []uint64{})
	h2 := hash.NewRandomH256()
	c2 := NewChangeset([]uint64{2}, []uint64{})

	overlay, err := newNonCanonicalOverlay[hash.H256, hash.H256](db)
	assert.NoError(t, err)
	insertion, err := overlay.Insert(h1, 1, hash.H256(make([]byte, 32)), c1)
	assert.NoError(t, err)
	db.Commit(insertion)
	insertion, err = overlay.Insert(h2, 1, hash.H256(make([]byte, 32)), c2)
	assert.NoError(t, err)
	db.Commit(insertion)

	overlay.Pin(h1)

	commit := CommitSet[hash.H256]{}
	_, err = overlay.Canonicalize(h2, &commit)
	assert.NoError(t, err)
	db.Commit(commit)
	assert.True(t, contains(overlay, 1))
	overlay.Unpin(h1)
	assert.False(t, contains(overlay, 1))
}

func TestKeepsPinnedRefCount(t *testing.T) {
	db := NewTestDB([]uint64{})

	// - 0 - 1_1
	//     \ 1_2
	//     \ 1_3

	// 1_1 and 1_2 both make the same change
	h1 := hash.NewRandomH256()
	c1 := NewChangeset([]uint64{1}, []uint64{})
	h2 := hash.NewRandomH256()
	c2 := NewChangeset([]uint64{1}, []uint64{})
	h3 := hash.NewRandomH256()
	c3 := NewChangeset([]uint64{}, []uint64{})

	overlay, err := newNonCanonicalOverlay[hash.H256, hash.H256](db)
	assert.NoError(t, err)
	insertion, err := overlay.Insert(h1, 1, hash.H256(make([]byte, 32)), c1)
	assert.NoError(t, err)
	db.Commit(insertion)
	insertion, err = overlay.Insert(h2, 1, hash.H256(make([]byte, 32)), c2)
	assert.NoError(t, err)
	db.Commit(insertion)
	insertion, err = overlay.Insert(h3, 1, hash.H256(make([]byte, 32)), c3)
	assert.NoError(t, err)
	db.Commit(insertion)

	overlay.Pin(h1)

	commit := CommitSet[hash.H256]{}
	_, err = overlay.Canonicalize(h3, &commit)
	assert.NoError(t, err)
	db.Commit(commit)

	assert.True(t, contains(overlay, 1))
	overlay.Unpin(h1)
	assert.False(t, contains(overlay, 1))
}

func TestPinsCanonicalized(t *testing.T) {
	db := NewTestDB([]uint64{})

	h1 := hash.NewRandomH256()
	c1 := NewChangeset([]uint64{1}, []uint64{})
	h2 := hash.NewRandomH256()
	c2 := NewChangeset([]uint64{2}, []uint64{})

	overlay, err := newNonCanonicalOverlay[hash.H256, hash.H256](db)
	assert.NoError(t, err)
	insertion, err := overlay.Insert(h1, 1, hash.H256(make([]byte, 32)), c1)
	assert.NoError(t, err)
	db.Commit(insertion)
	insertion, err = overlay.Insert(h2, 2, h1, c2)
	assert.NoError(t, err)
	db.Commit(insertion)

	commit := CommitSet[hash.H256]{}
	_, err = overlay.Canonicalize(h1, &commit)
	assert.NoError(t, err)
	_, err = overlay.Canonicalize(h2, &commit)
	assert.NoError(t, err)
	assert.True(t, contains(overlay, 1))
	assert.True(t, contains(overlay, 2))
	db.Commit(commit)
	overlay.Sync()
	assert.False(t, contains(overlay, 1))
	assert.False(t, contains(overlay, 2))
}

func TestPinKeepsParent(t *testing.T) {
	db := NewTestDB([]uint64{})

	// - 0 - 1_1 - 2_1
	//     \ 1_2

	h11 := hash.NewRandomH256()
	c11 := NewChangeset([]uint64{1}, []uint64{})
	h12 := hash.NewRandomH256()
	c12 := NewChangeset([]uint64{}, []uint64{})
	h21 := hash.NewRandomH256()
	c21 := NewChangeset([]uint64{}, []uint64{})

	overlay, err := newNonCanonicalOverlay[hash.H256, hash.H256](db)
	assert.NoError(t, err)
	insertion, err := overlay.Insert(h11, 1, hash.H256(make([]byte, 32)), c11)
	assert.NoError(t, err)
	db.Commit(insertion)
	insertion, err = overlay.Insert(h12, 1, hash.H256(make([]byte, 32)), c12)
	assert.NoError(t, err)
	db.Commit(insertion)
	insertion, err = overlay.Insert(h21, 2, h11, c21)
	assert.NoError(t, err)
	db.Commit(insertion)

	overlay.Pin(h21)

	commit := CommitSet[hash.H256]{}
	_, err = overlay.Canonicalize(h12, &commit)
	assert.NoError(t, err)
	db.Commit(commit)

	assert.True(t, contains(overlay, 1))
	overlay.Unpin(h21)
	assert.False(t, contains(overlay, 1))
	overlay.Unpin(h12)
	assert.Equal(t, 0, len(overlay.pinned))
}

func TestRestoreFromJournalAfterCanonicalizeNoFirst(t *testing.T) {
	// This test discards a branch that is journaled under a non-zero index on level 1,
	// making sure all journals are loaded for each level even if some of them are missing.
	root := hash.NewRandomH256()
	h1 := hash.NewRandomH256()
	h2 := hash.NewRandomH256()
	h11 := hash.NewRandomH256()
	h21 := hash.NewRandomH256()
	db := NewTestDB([]uint64{})
	overlay, err := newNonCanonicalOverlay[hash.H256, hash.H256](db)
	assert.NoError(t, err)
	insertion, err := overlay.Insert(root, 10, hash.H256(make([]byte, 32)), NewChangeset([]uint64{}, []uint64{}))
	assert.NoError(t, err)
	db.Commit(insertion)
	insertion, err = overlay.Insert(h1, 11, root, NewChangeset([]uint64{1}, []uint64{}))
	assert.NoError(t, err)
	db.Commit(insertion)
	insertion, err = overlay.Insert(h2, 11, root, NewChangeset([]uint64{2}, []uint64{}))
	assert.NoError(t, err)
	db.Commit(insertion)
	insertion, err = overlay.Insert(h11, 12, h1, NewChangeset([]uint64{11}, []uint64{}))
	assert.NoError(t, err)
	db.Commit(insertion)
	insertion, err = overlay.Insert(h21, 12, h2, NewChangeset([]uint64{21}, []uint64{}))
	assert.NoError(t, err)
	db.Commit(insertion)
	commit := CommitSet[hash.H256]{}
	_, err = overlay.Canonicalize(root, &commit)
	assert.NoError(t, err)
	_, err = overlay.Canonicalize(h2, &commit)
	assert.NoError(t, err)
	db.Commit(commit)
	assert.Equal(t, 1, overlay.levels.Len())
	assert.True(t, contains(overlay, 21))
	assert.False(t, contains(overlay, 11))
	val, err := db.GetMeta(toJournalKey(12, 1))
	assert.NoError(t, err)
	assert.NotNil(t, val)

	// Restore into a new overlay and check that journaled value exists.
	overlay, err = newNonCanonicalOverlay[hash.H256, hash.H256](db)
	assert.NoError(t, err)
	assert.True(t, contains(overlay, 21))

	commit = CommitSet[hash.H256]{}
	_, err = overlay.Canonicalize(h21, &commit)
	assert.NoError(t, err)
	db.Commit(commit)
	overlay.Sync()
	assert.False(t, contains(overlay, 21))
}

func TestIndexReuse(t *testing.T) {
	// This test discards a branch that is journaled under a non-zero index on level 1,
	// making sure all journals are loaded for each level even if some of them are missing.
	root := hash.NewRandomH256()
	h1 := hash.NewRandomH256()
	h2 := hash.NewRandomH256()
	h11 := hash.NewRandomH256()
	h21 := hash.NewRandomH256()
	db := NewTestDB([]uint64{})
	overlay, err := newNonCanonicalOverlay[hash.H256, hash.H256](db)
	assert.NoError(t, err)
	insertion, err := overlay.Insert(root, 10, hash.H256(make([]byte, 32)), NewChangeset([]uint64{}, []uint64{}))
	assert.NoError(t, err)
	db.Commit(insertion)
	insertion, err = overlay.Insert(h1, 11, root, NewChangeset([]uint64{1}, []uint64{}))
	assert.NoError(t, err)
	db.Commit(insertion)
	insertion, err = overlay.Insert(h2, 11, root, NewChangeset([]uint64{2}, []uint64{}))
	assert.NoError(t, err)
	db.Commit(insertion)
	insertion, err = overlay.Insert(h11, 12, h1, NewChangeset([]uint64{11}, []uint64{}))
	assert.NoError(t, err)
	db.Commit(insertion)
	insertion, err = overlay.Insert(h21, 12, h2, NewChangeset([]uint64{21}, []uint64{}))
	assert.NoError(t, err)
	db.Commit(insertion)
	commit := CommitSet[hash.H256]{}
	_, err = overlay.Canonicalize(root, &commit)
	assert.NoError(t, err)
	_, err = overlay.Canonicalize(h2, &commit)
	assert.NoError(t, err)
	db.Commit(commit)

	// add another block at top level. It should reuse journal index 0 of previously discarded
	// block
	h22 := hash.NewRandomH256()
	insertion, err = overlay.Insert(h22, 12, h2, NewChangeset([]uint64{22}, []uint64{}))
	assert.NoError(t, err)
	db.Commit(insertion)
	assert.Equal(t, uint64(1), overlay.levels.At(0).blocks[0].journalIndex)
	assert.Equal(t, uint64(0), overlay.levels.At(0).blocks[1].journalIndex)
}

func TestRemoveWorks(t *testing.T) {
	root := hash.NewRandomH256()
	h1 := hash.NewRandomH256()
	h2 := hash.NewRandomH256()
	h11 := hash.NewRandomH256()
	h21 := hash.NewRandomH256()
	db := NewTestDB([]uint64{})
	overlay, err := newNonCanonicalOverlay[hash.H256, hash.H256](db)
	assert.NoError(t, err)
	insertion, err := overlay.Insert(root, 10, hash.H256(make([]byte, 32)), NewChangeset([]uint64{}, []uint64{}))
	assert.NoError(t, err)
	db.Commit(insertion)
	insertion, err = overlay.Insert(h1, 11, root, NewChangeset([]uint64{1}, []uint64{}))
	assert.NoError(t, err)
	db.Commit(insertion)
	insertion, err = overlay.Insert(h2, 11, root, NewChangeset([]uint64{2}, []uint64{}))
	assert.NoError(t, err)
	db.Commit(insertion)
	insertion, err = overlay.Insert(h11, 12, h1, NewChangeset([]uint64{11}, []uint64{}))
	assert.NoError(t, err)
	db.Commit(insertion)
	insertion, err = overlay.Insert(h21, 12, h2, NewChangeset([]uint64{21}, []uint64{}))
	assert.NoError(t, err)
	db.Commit(insertion)
	assert.Nil(t, overlay.Remove(h1))
	assert.Nil(t, overlay.Remove(h2))
	assert.Equal(t, 3, overlay.levels.Len())

	commit := overlay.Remove(h11)
	assert.NotNil(t, commit)
	db.Commit(*commit)
	assert.False(t, contains(overlay, 11))

	commit = overlay.Remove(h21)
	assert.NotNil(t, commit)
	db.Commit(*commit)
	assert.Equal(t, 2, overlay.levels.Len())

	commit = overlay.Remove(h2)
	assert.NotNil(t, commit)
	db.Commit(*commit)
	assert.False(t, contains(overlay, 2))
}

package statedb

import (
	"testing"

	"github.com/ChainSafe/gossamer/internal/primitives/core/hash"
	"github.com/ChainSafe/gossamer/pkg/scale"
	"github.com/stretchr/testify/assert"
)

func TestRefWindow_CreatedFromEmptyDB(t *testing.T) {
	db := NewTestDB(nil)
	pruning, err := newRefWindow[hash.H256, hash.H256](db, defaultMaxBlockConstraint)
	assert.NoError(t, err)
	assert.Equal(t, uint64(0), pruning.base)
	queue := pruning.queue.(*deathRowQueueInMem[hash.H256, hash.H256])
	assert.Equal(t, 0, queue.deathRows.Len())
	assert.Empty(t, queue.deathIndex)
}

func TestRefWindow_PruneEmpty(t *testing.T) {
	db := NewTestDB(nil)
	pruning, err := newRefWindow[hash.H256, hash.H256](db, defaultMaxBlockConstraint)
	assert.NoError(t, err)
	var commit CommitSet[hash.H256]
	assert.ErrorIs(t, pruning.PruneOne(&commit), ErrBlockUnavailable)
	assert.Equal(t, uint64(0), pruning.base)
	queue := pruning.queue.(*deathRowQueueInMem[hash.H256, hash.H256])
	assert.Equal(t, 0, queue.deathRows.Len())
	assert.Empty(t, queue.deathIndex)
}

//	fn check_journal(pruning: &RefWindow<H256, H256, TestDb>, db: &TestDb) {
//		let count_insertions = matches!(pruning.queue, DeathRowQueue::Mem { .. });
//		let restored: RefWindow<H256, H256, TestDb> =
//			RefWindow::new(db.clone(), DEFAULT_MAX_BLOCK_CONSTRAINT, count_insertions).unwrap();
//		assert_eq!(pruning.base, restored.base);
//		assert_eq!(pruning.queue.get_mem_queue_state(), restored.queue.get_mem_queue_state());
//	}
func checkJournal(t *testing.T, pruning refWindow[hash.H256, hash.H256], db TestDB) {
	t.Helper()
	restored, err := newRefWindow[hash.H256, hash.H256](db, defaultMaxBlockConstraint)
	assert.NoError(t, err)
	assert.Equal(t, pruning.base, restored.base)
	queue := pruning.queue.(*deathRowQueueInMem[hash.H256, hash.H256])
	var actual []deathRow[hash.H256, hash.H256]
	actualIndex := queue.deathIndex
	for i := 0; i < queue.deathRows.Len(); i++ {
		actual = append(actual, queue.deathRows.At(i))
	}
	queue = restored.queue.(*deathRowQueueInMem[hash.H256, hash.H256])
	expectedIndex := queue.deathIndex
	var expected []deathRow[hash.H256, hash.H256]
	for i := 0; i < queue.deathRows.Len(); i++ {
		expected = append(expected, queue.deathRows.At(i))
	}
	assert.Equal(t, expected, actual)
	assert.Equal(t, expectedIndex, actualIndex)

}

func TestRefWindow_PruneOne(t *testing.T) {
	db := NewTestDB([]uint64{1, 2, 3})
	pruning, err := newRefWindow[hash.H256, hash.H256](db, defaultMaxBlockConstraint)
	assert.NoError(t, err)
	commit := NewCommit([]uint64{4, 5}, []uint64{1, 3})
	h := hash.NewRandomH256()
	err = pruning.NoteCanonical(h, 0, &commit)
	assert.NoError(t, err)
	db.Commit(commit)
	assert.Equal(t, haveBlockYes, pruning.HaveBlock(h, 0))
	assert.Empty(t, commit.Data.Deleted)
	queue := pruning.queue.(*deathRowQueueInMem[hash.H256, hash.H256])
	assert.Equal(t, 1, queue.deathRows.Len())
	assert.Equal(t, 2, len(queue.deathIndex))
	assert.Equal(t, NewTestDB([]uint64{1, 2, 3, 4, 5}).Data, db.Data)
	checkJournal(t, pruning, db)

	commit = CommitSet[hash.H256]{}
	assert.NoError(t, pruning.PruneOne(&commit))
	assert.Equal(t, haveBlockNo, pruning.HaveBlock(h, 0))
	db.Commit(commit)
	assert.Equal(t, haveBlockNo, pruning.HaveBlock(h, 0))
	assert.Equal(t, NewTestDB([]uint64{2, 4, 5}).Data, db.Data)
	assert.Equal(t, 0, queue.deathRows.Len())
	assert.Empty(t, queue.deathIndex)
	assert.Equal(t, uint64(1), pruning.base)
}

func TestRefWindow_PruneTwo(t *testing.T) {
	db := NewTestDB([]uint64{1, 2, 3})
	pruning, err := newRefWindow[hash.H256, hash.H256](db, defaultMaxBlockConstraint)
	assert.NoError(t, err)
	commit := NewCommit([]uint64{4}, []uint64{1})
	err = pruning.NoteCanonical(hash.NewRandomH256(), 0, &commit)
	assert.NoError(t, err)
	db.Commit(commit)
	commit = NewCommit([]uint64{5}, []uint64{2})
	err = pruning.NoteCanonical(hash.NewRandomH256(), 1, &commit)
	assert.NoError(t, err)
	db.Commit(commit)
	assert.Equal(t, NewTestDB([]uint64{1, 2, 3, 4, 5}).Data, db.Data)

	checkJournal(t, pruning, db)

	commit = CommitSet[hash.H256]{}
	assert.NoError(t, pruning.PruneOne(&commit))
	db.Commit(commit)
	assert.Equal(t, NewTestDB([]uint64{2, 3, 4, 5}).Data, db.Data)
	commit = CommitSet[hash.H256]{}
	assert.NoError(t, pruning.PruneOne(&commit))
	db.Commit(commit)
	assert.Equal(t, NewTestDB([]uint64{3, 4, 5}).Data, db.Data)
	assert.Equal(t, uint64(2), pruning.base)
}

func TestRefWindow_PruneTwoPending(t *testing.T) {
	db := NewTestDB([]uint64{1, 2, 3})
	pruning, err := newRefWindow[hash.H256, hash.H256](db, defaultMaxBlockConstraint)
	assert.NoError(t, err)
	commit := NewCommit([]uint64{4}, []uint64{1})
	err = pruning.NoteCanonical(hash.NewRandomH256(), 0, &commit)
	assert.NoError(t, err)
	db.Commit(commit)
	commit = NewCommit([]uint64{5}, []uint64{2})
	err = pruning.NoteCanonical(hash.NewRandomH256(), 1, &commit)
	assert.NoError(t, err)
	db.Commit(commit)
	assert.Equal(t, NewTestDB([]uint64{1, 2, 3, 4, 5}).Data, db.Data)
	commit = CommitSet[hash.H256]{}
	assert.NoError(t, pruning.PruneOne(&commit))
	db.Commit(commit)
	assert.Equal(t, NewTestDB([]uint64{2, 3, 4, 5}).Data, db.Data)
	commit = CommitSet[hash.H256]{}
	assert.NoError(t, pruning.PruneOne(&commit))
	db.Commit(commit)
	assert.Equal(t, NewTestDB([]uint64{3, 4, 5}).Data, db.Data)
	assert.Equal(t, uint64(2), pruning.base)
}

func TestRefWindow_ReinsertedSurvives(t *testing.T) {
	db := NewTestDB([]uint64{1, 2, 3})
	pruning, err := newRefWindow[hash.H256, hash.H256](db, defaultMaxBlockConstraint)
	assert.NoError(t, err)
	commit := NewCommit([]uint64{}, []uint64{2})
	err = pruning.NoteCanonical(hash.NewRandomH256(), 0, &commit)
	assert.NoError(t, err)
	db.Commit(commit)
	commit = NewCommit([]uint64{2}, []uint64{})
	err = pruning.NoteCanonical(hash.NewRandomH256(), 1, &commit)
	assert.NoError(t, err)
	db.Commit(commit)
	commit = NewCommit([]uint64{}, []uint64{2})
	err = pruning.NoteCanonical(hash.NewRandomH256(), 2, &commit)
	assert.NoError(t, err)
	db.Commit(commit)
	assert.Equal(t, NewTestDB([]uint64{1, 2, 3}).Data, db.Data)

	checkJournal(t, pruning, db)

	commit = CommitSet[hash.H256]{}
	assert.NoError(t, pruning.PruneOne(&commit))
	db.Commit(commit)
	assert.Equal(t, NewTestDB([]uint64{1, 2, 3}).Data, db.Data)
	commit = CommitSet[hash.H256]{}
	assert.NoError(t, pruning.PruneOne(&commit))
	db.Commit(commit)
	assert.Equal(t, NewTestDB([]uint64{1, 2, 3}).Data, db.Data)
	commit = CommitSet[hash.H256]{}
	assert.NoError(t, pruning.PruneOne(&commit))
	db.Commit(commit)
	assert.Equal(t, NewTestDB([]uint64{1, 3}).Data, db.Data)
	assert.Equal(t, uint64(3), pruning.base)
}

func TestRefWindow_ReinsertedSurvivesPending(t *testing.T) {
	db := NewTestDB([]uint64{1, 2, 3})
	pruning, err := newRefWindow[hash.H256, hash.H256](db, defaultMaxBlockConstraint)
	assert.NoError(t, err)
	commit := NewCommit([]uint64{}, []uint64{2})
	err = pruning.NoteCanonical(hash.NewRandomH256(), 0, &commit)
	assert.NoError(t, err)
	db.Commit(commit)
	commit = NewCommit([]uint64{2}, []uint64{})
	err = pruning.NoteCanonical(hash.NewRandomH256(), 1, &commit)
	assert.NoError(t, err)
	db.Commit(commit)
	commit = NewCommit([]uint64{}, []uint64{2})
	err = pruning.NoteCanonical(hash.NewRandomH256(), 2, &commit)
	assert.NoError(t, err)
	db.Commit(commit)
	assert.Equal(t, NewTestDB([]uint64{1, 2, 3}).Data, db.Data)

	commit = CommitSet[hash.H256]{}
	assert.NoError(t, pruning.PruneOne(&commit))
	db.Commit(commit)
	assert.Equal(t, NewTestDB([]uint64{1, 2, 3}).Data, db.Data)
	commit = CommitSet[hash.H256]{}
	assert.NoError(t, pruning.PruneOne(&commit))
	db.Commit(commit)
	assert.Equal(t, NewTestDB([]uint64{1, 2, 3}).Data, db.Data)
	commit = CommitSet[hash.H256]{}
	assert.NoError(t, pruning.PruneOne(&commit))
	db.Commit(commit)
	assert.Equal(t, NewTestDB([]uint64{1, 3}).Data, db.Data)
	assert.Equal(t, uint64(3), pruning.base)
}

// note: this will be used in tests if we have a backed by DB version of deathRowQueue
func pushLastCanonicalized[H Hash](hash H, block uint64, commit *CommitSet[H]) {
	commit.Meta.Inserted = append(commit.Meta.Inserted, HashDBValue[[]byte]{
		Hash: toMetaKey(lastCanonical, struct{}{}),
		DBValue: scale.MustMarshal(hashBlock[H]{
			Hash:  hash,
			Block: block,
		}),
	})
}

// / Ensure that after warp syncing the state is stored correctly in the db. The warp sync target
// / block is imported with all its state at once. This test ensures that after a restart
// / `pruning` still knows that this block was imported.
func TestRefWindow_StoreCorrectStateAfterWarpSyncing(t *testing.T) {
	db := NewTestDB([]uint64{})
	pruning, err := newRefWindow[hash.H256, hash.H256](db, defaultMaxBlockConstraint)
	assert.NoError(t, err)
	block := uint64(10000)

	// import blocks
	h := hash.NewRandomH256()
	commit := NewCommit([]uint64{}, []uint64{})
	err = pruning.NoteCanonical(h, block, &commit)
	assert.NoError(t, err)
	// pushLastCanonicalized(h, block, &commit)
	db.Commit(commit)

	assert.Equal(t, haveBlockYes, pruning.HaveBlock(h, block))

	// load a new queue from db
	// `cache` should be the same
	pruning, err = newRefWindow[hash.H256, hash.H256](db, defaultMaxBlockConstraint)
	assert.NoError(t, err)
	assert.Equal(t, haveBlockYes, pruning.HaveBlock(h, block))
}

package dispute

import (
	"github.com/ChainSafe/gossamer/pkg/scale"
	"testing"

	"github.com/ChainSafe/gossamer/dot/parachain/dispute/types"
	parachainTypes "github.com/ChainSafe/gossamer/dot/parachain/types"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/dgraph-io/badger/v4"
	"github.com/stretchr/testify/require"
	"github.com/tidwall/btree"
)

func getSessionIndex(index uint) *parachainTypes.SessionIndex {
	sessionIndex := parachainTypes.SessionIndex(index)
	return &sessionIndex
}

func compareBTrees(tree1, tree2 *btree.BTree) bool {
	equal := true

	tree1.Ascend(nil, func(i interface{}) bool {
		if tree2.Get(i) == false {
			equal = false
			return false
		}
		return true
	})

	if equal {
		tree2.Ascend(nil, func(i interface{}) bool {
			if item := tree1.Get(i); item == nil {
				equal = false
				return false
			}
			return true
		})
	}

	return equal
}

func compareBTreeMaps[K types.Ordered, V any](map1, map2 *btree.Map[K, V]) bool {
	if map1.Len() != map2.Len() {
		return false
	}

	mismatch := false

	map1.Ascend(nil, func(key K, value V) bool {
		// TODO: check if we can compare the values as well
		if _, ok := map2.Get(key); !ok {
			mismatch = true
			return false
		}
		return true
	})

	return !mismatch
}

func TestDBBackend_SetEarliestSession(t *testing.T) {
	t.Parallel()

	// with
	db, err := badger.Open(badger.DefaultOptions(t.TempDir()))
	require.NoError(t, err)
	testEarliestSession := getSessionIndex(2)

	// when
	backend := NewDBBackend(db)
	err = backend.SetEarliestSession(testEarliestSession)
	require.NoError(t, err)

	// then
	earliestSession, err := backend.GetEarliestSession()
	require.NoError(t, err)
	require.Equal(t, *testEarliestSession, *earliestSession)
}

func TestDBBackend_SetRecentDisputes(t *testing.T) {
	t.Parallel()

	// with
	db, err := badger.Open(badger.DefaultOptions(t.TempDir()))
	require.NoError(t, err)
	disputes := scale.NewBTree[types.Dispute](types.CompareDisputes)
	dispute1, err := types.DummyDispute(1, common.Hash{1}, types.DisputeStatusActive)
	require.NoError(t, err)
	disputes.Value.Set(dispute1)
	dispute2, err := types.DummyDispute(2, common.Hash{2}, types.DisputeStatusConcludedFor)
	require.NoError(t, err)
	disputes.Value.Set(dispute2)

	// when
	backend := NewDBBackend(db)
	err = backend.SetRecentDisputes(disputes)
	require.NoError(t, err)

	// then
	recentDisputes, err := backend.GetRecentDisputes()
	require.NoError(t, err)
	require.True(t, compareBTrees(disputes.Value, recentDisputes.Value))
}

func TestDBBackend_SetCandidateVotes(t *testing.T) {
	t.Parallel()

	// with
	db, err := badger.Open(badger.DefaultOptions(t.TempDir()))
	require.NoError(t, err)
	candidateVotes := types.DummyCandidateVotes(t)

	// when
	backend := NewDBBackend(db)
	err = backend.SetCandidateVotes(1, common.Hash{1}, candidateVotes)
	require.NoError(t, err)

	// then
	actualCandidateVotes, err := backend.GetCandidateVotes(1, common.Hash{1})
	require.NoError(t, err)
	require.Equal(t, candidateVotes.CandidateReceipt, actualCandidateVotes.CandidateReceipt)
	require.True(t, compareBTrees(candidateVotes.Valid.BTree.Value, actualCandidateVotes.Valid.BTree.Value))
	require.True(t, compareBTreeMaps(candidateVotes.Invalid, actualCandidateVotes.Invalid))
}

func TestDBBackend_Write(t *testing.T) {
	t.Parallel()

	// with
	db, err := badger.Open(badger.DefaultOptions(t.TempDir()))
	require.NoError(t, err)
	earliestSession := getSessionIndex(1)
	disputes := scale.NewBTree[types.Dispute](types.CompareDisputes)
	dispute1, err := types.DummyDispute(1, common.Hash{1}, types.DisputeStatusActive)
	require.NoError(t, err)
	disputes.Value.Set(dispute1)
	dispute2, err := types.DummyDispute(2, common.Hash{2}, types.DisputeStatusConcludedFor)
	require.NoError(t, err)
	disputes.Value.Set(dispute2)
	candidateVotes := make(map[types.Comparator]*types.CandidateVotes)
	candidateVotes[types.Comparator{
		SessionIndex:  1,
		CandidateHash: common.Hash{1},
	}] = types.DummyCandidateVotes(t)
	candidateVotes[types.Comparator{
		SessionIndex:  2,
		CandidateHash: common.Hash{2},
	}] = types.DummyCandidateVotes(t)

	// when
	backend := NewDBBackend(db)
	err = backend.Write(earliestSession, disputes, candidateVotes)
	require.NoError(t, err)

	// then
	actualEarliestSession, err := backend.GetEarliestSession()
	require.NoError(t, err)
	require.Equal(t, *earliestSession, *actualEarliestSession)

	actualRecentDisputes, err := backend.GetRecentDisputes()
	require.NoError(t, err)
	require.True(t, compareBTrees(disputes.Value, actualRecentDisputes.Value))

	actualCandidateVotes, err := backend.GetCandidateVotes(1, common.Hash{1})
	require.NoError(t, err)
	expectedCandidateVotes := candidateVotes[types.Comparator{
		SessionIndex:  1,
		CandidateHash: common.Hash{1},
	}]
	require.Equal(t, expectedCandidateVotes.CandidateReceipt, actualCandidateVotes.CandidateReceipt)
	require.True(t, compareBTrees(expectedCandidateVotes.Valid.BTree.Value, actualCandidateVotes.Valid.BTree.Value))
	require.True(t, compareBTreeMaps(expectedCandidateVotes.Invalid, actualCandidateVotes.Invalid))
}

func TestDBBackend_setVotesCleanupTxn(t *testing.T) {
	t.Parallel()

	// with
	db, err := badger.Open(badger.DefaultOptions(t.TempDir()))
	require.NoError(t, err)
	candidateVotes := types.DummyCandidateVotes(t)

	// when
	backend := NewDBBackend(db)
	err = backend.SetCandidateVotes(1, common.Hash{1}, candidateVotes)
	require.NoError(t, err)

	err = backend.SetCandidateVotes(2, common.Hash{2}, candidateVotes)
	require.NoError(t, err)

	err = backend.SetCandidateVotes(3, common.Hash{3}, candidateVotes)
	require.NoError(t, err)

	err = backend.db.Update(func(txn *badger.Txn) error {
		return backend.setVotesCleanupTxn(txn, 2)

	})
	require.NoError(t, err)

	// then
	_, err = backend.GetCandidateVotes(1, common.Hash{1})
	require.Error(t, err)

	_, err = backend.GetCandidateVotes(3, common.Hash{3})
	require.NoError(t, err)
}

func TestDBBackend_watermark(t *testing.T) {
	t.Parallel()

	// with
	db, err := badger.Open(badger.DefaultOptions(t.TempDir()))
	require.NoError(t, err)

	// when
	backend := NewDBBackend(db)
	err = backend.db.Update(func(txn *badger.Txn) error {
		return backend.setWatermarkTxn(txn, 1)
	})
	require.NoError(t, err)

	// then
	watermark, err := backend.getWatermark()
	require.NoError(t, err)
	require.Equal(t, parachainTypes.SessionIndex(1), watermark)
}

func BenchmarkBadgerBackend_SetEarliestSession(b *testing.B) {
	db, err := badger.Open(badger.DefaultOptions(b.TempDir()))
	require.NoError(b, err)
	backend := NewDBBackend(db)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		err = backend.SetEarliestSession(getSessionIndex(1))
		require.NoError(b, err)
	}
}

func BenchmarkBadgerBackend_SetRecentDisputes(b *testing.B) {
	db, err := badger.Open(badger.DefaultOptions(b.TempDir()))
	require.NoError(b, err)
	backend := NewDBBackend(db)

	disputes := scale.NewBTree[types.Dispute](types.CompareDisputes)
	for i := 0; i < 10000; i++ {
		dispute, err := types.DummyDispute(parachainTypes.SessionIndex(i), common.Hash{byte(i)}, types.DisputeStatusActive)
		require.NoError(b, err)
		disputes.Value.Set(dispute)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		err = backend.SetRecentDisputes(disputes)
		require.NoError(b, err)
	}
}

func BenchmarkBadgerBackend_SetCandidateVotes(b *testing.B) {
	db, err := badger.Open(badger.DefaultOptions(b.TempDir()))
	require.NoError(b, err)
	backend := NewDBBackend(db)

	candidateVotes := types.DummyCandidateVotes(&testing.T{})

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		err = backend.SetCandidateVotes(parachainTypes.SessionIndex(i), common.Hash{1}, candidateVotes)
		require.NoError(b, err)
	}
}

func BenchmarkBadgerBackend_GetEarliestSession(b *testing.B) {
	db, err := badger.Open(badger.DefaultOptions(b.TempDir()))
	require.NoError(b, err)
	backend := NewDBBackend(db)

	err = backend.SetEarliestSession(getSessionIndex(1))
	require.NoError(b, err)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err = backend.GetEarliestSession()
		require.NoError(b, err)
	}
}

func BenchmarkBadgerBackend_GetRecentDisputes(b *testing.B) {
	db, err := badger.Open(badger.DefaultOptions(b.TempDir()))
	require.NoError(b, err)
	backend := NewDBBackend(db)

	disputes := scale.NewBTree[types.Dispute](types.CompareDisputes)
	for i := 0; i < 1000; i++ {
		dispute, err := types.DummyDispute(parachainTypes.SessionIndex(i), common.Hash{byte(1)}, types.DisputeStatusActive)
		require.NoError(b, err)
		disputes.Value.Set(dispute)
	}

	err = backend.SetRecentDisputes(disputes)
	require.NoError(b, err)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err = backend.GetRecentDisputes()
		require.NoError(b, err)
	}
}

func BenchmarkBadgerBackend_GetCandidateVotes(b *testing.B) {
	db, err := badger.Open(badger.DefaultOptions(b.TempDir()))
	require.NoError(b, err)
	backend := NewDBBackend(db)

	candidateVotes := types.DummyCandidateVotes(&testing.T{})
	for i := 0; i < b.N; i++ {
		err = backend.SetCandidateVotes(parachainTypes.SessionIndex(i), common.Hash{1}, candidateVotes)
		require.NoError(b, err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err = backend.GetCandidateVotes(parachainTypes.SessionIndex(i), common.Hash{1})
		require.NoError(b, err)
	}
}

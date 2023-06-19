package dispute

import (
	"testing"

	"github.com/ChainSafe/gossamer/dot/parachain/dispute/types"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/lib/parachain"
	"github.com/dgraph-io/badger/v4"
	"github.com/google/btree"
	"github.com/stretchr/testify/require"
)

func getSessionIndex(index uint) *parachain.SessionIndex {
	sessionIndex := parachain.SessionIndex(index)
	return &sessionIndex
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
	disputes := btree.New(DefaultBtreeDegree)
	dispute1, err := types.NewTestDispute(1, common.Hash{1}, types.DisputeStatusActive)
	require.NoError(t, err)
	disputes.ReplaceOrInsert(dispute1)
	dispute2, err := types.NewTestDispute(2, common.Hash{2}, types.DisputeStatusConcludedFor)
	require.NoError(t, err)
	disputes.ReplaceOrInsert(dispute2)

	// when
	backend := NewDBBackend(db)
	err = backend.SetRecentDisputes(disputes)
	require.NoError(t, err)

	// then
	recentDisputes, err := backend.GetRecentDisputes()
	require.NoError(t, err)
	require.Equal(t, disputes, recentDisputes)
}

func TestDBBackend_SetCandidateVotes(t *testing.T) {
	t.Parallel()

	// with
	db, err := badger.Open(badger.DefaultOptions(t.TempDir()))
	require.NoError(t, err)
	candidateVotes := types.NewTestCandidateVotes(t)

	// when
	backend := NewDBBackend(db)
	err = backend.SetCandidateVotes(1, common.Hash{1}, candidateVotes)
	require.NoError(t, err)

	// then
	actualCandidateVotes, err := backend.GetCandidateVotes(1, common.Hash{1})
	require.NoError(t, err)
	require.Equal(t, candidateVotes, actualCandidateVotes)
}

func TestDBBackend_Write(t *testing.T) {
	t.Parallel()

	// with
	db, err := badger.Open(badger.DefaultOptions(t.TempDir()))
	require.NoError(t, err)
	earliestSession := getSessionIndex(1)
	disputes := btree.New(DefaultBtreeDegree)
	dispute1, err := types.NewTestDispute(1, common.Hash{1}, types.DisputeStatusActive)
	require.NoError(t, err)
	disputes.ReplaceOrInsert(dispute1)
	dispute2, err := types.NewTestDispute(2, common.Hash{2}, types.DisputeStatusConcludedFor)
	require.NoError(t, err)
	disputes.ReplaceOrInsert(dispute2)
	candidateVotes := make(map[types.Comparator]*types.CandidateVotes)
	candidateVotes[types.Comparator{
		SessionIndex:  1,
		CandidateHash: common.Hash{1},
	}] = types.NewTestCandidateVotes(t)
	candidateVotes[types.Comparator{
		SessionIndex:  2,
		CandidateHash: common.Hash{2},
	}] = types.NewTestCandidateVotes(t)

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
	require.Equal(t, disputes, actualRecentDisputes)

	actualCandidateVotes, err := backend.GetCandidateVotes(1, common.Hash{1})
	require.NoError(t, err)
	require.Equal(t, candidateVotes[types.Comparator{
		SessionIndex:  1,
		CandidateHash: common.Hash{1},
	}], actualCandidateVotes)
}

func TestDBBackend_setVotesCleanupTxn(t *testing.T) {
	t.Parallel()

	// with
	db, err := badger.Open(badger.DefaultOptions(t.TempDir()))
	require.NoError(t, err)
	candidateVotes := types.NewTestCandidateVotes(t)

	// when
	backend := NewDBBackend(db)
	err = backend.SetCandidateVotes(1, common.Hash{1}, candidateVotes)
	require.NoError(t, err)

	err = backend.SetCandidateVotes(1, common.Hash{2}, candidateVotes)
	require.NoError(t, err)

	err = backend.SetCandidateVotes(2, common.Hash{3}, candidateVotes)
	require.NoError(t, err)

	err = backend.db.Update(func(txn *badger.Txn) error {
		return backend.setVotesCleanupTxn(txn, 2)

	})
	require.NoError(t, err)

	// then
	_, err = backend.GetCandidateVotes(1, common.Hash{1})
	require.Error(t, err)

	_, err = backend.GetCandidateVotes(2, common.Hash{3})
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
	require.Equal(t, parachain.SessionIndex(1), watermark)
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

	disputes := btree.New(DefaultBtreeDegree)
	for i := 0; i < 10000; i++ {
		dispute, err := types.NewTestDispute(parachain.SessionIndex(i), common.Hash{byte(i)}, types.DisputeStatusActive)
		require.NoError(b, err)
		disputes.ReplaceOrInsert(dispute)
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

	candidateVotes := types.NewTestCandidateVotes(&testing.T{})

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		err = backend.SetCandidateVotes(parachain.SessionIndex(i), common.Hash{1}, candidateVotes)
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

	disputes := btree.New(DefaultBtreeDegree)
	for i := 0; i < 1000; i++ {
		dispute, err := types.NewTestDispute(parachain.SessionIndex(i), common.Hash{byte(1)}, types.DisputeStatusActive)
		require.NoError(b, err)
		disputes.ReplaceOrInsert(dispute)
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

	candidateVotes := types.NewTestCandidateVotes(&testing.T{})
	for i := 0; i < b.N; i++ {
		err = backend.SetCandidateVotes(parachain.SessionIndex(i), common.Hash{1}, candidateVotes)
		require.NoError(b, err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err = backend.GetCandidateVotes(parachain.SessionIndex(i), common.Hash{1})
		require.NoError(b, err)
	}
}

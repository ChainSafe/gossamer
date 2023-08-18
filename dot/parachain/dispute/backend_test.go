package dispute

import (
	"sync"
	"testing"

	"github.com/ChainSafe/gossamer/dot/parachain/dispute/types"

	parachainTypes "github.com/ChainSafe/gossamer/dot/parachain/types"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/dgraph-io/badger/v4"
	"github.com/stretchr/testify/require"
	"github.com/tidwall/btree"
)

func TestOverlayBackend_EarliestSession(t *testing.T) {
	t.Parallel()

	// with
	db, err := badger.Open(badger.DefaultOptions(t.TempDir()))
	require.NoError(t, err)

	// when
	backend := newOverlayBackend(db)
	err = backend.SetEarliestSession(getSessionIndex(1))
	require.NoError(t, err)

	// then
	earliestSession, err := backend.GetEarliestSession()
	require.NoError(t, err)

	require.Equal(t, parachainTypes.SessionIndex(1), *earliestSession)
}

func TestOverlayBackend_RecentDisputes(t *testing.T) {
	t.Parallel()

	// with
	db, err := badger.Open(badger.DefaultOptions(t.TempDir()))
	require.NoError(t, err)
	disputes := btree.New(types.DisputeComparator)

	dispute1, err := types.NewTestDispute(1, common.Hash{1}, types.DisputeStatusActive)
	require.NoError(t, err)
	disputes.Set(dispute1)

	dispute2, err := types.NewTestDispute(2, common.Hash{2}, types.DisputeStatusConcludedFor)
	require.NoError(t, err)
	disputes.Set(dispute2)

	// when
	backend := newOverlayBackend(db)
	err = backend.SetRecentDisputes(disputes)
	require.NoError(t, err)

	// then
	recentDisputes, err := backend.GetRecentDisputes()
	require.NoError(t, err)
	require.True(t, compareBTrees(disputes, recentDisputes))
}

func TestOverlayBackend_CandidateVotes(t *testing.T) {
	t.Parallel()

	// with
	db, err := badger.Open(badger.DefaultOptions(t.TempDir()))
	require.NoError(t, err)
	candidateVotes1 := types.NewTestCandidateVotes(t)

	// when
	backend := newOverlayBackend(db)
	err = backend.SetCandidateVotes(1, common.Hash{1}, candidateVotes1)
	require.NoError(t, err)

	// then
	candidateVotes, err := backend.GetCandidateVotes(1, common.Hash{1})
	require.NoError(t, err)
	require.Equal(t, candidateVotes1, candidateVotes)
}

func TestOverlayBackend_Concurrency(t *testing.T) {
	t.Parallel()

	// with
	db, err := badger.Open(badger.DefaultOptions(t.TempDir()))
	require.NoError(t, err)
	backend := newOverlayBackend(db)

	numGoroutines := 10
	numIterations := 1000

	var wg sync.WaitGroup
	wg.Add(numGoroutines * 3)

	// test CandidateVotes
	for i := 0; i < numGoroutines; i++ {
		go func() {
			defer wg.Done()

			for j := 0; j < numIterations; j++ {
				err := backend.SetCandidateVotes(parachainTypes.SessionIndex(j),
					common.Hash{byte(j)}, types.NewTestCandidateVotes(t))
				require.NoError(t, err)
				_, err = backend.GetCandidateVotes(parachainTypes.SessionIndex(j), common.Hash{byte(j)})
				require.NoError(t, err)
			}
		}()
	}

	// test EarliestSession
	for i := 0; i < numGoroutines; i++ {
		go func() {
			defer wg.Done()

			for j := 0; j < numIterations; j++ {
				err := backend.SetEarliestSession(getSessionIndex(uint(j)))
				require.NoError(t, err)
				_, err = backend.GetEarliestSession()
				require.NoError(t, err)
			}
		}()
	}

	// test RecentDisputes
	for i := 0; i < numGoroutines; i++ {
		go func() {
			defer wg.Done()

			for j := 0; j < numIterations; j++ {
				disputes := btree.New(types.DisputeComparator)

				dispute1, err := types.NewTestDispute(parachainTypes.SessionIndex(j),
					common.Hash{byte(j)},
					types.DisputeStatusActive,
				)
				require.NoError(t, err)
				disputes.Set(dispute1)

				dispute2, err := types.NewTestDispute(parachainTypes.SessionIndex(j),
					common.Hash{byte(j)}, types.DisputeStatusConcludedFor)
				require.NoError(t, err)
				disputes.Set(dispute2)

				err = backend.SetRecentDisputes(disputes)
				require.NoError(t, err)
				_, err = backend.GetRecentDisputes()
				require.NoError(t, err)
			}
		}()
	}

	wg.Wait()
}

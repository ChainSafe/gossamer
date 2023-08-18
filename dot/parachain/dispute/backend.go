package dispute

import (
	"sync"
	"time"

	"github.com/ChainSafe/gossamer/dot/parachain/dispute/types"
	parachainTypes "github.com/ChainSafe/gossamer/dot/parachain/types"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/dgraph-io/badger/v4"
	"github.com/tidwall/btree"
)

// Backend is the backend for the dispute coordinator module.
type Backend interface {
	// GetEarliestSession returns the earliest session index, if any.
	GetEarliestSession() (*parachainTypes.SessionIndex, error)
	// GetRecentDisputes returns the recent disputes, if any.
	GetRecentDisputes() (*btree.BTree, error)
	// GetCandidateVotes returns the votes for the given candidate for the specific session-candidate pair, if any.
	GetCandidateVotes(session parachainTypes.SessionIndex, candidateHash common.Hash) (*types.CandidateVotes, error)

	// SetEarliestSession sets the earliest session index.
	SetEarliestSession(session *parachainTypes.SessionIndex) error
	// SetRecentDisputes sets the recent disputes.
	SetRecentDisputes(recentDisputes *btree.BTree) error
	// SetCandidateVotes sets the votes for the given candidate for the specific session-candidate pair.
	SetCandidateVotes(session parachainTypes.SessionIndex, candidateHash common.Hash, votes *types.CandidateVotes) error
}

// OverlayBackend is the overlay backend for the dispute coordinator module.
type OverlayBackend interface {
	Backend

	// WriteToDB writes the given dispute to the database.
	WriteToDB() error
	// GetActiveDisputes returns the active disputes.
	GetActiveDisputes(now int64) (*btree.BTree, error)
}

// DBBackend is the backend for the dispute coordinator module that uses a database.
type DBBackend interface {
	Backend

	// Write writes the given data to the database.
	Write(earliestSession *parachainTypes.SessionIndex,
		recentDisputes *btree.BTree,
		candidateVotes map[types.Comparator]*types.CandidateVotes) error
}

type syncedEarliestSession struct {
	*sync.RWMutex
	*parachainTypes.SessionIndex
}

func newSyncedEarliestSession() syncedEarliestSession {
	return syncedEarliestSession{
		RWMutex: new(sync.RWMutex),
	}
}

type syncedRecentDisputes struct {
	*sync.RWMutex
	*btree.BTree
}

func newSyncedRecentDisputes() syncedRecentDisputes {
	return syncedRecentDisputes{
		RWMutex: new(sync.RWMutex),
		BTree:   btree.New(types.DisputeComparator),
	}
}

type syncedCandidateVotes struct {
	*sync.RWMutex
	votes map[types.Comparator]*types.CandidateVotes
}

func newSyncedCandidateVotes() syncedCandidateVotes {
	return syncedCandidateVotes{
		RWMutex: new(sync.RWMutex),
		votes:   make(map[types.Comparator]*types.CandidateVotes),
	}
}

// overlayBackend implements OverlayBackend.
type overlayBackend struct {
	inner           DBBackend
	earliestSession syncedEarliestSession
	recentDisputes  syncedRecentDisputes
	candidateVotes  syncedCandidateVotes
}

func (b *overlayBackend) GetEarliestSession() (*parachainTypes.SessionIndex, error) {
	b.earliestSession.RLock()
	defer b.earliestSession.RUnlock()
	if b.earliestSession.SessionIndex != nil {
		return b.earliestSession.SessionIndex, nil
	}

	return b.inner.GetEarliestSession()
}

func (b *overlayBackend) GetRecentDisputes() (*btree.BTree, error) {
	b.recentDisputes.RLock()
	defer b.recentDisputes.RUnlock()
	if b.recentDisputes.Len() > 0 {
		return b.recentDisputes.BTree.Copy(), nil
	}

	return b.inner.GetRecentDisputes()
}

func (b *overlayBackend) GetCandidateVotes(session parachainTypes.SessionIndex,
	candidateHash common.Hash,
) (*types.CandidateVotes, error) {
	b.candidateVotes.RLock()
	defer b.candidateVotes.RUnlock()

	key := types.Comparator{
		SessionIndex:  session,
		CandidateHash: candidateHash,
	}
	if v, ok := b.candidateVotes.votes[key]; ok {
		return v, nil
	}

	return b.inner.GetCandidateVotes(session, candidateHash)
}

func (b *overlayBackend) SetEarliestSession(session *parachainTypes.SessionIndex) error {
	b.earliestSession.Lock()
	defer b.earliestSession.Unlock()
	b.earliestSession.SessionIndex = session
	return nil
}

func (b *overlayBackend) SetRecentDisputes(recentDisputes *btree.BTree) error {
	b.recentDisputes.Lock()
	defer b.recentDisputes.Unlock()
	b.recentDisputes.BTree = recentDisputes
	return nil
}

func (b *overlayBackend) SetCandidateVotes(session parachainTypes.SessionIndex,
	candidateHash common.Hash,
	votes *types.CandidateVotes,
) error {
	b.candidateVotes.Lock()
	defer b.candidateVotes.Unlock()

	key := types.Comparator{
		SessionIndex:  session,
		CandidateHash: candidateHash,
	}
	b.candidateVotes.votes[key] = votes
	return nil
}

// ActiveDuration an arbitrary duration for how long a dispute is considered active.
const ActiveDuration = 180 * time.Second

// GetActiveDisputes returns the active disputes, if any.
func (b *overlayBackend) GetActiveDisputes(now int64) (*btree.BTree, error) {
	b.recentDisputes.RLock()
	recentDisputes := b.recentDisputes.Copy()
	b.recentDisputes.RUnlock()

	activeDisputes := btree.New(types.DisputeComparator)
	recentDisputes.Ascend(nil, func(i interface{}) bool {
		dispute, ok := i.(*types.Dispute)
		if !ok {
			logger.Errorf("cast to dispute. Expected *types.Dispute, got %T", i)
			return true
		}

		concludedAt, err := dispute.DisputeStatus.ConcludedAt()
		if err != nil {
			logger.Errorf("failed to get concluded at: %s", err)
			return true
		}

		if concludedAt != nil && *concludedAt+uint64(ActiveDuration.Seconds()) > uint64(now) {
			activeDisputes.Set(dispute)
		}

		return true
	})

	return activeDisputes, nil
}

func (b *overlayBackend) WriteToDB() error {
	return b.inner.Write(b.earliestSession.SessionIndex, b.recentDisputes.BTree.Copy(), b.candidateVotes.votes)
}

var _ OverlayBackend = (*overlayBackend)(nil)

// newOverlayBackend creates a new overlayBackend.
func newOverlayBackend(db *badger.DB) *overlayBackend {
	return &overlayBackend{
		inner:           NewDBBackend(db),
		earliestSession: newSyncedEarliestSession(),
		recentDisputes:  newSyncedRecentDisputes(),
		candidateVotes:  newSyncedCandidateVotes(),
	}
}

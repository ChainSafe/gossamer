package dispute

import (
	"sync"
	"time"

	"github.com/ChainSafe/gossamer/dot/parachain/dispute/types"
	parachainTypes "github.com/ChainSafe/gossamer/dot/parachain/types"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/dgraph-io/badger/v4"
	"github.com/google/btree"
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

// DefaultBtreeDegree is the degree of the btree.
const DefaultBtreeDegree = 32 // TODO: determine the optimal degree during integration testing

// overlayBackend implements OverlayBackend.
type overlayBackend struct {
	inner           DBBackend
	earliestSession *parachainTypes.SessionIndex
	recentDisputes  *btree.BTree
	candidateVotes  map[types.Comparator]*types.CandidateVotes

	earliestSessionLock *sync.RWMutex
	recentDisputesLock  *sync.RWMutex
	candidateVotesLock  *sync.RWMutex
}

func (b *overlayBackend) GetEarliestSession() (*parachainTypes.SessionIndex, error) {
	b.earliestSessionLock.RLock()
	defer b.earliestSessionLock.RUnlock()
	if b.earliestSession != nil {
		return b.earliestSession, nil
	}

	return b.inner.GetEarliestSession()
}

func (b *overlayBackend) GetRecentDisputes() (*btree.BTree, error) {
	b.recentDisputesLock.RLock()
	defer b.recentDisputesLock.RUnlock()
	if b.recentDisputes.Len() > 0 {
		return b.recentDisputes, nil
	}

	return b.inner.GetRecentDisputes()
}

func (b *overlayBackend) GetCandidateVotes(session parachainTypes.SessionIndex,
	candidateHash common.Hash,
) (*types.CandidateVotes, error) {
	b.candidateVotesLock.RLock()
	defer b.candidateVotesLock.RUnlock()

	key := types.Comparator{
		SessionIndex:  session,
		CandidateHash: candidateHash,
	}
	if v, ok := b.candidateVotes[key]; ok {
		return v, nil
	}

	return b.inner.GetCandidateVotes(session, candidateHash)
}

func (b *overlayBackend) SetEarliestSession(session *parachainTypes.SessionIndex) error {
	b.earliestSessionLock.Lock()
	defer b.earliestSessionLock.Unlock()
	b.earliestSession = session
	return nil
}

func (b *overlayBackend) SetRecentDisputes(recentDisputes *btree.BTree) error {
	b.recentDisputesLock.Lock()
	defer b.recentDisputesLock.Unlock()
	b.recentDisputes = recentDisputes
	return nil
}

func (b *overlayBackend) SetCandidateVotes(session parachainTypes.SessionIndex,
	candidateHash common.Hash,
	votes *types.CandidateVotes,
) error {
	b.candidateVotesLock.Lock()
	defer b.candidateVotesLock.Unlock()

	key := types.Comparator{
		SessionIndex:  session,
		CandidateHash: candidateHash,
	}
	b.candidateVotes[key] = votes
	return nil
}

// ActiveDuration an arbitrary duration for how long a dispute is considered active.
const ActiveDuration = 180 * time.Second

// GetActiveDisputes returns the active disputes, if any.
func (b *overlayBackend) GetActiveDisputes(now int64) (*btree.BTree, error) {
	b.recentDisputesLock.RLock()
	recentDisputes := b.recentDisputes.Clone()
	b.recentDisputesLock.RUnlock()

	activeDisputes := btree.New(DefaultBtreeDegree)
	recentDisputes.Ascend(func(i btree.Item) bool {
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
			activeDisputes.ReplaceOrInsert(dispute)
		}

		return true
	})

	return activeDisputes, nil
}

func (b *overlayBackend) WriteToDB() error {
	return b.inner.Write(b.earliestSession, b.recentDisputes, b.candidateVotes)
}

var _ OverlayBackend = (*overlayBackend)(nil)

// newOverlayBackend creates a new overlayBackend.
func newOverlayBackend(db *badger.DB) *overlayBackend {
	return &overlayBackend{
		inner:               NewDBBackend(db),
		recentDisputes:      btree.New(DefaultBtreeDegree),
		candidateVotes:      make(map[types.Comparator]*types.CandidateVotes),
		earliestSessionLock: new(sync.RWMutex),
		recentDisputesLock:  new(sync.RWMutex),
		candidateVotesLock:  new(sync.RWMutex),
	}
}

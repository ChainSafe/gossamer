package dispute

import (
	"fmt"
	"github.com/ChainSafe/gossamer/pkg/scale"
	"sync"
	"time"

	"github.com/ChainSafe/gossamer/dot/parachain/dispute/types"
	parachainTypes "github.com/ChainSafe/gossamer/dot/parachain/types"
	"github.com/ChainSafe/gossamer/lib/common"
)

// Backend is the backend for the disputes coordinator module.
type Backend interface {
	// GetEarliestSession returns the earliest session index, if any.
	GetEarliestSession() (*parachainTypes.SessionIndex, error)
	// GetRecentDisputes returns the recent disputes, if any.
	GetRecentDisputes() (scale.BTree, error)
	// GetCandidateVotes returns the votes for the given candidate for the specific session-candidate pair, if any.
	GetCandidateVotes(session parachainTypes.SessionIndex, candidateHash common.Hash) (*types.CandidateVotes, error)

	// SetEarliestSession sets the earliest session index.
	SetEarliestSession(session *parachainTypes.SessionIndex) error
	// SetRecentDisputes sets the recent disputes.
	SetRecentDisputes(recentDisputes scale.BTree) error
	// SetCandidateVotes sets the votes for the given candidate for the specific session-candidate pair.
	SetCandidateVotes(session parachainTypes.SessionIndex, candidateHash common.Hash, votes *types.CandidateVotes) error
}

// OverlayBackend is the overlay backend for the disputes coordinator module.
type OverlayBackend interface {
	Backend

	// IsEmpty returns true if the overlay backend is empty.
	IsEmpty() bool
	// WriteToDB writes the given dispute to the database.
	WriteToDB() error
	// GetActiveDisputes returns the active disputes.
	GetActiveDisputes(now uint64) (scale.BTree, error)
	// NoteEarliestSession prunes data in the DB based on the provided session index.
	NoteEarliestSession(session parachainTypes.SessionIndex) error
}

// DBBackend is the backend for the disputes coordinator module that uses a database.
type DBBackend interface {
	Backend

	// Write writes the given data to the database.
	Write(earliestSession *parachainTypes.SessionIndex,
		recentDisputes scale.BTree,
		candidateVotes map[types.Comparator]*types.CandidateVotes) error
}

type syncedEarliestSession struct {
	sync.RWMutex
	*parachainTypes.SessionIndex
}

func newSyncedEarliestSession() syncedEarliestSession {
	return syncedEarliestSession{}
}

type syncedRecentDisputes struct {
	sync.RWMutex
	BTree scale.BTree
}

func newSyncedRecentDisputes() syncedRecentDisputes {
	return syncedRecentDisputes{
		BTree: scale.NewBTree[types.Dispute](types.CompareDisputes),
	}
}

type syncedCandidateVotes struct {
	sync.RWMutex
	votes map[types.Comparator]*types.CandidateVotes
}

func newSyncedCandidateVotes() syncedCandidateVotes {
	return syncedCandidateVotes{
		votes: make(map[types.Comparator]*types.CandidateVotes),
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

func (b *overlayBackend) GetRecentDisputes() (scale.BTree, error) {
	b.recentDisputes.RLock()
	defer b.recentDisputes.RUnlock()
	if b.recentDisputes.BTree.Len() > 0 {
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

func (b *overlayBackend) SetRecentDisputes(recentDisputes scale.BTree) error {
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
func (b *overlayBackend) GetActiveDisputes(now uint64) (scale.BTree, error) {
	b.recentDisputes.RLock()
	recentDisputes := b.recentDisputes.BTree.Copy()
	b.recentDisputes.RUnlock()

	activeDisputes := scale.NewBTree[types.Dispute](types.CompareDisputes)
	recentDisputes.Ascend(nil, func(i interface{}) bool {
		dispute, ok := i.(*types.Dispute)
		if !ok {
			return true
		}

		isDisputeInactive := func(status types.DisputeStatusVDT) bool {
			concludedAt, err := dispute.DisputeStatus.ConcludedAt()
			if err != nil {
				return false
			}
			return concludedAt != nil && *concludedAt+uint64(ActiveDuration.Seconds()) < now
		}

		if !isDisputeInactive(dispute.DisputeStatus) {
			activeDisputes.Set(dispute)
		}

		return true
	})

	return activeDisputes, nil
}

func (b *overlayBackend) IsEmpty() bool {
	return b.earliestSession.SessionIndex == nil && b.recentDisputes.BTree.Len() == 0 && len(b.candidateVotes.votes) == 0
}

func (b *overlayBackend) WriteToDB() error {
	return b.inner.Write(b.earliestSession.SessionIndex, b.recentDisputes.BTree.Copy(), b.candidateVotes.votes)
}

func (b *overlayBackend) NoteEarliestSession(session parachainTypes.SessionIndex) error {
	if b.earliestSession.SessionIndex == nil {
		b.earliestSession.SessionIndex = &session
		return nil
	}

	if *b.earliestSession.SessionIndex > session {
		b.earliestSession.SessionIndex = &session
		// clear recent disputes metadata
		recentDisputes, err := b.GetRecentDisputes()
		if err != nil {
			return fmt.Errorf("get recent disputes: %w", err)
		}

		// determine new recent disputes
		newRecentDisputes := scale.NewBTree[types.Dispute](types.CompareDisputes)
		recentDisputes.Ascend(nil, func(item interface{}) bool {
			dispute := item.(*types.Dispute)
			if dispute.Comparator.SessionIndex >= session {
				newRecentDisputes.Set(dispute)
			}
			return true
		})

		// prune obsolete disputes
		recentDisputes.Ascend(nil, func(item interface{}) bool {
			dispute := item.(*types.Dispute)
			if dispute.Comparator.SessionIndex < session {
				recentDisputes.Delete(dispute)
			}
			return true
		})

		// update db
		if recentDisputes.Len() > 0 {
			if err = b.SetRecentDisputes(newRecentDisputes); err != nil {
				return fmt.Errorf("set recent disputes: %w", err)
			}
		}
	}

	return nil
}

var _ OverlayBackend = (*overlayBackend)(nil)

// newOverlayBackend creates a new overlayBackend.
func newOverlayBackend(dbBackend DBBackend) *overlayBackend {
	return &overlayBackend{
		inner:           dbBackend,
		earliestSession: newSyncedEarliestSession(),
		recentDisputes:  newSyncedRecentDisputes(),
		candidateVotes:  newSyncedCandidateVotes(),
	}
}

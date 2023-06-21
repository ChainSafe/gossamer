package parachain

import (
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/lib/parachain"
	"github.com/emirpasic/gods/sets/treeset"
)

// SpamCount is the number of spam votes for a particular validator and session
type SpamCount uint32

// MaxSpamVotes is the maximum number of spam votes a validator can have for a particular session
const MaxSpamVotes = 50

// SpamSlots is an interface for managing spam votes
type SpamSlots interface {
	// AddUnconfirmed adds a spam vote for the given validator and candidate
	// returns true if the vote was added, false if the validator has already voted too many times
	// This is called for any validator's invalid vote for any not yet confirmed dispute
	AddUnconfirmed(session parachain.SessionIndex, candidate common.Hash, validator parachain.ValidatorIndex) bool
	// Clear out spam slots for a given candidate in a given session
	// We call this once a dispute becomes obsolete or got confirmed and thus votes for it should no longer be treated
	// as potential spam.
	Clear(session parachain.SessionIndex, candidate common.Hash)
	// PruneOld prune all spam slots for sessions older than the given index.
	PruneOld(oldestIndex parachain.SessionIndex)
}

// spamSlots is an implementation of SpamSlots
type spamSlots struct {
	Slots       map[[2]interface{}]SpamCount
	Unconfirmed map[[2]interface{}]*treeset.Set
}

func (s spamSlots) AddUnconfirmed(session parachain.SessionIndex, candidate common.Hash, validator parachain.ValidatorIndex) bool {
	if s.Slots[[2]interface{}{session, validator}] >= MaxSpamVotes {
		return false
	}

	validators := s.Unconfirmed[[2]interface{}{session, candidate}]

	if !validators.Contains(validator) {
		validators.Add(validator)
		s.Slots[[2]interface{}{session, validator}]++
	}

	return true
}

func (s spamSlots) Clear(session parachain.SessionIndex, candidate common.Hash) {
	if validators, ok := s.Unconfirmed[[2]interface{}{session, candidate}]; ok {
		for validator := range validators.Values() {
			s.Slots[[2]interface{}{session, validator}]--
			if s.Slots[[2]interface{}{session, validator}] <= 0 {
				delete(s.Slots, [2]interface{}{session, validator})
			}
		}
	}
}

func (s spamSlots) PruneOld(oldestIndex parachain.SessionIndex) {
	for k := range s.Unconfirmed {
		if k[0].(parachain.SessionIndex) < oldestIndex {
			delete(s.Unconfirmed, k)
		}
	}

	for k := range s.Slots {
		if k[0].(parachain.SessionIndex) < oldestIndex {
			delete(s.Slots, k)
		}
	}
}

// NewSpamSlots returns a new SpamSlots instance
func NewSpamSlots() SpamSlots {
	return &spamSlots{
		Slots:       make(map[[2]interface{}]SpamCount),
		Unconfirmed: make(map[[2]interface{}]*treeset.Set),
	}
}

// NewSpamSlotsFromState returns a new SpamSlots instance from the given state
func NewSpamSlotsFromState(unconfirmedDisputes map[[2]interface{}]*treeset.Set) SpamSlots {
	slots := make(map[[2]interface{}]SpamCount)

	for k, v := range unconfirmedDisputes {
		for validator := range v.Values() {
			// increment the spam count for this validator and session
			slots[[2]interface{}{k[0], validator}]++
			if slots[[2]interface{}{k[0], validator}] > MaxSpamVotes {
				// TODO: log this
			}
		}
	}

	return &spamSlots{
		Slots:       slots,
		Unconfirmed: unconfirmedDisputes,
	}
}

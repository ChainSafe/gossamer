package dispute

import (
	"sync"

	parachainTypes "github.com/ChainSafe/gossamer/dot/parachain/types"
	"github.com/ChainSafe/gossamer/internal/log"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/emirpasic/gods/sets/treeset"
	"golang.org/x/exp/maps"
)

// MaxSpamVotes is the maximum number of spam votes a validator can have for a particular session
const MaxSpamVotes = 50

// SpamSlots is an interface for managing spam votes
type SpamSlots interface {
	// AddUnconfirmed adds a spam vote for the given validator and candidate
	// returns true if the vote was added, false if the validator has already voted too many times
	// This is called for any validator's invalid vote for any not yet confirmed dispute
	AddUnconfirmed(session parachainTypes.SessionIndex,
		candidate common.Hash,
		validator parachainTypes.ValidatorIndex,
	) bool
	// Clear out spam slots for a given candidate in a given session
	// We call this once a dispute becomes obsolete or got confirmed and thus votes for it should no longer be treated
	// as potential spam.
	Clear(session parachainTypes.SessionIndex, candidate common.Hash)
	// PruneOld prune all spam slots for sessions older than the given index.
	PruneOld(oldestIndex parachainTypes.SessionIndex)
}

type slotKey struct {
	session   parachainTypes.SessionIndex
	validator parachainTypes.ValidatorIndex
}

type unconfirmedKey struct {
	session   parachainTypes.SessionIndex
	candidate common.Hash
}

type syncedSlots struct {
	sync.RWMutex
	value map[slotKey]uint32
}

type syncedUnconfirmedDisputes struct {
	sync.RWMutex
	value map[unconfirmedKey]*treeset.Set
}

// SpamSlotsHandler is an implementation of SpamSlots
type SpamSlotsHandler struct {
	slots        syncedSlots
	unconfirmed  syncedUnconfirmedDisputes
	maxSpamVotes uint32
}

// byValidatorIndex is a comparator for ValidatorIndex
func byValidatorIndex(a, b interface{}) int {
	return int(a.(parachainTypes.ValidatorIndex) - b.(parachainTypes.ValidatorIndex))
}

func newSlotsKey(session parachainTypes.SessionIndex, validator parachainTypes.ValidatorIndex) slotKey {
	return slotKey{
		session:   session,
		validator: validator,
	}
}

func newUnconfirmedDisputesKey(session parachainTypes.SessionIndex, candidate common.Hash) unconfirmedKey {
	return unconfirmedKey{
		session:   session,
		candidate: candidate,
	}
}

func (s *SpamSlotsHandler) getSpamCount(session parachainTypes.SessionIndex,
	validator parachainTypes.ValidatorIndex) (uint32, bool) {
	s.slots.RLock()
	defer s.slots.RUnlock()
	spamCount, ok := s.slots.value[newSlotsKey(session, validator)]
	return spamCount, ok
}

func (s *SpamSlotsHandler) getValidators(session parachainTypes.SessionIndex,
	candidate common.Hash,
) (*treeset.Set, bool) {
	s.unconfirmed.RLock()
	defer s.unconfirmed.RUnlock()
	validators, ok := s.unconfirmed.value[newUnconfirmedDisputesKey(session, candidate)]
	return validators, ok
}

func (s *SpamSlotsHandler) clearValidators(session parachainTypes.SessionIndex, candidate common.Hash) {
	s.unconfirmed.Lock()
	defer s.unconfirmed.Unlock()
	delete(s.unconfirmed.value, newUnconfirmedDisputesKey(session, candidate))
}

func (s *SpamSlotsHandler) clearSlots(session parachainTypes.SessionIndex, validator parachainTypes.ValidatorIndex) {
	s.slots.Lock()
	defer s.slots.Unlock()
	delete(s.slots.value, newSlotsKey(session, validator))
}

func (s *SpamSlotsHandler) incrementSpamCount(session parachainTypes.SessionIndex,
	validator parachainTypes.ValidatorIndex) {
	s.slots.Lock()
	defer s.slots.Unlock()
	s.slots.value[newSlotsKey(session, validator)]++
}

func (s *SpamSlotsHandler) AddUnconfirmed(session parachainTypes.SessionIndex,
	candidate common.Hash,
	validator parachainTypes.ValidatorIndex) bool {
	if spamCount, _ := s.getSpamCount(session, validator); spamCount >= s.maxSpamVotes {
		return false
	}

	s.unconfirmed.Lock()
	defer s.unconfirmed.Unlock()
	unconfirmedDisputesKey := newUnconfirmedDisputesKey(session, candidate)
	validators, ok := s.unconfirmed.value[unconfirmedDisputesKey]
	if !ok || validators == nil {
		validators = treeset.NewWith(byValidatorIndex)
	}
	if !validators.Contains(validator) {
		validators.Add(validator)
		s.incrementSpamCount(session, validator)
	}
	s.unconfirmed.value[unconfirmedDisputesKey] = validators
	return true
}

func (s *SpamSlotsHandler) Clear(session parachainTypes.SessionIndex, candidate common.Hash) {
	validators, ok := s.getValidators(session, candidate)
	if !ok {
		return
	}

	validatorSet := validators.Values()
	s.clearValidators(session, candidate)
	for _, validator := range validatorSet {
		spamCount, ok := s.getSpamCount(session, validator.(parachainTypes.ValidatorIndex))
		if !ok {
			continue
		}
		if spamCount == 1 {
			s.clearSlots(session, validator.(parachainTypes.ValidatorIndex))
			continue
		}
		s.slots.Lock()
		s.slots.value[newSlotsKey(session, validator.(parachainTypes.ValidatorIndex))] = spamCount - 1
		s.slots.Unlock()
	}
}

func (s *SpamSlotsHandler) PruneOld(oldestIndex parachainTypes.SessionIndex) {
	s.unconfirmed.Lock()
	maps.DeleteFunc(s.unconfirmed.value, func(k unconfirmedKey, v *treeset.Set) bool {
		return k.session < oldestIndex
	})
	s.unconfirmed.Unlock()

	s.slots.Lock()
	maps.DeleteFunc(s.slots.value, func(k slotKey, v uint32) bool {
		return k.session < oldestIndex
	})
	s.slots.Unlock()
}

var _ SpamSlots = (*SpamSlotsHandler)(nil)

// NewSpamSlots returns a new SpamSlotsHandler instance
func NewSpamSlots(maxSpamVotes uint32) *SpamSlotsHandler {
	return &SpamSlotsHandler{
		slots:        syncedSlots{value: make(map[slotKey]uint32)},
		unconfirmed:  syncedUnconfirmedDisputes{value: make(map[unconfirmedKey]*treeset.Set)},
		maxSpamVotes: maxSpamVotes,
	}
}

// NewSpamSlotsFromState returns a new SpamSlotsHandler instance from the given state
func NewSpamSlotsFromState(unconfirmedDisputes map[unconfirmedKey]*treeset.Set, maxSpamVotes uint32) *SpamSlotsHandler {
	slots := make(map[slotKey]uint32)
	for k, v := range unconfirmedDisputes {
		for validator := range v.Values() {
			// increment the spam count for this validator and session
			key := newSlotsKey(k.session, parachainTypes.ValidatorIndex(validator))
			slots[key]++
			if slots[key] > maxSpamVotes {
				// TODO: improve this after we have a logger for dispute coordinator
				log.Errorf("Spam count for validator %d in session %d is greater than max spam votes %d",
					validator, k.session, maxSpamVotes)
			}
		}
	}

	return &SpamSlotsHandler{
		slots:        syncedSlots{value: slots},
		unconfirmed:  syncedUnconfirmedDisputes{value: unconfirmedDisputes},
		maxSpamVotes: maxSpamVotes,
	}
}

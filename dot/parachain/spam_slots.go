package parachain

import (
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/lib/parachain"
	"github.com/emirpasic/gods/sets/treeset"
	"golang.org/x/exp/maps"
	"sync"
)

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

type slotKey struct {
	session   parachain.SessionIndex
	validator parachain.ValidatorIndex
}

type unconfirmedKey struct {
	session   parachain.SessionIndex
	candidate common.Hash
}

// SpamSlotsHandler is an implementation of SpamSlots
type SpamSlotsHandler struct {
	slots     map[slotKey]uint32
	slotsLock sync.RWMutex

	unconfirmed     map[unconfirmedKey]*treeset.Set
	unconfirmedLock sync.RWMutex

	maxSpamVotes uint32
}

// byValidatorIndex is a comparator for ValidatorIndex
func byValidatorIndex(a, b interface{}) int {
	return int(a.(parachain.ValidatorIndex) - b.(parachain.ValidatorIndex))
}

func newSlotsKey(session parachain.SessionIndex, validator parachain.ValidatorIndex) slotKey {
	return slotKey{
		session:   session,
		validator: validator,
	}
}

func newUnconfirmedDisputesKey(session parachain.SessionIndex, candidate common.Hash) unconfirmedKey {
	return unconfirmedKey{
		session:   session,
		candidate: candidate,
	}
}

func (s *SpamSlotsHandler) getSpamCount(session parachain.SessionIndex, validator parachain.ValidatorIndex) (uint32, bool) {
	s.slotsLock.RLock()
	defer s.slotsLock.RUnlock()
	spamCount, ok := s.slots[newSlotsKey(session, validator)]
	return spamCount, ok
}

func (s *SpamSlotsHandler) getValidators(session parachain.SessionIndex, candidate common.Hash) (*treeset.Set, bool) {
	s.unconfirmedLock.RLock()
	defer s.unconfirmedLock.RUnlock()
	validators, ok := s.unconfirmed[newUnconfirmedDisputesKey(session, candidate)]
	return validators, ok
}

func (s *SpamSlotsHandler) clearValidators(session parachain.SessionIndex, candidate common.Hash) {
	s.unconfirmedLock.Lock()
	defer s.unconfirmedLock.Unlock()
	delete(s.unconfirmed, newUnconfirmedDisputesKey(session, candidate))
}

func (s *SpamSlotsHandler) clearSlots(session parachain.SessionIndex, validator parachain.ValidatorIndex) {
	s.slotsLock.Lock()
	defer s.slotsLock.Unlock()
	delete(s.slots, newSlotsKey(session, validator))
}

func (s *SpamSlotsHandler) incrementSpamCount(session parachain.SessionIndex, validator parachain.ValidatorIndex) {
	s.slotsLock.Lock()
	defer s.slotsLock.Unlock()
	s.slots[newSlotsKey(session, validator)]++
}

func (s *SpamSlotsHandler) AddUnconfirmed(session parachain.SessionIndex, candidate common.Hash, validator parachain.ValidatorIndex) bool {
	if spamCount, _ := s.getSpamCount(session, validator); spamCount >= s.maxSpamVotes {
		return false
	}

	s.unconfirmedLock.Lock()
	defer s.unconfirmedLock.Unlock()

	validators, ok := s.unconfirmed[newUnconfirmedDisputesKey(session, candidate)]
	if !ok || validators == nil {
		validators = treeset.NewWith(byValidatorIndex)
	}

	if !validators.Contains(validator) {
		validators.Add(validator)
		s.incrementSpamCount(session, validator)
	}

	s.unconfirmed[newUnconfirmedDisputesKey(session, candidate)] = validators

	return true
}

func (s *SpamSlotsHandler) Clear(session parachain.SessionIndex, candidate common.Hash) {
	validators, ok := s.getValidators(session, candidate)
	if !ok {
		return
	}

	validatorSet := validators.Values()
	s.clearValidators(session, candidate)

	for _, validator := range validatorSet {
		spamCount, ok := s.getSpamCount(session, validator.(parachain.ValidatorIndex))
		if !ok {
			continue
		}

		if spamCount == 1 {
			s.clearSlots(session, validator.(parachain.ValidatorIndex))
			continue
		}

		s.slotsLock.Lock()
		s.slots[newSlotsKey(session, validator.(parachain.ValidatorIndex))] = spamCount - 1
		s.slotsLock.Unlock()

	}

}

func (s *SpamSlotsHandler) PruneOld(oldestIndex parachain.SessionIndex) {
	s.unconfirmedLock.Lock()
	maps.DeleteFunc(s.unconfirmed, func(k unconfirmedKey, v *treeset.Set) bool {
		return k.session < oldestIndex
	})
	s.unconfirmedLock.Unlock()

	s.slotsLock.Lock()
	maps.DeleteFunc(s.slots, func(k slotKey, v uint32) bool {
		return k.session < oldestIndex
	})
	s.slotsLock.Unlock()
}

var _ SpamSlots = &SpamSlotsHandler{}

// NewSpamSlots returns a new SpamSlotsHandler instance
func NewSpamSlots(maxSpamVotes uint32) *SpamSlotsHandler {
	return &SpamSlotsHandler{
		slots:        make(map[slotKey]uint32),
		unconfirmed:  make(map[unconfirmedKey]*treeset.Set),
		maxSpamVotes: maxSpamVotes,
	}
}

// NewSpamSlotsFromState returns a new SpamSlotsHandler instance from the given state
func NewSpamSlotsFromState(unconfirmedDisputes map[unconfirmedKey]*treeset.Set, maxSpamVotes uint32) *SpamSlotsHandler {
	slots := make(map[slotKey]uint32)

	for k, v := range unconfirmedDisputes {
		for validator := range v.Values() {
			// increment the spam count for this validator and session
			key := newSlotsKey(k.session, parachain.ValidatorIndex(validator))
			slots[key]++
			if slots[key] > maxSpamVotes {
				// TODO: log this after we have a logger
			}
		}
	}

	return &SpamSlotsHandler{
		slots:        slots,
		unconfirmed:  unconfirmedDisputes,
		maxSpamVotes: maxSpamVotes,
	}
}

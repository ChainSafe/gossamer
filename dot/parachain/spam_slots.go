package parachain

import (
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/lib/parachain"
	"github.com/emirpasic/gods/sets/treeset"
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

// spamSlots is an implementation of SpamSlots
type spamSlots struct {
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

func slotsKey(session parachain.SessionIndex, validator parachain.ValidatorIndex) slotKey {
	return slotKey{
		session:   session,
		validator: validator,
	}
}

func unconfirmedDisputesKey(session parachain.SessionIndex, candidate common.Hash) unconfirmedKey {
	return unconfirmedKey{
		session:   session,
		candidate: candidate,
	}
}

func (s *spamSlots) getSpamCount(session parachain.SessionIndex, validator parachain.ValidatorIndex) (uint32, bool) {
	s.slotsLock.RLock()
	defer s.slotsLock.RUnlock()
	spamCount, ok := s.slots[slotsKey(session, validator)]
	return spamCount, ok
}

func (s *spamSlots) getValidators(session parachain.SessionIndex, candidate common.Hash) (*treeset.Set, bool) {
	s.unconfirmedLock.RLock()
	defer s.unconfirmedLock.RUnlock()
	validators, ok := s.unconfirmed[unconfirmedDisputesKey(session, candidate)]
	return validators, ok
}

func (s *spamSlots) clearValidators(session parachain.SessionIndex, candidate common.Hash) {
	s.unconfirmedLock.Lock()
	defer s.unconfirmedLock.Unlock()
	delete(s.unconfirmed, unconfirmedDisputesKey(session, candidate))
}

func (s *spamSlots) clearSlots(session parachain.SessionIndex, validator parachain.ValidatorIndex) {
	s.slotsLock.Lock()
	defer s.slotsLock.Unlock()
	delete(s.slots, slotsKey(session, validator))
}

func (s *spamSlots) incrementSpamCount(session parachain.SessionIndex, validator parachain.ValidatorIndex) {
	s.slotsLock.Lock()
	defer s.slotsLock.Unlock()
	s.slots[slotsKey(session, validator)]++
}

func (s *spamSlots) AddUnconfirmed(session parachain.SessionIndex, candidate common.Hash, validator parachain.ValidatorIndex) bool {
	if spamCount, _ := s.getSpamCount(session, validator); spamCount >= s.maxSpamVotes {
		return false
	}

	s.unconfirmedLock.Lock()
	defer s.unconfirmedLock.Unlock()

	validators, ok := s.unconfirmed[unconfirmedDisputesKey(session, candidate)]
	if !ok || validators == nil {
		validators = treeset.NewWith(byValidatorIndex)
	}

	if !validators.Contains(validator) {
		validators.Add(validator)
		s.incrementSpamCount(session, validator)
	}

	s.unconfirmed[unconfirmedDisputesKey(session, candidate)] = validators

	return true
}

func (s *spamSlots) Clear(session parachain.SessionIndex, candidate common.Hash) {
	if validators, ok := s.getValidators(session, candidate); ok {
		validatorSet := validators.Values()
		s.clearValidators(session, candidate)

		for _, validator := range validatorSet {
			spamCount, ok := s.getSpamCount(session, validator.(parachain.ValidatorIndex))
			if ok {
				if spamCount == 1 {
					s.clearSlots(session, validator.(parachain.ValidatorIndex))
					continue
				}

				s.slotsLock.Lock()
				s.slots[slotsKey(session, validator.(parachain.ValidatorIndex))] = spamCount - 1
				s.slotsLock.Unlock()
			}
		}
	}
}

func (s *spamSlots) PruneOld(oldestIndex parachain.SessionIndex) {
	s.unconfirmedLock.Lock()
	unconfirmedToDelete := make([]unconfirmedKey, 0)
	for k := range s.unconfirmed {
		if k.session < oldestIndex {
			unconfirmedToDelete = append(unconfirmedToDelete, k)
		}
	}

	for _, k := range unconfirmedToDelete {
		delete(s.unconfirmed, k)
	}
	s.unconfirmedLock.Unlock()

	s.slotsLock.Lock()
	slotsToDelete := make([]slotKey, 0)
	for k := range s.slots {
		if k.session < oldestIndex {
			slotsToDelete = append(slotsToDelete, k)
		}
	}

	for _, k := range slotsToDelete {
		delete(s.slots, k)
	}
	s.slotsLock.Unlock()
}

// NewSpamSlots returns a new SpamSlots instance
func NewSpamSlots(maxSpamVotes uint32) SpamSlots {
	return &spamSlots{
		slots:        make(map[slotKey]uint32),
		unconfirmed:  make(map[unconfirmedKey]*treeset.Set),
		maxSpamVotes: maxSpamVotes,
	}
}

// NewSpamSlotsFromState returns a new SpamSlots instance from the given state
func NewSpamSlotsFromState(unconfirmedDisputes map[unconfirmedKey]*treeset.Set, maxSpamVotes uint32) SpamSlots {
	slots := make(map[slotKey]uint32)

	for k, v := range unconfirmedDisputes {
		for validator := range v.Values() {
			// increment the spam count for this validator and session
			key := slotsKey(k.session, parachain.ValidatorIndex(validator))
			slots[key]++
			if slots[key] > maxSpamVotes {
				// TODO: log this
			}
		}
	}

	return &spamSlots{
		slots:        slots,
		unconfirmed:  unconfirmedDisputes,
		maxSpamVotes: maxSpamVotes,
	}
}

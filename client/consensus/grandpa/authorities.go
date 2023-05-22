// Copyright 2023 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package grandpa

import (
	"errors"
	"fmt"
	"github.com/ChainSafe/gossamer/dot/types"
	"github.com/ChainSafe/gossamer/lib/common"
)

// AuthorityList A list of Grandpa authorities with associated weights.
// TODO migrate this type and associated functions to this package
type AuthorityList []types.Authority

// PendingChange A pending change to the authority set.
//
// This will be applied when the announcing block is at some depth within
// the finalized or unfinalized chain.
type PendingChange struct {
	nextAuthorities AuthorityList
	delay           uint
	canonHeight     uint
	canonHash       common.Hash
	delayKind       DelayKind
}

// EffectiveNumber Returns the effective number this change will be applied at.
func (pc *PendingChange) EffectiveNumber() uint {
	return pc.canonHeight + pc.delay
}

// AuthoritySetChanges Tracks historical authority set changes. We store the block numbers for the last block
// of each authority set, once they have been finalized. These blocks are guaranteed to
// have a justification unless they were triggered by a forced change.
type AuthoritySetChanges []struct {
	setId       uint64
	blockNumber uint
}

// AuthoritySet A set of authorities.
type AuthoritySet struct {
	// The current active authorities.
	currentAuthorities AuthorityList
	// The current set id.
	setId uint64
	// Tree of pending standard changes across forks. Standard changes are
	// enacted on finality and must be enacted (i.e. finalized) in-order across
	// a given branch
	pendingStandardChanges ChangeTree
	// Pending forced changes across different forks (at most one per fork).
	// Forced changes are enacted on block depth (not finality), for this
	// reason only one forced change should exist per fork. When trying to
	// apply forced changes we keep track of any pending standard changes that
	// they may depend on, this is done by making sure that any pending change
	// that is an ancestor of the forced changed and its effective block number
	// is lower than the last finalized block (as signaled in the forced
	// change) must be applied beforehand.
	pendingForcedChanges []PendingChange
	// Track at which blocks the set id changed. This is useful when we need to prove finality for
	// a given block since we can figure out what set the block belongs to and when the set
	// started/ended.
	authoritySetChanges AuthoritySetChanges
}

// InvalidAuthorityList authority sets must be non-empty and all weights must be greater than 0
func (authSet *AuthoritySet) InvalidAuthorityList(authorities AuthorityList) bool {
	if len(authorities) == 0 {
		return true
	}

	for _, authority := range authorities {
		if authority.Weight == 0 {
			return true
		}
	}
	return false
}

// IsDescendentOf is a type to represent the function signature of a IsDescendentOf function
type IsDescendentOf func(h1 common.Hash, h2 common.Hash) (bool, error)

// addPendingChange Note an upcoming pending transition. Multiple pending standard changes
// on the same branch can be added as long as they don't overlap. Forced
// changes are restricted to one per fork. This method assumes that changes
// on the same branch will be added in-order. The given function
// `is_descendent_of` should return `true` if the second hash (target) is a
// descendent of the first hash (base).
func (authSet *AuthoritySet) addPendingChange(pending PendingChange, isDescendentOf IsDescendentOf) error {
	if authSet.InvalidAuthorityList(pending.nextAuthorities) {
		return errors.New("invalid authority set, either empty or with an authority weight set to 0")
	}

	switch pending.delayKind.value.(type) {
	case Finalized:
		return authSet.addStandardChange(pending, isDescendentOf)
	case Best:
		return authSet.addForcedChange(pending, isDescendentOf)
	default:
		panic("delayKind is invalid type")
	}

	return nil
}

// Key is used to represent a tuple ordered first by effective number and then by signal-block number
type Key struct {
	effectiveNumber   uint
	signalBlockNumber uint
}

// Equals compares two Keys to check is they are equal
func (k *Key) Equals(key Key) bool {
	if key.effectiveNumber == k.effectiveNumber &&
		key.signalBlockNumber == k.signalBlockNumber {
		return true
	}
	return false
}

func (authSet *AuthoritySet) addForcedChange(pending PendingChange, isDescendentOf IsDescendentOf) error {
	for _, change := range authSet.pendingForcedChanges {
		if change.canonHash == pending.canonHash {
			return errors.New("duplicate authority set change")
		}

		isDescendent, err := isDescendentOf(change.canonHash, pending.canonHash)
		if err != nil {
			return fmt.Errorf("checking isDescendentOf")
		}

		if isDescendent {
			return errors.New("multiple pending forced authority set changes are not allowed")
		}
	}

	key := Key{
		pending.EffectiveNumber(),
		pending.canonHeight,
	}

	idx, err := SearchKey(key, authSet.pendingForcedChanges)
	if err != nil {
		return err
	}

	logger.Debugf(
		"inserting potential forced set change at block number %d (delayed by %d blocks).",
		pending.canonHeight, pending.delay,
	)

	authSet.pendingForcedChanges[idx] = pending

	logger.Debugf(
		"there are now %d pending forced changes",
		len(authSet.pendingForcedChanges),
	)

	return nil
}

func (authSet *AuthoritySet) addStandardChange(pending PendingChange, isDescendentOf IsDescendentOf) error {
	hash := pending.canonHash
	number := pending.canonHeight

	logger.Debugf(
		"inserting potential standard set change signaled at block %d (delayed by %d blocks).",
		number, pending.delay,
	)

	_, err := authSet.pendingStandardChanges.Import(hash, number, pending, isDescendentOf)
	if err != nil {
		return err
	}

	// TODO substrate has a log here
	return nil
}

// CurrentLimit Get the earliest limit-block number, if any. If there are pending changes across
// different forks, this method will return the earliest effective number (across the
// different branches) that is higher or equal to the given min number.
//
// Only standard changes are taken into account for the current
// limit, since any existing forced change should preclude the voter from voting.
func (authSet *AuthoritySet) CurrentLimit(min uint) (limit *uint) {
	roots := authSet.pendingStandardChanges.Roots()
	for i := 0; i < len(roots); i++ {
		effectiveNumber := roots[i].change.EffectiveNumber()
		if effectiveNumber >= min {
			if limit == nil {
				limit = &effectiveNumber
			} else if effectiveNumber < *limit {
				*limit = effectiveNumber
			}
		}
	}
	return limit
}

// SharedAuthoritySet A shared authority set.
// TODO implement shared logic
type SharedAuthoritySet struct {
	authoritySet AuthoritySet
}

// Inner Returns access to the [`AuthoritySet`].
// TODO update with shared logic
func (sas *SharedAuthoritySet) Inner() AuthoritySet {
	return sas.authoritySet
}

// InnerLocked
// Returns access to the [`AuthoritySet`] and locks it.
//
// For more information see [`SharedDataLocked`].
// TODO update with shared logic
func (sas *SharedAuthoritySet) InnerLocked() AuthoritySet {
	return sas.authoritySet
}

// Copyright 2023 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package grandpa

import (
	"errors"
	"fmt"
	"github.com/ChainSafe/gossamer/dot/telemetry"
	"github.com/ChainSafe/gossamer/dot/types"
	"github.com/ChainSafe/gossamer/lib/common"
)

var (
	errInvalidAuthoritySet                           = errors.New("invalid authority set, either empty or with an authority weight set to 0")
	errDuplicateAuthoritySetChanges                  = errors.New("duplicate authority set change")
	errMultiplePendingForcedAuthoritySetChanges      = errors.New("multiple pending forced authority set changes are not allowed")
	errForcedAuthoritySetChangeDependencyUnsatisfied = errors.New("a pending forced authority set change could not be applied since it must be applied after the pending standard change")
)

// AuthorityList A list of Grandpa authorities with associated weights.
// TODO migrate this type and associated functions to this package
type AuthorityList []types.Authority

// PendingChange A pending change to the authority set.
//
// This will be applied when the announcing block is at some depth within
// the finalized or unfinalized chain.
type PendingChange struct {
	// The new authorities and weights to apply.
	nextAuthorities AuthorityList
	// How deep in the chain the announcing block must be
	// before the change is applied.
	delay uint
	// The announcing block's height.
	canonHeight uint
	// The announcing block's hash.
	canonHash common.Hash
	// The delay kind.
	delayKind DelayKind
}

// EffectiveNumber Returns the effective number this change will be applied at.
func (pc *PendingChange) EffectiveNumber() uint {
	return pc.canonHeight + pc.delay
}

type AuthorityChange struct {
	setId       uint64
	blockNumber uint
}

// AuthoritySetChanges Tracks historical authority set changes. We store the block numbers for the last block
// of each authority set, once they have been finalized. These blocks are guaranteed to
// have a justification unless they were triggered by a forced change.
type AuthoritySetChanges []AuthorityChange

// Block where set changed
type newSetBlockInfo struct {
	newSetBlockNumber uint
	newSetBlockHash   common.Hash
}

// Status of the set after changes were applied.
type Status struct {
	// Whether internal changes were made.
	changed bool
	// `Some` when underlying authority set has changed, containing the
	// block where that set changed.
	newSetBlock *newSetBlockInfo
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
	pendingStandardChanges ForkTree
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

// Genesis Get a genesis set with given authorities.
func Genesis(initial AuthorityList) (authSet *AuthoritySet) {
	if authSet.InvalidAuthorityList(initial) {
		return nil
	}

	authSet = &AuthoritySet{
		currentAuthorities:     initial,
		setId:                  0,
		pendingStandardChanges: NewChangeTree(),
		pendingForcedChanges:   nil,
		authoritySetChanges:    nil,
	}
	return
}

func NewAuthoritySet(authorities AuthorityList,
	setId uint64,
	pendingStandardChanges ForkTree,
	pendingForcedChanges []PendingChange,
	authoritySetChanges AuthoritySetChanges,
) (authSet *AuthoritySet) {
	if authSet.InvalidAuthorityList(authorities) {
		return nil
	}

	authSet = &AuthoritySet{
		currentAuthorities:     authorities,
		setId:                  setId,
		pendingStandardChanges: pendingStandardChanges,
		pendingForcedChanges:   pendingForcedChanges,
		authoritySetChanges:    authoritySetChanges,
	}
	return
}

type predicate[T any] func(T) bool

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
		return errInvalidAuthoritySet
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
			return errDuplicateAuthoritySetChanges
			//return errors.New("duplicate authority set change")
		}

		isDescendent, err := isDescendentOf(change.canonHash, pending.canonHash)
		if err != nil {
			return fmt.Errorf("checking isDescendentOf")
		}

		if isDescendent {
			return errMultiplePendingForcedAuthoritySetChanges
		}
	}

	key := Key{
		pending.EffectiveNumber(),
		pending.canonHeight,
	}

	// Search by effective key
	idx := SearchKey(key, authSet.pendingForcedChanges)

	logger.Debugf(
		"inserting potential forced set change at block number %d (delayed by %d blocks).",
		pending.canonHeight, pending.delay,
	)

	// Insert change at index
	if len(authSet.pendingForcedChanges) <= idx {
		authSet.pendingForcedChanges = append(authSet.pendingForcedChanges, pending)
	} else {
		authSet.pendingForcedChanges = append(
			authSet.pendingForcedChanges[:idx+1], authSet.pendingForcedChanges[idx:]...)
		authSet.pendingForcedChanges[idx] = pending
	}

	logger.Debugf(
		"there are now %d pending forced changes",
		len(authSet.pendingForcedChanges),
	)

	return nil
}

type Change struct {
	hash   common.Hash
	number uint
}

// Returns the block hash and height at which the next pending change in
// the given chain (i.e. it includes `best_hash`) was signalled, `None` if
// there are no pending changes for the given chain.
//
// This is useful since we know that when a change is signalled the
// underlying runtime authority set management module (e.g. session module)
// has updated its internal state (e.g. a new session started).
func (authSet *AuthoritySet) nextChange(bestHash common.Hash, isDescendentOf IsDescendentOf) (*Change, error) {
	var forced *Change
	for _, change := range authSet.pendingForcedChanges {
		isDesc, err := isDescendentOf(change.canonHash, bestHash)
		if err != nil {
			return nil, err
		}
		if isDesc {
			forced = &Change{
				hash:   change.canonHash,
				number: change.canonHeight,
			}
			break
		}
	}

	var standard *Change
	for _, changeNode := range authSet.pendingStandardChanges.Roots() {
		change := changeNode.change
		isDesc, err := isDescendentOf(change.canonHash, bestHash)
		if err != nil {
			return nil, err
		}
		if isDesc {
			standard = &Change{
				hash:   change.canonHash,
				number: change.canonHeight,
			}
			break
		}
	}

	if standard != nil && forced != nil {
		if forced.number < standard.number {
			return forced, nil
		} else {
			return standard, nil
		}
	} else if forced != nil {
		return forced, nil
	} else if standard != nil {
		return standard, nil
	}
	return nil, nil
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

// PendingChanges Inspect pending changes. Standard pending changes are iterated first,
// and the changes in the roots are traversed in pre-order, afterwards all
// forced changes are iterated.
func (authSet *AuthoritySet) PendingChanges() []PendingChange {
	// get everything from standard change roots
	changes := authSet.pendingStandardChanges.GetPreOrder()

	// then get everything from forced changes
	changes = append(changes, authSet.pendingForcedChanges...)

	return changes
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

type AppliedChanges struct {
	num uint
	set AuthoritySet
}

// ApplyForcedChanges Apply or prune any pending transitions based on a best-block trigger.
//
// Returns `Ok((median, new_set))` when a forced change has occurred. The
// median represents the median last finalized block at the time the change
// was signaled, and it should be used as the canon block when starting the
// new grandpa voter. Only alters the internal state in this case.
//
// These transitions are always forced and do not lead to justifications
// which light clients can follow.
//
// Forced changes can only be applied after all pending standard changes
// that it depends on have been applied. If any pending standard change
// exists that is an ancestor of a given forced changed and which effective
// block number is lower than the last finalized block (as defined by the
// forced change), then the forced change cannot be applied. An error will
// be returned in that case which will prevent block import.
func (authSet *AuthoritySet) applyForcedChanges(bestHash common.Hash,
	bestNumber uint,
	isDescendentOf IsDescendentOf,
	initialSync bool,
	telemetry *telemetry.Client) (newSet *AppliedChanges, err error) {

	for _, change := range authSet.pendingForcedChanges {
		// double check this logic for what to iterate over, but try this for now
		effectiveNumber := change.EffectiveNumber()
		if effectiveNumber > bestNumber {
			continue
		} else if effectiveNumber == bestNumber {
			// check if the given best block is in the same branch as
			// the block that signaled the change.
			isDesc, err := isDescendentOf(change.canonHash, bestHash)
			if err != nil {
				return nil, err
			}
			if change.canonHash == bestHash || isDesc {
				// I think this cast is okay since we should probably panic if we hit this
				medianLastFinalized := change.delayKind.value.(Best).medianLastFinalized

				roots := authSet.pendingStandardChanges.Roots()
				for _, standardChangeNode := range roots {
					standardChange := standardChangeNode.change

					isDescStandard, err := isDescendentOf(standardChange.canonHash, change.canonHash)
					if err != nil {
						return nil, err
					}
					if standardChange.EffectiveNumber() <= medianLastFinalized && isDescStandard {
						// TODO log here
						return nil, errForcedAuthoritySetChangeDependencyUnsatisfied
					}
				}

				// TODO grandpa log

				// TODO telemetry

				authorityChange := AuthorityChange{
					setId:       authSet.setId,
					blockNumber: medianLastFinalized,
				}

				authSetChanges := authSet.authoritySetChanges
				authSetChanges = append(authSetChanges, authorityChange)
				newSet = &AppliedChanges{
					medianLastFinalized,
					AuthoritySet{
						currentAuthorities:     change.nextAuthorities,
						setId:                  authSet.setId + 1,
						pendingStandardChanges: NewChangeTree(), // new set, new changes
						pendingForcedChanges:   nil,
						authoritySetChanges:    authSetChanges,
					},
				}
				return newSet, nil
			}
		}
	}

	return newSet, nil
}

func applyStandardChangesPredicate(finalizedNumber uint) predicate[*PendingChange] {
	return func(change *PendingChange) bool {
		return change.EffectiveNumber() <= finalizedNumber
	}
}

func enactStandardChangesPredicate(finalizedNumber uint) predicate[*PendingChange] {
	return func(change *PendingChange) bool {
		return change.EffectiveNumber() == finalizedNumber
	}
}

// ApplyStandardChanges Apply or prune any pending transitions based on a finality trigger. This
// method ensures that if there are multiple changes in the same branch,
// finalizing this block won't finalize past multiple transitions (i.e.
// transitions must be finalized in-order). The given function
// `is_descendent_of` should return `true` if the second hash (target) is a
// descendent of the first hash (base).
//
// When the set has changed, the return value will be `Ok(Some((H, N)))`
// which is the canonical block where the set last changed (i.e. the given
// hash and number).
func (authSet *AuthoritySet) ApplyStandardChanges(
	finalizedHash common.Hash,
	finalizedNumber uint,
	isDescendentOf IsDescendentOf,
	initialSync bool,
	telemetry *telemetry.Client) (Status, error) {
	// TODO telemetry here is just a place holder, replace with real

	status := Status{}
	finalizationResult, err := authSet.pendingStandardChanges.FinalizeWithDescendentIf(&finalizedHash, finalizedNumber, isDescendentOf, applyStandardChangesPredicate(finalizedNumber))
	if err != nil {
		return status, err
	}

	if finalizationResult == nil {
		return status, nil
	}

	// Changed Case
	status.changed = true

	// Flush pending forced changes to re add
	pendingForcedChanges := authSet.pendingForcedChanges
	authSet.pendingForcedChanges = []PendingChange{}

	// we will keep all forced changes for any later blocks and that are a
	// descendent of the finalized block (i.e. they are part of this branch).
	for i := 0; i < len(pendingForcedChanges); i++ {
		forcedChange := pendingForcedChanges[i]
		isDesc, err := isDescendentOf(finalizedHash, forcedChange.canonHash)
		if err != nil {
			return status, err
		}
		if forcedChange.EffectiveNumber() > finalizedNumber && isDesc {
			authSet.pendingForcedChanges = append(authSet.pendingForcedChanges, forcedChange)
		}
	}

	if finalizationResult.value != nil {
		// TODO add grandpa log

		// TODO add telemetry

		authoritySetChange := AuthorityChange{
			setId:       authSet.setId,
			blockNumber: finalizedNumber,
		}
		authSet.authoritySetChanges = append(authSet.authoritySetChanges, authoritySetChange)
		authSet.currentAuthorities = finalizationResult.value.nextAuthorities
		authSet.setId++

		status.newSetBlock = &newSetBlockInfo{
			newSetBlockNumber: finalizedNumber,
			newSetBlockHash:   finalizedHash,
		}
	}

	return status, nil
}

// EnactsStandardChange Check whether the given finalized block number enacts any standard
// authority set change (without triggering it), ensuring that if there are
// multiple changes in the same branch, finalizing this block won't
// finalize past multiple transitions (i.e. transitions must be finalized
// in-order). Returns `Some(true)` if the block being finalized enacts a
// change that can be immediately applied, `Some(false)` if the block being
// finalized enacts a change but it cannot be applied yet since there are
// other dependent changes, and `None` if no change is enacted. The given
// function `is_descendent_of` should return `true` if the second hash
// (target) is a descendent of the first hash (base).
func (authSet *AuthoritySet) EnactsStandardChange(
	finalizedHash common.Hash, finalizedNumber uint, isDescendentOf IsDescendentOf) (*bool, error) {
	return authSet.pendingStandardChanges.FinalizeAnyWithDescendentIf(&finalizedHash, finalizedNumber, isDescendentOf, enactStandardChangesPredicate(finalizedNumber))
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

// Copyright 2023 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package grandpa

import (
	"errors"
	"fmt"
	"sync"

	"golang.org/x/exp/constraints"
	"golang.org/x/exp/slices"
)

var (
	errInvalidAuthoritySet                           = errors.New("invalid authority set, either empty or with an authority weight set to 0")
	errDuplicateAuthoritySetChanges                  = errors.New("duplicate authority set hashNumber")
	errMultiplePendingForcedAuthoritySetChanges      = errors.New("multiple pending forced authority set changes are not allowed")
	errForcedAuthoritySetChangeDependencyUnsatisfied = errors.New("a pending forced authority set hashNumber could not be applied since it must be applied after the pending standard hashNumber")
	errForkTree                                      = errors.New("invalid operation in the pending hashNumber tree")
)

// SharedAuthoritySet A shared authority set
type SharedAuthoritySet[H comparable, N constraints.Unsigned] struct {
	mtx   sync.Mutex
	inner AuthoritySet[H, N]
}

// delayedKinds Kinds of delays for pending changes.
type delayedKinds[N constraints.Unsigned] interface {
	Finalized | Best[N]
}

// delayKind struct to represent delayedKinds
type delayKind struct {
	Value interface{}
}

func setDelayKind[N constraints.Unsigned, T delayedKinds[N]](delayKind *delayKind, val T) {
	delayKind.Value = val
}

func newDelayKind[N constraints.Unsigned, T delayedKinds[N]](val T) delayKind {
	delayKind := delayKind{}
	setDelayKind[N](&delayKind, val)
	return delayKind
}

// Finalized Depth in finalized chain.
type Finalized struct{}

// Best Depth in best chain. The median last finalized block is calculated at the time the
// hashNumber was signaled.
type Best[N constraints.Unsigned] struct {
	medianLastFinalized N
}

// IsDescendentOf is the function definition to determine if target is a descendant of base
type IsDescendentOf[H comparable] func(base, target H) (bool, error)

// setIDNumber represents the set id and block number of an authority set hashNumber
type setIDNumber[N constraints.Unsigned] struct {
	SetID       uint64
	BlockNumber N
}

// generic representation of hash and number tuple
type hashNumber[H, N any] struct {
	hash   H
	number N
}

// key is used to represent a tuple ordered first by effective number and then by signal-block number
type key[N any] struct {
	effectiveNumber   N
	signalBlockNumber N
}

// appliedChanges represents the median and new set when a forced hashNumber has occured
type appliedChanges[H comparable, N constraints.Unsigned] struct {
	median N
	set    AuthoritySet[H, N]
}

// invalidAuthorityList authority sets must be non-empty and all weights must be greater than 0
func (sas *SharedAuthoritySet[H, N]) invalidAuthorityList(authorities []Authority) bool {
	sas.mtx.Lock()
	defer sas.mtx.Unlock()
	return invalidAuthorityList(authorities)
}

// Current Get the current set id and a reference to the current authority set.
func (sas *SharedAuthoritySet[H, N]) Current() (uint64, *[]Authority) {
	sas.mtx.Lock()
	defer sas.mtx.Unlock()
	return sas.inner.current()
}

func (sas *SharedAuthoritySet[H, N]) revert() {
	sas.mtx.Lock()
	defer sas.mtx.Unlock()
	sas.inner.revert()
}

func (sas *SharedAuthoritySet[H, N]) nextChange(bestHash H, isDescendentOf IsDescendentOf[H]) (*hashNumber[H, N], error) {
	sas.mtx.Lock()
	defer sas.mtx.Unlock()
	return sas.inner.nextChange(bestHash, isDescendentOf)
}

func (sas *SharedAuthoritySet[H, N]) addStandardChange(pending PendingChange[H, N], isDescendentOf IsDescendentOf[H]) error {
	sas.mtx.Lock()
	defer sas.mtx.Unlock()
	return sas.inner.addStandardChange(pending, isDescendentOf)
}

func (sas *SharedAuthoritySet[H, N]) addForcedChange(pending PendingChange[H, N], isDescendentOf IsDescendentOf[H]) error {
	sas.mtx.Lock()
	defer sas.mtx.Unlock()
	return sas.inner.addForcedChange(pending, isDescendentOf)
}

func (sas *SharedAuthoritySet[H, N]) addPendingChange(pending PendingChange[H, N], isDescendentOf IsDescendentOf[H]) error {
	sas.mtx.Lock()
	defer sas.mtx.Unlock()
	return sas.inner.addPendingChange(pending, isDescendentOf)
}

// PendingChanges Inspect pending changes. Standard pending changes are iterated first,
// and the changes in the roots are traversed in pre-order, afterwards all
// forced changes are iterated.
func (sas *SharedAuthoritySet[H, N]) PendingChanges() []PendingChange[H, N] {
	sas.mtx.Lock()
	defer sas.mtx.Unlock()
	return sas.inner.pendingChanges()
}

// currentLimit Get the earliest limit-block number, if any. If there are pending changes across
// different forks, this method will return the earliest effective number (across the
// different branches) that is higher or equal to the given min number.
//
// Only standard changes are taken into account for the current
// limit, since any existing forced change should preclude the voter from voting.
func (sas *SharedAuthoritySet[H, N]) currentLimit(min N) (limit *N) {
	sas.mtx.Lock()
	defer sas.mtx.Unlock()
	return sas.inner.currentLimit(min)
}

func (sas *SharedAuthoritySet[H, N]) applyForcedChanges(bestHash H, bestNumber N, isDescendentOf IsDescendentOf[H], telemetry *Telemetry) (newSet *appliedChanges[H, N], err error) {
	sas.mtx.Lock()
	defer sas.mtx.Unlock()
	return sas.inner.applyForcedChanges(bestHash, bestNumber, isDescendentOf, telemetry)
}

// ApplyStandardChanges Apply or prune any pending transitions based on a finality trigger. This
// method ensures that if there are multiple changes in the same branch,
// finalizing this block won't finalize past multiple transitions (i.e.
// transitions must be finalized in-order). The given function
// `is_descendent_of` should return `true` if the second hash (target) is a
// descendent of the first hash (base).
//
// When the set has changed, the return value will be a Status type where newSetBlockInfo
// is the canonical block where the set last changed (i.e. the given
// hash and number).
func (sas *SharedAuthoritySet[H, N]) ApplyStandardChanges(finalizedHash H, finalizedNumber N, isDescendentOf IsDescendentOf[H], telemetry *Telemetry) (Status[H, N], error) {
	sas.mtx.Lock()
	defer sas.mtx.Unlock()
	return sas.inner.applyStandardChanges(finalizedHash, finalizedNumber, isDescendentOf, telemetry)
}

// EnactsStandardChange Check whether the given finalized block number enacts any standard
// authority set change (without triggering it), ensuring that if there are
// multiple changes in the same branch, finalizing this block won't
// finalize past multiple transitions (i.e. transitions must be finalized
// in-order). Returns *true if the block being finalized enacts a
// change that can be immediately applied, *false if the block being
// finalized enacts a change but it cannot be applied yet since there are
// other dependent changes, and nil if no change is enacted. The given
// function `is_descendent_of` should return `true` if the second hash
// (target) is a descendent of the first hash (base).
func (sas *SharedAuthoritySet[H, N]) EnactsStandardChange(finalizedHash H, finalizedNumber N, isDescendentOf IsDescendentOf[H]) (*bool, error) {
	sas.mtx.Lock()
	defer sas.mtx.Unlock()
	return sas.inner.EnactsStandardChange(finalizedHash, finalizedNumber, isDescendentOf)
}

// Status of the set after changes were applied.
type Status[H comparable, N constraints.Unsigned] struct {
	// Whether internal changes were made.
	Changed bool
	// Not nil when underlying authority set has changed, containing the
	// block where that set changed.
	NewSetBlock *hashNumber[H, N]
}

// AuthoritySet A set of authorities.
type AuthoritySet[H comparable, N constraints.Unsigned] struct {
	// The current active authorities.
	CurrentAuthorities []Authority
	// The current set id.
	SetID uint64
	// Tree of pending standard changes across forks. Standard changes are
	// enacted on finality and must be enacted (i.e. finalized) in-order across
	// a given branch
	PendingStandardChanges ForkTree[H, N]
	// Pending forced changes across different forks (at most one per fork).
	// Forced changes are enacted on block depth (not finality), for this
	// reason only one forced hashNumber should exist per fork. When trying to
	// apply forced changes we keep track of any pending standard changes that
	// they may depend on, this is done by making sure that any pending hashNumber
	// that is an ancestor of the forced changed and its effective block number
	// is lower than the last finalized block (as signaled in the forced
	// hashNumber) must be applied beforehand.
	PendingForcedChanges []PendingChange[H, N]
	// Track at which blocks the set id changed. This is useful when we need to prove finality for
	// a given block since we can figure out what set the block belongs to and when the set
	// started/ended.
	AuthoritySetChanges AuthoritySetChanges[N]
}

// invalidAuthorityList authority sets must be non-empty and all weights must be greater than 0
func invalidAuthorityList(authorities []Authority) bool {
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

// NewGenesisAuthoritySet Get a genesis set with given authorities.
func NewGenesisAuthoritySet[H comparable, N constraints.Unsigned](initial []Authority) (authSet *AuthoritySet[H, N]) {
	if invalidAuthorityList(initial) {
		return nil
	}

	return &AuthoritySet[H, N]{
		CurrentAuthorities:     initial,
		PendingStandardChanges: NewChangeTree[H, N](),
	}
}

// NewAuthoritySet creates a new AuthoritySet
func NewAuthoritySet[H comparable, N constraints.Unsigned](authorities []Authority,
	setID uint64,
	pendingStandardChanges ForkTree[H, N],
	pendingForcedChanges []PendingChange[H, N],
	authoritySetChanges AuthoritySetChanges[N],
) (authSet *AuthoritySet[H, N]) {
	if invalidAuthorityList(authorities) {
		return nil
	}

	return &AuthoritySet[H, N]{
		CurrentAuthorities:     authorities,
		SetID:                  setID,
		PendingStandardChanges: pendingStandardChanges,
		PendingForcedChanges:   pendingForcedChanges,
		AuthoritySetChanges:    authoritySetChanges,
	}
}

// current Get the current set id and a reference to the current authority set.
func (authSet *AuthoritySet[H, N]) current() (uint64, *[]Authority) {
	return authSet.SetID, &authSet.CurrentAuthorities
}

func (authSet *AuthoritySet[H, N]) revert() {
	panic("AuthoritySet.revert not implemented yet")
}

// Returns the block hash and height at which the next pending hashNumber in
// the given chain (i.e. it includes `best_hash`) was signalled, nil if
// there are no pending changes for the given chain.
func (authSet *AuthoritySet[H, N]) nextChange(bestHash H, isDescendentOf IsDescendentOf[H]) (*hashNumber[H, N], error) {
	var forced *hashNumber[H, N]
	for _, c := range authSet.PendingForcedChanges {
		isDesc, err := isDescendentOf(c.CanonHash, bestHash)
		if err != nil {
			return nil, err
		}
		if !isDesc {
			continue
		}
		forced = &hashNumber[H, N]{
			hash:   c.CanonHash,
			number: c.CanonHeight,
		}
		break
	}

	var standard *hashNumber[H, N]
	for _, changeNode := range authSet.PendingStandardChanges.Roots() {
		c := changeNode.Change
		isDesc, err := isDescendentOf(c.CanonHash, bestHash)
		if err != nil {
			return nil, err
		}
		if !isDesc {
			continue
		}
		standard = &hashNumber[H, N]{
			hash:   c.CanonHash,
			number: c.CanonHeight,
		}
		break
	}

	switch {
	case standard != nil && forced != nil:
		if forced.number < standard.number {
			return forced, nil
		}
		return standard, nil
	case forced != nil:
		return forced, nil
	case standard != nil:
		return standard, nil
	default:
		return nil, nil
	}
}

func (authSet *AuthoritySet[H, N]) addStandardChange(pending PendingChange[H, N], isDescendentOf IsDescendentOf[H]) error {
	hash := pending.CanonHash
	number := pending.CanonHeight

	logger.Debugf(
		"inserting potential standard set hashNumber signaled at block %d (delayed by %d blocks).",
		number, pending.Delay,
	)

	_, err := authSet.PendingStandardChanges.Import(hash, number, pending, isDescendentOf)
	if err != nil {
		return err
	}

	logger.Debugf(
		"There are now %d alternatives for the next pending standard hashNumber (roots), "+
			"and a total of %d pending standard changes (across all forks)",
		len(authSet.PendingStandardChanges.Roots()), len(authSet.PendingStandardChanges.PendingChanges()),
	)

	return nil
}

func (authSet *AuthoritySet[H, N]) addForcedChange(pending PendingChange[H, N], isDescendentOf IsDescendentOf[H]) error {
	for _, change := range authSet.PendingForcedChanges {
		if change.CanonHash == pending.CanonHash {
			return errDuplicateAuthoritySetChanges
		}

		isDescendent, err := isDescendentOf(change.CanonHash, pending.CanonHash)
		if err != nil {
			return fmt.Errorf("addForcedChange: checking isDescendentOf: %w", err)
		}

		if isDescendent {
			return errMultiplePendingForcedAuthoritySetChanges
		}
	}

	k := key[N]{
		pending.EffectiveNumber(),
		pending.CanonHeight,
	}

	// Search by effective key
	idx, _ := slices.BinarySearchFunc(
		authSet.PendingForcedChanges,
		k,
		func(change PendingChange[H, N], k key[N]) int {
			switch {
			case change.EffectiveNumber() == k.effectiveNumber && change.CanonHeight == k.signalBlockNumber:
				return 0
			case change.EffectiveNumber() > k.effectiveNumber && change.CanonHeight > k.signalBlockNumber:
				return 1
			case change.EffectiveNumber() < k.effectiveNumber && change.CanonHeight < k.signalBlockNumber:
				return -1
			default:
				panic("huh?")
			}
		},
	)

	logger.Debugf(
		"inserting potential forced set hashNumber at block number %d (delayed by %d blocks).",
		pending.CanonHeight, pending.Delay,
	)

	// Insert hashNumber at index
	if len(authSet.PendingForcedChanges) == idx {
		authSet.PendingForcedChanges = append(authSet.PendingForcedChanges, pending)
	} else if len(authSet.PendingForcedChanges) > idx {
		authSet.PendingForcedChanges = append(
			authSet.PendingForcedChanges[:idx+1], authSet.PendingForcedChanges[idx:]...)
		authSet.PendingForcedChanges[idx] = pending
	} else {
		panic("invalid insertion into pending forced changes")
	}

	logger.Debugf(
		"there are now %d pending forced changes",
		len(authSet.PendingForcedChanges),
	)

	return nil
}

// addPendingChange Note an upcoming pending transition. Multiple pending standard changes
// on the same branch can be added as long as they don't overlap. Forced
// changes are restricted to one per fork. This method assumes that changes
// on the same branch will be added in-order. The given function
// `is_descendent_of` should return `true` if the second hash (target) is a
// descendent of the first hash (base).
func (authSet *AuthoritySet[H, N]) addPendingChange(pending PendingChange[H, N], isDescendentOf IsDescendentOf[H]) error {
	if invalidAuthorityList(pending.NextAuthorities) {
		return errInvalidAuthoritySet
	}

	switch pending.DelayKind.Value.(type) {
	case Finalized:
		return authSet.addStandardChange(pending, isDescendentOf)
	case Best[N]:
		return authSet.addForcedChange(pending, isDescendentOf)
	default:
		panic("DelayKind is invalid type")
	}
}

// pendingChanges Inspect pending changes. Standard pending changes are iterated first,
// and the changes in the roots are traversed in pre-order, afterwards all
// forced changes are iterated.
func (authSet *AuthoritySet[H, N]) pendingChanges() []PendingChange[H, N] {
	// get everything from standard hashNumber roots
	changes := authSet.PendingStandardChanges.PendingChanges()

	// append forced changes
	changes = append(changes, authSet.PendingForcedChanges...)

	return changes
}

// currentLimit Get the earliest limit-block number, if any. If there are pending changes across
// different forks, this method will return the earliest effective number (across the
// different branches) that is higher or equal to the given min number.
//
// Only standard changes are taken into account for the current
// limit, since any existing forced hashNumber should preclude the voter from voting.
func (authSet *AuthoritySet[H, N]) currentLimit(min N) (limit *N) {
	roots := authSet.PendingStandardChanges.Roots()
	for i := 0; i < len(roots); i++ {
		effectiveNumber := roots[i].Change.EffectiveNumber()
		if effectiveNumber >= min {
			if limit == nil {
				limit = &effectiveNumber
			} else if effectiveNumber < *limit {
				limit = &effectiveNumber
			}
		}
	}
	return limit
}

// ApplyForcedChanges Apply or prune any pending transitions based on a best-block trigger.
//
// Returns a pointer to the median and new_set when a forced hashNumber has occurred. The
// median represents the median last finalized block at the time the hashNumber
// was signaled, and it should be used as the canon block when starting the
// new grandpa voter. Only alters the internal state in this case.
//
// These transitions are always forced and do not lead to justifications
// which light clients can follow.
//
// Forced changes can only be applied after all pending standard changes
// that it depends on have been applied. If any pending standard hashNumber
// exists that is an ancestor of a given forced changed and which effective
// block number is lower than the last finalized block (as defined by the
// forced hashNumber), then the forced hashNumber cannot be applied. An error will
// be returned in that case which will prevent block import.
func (authSet *AuthoritySet[H, N]) applyForcedChanges(bestHash H,
	bestNumber N,
	isDescendentOf IsDescendentOf[H],
	_ Telemetry) (newSet *appliedChanges[H, N], err error) {

	for _, change := range authSet.PendingForcedChanges {
		effectiveNumber := change.EffectiveNumber()
		if effectiveNumber > bestNumber {
			continue
		} else if effectiveNumber == bestNumber {
			// check if the given best block is in the same branch as
			// the block that signaled the hashNumber.
			isDesc, err := isDescendentOf(change.CanonHash, bestHash)
			// Avoid case where err is returned because canonHash == bestHash
			if change.CanonHash != bestHash && err != nil {
				return nil, err
			}
			if change.CanonHash == bestHash || isDesc {
				switch delayKindType := change.DelayKind.Value.(type) {
				case Best[N]:
					medianLastFinalized := delayKindType.medianLastFinalized
					roots := authSet.PendingStandardChanges.Roots()
					for _, standardChangeNode := range roots {
						standardChange := standardChangeNode.Change

						isDescStandard, err := isDescendentOf(standardChange.CanonHash, change.CanonHash)
						if err != nil {
							return nil, err
						}
						if standardChange.EffectiveNumber() <= medianLastFinalized && isDescStandard {
							logger.Infof(
								"Not applying authority set hashNumber forced at block %d, due to pending standard hashNumber at block %d",
								change.CanonHeight, standardChange.EffectiveNumber())
							return nil, errForcedAuthoritySetChangeDependencyUnsatisfied
						}
					}

					// apply this hashNumber: make the set canonical
					logger.Infof("👴 Applying authority set hashNumber forced at block #%d", change.CanonHeight)

					// TODO telemetry

					authSetChanges := authSet.AuthoritySetChanges
					authSetChanges.append(authSet.SetID, medianLastFinalized)
					newSet = &appliedChanges[H, N]{
						medianLastFinalized,
						AuthoritySet[H, N]{
							CurrentAuthorities:     change.NextAuthorities,
							SetID:                  authSet.SetID + 1,
							PendingStandardChanges: NewChangeTree[H, N](), // new set, new changes
							PendingForcedChanges:   []PendingChange[H, N]{},
							AuthoritySetChanges:    authSetChanges,
						},
					}
					return newSet, nil
				default:
					panic("pending_forced_changes only contains forced changes; forced changes have delay kind Best; qed.")
				}
			}
		}
	}

	return newSet, nil
}

// applyStandardChanges Apply or prune any pending transitions based on a finality trigger. This
// method ensures that if there are multiple changes in the same branch,
// finalizing this block won't finalize past multiple transitions (i.e.
// transitions must be finalized in-order). The given function
// `is_descendent_of` should return `true` if the second hash (target) is a
// descendent of the first hash (base).
//
// When the set has changed, the return value will be a Status type where newSetBlock
// is the canonical block where the set last changed (i.e. the given
// hash and number).
func (authSet *AuthoritySet[H, N]) applyStandardChanges(
	finalizedHash H,
	finalizedNumber N,
	isDescendentOf IsDescendentOf[H],
	_ Telemetry) (Status[H, N], error) {
	// TODO telemetry here is just a place holder, replace with real

	status := Status[H, N]{}
	finalizationResult, err := authSet.PendingStandardChanges.FinalizeWithDescendentIf(&finalizedHash, finalizedNumber, isDescendentOf, func(change *PendingChange[H, N]) bool {
		return change.EffectiveNumber() <= finalizedNumber
	})
	if err != nil {
		return status, err
	}

	if finalizationResult == nil {
		return status, nil
	}

	// Changed Case
	status.Changed = true

	// Flush pending forced changes to re add
	pendingForcedChanges := authSet.PendingForcedChanges
	authSet.PendingForcedChanges = []PendingChange[H, N]{}

	// we will keep all forced changes for any later blocks and that are a
	// descendent of the finalized block (i.e. they are part of this branch).
	for _, forcedChange := range pendingForcedChanges {
		isDesc, err := isDescendentOf(finalizedHash, forcedChange.CanonHash)
		if err != nil {
			return status, err
		}
		if forcedChange.EffectiveNumber() > finalizedNumber && isDesc {
			authSet.PendingForcedChanges = append(authSet.PendingForcedChanges, forcedChange)
		}
	}

	if finalizationResult.Value != nil {
		logger.Infof("👴 Applying authority set hashNumber forced at block #%d", *finalizationResult.Value)

		// TODO add telemetry

		// Store the set_id together with the last block_number for the set
		authSet.AuthoritySetChanges.append(authSet.SetID, finalizedNumber)
		authSet.CurrentAuthorities = finalizationResult.Value.NextAuthorities
		authSet.SetID++

		status.NewSetBlock = &hashNumber[H, N]{
			hash:   finalizedHash,
			number: finalizedNumber,
		}
	}

	return status, nil
}

// EnactsStandardChange Check whether the given finalized block number enacts any standard
// authority set hashNumber (without triggering it), ensuring that if there are
// multiple changes in the same branch, finalizing this block won't
// finalize past multiple transitions (i.e. transitions must be finalized
// in-order). Returns *true if the block being finalized enacts a
// hashNumber that can be immediately applied, *false if the block being
// finalized enacts a hashNumber but it cannot be applied yet since there are
// other dependent changes, and nil if no hashNumber is enacted. The given
// function `is_descendent_of` should return `true` if the second hash
// (target) is a descendent of the first hash (base).
func (authSet *AuthoritySet[H, N]) EnactsStandardChange(
	finalizedHash H, finalizedNumber N, isDescendentOf IsDescendentOf[H]) (*bool, error) {
	applied, err := authSet.PendingStandardChanges.FinalizesAnyWithDescendentIf(&finalizedHash, finalizedNumber, isDescendentOf, func(change *PendingChange[H, N]) bool {
		return change.EffectiveNumber() == finalizedNumber
	})
	if err != nil {
		return nil, fmt.Errorf("%w: %v", errForkTree, err)
	}
	return applied, nil
}

// PendingChange A pending hashNumber to the authority set.
//
// This will be applied when the announcing block is at some depth within
// the finalized or unfinalized chain.
type PendingChange[H comparable, N constraints.Unsigned] struct {
	// The new authorities and weights to apply.
	NextAuthorities []Authority
	// How deep in the chain the announcing block must be
	// before the hashNumber is applied.
	Delay N
	// The announcing block's height.
	CanonHeight N
	// The announcing block's hash.
	CanonHash H
	// The Delay kind.
	DelayKind delayKind
}

// EffectiveNumber Returns the effective number this hashNumber will be applied at.
func (pc *PendingChange[H, N]) EffectiveNumber() N {
	return pc.CanonHeight + pc.Delay
}

// AuthoritySetChanges Tracks historical authority set changes. We store the block numbers for the last block
// of each authority set, once they have been finalized. These blocks are guaranteed to
// have a justification unless they were triggered by a forced hashNumber.
type AuthoritySetChanges[N constraints.Unsigned] []setIDNumber[N]

// append an setIDNumber to AuthoritySetChanges
func (asc *AuthoritySetChanges[N]) append(setID uint64, blockNumber N) {
	*asc = append(*asc, setIDNumber[N]{
		SetID:       setID,
		BlockNumber: blockNumber,
	})
}

// Three states that can be returned: Latest, Set (tuple), Unknown
// Latest => bool
// Set => &AuthorityChange
// Unknown => nil
// TODO for reviewers, this can be a VDT but I'm not sure its needed
func (asc *AuthoritySetChanges[N]) getSetID(blockNumber N) (latest bool, set *setIDNumber[N], err error) {
	if asc == nil {
		return false, nil, fmt.Errorf("getSetID: authSetChanges is nil")
	}
	authSet := *asc
	last := authSet[len(authSet)-1]
	if last.BlockNumber < blockNumber {
		return true, nil, nil // Latest case
	}

	idx, _ := slices.BinarySearchFunc(
		authSet,
		blockNumber,
		func(a setIDNumber[N], b N) int {
			switch {
			case a.BlockNumber == b:
				return 0
			case a.BlockNumber > b:
				return 1
			case a.BlockNumber < b:
				return -1
			default:
				panic("huh?")
			}
		},
	)
	if idx < len(authSet) {
		authChange := authSet[idx]

		// if this is the first index but not the first set id then we are missing data.
		if idx == 0 && authChange.SetID != 0 {
			return false, nil, nil // Unknown case
		}

		return false, &authChange, nil // Set case
	}

	return false, nil, nil // Unknown case
}

func (asc *AuthoritySetChanges[N]) insert(blockNumber N) {
	var idx int
	if asc == nil {
		panic("authority set changes must be initialized")
	} else {
		idx, _ = slices.BinarySearchFunc(
			*asc,
			blockNumber,
			func(a setIDNumber[N], b N) int {
				switch {
				case a.BlockNumber == b:
					return 0
				case a.BlockNumber > b:
					return 1
				case a.BlockNumber < b:
					return -1
				default:
					panic("huh?")
				}
			},
		)
	}

	set := *asc

	var setID uint64
	if idx == 0 {
		setID = 0
	} else {
		setID = set[idx-1].SetID + 1
	}

	if idx != len(set) && set[idx].SetID == setID {
		panic("inserting authority set hashNumber")
	}

	change := setIDNumber[N]{
		SetID:       setID,
		BlockNumber: blockNumber,
	}

	// Insert hashNumber at index
	if len(set) <= idx {
		set = append(set, change)
	} else {
		set = append(set[:idx+1], set[idx:]...)
		set[idx] = change
	}
	*asc = set
}

// IterFrom This logic is used in warp sync proof
func (asc *AuthoritySetChanges[N]) IterFrom(blockNumber N) *AuthoritySetChanges[N] {
	if asc == nil {
		return nil
	}
	authSet := *asc

	idx, found := slices.BinarySearchFunc(
		*asc,
		blockNumber,
		func(a setIDNumber[N], b N) int {
			switch {
			case a.BlockNumber == b:
				return 0
			case a.BlockNumber > b:
				return 1
			case a.BlockNumber < b:
				return -1
			default:
				panic("huh?")
			}
		},
	)
	if found {
		// if there was a hashNumber at the given block number then we should start on the next
		// index since we want to exclude the current block number
		idx += 1
	}

	if idx < len(*asc) {
		authChange := authSet[idx]

		// if this is the first index but not the first set id then we are missing data.
		if idx == 0 && authChange.SetID != 0 {
			return nil
		}
	}

	iterChanges := authSet[idx:]
	return &iterChanges
}

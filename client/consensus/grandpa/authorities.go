// Copyright 2023 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package grandpa

import (
	"errors"
	"fmt"
	"sync"

	"github.com/ChainSafe/gossamer/pkg/scale"
	"golang.org/x/exp/constraints"
	"golang.org/x/exp/slices"
)

var (
	errInvalidAuthoritySet = errors.New("invalid authority set, either empty or with" +
		" an authority weight set to 0")
	errDuplicateAuthoritySetChanges             = errors.New("duplicate authority set hashNumber")
	errMultiplePendingForcedAuthoritySetChanges = errors.New("multiple pending forced authority set " +
		"changes are not allowed")
	errForcedAuthoritySetChangeDependencyUnsatisfied = errors.New("a pending forced authority set hashNumber " +
		"could not be applied since it must be applied after the pending standard hashNumber")
	errForkTree             = errors.New("invalid operation in the pending hashNumber tree")
	errInvalidAuthorityList = errors.New("invalid authority list")
)

// SharedAuthoritySet A shared authority set
type SharedAuthoritySet[H comparable, N constraints.Unsigned, ID AuthorityID] struct {
	mtx   sync.Mutex
	inner AuthoritySet[H, N, ID]
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

// medianAuthoritySet represents the median and new set when a forced hashNumber has occurred
type medianAuthoritySet[H comparable, N constraints.Unsigned, ID AuthorityID] struct {
	median N
	set    AuthoritySet[H, N, ID]
}

// Current Get the current set id and a reference to the current authority set.
func (sas *SharedAuthoritySet[H, N, ID]) Current() (uint64, *[]Authority[ID]) {
	sas.mtx.Lock()
	defer sas.mtx.Unlock()
	return sas.inner.current()
}

func (sas *SharedAuthoritySet[H, N, ID]) revert() { //nolint //skipcq: SCC-U1000
	sas.mtx.Lock()
	defer sas.mtx.Unlock()
	sas.inner.revert()
}

func (sas *SharedAuthoritySet[H, N, ID]) nextChange(bestHash H, //nolint //skipcq: SCC-U1000
	isDescendentOf IsDescendentOf[H]) (*hashNumber[H, N], error) {
	sas.mtx.Lock()
	defer sas.mtx.Unlock()
	return sas.inner.nextChange(bestHash, isDescendentOf)
}

func (sas *SharedAuthoritySet[H, N, ID]) addStandardChange(pending PendingChange[H, N, ID], //nolint //skipcq: SCC-U1000
	isDescendentOf IsDescendentOf[H]) error {
	sas.mtx.Lock()
	defer sas.mtx.Unlock()
	return sas.inner.addStandardChange(pending, isDescendentOf)
}

func (sas *SharedAuthoritySet[H, N, ID]) addForcedChange(pending PendingChange[H, N, ID], //nolint //skipcq: SCC-U1000
	isDescendentOf IsDescendentOf[H]) error {
	sas.mtx.Lock()
	defer sas.mtx.Unlock()
	return sas.inner.addForcedChange(pending, isDescendentOf)
}

func (sas *SharedAuthoritySet[H, N, ID]) addPendingChange(pending PendingChange[H, N, ID], //nolint //skipcq: SCC-U1000
	isDescendentOf IsDescendentOf[H]) error {
	sas.mtx.Lock()
	defer sas.mtx.Unlock()
	return sas.inner.addPendingChange(pending, isDescendentOf)
}

// PendingChanges Inspect pending changes. Standard pending changes are iterated first,
// and the changes in the roots are traversed in pre-order, afterwards all
// forced changes are iterated.
func (sas *SharedAuthoritySet[H, N, ID]) PendingChanges() []PendingChange[H, N, ID] {
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
func (sas *SharedAuthoritySet[H, N, ID]) currentLimit(min N) (limit *N) { //nolint //skipcq: SCC-U1000
	sas.mtx.Lock()
	defer sas.mtx.Unlock()
	return sas.inner.currentLimit(min)
}

func (sas *SharedAuthoritySet[H, N, ID]) applyForcedChanges(bestHash H, //nolint //skipcq: SCC-U1000
	bestNumber N,
	isDescendentOf IsDescendentOf[H],
	telemetry *Telemetry) (newSet *medianAuthoritySet[H, N, ID], err error) {
	sas.mtx.Lock()
	defer sas.mtx.Unlock()
	return sas.inner.applyForcedChanges(bestHash, bestNumber, isDescendentOf, telemetry)
}

// applyStandardChanges Apply or prune any pending transitions based on a finality trigger. This
// method ensures that if there are multiple changes in the same branch,
// finalising this block won't finalise past multiple transitions (i.e.
// transitions must be finalised in-order). The given function
// `is_descendent_of` should return `true` if the second hash (target) is a
// descendent of the first hash (base).
//
// When the set has changed, the return value will be a status type where newSetBlockInfo
// is the canonical block where the set last changed (i.e. the given
// hash and number).
func (sas *SharedAuthoritySet[H, N, ID]) applyStandardChanges(finalisedHash H, //nolint //skipcq: SCC-U1000
	finalisedNumber N,
	isDescendentOf IsDescendentOf[H],
	telemetry *Telemetry) (status[H, N], error) {
	sas.mtx.Lock()
	defer sas.mtx.Unlock()
	return sas.inner.applyStandardChanges(finalisedHash, finalisedNumber, isDescendentOf, telemetry)
}

// EnactsStandardChange Check whether the given finalised block number enacts any standard
// authority set change (without triggering it), ensuring that if there are
// multiple changes in the same branch, finalising this block won't
// finalise past multiple transitions (i.e. transitions must be finalised
// in-order). Returns *true if the block being finalised enacts a
// change that can be immediately applied, *false if the block being
// finalised enacts a change but it cannot be applied yet since there are
// other dependent changes, and nil if no change is enacted. The given
// function `is_descendent_of` should return `true` if the second hash
// (target) is a descendent of the first hash (base).
func (sas *SharedAuthoritySet[H, N, ID]) EnactsStandardChange(finalisedHash H,
	finalisedNumber N,
	isDescendentOf IsDescendentOf[H]) (*bool, error) {
	sas.mtx.Lock()
	defer sas.mtx.Unlock()
	return sas.inner.EnactsStandardChange(finalisedHash, finalisedNumber, isDescendentOf)
}

// status of the set after changes were applied.
type status[H comparable, N constraints.Unsigned] struct {
	// Whether internal changes were made.
	Changed bool
	// Not nil when underlying authority set has changed, containing the
	// block where that set changed.
	NewSetBlock *hashNumber[H, N]
}

// AuthoritySet A set of authorities.
type AuthoritySet[H comparable, N constraints.Unsigned, ID AuthorityID] struct {
	// The current active authorities.
	CurrentAuthorities []Authority[ID]
	// The current set id.
	SetID uint64
	// Tree of pending standard changes across forks. Standard changes are
	// enacted on finality and must be enacted (i.e. finalised) in-order across
	// a given branch
	PendingStandardChanges ChangeTree[H, N, ID]
	// Pending forced changes across different forks (at most one per fork).
	// Forced changes are enacted on block depth (not finality), for this
	// reason only one forced hashNumber should exist per fork. When trying to
	// apply forced changes we keep track of any pending standard changes that
	// they may depend on, this is done by making sure that any pending hashNumber
	// that is an ancestor of the forced changed and its effective block number
	// is lower than the last finalised block (as signalled in the forced
	// hashNumber) must be applied beforehand.
	PendingForcedChanges []PendingChange[H, N, ID]
	// Track at which blocks the set id changed. This is useful when we need to prove finality for
	// a given block since we can figure out what set the block belongs to and when the set
	// started/ended.
	AuthoritySetChanges AuthoritySetChanges[N]
}

// invalidAuthorityList authority sets must be non-empty and all weights must be greater than 0
func invalidAuthorityList[ID AuthorityID](authorities []Authority[ID]) bool { //skipcq:  RVV-B0001
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
func NewGenesisAuthoritySet[H comparable, N constraints.Unsigned, ID AuthorityID](initial []Authority[ID]) (authSet *AuthoritySet[H, N, ID], err error) {
	if invalidAuthorityList(initial) {
		return nil, errInvalidAuthorityList
	}

	return &AuthoritySet[H, N, ID]{
		CurrentAuthorities: initial,
	}, nil
}

// NewAuthoritySet creates a new AuthoritySet
func NewAuthoritySet[H comparable, N constraints.Unsigned, ID AuthorityID](authorities []Authority[ID],
	setID uint64,
	pendingStandardChanges ChangeTree[H, N, ID],
	pendingForcedChanges []PendingChange[H, N, ID],
	authoritySetChanges AuthoritySetChanges[N],
) (authSet *AuthoritySet[H, N, ID], err error) {
	if invalidAuthorityList(authorities) {
		return nil, errInvalidAuthorityList
	}

	return &AuthoritySet[H, N, ID]{
		CurrentAuthorities:     authorities,
		SetID:                  setID,
		PendingStandardChanges: pendingStandardChanges,
		PendingForcedChanges:   pendingForcedChanges,
		AuthoritySetChanges:    authoritySetChanges,
	}, nil
}

// current Get the current set id and a reference to the current authority set.
func (authSet *AuthoritySet[H, N, ID]) current() (uint64, *[]Authority[ID]) {
	return authSet.SetID, &authSet.CurrentAuthorities
}

// Revert to a specified block given its `hash` and `number`.
// This removes all the authority set changes that were announced after
// the revert point.
// Revert point is identified by `number` and `hash`.
func (authSet *AuthoritySet[H, N, ID]) revert() { //nolint //skipcq: SCC-U1000 //skipcq:  RVV-B0001
	panic("AuthoritySet.revert not implemented yet")
}

// Returns the block hash and height at which the next pending hashNumber in
// the given chain (i.e. it includes `best_hash`) was signalled, nil if
// there are no pending changes for the given chain.
func (authSet *AuthoritySet[H, N, ID]) nextChange(bestHash H, //skipcq:  RVV-B0001
	isDescendentOf IsDescendentOf[H]) (*hashNumber[H, N], error) {
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

func (authSet *AuthoritySet[H, N, ID]) addStandardChange(pending PendingChange[H, N, ID], isDescendentOf IsDescendentOf[H]) error {
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

func (pc PendingChange[H, N, ID]) GreaterThan(other PendingChange[H, N, ID]) bool {
	effectiveNumberGreaterThan := pc.EffectiveNumber() > other.EffectiveNumber()
	cannonHeighGreaterThan := pc.EffectiveNumber() == other.EffectiveNumber() &&
		pc.CanonHeight > other.CanonHeight

	return effectiveNumberGreaterThan || cannonHeighGreaterThan
}

func (pc PendingChange[H, N, ID]) LessThan(other PendingChange[H, N, ID]) bool {
	effectiveNumberLessThan := pc.EffectiveNumber() < other.EffectiveNumber()
	cannonHeighLessThan := pc.EffectiveNumber() == other.EffectiveNumber() &&
		pc.CanonHeight < other.CanonHeight

	return effectiveNumberLessThan || cannonHeighLessThan
}

func (authSet *AuthoritySet[H, N, ID]) addForcedChange(pending PendingChange[H, N, ID], isDescendentOf IsDescendentOf[H]) error {
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

	// Changes are inserted in ascending order
	idx, _ := slices.BinarySearchFunc(
		authSet.PendingForcedChanges,
		pending,
		func(change, toInsert PendingChange[H, N, ID]) int {
			switch {
			case toInsert.LessThan(change):
				return 1
			case toInsert.GreaterThan(change):
				return -1
			default:
				return 0
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
func (authSet *AuthoritySet[H, N, ID]) addPendingChange(pending PendingChange[H, N, ID], isDescendentOf IsDescendentOf[H]) error {
	if invalidAuthorityList[ID](pending.NextAuthorities) {
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
func (authSet *AuthoritySet[H, N, ID]) pendingChanges() []PendingChange[H, N, ID] { //skipcq:  RVV-B0001
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
func (authSet *AuthoritySet[H, N, ID]) currentLimit(min N) (limit *N) {
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
// median represents the median last finalised block at the time the hashNumber
// was signalled, and it should be used as the canon block when starting the
// new grandpa voter. Only alters the internal state in this case.
//
// These transitions are always forced and do not lead to justifications
// which light clients can follow.
//
// Forced changes can only be applied after all pending standard changes
// that it depends on have been applied. If any pending standard hashNumber
// exists that is an ancestor of a given forced changed and which effective
// block number is lower than the last finalised block (as defined by the
// forced hashNumber), then the forced hashNumber cannot be applied. An error will
// be returned in that case which will prevent block import.
func (authSet *AuthoritySet[H, N, ID]) applyForcedChanges(bestHash H, //skipcq:  RVV-B0001
	bestNumber N,
	isDescendentOf IsDescendentOf[H],
	_ Telemetry) (newSet *medianAuthoritySet[H, N, ID], err error) {

	for _, change := range authSet.PendingForcedChanges {
		effectiveNumber := change.EffectiveNumber()
		if effectiveNumber != bestNumber {
			continue
		}
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
				newSet = &medianAuthoritySet[H, N, ID]{
					medianLastFinalized,
					AuthoritySet[H, N, ID]{
						CurrentAuthorities:     change.NextAuthorities,
						SetID:                  authSet.SetID + 1,
						PendingStandardChanges: NewChangeTree[H, N, ID](), // new set, new changes
						PendingForcedChanges:   []PendingChange[H, N, ID]{},
						AuthoritySetChanges:    authSetChanges,
					},
				}
				return newSet, nil
			default:
				panic("pending_forced_changes only contains forced changes; forced changes have delay kind Best")
			}
		}
	}

	return newSet, nil
}

// applyStandardChanges Apply or prune any pending transitions based on a finality trigger. This
// method ensures that if there are multiple changes in the same branch,
// finalising this block won't finalise past multiple transitions (i.e.
// transitions must be finalised in-order). The given function
// `is_descendent_of` should return `true` if the second hash (target) is a
// descendent of the first hash (base).
//
// When the set has changed, the return value will be a status type where newSetBlock
// is the canonical block where the set last changed (i.e. the given
// hash and number).
func (authSet *AuthoritySet[H, N, ID]) applyStandardChanges( //skipcq:  RVV-B0001
	finalisedHash H,
	finalisedNumber N,
	isDescendentOf IsDescendentOf[H],
	_ Telemetry) (status[H, N], error) {
	// TODO telemetry here is just a place holder, replace with real

	status := status[H, N]{}
	finalisationResult, err := authSet.PendingStandardChanges.FinalizeWithDescendentIf(&finalisedHash,
		finalisedNumber,
		isDescendentOf,
		func(change *PendingChange[H, N, ID]) bool {
			return change.EffectiveNumber() <= finalisedNumber
		})
	if err != nil {
		return status, err
	}

	finalisationResultVal, err := finalisationResult.Value()
	if err != nil {
		return status, err
	}
	switch val := finalisationResultVal.(type) {
	case unchanged:
		return status, nil
	case changed[H, N, ID]:
		// Changed Case
		status.Changed = true

		// Flush pending forced changes to re add
		pendingForcedChanges := authSet.PendingForcedChanges
		authSet.PendingForcedChanges = []PendingChange[H, N, ID]{}

		// we will keep all forced changes for any later blocks and that are a
		// descendent of the finalised block (i.e. they are part of this branch).
		for _, forcedChange := range pendingForcedChanges {
			isDesc, err := isDescendentOf(finalisedHash, forcedChange.CanonHash)
			if err != nil {
				return status, err
			}
			if forcedChange.EffectiveNumber() > finalisedNumber && isDesc {
				authSet.PendingForcedChanges = append(authSet.PendingForcedChanges, forcedChange)
			}
		}

		if val.value != nil {
			logger.Infof("👴 Applying authority set hashNumber forced at block #%d", *val.value)

			// TODO add telemetry

			// Store the set_id together with the last block_number for the set
			authSet.AuthoritySetChanges.append(authSet.SetID, finalisedNumber)
			authSet.CurrentAuthorities = val.value.NextAuthorities
			authSet.SetID++

			status.NewSetBlock = &hashNumber[H, N]{
				hash:   finalisedHash,
				number: finalisedNumber,
			}
		}

		return status, nil
	default:
		panic("invalid type for FinalizationResult")
	}
}

// EnactsStandardChange Check whether the given finalised block number enacts any standard
// authority set hashNumber (without triggering it), ensuring that if there are
// multiple changes in the same branch, finalising this block won't
// finalise past multiple transitions (i.e. transitions must be finalised
// in-order). Returns *true if the block being finalised enacts a
// hashNumber that can be immediately applied, *false if the block being
// finalised enacts a hashNumber but it cannot be applied yet since there are
// other dependent changes, and nil if no hashNumber is enacted. The given
// function `is_descendent_of` should return `true` if the second hash
// (target) is a descendent of the first hash (base).
func (authSet *AuthoritySet[H, N, ID]) EnactsStandardChange( //skipcq:  RVV-B0001
	finalisedHash H, finalisedNumber N, isDescendentOf IsDescendentOf[H]) (*bool, error) {
	applied, err := authSet.PendingStandardChanges.FinalizesAnyWithDescendentIf(&finalisedHash,
		finalisedNumber,
		isDescendentOf,
		func(change *PendingChange[H, N, ID]) bool {
			return change.EffectiveNumber() == finalisedNumber
		})
	if err != nil {
		return nil, fmt.Errorf("%w: %v", errForkTree, err)
	}
	return applied, nil
}

// delayKinds Kinds of delays for pending changes.
type delayKinds[N constraints.Unsigned] interface {
	Finalized | Best[N]
}

// delayKind struct to represent delayedKinds
type delayKind struct {
	Value interface{}
}

func newDelayKind[N constraints.Unsigned, T delayKinds[N]](val T) delayKind {
	return delayKind{
		Value: val,
	}
}

// Finalized Depth in finalised chain.
type Finalized struct{}

// Best Depth in best chain. The median last finalised block is calculated at the time the
// hashNumber was signalled.
type Best[N constraints.Unsigned] struct {
	medianLastFinalized N
}

// PendingChange A pending hashNumber to the authority set.
//
// This will be applied when the announcing block is at some depth within
// the finalised or unfinalised chain.
type PendingChange[H comparable, N constraints.Unsigned, ID AuthorityID] struct {
	// The new authorities and weights to apply.
	NextAuthorities []Authority[ID]
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
func (pc *PendingChange[H, N, ID]) EffectiveNumber() N {
	return pc.CanonHeight + pc.Delay
}

// AuthoritySetChanges Tracks historical authority set changes. We store the block numbers for the last block
// of each authority set, once they have been finalised. These blocks are guaranteed to
// have a justification unless they were triggered by a forced hashNumber.
type AuthoritySetChanges[N constraints.Unsigned] []setIDNumber[N]

// append an setIDNumber to AuthoritySetChanges
func (asc *AuthoritySetChanges[N]) append(setID uint64, blockNumber N) {
	*asc = append(*asc, setIDNumber[N]{
		SetID:       setID,
		BlockNumber: blockNumber,
	})
}

type authoritySetChangeID scale.VaryingDataType

// Set will set a VaryingDataTypeValue using the underlying VaryingDataType
func (asc *authoritySetChangeID) Set(val scale.VaryingDataTypeValue) (err error) {
	vdt := scale.VaryingDataType(*asc)
	err = vdt.Set(val)
	if err != nil {
		return
	}
	*asc = authoritySetChangeID(vdt)
	return
}

// Value will return value from underying VaryingDataType
func (asc *authoritySetChangeID) Value() (val scale.VaryingDataTypeValue, err error) {
	vdt := scale.VaryingDataType(*asc)
	return vdt.Value()
}

func newAuthoritySetChangeID[N constraints.Unsigned]() authoritySetChangeID {
	vdt := scale.MustNewVaryingDataType(latest{}, set[N]{}, unknown{})
	return authoritySetChangeID(vdt)
}

type latest struct{}

func (latest) Index() uint {
	return 0
}

type set[N constraints.Unsigned] struct {
	inner setIDNumber[N]
}

func (set[N]) Index() uint {
	return 1
}

type unknown struct{}

func (unknown) Index() uint {
	return 2
}

// Three states that can be returned: Latest, Set (tuple), Unknown
func (asc *AuthoritySetChanges[N]) getSetID(blockNumber N) (authSetChangeID authoritySetChangeID, err error) {
	if asc == nil {
		return authSetChangeID, fmt.Errorf("getSetID: authSetChanges is nil")
	}
	authSetChangeID = newAuthoritySetChangeID[N]()
	authSet := *asc
	last := authSet[len(authSet)-1]
	if last.BlockNumber < blockNumber {
		err = authSetChangeID.Set(latest{})
		if err != nil {
			return authSetChangeID, err
		}
		return authSetChangeID, nil
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
				panic("invalid return in binary search")
			}
		},
	)
	if idx < len(authSet) {
		authChange := authSet[idx]

		// if this is the first index but not the first set id then we are missing data.
		if idx == 0 && authChange.SetID != 0 {
			err = authSetChangeID.Set(unknown{})
			if err != nil {
				return authSetChangeID, err
			}
			return authSetChangeID, nil
		}
		err = authSetChangeID.Set(set[N]{
			authChange,
		})
		if err != nil {
			return authSetChangeID, err
		}
		return authSetChangeID, nil
	}

	err = authSetChangeID.Set(unknown{})
	if err != nil {
		return authSetChangeID, err
	}
	return authSetChangeID, nil
}

func (asc *AuthoritySetChanges[N]) insert(blockNumber N) {
	var idx int
	if asc == nil {
		panic("authority set changes must be initialised")
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
					panic("invalid return in binary search")
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
				panic("invalid return in binary search")
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

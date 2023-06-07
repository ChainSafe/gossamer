// Copyright 2023 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only
package grandpa

import (
	"github.com/ChainSafe/gossamer/dot/types"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/lib/crypto/ed25519"
	"github.com/stretchr/testify/require"
	"strings"
	"testing"
)

func staticIsDescendentOf(value bool) IsDescendentOf {
	return func(common.Hash, common.Hash) (bool, error) { return value, nil }
}

func isDescendentof(f IsDescendentOf) IsDescendentOf {
	return func(h1 common.Hash, h2 common.Hash) (bool, error) { return f(h1, h2) }
}

func TestCurrentLimitFiltersMin(t *testing.T) {
	var currentAuthorities AuthorityList
	kp, err := ed25519.GenerateKeypair()
	require.NoError(t, err)
	currentAuthorities = append(currentAuthorities, types.Authority{
		Key:    kp.Public(),
		Weight: 1,
	})

	finalizedKind := Finalized{}
	delayKind := newDelayKind(finalizedKind)

	pendingChange1 := PendingChange{
		nextAuthorities: currentAuthorities,
		delay:           0,
		canonHeight:     1,
		canonHash:       common.BytesToHash([]byte{1}),
		delayKind:       delayKind,
	}

	pendingChange2 := PendingChange{
		nextAuthorities: currentAuthorities,
		delay:           0,
		canonHeight:     2,
		canonHash:       common.BytesToHash([]byte{2}),
		delayKind:       delayKind,
	}

	authorities := AuthoritySet{
		currentAuthorities:     currentAuthorities,
		setId:                  0,
		pendingStandardChanges: NewChangeTree(),
		pendingForcedChanges:   []PendingChange{},
		authoritySetChanges:    AuthoritySetChanges{},
	}

	err = authorities.addPendingChange(pendingChange1, staticIsDescendentOf(false))
	require.NoError(t, err)

	err = authorities.addPendingChange(pendingChange2, staticIsDescendentOf(false))
	require.NoError(t, err)

	require.Equal(t, uint(1), *authorities.CurrentLimit(0))
	require.Equal(t, uint(1), *authorities.CurrentLimit(1))
	require.Equal(t, uint(2), *authorities.CurrentLimit(2))
	require.Nil(t, authorities.CurrentLimit(3))
}

func TestChangesIteratedInPreOrder(t *testing.T) {
	var currentAuthorities AuthorityList
	kp, err := ed25519.GenerateKeypair()
	require.NoError(t, err)
	currentAuthorities = append(currentAuthorities, types.Authority{
		Key:    kp.Public(),
		Weight: 1,
	})

	finalizedKind := Finalized{}
	delayKindFinalized := newDelayKind(finalizedKind)

	bestKind := Best{}
	delayKindBest := newDelayKind(bestKind)

	authorities := AuthoritySet{
		currentAuthorities:     currentAuthorities,
		setId:                  0,
		pendingStandardChanges: NewChangeTree(),
		pendingForcedChanges:   []PendingChange{},
		authoritySetChanges:    AuthoritySetChanges{},
	}

	changeA := PendingChange{
		nextAuthorities: currentAuthorities,
		delay:           10,
		canonHeight:     5,
		canonHash:       common.BytesToHash([]byte("hash_a")),
		delayKind:       delayKindFinalized,
	}

	changeB := PendingChange{
		nextAuthorities: currentAuthorities,
		delay:           0,
		canonHeight:     5,
		canonHash:       common.BytesToHash([]byte("hash_b")),
		delayKind:       delayKindFinalized,
	}

	changeC := PendingChange{
		nextAuthorities: currentAuthorities,
		delay:           5,
		canonHeight:     10,
		canonHash:       common.BytesToHash([]byte("hash_c")),
		delayKind:       delayKindFinalized,
	}

	err = authorities.addPendingChange(changeA, staticIsDescendentOf(false))
	require.NoError(t, err)

	err = authorities.addPendingChange(changeB, staticIsDescendentOf(false))
	require.NoError(t, err)

	err = authorities.addPendingChange(changeC, isDescendentof(func(h1 common.Hash, h2 common.Hash) (bool, error) {
		if h1 == common.BytesToHash([]byte("hash_a")) && h2 == common.BytesToHash([]byte("hash_c")) {
			return true, nil
		} else if h1 == common.BytesToHash([]byte("hash_b")) && h2 == common.BytesToHash([]byte("hash_c")) {
			return false, nil
		} else {
			panic("unreachable")
		}
	}))
	require.NoError(t, err)

	changeD := PendingChange{
		nextAuthorities: currentAuthorities,
		delay:           2,
		canonHeight:     1,
		canonHash:       common.BytesToHash([]byte("hash_d")),
		delayKind:       delayKindBest,
	}

	changeE := PendingChange{
		nextAuthorities: currentAuthorities,
		delay:           2,
		canonHeight:     0,
		canonHash:       common.BytesToHash([]byte("hash_e")),
		delayKind:       delayKindBest,
	}

	err = authorities.addPendingChange(changeD, staticIsDescendentOf(false))
	require.NoError(t, err)

	err = authorities.addPendingChange(changeE, staticIsDescendentOf(false))
	require.NoError(t, err)

	expectedChanges := []PendingChange{
		changeA, changeC, changeB, changeE, changeD,
	}
	pendingChanges := authorities.PendingChanges()
	require.Equal(t, expectedChanges, pendingChanges)
}

func TestApplyChange(t *testing.T) {
	authorities := AuthoritySet{
		currentAuthorities:     AuthorityList{},
		setId:                  0,
		pendingStandardChanges: NewChangeTree(),
		pendingForcedChanges:   []PendingChange{},
		authoritySetChanges:    AuthoritySetChanges{},
	}

	var setA AuthorityList
	kpA, err := ed25519.GenerateKeypair()
	require.NoError(t, err)
	setA = append(setA, types.Authority{
		Key:    kpA.Public(),
		Weight: 5,
	})

	var setB AuthorityList
	kpB, err := ed25519.GenerateKeypair()
	require.NoError(t, err)
	setB = append(setB, types.Authority{
		Key:    kpB.Public(),
		Weight: 5,
	})

	finalizedKind := Finalized{}
	delayKindFinalized := newDelayKind(finalizedKind)

	changeA := PendingChange{
		nextAuthorities: setA,
		delay:           10,
		canonHeight:     5,
		canonHash:       common.BytesToHash([]byte("hash_a")),
		delayKind:       delayKindFinalized,
	}

	changeB := PendingChange{
		nextAuthorities: setB,
		delay:           10,
		canonHeight:     5,
		canonHash:       common.BytesToHash([]byte("hash_b")),
		delayKind:       delayKindFinalized,
	}

	err = authorities.addPendingChange(changeA, staticIsDescendentOf(true))
	require.NoError(t, err)

	err = authorities.addPendingChange(changeB, staticIsDescendentOf(true))
	require.NoError(t, err)

	expectedChanges := []PendingChange{
		changeA, changeB,
	}
	pendingChanges := authorities.PendingChanges()
	require.Equal(t, expectedChanges, pendingChanges)

	// finalizing "hash_c" won't enact the change signaled at "hash_a" but it will prune out
	// "hash_b"
	status, err := authorities.ApplyStandardChanges(
		common.BytesToHash([]byte("hash_c")),
		11,
		isDescendentof(func(h1 common.Hash, h2 common.Hash) (bool, error) {
			if h1 == common.BytesToHash([]byte("hash_a")) && h2 == common.BytesToHash([]byte("hash_c")) {
				return true, nil
			} else if h1 == common.BytesToHash([]byte("hash_b")) && h2 == common.BytesToHash([]byte("hash_c")) {
				return false, nil
			} else {
				panic("unreachable")
			}
		}),
		false,
		nil,
	)

	require.NoError(t, err)
	require.True(t, status.changed)
	require.Nil(t, status.newSetBlock)

	expectedChanges = []PendingChange{
		changeA,
	}
	pendingChanges = authorities.PendingChanges()
	require.Equal(t, expectedChanges, pendingChanges)
	require.True(t, len(authorities.authoritySetChanges) == 0)

	status, err = authorities.ApplyStandardChanges(
		common.BytesToHash([]byte("hash_d")),
		15,
		isDescendentof(func(h1 common.Hash, h2 common.Hash) (bool, error) {
			if h1 == common.BytesToHash([]byte("hash_a")) && h2 == common.BytesToHash([]byte("hash_d")) {
				return true, nil
			} else {
				panic("unreachable")
			}
		}),
		false,
		nil,
	)

	expectedBlockInfo := &newSetBlockInfo{
		newSetBlockNumber: 15,
		newSetBlockHash:   common.BytesToHash([]byte("hash_d")),
	}

	require.True(t, status.changed)
	require.Equal(t, status.newSetBlock, expectedBlockInfo)
	require.Equal(t, authorities.currentAuthorities, setA)
	require.Equal(t, authorities.setId, uint64(1))

	pendingChanges = authorities.PendingChanges()
	require.Equal(t, 0, len(pendingChanges))
	expChange := AuthorityChange{
		setId:       0,
		blockNumber: 15,
	}
	require.Equal(t, authorities.authoritySetChanges, AuthoritySetChanges{expChange})
}

func TestDisallowMultipleChangesBeingFinalizedAtOnce(t *testing.T) {
	authorities := AuthoritySet{
		currentAuthorities:     AuthorityList{},
		setId:                  0,
		pendingStandardChanges: NewChangeTree(),
		pendingForcedChanges:   []PendingChange{},
		authoritySetChanges:    AuthoritySetChanges{},
	}

	var setA AuthorityList
	kpA, err := ed25519.GenerateKeypair()
	require.NoError(t, err)
	setA = append(setA, types.Authority{
		Key:    kpA.Public(),
		Weight: 5,
	})

	var setC AuthorityList
	kpC, err := ed25519.GenerateKeypair()
	require.NoError(t, err)
	setC = append(setC, types.Authority{
		Key:    kpC.Public(),
		Weight: 5,
	})

	finalizedKind := Finalized{}
	delayKindFinalized := newDelayKind(finalizedKind)

	changeA := PendingChange{
		nextAuthorities: setA,
		delay:           10,
		canonHeight:     5,
		canonHash:       common.BytesToHash([]byte("hash_a")),
		delayKind:       delayKindFinalized,
	}

	changeC := PendingChange{
		nextAuthorities: setC,
		delay:           10,
		canonHeight:     30,
		canonHash:       common.BytesToHash([]byte("hash_c")),
		delayKind:       delayKindFinalized,
	}

	err = authorities.addPendingChange(changeA, staticIsDescendentOf(true))
	require.NoError(t, err)

	err = authorities.addPendingChange(changeC, staticIsDescendentOf(true))
	require.NoError(t, err)

	isDescOf := isDescendentof(func(h1 common.Hash, h2 common.Hash) (bool, error) {
		if h1 == common.BytesToHash([]byte("hash_a")) && h2 == common.BytesToHash([]byte("hash_b")) ||
			h1 == common.BytesToHash([]byte("hash_a")) && h2 == common.BytesToHash([]byte("hash_c")) ||
			h1 == common.BytesToHash([]byte("hash_a")) && h2 == common.BytesToHash([]byte("hash_d")) ||
			h1 == common.BytesToHash([]byte("hash_c")) && h2 == common.BytesToHash([]byte("hash_d")) ||
			h1 == common.BytesToHash([]byte("hash_b")) && h2 == common.BytesToHash([]byte("hash_c")) {
			return true, nil
		} else if h1 == common.BytesToHash([]byte("hash_c")) && h2 == common.BytesToHash([]byte("hash_b")) {
			return false, nil
		} else {
			panic("unreachable")
		}
	})

	// trying to finalize past `change_c` without finalizing `change_a` first
	_, err = authorities.ApplyStandardChanges(
		common.BytesToHash([]byte("hash_d")),
		40,
		isDescOf,
		false,
		nil,
	)

	require.ErrorIs(t, err, errUnfinalizedAncestor)
	require.Equal(t, AuthoritySetChanges{}, authorities.authoritySetChanges)

	status, err := authorities.ApplyStandardChanges(
		common.BytesToHash([]byte("hash_b")),
		15,
		isDescOf,
		false,
		nil,
	)
	require.True(t, status.changed)

	expectedBlockInfo := &newSetBlockInfo{
		newSetBlockNumber: 15,
		newSetBlockHash:   common.BytesToHash([]byte("hash_b")),
	}
	expAuthSetChange := AuthoritySetChanges{AuthorityChange{
		setId:       0,
		blockNumber: 15,
	}}
	require.Equal(t, expectedBlockInfo, status.newSetBlock)
	require.Equal(t, setA, authorities.currentAuthorities)
	require.Equal(t, uint64(1), authorities.setId)
	require.Equal(t, expAuthSetChange, authorities.authoritySetChanges)

	status, err = authorities.ApplyStandardChanges(
		common.BytesToHash([]byte("hash_d")),
		40,
		isDescOf,
		false,
		nil,
	)
	require.True(t, status.changed)

	expectedBlockInfo = &newSetBlockInfo{
		newSetBlockNumber: 40,
		newSetBlockHash:   common.BytesToHash([]byte("hash_d")),
	}
	expAuthSetChange = AuthoritySetChanges{
		AuthorityChange{
			setId:       0,
			blockNumber: 15,
		},
		AuthorityChange{
			setId:       1,
			blockNumber: 40,
		},
	}

	require.Equal(t, expectedBlockInfo, status.newSetBlock)
	require.Equal(t, setC, authorities.currentAuthorities)
	require.Equal(t, uint64(2), authorities.setId)
	require.Equal(t, expAuthSetChange, authorities.authoritySetChanges)
}

func TestEnactsStandardChangeWorks(t *testing.T) {
	authorities := AuthoritySet{
		currentAuthorities:     AuthorityList{},
		setId:                  0,
		pendingStandardChanges: NewChangeTree(),
		pendingForcedChanges:   []PendingChange{},
		authoritySetChanges:    AuthoritySetChanges{},
	}

	var setA AuthorityList
	kpA, err := ed25519.GenerateKeypair()
	require.NoError(t, err)
	setA = append(setA, types.Authority{
		Key:    kpA.Public(),
		Weight: 5,
	})

	finalizedKind := Finalized{}
	delayKindFinalized := newDelayKind(finalizedKind)

	changeA := PendingChange{
		nextAuthorities: setA,
		delay:           10,
		canonHeight:     5,
		canonHash:       common.BytesToHash([]byte("hash_a")),
		delayKind:       delayKindFinalized,
	}

	changeB := PendingChange{
		nextAuthorities: setA,
		delay:           10,
		canonHeight:     20,
		canonHash:       common.BytesToHash([]byte("hash_b")),
		delayKind:       delayKindFinalized,
	}

	err = authorities.addPendingChange(changeA, staticIsDescendentOf(false))
	require.NoError(t, err)

	err = authorities.addPendingChange(changeB, staticIsDescendentOf(true))
	require.NoError(t, err)

	isDescOf := isDescendentof(func(h1 common.Hash, h2 common.Hash) (bool, error) {
		if h1 == common.BytesToHash([]byte("hash_a")) && h2 == common.BytesToHash([]byte("hash_d")) ||
			h1 == common.BytesToHash([]byte("hash_a")) && h2 == common.BytesToHash([]byte("hash_e")) ||
			h1 == common.BytesToHash([]byte("hash_b")) && h2 == common.BytesToHash([]byte("hash_d")) ||
			h1 == common.BytesToHash([]byte("hash_b")) && h2 == common.BytesToHash([]byte("hash_e")) {
			return true, nil
		} else if h1 == common.BytesToHash([]byte("hash_a")) && h2 == common.BytesToHash([]byte("hash_c")) ||
			h1 == common.BytesToHash([]byte("hash_b")) && h2 == common.BytesToHash([]byte("hash_c")) {
			return false, nil
		} else {
			panic("unreachable")
		}
	})

	// "hash_c" won't finalize the existing change since it isn't a descendent
	res, err := authorities.EnactsStandardChange(common.BytesToHash([]byte("hash_c")), 15, isDescOf)
	require.NoError(t, err)
	require.Nil(t, res)

	// "hash_d" at depth 14 won't work either
	res, err = authorities.EnactsStandardChange(common.BytesToHash([]byte("hash_d")), 14, isDescOf)
	require.NoError(t, err)
	require.Nil(t, res)

	// but it should work at depth 15 (change height + depth)
	res, err = authorities.EnactsStandardChange(common.BytesToHash([]byte("hash_d")), 15, isDescOf)
	require.NoError(t, err)
	require.Equal(t, true, *res)

	// finalizing "hash_e" at depth 20 will trigger change at "hash_b", but
	// it can't be applied yet since "hash_a" must be applied first
	res, err = authorities.EnactsStandardChange(common.BytesToHash([]byte("hash_e")), 30, isDescOf)
	require.NoError(t, err)
	require.Equal(t, false, *res)
}

func TestForceChanges(t *testing.T) {
	authorities := AuthoritySet{
		currentAuthorities:     AuthorityList{},
		setId:                  0,
		pendingStandardChanges: NewChangeTree(),
		pendingForcedChanges:   []PendingChange{},
		authoritySetChanges:    AuthoritySetChanges{},
	}

	var setA AuthorityList
	kpA, err := ed25519.GenerateKeypair()
	require.NoError(t, err)
	setA = append(setA, types.Authority{
		Key:    kpA.Public(),
		Weight: 5,
	})

	var setB AuthorityList
	kpB, err := ed25519.GenerateKeypair()
	require.NoError(t, err)
	setB = append(setB, types.Authority{
		Key:    kpB.Public(),
		Weight: 5,
	})

	finalizedKindA := Best{42}
	delayKindFinalizedA := newDelayKind(finalizedKindA)

	finalizedKindB := Best{0}
	delayKindFinalizedB := newDelayKind(finalizedKindB)

	changeA := PendingChange{
		nextAuthorities: setA,
		delay:           10,
		canonHeight:     5,
		canonHash:       common.BytesToHash([]byte("hash_a")),
		delayKind:       delayKindFinalizedA,
	}

	changeB := PendingChange{
		nextAuthorities: setA,
		delay:           10,
		canonHeight:     5,
		canonHash:       common.BytesToHash([]byte("hash_b")),
		delayKind:       delayKindFinalizedB,
	}

	err = authorities.addPendingChange(changeA, staticIsDescendentOf(false))
	require.NoError(t, err)

	err = authorities.addPendingChange(changeB, staticIsDescendentOf(false))
	require.NoError(t, err)

	// no duplicates are allowed
	err = authorities.addPendingChange(changeB, staticIsDescendentOf(false))
	require.ErrorIs(t, err, errDuplicateAuthoritySetChanges)

	res, err := authorities.EnactsStandardChange(common.BytesToHash([]byte("hash_c")), 1, staticIsDescendentOf(true))
	require.NoError(t, err)
	require.Nil(t, res)

	changeC := PendingChange{
		nextAuthorities: setA,
		delay:           3,
		canonHeight:     8,
		canonHash:       common.BytesToHash([]byte("hash_a8")),
		delayKind:       delayKindFinalizedB,
	}

	isDescOfA := isDescendentof(func(h1 common.Hash, _ common.Hash) (bool, error) {
		return strings.HasPrefix(h1.String(), common.BytesToHash([]byte("hash_a")).String()), nil
	})

	err = authorities.addPendingChange(changeC, isDescOfA)
	require.ErrorIs(t, err, errMultiplePendingForcedAuthoritySetChanges)

	// let's try and apply the forced changes.
	// too early and there's no forced changes to apply
	resForced, err := authorities.applyForcedChanges(common.BytesToHash([]byte("hash_a10")), 10, staticIsDescendentOf(true), false, nil)
	require.NoError(t, err)
	require.Nil(t, resForced)

	// too late
	resForced, err = authorities.applyForcedChanges(common.BytesToHash([]byte("hash_a16")), 16, isDescOfA, false, nil)
	require.NoError(t, err)
	require.Nil(t, resForced)

	// on time -- chooses the right change for this fork
	exp := AppliedChanges{
		num: 42,
		set: AuthoritySet{
			currentAuthorities:     setA,
			setId:                  1,
			pendingStandardChanges: NewChangeTree(),
			pendingForcedChanges:   nil,
			authoritySetChanges: AuthoritySetChanges{
				AuthorityChange{
					setId:       0,
					blockNumber: 42,
				},
			},
		},
	}
	resForced, err = authorities.applyForcedChanges(common.BytesToHash([]byte("hash_a15")), 15, isDescOfA, false, nil)
	require.NoError(t, err)
	require.NotNil(t, resForced)
	require.Equal(t, exp, *resForced)
}

func TestForceChangesWithNoDelay(t *testing.T) {
	// NOTE: this is a regression test
	authorities := AuthoritySet{
		currentAuthorities:     AuthorityList{},
		setId:                  0,
		pendingStandardChanges: NewChangeTree(),
		pendingForcedChanges:   []PendingChange{},
		authoritySetChanges:    AuthoritySetChanges{},
	}

	var setA AuthorityList
	kpA, err := ed25519.GenerateKeypair()
	require.NoError(t, err)
	setA = append(setA, types.Authority{
		Key:    kpA.Public(),
		Weight: 5,
	})

	finalizedKind := Best{0}
	delayKindFinalized := newDelayKind(finalizedKind)

	// we create a forced change with no delay
	changeA := PendingChange{
		nextAuthorities: setA,
		delay:           0,
		canonHeight:     5,
		canonHash:       common.BytesToHash([]byte("hash_a")),
		delayKind:       delayKindFinalized,
	}

	// and import it
	err = authorities.addPendingChange(changeA, staticIsDescendentOf(false))
	require.NoError(t, err)

	// it should be enacted at the same block that signaled it
	resForced, err := authorities.applyForcedChanges(common.BytesToHash([]byte("hash_a")), 5, staticIsDescendentOf(false), false, nil)
	require.NoError(t, err)
	require.NotNil(t, resForced)
}

func TestForceChangesBlockedByStandardChanges(t *testing.T) {
	authorities := AuthoritySet{
		currentAuthorities:     AuthorityList{},
		setId:                  0,
		pendingStandardChanges: NewChangeTree(),
		pendingForcedChanges:   []PendingChange{},
		authoritySetChanges:    AuthoritySetChanges{},
	}

	var setA AuthorityList
	kpA, err := ed25519.GenerateKeypair()
	require.NoError(t, err)
	setA = append(setA, types.Authority{
		Key:    kpA.Public(),
		Weight: 5,
	})

	finalizedKind := Finalized{}
	delayKindFinalized := newDelayKind(finalizedKind)

	// effective at #15
	changeA := PendingChange{
		nextAuthorities: setA,
		delay:           5,
		canonHeight:     10,
		canonHash:       common.BytesToHash([]byte("hash_a")),
		delayKind:       delayKindFinalized,
	}

	// effective #20
	changeB := PendingChange{
		nextAuthorities: setA,
		delay:           0,
		canonHeight:     20,
		canonHash:       common.BytesToHash([]byte("hash_b")),
		delayKind:       delayKindFinalized,
	}

	// effective at #35
	changeC := PendingChange{
		nextAuthorities: setA,
		delay:           5,
		canonHeight:     30,
		canonHash:       common.BytesToHash([]byte("hash_c")),
		delayKind:       delayKindFinalized,
	}

	// add some pending standard changes all on the same fork
	err = authorities.addPendingChange(changeA, staticIsDescendentOf(true))
	require.NoError(t, err)

	err = authorities.addPendingChange(changeB, staticIsDescendentOf(true))
	require.NoError(t, err)

	err = authorities.addPendingChange(changeC, staticIsDescendentOf(true))
	require.NoError(t, err)

	finalizedKind2 := Best{31}
	delayKindFinalized2 := newDelayKind(finalizedKind2)

	// effective at #45
	changeD := PendingChange{
		nextAuthorities: setA,
		delay:           5,
		canonHeight:     40,
		canonHash:       common.BytesToHash([]byte("hash_d")),
		delayKind:       delayKindFinalized2,
	}

	err = authorities.addPendingChange(changeD, staticIsDescendentOf(true))
	require.NoError(t, err)

	// the forced change cannot be applied since the pending changes it depends on
	// have not been applied yet.
	_, err = authorities.applyForcedChanges(common.BytesToHash([]byte("hash_d45")), 45, staticIsDescendentOf(true), false, nil)
	require.ErrorIs(t, err, errForcedAuthoritySetChangeDependencyUnsatisfied)
	require.Equal(t, 0, len(authorities.authoritySetChanges))

	// we apply the first pending standard change at #15
	expChanges := AuthoritySetChanges{
		AuthorityChange{
			setId:       0,
			blockNumber: 15,
		},
	}
	_, err = authorities.ApplyStandardChanges(common.BytesToHash([]byte("hash_a15")), 15, staticIsDescendentOf(true), false, nil)
	require.Equal(t, expChanges, authorities.authoritySetChanges)

	// but the forced change still depends on the next standard change
	_, err = authorities.applyForcedChanges(common.BytesToHash([]byte("hash_d45")), 45, staticIsDescendentOf(true), false, nil)
	require.ErrorIs(t, err, errForcedAuthoritySetChangeDependencyUnsatisfied)
	require.Equal(t, expChanges, authorities.authoritySetChanges)

	// we apply the pending standard change at #20
	expChanges = append(expChanges, AuthorityChange{
		setId:       1,
		blockNumber: 20,
	})
	_, err = authorities.ApplyStandardChanges(common.BytesToHash([]byte("hash_b")), 20, staticIsDescendentOf(true), false, nil)
	require.Equal(t, expChanges, authorities.authoritySetChanges)

	// afterwards the forced change at #45 can already be applied since it signals
	// that finality stalled at #31, and the next pending standard change is effective
	// at #35. subsequent forced changes on the same branch must be kept
	expChanges = append(expChanges, AuthorityChange{
		setId:       2,
		blockNumber: 31,
	})
	exp := AppliedChanges{
		num: 31,
		set: AuthoritySet{
			currentAuthorities:     setA,
			setId:                  3,
			pendingStandardChanges: NewChangeTree(),
			pendingForcedChanges:   nil,
			authoritySetChanges:    expChanges,
		},
	}
	resForced, err := authorities.applyForcedChanges(common.BytesToHash([]byte("hash_d")), 45, staticIsDescendentOf(true), false, nil)
	require.NoError(t, err)
	require.NotNil(t, resForced)
	require.Equal(t, exp, *resForced)
}

func TestNextChangeWorks(t *testing.T) {
	var currentAuthorities AuthorityList
	kpA, err := ed25519.GenerateKeypair()
	require.NoError(t, err)
	currentAuthorities = append(currentAuthorities, types.Authority{
		Key:    kpA.Public(),
		Weight: 1,
	})

	authorities := AuthoritySet{
		currentAuthorities:     currentAuthorities,
		setId:                  0,
		pendingStandardChanges: NewChangeTree(),
		pendingForcedChanges:   []PendingChange{},
		authoritySetChanges:    AuthoritySetChanges{},
	}

	finalizedKind := Finalized{}
	delayKindFinalized := newDelayKind(finalizedKind)

	// We have three pending changes with 2 possible roots that are enacted
	// immediately on finality (i.e. standard changes).
	changeA0 := PendingChange{
		nextAuthorities: currentAuthorities,
		delay:           0,
		canonHeight:     5,
		canonHash:       common.BytesToHash([]byte("hash_a0")),
		delayKind:       delayKindFinalized,
	}

	changeA1 := PendingChange{
		nextAuthorities: currentAuthorities,
		delay:           0,
		canonHeight:     10,
		canonHash:       common.BytesToHash([]byte("hash_a1")),
		delayKind:       delayKindFinalized,
	}

	changeB := PendingChange{
		nextAuthorities: currentAuthorities,
		delay:           0,
		canonHeight:     4,
		canonHash:       common.BytesToHash([]byte("hash_b")),
		delayKind:       delayKindFinalized,
	}

	// A0 (#5) <- A10 (#8) <- A1 (#10) <- best_a
	// B (#4) <- best_b
	isDescOf := isDescendentof(func(h1 common.Hash, h2 common.Hash) (bool, error) {
		if h1 == common.BytesToHash([]byte("hash_a0")) && h2 == common.BytesToHash([]byte("hash_a1")) ||
			h1 == common.BytesToHash([]byte("hash_a0")) && h2 == common.BytesToHash([]byte("hash_b")) ||
			h1 == common.BytesToHash([]byte("hash_a1")) && h2 == common.BytesToHash([]byte("best_a")) ||
			h1 == common.BytesToHash([]byte("hash_a10")) && h2 == common.BytesToHash([]byte("best_a")) ||
			h1 == common.BytesToHash([]byte("hash_b")) && h2 == common.BytesToHash([]byte("best_b")) {
			return true, nil
		} else {
			return false, nil
		}
	})

	// add the three pending changes
	err = authorities.addPendingChange(changeB, isDescOf)
	require.NoError(t, err)

	err = authorities.addPendingChange(changeA0, isDescOf)
	require.NoError(t, err)

	err = authorities.addPendingChange(changeA1, isDescOf)
	require.NoError(t, err)

	// the earliest change at block `best_a` should be the change at A0 (#5)
	expChange := &Change{
		hash:   common.BytesToHash([]byte("hash_a0")),
		number: 5,
	}
	change, err := authorities.nextChange(common.BytesToHash([]byte("hash_b")), isDescOf)
	require.NoError(t, err)
	require.Equal(t, expChange, change)

	// the earliest change at block `best_b` should be the change at B (#4)
	expChange = &Change{
		hash:   common.BytesToHash([]byte("hash_b")),
		number: 4,
	}
	change, err = authorities.nextChange(common.BytesToHash([]byte("best_b")), isDescOf)
	require.NoError(t, err)
	require.Equal(t, expChange, change)

	// we apply the change at A0 which should prune it and the fork at B
	_, err = authorities.ApplyStandardChanges(common.BytesToHash([]byte("hash_a0")), 5, isDescOf, false, nil)
	require.NoError(t, err)

	// the next change is now at A1 (#10)
	expChange = &Change{
		hash:   common.BytesToHash([]byte("hash_a1")),
		number: 10,
	}
	change, err = authorities.nextChange(common.BytesToHash([]byte("best_a")), isDescOf)
	require.NoError(t, err)
	require.Equal(t, expChange, change)

	// there's no longer any pending change at `best_b` fork
	change, err = authorities.nextChange(common.BytesToHash([]byte("best_b")), isDescOf)
	require.NoError(t, err)
	require.Nil(t, change)

	// we a forced change at A10 (#8)
	finalizedKind2 := Best{0}
	delayKindFinalized2 := newDelayKind(finalizedKind2)
	changeA10 := PendingChange{
		nextAuthorities: currentAuthorities,
		delay:           0,
		canonHeight:     8,
		canonHash:       common.BytesToHash([]byte("hash_a10")),
		delayKind:       delayKindFinalized2,
	}

	err = authorities.addPendingChange(changeA10, staticIsDescendentOf(false))
	require.NoError(t, err)

	// it should take precedence over the change at A1 (#10)
	expChange = &Change{
		hash:   common.BytesToHash([]byte("hash_a10")),
		number: 8,
	}
	change, err = authorities.nextChange(common.BytesToHash([]byte("best_a")), isDescOf)
	require.NoError(t, err)
	require.Equal(t, expChange, change)
}

func TestMaintainsAuthorityListInvariants(t *testing.T) {
	// empty authority lists are invalid
	require.Nil(t, Genesis(AuthorityList{}))
	require.Nil(t, NewAuthoritySet(AuthorityList{}, 0, NewChangeTree(), nil, nil))

	kpA, err := ed25519.GenerateKeypair()
	require.NoError(t, err)

	kpB, err := ed25519.GenerateKeypair()
	require.NoError(t, err)

	invalidAuthoritiesWeight := AuthorityList{
		{
			Key:    kpA.Public(),
			Weight: 5,
		},
		{
			Key:    kpB.Public(),
			Weight: 0,
		},
	}

	// authority weight of zero is invalid
	require.Nil(t, Genesis(invalidAuthoritiesWeight))
	require.Nil(t, NewAuthoritySet(invalidAuthoritiesWeight, 0, NewChangeTree(), nil, nil))

	authoritySet := Genesis(AuthorityList{types.Authority{
		Key:    kpA.Public(),
		Weight: 5,
	}})

	finalizedKind := Finalized{}
	delayKindFinalized := newDelayKind(finalizedKind)
	invalidChangeEmptyAuthorities := PendingChange{
		nextAuthorities: nil,
		delay:           10,
		canonHeight:     5,
		canonHash:       common.Hash{},
		delayKind:       delayKindFinalized,
	}

	// pending change contains an empty authority set
	err = authoritySet.addPendingChange(invalidChangeEmptyAuthorities, staticIsDescendentOf(false))
	require.ErrorIs(t, err, errInvalidAuthoritySet)

	delayKind := Best{0}
	delayKindBest := newDelayKind(delayKind)

	invalidChangeAuthoritiesWeight := PendingChange{
		nextAuthorities: invalidAuthoritiesWeight,
		delay:           10,
		canonHeight:     5,
		canonHash:       common.Hash{},
		delayKind:       delayKindBest,
	}

	// pending change contains an authority set
	// where one authority has weight of 0
	err = authoritySet.addPendingChange(invalidChangeAuthoritiesWeight, staticIsDescendentOf(false))
	require.ErrorIs(t, err, errInvalidAuthoritySet)
}

func TestCleanUpStaleForcedChangesWhenApplyingStandardChange(t *testing.T) {
	var currentAuthorities AuthorityList
	kpA, err := ed25519.GenerateKeypair()
	require.NoError(t, err)
	currentAuthorities = append(currentAuthorities, types.Authority{
		Key:    kpA.Public(),
		Weight: 1,
	})

	authorities := AuthoritySet{
		currentAuthorities:     currentAuthorities,
		setId:                  0,
		pendingStandardChanges: NewChangeTree(),
		pendingForcedChanges:   []PendingChange{},
		authoritySetChanges:    AuthoritySetChanges{},
	}

	// Create the following pending changes tree:
	//
	//               [#C3]
	//              /
	//             /- (#C2)
	//            /
	// (#A) - (#B) - [#C1]
	//            \
	//             (#C0) - [#D]
	//
	// () - Standard change
	// [] - Forced change
	isDescOf := isDescendentof(func(h1 common.Hash, h2 common.Hash) (bool, error) {
		hashes := []common.Hash{
			common.BytesToHash([]byte("B")),
			common.BytesToHash([]byte("C0")),
			common.BytesToHash([]byte("C1")),
			common.BytesToHash([]byte("C2")),
			common.BytesToHash([]byte("C3")),
			common.BytesToHash([]byte("D")),
		}

		if h1 == common.BytesToHash([]byte("B")) && h2 == common.BytesToHash([]byte("B")) {
			return false, nil
		} else if h1 == common.BytesToHash([]byte("A")) || h1 == common.BytesToHash([]byte("B")) {
			for _, val := range hashes {
				if val == h2 {
					return true, nil
				}
			}
			return false, nil
		} else if h1 == common.BytesToHash([]byte("C0")) && h2 == common.BytesToHash([]byte("D")) {
			return true, nil
		}
		return false, nil
	})

	addPendingChangeFunction := func(canonHeight uint, canonHash common.Hash, forced bool) {
		var change PendingChange
		if forced {
			delayKind := Best{0}
			delayKindBest := newDelayKind(delayKind)
			change = PendingChange{
				nextAuthorities: currentAuthorities,
				delay:           0,
				canonHeight:     canonHeight,
				canonHash:       canonHash,
				delayKind:       delayKindBest,
			}
		} else {
			delayKind := Finalized{}
			delayKindFinalized := newDelayKind(delayKind)
			change = PendingChange{
				nextAuthorities: currentAuthorities,
				delay:           0,
				canonHeight:     canonHeight,
				canonHash:       canonHash,
				delayKind:       delayKindFinalized,
			}
		}

		err := authorities.addPendingChange(change, isDescOf)
		require.NoError(t, err)
	}

	addPendingChangeFunction(5, common.BytesToHash([]byte("A")), false)
	addPendingChangeFunction(10, common.BytesToHash([]byte("B")), false)
	addPendingChangeFunction(15, common.BytesToHash([]byte("C0")), false)
	addPendingChangeFunction(15, common.BytesToHash([]byte("C1")), true)
	addPendingChangeFunction(15, common.BytesToHash([]byte("C2")), false)
	addPendingChangeFunction(15, common.BytesToHash([]byte("C3")), true)
	addPendingChangeFunction(20, common.BytesToHash([]byte("D")), true)

	// applying the standard change at A should not prune anything
	// other then the change that was applied
	_, err = authorities.ApplyStandardChanges(common.BytesToHash([]byte("A")), 5, isDescOf, false, nil)
	require.NoError(t, err)
	require.Equal(t, 6, len(authorities.PendingChanges()))

	// same for B
	_, err = authorities.ApplyStandardChanges(common.BytesToHash([]byte("B")), 10, isDescOf, false, nil)
	require.NoError(t, err)
	require.Equal(t, 5, len(authorities.PendingChanges()))

	// finalizing C2 should clear all forced changes
	_, err = authorities.ApplyStandardChanges(common.BytesToHash([]byte("C2")), 15, isDescOf, false, nil)
	require.NoError(t, err)
	require.Equal(t, 0, len(authorities.pendingForcedChanges))
}

func TestCleanUpStaleForcedChangesWhenApplyingStandardChangeAlternateCase(t *testing.T) {
	var currentAuthorities AuthorityList
	kpA, err := ed25519.GenerateKeypair()
	require.NoError(t, err)
	currentAuthorities = append(currentAuthorities, types.Authority{
		Key:    kpA.Public(),
		Weight: 1,
	})

	authorities := AuthoritySet{
		currentAuthorities:     currentAuthorities,
		setId:                  0,
		pendingStandardChanges: NewChangeTree(),
		pendingForcedChanges:   []PendingChange{},
		authoritySetChanges:    AuthoritySetChanges{},
	}

	// Create the following pending changes tree:
	//
	//               [#C3]
	//              /
	//             /- (#C2)
	//            /
	// (#A) - (#B) - [#C1]
	//            \
	//             (#C0) - [#D]
	//
	// () - Standard change
	// [] - Forced change
	isDescOf := isDescendentof(func(h1 common.Hash, h2 common.Hash) (bool, error) {
		hashes := []common.Hash{
			common.BytesToHash([]byte("B")),
			common.BytesToHash([]byte("C0")),
			common.BytesToHash([]byte("C1")),
			common.BytesToHash([]byte("C2")),
			common.BytesToHash([]byte("C3")),
			common.BytesToHash([]byte("D")),
		}

		if h1 == common.BytesToHash([]byte("B")) && h2 == common.BytesToHash([]byte("B")) {
			return false, nil
		} else if h1 == common.BytesToHash([]byte("A")) || h1 == common.BytesToHash([]byte("B")) {
			for _, val := range hashes {
				if val == h2 {
					return true, nil
				}
			}
			return false, nil
		} else if h1 == common.BytesToHash([]byte("C0")) && h2 == common.BytesToHash([]byte("D")) {
			return true, nil
		}
		return false, nil
	})

	addPendingChangeFunction := func(canonHeight uint, canonHash common.Hash, forced bool) {
		var change PendingChange
		if forced {
			delayKind := Best{0}
			delayKindBest := newDelayKind(delayKind)
			change = PendingChange{
				nextAuthorities: currentAuthorities,
				delay:           0,
				canonHeight:     canonHeight,
				canonHash:       canonHash,
				delayKind:       delayKindBest,
			}
		} else {
			delayKind := Finalized{}
			delayKindFinalized := newDelayKind(delayKind)
			change = PendingChange{
				nextAuthorities: currentAuthorities,
				delay:           0,
				canonHeight:     canonHeight,
				canonHash:       canonHash,
				delayKind:       delayKindFinalized,
			}
		}

		err := authorities.addPendingChange(change, isDescOf)
		require.NoError(t, err)
	}

	addPendingChangeFunction(5, common.BytesToHash([]byte("A")), false)
	addPendingChangeFunction(10, common.BytesToHash([]byte("B")), false)
	addPendingChangeFunction(15, common.BytesToHash([]byte("C0")), false)
	addPendingChangeFunction(15, common.BytesToHash([]byte("C1")), true)
	addPendingChangeFunction(15, common.BytesToHash([]byte("C2")), false)
	addPendingChangeFunction(15, common.BytesToHash([]byte("C3")), true)
	addPendingChangeFunction(20, common.BytesToHash([]byte("D")), true)

	// applying the standard change at A should not prune anything
	// other then the change that was applied
	_, err = authorities.ApplyStandardChanges(common.BytesToHash([]byte("A")), 5, isDescOf, false, nil)
	require.NoError(t, err)
	require.Equal(t, 6, len(authorities.PendingChanges()))

	// same for B
	_, err = authorities.ApplyStandardChanges(common.BytesToHash([]byte("B")), 10, isDescOf, false, nil)
	require.NoError(t, err)
	require.Equal(t, 5, len(authorities.PendingChanges()))

	// finalizing C0 should clear all forced changes but D
	_, err = authorities.ApplyStandardChanges(common.BytesToHash([]byte("C0")), 15, isDescOf, false, nil)
	require.NoError(t, err)
	require.Equal(t, 1, len(authorities.pendingForcedChanges))
	require.Equal(t, common.BytesToHash([]byte("D")), authorities.pendingForcedChanges[0].canonHash)
}

func TestAuthoritySetChangesInsert(t *testing.T) {
	// TODO impl
	authoritySetChanges := AuthoritySetChanges{}
	authoritySetChanges.Append(0, 41)
	authoritySetChanges.Append(1, 81)
	authoritySetChanges.Append(4, 121)

	authoritySetChanges.Insert(101)
}

func TestAuthoritySetChangesForCompleteData(t *testing.T) {
	// TODO impl
}

func TestAuthoritySetChangesForIncompleteData(t *testing.T) {
	// TODO impl
}

func TestIterFromWorks(t *testing.T) {
	// TODO impl
}

func TestAuthoritySet_InvalidAuthorityList(t *testing.T) {
	type args struct {
		authorities  AuthorityList
		authoritySet AuthoritySet
	}
	tests := []struct {
		name string
		args args
		exp  bool
	}{
		{
			name: "nil authorities",
			args: args{
				authorities:  nil,
				authoritySet: AuthoritySet{},
			},
			exp: true,
		},
		{
			name: "empty authorities",
			args: args{
				authorities:  AuthorityList{},
				authoritySet: AuthoritySet{},
			},
			exp: true,
		},
		{
			name: "invalid authorities weight",
			args: args{
				authorities: AuthorityList{
					types.Authority{
						Weight: 0,
					},
				},
				authoritySet: AuthoritySet{},
			},
			exp: true,
		},
		{
			name: "valid authority list",
			args: args{
				authorities: AuthorityList{
					types.Authority{
						Weight: 1,
					},
				},
				authoritySet: AuthoritySet{},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.args.authoritySet.InvalidAuthorityList(tt.args.authorities); got != tt.exp {
				t.Errorf("InvalidAuthorityList() = %v, want %v", got, tt.exp)
			}
		})
	}
}

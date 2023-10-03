// Copyright 2023 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only
package grandpa

import (
	"fmt"
	"github.com/ChainSafe/gossamer/lib/crypto/ed25519"
	"github.com/stretchr/testify/require"
	"strings"
	"testing"
)

func staticIsDescendentOf[H comparable](value bool) IsDescendentOf[H] {
	return func(H, H) (bool, error) { return value, nil }
}

func isDescendentof[H comparable](f IsDescendentOf[H]) IsDescendentOf[H] {
	return func(h1, h2 H) (bool, error) { return f(h1, h2) }
}

func TestDelayKind(t *testing.T) {
	finalizedKind := Finalized{}
	delayKind := newDelayKind[uint](finalizedKind)
	_, isFinalizedType := delayKind.value.(Finalized)
	require.True(t, isFinalizedType)

	medLastFinalized := uint(3)
	bestKind := Best[uint]{medianLastFinalized: medLastFinalized}
	delayKind = newDelayKind[uint](bestKind)
	best, isBestType := delayKind.value.(Best[uint])
	require.True(t, isBestType)
	require.Equal(t, medLastFinalized, best.medianLastFinalized)
}

func TestCurrentLimitFiltersMin(t *testing.T) {
	var currentAuthorities []Authority
	kp, err := ed25519.GenerateKeypair()
	require.NoError(t, err)
	currentAuthorities = append(currentAuthorities, Authority{
		Key:    kp.Public(),
		Weight: 1,
	})

	finalizedKind := Finalized{}
	delayKind := newDelayKind[uint](finalizedKind)

	pendingChange1 := PendingChange[string, uint]{
		nextAuthorities: currentAuthorities,
		delay:           0,
		canonHeight:     1,
		canonHash:       "a",
		delayKind:       delayKind,
	}

	pendingChange2 := PendingChange[string, uint]{
		nextAuthorities: currentAuthorities,
		delay:           0,
		canonHeight:     2,
		canonHash:       "b",
		delayKind:       delayKind,
	}

	authorities := AuthoritySet[string, uint]{
		currentAuthorities:     currentAuthorities,
		setID:                  0,
		pendingStandardChanges: NewChangeTree[string, uint](),
		pendingForcedChanges:   []PendingChange[string, uint]{},
		authoritySetChanges:    AuthoritySetChanges[uint]{},
	}

	err = authorities.addPendingChange(pendingChange1, staticIsDescendentOf[string](false))
	require.NoError(t, err)

	err = authorities.addPendingChange(pendingChange2, staticIsDescendentOf[string](false))
	require.NoError(t, err)

	require.Equal(t, uint(1), *authorities.currentLimit(0))
	require.Equal(t, uint(1), *authorities.currentLimit(1))
	require.Equal(t, uint(2), *authorities.currentLimit(2))
	require.Nil(t, authorities.currentLimit(3))
}

func TestChangesIteratedInPreOrder(t *testing.T) {
	var currentAuthorities []Authority
	kp, err := ed25519.GenerateKeypair()
	require.NoError(t, err)
	currentAuthorities = append(currentAuthorities, Authority{
		Key:    kp.Public(),
		Weight: 1,
	})

	finalizedKind := Finalized{}
	delayKindFinalized := newDelayKind[uint](finalizedKind)

	bestKind := Best[uint]{}
	delayKindBest := newDelayKind[uint](bestKind)

	authorities := AuthoritySet[string, uint]{
		currentAuthorities:     currentAuthorities,
		setID:                  0,
		pendingStandardChanges: NewChangeTree[string, uint](),
		pendingForcedChanges:   []PendingChange[string, uint]{},
		authoritySetChanges:    AuthoritySetChanges[uint]{},
	}

	changeA := PendingChange[string, uint]{
		nextAuthorities: currentAuthorities,
		delay:           10,
		canonHeight:     5,
		canonHash:       "hash_a",
		delayKind:       delayKindFinalized,
	}

	changeB := PendingChange[string, uint]{
		nextAuthorities: currentAuthorities,
		delay:           0,
		canonHeight:     5,
		canonHash:       "hash_b",
		delayKind:       delayKindFinalized,
	}

	changeC := PendingChange[string, uint]{
		nextAuthorities: currentAuthorities,
		delay:           5,
		canonHeight:     10,
		canonHash:       "hash_c",
		delayKind:       delayKindFinalized,
	}

	err = authorities.addPendingChange(changeA, staticIsDescendentOf[string](false))
	require.NoError(t, err)

	err = authorities.addPendingChange(changeB, staticIsDescendentOf[string](false))
	require.NoError(t, err)

	err = authorities.addPendingChange(changeC, isDescendentof(func(h1, h2 string) (bool, error) {
		if h1 == "hash_a" && h2 == "hash_c" {
			return true, nil
		} else if h1 == "hash_b" && h2 == "hash_c" {
			return false, nil
		} else {
			panic("unreachable")
		}
	}))
	require.NoError(t, err)

	changeD := PendingChange[string, uint]{
		nextAuthorities: currentAuthorities,
		delay:           2,
		canonHeight:     1,
		canonHash:       "hash_d",
		delayKind:       delayKindBest,
	}

	changeE := PendingChange[string, uint]{
		nextAuthorities: currentAuthorities,
		delay:           2,
		canonHeight:     0,
		canonHash:       "hash_e",
		delayKind:       delayKindBest,
	}

	err = authorities.addPendingChange(changeD, staticIsDescendentOf[string](false))
	require.NoError(t, err)

	err = authorities.addPendingChange(changeE, staticIsDescendentOf[string](false))
	require.NoError(t, err)

	expectedChanges := []PendingChange[string, uint]{
		changeA, changeC, changeB, changeE, changeD,
	}
	pendingChanges := authorities.pendingChanges()
	require.Equal(t, expectedChanges, pendingChanges)
}

func TestApplyChange(t *testing.T) {
	authorities := AuthoritySet[string, uint]{
		currentAuthorities:     []Authority{},
		setID:                  0,
		pendingStandardChanges: NewChangeTree[string, uint](),
		pendingForcedChanges:   []PendingChange[string, uint]{},
		authoritySetChanges:    AuthoritySetChanges[uint]{},
	}

	var setA []Authority
	kpA, err := ed25519.GenerateKeypair()
	require.NoError(t, err)
	setA = append(setA, Authority{
		Key:    kpA.Public(),
		Weight: 5,
	})

	var setB []Authority
	kpB, err := ed25519.GenerateKeypair()
	require.NoError(t, err)
	setB = append(setB, Authority{
		Key:    kpB.Public(),
		Weight: 5,
	})

	finalizedKind := Finalized{}
	delayKindFinalized := newDelayKind[uint](finalizedKind)

	changeA := PendingChange[string, uint]{
		nextAuthorities: setA,
		delay:           10,
		canonHeight:     5,
		canonHash:       "hash_a",
		delayKind:       delayKindFinalized,
	}

	changeB := PendingChange[string, uint]{
		nextAuthorities: setB,
		delay:           10,
		canonHeight:     5,
		canonHash:       "hash_b",
		delayKind:       delayKindFinalized,
	}

	err = authorities.addPendingChange(changeA, staticIsDescendentOf[string](true))
	require.NoError(t, err)

	err = authorities.addPendingChange(changeB, staticIsDescendentOf[string](true))
	require.NoError(t, err)

	expectedChanges := []PendingChange[string, uint]{
		changeA, changeB,
	}
	pendingChanges := authorities.pendingChanges()
	require.Equal(t, expectedChanges, pendingChanges)

	// finalizing "hash_c" won't enact the hashNumber signaled at "hash_a" but it will prune out
	// "hash_b"
	status, err := authorities.applyStandardChanges(
		"hash_c",
		11,
		isDescendentof(func(h1 string, h2 string) (bool, error) {
			if h1 == "hash_a" && h2 == "hash_c" {
				return true, nil
			} else if h1 == "hash_b" && h2 == "hash_c" {
				return false, nil
			} else {
				panic("unreachable")
			}
		}),
		nil,
	)

	require.NoError(t, err)
	require.True(t, status.changed)
	require.Nil(t, status.newSetBlock)

	expectedChanges = []PendingChange[string, uint]{
		changeA,
	}
	pendingChanges = authorities.pendingChanges()
	require.Equal(t, expectedChanges, pendingChanges)
	require.True(t, len(authorities.authoritySetChanges) == 0)

	status, err = authorities.applyStandardChanges(
		"hash_d",
		15,
		isDescendentof(func(h1 string, h2 string) (bool, error) {
			if h1 == "hash_a" && h2 == "hash_d" {
				return true, nil
			} else {
				panic("unreachable")
			}
		}),
		nil,
	)

	expectedBlockInfo := &hashNumber[string, uint]{
		hash:   "hash_d",
		number: 15,
	}

	require.True(t, status.changed)
	require.Equal(t, status.newSetBlock, expectedBlockInfo)
	require.Equal(t, authorities.currentAuthorities, setA)
	require.Equal(t, authorities.setID, uint64(1))

	pendingChanges = authorities.pendingChanges()
	require.Equal(t, 0, len(pendingChanges))
	expChange := setIDNumber[uint]{
		setID:       0,
		blockNumber: 15,
	}
	require.Equal(t, authorities.authoritySetChanges, AuthoritySetChanges[uint]{expChange})
}

func TestDisallowMultipleChangesBeingFinalizedAtOnce(t *testing.T) {
	authorities := AuthoritySet[string, uint]{
		currentAuthorities:     []Authority{},
		setID:                  0,
		pendingStandardChanges: NewChangeTree[string, uint](),
		pendingForcedChanges:   []PendingChange[string, uint]{},
		authoritySetChanges:    AuthoritySetChanges[uint]{},
	}

	var setA []Authority
	kpA, err := ed25519.GenerateKeypair()
	require.NoError(t, err)
	setA = append(setA, Authority{
		Key:    kpA.Public(),
		Weight: 5,
	})

	var setC []Authority
	kpC, err := ed25519.GenerateKeypair()
	require.NoError(t, err)
	setC = append(setC, Authority{
		Key:    kpC.Public(),
		Weight: 5,
	})

	finalizedKind := Finalized{}
	delayKindFinalized := newDelayKind[uint](finalizedKind)

	changeA := PendingChange[string, uint]{
		nextAuthorities: setA,
		delay:           10,
		canonHeight:     5,
		canonHash:       "hash_a",
		delayKind:       delayKindFinalized,
	}

	changeC := PendingChange[string, uint]{
		nextAuthorities: setC,
		delay:           10,
		canonHeight:     30,
		canonHash:       "hash_c",
		delayKind:       delayKindFinalized,
	}

	err = authorities.addPendingChange(changeA, staticIsDescendentOf[string](true))
	require.NoError(t, err)

	err = authorities.addPendingChange(changeC, staticIsDescendentOf[string](true))
	require.NoError(t, err)

	isDescOf := isDescendentof(func(h1 string, h2 string) (bool, error) {
		if h1 == "hash_a" && h2 == "hash_b" ||
			h1 == "hash_a" && h2 == "hash_c" ||
			h1 == "hash_a" && h2 == "hash_d" ||
			h1 == "hash_c" && h2 == "hash_d" ||
			h1 == "hash_b" && h2 == "hash_c" {
			return true, nil
		} else if h1 == "hash_c" && h2 == "hash_b" {
			return false, nil
		} else {
			panic("unreachable")
		}
	})

	// trying to finalize past `change_c` without finalizing `change_a` first
	_, err = authorities.applyStandardChanges(
		"hash_d",
		40,
		isDescOf,
		nil,
	)

	require.ErrorIs(t, err, errUnfinalizedAncestor)
	require.Equal(t, AuthoritySetChanges[uint]{}, authorities.authoritySetChanges)

	status, err := authorities.applyStandardChanges(
		"hash_b",
		15,
		isDescOf,
		nil,
	)
	require.True(t, status.changed)

	expectedBlockInfo := &hashNumber[string, uint]{
		hash:   "hash_b",
		number: 15,
	}
	expAuthSetChange := AuthoritySetChanges[uint]{setIDNumber[uint]{
		setID:       0,
		blockNumber: 15,
	}}
	require.Equal(t, expectedBlockInfo, status.newSetBlock)
	require.Equal(t, setA, authorities.currentAuthorities)
	require.Equal(t, uint64(1), authorities.setID)
	require.Equal(t, expAuthSetChange, authorities.authoritySetChanges)

	status, err = authorities.applyStandardChanges(
		"hash_d",
		40,
		isDescOf,
		nil,
	)
	require.True(t, status.changed)

	expectedBlockInfo = &hashNumber[string, uint]{
		hash:   "hash_d",
		number: 40,
	}
	expAuthSetChange = AuthoritySetChanges[uint]{
		setIDNumber[uint]{
			setID:       0,
			blockNumber: 15,
		},
		setIDNumber[uint]{
			setID:       1,
			blockNumber: 40,
		},
	}

	require.Equal(t, expectedBlockInfo, status.newSetBlock)
	require.Equal(t, setC, authorities.currentAuthorities)
	require.Equal(t, uint64(2), authorities.setID)
	require.Equal(t, expAuthSetChange, authorities.authoritySetChanges)
}

func TestEnactsStandardChangeWorks(t *testing.T) {
	authorities := AuthoritySet[string, uint]{
		currentAuthorities:     []Authority{},
		setID:                  0,
		pendingStandardChanges: NewChangeTree[string, uint](),
		pendingForcedChanges:   []PendingChange[string, uint]{},
		authoritySetChanges:    AuthoritySetChanges[uint]{},
	}

	var setA []Authority
	kpA, err := ed25519.GenerateKeypair()
	require.NoError(t, err)
	setA = append(setA, Authority{
		Key:    kpA.Public(),
		Weight: 5,
	})

	finalizedKind := Finalized{}
	delayKindFinalized := newDelayKind[uint](finalizedKind)

	changeA := PendingChange[string, uint]{
		nextAuthorities: setA,
		delay:           10,
		canonHeight:     5,
		canonHash:       "hash_a",
		delayKind:       delayKindFinalized,
	}

	changeB := PendingChange[string, uint]{
		nextAuthorities: setA,
		delay:           10,
		canonHeight:     20,
		canonHash:       "hash_b",
		delayKind:       delayKindFinalized,
	}

	err = authorities.addPendingChange(changeA, staticIsDescendentOf[string](false))
	require.NoError(t, err)

	err = authorities.addPendingChange(changeB, staticIsDescendentOf[string](true))
	require.NoError(t, err)

	isDescOf := isDescendentof(func(h1 string, h2 string) (bool, error) {
		if h1 == "hash_a" && h2 == "hash_d" ||
			h1 == "hash_a" && h2 == "hash_e" ||
			h1 == "hash_b" && h2 == "hash_d" ||
			h1 == "hash_b" && h2 == "hash_e" {
			return true, nil
		} else if h1 == "hash_a" && h2 == "hash_c" ||
			h1 == "hash_b" && h2 == "hash_c" {
			return false, nil
		} else {
			panic("unreachable")
		}
	})

	// "hash_c" won't finalize the existing hashNumber since it isn't a descendent
	res, err := authorities.EnactsStandardChange("hash_c", 15, isDescOf)
	require.NoError(t, err)
	require.Nil(t, res)

	// "hash_d" at depth 14 won't work either
	res, err = authorities.EnactsStandardChange("hash_d", 14, isDescOf)
	require.NoError(t, err)
	require.Nil(t, res)

	// but it should work at depth 15 (hashNumber height + depth)
	res, err = authorities.EnactsStandardChange("hash_d", 15, isDescOf)
	require.NoError(t, err)
	require.Equal(t, true, *res)

	// finalizing "hash_e" at depth 20 will trigger hashNumber at "hash_b", but
	// it can't be applied yet since "hash_a" must be applied first
	res, err = authorities.EnactsStandardChange("hash_e", 30, isDescOf)
	require.NoError(t, err)
	require.Equal(t, false, *res)
}

func TestForceChanges(t *testing.T) {
	authorities := AuthoritySet[string, uint]{
		currentAuthorities:     []Authority{},
		setID:                  0,
		pendingStandardChanges: NewChangeTree[string, uint](),
		pendingForcedChanges:   []PendingChange[string, uint]{},
		authoritySetChanges:    AuthoritySetChanges[uint]{},
	}

	var setA []Authority
	kpA, err := ed25519.GenerateKeypair()
	require.NoError(t, err)
	setA = append(setA, Authority{
		Key:    kpA.Public(),
		Weight: 5,
	})

	var setB []Authority
	kpB, err := ed25519.GenerateKeypair()
	require.NoError(t, err)
	setB = append(setB, Authority{
		Key:    kpB.Public(),
		Weight: 5,
	})

	finalizedKindA := Best[uint]{42}
	delayKindFinalizedA := newDelayKind[uint](finalizedKindA)

	finalizedKindB := Best[uint]{0}
	delayKindFinalizedB := newDelayKind[uint](finalizedKindB)

	changeA := PendingChange[string, uint]{
		nextAuthorities: setA,
		delay:           10,
		canonHeight:     5,
		canonHash:       "hash_a",
		delayKind:       delayKindFinalizedA,
	}

	changeB := PendingChange[string, uint]{
		nextAuthorities: setA,
		delay:           10,
		canonHeight:     5,
		canonHash:       "hash_b",
		delayKind:       delayKindFinalizedB,
	}

	err = authorities.addPendingChange(changeA, staticIsDescendentOf[string](false))
	require.NoError(t, err)

	err = authorities.addPendingChange(changeB, staticIsDescendentOf[string](false))
	require.NoError(t, err)

	// no duplicates are allowed
	err = authorities.addPendingChange(changeB, staticIsDescendentOf[string](false))
	require.ErrorIs(t, err, errDuplicateAuthoritySetChanges)

	res, err := authorities.EnactsStandardChange("hash_c", 1, staticIsDescendentOf[string](true))
	require.NoError(t, err)
	require.Nil(t, res)

	changeC := PendingChange[string, uint]{
		nextAuthorities: setA,
		delay:           3,
		canonHeight:     8,
		canonHash:       "hash_a8",
		delayKind:       delayKindFinalizedB,
	}

	isDescOfA := isDescendentof(func(h1 string, _ string) (bool, error) {
		return strings.HasPrefix(h1, "hash_a"), nil
	})

	err = authorities.addPendingChange(changeC, isDescOfA)
	require.ErrorIs(t, err, errMultiplePendingForcedAuthoritySetChanges)

	// let's try and apply the forced changes.
	// too early and there's no forced changes to apply
	resForced, err := authorities.applyForcedChanges("hash_a10", 10, staticIsDescendentOf[string](true), nil)
	require.NoError(t, err)
	require.Nil(t, resForced)

	// too late
	resForced, err = authorities.applyForcedChanges("hash_a16", 16, isDescOfA, nil)
	require.NoError(t, err)
	require.Nil(t, resForced)

	// on time -- chooses the right hashNumber for this fork
	exp := appliedChanges[string, uint]{
		median: 42,
		set: AuthoritySet[string, uint]{
			currentAuthorities:     setA,
			setID:                  1,
			pendingStandardChanges: NewChangeTree[string, uint](),
			pendingForcedChanges:   []PendingChange[string, uint]{},
			authoritySetChanges: AuthoritySetChanges[uint]{
				setIDNumber[uint]{
					setID:       0,
					blockNumber: 42,
				},
			},
		},
	}
	resForced, err = authorities.applyForcedChanges("hash_a15", 15, isDescOfA, nil)
	require.NoError(t, err)
	require.NotNil(t, resForced)
	require.Equal(t, exp, *resForced)
}

func TestForceChangesWithNoDelay(t *testing.T) {
	// NOTE: this is a regression test
	authorities := AuthoritySet[string, uint]{
		currentAuthorities:     []Authority{},
		setID:                  0,
		pendingStandardChanges: NewChangeTree[string, uint](),
		pendingForcedChanges:   []PendingChange[string, uint]{},
		authoritySetChanges:    AuthoritySetChanges[uint]{},
	}

	var setA []Authority
	kpA, err := ed25519.GenerateKeypair()
	require.NoError(t, err)
	setA = append(setA, Authority{
		Key:    kpA.Public(),
		Weight: 5,
	})

	finalizedKind := Best[uint]{0}
	delayKindFinalized := newDelayKind[uint](finalizedKind)

	// we create a forced hashNumber with no delay
	changeA := PendingChange[string, uint]{
		nextAuthorities: setA,
		delay:           0,
		canonHeight:     5,
		canonHash:       "hash_a",
		delayKind:       delayKindFinalized,
	}

	// and import it
	err = authorities.addPendingChange(changeA, staticIsDescendentOf[string](false))
	require.NoError(t, err)

	// it should be enacted at the same block that signaled it
	resForced, err := authorities.applyForcedChanges("hash_a", 5, staticIsDescendentOf[string](false), nil)
	require.NoError(t, err)
	require.NotNil(t, resForced)
}

func TestForceChangesBlockedByStandardChanges(t *testing.T) {
	authorities := AuthoritySet[string, uint]{
		currentAuthorities:     []Authority{},
		setID:                  0,
		pendingStandardChanges: NewChangeTree[string, uint](),
		pendingForcedChanges:   []PendingChange[string, uint]{},
		authoritySetChanges:    AuthoritySetChanges[uint]{},
	}

	var setA []Authority
	kpA, err := ed25519.GenerateKeypair()
	require.NoError(t, err)
	setA = append(setA, Authority{
		Key:    kpA.Public(),
		Weight: 5,
	})

	finalizedKind := Finalized{}
	delayKindFinalized := newDelayKind[uint](finalizedKind)

	// effective at #15
	changeA := PendingChange[string, uint]{
		nextAuthorities: setA,
		delay:           5,
		canonHeight:     10,
		canonHash:       "hash_a",
		delayKind:       delayKindFinalized,
	}

	// effective #20
	changeB := PendingChange[string, uint]{
		nextAuthorities: setA,
		delay:           0,
		canonHeight:     20,
		canonHash:       "hash_b",
		delayKind:       delayKindFinalized,
	}

	// effective at #35
	changeC := PendingChange[string, uint]{
		nextAuthorities: setA,
		delay:           5,
		canonHeight:     30,
		canonHash:       "hash_c",
		delayKind:       delayKindFinalized,
	}

	// add some pending standard changes all on the same fork
	err = authorities.addPendingChange(changeA, staticIsDescendentOf[string](true))
	require.NoError(t, err)

	err = authorities.addPendingChange(changeB, staticIsDescendentOf[string](true))
	require.NoError(t, err)

	err = authorities.addPendingChange(changeC, staticIsDescendentOf[string](true))
	require.NoError(t, err)

	finalizedKind2 := Best[uint]{31}
	delayKindFinalized2 := newDelayKind[uint](finalizedKind2)

	// effective at #45
	changeD := PendingChange[string, uint]{
		nextAuthorities: setA,
		delay:           5,
		canonHeight:     40,
		canonHash:       "hash_d",
		delayKind:       delayKindFinalized2,
	}

	err = authorities.addPendingChange(changeD, staticIsDescendentOf[string](true))
	require.NoError(t, err)

	// the forced hashNumber cannot be applied since the pending changes it depends on
	// have not been applied yet.
	_, err = authorities.applyForcedChanges("hash_d45", 45, staticIsDescendentOf[string](true), nil)
	require.ErrorIs(t, err, errForcedAuthoritySetChangeDependencyUnsatisfied)
	require.Equal(t, 0, len(authorities.authoritySetChanges))

	// we apply the first pending standard hashNumber at #15
	expChanges := AuthoritySetChanges[uint]{
		setIDNumber[uint]{
			setID:       0,
			blockNumber: 15,
		},
	}
	_, err = authorities.applyStandardChanges("hash_a15", 15, staticIsDescendentOf[string](true), nil)
	require.Equal(t, expChanges, authorities.authoritySetChanges)

	// but the forced hashNumber still depends on the next standard hashNumber
	_, err = authorities.applyForcedChanges("hash_d45", 45, staticIsDescendentOf[string](true), nil)
	require.ErrorIs(t, err, errForcedAuthoritySetChangeDependencyUnsatisfied)
	require.Equal(t, expChanges, authorities.authoritySetChanges)

	// we apply the pending standard hashNumber at #20
	expChanges = append(expChanges, setIDNumber[uint]{
		setID:       1,
		blockNumber: 20,
	})
	_, err = authorities.applyStandardChanges("hash_b", 20, staticIsDescendentOf[string](true), nil)
	require.Equal(t, expChanges, authorities.authoritySetChanges)

	// afterwards the forced hashNumber at #45 can already be applied since it signals
	// that finality stalled at #31, and the next pending standard hashNumber is effective
	// at #35. subsequent forced changes on the same branch must be kept
	expChanges = append(expChanges, setIDNumber[uint]{
		setID:       2,
		blockNumber: 31,
	})
	exp := appliedChanges[string, uint]{
		median: 31,
		set: AuthoritySet[string, uint]{
			currentAuthorities:     setA,
			setID:                  3,
			pendingStandardChanges: NewChangeTree[string, uint](),
			pendingForcedChanges:   []PendingChange[string, uint]{},
			authoritySetChanges:    expChanges,
		},
	}
	resForced, err := authorities.applyForcedChanges("hash_d", 45, staticIsDescendentOf[string](true), nil)
	require.NoError(t, err)
	require.NotNil(t, resForced)
	require.Equal(t, exp, *resForced)
}

func TestNextChangeWorks(t *testing.T) {
	var currentAuthorities []Authority
	kpA, err := ed25519.GenerateKeypair()
	require.NoError(t, err)
	currentAuthorities = append(currentAuthorities, Authority{
		Key:    kpA.Public(),
		Weight: 1,
	})

	authorities := AuthoritySet[string, uint]{
		currentAuthorities:     currentAuthorities,
		setID:                  0,
		pendingStandardChanges: NewChangeTree[string, uint](),
		pendingForcedChanges:   []PendingChange[string, uint]{},
		authoritySetChanges:    AuthoritySetChanges[uint]{},
	}

	finalizedKind := Finalized{}
	delayKindFinalized := newDelayKind[uint](finalizedKind)

	// We have three pending changes with 2 possible roots that are enacted
	// immediately on finality (i.e. standard changes).
	changeA0 := PendingChange[string, uint]{
		nextAuthorities: currentAuthorities,
		delay:           0,
		canonHeight:     5,
		canonHash:       "hash_a0",
		delayKind:       delayKindFinalized,
	}

	changeA1 := PendingChange[string, uint]{
		nextAuthorities: currentAuthorities,
		delay:           0,
		canonHeight:     10,
		canonHash:       "hash_a1",
		delayKind:       delayKindFinalized,
	}

	changeB := PendingChange[string, uint]{
		nextAuthorities: currentAuthorities,
		delay:           0,
		canonHeight:     4,
		canonHash:       "hash_b",
		delayKind:       delayKindFinalized,
	}

	// A0 (#5) <- A10 (#8) <- A1 (#10) <- best_a
	// B (#4) <- best_b
	isDescOf := isDescendentof(func(h1 string, h2 string) (bool, error) {
		if h1 == "hash_a0" && h2 == "hash_a1" ||
			h1 == "hash_a0" && h2 == "hash_b" ||
			h1 == "hash_a1" && h2 == "best_a" ||
			h1 == "hash_a10" && h2 == "best_a" ||
			h1 == "hash_b" && h2 == "best_b" {
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

	// the earliest hashNumber at block `best_a` should be the hashNumber at A0 (#5)
	expChange := &hashNumber[string, uint]{
		hash:   "hash_a0",
		number: 5,
	}
	c, err := authorities.nextChange("hash_b", isDescOf)
	require.NoError(t, err)
	require.Equal(t, expChange, c)

	// the earliest hashNumber at block `best_b` should be the hashNumber at B (#4)
	expChange = &hashNumber[string, uint]{
		hash:   "hash_b",
		number: 4,
	}
	c, err = authorities.nextChange("best_b", isDescOf)
	require.NoError(t, err)
	require.Equal(t, expChange, c)

	// we apply the hashNumber at A0 which should prune it and the fork at B
	_, err = authorities.applyStandardChanges("hash_a0", 5, isDescOf, nil)
	require.NoError(t, err)

	// the next hashNumber is now at A1 (#10)
	expChange = &hashNumber[string, uint]{
		hash:   "hash_a1",
		number: 10,
	}
	c, err = authorities.nextChange("best_a", isDescOf)
	require.NoError(t, err)
	require.Equal(t, expChange, c)

	// there's no longer any pending hashNumber at `best_b` fork
	c, err = authorities.nextChange("best_b", isDescOf)
	require.NoError(t, err)
	require.Nil(t, c)

	// we a forced hashNumber at A10 (#8)
	finalizedKind2 := Best[uint]{0}
	delayKindFinalized2 := newDelayKind[uint](finalizedKind2)
	changeA10 := PendingChange[string, uint]{
		nextAuthorities: currentAuthorities,
		delay:           0,
		canonHeight:     8,
		canonHash:       "hash_a10",
		delayKind:       delayKindFinalized2,
	}

	err = authorities.addPendingChange(changeA10, staticIsDescendentOf[string](false))
	require.NoError(t, err)

	// it should take precedence over the hashNumber at A1 (#10)
	expChange = &hashNumber[string, uint]{
		hash:   "hash_a10",
		number: 8,
	}
	c, err = authorities.nextChange("best_a", isDescOf)
	require.NoError(t, err)
	require.Equal(t, expChange, c)
}

func TestMaintainsAuthorityListInvariants(t *testing.T) {
	// empty authority lists are invalid
	_, err := NewGenesisAuthoritySet[string, uint]([]Authority{})
	require.NotNil(t, err)
	_, err = NewAuthoritySet[string, uint]([]Authority{}, 0, NewChangeTree[string, uint](), nil, nil)
	require.NotNil(t, err)

	kpA, err := ed25519.GenerateKeypair()
	require.NoError(t, err)

	kpB, err := ed25519.GenerateKeypair()
	require.NoError(t, err)

	invalidAuthoritiesWeight := []Authority{
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
	_, err = NewGenesisAuthoritySet[string, uint](invalidAuthoritiesWeight)
	require.NotNil(t, err)
	_, err = NewAuthoritySet[string, uint](invalidAuthoritiesWeight, 0, NewChangeTree[string, uint](), nil, nil)
	require.NotNil(t, err)

	authoritySet, err := NewGenesisAuthoritySet[string, uint]([]Authority{Authority{
		Key:    kpA.Public(),
		Weight: 5,
	}})
	require.NoError(t, err)

	finalizedKind := Finalized{}
	delayKindFinalized := newDelayKind[uint](finalizedKind)
	invalidChangeEmptyAuthorities := PendingChange[string, uint]{
		nextAuthorities: nil,
		delay:           10,
		canonHeight:     5,
		canonHash:       "",
		delayKind:       delayKindFinalized,
	}

	// pending hashNumber contains an empty authority set
	err = authoritySet.addPendingChange(invalidChangeEmptyAuthorities, staticIsDescendentOf[string](false))
	require.ErrorIs(t, err, errInvalidAuthoritySet)

	delayKind := Best[uint]{0}
	delayKindBest := newDelayKind[uint](delayKind)

	invalidChangeAuthoritiesWeight := PendingChange[string, uint]{
		nextAuthorities: invalidAuthoritiesWeight,
		delay:           10,
		canonHeight:     5,
		canonHash:       "",
		delayKind:       delayKindBest,
	}

	// pending hashNumber contains an authority set
	// where one authority has weight of 0
	err = authoritySet.addPendingChange(invalidChangeAuthoritiesWeight, staticIsDescendentOf[string](false))
	require.ErrorIs(t, err, errInvalidAuthoritySet)
}

func TestCleanUpStaleForcedChangesWhenApplyingStandardChange(t *testing.T) {
	var currentAuthorities []Authority
	kpA, err := ed25519.GenerateKeypair()
	require.NoError(t, err)
	currentAuthorities = append(currentAuthorities, Authority{
		Key:    kpA.Public(),
		Weight: 1,
	})

	authorities := AuthoritySet[string, uint]{
		currentAuthorities:     currentAuthorities,
		setID:                  0,
		pendingStandardChanges: NewChangeTree[string, uint](),
		pendingForcedChanges:   []PendingChange[string, uint]{},
		authoritySetChanges:    AuthoritySetChanges[uint]{},
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
	// () - Standard hashNumber
	// [] - Forced hashNumber
	isDescOf := isDescendentof(func(h1 string, h2 string) (bool, error) {
		hashes := []string{
			"B",
			"C0",
			"C1",
			"C2",
			"C3",
			"D",
		}

		if h1 == "B" && h2 == "B" {
			return false, nil
		} else if h1 == "A" || h1 == "B" {
			for _, val := range hashes {
				if val == h2 {
					return true, nil
				}
			}
			return false, nil
		} else if h1 == "C0" && h2 == "D" {
			return true, nil
		}
		return false, nil
	})

	addPendingChangeFunction := func(canonHeight uint, canonHash string, forced bool) {
		var change PendingChange[string, uint]
		if forced {
			delayKind := Best[uint]{0}
			delayKindBest := newDelayKind[uint](delayKind)
			change = PendingChange[string, uint]{
				nextAuthorities: currentAuthorities,
				delay:           0,
				canonHeight:     canonHeight,
				canonHash:       canonHash,
				delayKind:       delayKindBest,
			}
		} else {
			delayKind := Finalized{}
			delayKindFinalized := newDelayKind[uint](delayKind)
			change = PendingChange[string, uint]{
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

	addPendingChangeFunction(5, "A", false)
	addPendingChangeFunction(10, "B", false)
	addPendingChangeFunction(15, "C0", false)
	addPendingChangeFunction(15, "C1", true)
	addPendingChangeFunction(15, "C2", false)
	addPendingChangeFunction(15, "C3", true)
	addPendingChangeFunction(20, "D", true)

	// applying the standard hashNumber at A should not prune anything
	// other then the hashNumber that was applied
	_, err = authorities.applyStandardChanges("A", 5, isDescOf, nil)
	require.NoError(t, err)
	require.Equal(t, 6, len(authorities.pendingChanges()))

	// same for B
	_, err = authorities.applyStandardChanges("B", 10, isDescOf, nil)
	require.NoError(t, err)
	require.Equal(t, 5, len(authorities.pendingChanges()))

	// finalizing C2 should clear all forced changes
	_, err = authorities.applyStandardChanges("C2", 15, isDescOf, nil)
	require.NoError(t, err)
	require.Equal(t, 0, len(authorities.pendingForcedChanges))
}

func TestCleanUpStaleForcedChangesWhenApplyingStandardChangeAlternateCase(t *testing.T) {
	var currentAuthorities []Authority
	kpA, err := ed25519.GenerateKeypair()
	require.NoError(t, err)
	currentAuthorities = append(currentAuthorities, Authority{
		Key:    kpA.Public(),
		Weight: 1,
	})

	authorities := AuthoritySet[string, uint]{
		currentAuthorities:     currentAuthorities,
		setID:                  0,
		pendingStandardChanges: NewChangeTree[string, uint](),
		pendingForcedChanges:   []PendingChange[string, uint]{},
		authoritySetChanges:    AuthoritySetChanges[uint]{},
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
	// () - Standard hashNumber
	// [] - Forced hashNumber
	isDescOf := isDescendentof(func(h1 string, h2 string) (bool, error) {
		hashes := []string{
			"B",
			"C0",
			"C1",
			"C2",
			"C3",
			"D",
		}

		if h1 == "B" && h2 == "B" {
			return false, nil
		} else if h1 == "A" || h1 == "B" {
			for _, val := range hashes {
				if val == h2 {
					return true, nil
				}
			}
			return false, nil
		} else if h1 == "C0" && h2 == "D" {
			return true, nil
		}
		return false, nil
	})

	addPendingChangeFunction := func(canonHeight uint, canonHash string, forced bool) {
		var change PendingChange[string, uint]
		if forced {
			delayKind := Best[uint]{0}
			delayKindBest := newDelayKind[uint](delayKind)
			change = PendingChange[string, uint]{
				nextAuthorities: currentAuthorities,
				delay:           0,
				canonHeight:     canonHeight,
				canonHash:       canonHash,
				delayKind:       delayKindBest,
			}
		} else {
			delayKind := Finalized{}
			delayKindFinalized := newDelayKind[uint](delayKind)
			change = PendingChange[string, uint]{
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

	addPendingChangeFunction(5, "A", false)
	addPendingChangeFunction(10, "B", false)
	addPendingChangeFunction(15, "C0", false)
	addPendingChangeFunction(15, "C1", true)
	addPendingChangeFunction(15, "C2", false)
	addPendingChangeFunction(15, "C3", true)
	addPendingChangeFunction(20, "D", true)

	// applying the standard hashNumber at A should not prune anything
	// other then the hashNumber that was applied
	_, err = authorities.applyStandardChanges("A", 5, isDescOf, nil)
	require.NoError(t, err)
	require.Equal(t, 6, len(authorities.pendingChanges()))

	// same for B
	_, err = authorities.applyStandardChanges("B", 10, isDescOf, nil)
	require.NoError(t, err)
	require.Equal(t, 5, len(authorities.pendingChanges()))

	// finalizing C0 should clear all forced changes but D
	_, err = authorities.applyStandardChanges("C0", 15, isDescOf, nil)
	require.NoError(t, err)
	require.Equal(t, 1, len(authorities.pendingForcedChanges))
	require.Equal(t, "D", authorities.pendingForcedChanges[0].canonHash)
}

func assertExpectedSet(t *testing.T, authSetID authoritySetChangeID, expected setIDNumber[uint]) {
	t.Helper()
	authSetVal, err := authSetID.Value()
	require.NoError(t, err)
	switch val := authSetVal.(type) {
	case set[uint]:
		require.Equal(t, expected, val.inner)
	default:
		err = fmt.Errorf("invalid authSetID type")
	}
	require.NoError(t, err)
}

func assertUnknown(t *testing.T, authSetID authoritySetChangeID) {
	t.Helper()
	authSetVal, err := authSetID.Value()
	require.NoError(t, err)
	isUnknown := false
	switch authSetVal.(type) {
	case unknown:
		isUnknown = true
	}
	require.True(t, isUnknown)
}

func assertLatest(t *testing.T, authSetID authoritySetChangeID) {
	t.Helper()
	authSetVal, err := authSetID.Value()
	require.NoError(t, err)
	isLatest := false
	switch authSetVal.(type) {
	case latest:
		isLatest = true
	}
	require.True(t, isLatest)
}

func TestAuthoritySetChangesInsert(t *testing.T) {
	authoritySetChanges := AuthoritySetChanges[uint]{}
	authoritySetChanges.append(0, 41)
	authoritySetChanges.append(1, 81)
	authoritySetChanges.append(4, 121)

	authoritySetChanges.insert(101)

	expChange := setIDNumber[uint]{
		setID:       2,
		blockNumber: 101,
	}

	authSetID, err := authoritySetChanges.getSetID(100)
	require.NoError(t, err)
	assertExpectedSet(t, authSetID, expChange)

	authSetID, err = authoritySetChanges.getSetID(101)
	require.NoError(t, err)
	assertExpectedSet(t, authSetID, expChange)
}

func TestAuthoritySetChangesForCompleteData(t *testing.T) {
	authoritySetChanges := AuthoritySetChanges[uint]{}
	authoritySetChanges.append(0, 41)
	authoritySetChanges.append(1, 81)
	authoritySetChanges.append(2, 121)

	expChange0 := setIDNumber[uint]{
		setID:       0,
		blockNumber: 41,
	}

	expChange1 := setIDNumber[uint]{
		setID:       1,
		blockNumber: 81,
	}

	authSetID, err := authoritySetChanges.getSetID(20)
	require.NoError(t, err)
	assertExpectedSet(t, authSetID, expChange0)

	authSetID, err = authoritySetChanges.getSetID(40)
	require.NoError(t, err)
	assertExpectedSet(t, authSetID, expChange0)

	authSetID, err = authoritySetChanges.getSetID(41)
	require.NoError(t, err)
	assertExpectedSet(t, authSetID, expChange0)

	authSetID, err = authoritySetChanges.getSetID(42)
	require.NoError(t, err)
	assertExpectedSet(t, authSetID, expChange1)

	authSetID, err = authoritySetChanges.getSetID(141)
	require.NoError(t, err)
	assertLatest(t, authSetID)
}

func TestAuthoritySetChangesForIncompleteData(t *testing.T) {
	authoritySetChanges := AuthoritySetChanges[uint]{}
	authoritySetChanges.append(2, 41)
	authoritySetChanges.append(3, 81)
	authoritySetChanges.append(4, 121)

	expChange := setIDNumber[uint]{
		setID:       3,
		blockNumber: 81,
	}

	authSetID, err := authoritySetChanges.getSetID(20)
	require.NoError(t, err)
	assertUnknown(t, authSetID)

	authSetID, err = authoritySetChanges.getSetID(40)
	require.NoError(t, err)
	assertUnknown(t, authSetID)

	authSetID, err = authoritySetChanges.getSetID(41)
	require.NoError(t, err)
	assertUnknown(t, authSetID)

	authSetID, err = authoritySetChanges.getSetID(42)
	require.NoError(t, err)
	assertExpectedSet(t, authSetID, expChange)

	authSetID, err = authoritySetChanges.getSetID(141)
	require.NoError(t, err)
	assertLatest(t, authSetID)
}

func TestIterFromWorks(t *testing.T) {
	authoritySetChanges := AuthoritySetChanges[uint]{}
	authoritySetChanges.append(1, 41)
	authoritySetChanges.append(2, 81)

	// we are missing the data for the first set, therefore we should return `None`
	iterSet := authoritySetChanges.IterFrom(40)
	require.Nil(t, iterSet)

	// after adding the data for the first set the same query should work
	authoritySetChanges = AuthoritySetChanges[uint]{}
	authoritySetChanges.append(0, 21)
	authoritySetChanges.append(1, 41)
	authoritySetChanges.append(2, 81)
	authoritySetChanges.append(3, 121)

	expectedChanges := &AuthoritySetChanges[uint]{
		setIDNumber[uint]{
			setID:       1,
			blockNumber: 41,
		},
		setIDNumber[uint]{
			setID:       2,
			blockNumber: 81,
		},
		setIDNumber[uint]{
			setID:       3,
			blockNumber: 121,
		},
	}

	iterSet = authoritySetChanges.IterFrom(40)
	require.Equal(t, expectedChanges, iterSet)

	expectedChanges = &AuthoritySetChanges[uint]{
		setIDNumber[uint]{
			setID:       2,
			blockNumber: 81,
		},
		setIDNumber[uint]{
			setID:       3,
			blockNumber: 121,
		},
	}

	iterSet = authoritySetChanges.IterFrom(41)
	require.Equal(t, expectedChanges, iterSet)

	iterSet = authoritySetChanges.IterFrom(121)
	require.Equal(t, 0, len(*iterSet))

	iterSet = authoritySetChanges.IterFrom(200)
	require.Equal(t, 0, len(*iterSet))
}

func TestAuthoritySet_InvalidAuthorityList(t *testing.T) {
	type args struct {
		authorities []Authority
	}
	tests := []struct {
		name string
		args args
		exp  bool
	}{
		{
			name: "nil authorities",
			args: args{
				authorities: nil,
			},
			exp: true,
		},
		{
			name: "empty authorities",
			args: args{
				authorities: []Authority{},
			},
			exp: true,
		},
		{
			name: "invalid authorities weight",
			args: args{
				authorities: []Authority{
					{
						Weight: 0,
					},
				},
			},
			exp: true,
		},
		{
			name: "valid authority list",
			args: args{
				authorities: []Authority{
					{
						Weight: 1,
					},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := invalidAuthorityList(tt.args.authorities); got != tt.exp {
				t.Errorf("invalidAuthorityList() = %v, want %v", got, tt.exp)
			}
		})
	}
}

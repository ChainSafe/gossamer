// Copyright 2023 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only
package grandpa

import (
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

func TestCurrentLimitFiltersMin(t *testing.T) {
	var currentAuthorities AuthorityList
	kp, err := ed25519.GenerateKeypair()
	require.NoError(t, err)
	currentAuthorities = append(currentAuthorities, Authority{
		Key:    kp.Public(),
		Weight: 1,
	})

	finalizedKind := Finalized{}
	delayKind := newDelayKind(finalizedKind)

	pendingChange1 := PendingChange[Hash, uint]{
		nextAuthorities: currentAuthorities,
		delay:           0,
		canonHeight:     1,
		canonHash:       bytesToHash([]byte{1}),
		delayKind:       delayKind,
	}

	pendingChange2 := PendingChange[Hash, uint]{
		nextAuthorities: currentAuthorities,
		delay:           0,
		canonHeight:     2,
		canonHash:       bytesToHash([]byte{2}),
		delayKind:       delayKind,
	}

	authorities := AuthoritySet[Hash, uint]{
		currentAuthorities:     currentAuthorities,
		setId:                  0,
		pendingStandardChanges: NewChangeTree[Hash, uint](),
		pendingForcedChanges:   []PendingChange[Hash, uint]{},
		authoritySetChanges:    AuthoritySetChanges[uint]{},
	}

	err = authorities.addPendingChange(pendingChange1, staticIsDescendentOf[Hash](false))
	require.NoError(t, err)

	err = authorities.addPendingChange(pendingChange2, staticIsDescendentOf[Hash](false))
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
	currentAuthorities = append(currentAuthorities, Authority{
		Key:    kp.Public(),
		Weight: 1,
	})

	finalizedKind := Finalized{}
	delayKindFinalized := newDelayKind(finalizedKind)

	bestKind := Best{}
	delayKindBest := newDelayKind(bestKind)

	authorities := AuthoritySet[Hash, uint]{
		currentAuthorities:     currentAuthorities,
		setId:                  0,
		pendingStandardChanges: NewChangeTree[Hash, uint](),
		pendingForcedChanges:   []PendingChange[Hash, uint]{},
		authoritySetChanges:    AuthoritySetChanges[uint]{},
	}

	changeA := PendingChange[Hash, uint]{
		nextAuthorities: currentAuthorities,
		delay:           10,
		canonHeight:     5,
		canonHash:       bytesToHash([]byte("hash_a")),
		delayKind:       delayKindFinalized,
	}

	changeB := PendingChange[Hash, uint]{
		nextAuthorities: currentAuthorities,
		delay:           0,
		canonHeight:     5,
		canonHash:       bytesToHash([]byte("hash_b")),
		delayKind:       delayKindFinalized,
	}

	changeC := PendingChange[Hash, uint]{
		nextAuthorities: currentAuthorities,
		delay:           5,
		canonHeight:     10,
		canonHash:       bytesToHash([]byte("hash_c")),
		delayKind:       delayKindFinalized,
	}

	err = authorities.addPendingChange(changeA, staticIsDescendentOf[Hash](false))
	require.NoError(t, err)

	err = authorities.addPendingChange(changeB, staticIsDescendentOf[Hash](false))
	require.NoError(t, err)

	err = authorities.addPendingChange(changeC, isDescendentof(func(h1 Hash, h2 Hash) (bool, error) {
		if h1 == bytesToHash([]byte("hash_a")) && h2 == bytesToHash([]byte("hash_c")) {
			return true, nil
		} else if h1 == bytesToHash([]byte("hash_b")) && h2 == bytesToHash([]byte("hash_c")) {
			return false, nil
		} else {
			panic("unreachable")
		}
	}))
	require.NoError(t, err)

	changeD := PendingChange[Hash, uint]{
		nextAuthorities: currentAuthorities,
		delay:           2,
		canonHeight:     1,
		canonHash:       bytesToHash([]byte("hash_d")),
		delayKind:       delayKindBest,
	}

	changeE := PendingChange[Hash, uint]{
		nextAuthorities: currentAuthorities,
		delay:           2,
		canonHeight:     0,
		canonHash:       bytesToHash([]byte("hash_e")),
		delayKind:       delayKindBest,
	}

	err = authorities.addPendingChange(changeD, staticIsDescendentOf[Hash](false))
	require.NoError(t, err)

	err = authorities.addPendingChange(changeE, staticIsDescendentOf[Hash](false))
	require.NoError(t, err)

	expectedChanges := []PendingChange[Hash, uint]{
		changeA, changeC, changeB, changeE, changeD,
	}
	pendingChanges := authorities.pendingChanges()
	require.Equal(t, expectedChanges, pendingChanges)
}

func TestApplyChange(t *testing.T) {
	authorities := AuthoritySet[Hash, uint]{
		currentAuthorities:     AuthorityList{},
		setId:                  0,
		pendingStandardChanges: NewChangeTree[Hash, uint](),
		pendingForcedChanges:   []PendingChange[Hash, uint]{},
		authoritySetChanges:    AuthoritySetChanges[uint]{},
	}

	var setA AuthorityList
	kpA, err := ed25519.GenerateKeypair()
	require.NoError(t, err)
	setA = append(setA, Authority{
		Key:    kpA.Public(),
		Weight: 5,
	})

	var setB AuthorityList
	kpB, err := ed25519.GenerateKeypair()
	require.NoError(t, err)
	setB = append(setB, Authority{
		Key:    kpB.Public(),
		Weight: 5,
	})

	finalizedKind := Finalized{}
	delayKindFinalized := newDelayKind(finalizedKind)

	changeA := PendingChange[Hash, uint]{
		nextAuthorities: setA,
		delay:           10,
		canonHeight:     5,
		canonHash:       bytesToHash([]byte("hash_a")),
		delayKind:       delayKindFinalized,
	}

	changeB := PendingChange[Hash, uint]{
		nextAuthorities: setB,
		delay:           10,
		canonHeight:     5,
		canonHash:       bytesToHash([]byte("hash_b")),
		delayKind:       delayKindFinalized,
	}

	err = authorities.addPendingChange(changeA, staticIsDescendentOf[Hash](true))
	require.NoError(t, err)

	err = authorities.addPendingChange(changeB, staticIsDescendentOf[Hash](true))
	require.NoError(t, err)

	expectedChanges := []PendingChange[Hash, uint]{
		changeA, changeB,
	}
	pendingChanges := authorities.pendingChanges()
	require.Equal(t, expectedChanges, pendingChanges)

	// finalizing "hash_c" won't enact the hashNumber signaled at "hash_a" but it will prune out
	// "hash_b"
	status, err := authorities.applyStandardChanges(
		bytesToHash([]byte("hash_c")),
		11,
		isDescendentof(func(h1 Hash, h2 Hash) (bool, error) {
			if h1 == bytesToHash([]byte("hash_a")) && h2 == bytesToHash([]byte("hash_c")) {
				return true, nil
			} else if h1 == bytesToHash([]byte("hash_b")) && h2 == bytesToHash([]byte("hash_c")) {
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

	expectedChanges = []PendingChange[Hash, uint]{
		changeA,
	}
	pendingChanges = authorities.pendingChanges()
	require.Equal(t, expectedChanges, pendingChanges)
	require.True(t, len(authorities.authoritySetChanges) == 0)

	status, err = authorities.applyStandardChanges(
		bytesToHash([]byte("hash_d")),
		15,
		isDescendentof(func(h1 Hash, h2 Hash) (bool, error) {
			if h1 == bytesToHash([]byte("hash_a")) && h2 == bytesToHash([]byte("hash_d")) {
				return true, nil
			} else {
				panic("unreachable")
			}
		}),
		nil,
	)

	expectedBlockInfo := &hashNumber[Hash, uint]{
		hash:   bytesToHash([]byte("hash_d")),
		number: 15,
	}

	require.True(t, status.changed)
	require.Equal(t, status.newSetBlock, expectedBlockInfo)
	require.Equal(t, authorities.currentAuthorities, setA)
	require.Equal(t, authorities.setId, uint64(1))

	pendingChanges = authorities.pendingChanges()
	require.Equal(t, 0, len(pendingChanges))
	expChange := authorityChange[uint]{
		setId:       0,
		blockNumber: 15,
	}
	require.Equal(t, authorities.authoritySetChanges, AuthoritySetChanges[uint]{expChange})
}

func TestDisallowMultipleChangesBeingFinalizedAtOnce(t *testing.T) {
	authorities := AuthoritySet[Hash, uint]{
		currentAuthorities:     AuthorityList{},
		setId:                  0,
		pendingStandardChanges: NewChangeTree[Hash, uint](),
		pendingForcedChanges:   []PendingChange[Hash, uint]{},
		authoritySetChanges:    AuthoritySetChanges[uint]{},
	}

	var setA AuthorityList
	kpA, err := ed25519.GenerateKeypair()
	require.NoError(t, err)
	setA = append(setA, Authority{
		Key:    kpA.Public(),
		Weight: 5,
	})

	var setC AuthorityList
	kpC, err := ed25519.GenerateKeypair()
	require.NoError(t, err)
	setC = append(setC, Authority{
		Key:    kpC.Public(),
		Weight: 5,
	})

	finalizedKind := Finalized{}
	delayKindFinalized := newDelayKind(finalizedKind)

	changeA := PendingChange[Hash, uint]{
		nextAuthorities: setA,
		delay:           10,
		canonHeight:     5,
		canonHash:       bytesToHash([]byte("hash_a")),
		delayKind:       delayKindFinalized,
	}

	changeC := PendingChange[Hash, uint]{
		nextAuthorities: setC,
		delay:           10,
		canonHeight:     30,
		canonHash:       bytesToHash([]byte("hash_c")),
		delayKind:       delayKindFinalized,
	}

	err = authorities.addPendingChange(changeA, staticIsDescendentOf[Hash](true))
	require.NoError(t, err)

	err = authorities.addPendingChange(changeC, staticIsDescendentOf[Hash](true))
	require.NoError(t, err)

	isDescOf := isDescendentof(func(h1 Hash, h2 Hash) (bool, error) {
		if h1 == bytesToHash([]byte("hash_a")) && h2 == bytesToHash([]byte("hash_b")) ||
			h1 == bytesToHash([]byte("hash_a")) && h2 == bytesToHash([]byte("hash_c")) ||
			h1 == bytesToHash([]byte("hash_a")) && h2 == bytesToHash([]byte("hash_d")) ||
			h1 == bytesToHash([]byte("hash_c")) && h2 == bytesToHash([]byte("hash_d")) ||
			h1 == bytesToHash([]byte("hash_b")) && h2 == bytesToHash([]byte("hash_c")) {
			return true, nil
		} else if h1 == bytesToHash([]byte("hash_c")) && h2 == bytesToHash([]byte("hash_b")) {
			return false, nil
		} else {
			panic("unreachable")
		}
	})

	// trying to finalize past `change_c` without finalizing `change_a` first
	_, err = authorities.applyStandardChanges(
		bytesToHash([]byte("hash_d")),
		40,
		isDescOf,
		nil,
	)

	require.ErrorIs(t, err, errUnfinalizedAncestor)
	require.Equal(t, AuthoritySetChanges[uint]{}, authorities.authoritySetChanges)

	status, err := authorities.applyStandardChanges(
		bytesToHash([]byte("hash_b")),
		15,
		isDescOf,
		nil,
	)
	require.True(t, status.changed)

	expectedBlockInfo := &hashNumber[Hash, uint]{
		hash:   bytesToHash([]byte("hash_b")),
		number: 15,
	}
	expAuthSetChange := AuthoritySetChanges[uint]{authorityChange[uint]{
		setId:       0,
		blockNumber: 15,
	}}
	require.Equal(t, expectedBlockInfo, status.newSetBlock)
	require.Equal(t, setA, authorities.currentAuthorities)
	require.Equal(t, uint64(1), authorities.setId)
	require.Equal(t, expAuthSetChange, authorities.authoritySetChanges)

	status, err = authorities.applyStandardChanges(
		bytesToHash([]byte("hash_d")),
		40,
		isDescOf,
		nil,
	)
	require.True(t, status.changed)

	expectedBlockInfo = &hashNumber[Hash, uint]{
		hash:   bytesToHash([]byte("hash_d")),
		number: 40,
	}
	expAuthSetChange = AuthoritySetChanges[uint]{
		authorityChange[uint]{
			setId:       0,
			blockNumber: 15,
		},
		authorityChange[uint]{
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
	authorities := AuthoritySet[Hash, uint]{
		currentAuthorities:     AuthorityList{},
		setId:                  0,
		pendingStandardChanges: NewChangeTree[Hash, uint](),
		pendingForcedChanges:   []PendingChange[Hash, uint]{},
		authoritySetChanges:    AuthoritySetChanges[uint]{},
	}

	var setA AuthorityList
	kpA, err := ed25519.GenerateKeypair()
	require.NoError(t, err)
	setA = append(setA, Authority{
		Key:    kpA.Public(),
		Weight: 5,
	})

	finalizedKind := Finalized{}
	delayKindFinalized := newDelayKind(finalizedKind)

	changeA := PendingChange[Hash, uint]{
		nextAuthorities: setA,
		delay:           10,
		canonHeight:     5,
		canonHash:       bytesToHash([]byte("hash_a")),
		delayKind:       delayKindFinalized,
	}

	changeB := PendingChange[Hash, uint]{
		nextAuthorities: setA,
		delay:           10,
		canonHeight:     20,
		canonHash:       bytesToHash([]byte("hash_b")),
		delayKind:       delayKindFinalized,
	}

	err = authorities.addPendingChange(changeA, staticIsDescendentOf[Hash](false))
	require.NoError(t, err)

	err = authorities.addPendingChange(changeB, staticIsDescendentOf[Hash](true))
	require.NoError(t, err)

	isDescOf := isDescendentof(func(h1 Hash, h2 Hash) (bool, error) {
		if h1 == bytesToHash([]byte("hash_a")) && h2 == bytesToHash([]byte("hash_d")) ||
			h1 == bytesToHash([]byte("hash_a")) && h2 == bytesToHash([]byte("hash_e")) ||
			h1 == bytesToHash([]byte("hash_b")) && h2 == bytesToHash([]byte("hash_d")) ||
			h1 == bytesToHash([]byte("hash_b")) && h2 == bytesToHash([]byte("hash_e")) {
			return true, nil
		} else if h1 == bytesToHash([]byte("hash_a")) && h2 == bytesToHash([]byte("hash_c")) ||
			h1 == bytesToHash([]byte("hash_b")) && h2 == bytesToHash([]byte("hash_c")) {
			return false, nil
		} else {
			panic("unreachable")
		}
	})

	// "hash_c" won't finalize the existing hashNumber since it isn't a descendent
	res, err := authorities.EnactsStandardChange(bytesToHash([]byte("hash_c")), 15, isDescOf)
	require.NoError(t, err)
	require.Nil(t, res)

	// "hash_d" at depth 14 won't work either
	res, err = authorities.EnactsStandardChange(bytesToHash([]byte("hash_d")), 14, isDescOf)
	require.NoError(t, err)
	require.Nil(t, res)

	// but it should work at depth 15 (hashNumber height + depth)
	res, err = authorities.EnactsStandardChange(bytesToHash([]byte("hash_d")), 15, isDescOf)
	require.NoError(t, err)
	require.Equal(t, true, *res)

	// finalizing "hash_e" at depth 20 will trigger hashNumber at "hash_b", but
	// it can't be applied yet since "hash_a" must be applied first
	res, err = authorities.EnactsStandardChange(bytesToHash([]byte("hash_e")), 30, isDescOf)
	require.NoError(t, err)
	require.Equal(t, false, *res)
}

func TestForceChanges(t *testing.T) {
	authorities := AuthoritySet[Hash, uint]{
		currentAuthorities:     AuthorityList{},
		setId:                  0,
		pendingStandardChanges: NewChangeTree[Hash, uint](),
		pendingForcedChanges:   []PendingChange[Hash, uint]{},
		authoritySetChanges:    AuthoritySetChanges[uint]{},
	}

	var setA AuthorityList
	kpA, err := ed25519.GenerateKeypair()
	require.NoError(t, err)
	setA = append(setA, Authority{
		Key:    kpA.Public(),
		Weight: 5,
	})

	var setB AuthorityList
	kpB, err := ed25519.GenerateKeypair()
	require.NoError(t, err)
	setB = append(setB, Authority{
		Key:    kpB.Public(),
		Weight: 5,
	})

	finalizedKindA := Best{42}
	delayKindFinalizedA := newDelayKind(finalizedKindA)

	finalizedKindB := Best{0}
	delayKindFinalizedB := newDelayKind(finalizedKindB)

	changeA := PendingChange[Hash, uint]{
		nextAuthorities: setA,
		delay:           10,
		canonHeight:     5,
		canonHash:       bytesToHash([]byte("hash_a")),
		delayKind:       delayKindFinalizedA,
	}

	changeB := PendingChange[Hash, uint]{
		nextAuthorities: setA,
		delay:           10,
		canonHeight:     5,
		canonHash:       bytesToHash([]byte("hash_b")),
		delayKind:       delayKindFinalizedB,
	}

	err = authorities.addPendingChange(changeA, staticIsDescendentOf[Hash](false))
	require.NoError(t, err)

	err = authorities.addPendingChange(changeB, staticIsDescendentOf[Hash](false))
	require.NoError(t, err)

	// no duplicates are allowed
	err = authorities.addPendingChange(changeB, staticIsDescendentOf[Hash](false))
	require.ErrorIs(t, err, errDuplicateAuthoritySetChanges)

	res, err := authorities.EnactsStandardChange(bytesToHash([]byte("hash_c")), 1, staticIsDescendentOf[Hash](true))
	require.NoError(t, err)
	require.Nil(t, res)

	changeC := PendingChange[Hash, uint]{
		nextAuthorities: setA,
		delay:           3,
		canonHeight:     8,
		canonHash:       bytesToHash([]byte("hash_a8")),
		delayKind:       delayKindFinalizedB,
	}

	isDescOfA := isDescendentof(func(h1 Hash, _ Hash) (bool, error) {
		return strings.HasPrefix(h1.String(), bytesToHash([]byte("hash_a")).String()), nil
	})

	err = authorities.addPendingChange(changeC, isDescOfA)
	require.ErrorIs(t, err, errMultiplePendingForcedAuthoritySetChanges)

	// let's try and apply the forced changes.
	// too early and there's no forced changes to apply
	resForced, err := authorities.applyForcedChanges(bytesToHash([]byte("hash_a10")), 10, staticIsDescendentOf[Hash](true), nil)
	require.NoError(t, err)
	require.Nil(t, resForced)

	// too late
	resForced, err = authorities.applyForcedChanges(bytesToHash([]byte("hash_a16")), 16, isDescOfA, nil)
	require.NoError(t, err)
	require.Nil(t, resForced)

	// on time -- chooses the right hashNumber for this fork
	exp := appliedChanges[Hash, uint]{
		median: 42,
		set: AuthoritySet[Hash, uint]{
			currentAuthorities:     setA,
			setId:                  1,
			pendingStandardChanges: NewChangeTree[Hash, uint](),
			pendingForcedChanges:   nil,
			authoritySetChanges: AuthoritySetChanges[uint]{
				authorityChange[uint]{
					setId:       0,
					blockNumber: 42,
				},
			},
		},
	}
	resForced, err = authorities.applyForcedChanges(bytesToHash([]byte("hash_a15")), 15, isDescOfA, nil)
	require.NoError(t, err)
	require.NotNil(t, resForced)
	require.Equal(t, exp, *resForced)
}

func TestForceChangesWithNoDelay(t *testing.T) {
	// NOTE: this is a regression test
	authorities := AuthoritySet[Hash, uint]{
		currentAuthorities:     AuthorityList{},
		setId:                  0,
		pendingStandardChanges: NewChangeTree[Hash, uint](),
		pendingForcedChanges:   []PendingChange[Hash, uint]{},
		authoritySetChanges:    AuthoritySetChanges[uint]{},
	}

	var setA AuthorityList
	kpA, err := ed25519.GenerateKeypair()
	require.NoError(t, err)
	setA = append(setA, Authority{
		Key:    kpA.Public(),
		Weight: 5,
	})

	finalizedKind := Best{0}
	delayKindFinalized := newDelayKind(finalizedKind)

	// we create a forced hashNumber with no delay
	changeA := PendingChange[Hash, uint]{
		nextAuthorities: setA,
		delay:           0,
		canonHeight:     5,
		canonHash:       bytesToHash([]byte("hash_a")),
		delayKind:       delayKindFinalized,
	}

	// and import it
	err = authorities.addPendingChange(changeA, staticIsDescendentOf[Hash](false))
	require.NoError(t, err)

	// it should be enacted at the same block that signaled it
	resForced, err := authorities.applyForcedChanges(bytesToHash([]byte("hash_a")), 5, staticIsDescendentOf[Hash](false), nil)
	require.NoError(t, err)
	require.NotNil(t, resForced)
}

func TestForceChangesBlockedByStandardChanges(t *testing.T) {
	authorities := AuthoritySet[Hash, uint]{
		currentAuthorities:     AuthorityList{},
		setId:                  0,
		pendingStandardChanges: NewChangeTree[Hash, uint](),
		pendingForcedChanges:   []PendingChange[Hash, uint]{},
		authoritySetChanges:    AuthoritySetChanges[uint]{},
	}

	var setA AuthorityList
	kpA, err := ed25519.GenerateKeypair()
	require.NoError(t, err)
	setA = append(setA, Authority{
		Key:    kpA.Public(),
		Weight: 5,
	})

	finalizedKind := Finalized{}
	delayKindFinalized := newDelayKind(finalizedKind)

	// effective at #15
	changeA := PendingChange[Hash, uint]{
		nextAuthorities: setA,
		delay:           5,
		canonHeight:     10,
		canonHash:       bytesToHash([]byte("hash_a")),
		delayKind:       delayKindFinalized,
	}

	// effective #20
	changeB := PendingChange[Hash, uint]{
		nextAuthorities: setA,
		delay:           0,
		canonHeight:     20,
		canonHash:       bytesToHash([]byte("hash_b")),
		delayKind:       delayKindFinalized,
	}

	// effective at #35
	changeC := PendingChange[Hash, uint]{
		nextAuthorities: setA,
		delay:           5,
		canonHeight:     30,
		canonHash:       bytesToHash([]byte("hash_c")),
		delayKind:       delayKindFinalized,
	}

	// add some pending standard changes all on the same fork
	err = authorities.addPendingChange(changeA, staticIsDescendentOf[Hash](true))
	require.NoError(t, err)

	err = authorities.addPendingChange(changeB, staticIsDescendentOf[Hash](true))
	require.NoError(t, err)

	err = authorities.addPendingChange(changeC, staticIsDescendentOf[Hash](true))
	require.NoError(t, err)

	finalizedKind2 := Best{31}
	delayKindFinalized2 := newDelayKind(finalizedKind2)

	// effective at #45
	changeD := PendingChange[Hash, uint]{
		nextAuthorities: setA,
		delay:           5,
		canonHeight:     40,
		canonHash:       bytesToHash([]byte("hash_d")),
		delayKind:       delayKindFinalized2,
	}

	err = authorities.addPendingChange(changeD, staticIsDescendentOf[Hash](true))
	require.NoError(t, err)

	// the forced hashNumber cannot be applied since the pending changes it depends on
	// have not been applied yet.
	_, err = authorities.applyForcedChanges(bytesToHash([]byte("hash_d45")), 45, staticIsDescendentOf[Hash](true), nil)
	require.ErrorIs(t, err, errForcedAuthoritySetChangeDependencyUnsatisfied)
	require.Equal(t, 0, len(authorities.authoritySetChanges))

	// we apply the first pending standard hashNumber at #15
	expChanges := AuthoritySetChanges[uint]{
		authorityChange[uint]{
			setId:       0,
			blockNumber: 15,
		},
	}
	_, err = authorities.applyStandardChanges(bytesToHash([]byte("hash_a15")), 15, staticIsDescendentOf[Hash](true), nil)
	require.Equal(t, expChanges, authorities.authoritySetChanges)

	// but the forced hashNumber still depends on the next standard hashNumber
	_, err = authorities.applyForcedChanges(bytesToHash([]byte("hash_d45")), 45, staticIsDescendentOf[Hash](true), nil)
	require.ErrorIs(t, err, errForcedAuthoritySetChangeDependencyUnsatisfied)
	require.Equal(t, expChanges, authorities.authoritySetChanges)

	// we apply the pending standard hashNumber at #20
	expChanges = append(expChanges, authorityChange[uint]{
		setId:       1,
		blockNumber: 20,
	})
	_, err = authorities.applyStandardChanges(bytesToHash([]byte("hash_b")), 20, staticIsDescendentOf[Hash](true), nil)
	require.Equal(t, expChanges, authorities.authoritySetChanges)

	// afterwards the forced hashNumber at #45 can already be applied since it signals
	// that finality stalled at #31, and the next pending standard hashNumber is effective
	// at #35. subsequent forced changes on the same branch must be kept
	expChanges = append(expChanges, authorityChange[uint]{
		setId:       2,
		blockNumber: 31,
	})
	exp := appliedChanges[Hash, uint]{
		median: 31,
		set: AuthoritySet[Hash, uint]{
			currentAuthorities:     setA,
			setId:                  3,
			pendingStandardChanges: NewChangeTree[Hash, uint](),
			pendingForcedChanges:   nil,
			authoritySetChanges:    expChanges,
		},
	}
	resForced, err := authorities.applyForcedChanges(bytesToHash([]byte("hash_d")), 45, staticIsDescendentOf[Hash](true), nil)
	require.NoError(t, err)
	require.NotNil(t, resForced)
	require.Equal(t, exp, *resForced)
}

func TestNextChangeWorks(t *testing.T) {
	var currentAuthorities AuthorityList
	kpA, err := ed25519.GenerateKeypair()
	require.NoError(t, err)
	currentAuthorities = append(currentAuthorities, Authority{
		Key:    kpA.Public(),
		Weight: 1,
	})

	authorities := AuthoritySet[Hash, uint]{
		currentAuthorities:     currentAuthorities,
		setId:                  0,
		pendingStandardChanges: NewChangeTree[Hash, uint](),
		pendingForcedChanges:   []PendingChange[Hash, uint]{},
		authoritySetChanges:    AuthoritySetChanges[uint]{},
	}

	finalizedKind := Finalized{}
	delayKindFinalized := newDelayKind(finalizedKind)

	// We have three pending changes with 2 possible roots that are enacted
	// immediately on finality (i.e. standard changes).
	changeA0 := PendingChange[Hash, uint]{
		nextAuthorities: currentAuthorities,
		delay:           0,
		canonHeight:     5,
		canonHash:       bytesToHash([]byte("hash_a0")),
		delayKind:       delayKindFinalized,
	}

	changeA1 := PendingChange[Hash, uint]{
		nextAuthorities: currentAuthorities,
		delay:           0,
		canonHeight:     10,
		canonHash:       bytesToHash([]byte("hash_a1")),
		delayKind:       delayKindFinalized,
	}

	changeB := PendingChange[Hash, uint]{
		nextAuthorities: currentAuthorities,
		delay:           0,
		canonHeight:     4,
		canonHash:       bytesToHash([]byte("hash_b")),
		delayKind:       delayKindFinalized,
	}

	// A0 (#5) <- A10 (#8) <- A1 (#10) <- best_a
	// B (#4) <- best_b
	isDescOf := isDescendentof(func(h1 Hash, h2 Hash) (bool, error) {
		if h1 == bytesToHash([]byte("hash_a0")) && h2 == bytesToHash([]byte("hash_a1")) ||
			h1 == bytesToHash([]byte("hash_a0")) && h2 == bytesToHash([]byte("hash_b")) ||
			h1 == bytesToHash([]byte("hash_a1")) && h2 == bytesToHash([]byte("best_a")) ||
			h1 == bytesToHash([]byte("hash_a10")) && h2 == bytesToHash([]byte("best_a")) ||
			h1 == bytesToHash([]byte("hash_b")) && h2 == bytesToHash([]byte("best_b")) {
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
	expChange := &hashNumber[Hash, uint]{
		hash:   bytesToHash([]byte("hash_a0")),
		number: 5,
	}
	c, err := authorities.nextChange(bytesToHash([]byte("hash_b")), isDescOf)
	require.NoError(t, err)
	require.Equal(t, expChange, c)

	// the earliest hashNumber at block `best_b` should be the hashNumber at B (#4)
	expChange = &hashNumber[Hash, uint]{
		hash:   bytesToHash([]byte("hash_b")),
		number: 4,
	}
	c, err = authorities.nextChange(bytesToHash([]byte("best_b")), isDescOf)
	require.NoError(t, err)
	require.Equal(t, expChange, c)

	// we apply the hashNumber at A0 which should prune it and the fork at B
	_, err = authorities.applyStandardChanges(bytesToHash([]byte("hash_a0")), 5, isDescOf, nil)
	require.NoError(t, err)

	// the next hashNumber is now at A1 (#10)
	expChange = &hashNumber[Hash, uint]{
		hash:   bytesToHash([]byte("hash_a1")),
		number: 10,
	}
	c, err = authorities.nextChange(bytesToHash([]byte("best_a")), isDescOf)
	require.NoError(t, err)
	require.Equal(t, expChange, c)

	// there's no longer any pending hashNumber at `best_b` fork
	c, err = authorities.nextChange(bytesToHash([]byte("best_b")), isDescOf)
	require.NoError(t, err)
	require.Nil(t, c)

	// we a forced hashNumber at A10 (#8)
	finalizedKind2 := Best{0}
	delayKindFinalized2 := newDelayKind(finalizedKind2)
	changeA10 := PendingChange[Hash, uint]{
		nextAuthorities: currentAuthorities,
		delay:           0,
		canonHeight:     8,
		canonHash:       bytesToHash([]byte("hash_a10")),
		delayKind:       delayKindFinalized2,
	}

	err = authorities.addPendingChange(changeA10, staticIsDescendentOf[Hash](false))
	require.NoError(t, err)

	// it should take precedence over the hashNumber at A1 (#10)
	expChange = &hashNumber[Hash, uint]{
		hash:   bytesToHash([]byte("hash_a10")),
		number: 8,
	}
	c, err = authorities.nextChange(bytesToHash([]byte("best_a")), isDescOf)
	require.NoError(t, err)
	require.Equal(t, expChange, c)
}

func TestMaintainsAuthorityListInvariants(t *testing.T) {
	// empty authority lists are invalid
	require.Nil(t, NewGenesisAuthoritySet[Hash, uint](AuthorityList{}))
	require.Nil(t, NewAuthoritySet[Hash, uint](AuthorityList{}, 0, NewChangeTree[Hash, uint](), nil, nil))

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
	require.Nil(t, NewGenesisAuthoritySet[Hash, uint](invalidAuthoritiesWeight))
	require.Nil(t, NewAuthoritySet[Hash, uint](invalidAuthoritiesWeight, 0, NewChangeTree[Hash, uint](), nil, nil))

	authoritySet := NewGenesisAuthoritySet[Hash, uint](AuthorityList{Authority{
		Key:    kpA.Public(),
		Weight: 5,
	}})

	finalizedKind := Finalized{}
	delayKindFinalized := newDelayKind(finalizedKind)
	invalidChangeEmptyAuthorities := PendingChange[Hash, uint]{
		nextAuthorities: nil,
		delay:           10,
		canonHeight:     5,
		canonHash:       Hash{},
		delayKind:       delayKindFinalized,
	}

	// pending hashNumber contains an empty authority set
	err = authoritySet.addPendingChange(invalidChangeEmptyAuthorities, staticIsDescendentOf[Hash](false))
	require.ErrorIs(t, err, errInvalidAuthoritySet)

	delayKind := Best{0}
	delayKindBest := newDelayKind(delayKind)

	invalidChangeAuthoritiesWeight := PendingChange[Hash, uint]{
		nextAuthorities: invalidAuthoritiesWeight,
		delay:           10,
		canonHeight:     5,
		canonHash:       Hash{},
		delayKind:       delayKindBest,
	}

	// pending hashNumber contains an authority set
	// where one authority has weight of 0
	err = authoritySet.addPendingChange(invalidChangeAuthoritiesWeight, staticIsDescendentOf[Hash](false))
	require.ErrorIs(t, err, errInvalidAuthoritySet)
}

func TestCleanUpStaleForcedChangesWhenApplyingStandardChange(t *testing.T) {
	var currentAuthorities AuthorityList
	kpA, err := ed25519.GenerateKeypair()
	require.NoError(t, err)
	currentAuthorities = append(currentAuthorities, Authority{
		Key:    kpA.Public(),
		Weight: 1,
	})

	authorities := AuthoritySet[Hash, uint]{
		currentAuthorities:     currentAuthorities,
		setId:                  0,
		pendingStandardChanges: NewChangeTree[Hash, uint](),
		pendingForcedChanges:   []PendingChange[Hash, uint]{},
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
	isDescOf := isDescendentof(func(h1 Hash, h2 Hash) (bool, error) {
		hashes := []Hash{
			bytesToHash([]byte("B")),
			bytesToHash([]byte("C0")),
			bytesToHash([]byte("C1")),
			bytesToHash([]byte("C2")),
			bytesToHash([]byte("C3")),
			bytesToHash([]byte("D")),
		}

		if h1 == bytesToHash([]byte("B")) && h2 == bytesToHash([]byte("B")) {
			return false, nil
		} else if h1 == bytesToHash([]byte("A")) || h1 == bytesToHash([]byte("B")) {
			for _, val := range hashes {
				if val == h2 {
					return true, nil
				}
			}
			return false, nil
		} else if h1 == bytesToHash([]byte("C0")) && h2 == bytesToHash([]byte("D")) {
			return true, nil
		}
		return false, nil
	})

	addPendingChangeFunction := func(canonHeight uint, canonHash Hash, forced bool) {
		var change PendingChange[Hash, uint]
		if forced {
			delayKind := Best{0}
			delayKindBest := newDelayKind(delayKind)
			change = PendingChange[Hash, uint]{
				nextAuthorities: currentAuthorities,
				delay:           0,
				canonHeight:     canonHeight,
				canonHash:       canonHash,
				delayKind:       delayKindBest,
			}
		} else {
			delayKind := Finalized{}
			delayKindFinalized := newDelayKind(delayKind)
			change = PendingChange[Hash, uint]{
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

	addPendingChangeFunction(5, bytesToHash([]byte("A")), false)
	addPendingChangeFunction(10, bytesToHash([]byte("B")), false)
	addPendingChangeFunction(15, bytesToHash([]byte("C0")), false)
	addPendingChangeFunction(15, bytesToHash([]byte("C1")), true)
	addPendingChangeFunction(15, bytesToHash([]byte("C2")), false)
	addPendingChangeFunction(15, bytesToHash([]byte("C3")), true)
	addPendingChangeFunction(20, bytesToHash([]byte("D")), true)

	// applying the standard hashNumber at A should not prune anything
	// other then the hashNumber that was applied
	_, err = authorities.applyStandardChanges(bytesToHash([]byte("A")), 5, isDescOf, nil)
	require.NoError(t, err)
	require.Equal(t, 6, len(authorities.pendingChanges()))

	// same for B
	_, err = authorities.applyStandardChanges(bytesToHash([]byte("B")), 10, isDescOf, nil)
	require.NoError(t, err)
	require.Equal(t, 5, len(authorities.pendingChanges()))

	// finalizing C2 should clear all forced changes
	_, err = authorities.applyStandardChanges(bytesToHash([]byte("C2")), 15, isDescOf, nil)
	require.NoError(t, err)
	require.Equal(t, 0, len(authorities.pendingForcedChanges))
}

func TestCleanUpStaleForcedChangesWhenApplyingStandardChangeAlternateCase(t *testing.T) {
	var currentAuthorities AuthorityList
	kpA, err := ed25519.GenerateKeypair()
	require.NoError(t, err)
	currentAuthorities = append(currentAuthorities, Authority{
		Key:    kpA.Public(),
		Weight: 1,
	})

	authorities := AuthoritySet[Hash, uint]{
		currentAuthorities:     currentAuthorities,
		setId:                  0,
		pendingStandardChanges: NewChangeTree[Hash, uint](),
		pendingForcedChanges:   []PendingChange[Hash, uint]{},
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
	isDescOf := isDescendentof(func(h1 Hash, h2 Hash) (bool, error) {
		hashes := []Hash{
			bytesToHash([]byte("B")),
			bytesToHash([]byte("C0")),
			bytesToHash([]byte("C1")),
			bytesToHash([]byte("C2")),
			bytesToHash([]byte("C3")),
			bytesToHash([]byte("D")),
		}

		if h1 == bytesToHash([]byte("B")) && h2 == bytesToHash([]byte("B")) {
			return false, nil
		} else if h1 == bytesToHash([]byte("A")) || h1 == bytesToHash([]byte("B")) {
			for _, val := range hashes {
				if val == h2 {
					return true, nil
				}
			}
			return false, nil
		} else if h1 == bytesToHash([]byte("C0")) && h2 == bytesToHash([]byte("D")) {
			return true, nil
		}
		return false, nil
	})

	addPendingChangeFunction := func(canonHeight uint, canonHash Hash, forced bool) {
		var change PendingChange[Hash, uint]
		if forced {
			delayKind := Best{0}
			delayKindBest := newDelayKind(delayKind)
			change = PendingChange[Hash, uint]{
				nextAuthorities: currentAuthorities,
				delay:           0,
				canonHeight:     canonHeight,
				canonHash:       canonHash,
				delayKind:       delayKindBest,
			}
		} else {
			delayKind := Finalized{}
			delayKindFinalized := newDelayKind(delayKind)
			change = PendingChange[Hash, uint]{
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

	addPendingChangeFunction(5, bytesToHash([]byte("A")), false)
	addPendingChangeFunction(10, bytesToHash([]byte("B")), false)
	addPendingChangeFunction(15, bytesToHash([]byte("C0")), false)
	addPendingChangeFunction(15, bytesToHash([]byte("C1")), true)
	addPendingChangeFunction(15, bytesToHash([]byte("C2")), false)
	addPendingChangeFunction(15, bytesToHash([]byte("C3")), true)
	addPendingChangeFunction(20, bytesToHash([]byte("D")), true)

	// applying the standard hashNumber at A should not prune anything
	// other then the hashNumber that was applied
	_, err = authorities.applyStandardChanges(bytesToHash([]byte("A")), 5, isDescOf, nil)
	require.NoError(t, err)
	require.Equal(t, 6, len(authorities.pendingChanges()))

	// same for B
	_, err = authorities.applyStandardChanges(bytesToHash([]byte("B")), 10, isDescOf, nil)
	require.NoError(t, err)
	require.Equal(t, 5, len(authorities.pendingChanges()))

	// finalizing C0 should clear all forced changes but D
	_, err = authorities.applyStandardChanges(bytesToHash([]byte("C0")), 15, isDescOf, nil)
	require.NoError(t, err)
	require.Equal(t, 1, len(authorities.pendingForcedChanges))
	require.Equal(t, bytesToHash([]byte("D")), authorities.pendingForcedChanges[0].canonHash)
}

func TestAuthoritySetChangesInsert(t *testing.T) {
	authoritySetChanges := AuthoritySetChanges[uint]{}
	authoritySetChanges.append(0, 41)
	authoritySetChanges.append(1, 81)
	authoritySetChanges.append(4, 121)

	authoritySetChanges.insert(101)

	expChange := authorityChange[uint]{
		setId:       2,
		blockNumber: 101,
	}
	_, set, err := authoritySetChanges.getSetID(100)
	require.NoError(t, err)
	require.Equal(t, expChange, *set)

	_, set, err = authoritySetChanges.getSetID(101)
	require.NoError(t, err)
	require.Equal(t, expChange, *set)
}

func TestAuthoritySetChangesForCompleteData(t *testing.T) {
	authoritySetChanges := AuthoritySetChanges[uint]{}
	authoritySetChanges.append(0, 41)
	authoritySetChanges.append(1, 81)
	authoritySetChanges.append(2, 121)

	expChange0 := authorityChange[uint]{
		setId:       0,
		blockNumber: 41,
	}

	expChange1 := authorityChange[uint]{
		setId:       1,
		blockNumber: 81,
	}

	_, set, err := authoritySetChanges.getSetID(20)
	require.NoError(t, err)
	require.Equal(t, expChange0, *set)

	_, set, err = authoritySetChanges.getSetID(40)
	require.NoError(t, err)
	require.Equal(t, expChange0, *set)

	_, set, err = authoritySetChanges.getSetID(41)
	require.NoError(t, err)
	require.Equal(t, expChange0, *set)

	_, set, err = authoritySetChanges.getSetID(42)
	require.NoError(t, err)
	require.Equal(t, expChange1, *set)

	latest, _, err := authoritySetChanges.getSetID(141)
	require.NoError(t, err)
	require.True(t, latest)
}

func TestAuthoritySetChangesForIncompleteData(t *testing.T) {
	authoritySetChanges := AuthoritySetChanges[uint]{}
	authoritySetChanges.append(2, 41)
	authoritySetChanges.append(3, 81)
	authoritySetChanges.append(4, 121)

	expChange := authorityChange[uint]{
		setId:       3,
		blockNumber: 81,
	}

	_, set, err := authoritySetChanges.getSetID(20)
	require.NoError(t, err)
	require.Nil(t, set)

	_, set, err = authoritySetChanges.getSetID(40)
	require.NoError(t, err)
	require.Nil(t, set)

	_, set, err = authoritySetChanges.getSetID(41)
	require.NoError(t, err)
	require.Nil(t, set)

	_, set, err = authoritySetChanges.getSetID(42)
	require.NoError(t, err)
	require.Equal(t, expChange, *set)

	latest, _, err := authoritySetChanges.getSetID(141)
	require.NoError(t, err)
	require.True(t, latest)
}

func TestIterFromWorks(t *testing.T) {
	authoritySetChanges := AuthoritySetChanges[uint]{}
	authoritySetChanges.append(1, 41)
	authoritySetChanges.append(2, 81)

	// we are missing the data for the first set, therefore we should return `None`
	iterSet := authoritySetChanges.iterFrom(40)
	require.Nil(t, iterSet)

	// after adding the data for the first set the same query should work
	authoritySetChanges = AuthoritySetChanges[uint]{}
	authoritySetChanges.append(0, 21)
	authoritySetChanges.append(1, 41)
	authoritySetChanges.append(2, 81)
	authoritySetChanges.append(3, 121)

	expectedChanges := &AuthoritySetChanges[uint]{
		authorityChange[uint]{
			setId:       1,
			blockNumber: 41,
		},
		authorityChange[uint]{
			setId:       2,
			blockNumber: 81,
		},
		authorityChange[uint]{
			setId:       3,
			blockNumber: 121,
		},
	}

	iterSet = authoritySetChanges.iterFrom(40)
	require.Equal(t, expectedChanges, iterSet)

	expectedChanges = &AuthoritySetChanges[uint]{
		authorityChange[uint]{
			setId:       2,
			blockNumber: 81,
		},
		authorityChange[uint]{
			setId:       3,
			blockNumber: 121,
		},
	}

	iterSet = authoritySetChanges.iterFrom(41)
	require.Equal(t, expectedChanges, iterSet)

	iterSet = authoritySetChanges.iterFrom(121)
	require.Equal(t, 0, len(*iterSet))

	iterSet = authoritySetChanges.iterFrom(200)
	require.Equal(t, 0, len(*iterSet))
}

func TestAuthoritySet_InvalidAuthorityList(t *testing.T) {
	type args struct {
		authorities AuthorityList
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
				authorities: AuthorityList{},
			},
			exp: true,
		},
		{
			name: "invalid authorities weight",
			args: args{
				authorities: AuthorityList{
					Authority{
						Weight: 0,
					},
				},
			},
			exp: true,
		},
		{
			name: "valid authority list",
			args: args{
				authorities: AuthorityList{
					Authority{
						Weight: 1,
					},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := InvalidAuthorityList(tt.args.authorities); got != tt.exp {
				t.Errorf("InvalidAuthorityList() = %v, want %v", got, tt.exp)
			}
		})
	}
}

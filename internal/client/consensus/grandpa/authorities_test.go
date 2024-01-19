// Copyright 2023 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only
package grandpa

import (
	"strings"
	"testing"

	pgrandpa "github.com/ChainSafe/gossamer/internal/primitives/consensus/grandpa"
	"github.com/ChainSafe/gossamer/internal/primitives/consensus/grandpa/app"
	"github.com/stretchr/testify/require"
)

const (
	hashA = "hash_a"
	hashB = "hash_b"
	hashC = "hash_c"
	hashD = "hash_d"
	// key   = dummyAuthID(1)
	// key2  = dummyAuthID(2)
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
	_, isFinalizedType := delayKind.Value.(Finalized)
	require.True(t, isFinalizedType)

	medLastFinalized := uint(3)
	bestKind := Best[uint]{medianLastFinalized: medLastFinalized}
	delayKind = newDelayKind[uint](bestKind)
	best, isBestType := delayKind.Value.(Best[uint])
	require.True(t, isBestType)
	require.Equal(t, medLastFinalized, best.medianLastFinalized)
}

func newTestPublic(t *testing.T, index uint8) app.Public {
	t.Helper()
	data := make([]byte, 32)
	for i := range data {
		data[i] = byte(index)
	}
	pub, err := app.NewPublicFromSlice(data)
	if err != nil {
		t.Fatal(err)
	}
	return pub
}

func TestCurrentLimitFiltersMin(t *testing.T) {
	currentAuthorities := pgrandpa.AuthorityList{
		pgrandpa.AuthorityIDWeight{
			AuthorityID:     newTestPublic(t, 1),
			AuthorityWeight: 1,
		},
	}
	finalisedKind := Finalized{}
	delayKind := newDelayKind[uint](finalisedKind)

	pendingChange1 := PendingChange[string, uint]{
		NextAuthorities: currentAuthorities,
		Delay:           0,
		CanonHeight:     1,
		CanonHash:       "a",
		DelayKind:       delayKind,
	}

	pendingChange2 := PendingChange[string, uint]{
		NextAuthorities: currentAuthorities,
		Delay:           0,
		CanonHeight:     2,
		CanonHash:       "b",
		DelayKind:       delayKind,
	}

	authorities := AuthoritySet[string, uint]{
		CurrentAuthorities:     currentAuthorities,
		SetID:                  0,
		PendingStandardChanges: NewChangeTree[string, uint](),
		PendingForcedChanges:   []PendingChange[string, uint]{},
		AuthoritySetChanges:    AuthoritySetChanges[uint]{},
	}

	err := authorities.addPendingChange(pendingChange1, staticIsDescendentOf[string](false))
	require.NoError(t, err)

	err = authorities.addPendingChange(pendingChange2, staticIsDescendentOf[string](false))
	require.NoError(t, err)

	require.Equal(t, uint(1), *authorities.currentLimit(0))
	require.Equal(t, uint(1), *authorities.currentLimit(1))
	require.Equal(t, uint(2), *authorities.currentLimit(2))
	require.Nil(t, authorities.currentLimit(3))
}

func TestChangesIteratedInPreOrder(t *testing.T) {
	currentAuthorities := pgrandpa.AuthorityList{
		pgrandpa.AuthorityIDWeight{
			AuthorityID:     newTestPublic(t, 1),
			AuthorityWeight: 1,
		},
	}

	finalisedKind := Finalized{}
	delayKindFinalized := newDelayKind[uint](finalisedKind)

	bestKind := Best[uint]{}
	delayKindBest := newDelayKind[uint](bestKind)

	authorities := AuthoritySet[string, uint]{
		CurrentAuthorities:     currentAuthorities,
		SetID:                  0,
		PendingStandardChanges: NewChangeTree[string, uint](),
		PendingForcedChanges:   []PendingChange[string, uint]{},
		AuthoritySetChanges:    AuthoritySetChanges[uint]{},
	}

	changeA := PendingChange[string, uint]{
		NextAuthorities: currentAuthorities,
		Delay:           10,
		CanonHeight:     5,
		CanonHash:       "hash_a",
		DelayKind:       delayKindFinalized,
	}

	changeB := PendingChange[string, uint]{
		NextAuthorities: currentAuthorities,
		Delay:           0,
		CanonHeight:     5,
		CanonHash:       "hash_b",
		DelayKind:       delayKindFinalized,
	}

	changeC := PendingChange[string, uint]{
		NextAuthorities: currentAuthorities,
		Delay:           5,
		CanonHeight:     10,
		CanonHash:       "hash_c",
		DelayKind:       delayKindFinalized,
	}

	err := authorities.addPendingChange(changeA, staticIsDescendentOf[string](false))
	require.NoError(t, err)

	err = authorities.addPendingChange(changeB, staticIsDescendentOf[string](false))
	require.NoError(t, err)

	err = authorities.addPendingChange(changeC, isDescendentof(func(h1, h2 string) (bool, error) {
		if h1 == hashA && h2 == hashC {
			return true, nil
		} else if h1 == hashB && h2 == hashC {
			return false, nil
		} else {
			panic("unreachable")
		}
	}))
	require.NoError(t, err)

	changeD := PendingChange[string, uint]{
		NextAuthorities: currentAuthorities,
		Delay:           2,
		CanonHeight:     1,
		CanonHash:       hashD,
		DelayKind:       delayKindBest,
	}

	changeE := PendingChange[string, uint]{
		NextAuthorities: currentAuthorities,
		Delay:           2,
		CanonHeight:     0,
		CanonHash:       "hash_e",
		DelayKind:       delayKindBest,
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
		SetID:                  0,
		PendingStandardChanges: NewChangeTree[string, uint](),
		PendingForcedChanges:   []PendingChange[string, uint]{},
		AuthoritySetChanges:    AuthoritySetChanges[uint]{},
	}

	setA := pgrandpa.AuthorityList{
		pgrandpa.AuthorityIDWeight{
			AuthorityID:     newTestPublic(t, 1),
			AuthorityWeight: 5,
		},
	}

	setB := pgrandpa.AuthorityList{
		pgrandpa.AuthorityIDWeight{
			AuthorityID:     newTestPublic(t, 2),
			AuthorityWeight: 5,
		},
	}

	finalisedKind := Finalized{}
	delayKindFinalized := newDelayKind[uint](finalisedKind)

	changeA := PendingChange[string, uint]{
		NextAuthorities: setA,
		Delay:           10,
		CanonHeight:     5,
		CanonHash:       hashA,
		DelayKind:       delayKindFinalized,
	}

	changeB := PendingChange[string, uint]{
		NextAuthorities: setB,
		Delay:           10,
		CanonHeight:     5,
		CanonHash:       hashB,
		DelayKind:       delayKindFinalized,
	}

	err := authorities.addPendingChange(changeA, staticIsDescendentOf[string](true))
	require.NoError(t, err)

	err = authorities.addPendingChange(changeB, staticIsDescendentOf[string](true))
	require.NoError(t, err)

	expectedChanges := []PendingChange[string, uint]{
		changeA, changeB,
	}
	pendingChanges := authorities.pendingChanges()
	require.Equal(t, expectedChanges, pendingChanges)

	// finalising hashC won't enact the hashNumber signalled at hashA but it will prune out
	// hashB
	status, err := authorities.applyStandardChanges(
		hashC,
		11,
		isDescendentof(func(h1 string, h2 string) (bool, error) {
			if h1 == hashA && h2 == hashC {
				return true, nil
			} else if h1 == hashB && h2 == hashC {
				return false, nil
			} else {
				panic("unreachable")
			}
		}),
		nil,
	)

	require.NoError(t, err)
	require.True(t, status.Changed)
	require.Nil(t, status.NewSetBlock)

	expectedChanges = []PendingChange[string, uint]{
		changeA,
	}
	pendingChanges = authorities.pendingChanges()
	require.Equal(t, expectedChanges, pendingChanges)
	require.True(t, len(authorities.AuthoritySetChanges) == 0)

	status, err = authorities.applyStandardChanges(
		hashD,
		15,
		isDescendentof(func(h1 string, h2 string) (bool, error) {
			if h1 == hashA && h2 == hashD {
				return true, nil
			} else {
				panic("unreachable")
			}
		}),
		nil,
	)
	require.NoError(t, err)

	expectedBlockInfo := &hashNumber[string, uint]{
		hash:   hashD,
		number: 15,
	}

	require.True(t, status.Changed)
	require.Equal(t, status.NewSetBlock, expectedBlockInfo)
	require.Equal(t, authorities.CurrentAuthorities, setA)
	require.Equal(t, authorities.SetID, uint64(1))

	pendingChanges = authorities.pendingChanges()
	require.Equal(t, 0, len(pendingChanges))
	expChange := setIDNumber[uint]{
		SetID:       0,
		BlockNumber: 15,
	}
	require.Equal(t, authorities.AuthoritySetChanges, AuthoritySetChanges[uint]{expChange})
}

func TestDisallowMultipleChangesBeingFinalizedAtOnce(t *testing.T) {
	authorities := AuthoritySet[string, uint]{
		SetID:                  0,
		PendingStandardChanges: NewChangeTree[string, uint](),
		PendingForcedChanges:   []PendingChange[string, uint]{},
		AuthoritySetChanges:    AuthoritySetChanges[uint]{},
	}

	setA := pgrandpa.AuthorityList{
		pgrandpa.AuthorityIDWeight{
			AuthorityID:     newTestPublic(t, 1),
			AuthorityWeight: 5,
		},
	}

	setC := pgrandpa.AuthorityList{
		pgrandpa.AuthorityIDWeight{
			AuthorityID:     newTestPublic(t, 2),
			AuthorityWeight: 5,
		},
	}

	finalisedKind := Finalized{}
	delayKindFinalized := newDelayKind[uint](finalisedKind)

	changeA := PendingChange[string, uint]{
		NextAuthorities: setA,
		Delay:           10,
		CanonHeight:     5,
		CanonHash:       hashA,
		DelayKind:       delayKindFinalized,
	}

	changeC := PendingChange[string, uint]{
		NextAuthorities: setC,
		Delay:           10,
		CanonHeight:     30,
		CanonHash:       hashC,
		DelayKind:       delayKindFinalized,
	}

	err := authorities.addPendingChange(changeA, staticIsDescendentOf[string](true))
	require.NoError(t, err)

	err = authorities.addPendingChange(changeC, staticIsDescendentOf[string](true))
	require.NoError(t, err)

	isDescOf := isDescendentof(func(h1 string, h2 string) (bool, error) {
		if h1 == hashA && h2 == hashB ||
			h1 == hashA && h2 == hashC ||
			h1 == hashA && h2 == hashD ||
			h1 == hashC && h2 == hashD ||
			h1 == hashB && h2 == hashC {
			return true, nil
		} else if h1 == hashC && h2 == hashB {
			return false, nil
		} else {
			panic("unreachable")
		}
	})

	// trying to finalise past `change_c` without finalising `change_a` first
	_, err = authorities.applyStandardChanges(
		hashD,
		40,
		isDescOf,
		nil,
	)

	require.ErrorIs(t, err, errUnfinalisedAncestor)
	require.Equal(t, AuthoritySetChanges[uint]{}, authorities.AuthoritySetChanges)

	status, err := authorities.applyStandardChanges(
		hashB,
		15,
		isDescOf,
		nil,
	)
	require.NoError(t, err)
	require.True(t, status.Changed)

	expectedBlockInfo := &hashNumber[string, uint]{
		hash:   hashB,
		number: 15,
	}
	expAuthSetChange := AuthoritySetChanges[uint]{setIDNumber[uint]{
		SetID:       0,
		BlockNumber: 15,
	}}
	require.Equal(t, expectedBlockInfo, status.NewSetBlock)
	require.Equal(t, setA, authorities.CurrentAuthorities)
	require.Equal(t, uint64(1), authorities.SetID)
	require.Equal(t, expAuthSetChange, authorities.AuthoritySetChanges)

	status, err = authorities.applyStandardChanges(
		hashD,
		40,
		isDescOf,
		nil,
	)
	require.NoError(t, err)
	require.True(t, status.Changed)

	expectedBlockInfo = &hashNumber[string, uint]{
		hash:   hashD,
		number: 40,
	}
	expAuthSetChange = AuthoritySetChanges[uint]{
		setIDNumber[uint]{
			SetID:       0,
			BlockNumber: 15,
		},
		setIDNumber[uint]{
			SetID:       1,
			BlockNumber: 40,
		},
	}

	require.Equal(t, expectedBlockInfo, status.NewSetBlock)
	require.Equal(t, setC, authorities.CurrentAuthorities)
	require.Equal(t, uint64(2), authorities.SetID)
	require.Equal(t, expAuthSetChange, authorities.AuthoritySetChanges)
}

func TestEnactsStandardChangeWorks(t *testing.T) {
	authorities := AuthoritySet[string, uint]{
		SetID:                  0,
		PendingStandardChanges: NewChangeTree[string, uint](),
		PendingForcedChanges:   []PendingChange[string, uint]{},
		AuthoritySetChanges:    AuthoritySetChanges[uint]{},
	}

	setA := pgrandpa.AuthorityList{
		pgrandpa.AuthorityIDWeight{
			AuthorityID:     newTestPublic(t, 1),
			AuthorityWeight: 5,
		},
	}

	finalisedKind := Finalized{}
	delayKindFinalized := newDelayKind[uint](finalisedKind)

	changeA := PendingChange[string, uint]{
		NextAuthorities: setA,
		Delay:           10,
		CanonHeight:     5,
		CanonHash:       hashA,
		DelayKind:       delayKindFinalized,
	}

	changeB := PendingChange[string, uint]{
		NextAuthorities: setA,
		Delay:           10,
		CanonHeight:     20,
		CanonHash:       hashB,
		DelayKind:       delayKindFinalized,
	}

	err := authorities.addPendingChange(changeA, staticIsDescendentOf[string](false))
	require.NoError(t, err)

	err = authorities.addPendingChange(changeB, staticIsDescendentOf[string](true))
	require.NoError(t, err)

	isDescOf := isDescendentof(func(h1 string, h2 string) (bool, error) {
		if h1 == hashA && h2 == hashD ||
			h1 == hashA && h2 == "hash_e" ||
			h1 == hashB && h2 == hashD ||
			h1 == hashB && h2 == "hash_e" {
			return true, nil
		} else if h1 == hashA && h2 == hashC ||
			h1 == hashB && h2 == hashC {
			return false, nil
		} else {
			panic("unreachable")
		}
	})

	// hashC won't finalise the existing hashNumber since it isn't a descendent
	res, err := authorities.EnactsStandardChange(hashC, 15, isDescOf)
	require.NoError(t, err)
	require.Nil(t, res)

	// hashD at depth 14 won't work either
	res, err = authorities.EnactsStandardChange(hashD, 14, isDescOf)
	require.NoError(t, err)
	require.Nil(t, res)

	// but it should work at depth 15 (hashNumber height + depth)
	res, err = authorities.EnactsStandardChange(hashD, 15, isDescOf)
	require.NoError(t, err)
	require.Equal(t, true, *res)

	// finalising "hash_e" at depth 20 will trigger hashNumber at hashB, but
	// it can't be applied yet since hashA must be applied first
	res, err = authorities.EnactsStandardChange("hash_e", 30, isDescOf)
	require.NoError(t, err)
	require.Equal(t, false, *res)
}

func TestForceChanges(t *testing.T) {
	authorities := AuthoritySet[string, uint]{
		SetID:                  0,
		PendingStandardChanges: NewChangeTree[string, uint](),
		PendingForcedChanges:   []PendingChange[string, uint]{},
		AuthoritySetChanges:    AuthoritySetChanges[uint]{},
	}

	setA := pgrandpa.AuthorityList{
		pgrandpa.AuthorityIDWeight{
			AuthorityID:     newTestPublic(t, 1),
			AuthorityWeight: 5,
		},
	}

	setB := pgrandpa.AuthorityList{
		pgrandpa.AuthorityIDWeight{
			AuthorityID:     newTestPublic(t, 2),
			AuthorityWeight: 5,
		},
	}

	finalisedKindA := Best[uint]{42}
	delayKindFinalizedA := newDelayKind[uint](finalisedKindA)

	finalisedKindB := Best[uint]{0}
	delayKindFinalizedB := newDelayKind[uint](finalisedKindB)

	changeA := PendingChange[string, uint]{
		NextAuthorities: setA,
		Delay:           10,
		CanonHeight:     5,
		CanonHash:       hashA,
		DelayKind:       delayKindFinalizedA,
	}

	changeB := PendingChange[string, uint]{
		NextAuthorities: setB,
		Delay:           10,
		CanonHeight:     5,
		CanonHash:       hashB,
		DelayKind:       delayKindFinalizedB,
	}

	err := authorities.addPendingChange(changeA, staticIsDescendentOf[string](false))
	require.NoError(t, err)

	err = authorities.addPendingChange(changeB, staticIsDescendentOf[string](false))
	require.NoError(t, err)

	// no duplicates are allowed
	err = authorities.addPendingChange(changeB, staticIsDescendentOf[string](false))
	require.ErrorIs(t, err, errDuplicateAuthoritySetChanges)

	res, err := authorities.EnactsStandardChange(hashC, 1, staticIsDescendentOf[string](true))
	require.NoError(t, err)
	require.Nil(t, res)

	changeC := PendingChange[string, uint]{
		NextAuthorities: setA,
		Delay:           3,
		CanonHeight:     8,
		CanonHash:       "hash_a8",
		DelayKind:       delayKindFinalizedB,
	}

	isDescOfA := isDescendentof(func(h1 string, _ string) (bool, error) {
		return strings.HasPrefix(h1, hashA), nil
	})

	err = authorities.addPendingChange(changeC, isDescOfA)
	require.ErrorIs(t, err, errMultiplePendingForcedAuthoritySetChanges)

	// let's try and apply the forced changes.
	// too early and there's no forced changes to apply
	resForced, err := authorities.applyForcedChanges(
		"hash_a10",
		10,
		staticIsDescendentOf[string](true),
		nil,
	)
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
			CurrentAuthorities:     setA,
			SetID:                  1,
			PendingStandardChanges: NewChangeTree[string, uint](),
			PendingForcedChanges:   []PendingChange[string, uint]{},
			AuthoritySetChanges: AuthoritySetChanges[uint]{
				setIDNumber[uint]{
					SetID:       0,
					BlockNumber: 42,
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
		SetID:                  0,
		PendingStandardChanges: NewChangeTree[string, uint](),
		PendingForcedChanges:   []PendingChange[string, uint]{},
		AuthoritySetChanges:    AuthoritySetChanges[uint]{},
	}

	setA := pgrandpa.AuthorityList{
		pgrandpa.AuthorityIDWeight{
			AuthorityID:     newTestPublic(t, 1),
			AuthorityWeight: 5,
		},
	}

	finalisedKind := Best[uint]{0}
	delayKindFinalized := newDelayKind[uint](finalisedKind)

	// we create a forced hashNumber with no Delay
	changeA := PendingChange[string, uint]{
		NextAuthorities: setA,
		Delay:           0,
		CanonHeight:     5,
		CanonHash:       hashA,
		DelayKind:       delayKindFinalized,
	}

	// and import it
	err := authorities.addPendingChange(changeA, staticIsDescendentOf[string](false))
	require.NoError(t, err)

	// it should be enacted at the same block that signalled it
	resForced, err := authorities.applyForcedChanges(
		hashA,
		5,
		staticIsDescendentOf[string](false),
		nil,
	)
	require.NoError(t, err)
	require.NotNil(t, resForced)
}

func TestForceChangesBlockedByStandardChanges(t *testing.T) {
	authorities := AuthoritySet[string, uint]{
		SetID:                  0,
		PendingStandardChanges: NewChangeTree[string, uint](),
		PendingForcedChanges:   []PendingChange[string, uint]{},
		AuthoritySetChanges:    AuthoritySetChanges[uint]{},
	}

	setA := pgrandpa.AuthorityList{
		pgrandpa.AuthorityIDWeight{
			AuthorityID:     newTestPublic(t, 1),
			AuthorityWeight: 5,
		},
	}

	finalisedKind := Finalized{}
	delayKindFinalized := newDelayKind[uint](finalisedKind)

	// effective at #15
	changeA := PendingChange[string, uint]{
		NextAuthorities: setA,
		Delay:           5,
		CanonHeight:     10,
		CanonHash:       hashA,
		DelayKind:       delayKindFinalized,
	}

	// effective #20
	changeB := PendingChange[string, uint]{
		NextAuthorities: setA,
		Delay:           0,
		CanonHeight:     20,
		CanonHash:       hashB,
		DelayKind:       delayKindFinalized,
	}

	// effective at #35
	changeC := PendingChange[string, uint]{
		NextAuthorities: setA,
		Delay:           5,
		CanonHeight:     30,
		CanonHash:       hashC,
		DelayKind:       delayKindFinalized,
	}

	// add some pending standard changes all on the same fork
	err := authorities.addPendingChange(changeA, staticIsDescendentOf[string](true))
	require.NoError(t, err)

	err = authorities.addPendingChange(changeB, staticIsDescendentOf[string](true))
	require.NoError(t, err)

	err = authorities.addPendingChange(changeC, staticIsDescendentOf[string](true))
	require.NoError(t, err)

	finalisedKind2 := Best[uint]{31}
	delayKindFinalized2 := newDelayKind[uint](finalisedKind2)

	// effective at #45
	changeD := PendingChange[string, uint]{
		NextAuthorities: setA,
		Delay:           5,
		CanonHeight:     40,
		CanonHash:       hashD,
		DelayKind:       delayKindFinalized2,
	}

	err = authorities.addPendingChange(changeD, staticIsDescendentOf[string](true))
	require.NoError(t, err)

	// the forced hashNumber cannot be applied since the pending changes it depends on
	// have not been applied yet.
	_, err = authorities.applyForcedChanges(
		"hash_d45",
		45,
		staticIsDescendentOf[string](true),
		nil,
	)
	require.ErrorIs(t, err, errForcedAuthoritySetChangeDependencyUnsatisfied)
	require.Equal(t, 0, len(authorities.AuthoritySetChanges))

	// we apply the first pending standard hashNumber at #15
	expChanges := AuthoritySetChanges[uint]{
		setIDNumber[uint]{
			SetID:       0,
			BlockNumber: 15,
		},
	}
	_, err = authorities.applyStandardChanges(
		"hash_a15",
		15,
		staticIsDescendentOf[string](true),
		nil,
	)
	require.NoError(t, err)
	require.Equal(t, expChanges, authorities.AuthoritySetChanges)

	// but the forced hashNumber still depends on the next standard hashNumber
	_, err = authorities.applyForcedChanges(
		"hash_d45",
		45,
		staticIsDescendentOf[string](true),
		nil,
	)
	require.ErrorIs(t, err, errForcedAuthoritySetChangeDependencyUnsatisfied)
	require.Equal(t, expChanges, authorities.AuthoritySetChanges)

	// we apply the pending standard hashNumber at #20
	expChanges = append(expChanges, setIDNumber[uint]{
		SetID:       1,
		BlockNumber: 20,
	})
	_, err = authorities.applyStandardChanges(
		hashB,
		20,
		staticIsDescendentOf[string](true),
		nil,
	)
	require.NoError(t, err)
	require.Equal(t, expChanges, authorities.AuthoritySetChanges)

	// afterwards the forced hashNumber at #45 can already be applied since it signals
	// that finality stalled at #31, and the next pending standard hashNumber is effective
	// at #35. subsequent forced changes on the same branch must be kept
	expChanges = append(expChanges, setIDNumber[uint]{
		SetID:       2,
		BlockNumber: 31,
	})
	exp := appliedChanges[string, uint]{
		median: 31,
		set: AuthoritySet[string, uint]{
			CurrentAuthorities:     setA,
			SetID:                  3,
			PendingStandardChanges: NewChangeTree[string, uint](),
			PendingForcedChanges:   []PendingChange[string, uint]{},
			AuthoritySetChanges:    expChanges,
		},
	}
	resForced, err := authorities.applyForcedChanges(
		hashD,
		45,
		staticIsDescendentOf[string](true),
		nil)
	require.NoError(t, err)
	require.NotNil(t, resForced)
	require.Equal(t, exp, *resForced)
}

func TestNextChangeWorks(t *testing.T) {
	currentAuthorities := pgrandpa.AuthorityList{
		pgrandpa.AuthorityIDWeight{
			AuthorityID:     newTestPublic(t, 1),
			AuthorityWeight: 1,
		},
	}

	authorities := AuthoritySet[string, uint]{
		CurrentAuthorities:     currentAuthorities,
		SetID:                  0,
		PendingStandardChanges: NewChangeTree[string, uint](),
		PendingForcedChanges:   []PendingChange[string, uint]{},
		AuthoritySetChanges:    AuthoritySetChanges[uint]{},
	}

	finalisedKind := Finalized{}
	delayKindFinalized := newDelayKind[uint](finalisedKind)

	// We have three pending changes with 2 possible roots that are enacted
	// immediately on finality (i.e. standard changes).
	changeA0 := PendingChange[string, uint]{
		NextAuthorities: currentAuthorities,
		Delay:           0,
		CanonHeight:     5,
		CanonHash:       "hash_a0",
		DelayKind:       delayKindFinalized,
	}

	changeA1 := PendingChange[string, uint]{
		NextAuthorities: currentAuthorities,
		Delay:           0,
		CanonHeight:     10,
		CanonHash:       "hash_a1",
		DelayKind:       delayKindFinalized,
	}

	changeB := PendingChange[string, uint]{
		NextAuthorities: currentAuthorities,
		Delay:           0,
		CanonHeight:     4,
		CanonHash:       hashB,
		DelayKind:       delayKindFinalized,
	}

	// A0 (#5) <- A10 (#8) <- A1 (#10) <- best_a
	// B (#4) <- best_b
	isDescOf := isDescendentof(func(h1 string, h2 string) (bool, error) {
		if h1 == "hash_a0" && h2 == "hash_a1" ||
			h1 == "hash_a0" && h2 == hashB ||
			h1 == "hash_a1" && h2 == "best_a" ||
			h1 == "hash_a10" && h2 == "best_a" ||
			h1 == hashB && h2 == "best_b" {
			return true, nil
		} else {
			return false, nil
		}
	})

	// add the three pending changes
	err := authorities.addPendingChange(changeB, isDescOf)
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
	c, err := authorities.nextChange(hashB, isDescOf)
	require.NoError(t, err)
	require.Equal(t, expChange, c)

	// the earliest hashNumber at block `best_b` should be the hashNumber at B (#4)
	expChange = &hashNumber[string, uint]{
		hash:   hashB,
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
	finalisedKind2 := Best[uint]{0}
	delayKindFinalized2 := newDelayKind[uint](finalisedKind2)
	changeA10 := PendingChange[string, uint]{
		NextAuthorities: currentAuthorities,
		Delay:           0,
		CanonHeight:     8,
		CanonHash:       "hash_a10",
		DelayKind:       delayKindFinalized2,
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
	_, err := NewGenesisAuthoritySet[string, uint](nil)
	require.NotNil(t, err)
	_, err = NewAuthoritySet[string, uint](
		// []Authority[dummyAuthID]{},
		nil,
		0,
		NewChangeTree[string, uint](),
		nil,
		nil,
	)
	require.NotNil(t, err)

	invalidAuthoritiesWeight := pgrandpa.AuthorityList{
		pgrandpa.AuthorityIDWeight{
			AuthorityID:     newTestPublic(t, 1),
			AuthorityWeight: 5,
		},
		pgrandpa.AuthorityIDWeight{
			AuthorityID:     newTestPublic(t, 2),
			AuthorityWeight: 0,
		},
	}

	// authority weight of zero is invalid
	_, err = NewGenesisAuthoritySet[string, uint](invalidAuthoritiesWeight)
	require.NotNil(t, err)
	_, err = NewAuthoritySet[string, uint](
		invalidAuthoritiesWeight,
		0,
		NewChangeTree[string, uint](),
		nil,
		nil,
	)
	require.NotNil(t, err)

	authoritySet, err := NewGenesisAuthoritySet[string, uint](pgrandpa.AuthorityList{
		pgrandpa.AuthorityIDWeight{
			AuthorityID:     newTestPublic(t, 1),
			AuthorityWeight: 5,
		},
	})
	require.NoError(t, err)

	finalisedKind := Finalized{}
	delayKindFinalized := newDelayKind[uint](finalisedKind)
	invalidChangeEmptyAuthorities := PendingChange[string, uint]{
		NextAuthorities: nil,
		Delay:           10,
		CanonHeight:     5,
		CanonHash:       "",
		DelayKind:       delayKindFinalized,
	}

	// pending hashNumber contains an empty authority set
	err = authoritySet.addPendingChange(invalidChangeEmptyAuthorities, staticIsDescendentOf[string](false))
	require.ErrorIs(t, err, errInvalidAuthoritySet)

	delayKind := Best[uint]{0}
	delayKindBest := newDelayKind[uint](delayKind)

	invalidChangeAuthoritiesWeight := PendingChange[string, uint]{
		NextAuthorities: invalidAuthoritiesWeight,
		Delay:           10,
		CanonHeight:     5,
		CanonHash:       "",
		DelayKind:       delayKindBest,
	}

	// pending hashNumber contains an authority set
	// where one authority has weight of 0
	err = authoritySet.addPendingChange(invalidChangeAuthoritiesWeight, staticIsDescendentOf[string](false))
	require.ErrorIs(t, err, errInvalidAuthoritySet)
}

func TestCleanUpStaleForcedChangesWhenApplyingStandardChange(t *testing.T) {
	currentAuthorities := pgrandpa.AuthorityList{
		pgrandpa.AuthorityIDWeight{
			AuthorityID:     newTestPublic(t, 1),
			AuthorityWeight: 1,
		},
	}

	authorities := AuthoritySet[string, uint]{
		CurrentAuthorities:     currentAuthorities,
		SetID:                  0,
		PendingStandardChanges: NewChangeTree[string, uint](),
		PendingForcedChanges:   []PendingChange[string, uint]{},
		AuthoritySetChanges:    AuthoritySetChanges[uint]{},
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
				NextAuthorities: currentAuthorities,
				Delay:           0,
				CanonHeight:     canonHeight,
				CanonHash:       canonHash,
				DelayKind:       delayKindBest,
			}
		} else {
			delayKind := Finalized{}
			delayKindFinalized := newDelayKind[uint](delayKind)
			change = PendingChange[string, uint]{
				NextAuthorities: currentAuthorities,
				Delay:           0,
				CanonHeight:     canonHeight,
				CanonHash:       canonHash,
				DelayKind:       delayKindFinalized,
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
	_, err := authorities.applyStandardChanges("A", 5, isDescOf, nil)
	require.NoError(t, err)
	require.Equal(t, 6, len(authorities.pendingChanges()))

	// same for B
	_, err = authorities.applyStandardChanges("B", 10, isDescOf, nil)
	require.NoError(t, err)
	require.Equal(t, 5, len(authorities.pendingChanges()))

	// finalising C2 should clear all forced changes
	_, err = authorities.applyStandardChanges("C2", 15, isDescOf, nil)
	require.NoError(t, err)
	require.Equal(t, 0, len(authorities.PendingForcedChanges))
}

func TestCleanUpStaleForcedChangesWhenApplyingStandardChangeAlternateCase(t *testing.T) {
	currentAuthorities := pgrandpa.AuthorityList{
		pgrandpa.AuthorityIDWeight{
			AuthorityID:     newTestPublic(t, 1),
			AuthorityWeight: 1,
		},
	}

	authorities := AuthoritySet[string, uint]{
		CurrentAuthorities:     currentAuthorities,
		SetID:                  0,
		PendingStandardChanges: NewChangeTree[string, uint](),
		PendingForcedChanges:   []PendingChange[string, uint]{},
		AuthoritySetChanges:    AuthoritySetChanges[uint]{},
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
				NextAuthorities: currentAuthorities,
				Delay:           0,
				CanonHeight:     canonHeight,
				CanonHash:       canonHash,
				DelayKind:       delayKindBest,
			}
		} else {
			delayKind := Finalized{}
			delayKindFinalized := newDelayKind[uint](delayKind)
			change = PendingChange[string, uint]{
				NextAuthorities: currentAuthorities,
				Delay:           0,
				CanonHeight:     canonHeight,
				CanonHash:       canonHash,
				DelayKind:       delayKindFinalized,
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
	_, err := authorities.applyStandardChanges("A", 5, isDescOf, nil)
	require.NoError(t, err)
	require.Equal(t, 6, len(authorities.pendingChanges()))

	// same for B
	_, err = authorities.applyStandardChanges("B", 10, isDescOf, nil)
	require.NoError(t, err)
	require.Equal(t, 5, len(authorities.pendingChanges()))

	// finalising C0 should clear all forced changes but D
	_, err = authorities.applyStandardChanges("C0", 15, isDescOf, nil)
	require.NoError(t, err)
	require.Equal(t, 1, len(authorities.PendingForcedChanges))
	require.Equal(t, "D", authorities.PendingForcedChanges[0].CanonHash)
}

func assertExpectedSet(t *testing.T, authSetID authoritySetChangeID, expected setIDNumber[uint]) {
	t.Helper()
	switch val := authSetID.(type) {
	case authoritySetChangeIDSet[uint]:
		require.Equal(t, expected, val.inner)
	default:
		t.FailNow()
	}
}

func assertUnknown(t *testing.T, authSetID authoritySetChangeID) {
	t.Helper()
	isUnknown := false
	switch authSetID.(type) {
	case authoritySetChangeIDUnknown:
		isUnknown = true
	}
	require.True(t, isUnknown)
}

func assertLatest(t *testing.T, authSetID authoritySetChangeID) {
	t.Helper()
	isLatest := false
	switch authSetID.(type) {
	case authoritySetChangeIDLatest:
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
		SetID:       2,
		BlockNumber: 101,
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
		SetID:       0,
		BlockNumber: 41,
	}

	expChange1 := setIDNumber[uint]{
		SetID:       1,
		BlockNumber: 81,
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
		SetID:       3,
		BlockNumber: 81,
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
			SetID:       1,
			BlockNumber: 41,
		},
		setIDNumber[uint]{
			SetID:       2,
			BlockNumber: 81,
		},
		setIDNumber[uint]{
			SetID:       3,
			BlockNumber: 121,
		},
	}

	iterSet = authoritySetChanges.IterFrom(40)
	require.Equal(t, expectedChanges, iterSet)

	expectedChanges = &AuthoritySetChanges[uint]{
		setIDNumber[uint]{
			SetID:       2,
			BlockNumber: 81,
		},
		setIDNumber[uint]{
			SetID:       3,
			BlockNumber: 121,
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
		authorities pgrandpa.AuthorityList
	}
	tests := []struct {
		name string
		args args
		exp  bool
	}{
		{
			name: "nilAuthorities",
			args: args{
				authorities: nil,
			},
			exp: true,
		},
		{
			name: "emptyAuthorities",
			args: args{
				authorities: pgrandpa.AuthorityList{},
			},
			exp: true,
		},
		{
			name: "invalidAuthoritiesWeight",
			args: args{
				authorities: pgrandpa.AuthorityList{
					{
						AuthorityWeight: 0,
					},
				},
			},
			exp: true,
		},
		{
			name: "validAuthorityList",
			args: args{
				authorities: pgrandpa.AuthorityList{
					{
						AuthorityWeight: 1,
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

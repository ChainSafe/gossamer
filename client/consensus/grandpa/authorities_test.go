// Copyright 2023 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only
package grandpa

import (
	"github.com/ChainSafe/gossamer/dot/types"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/lib/crypto/ed25519"
	"github.com/stretchr/testify/require"
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

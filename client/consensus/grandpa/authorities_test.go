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

//fn is_descendent_of<A, F>(f: F) -> impl Fn(&A, &A) -> Result<bool, std::io::Error>
//where
//F: Fn(&A, &A) -> bool,
//{
//move |base, hash| Ok(f(base, hash))
//}

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

	//// ordered by subtree depth
	//assert_eq!(
	//	authorities.pending_changes().collect::<Vec<_>>(),
	//	vec![&change_a, &change_c, &change_b, &change_e, &change_d],

	//fmt.Println(common.BytesToHash([]byte("hash_a")))
	//fmt.Println(common.BytesToHash([]byte("hash_b")))
	//fmt.Println(common.BytesToHash([]byte("hash_c")))
	//
	//fmt.Println("ok")

	// For now just print these, standard changes seem to be printed in order
	_ = authorities.PendingChanges()

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

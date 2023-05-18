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
		pendingStandardChanges: ChangeTree{},
		pendingForcedChanges:   []PendingChange{},
		authoritySetChanges:    AuthoritySetChanges{},
	}

	err = authorities.addPendingChange(pendingChange1, staticIsDescendentOf(false))
	require.NoError(t, err)

	err = authorities.addPendingChange(pendingChange2, staticIsDescendentOf(false))
	require.NoError(t, err)

	require.Equal(t, uint(2), authorities.pendingStandardChanges.count)

	// TODO use ForkTree to assert test cases
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

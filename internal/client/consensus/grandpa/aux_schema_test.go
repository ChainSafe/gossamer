// Copyright 2023 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package grandpa

import (
	"slices"
	"testing"

	"github.com/ChainSafe/gossamer/internal/client/api"
	primitives "github.com/ChainSafe/gossamer/internal/primitives/consensus/grandpa"
	forktree "github.com/ChainSafe/gossamer/internal/utils/fork-tree"
	grandpa "github.com/ChainSafe/gossamer/pkg/finality-grandpa"
	"github.com/ChainSafe/gossamer/pkg/scale"
	"github.com/stretchr/testify/require"
)

func write(store api.AuxStore) writeAux {
	return func(insertions []api.KeyValue) error {
		return store.InsertAux(insertions, nil)
	}
}

type dummyStore []api.KeyValue

func (client *dummyStore) InsertAux(insert []api.KeyValue, deleted [][]byte) error {
	for _, val := range insert {
		*client = append(*client, val)
	}
	newStore := make(dummyStore, 0)
	for _, value := range *client {
		// append if not in deleted
		found := false
		for _, del := range deleted {
			if slices.Equal(value.Key, del) {
				found = true
			}
		}
		if !found {
			newStore = append(newStore, value)
		}
	}

	*client = newStore
	return nil

}

func (client *dummyStore) GetAux(key []byte) ([]byte, error) {
	for _, value := range *client {
		if slices.Equal(value.Key, key) {
			return value.Value, nil
		}
	}
	return nil, nil
}

var _ api.AuxStore = &dummyStore{}

func newDummyStore() *dummyStore {
	return &dummyStore{}
}

func TestDummyStore(t *testing.T) {
	store := newDummyStore()
	insert := []api.KeyValue{
		{Key: authoritySetKey, Value: scale.MustMarshal([]byte{1})},
		{Key: setStateKey, Value: scale.MustMarshal([]byte{2})},
	}
	err := store.InsertAux(insert, nil)
	require.NoError(t, err)
	require.True(t, len(*store) == 2)

	del := [][]byte{setStateKey}
	err = store.InsertAux(nil, del)
	require.NoError(t, err)
	require.True(t, len(*store) == 1)

	data, err := store.GetAux(authoritySetKey)
	require.NoError(t, err)
	require.NotNil(t, data)
	require.Equal(t, scale.MustMarshal([]byte{1}), data)

	data, err = store.GetAux(setStateKey)
	require.NoError(t, err)
	require.Nil(t, data)
}

type TestString string

func (ts TestString) Bytes() []byte {
	return []byte(ts)
}

// String returns a unique string representation of the hash
func (ts TestString) String() string {
	return string(ts)
}

// Length return the byte length of the hash
func (ts TestString) Length() int {
	return len(ts)
}

func TestUpdateAuthoritySet(t *testing.T) {
	// Test no new set case
	store := newDummyStore()
	authorities := AuthoritySet[TestString, uint]{
		SetID:                  1,
		PendingStandardChanges: forktree.NewForkTree[TestString, uint, PendingChange[TestString, uint]](),
	}

	err := updateAuthoritySet[TestString, uint](authorities, nil, write(store))
	require.NoError(t, err)

	encData, err := store.GetAux(authoritySetKey)
	require.NoError(t, err)
	require.NotNil(t, encData)

	newAuthorities := AuthoritySet[TestString, uint]{
		PendingStandardChanges: forktree.NewForkTree[TestString, uint, PendingChange[TestString, uint]](),
	}
	err = scale.Unmarshal(encData, &newAuthorities)
	require.NoError(t, err)
	require.Equal(t, authorities, newAuthorities)

	// New set case
	store = newDummyStore()
	authorities = AuthoritySet[TestString, uint]{
		SetID:                  1,
		PendingStandardChanges: forktree.NewForkTree[TestString, uint, PendingChange[TestString, uint]](),
	}

	newAuthSet := &newAuthoritySet[TestString, uint]{
		CanonNumber: 4,
		SetID:       2,
	}

	err = updateAuthoritySet[TestString, uint](authorities, newAuthSet, write(store))
	require.NoError(t, err)

	encData, err = store.GetAux(authoritySetKey)
	require.NoError(t, err)
	require.NotNil(t, encData)

	newAuthorities = AuthoritySet[TestString, uint]{
		PendingStandardChanges: forktree.NewForkTree[TestString, uint, PendingChange[TestString, uint]](),
	}
	err = scale.Unmarshal(encData, &newAuthorities)
	require.NoError(t, err)
	require.Equal(t, authorities, newAuthorities)

	encState, err := store.GetAux(setStateKey)
	require.NoError(t, err)
	require.NotNil(t, encState)

	genesisState := grandpa.HashNumber[TestString, uint]{
		Number: newAuthSet.CanonNumber,
	}
	setState := newVoterSetStateLive[TestString, uint](
		newAuthSet.SetID,
		authorities,
		genesisState,
	)
	require.NoError(t, err)
	setStateVDT := newVoterSetStateVDT[TestString, uint]()
	setStateVDT.inner = setState

	encodedVoterSet, err := scale.Marshal(*setStateVDT)
	require.NoError(t, err)
	require.Equal(t, encodedVoterSet, encState)
}

func TestWriteVoterSetState(t *testing.T) {
	store := newDummyStore()
	authorities := AuthoritySet[TestString, uint]{
		CurrentAuthorities:     primitives.AuthorityList{},
		SetID:                  1,
		PendingStandardChanges: forktree.NewForkTree[TestString, uint, PendingChange[TestString, uint]](),
		PendingForcedChanges:   []PendingChange[TestString, uint]{},
		AuthoritySetChanges:    AuthoritySetChanges[uint]{},
	}

	dummyHashNumber := grandpa.HashNumber[TestString, uint]{
		Hash:   "a",
		Number: 1,
	}

	completedRound := completedRound[TestString, uint]{
		Number: 1,
		State: grandpa.RoundState[TestString, uint]{
			PrevoteGHOST: &dummyHashNumber,
			Finalized:    &dummyHashNumber,
			Estimate:     &dummyHashNumber,
			Completable:  true,
		},
		Base: dummyHashNumber,
	}

	completedRounds := newCompletedRounds[TestString, uint](completedRound, 1, authorities)
	currentRounds := make(currentRounds[TestString, uint])

	liveState := voterSetStateLive[TestString, uint]{
		CompletedRounds: completedRounds,
		CurrentRounds:   currentRounds,
	}

	voterSetState := newVoterSetStateVDT[TestString, uint]()
	err := voterSetState.SetValue(liveState)
	require.NoError(t, err)
	require.NotNil(t, voterSetState)

	err = writeVoterSetState[TestString, uint](store, liveState)
	require.NoError(t, err)

	encVoterSet, err := scale.Marshal(*voterSetState)
	require.NoError(t, err)

	val, err := store.GetAux(setStateKey)
	require.NoError(t, err)
	require.NotNil(t, val)
	require.Equal(t, encVoterSet, val)
}

func TestWriteConcludedRound(t *testing.T) {
	store := newDummyStore()
	authorities := AuthoritySet[TestString, uint]{
		CurrentAuthorities:     primitives.AuthorityList{},
		SetID:                  1,
		PendingStandardChanges: forktree.NewForkTree[TestString, uint, PendingChange[TestString, uint]](),
		PendingForcedChanges:   []PendingChange[TestString, uint]{},
		AuthoritySetChanges:    AuthoritySetChanges[uint]{},
	}

	dummyHashNumber := grandpa.HashNumber[TestString, uint]{
		Hash:   "a",
		Number: 1,
	}

	completedRound := completedRound[TestString, uint]{
		Number: 1,
		State: grandpa.RoundState[TestString, uint]{
			PrevoteGHOST: &dummyHashNumber,
			Finalized:    &dummyHashNumber,
			Estimate:     &dummyHashNumber,
			Completable:  true,
		},
		Base: dummyHashNumber,
	}

	completedRounds := newCompletedRounds[TestString, uint](completedRound, 1, authorities)
	currentRounds := make(currentRounds[TestString, uint])

	liveState := voterSetStateLive[TestString, uint]{
		CompletedRounds: completedRounds,
		CurrentRounds:   currentRounds,
	}

	voterSetState := newVoterSetStateVDT[TestString, uint]()
	err := voterSetState.SetValue(liveState)
	require.NoError(t, err)
	require.NotNil(t, voterSetState)

	err = writeConcludedRound[TestString, uint](store, completedRound)
	require.NoError(t, err)

	key := concludedRounds
	encodedRoundNumber := scale.MustMarshal(completedRound.Number)
	key = append(key, encodedRoundNumber...)

	encRoundData := scale.MustMarshal(completedRound)

	val, err := store.GetAux(key)
	require.NoError(t, err)
	require.NotNil(t, val)
	require.Equal(t, encRoundData, val)
}

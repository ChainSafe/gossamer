// Copyright 2023 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package grandpa

import (
	"github.com/ChainSafe/gossamer/client/api"
	finalityGrandpa "github.com/ChainSafe/gossamer/pkg/finality-grandpa"
	"github.com/ChainSafe/gossamer/pkg/scale"
	"github.com/stretchr/testify/require"
	"golang.org/x/exp/slices"
	"testing"
)

func genesisAuthorities(auths []Authority[uint], err error) getGenesisAuthorities[uint] {
	return func() ([]Authority[uint], error) { return auths, err }
}

func write(store api.AuxStore) writeAux {
	return func(insertions []api.KeyValue) error {
		return store.Insert(insertions, nil)
	}
}

type dummyStore []api.KeyValue

func (client *dummyStore) Insert(insert []api.KeyValue, deleted []api.Key) error {
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

func (client *dummyStore) Get(key api.Key) (*[]byte, error) {
	for _, value := range *client {
		if slices.Equal(value.Key, key) {
			return &value.Value, nil
		}
	}
	return nil, nil
}

func newDummyStore(t *testing.T) *dummyStore {
	t.Helper()
	return &dummyStore{}
}

func TestDummyStore(t *testing.T) {
	store := newDummyStore(t)
	insert := []api.KeyValue{
		{authoritySetKey, scale.MustMarshal([]byte{1})},
		{setStateKey, scale.MustMarshal([]byte{2})},
	}
	err := store.Insert(insert, nil)
	require.NoError(t, err)
	require.True(t, len(*store) == 2)

	del := []api.Key{setStateKey}

	err = store.Insert(nil, del)
	require.NoError(t, err)
	require.True(t, len(*store) == 1)

	data, err := store.Get(authoritySetKey)
	require.NoError(t, err)
	require.NotNil(t, data)
	require.Equal(t, scale.MustMarshal([]byte{1}), *data)

	data, err = store.Get(setStateKey)
	require.NoError(t, err)
	require.Nil(t, data)
}

func TestLoadPersistentGenesis(t *testing.T) {
	// Test genesis case, call with nothing written then assert on db.Gets
	store := newDummyStore(t)
	genesisHash := "a"
	genesisNumber := uint(21)
	genesisAuths := []Authority[uint]{{
		Key:    key,
		Weight: 1,
	}}

	// Genesis Case
	persistentData, err := loadPersistent[string, uint, uint, uint](store, genesisHash, genesisNumber, genesisAuthorities(genesisAuths, nil))
	require.NoError(t, err)
	require.NotNil(t, persistentData)

	genesisSet, err := NewGenesisAuthoritySet[string, uint, uint](genesisAuths)
	require.NoError(t, err)

	state := finalityGrandpa.NewRoundState(finalityGrandpa.HashNumber[string, uint]{Hash: genesisHash, Number: genesisNumber})
	base := state.PrevoteGHOST
	genesisState, err := NewLiveVoterSetState[string, uint, uint, uint](0, *genesisSet, *base)
	require.NoError(t, err)

	require.Equal(t, persistentData.authoritySet.inner, *genesisSet)
	require.Equal(t, persistentData.setState.Inner.Inner, genesisState)

	// Assert db values
	encAuthData, err := store.Get(authoritySetKey)
	require.NoError(t, err)
	require.NotNil(t, encAuthData)

	encSetData, err := store.Get(setStateKey)
	require.NoError(t, err)
	require.NotNil(t, encSetData)

	require.Equal(t, scale.MustMarshal(*genesisSet), *encAuthData)
	require.Equal(t, scale.MustMarshal(genesisState), *encSetData)
}

func TestLoadPersistentNotGenesis(t *testing.T) {
	store := newDummyStore(t)
	genesisHash := "a"
	genesisNumber := uint(21)
	genesisAuths := []Authority[uint]{{
		Key:    key,
		Weight: 1,
	}}

	// Auth set and Set state both written
	genesisSet, err := NewGenesisAuthoritySet[string, uint, uint](genesisAuths)
	require.NoError(t, err)

	state := finalityGrandpa.NewRoundState(finalityGrandpa.HashNumber[string, uint]{Hash: genesisHash, Number: genesisNumber})
	base := state.PrevoteGHOST
	genesisState, err := NewLiveVoterSetState[string, uint, uint, uint](0, *genesisSet, *base)
	require.NoError(t, err)

	insert := []api.KeyValue{
		{authoritySetKey, scale.MustMarshal(*genesisSet)},
		{setStateKey, scale.MustMarshal(genesisState)},
	}

	err = store.Insert(insert, nil)
	require.NoError(t, err)
	persistentData, err := loadPersistent[string, uint, uint, uint](store, genesisHash, genesisNumber, genesisAuthorities(genesisAuths, nil))
	require.NoError(t, err)
	require.NotNil(t, persistentData)
	require.Equal(t, *genesisSet, persistentData.authoritySet.inner)

	expVal, err := genesisState.Value()
	require.NoError(t, err)
	actualVal, err := persistentData.setState.Inner.Inner.Value()
	require.NoError(t, err)
	require.Equal(t, expVal, actualVal)

	// Auth set written but not set state
	store = newDummyStore(t)
	insert = []api.KeyValue{
		{authoritySetKey, scale.MustMarshal(*genesisSet)},
	}

	err = store.Insert(insert, nil)
	require.NoError(t, err)
	persistentData, err = loadPersistent[string, uint, uint, uint](store, genesisHash, genesisNumber, genesisAuthorities(genesisAuths, nil))
	require.NoError(t, err)

	newState, err := NewLiveVoterSetState[string, uint, uint, uint](genesisSet.SetID, *genesisSet, *base)
	require.NoError(t, err)

	require.Equal(t, *genesisSet, persistentData.authoritySet.inner)
	expVal, err = newState.Value()
	require.NoError(t, err)
	actualVal, err = persistentData.setState.Inner.Inner.Value()
	require.NoError(t, err)
	require.Equal(t, expVal, actualVal)
}

func TestUpdateAuthoritySet(t *testing.T) {
	// Test no new set case
	store := newDummyStore(t)
	authorities := AuthoritySet[string, uint, uint]{
		SetID:                  1,
		PendingStandardChanges: NewChangeTree[string, uint, uint](),
	}

	err := UpdateAuthoritySet[string, uint, uint, uint](authorities, nil, write(store))
	require.NoError(t, err)

	encData, err := store.Get(authoritySetKey)
	require.NoError(t, err)
	require.NotNil(t, encData)

	newAuthorities := AuthoritySet[string, uint, uint]{
		PendingStandardChanges: NewChangeTree[string, uint, uint](),
	}
	err = scale.Unmarshal(*encData, &newAuthorities)
	require.NoError(t, err)
	require.Equal(t, authorities, newAuthorities)

	// New set case
	store = newDummyStore(t)
	authorities = AuthoritySet[string, uint, uint]{
		SetID:                  1,
		PendingStandardChanges: NewChangeTree[string, uint, uint](),
	}

	newAuthSet := &NewAuthoritySetStruct[string, uint, uint]{
		CanonNumber: 4,
		SetId:       2,
	}

	err = UpdateAuthoritySet[string, uint, uint, uint](authorities, newAuthSet, write(store))
	require.NoError(t, err)

	encData, err = store.Get(authoritySetKey)
	require.NoError(t, err)
	require.NotNil(t, encData)

	newAuthorities = AuthoritySet[string, uint, uint]{
		PendingStandardChanges: NewChangeTree[string, uint, uint](),
	}
	err = scale.Unmarshal(*encData, &newAuthorities)
	require.NoError(t, err)
	require.Equal(t, authorities, newAuthorities)

	encState, err := store.Get(setStateKey)
	require.NoError(t, err)
	require.NotNil(t, encState)

	genesisState := finalityGrandpa.HashNumber[string, uint]{
		Number: newAuthSet.CanonNumber,
	}

	setState, err := NewLiveVoterSetState[string, uint, uint, uint](uint64(newAuthSet.SetId), authorities, genesisState)
	require.NoError(t, err)

	encodedVoterSet, err := scale.Marshal(setState)
	require.NoError(t, err)
	require.Equal(t, encodedVoterSet, *encState)
}

func TestWriteVoterSetState(t *testing.T) {
	store := newDummyStore(t)
	authorities := AuthoritySet[string, uint, uint]{
		CurrentAuthorities:     []Authority[uint]{},
		SetID:                  1,
		PendingStandardChanges: NewChangeTree[string, uint, uint](),
		PendingForcedChanges:   []PendingChange[string, uint, uint]{},
		AuthoritySetChanges:    AuthoritySetChanges[uint]{},
	}

	dummyHashNumber := finalityGrandpa.HashNumber[string, uint]{
		Hash:   "a",
		Number: 1,
	}

	completedRound := completedRound[string, uint, uint, uint]{
		Number: 1,
		State: finalityGrandpa.RoundState[string, uint]{
			PrevoteGHOST: &dummyHashNumber,
			Finalized:    &dummyHashNumber,
			Estimate:     &dummyHashNumber,
			Completable:  true,
		},
		Base: dummyHashNumber,
	}

	completedRounds := NewCompletedRounds[string, uint](completedRound, 1, authorities)
	currentRounds := make(map[uint64]hasVoted[string, uint])

	liveState := voterSetStateLive[string, uint, uint, uint]{
		CompletedRounds: completedRounds,
		CurrentRounds:   currentRounds,
	}

	voterSetState := NewVoterSetState[string, uint, uint, uint]()
	err := voterSetState.Set(liveState)
	require.NoError(t, err)
	require.NotNil(t, voterSetState)

	err = WriteVoterSetState[string, uint](*voterSetState, write(store))
	require.NoError(t, err)

	encVoterSet, err := scale.Marshal(*voterSetState)
	require.NoError(t, err)

	val, err := store.Get(setStateKey)
	require.NoError(t, err)
	require.NotNil(t, val)
	require.Equal(t, encVoterSet, *val)
}

func TestWriteConcludedRound(t *testing.T) {
	store := newDummyStore(t)
	authorities := AuthoritySet[string, uint, uint]{
		CurrentAuthorities:     []Authority[uint]{},
		SetID:                  1,
		PendingStandardChanges: NewChangeTree[string, uint, uint](),
		PendingForcedChanges:   []PendingChange[string, uint, uint]{},
		AuthoritySetChanges:    AuthoritySetChanges[uint]{},
	}

	dummyHashNumber := finalityGrandpa.HashNumber[string, uint]{
		Hash:   "a",
		Number: 1,
	}

	completedRound := completedRound[string, uint, uint, uint]{
		Number: 1,
		State: finalityGrandpa.RoundState[string, uint]{
			PrevoteGHOST: &dummyHashNumber,
			Finalized:    &dummyHashNumber,
			Estimate:     &dummyHashNumber,
			Completable:  true,
		},
		Base: dummyHashNumber,
	}

	completedRounds := NewCompletedRounds[string, uint](completedRound, 1, authorities)
	currentRounds := make(map[uint64]hasVoted[string, uint])

	liveState := voterSetStateLive[string, uint, uint, uint]{
		CompletedRounds: completedRounds,
		CurrentRounds:   currentRounds,
	}

	voterSetState := NewVoterSetState[string, uint, uint, uint]()
	err := voterSetState.Set(liveState)
	require.NoError(t, err)
	require.NotNil(t, voterSetState)

	err = WriteConcludedRound[string, uint](completedRound, write(store))
	require.NoError(t, err)

	key := concludedRounds
	encodedRoundNumber := scale.MustMarshal(completedRound.Number)
	key = append(key, encodedRoundNumber...)

	encRoundData := scale.MustMarshal(completedRound)

	val, err := store.Get(key)
	require.NoError(t, err)
	require.NotNil(t, val)
	require.Equal(t, encRoundData, *val)
}

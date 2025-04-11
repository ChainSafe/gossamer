// Copyright 2023 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package grandpa

import (
	"slices"
	"testing"

	"github.com/ChainSafe/gossamer/internal/client/api"
	primitives "github.com/ChainSafe/gossamer/internal/primitives/consensus/grandpa"
	grandpa "github.com/ChainSafe/gossamer/pkg/finality-grandpa"
	"github.com/ChainSafe/gossamer/pkg/scale"
	"github.com/stretchr/testify/require"
)

// func genesisAuthorities(auths primitives.AuthorityList, err error) getGenesisAuthorities {
// 	return func() (primitives.AuthorityList, error) { return auths, err }
// }

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

// func TestLoadPersistentGenesis(t *testing.T) {
// 	// Test genesis case, call with nothing written then assert on db.Gets
// 	store := newDummyStore()
// 	genesisHash := "a"
// 	genesisNumber := uint(21)
// 	genesisAuths := primitives.AuthorityList{{
// 		AuthorityID:     newTestPublic(t, 1),
// 		AuthorityWeight: 1,
// 	}}

// 	// Genesis Case
// 	persistentData, err := loadPersistent[TestString, uint](
// 		store,
// 		genesisHash,
// 		genesisNumber,
// 		genesisAuthorities(genesisAuths, nil))
// 	require.NoError(t, err)
// 	require.NotNil(t, persistentData)

// 	genesisSet, err := NewGenesisAuthoritySet[TestString, uint](genesisAuths)
// 	require.NoError(t, err)

// 	state := grandpa.NewRoundState(grandpa.HashNumber[TestString, uint]{
// 		Hash:   genesisHash,
// 		Number: genesisNumber})
// 	base := state.PrevoteGHOST
// 	genesisState, err := NewLiveVoterSetState[TestString, uint](0, *genesisSet, *base)
// 	require.NoError(t, err)

// 	require.Equal(t, persistentData.authoritySet.inner, *genesisSet)
// 	require.Equal(t, persistentData.setState.Inner.Inner, genesisState)

// 	// Assert db values
// 	encAuthData, err := store.GetAux(authoritySetKey)
// 	require.NoError(t, err)
// 	require.NotNil(t, encAuthData)

// 	encSetData, err := store.GetAux(setStateKey)
// 	require.NoError(t, err)
// 	require.NotNil(t, encSetData)

// 	require.Equal(t, scale.MustMarshal(*genesisSet), *encAuthData)
// 	require.Equal(t, scale.MustMarshal(genesisState), *encSetData)
// }

// func TestLoadPersistentNotGenesis(t *testing.T) {
// 	store := newDummyStore()
// 	genesisHash := "a"
// 	genesisNumber := uint(21)
// 	genesisAuths := primitives.AuthorityList{{
// 		AuthorityID:     newTestPublic(t, 1),
// 		AuthorityWeight: 1,
// 	}}

// 	// Auth set and Set state both written
// 	genesisSet, err := NewGenesisAuthoritySet[TestString, uint](genesisAuths)
// 	require.NoError(t, err)

// 	state := grandpa.NewRoundState(grandpa.HashNumber[TestString, uint]{
// 		Hash:   genesisHash,
// 		Number: genesisNumber})
// 	base := state.PrevoteGHOST
// 	genesisState, err := NewLiveVoterSetState[TestString, uint](0, *genesisSet, *base)
// 	require.NoError(t, err)

// 	insert := []api.KeyValue{
// 		{Key: authoritySetKey, Value: scale.MustMarshal(*genesisSet)},
// 		{Key: setStateKey, Value: scale.MustMarshal(genesisState)},
// 	}

// 	err = store.InsertAux(insert, nil)
// 	require.NoError(t, err)
// 	persistentData, err := loadPersistent[TestString, uint](
// 		store,
// 		genesisHash,
// 		genesisNumber,
// 		genesisAuthorities(genesisAuths, nil))
// 	require.NoError(t, err)
// 	require.NotNil(t, persistentData)
// 	require.Equal(t, *genesisSet, persistentData.authoritySet.inner)

// 	expVal, err := genesisState.Value()
// 	require.NoError(t, err)
// 	actualVal, err := persistentData.setState.Inner.Inner.Value()
// 	require.NoError(t, err)
// 	require.Equal(t, expVal, actualVal)

// 	// Auth set written but not set state
// 	store = newDummyStore()
// 	insert = []api.KeyValue{
// 		{Key: authoritySetKey, Value: scale.MustMarshal(*genesisSet)},
// 	}

// 	err = store.InsertAux(insert, nil)
// 	require.NoError(t, err)
// 	persistentData, err = loadPersistent[TestString, uint](
// 		store,
// 		genesisHash,
// 		genesisNumber,
// 		genesisAuthorities(genesisAuths, nil))
// 	require.NoError(t, err)

// 	newState, err := NewLiveVoterSetState[TestString, uint](genesisSet.SetID, *genesisSet, *base)
// 	require.NoError(t, err)

// 	require.Equal(t, *genesisSet, persistentData.authoritySet.inner)
// 	expVal, err = newState.Value()
// 	require.NoError(t, err)
// 	actualVal, err = persistentData.setState.Inner.Inner.Value()
// 	require.NoError(t, err)
// 	require.Equal(t, expVal, actualVal)
// }

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
		PendingStandardChanges: NewChangeTree[TestString, uint](),
	}

	err := UpdateAuthoritySet[TestString, uint](authorities, nil, write(store))
	require.NoError(t, err)

	encData, err := store.GetAux(authoritySetKey)
	require.NoError(t, err)
	require.NotNil(t, encData)

	newAuthorities := AuthoritySet[TestString, uint]{
		PendingStandardChanges: NewChangeTree[TestString, uint](),
	}
	err = scale.Unmarshal(encData, &newAuthorities)
	require.NoError(t, err)
	require.Equal(t, authorities, newAuthorities)

	// New set case
	store = newDummyStore()
	authorities = AuthoritySet[TestString, uint]{
		SetID:                  1,
		PendingStandardChanges: NewChangeTree[TestString, uint](),
	}

	newAuthSet := &newAuthoritySet[TestString, uint]{
		CanonNumber: 4,
		SetID:       2,
	}

	err = UpdateAuthoritySet[TestString, uint](authorities, newAuthSet, write(store))
	require.NoError(t, err)

	encData, err = store.GetAux(authoritySetKey)
	require.NoError(t, err)
	require.NotNil(t, encData)

	newAuthorities = AuthoritySet[TestString, uint]{
		PendingStandardChanges: NewChangeTree[TestString, uint](),
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
		PendingStandardChanges: NewChangeTree[TestString, uint](),
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
		PendingStandardChanges: NewChangeTree[TestString, uint](),
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

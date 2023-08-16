// Copyright 2023 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package grandpa

import (
	"github.com/ChainSafe/gossamer/lib/crypto/ed25519"
	finalityGrandpa "github.com/ChainSafe/gossamer/pkg/finality-grandpa"
	"github.com/ChainSafe/gossamer/pkg/scale"
	"github.com/stretchr/testify/require"
	"golang.org/x/exp/slices"
	"testing"
)

type dummyStore []KeyValue

func (client *dummyStore) Insert(insert []KeyValue, deleted [][]byte) error {
	for _, val := range insert {
		*client = append(*client, val)
	}
	newStore := make(dummyStore, 0)
	for _, value := range *client {
		// append if not in deleted
		found := false
		for _, del := range deleted {
			if slices.Equal(value.key, del) {
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

func (client *dummyStore) Get(key []byte) (*[]byte, error) {
	for _, value := range *client {
		if slices.Equal(value.key, key) {
			return &value.value, nil
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
	insert := []KeyValue{
		{AUTHORITY_SET_KEY, scale.MustMarshal([]byte{1})},
		{SET_STATE_KEY, scale.MustMarshal([]byte{2})},
	}
	err := store.Insert(insert, nil)
	require.NoError(t, err)
	require.True(t, len(*store) == 2)

	del := [][]byte{SET_STATE_KEY}

	err = store.Insert(nil, del)
	require.NoError(t, err)
	require.True(t, len(*store) == 1)

	data, err := store.Get(AUTHORITY_SET_KEY)
	require.NoError(t, err)
	require.NotNil(t, data)
	require.Equal(t, scale.MustMarshal([]byte{1}), *data)

	data, err = store.Get(SET_STATE_KEY)
	require.NoError(t, err)
	require.Nil(t, data)
}

func TestLoadPersistentGenesis(t *testing.T) {
	// Test genesis case, call with nothing written then assert on db.Gets
	pub, err := ed25519.NewPublicKey([]byte{1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1})
	require.NoError(t, err)

	store := newDummyStore(t)
	genesisHash := Hash{1}
	genesisNumber := uint(21)
	genesisAuthorities := AuthorityList{Authority{
		Key:    *pub,
		Weight: 1,
	}}

	// Genesis Case
	persistentData, err := loadPersistent[Hash, uint](store, genesisHash, genesisNumber, genesisAuthorities)
	require.NoError(t, err)
	require.NotNil(t, persistentData)

	genesisSet := NewGenesisAuthoritySet[Hash, uint](genesisAuthorities)

	state := finalityGrandpa.NewRoundState(finalityGrandpa.HashNumber[Hash, uint]{Hash: genesisHash, Number: genesisNumber})
	base := state.PrevoteGHOST
	voterSetState := NewVoterSetState[Hash, uint]()
	genesisState, err := voterSetState.Live(0, *genesisSet, *base)
	require.NoError(t, err)

	require.Equal(t, persistentData.authoritySet.inner, *genesisSet)
	require.Equal(t, persistentData.setState.inner, genesisState)

	// Assert db values
	encAuthData, err := store.Get(AUTHORITY_SET_KEY)
	require.NoError(t, err)
	require.NotNil(t, encAuthData)

	encSetData, err := store.Get(SET_STATE_KEY)
	require.NoError(t, err)
	require.NotNil(t, encSetData)

	require.Equal(t, scale.MustMarshal(*genesisSet), *encAuthData)
	require.Equal(t, scale.MustMarshal(genesisState), *encSetData)
}

func TestLoadPersistentNotGenesis(t *testing.T) {
	pub, err := ed25519.NewPublicKey([]byte{1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1})
	require.NoError(t, err)

	store := newDummyStore(t)
	genesisHash := Hash{1}
	genesisNumber := uint(21)
	genesisAuthorities := AuthorityList{Authority{
		Key:    *pub,
		Weight: 1,
	}}

	// Auth set and Set state both written
	genesisSet := NewGenesisAuthoritySet[Hash, uint](genesisAuthorities)

	state := finalityGrandpa.NewRoundState(finalityGrandpa.HashNumber[Hash, uint]{Hash: genesisHash, Number: genesisNumber})
	base := state.PrevoteGHOST
	voterSetState := NewVoterSetState[Hash, uint]()
	genesisState, err := voterSetState.Live(0, *genesisSet, *base)
	require.NoError(t, err)

	insert := []KeyValue{
		{AUTHORITY_SET_KEY, scale.MustMarshal(*genesisSet)},
		{SET_STATE_KEY, scale.MustMarshal(genesisState)},
	}

	err = store.Insert(insert, nil)
	require.NoError(t, err)
	persistentData, err := loadPersistent[Hash, uint](store, genesisHash, genesisNumber, genesisAuthorities)
	require.NoError(t, err)
	require.NotNil(t, persistentData)
	require.Equal(t, *genesisSet, persistentData.authoritySet.inner)

	expVal, err := genesisState.Value()
	require.NoError(t, err)
	actualVal, err := persistentData.setState.inner.Value()
	require.NoError(t, err)
	require.Equal(t, expVal, actualVal)

	// Auth set written but not set state
	store = newDummyStore(t)
	insert = []KeyValue{
		{AUTHORITY_SET_KEY, scale.MustMarshal(*genesisSet)},
	}

	err = store.Insert(insert, nil)
	require.NoError(t, err)
	persistentData, err = loadPersistent[Hash, uint](store, genesisHash, genesisNumber, genesisAuthorities)
	require.NoError(t, err)

	newVoterSetState := NewVoterSetState[Hash, uint]()
	newState, err := newVoterSetState.Live(genesisSet.SetID, *genesisSet, *base)
	require.NoError(t, err)

	require.Equal(t, *genesisSet, persistentData.authoritySet.inner)
	expVal, err = newState.Value()
	require.NoError(t, err)
	actualVal, err = persistentData.setState.inner.Value()
	require.NoError(t, err)
	require.Equal(t, expVal, actualVal)
}

func TestUpdateAuthoritySet(t *testing.T) {
	// Test no new set case
	store := newDummyStore(t)
	authorities := AuthoritySet[Hash, uint]{
		SetID:                  1,
		PendingStandardChanges: NewChangeTree[Hash, uint](),
	}

	err := UpdateAuthoritySet[Hash, uint](store, authorities, nil)
	require.NoError(t, err)

	encData, err := store.Get(AUTHORITY_SET_KEY)
	require.NoError(t, err)
	require.NotNil(t, encData)

	newAuthorities := AuthoritySet[Hash, uint]{
		PendingStandardChanges: NewChangeTree[Hash, uint](),
	}
	err = scale.Unmarshal(*encData, &newAuthorities)
	require.NoError(t, err)
	require.Equal(t, authorities, newAuthorities)

	// New set case
	store = newDummyStore(t)
	authorities = AuthoritySet[Hash, uint]{
		SetID:                  1,
		PendingStandardChanges: NewChangeTree[Hash, uint](),
	}

	newAuthSet := &NewAuthoritySetStruct[Hash, uint]{
		CanonNumber: 4,
		SetId:       2,
	}

	err = UpdateAuthoritySet[Hash, uint](store, authorities, newAuthSet)
	require.NoError(t, err)

	encData, err = store.Get(AUTHORITY_SET_KEY)
	require.NoError(t, err)
	require.NotNil(t, encData)

	newAuthorities = AuthoritySet[Hash, uint]{
		PendingStandardChanges: NewChangeTree[Hash, uint](),
	}
	err = scale.Unmarshal(*encData, &newAuthorities)
	require.NoError(t, err)
	require.Equal(t, authorities, newAuthorities)

	encState, err := store.Get(SET_STATE_KEY)
	require.NoError(t, err)
	require.NotNil(t, encState)

	genesisState := finalityGrandpa.HashNumber[Hash, uint]{
		Number: newAuthSet.CanonNumber,
	}

	voterSetState := NewVoterSetState[Hash, uint]()
	setState, err := voterSetState.Live(uint64(newAuthSet.SetId), authorities, genesisState)
	require.NoError(t, err)

	encodedVoterSet, err := scale.Marshal(setState)
	require.NoError(t, err)
	require.Equal(t, encodedVoterSet, *encState)
}

func TestWriteVoterSetState(t *testing.T) {
	store := newDummyStore(t)
	authorities := AuthoritySet[Hash, uint]{
		CurrentAuthorities:     AuthorityList{},
		SetID:                  1,
		PendingStandardChanges: NewChangeTree[Hash, uint](),
		PendingForcedChanges:   []PendingChange[Hash, uint]{},
		AuthoritySetChanges:    AuthoritySetChanges[uint]{},
	}

	dummyHashNumber := finalityGrandpa.HashNumber[Hash, uint]{
		Hash:   bytesToHash([]byte{1}),
		Number: 1,
	}

	completedRound := &CompletedRound[Hash, uint]{
		Number: 1,
		State: finalityGrandpa.RoundState[Hash, uint]{
			PrevoteGHOST: &dummyHashNumber,
			Finalized:    &dummyHashNumber,
			Estimate:     &dummyHashNumber,
			Completable:  true,
		},
		Base: dummyHashNumber,
	}

	completedRounds := NewCompletedRounds[Hash, uint](completedRound, 1, authorities)
	currentRounds := make(map[uint64]HasVoted[Hash, uint])

	liveState := Live[Hash, uint]{
		CompletedRounds: completedRounds,
		CurrentRounds:   currentRounds,
	}

	voterSetState := NewVoterSetState[Hash, uint]()
	err := voterSetState.Set(liveState)
	require.NoError(t, err)
	require.NotNil(t, voterSetState)

	err = WriteVoterSetState[Hash, uint](store, *voterSetState)
	require.NoError(t, err)

	encVoterSet, err := scale.Marshal(*voterSetState)
	require.NoError(t, err)

	val, err := store.Get(SET_STATE_KEY)
	require.NoError(t, err)
	require.NotNil(t, val)
	require.Equal(t, encVoterSet, *val)
}

func TestWriteConcludedRound(t *testing.T) {
	store := newDummyStore(t)
	authorities := AuthoritySet[Hash, uint]{
		CurrentAuthorities:     AuthorityList{},
		SetID:                  1,
		PendingStandardChanges: NewChangeTree[Hash, uint](),
		PendingForcedChanges:   []PendingChange[Hash, uint]{},
		AuthoritySetChanges:    AuthoritySetChanges[uint]{},
	}

	dummyHashNumber := finalityGrandpa.HashNumber[Hash, uint]{
		Hash:   bytesToHash([]byte{1}),
		Number: 1,
	}

	completedRound := &CompletedRound[Hash, uint]{
		Number: 1,
		State: finalityGrandpa.RoundState[Hash, uint]{
			PrevoteGHOST: &dummyHashNumber,
			Finalized:    &dummyHashNumber,
			Estimate:     &dummyHashNumber,
			Completable:  true,
		},
		Base: dummyHashNumber,
	}

	completedRounds := NewCompletedRounds[Hash, uint](completedRound, 1, authorities)
	currentRounds := make(map[uint64]HasVoted[Hash, uint])

	liveState := Live[Hash, uint]{
		CompletedRounds: completedRounds,
		CurrentRounds:   currentRounds,
	}

	voterSetState := NewVoterSetState[Hash, uint]()
	err := voterSetState.Set(liveState)
	require.NoError(t, err)
	require.NotNil(t, voterSetState)

	err = WriteConcludedRound[Hash, uint](store, *completedRound)
	require.NoError(t, err)

	key := CONCLUDED_ROUNDS
	encodedRoundNumber := scale.MustMarshal(completedRound.Number)
	key = append(key, encodedRoundNumber...)

	encRoundData := scale.MustMarshal(*completedRound)

	val, err := store.Get(key)
	require.NoError(t, err)
	require.NotNil(t, val)
	require.Equal(t, encRoundData, *val)
}

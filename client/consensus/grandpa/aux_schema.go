// Copyright 2023 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package grandpa

import (
	"errors"
	"github.com/ChainSafe/gossamer/pkg/scale"
	"golang.org/x/exp/constraints"

	"github.com/ChainSafe/gossamer/client/api"
	grandpa "github.com/ChainSafe/gossamer/pkg/finality-grandpa"
)

var (
	setStateKey       = []byte("grandpa_completed_round")
	concludedRounds   = []byte("grandpa_concluded_rounds")
	authoritySetKey   = []byte("grandpa_voters")
	bestJustification = []byte("grandpa_best_justification")

	errValueNotFound = errors.New("value not found")
)

type writeAux func(insertions []api.KeyValue) error

type getGenesisAuthorities func() ([]Authority, error)

type persistentData[H comparable, N constraints.Unsigned] struct {
	authoritySet SharedAuthoritySet[H, N]
	setState     SharedVoterSetState[H, N]
}

// TODO determine if I can actually do this
func loadDecoded(store api.AuxStore, key []byte, destination any) (any, error) {
	// This causes a panic: reflect.Value.Convert: value of type interface {} cannot be converted to type scale.VaryingDataType
	encodedValue, err := store.Get(key)
	if err != nil {
		return nil, err
	}

	if encodedValue != nil {
		err = scale.Unmarshal(*encodedValue, &destination)
		if err != nil {
			return nil, err
		}

		return destination, nil
	}

	return nil, errValueNotFound
}

//func loadPersistent[H comparable, N constraints.Unsigned](store api.AuxStore, genesisHash H, genesisNumber N, genesisAuths getGenesisAuthorities) (*persistentData[H, N], error) {
//	genesis := grandpa.HashNumber[H, N]{Hash: genesisHash, Number: genesisNumber}
//	makeGenesisRound := grandpa.NewRoundState[H, N]
//
//	authSetOld, err := loadDecoded(store, authoritySetKey, AuthoritySet[H, N]{})
//	if err != nil && !errors.Is(err, errValueNotFound) {
//		return nil, err
//	}
//
//	authSet := authSetOld.(AuthoritySet[H, N])
//
//	if !errors.Is(err, errValueNotFound) {
//		// This fails currently
//		setStateOld, err := loadDecoded(store, setStateKey, voterSetState[H, N]{})
//		if err != nil && !errors.Is(err, errValueNotFound) {
//			return nil, err
//		}
//
//		setState := setStateOld.(voterSetState[H, N])
//
//		if !errors.Is(err, errValueNotFound) {
//			state := makeGenesisRound(genesis)
//			base := state.PrevoteGHOST
//			if base != nil {
//				state, err := NewLiveVoterSetState(authSet.SetID, authSet, *base)
//				if err != nil {
//					return nil, err
//				}
//				setStateOld = &state
//
//			} else {
//				panic("state is for completed round; completed rounds must have a prevote ghost; qed")
//			}
//		}
//
//		newSharedVoterSetState := sharedVoterSetState[H, N]{
//			Inner: setState,
//		}
//
//		return &persistentData[H, N]{
//			authoritySet: SharedAuthoritySet[H, N]{inner: authSet},
//			setState:     SharedVoterSetState[H, N]{Inner: newSharedVoterSetState},
//		}, nil
//	}
//
//	logger.Info("👴 Loading GRANDPA authority set from genesis on what appears to be first startup")
//	genesisAuthorities, err := genesisAuths()
//	if err != nil {
//		return nil, err
//	}
//	genesisSet, err := NewGenesisAuthoritySet[H, N](genesisAuthorities)
//	if err != nil {
//		return nil, err
//	}
//
//	state := grandpa.NewRoundState(grandpa.HashNumber[H, N]{Hash: genesisHash, Number: genesisNumber})
//	base := state.PrevoteGHOST
//	if base == nil {
//		panic("state is for completed round; completed rounds must have a prevote ghost; qed.")
//	}
//
//	genesisState, err := NewLiveVoterSetState(0, *genesisSet, *base)
//	if err != nil {
//		return nil, err
//	}
//
//	insert := []api.KeyValue{
//		{authoritySetKey, scale.MustMarshal(*genesisSet)},
//		{setStateKey, scale.MustMarshal(genesisState)},
//	}
//
//	err = store.Insert(insert, nil)
//	if err != nil {
//		return nil, err
//	}
//
//	newSharedVoterSetState := sharedVoterSetState[H, N]{
//		Inner: genesisState,
//	}
//
//	return &persistentData[H, N]{
//		authoritySet: SharedAuthoritySet[H, N]{inner: *genesisSet},
//		setState:     SharedVoterSetState[H, N]{Inner: newSharedVoterSetState},
//	}, nil
//}

func loadPersistent[H comparable, N constraints.Unsigned](store api.AuxStore, genesisHash H, genesisNumber N, genesisAuths getGenesisAuthorities) (*persistentData[H, N], error) {
	genesis := grandpa.HashNumber[H, N]{Hash: genesisHash, Number: genesisNumber}
	makeGenesisRound := grandpa.NewRoundState[H, N]

	set := AuthoritySet[H, N]{
		PendingStandardChanges: NewChangeTree[H, N](),
	}
	var setState voterSetState[H, N]
	encodedAuthSet, err := store.Get(authoritySetKey)
	if err != nil {
		return nil, err
	}

	// If there is no authoritySet data stored, then we are starting from genesis
	if encodedAuthSet != nil {
		err = scale.Unmarshal(*encodedAuthSet, &set)
		if err != nil {
			return nil, err
		}

		encodedSetState, err := store.Get(setStateKey)
		if err != nil {
			return nil, err
		}

		if encodedSetState != nil {
			// Some case
			err = scale.Unmarshal(*encodedSetState, &setState)
			if err != nil {
				return nil, err
			}
		} else {
			// None case
			state := makeGenesisRound(genesis)
			base := state.PrevoteGHOST
			if base != nil {
				state, err := NewLiveVoterSetState(set.SetID, set, *base)
				if err != nil {
					return nil, err
				}
				setState = state

			} else {
				panic("state is for completed round; completed rounds must have a prevote ghost; qed")
			}
		}

		newSharedVoterSetState := sharedVoterSetState[H, N]{
			Inner: setState,
		}

		return &persistentData[H, N]{
			authoritySet: SharedAuthoritySet[H, N]{inner: set},
			setState:     SharedVoterSetState[H, N]{Inner: newSharedVoterSetState},
		}, nil
	}

	logger.Info("👴 Loading GRANDPA authority set from genesis on what appears to be first startup")
	genesisAuthorities, err := genesisAuths()
	if err != nil {
		return nil, err
	}
	genesisSet, err := NewGenesisAuthoritySet[H, N](genesisAuthorities)
	if err != nil {
		return nil, err
	}

	state := grandpa.NewRoundState(grandpa.HashNumber[H, N]{Hash: genesisHash, Number: genesisNumber})
	base := state.PrevoteGHOST
	if base == nil {
		panic("state is for completed round; completed rounds must have a prevote ghost; qed.")
	}

	genesisState, err := NewLiveVoterSetState(0, *genesisSet, *base)
	if err != nil {
		return nil, err
	}

	insert := []api.KeyValue{
		{authoritySetKey, scale.MustMarshal(*genesisSet)},
		{setStateKey, scale.MustMarshal(genesisState)},
	}

	err = store.Insert(insert, nil)
	if err != nil {
		return nil, err
	}

	newSharedVoterSetState := sharedVoterSetState[H, N]{
		Inner: genesisState,
	}
	return &persistentData[H, N]{
		authoritySet: SharedAuthoritySet[H, N]{inner: *genesisSet},
		setState:     SharedVoterSetState[H, N]{Inner: newSharedVoterSetState},
	}, nil
}

// UpdateAuthoritySet Update the authority set on disk after a change.
//
// If there has just been a handoff, pass a `new_set` parameter that describes the
// handoff. `set` in all cases should reflect the current authority set, with all
// changes and handoffs applied.
func UpdateAuthoritySet[H comparable, N constraints.Unsigned](set AuthoritySet[H, N], newSet *NewAuthoritySetStruct[H, N], write writeAux) error {
	// TODO make sure that Insert has affect of both insert and update depending on use case
	encodedAuthSet, err := scale.Marshal(set)
	if err != nil {
		return err
	}

	if newSet != nil {
		// we also overwrite the "last completed round" entry with a blank slate
		// because from the perspective of the finality gadget, the chain has
		// reset.
		genesisState := grandpa.HashNumber[H, N]{
			Hash:   newSet.CanonHash,
			Number: newSet.CanonNumber,
		}
		setState, err := NewLiveVoterSetState(uint64(newSet.SetId), set, genesisState)
		if err != nil {
			return err
		}

		encodedVoterSet, err := scale.Marshal(setState)
		if err != nil {
			return err
		}

		insert := []api.KeyValue{
			{authoritySetKey, encodedAuthSet},
			{setStateKey, encodedVoterSet},
		}
		err = write(insert)
		if err != nil {
			return err
		}

	} else {
		insert := []api.KeyValue{
			{authoritySetKey, encodedAuthSet},
		}

		err = write(insert)
		if err != nil {
			return err
		}
	}
	return nil
}

// UpdateBestJustification Update the justification for the latest finalized block on-disk.
//
// We always keep around the justification for the best finalized block and overwrite it
// as we finalize new blocks, this makes sure that we don't store useless justifications
// but can always prove finality of the latest block.
func UpdateBestJustification() {
	// TODO impl when we have justification logic
	panic("impl")
}

// BestJustification  Fetch the justification for the latest block finalized by GRANDPA, if any.
func BestJustification() {
	// TODO impl when we have justification logic
	panic("impl")
}

// WriteVoterSetState Write voter set state.
func WriteVoterSetState[H comparable, N constraints.Unsigned](setState voterSetState[H, N], write writeAux) error {
	encodedVoterSet, err := scale.Marshal(setState)
	if err != nil {
		return err
	}
	insert := []api.KeyValue{
		{setStateKey, encodedVoterSet},
	}
	err = write(insert)
	if err != nil {
		return err
	}
	return nil
}

// WriteConcludedRound Write concluded round.
func WriteConcludedRound[H comparable, N constraints.Unsigned](roundData completedRound[H, N], write writeAux) error {
	key := concludedRounds
	encodedRoundNumber, err := scale.Marshal(roundData.Number)
	if err != nil {
		return err
	}

	key = append(key, encodedRoundNumber...)

	encRoundData, err := scale.Marshal(roundData)
	if err != nil {
		return err
	}

	insert := []api.KeyValue{
		{key, encRoundData},
	}
	err = write(insert)
	if err != nil {
		return err
	}
	return nil

}

func LoadAuthorities() {
	// Not sure if this is really needed or just used in tests
	panic("impl")
}

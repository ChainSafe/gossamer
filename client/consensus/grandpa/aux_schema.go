// Copyright 2023 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package grandpa

import (
	"github.com/ChainSafe/gossamer/pkg/scale"
	"golang.org/x/exp/constraints"

	finalityGrandpa "github.com/ChainSafe/gossamer/pkg/finality-grandpa"
)

var (
	SET_STATE_KEY      = []byte("grandpa_completed_round")
	CONCLUDED_ROUNDS   = []byte("grandpa_concluded_rounds")
	AUTHORITY_SET_KEY  = []byte("grandpa_voters")
	BEST_JUSTIFICATION = []byte("grandpa_best_justification")
)

//type databaseKeyValue [2][]byte

type KeyValue struct {
	key   []byte
	value []byte
}

// SharedVoterSetState .
type SharedVoterSetState[H comparable, N constraints.Unsigned] struct {
	// TODO move and implement
	inner VoterSetState[H, N]
}

type persistentData[H comparable, N constraints.Unsigned] struct {
	authoritySet SharedAuthoritySet[H, N] // TODO this needs to be shared auth set
	setState     SharedVoterSetState[H, N]
}

func loadPersistent[H comparable, N constraints.Unsigned](store AuxStore, genesisHash H, genesisNumber N, genesisAuthorities []Authority) (*persistentData[H, N], error) {
	genesis := finalityGrandpa.HashNumber[H, N]{Hash: genesisHash, Number: genesisNumber}
	makeGenesisRound := finalityGrandpa.NewRoundState[H, N]

	set := AuthoritySet[H, N]{
		PendingStandardChanges: NewChangeTree[H, N](),
		PendingForcedChanges:   make([]PendingChange[H, N], 0),
		AuthoritySetChanges:    make(AuthoritySetChanges[N], 0),
	}
	setState := *NewVoterSetState[H, N]()
	encodedAuthSet, err := store.Get(AUTHORITY_SET_KEY)
	if err != nil {
		return nil, err
	}

	// If there is no authoritySet data stored, then we are starting from genesis
	if encodedAuthSet != nil {
		err = scale.Unmarshal(*encodedAuthSet, &set)
		if err != nil {
			return nil, err
		}

		encodedSetState, err := store.Get(SET_STATE_KEY)
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
				voterSetState := NewVoterSetState[H, N]()
				state, err := voterSetState.Live(set.SetID, set, *base)
				if err != nil {
					return nil, err
				}
				setState = state

			} else {
				panic("state is for completed round; completed rounds must have a prevote ghost; qed")
			}
		}
		return &persistentData[H, N]{
			authoritySet: SharedAuthoritySet[H, N]{inner: set},
			setState:     SharedVoterSetState[H, N]{inner: setState},
		}, nil
	} else {
		logger.Info("👴 Loading GRANDPA authority set from genesis on what appears to be first startup")
		genesisSet := NewGenesisAuthoritySet[H, N](genesisAuthorities)

		state := finalityGrandpa.NewRoundState(finalityGrandpa.HashNumber[H, N]{Hash: genesisHash, Number: genesisNumber})
		base := state.PrevoteGHOST
		if base == nil {
			panic("state is for completed round; completed rounds must have a prevote ghost; qed.")
		}

		voterSetState := NewVoterSetState[H, N]()
		genesisState, err := voterSetState.Live(0, *genesisSet, *base)
		if err != nil {
			return nil, err
		}

		insert := []KeyValue{
			{AUTHORITY_SET_KEY, scale.MustMarshal(*genesisSet)},
			{SET_STATE_KEY, scale.MustMarshal(genesisState)},
		}

		err = store.Insert(insert, nil)
		if err != nil {
			return nil, err
		}

		return &persistentData[H, N]{
			authoritySet: SharedAuthoritySet[H, N]{inner: *genesisSet},
			setState:     SharedVoterSetState[H, N]{inner: genesisState},
		}, nil
	}
}

// UpdateAuthoritySet Update the authority set on disk after a change.
//
// If there has just been a handoff, pass a `new_set` parameter that describes the
// handoff. `set` in all cases should reflect the current authority set, with all
// changes and handoffs applied.
func UpdateAuthoritySet[H comparable, N constraints.Unsigned](store AuxStore, set AuthoritySet[H, N], newSet *NewAuthoritySetStruct[H, N]) error {
	// TODO make sure that Insert has affect of both insert and update depending on use case
	encodedAuthSet, err := scale.Marshal(set)
	if err != nil {
		return err
	}

	if newSet != nil {
		// we also overwrite the "last completed round" entry with a blank slate
		// because from the perspective of the finality gadget, the chain has
		// reset.
		genesisState := finalityGrandpa.HashNumber[H, N]{
			Hash:   newSet.CanonHash,
			Number: newSet.CanonNumber,
		}
		voterSetState := NewVoterSetState[H, N]()
		setState, err := voterSetState.Live(uint64(newSet.SetId), set, genesisState)
		if err != nil {
			return err
		}

		encodedVoterSet, err := scale.Marshal(setState)
		if err != nil {
			return err
		}

		insert := []KeyValue{
			{AUTHORITY_SET_KEY, encodedAuthSet},
			{SET_STATE_KEY, encodedVoterSet},
		}
		err = store.Insert(insert, nil)
		if err != nil {
			return err
		}

	} else {
		insert := []KeyValue{
			{AUTHORITY_SET_KEY, encodedAuthSet},
		}

		err = store.Insert(insert, nil)
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
func WriteVoterSetState[H comparable, N constraints.Unsigned](store AuxStore, setState VoterSetState[H, N]) error {
	encodedVoterSet, err := scale.Marshal(setState)
	if err != nil {
		return err
	}
	insert := []KeyValue{
		{SET_STATE_KEY, encodedVoterSet},
	}
	err = store.Insert(insert, nil)
	if err != nil {
		return err
	}
	return nil
}

// WriteConcludedRound Write concluded round.
func WriteConcludedRound[H comparable, N constraints.Unsigned](store AuxStore, roundData CompletedRound[H, N]) error {
	key := CONCLUDED_ROUNDS
	encodedRoundNumber, err := scale.Marshal(roundData.Number)
	if err != nil {
		return err
	}

	key = append(key, encodedRoundNumber...)

	encRoundData, err := scale.Marshal(roundData)
	if err != nil {
		return err
	}

	insert := []KeyValue{
		{key, encRoundData},
	}
	err = store.Insert(insert, nil)
	if err != nil {
		return err
	}
	return nil

}

func LoadAuthorities() {
	// Not sure if this is really needed or just used in tests
	panic("impl")
}

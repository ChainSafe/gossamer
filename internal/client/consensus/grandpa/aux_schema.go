// Copyright 2023 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package grandpa

import (
	"errors"
	"fmt"

	"github.com/ChainSafe/gossamer/internal/client/api"
	pgrandpa "github.com/ChainSafe/gossamer/internal/primitives/consensus/grandpa"
	"github.com/ChainSafe/gossamer/internal/primitives/runtime"
	grandpa "github.com/ChainSafe/gossamer/pkg/finality-grandpa"
	"github.com/ChainSafe/gossamer/pkg/scale"
	"golang.org/x/exp/constraints"
)

var (
	setStateKey       = []byte("grandpa_completed_round")
	concludedRounds   = []byte("grandpa_concluded_rounds")
	authoritySetKey   = []byte("grandpa_voters")
	bestJustification = []byte("grandpa_best_justification")

	errValueNotFound = errors.New("value not found")
)

type writeAux func(insertions []api.KeyValue) error

type getGenesisAuthorities func() (pgrandpa.AuthorityList, error)

type persistentData[H comparable, N constraints.Unsigned] struct {
	authoritySet SharedAuthoritySet[H, N]
	setState     SharedVoterSetState[H, N]
}

func loadDecoded(store api.AuxStore, key []byte, destination any) error {
	encodedValue, err := store.GetAux(key)
	if err != nil {
		return err
	}

	if encodedValue != nil {
		err = scale.Unmarshal(*encodedValue, destination)
		if err != nil {
			return err
		}

		return nil
	}

	return errValueNotFound
}

func loadPersistent[H comparable, N constraints.Unsigned](
	store api.AuxStore,
	genesisHash H,
	genesisNumber N,
	genesisAuths getGenesisAuthorities,
) (*persistentData[H, N], error) {
	genesis := grandpa.HashNumber[H, N]{Hash: genesisHash, Number: genesisNumber}
	makeGenesisRound := grandpa.NewRoundState[H, N]

	authSet := &AuthoritySet[H, N]{}
	err := loadDecoded(store, authoritySetKey, authSet)
	if err != nil && !errors.Is(err, errValueNotFound) {
		return nil, err
	}

	if !errors.Is(err, errValueNotFound) {
		setStateOld := voterSetState[H, N]{}
		setState := setStateOld.New()
		err = loadDecoded(store, setStateKey, &setState)
		if err != nil && !errors.Is(err, errValueNotFound) {
			return nil, err
		}

		if errors.Is(err, errValueNotFound) {
			state := makeGenesisRound(genesis)
			base := state.PrevoteGHOST
			if base != nil {
				state, err := NewLiveVoterSetState[H, N](authSet.SetID, *authSet, *base)
				if err != nil {
					return nil, err
				}
				setState = state
			} else {
				panic("state is for completed round; completed rounds must have a prevote ghost; qed")
			}
		}

		return &persistentData[H, N]{
			authoritySet: SharedAuthoritySet[H, N]{inner: *authSet},
			setState: SharedVoterSetState[H, N]{Inner: sharedVoterSetState[H, N]{
				Inner: setState,
			}},
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

	genesisState, err := NewLiveVoterSetState[H, N](0, *genesisSet, *base)
	if err != nil {
		return nil, err
	}

	insert := []api.KeyValue{
		{authoritySetKey, scale.MustMarshal(*genesisSet)},
		{setStateKey, scale.MustMarshal(genesisState)},
	}

	err = store.InsertAux(insert, nil)
	if err != nil {
		return nil, err
	}

	return &persistentData[H, N]{
		authoritySet: SharedAuthoritySet[H, N]{inner: *genesisSet},
		setState: SharedVoterSetState[H, N]{Inner: sharedVoterSetState[H, N]{
			Inner: genesisState,
		}},
	}, nil
}

// UpdateAuthoritySet Update the authority set on disk after a change.
//
// If there has just been a handoff, pass a `new_set` parameter that describes the
// handoff. `set` in all cases should reflect the current authority set, with all
// changes and handoffs applied.
func UpdateAuthoritySet[H comparable, N constraints.Unsigned, ID pgrandpa.AuthorityID](
	set AuthoritySet[H, N],
	newSet *newAuthoritySet[H, N],
	write writeAux) error {
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
		setState, err := NewLiveVoterSetState[H, N](uint64(newSet.SetId), set, genesisState)
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
func updateBestJustification[
	Hash constraints.Ordered,
	N runtime.Number,
	S comparable,
	ID pgrandpa.AuthorityID,
](
	justification GrandpaJustification[Hash, N],
	write writeAux) error {
	encodedJustificaiton, err := scale.Marshal(justification)
	if err != nil {
		return fmt.Errorf("marshalling: %w", err)
	}

	insert := []api.KeyValue{
		{bestJustification, encodedJustificaiton},
	}
	err = write(insert)
	if err != nil {
		return fmt.Errorf("inserting justification: %w", err)
	}
	return nil
}

// BestJustification  Fetch the justification for the latest block finalized by GRANDPA, if any.
func BestJustification[
	Hash constraints.Ordered,
	N runtime.Number,
](store api.AuxStore) (*GrandpaJustification[Hash, N], error) {
	justification := decodeGrandpaJustification[Hash, N]{}
	err := loadDecoded(store, bestJustification, &justification)
	if err != nil {
		return nil, err
	}

	return justification.GrandpaJustification(), nil
}

// WriteVoterSetState Write voter set state.
func WriteVoterSetState[H comparable, N constraints.Unsigned](
	setState voterSetState[H, N],
	write writeAux) error {
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
func WriteConcludedRound[H comparable, N constraints.Unsigned](
	roundData completedRound[H, N],
	write writeAux) error {
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

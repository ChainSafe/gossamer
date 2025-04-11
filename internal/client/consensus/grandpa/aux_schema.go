// Copyright 2023 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package grandpa

import (
	"fmt"

	"github.com/ChainSafe/gossamer/internal/client/api"
	"github.com/ChainSafe/gossamer/internal/primitives/runtime"
	grandpa "github.com/ChainSafe/gossamer/pkg/finality-grandpa"
	"github.com/ChainSafe/gossamer/pkg/scale"
)

var (
	setStateKey     = []byte("grandpa_completed_round")
	concludedRounds = []byte("grandpa_concluded_rounds")

	authoritySetKey   = []byte("grandpa_voters")
	bestJustification = []byte("grandpa_best_justification")

// errValueNotFound = errors.New("value not found")
)

type writeAux func(insertions []api.KeyValue) error

// type getGenesisAuthorities func() (primitives.AuthorityList, error)

// type persistentData[H comparable, N constraints.Unsigned] struct {
// 	authoritySet SharedAuthoritySet[H, N]
// 	setState     SharedVoterSetState[H, N]
// }

// func loadDecoded(store api.AuxStore, key []byte, destination any) error {
// 	encodedValue, err := store.GetAux(key)
// 	if err != nil {
// 		return err
// 	}

// 	if encodedValue != nil {
// 		err = scale.Unmarshal(*encodedValue, destination)
// 		if err != nil {
// 			return err
// 		}

// 		return nil
// 	}

// 	return errValueNotFound
// }

// func loadPersistent[H comparable, N constraints.Unsigned](
// 	store api.AuxStore,
// 	genesisHash H,
// 	genesisNumber N,
// 	genesisAuths getGenesisAuthorities,
// ) (*persistentData[H, N], error) {
// 	genesis := grandpa.HashNumber[H, N]{Hash: genesisHash, Number: genesisNumber}
// 	makeGenesisRound := grandpa.NewRoundState[H, N]

// 	authSet := &AuthoritySet[H, N]{}
// 	err := loadDecoded(store, authoritySetKey, authSet)
// 	if err != nil && !errors.Is(err, errValueNotFound) {
// 		return nil, err
// 	}

// 	if !errors.Is(err, errValueNotFound) {
// 		setState := voterSetState[H, N]{}
// 		err = loadDecoded(store, setStateKey, &setState)
// 		if err != nil && !errors.Is(err, errValueNotFound) {
// 			return nil, err
// 		}

// 		if errors.Is(err, errValueNotFound) {
// 			state := makeGenesisRound(genesis)
// 			base := state.PrevoteGHOST
// 			if base != nil {
// 				state, err := NewLiveVoterSetState[H, N](authSet.SetID, *authSet, *base)
// 				if err != nil {
// 					return nil, err
// 				}
// 				setState = state
// 			} else {
// 				panic("state is for completed round; completed rounds must have a prevote ghost; qed")
// 			}
// 		}

// 		return &persistentData[H, N]{
// 			authoritySet: SharedAuthoritySet[H, N]{inner: *authSet},
// 			setState: SharedVoterSetState[H, N]{Inner: sharedVoterSetState[H, N]{
// 				Inner: setState,
// 			}},
// 		}, nil
// 	}

// 	logger.Info("👴 Loading GRANDPA authority set from genesis on what appears to be first startup")
// 	genesisAuthorities, err := genesisAuths()
// 	if err != nil {
// 		return nil, err
// 	}
// 	genesisSet, err := NewGenesisAuthoritySet[H, N](genesisAuthorities)
// 	if err != nil {
// 		return nil, err
// 	}

// 	state := grandpa.NewRoundState(grandpa.HashNumber[H, N]{Hash: genesisHash, Number: genesisNumber})
// 	base := state.PrevoteGHOST
// 	if base == nil {
// 		panic("state is for completed round; completed rounds must have a prevote ghost; qed.")
// 	}

// 	genesisState, err := NewLiveVoterSetState[H, N](0, *genesisSet, *base)
// 	if err != nil {
// 		return nil, err
// 	}

// 	insert := []api.KeyValue{
// 		{Key: authoritySetKey, Value: scale.MustMarshal(*genesisSet)},
// 		{Key: setStateKey, Value: scale.MustMarshal(genesisState)},
// 	}

// 	err = store.InsertAux(insert, nil)
// 	if err != nil {
// 		return nil, err
// 	}

// 	return &persistentData[H, N]{
// 		authoritySet: SharedAuthoritySet[H, N]{inner: *genesisSet},
// 		setState: SharedVoterSetState[H, N]{Inner: sharedVoterSetState[H, N]{
// 			Inner: genesisState,
// 		}},
// 	}, nil
// }

// updateAuthoritySet Update the authority set on disk after a change.
//
// If there has just been a handoff, pass a `new_set` parameter that describes the
// handoff. `set` in all cases should reflect the current authority set, with all
// changes and handoffs applied.
func updateAuthoritySet[H runtime.Hash, N runtime.Number](
	set AuthoritySet[H, N],
	newSet *newAuthoritySet[H, N],
	write writeAux,
) error {
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
		setState := newVoterSetStateLive[H, N](newSet.SetID, set, genesisState)
		vdt := newVoterSetStateVDT[H, N]()
		vdt.inner = setState

		encodedVoterSet, err := scale.Marshal(*vdt)
		if err != nil {
			return err
		}

		insert := []api.KeyValue{
			{Key: authoritySetKey, Value: encodedAuthSet},
			{Key: setStateKey, Value: encodedVoterSet},
		}
		err = write(insert)
		if err != nil {
			return err
		}

	} else {
		insert := []api.KeyValue{
			{Key: authoritySetKey, Value: encodedAuthSet},
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
	Hash runtime.Hash,
	N runtime.Number,
	Header runtime.Header[N, Hash],
](
	justification GrandpaJustification[Hash, N, Header],
	write writeAux,
) error {
	encodedJustificaiton, err := scale.Marshal(justification)
	if err != nil {
		return fmt.Errorf("marshalling: %w", err)
	}

	insert := []api.KeyValue{
		{Key: bestJustification, Value: encodedJustificaiton},
	}
	err = write(insert)
	if err != nil {
		return fmt.Errorf("inserting justification: %w", err)
	}
	return nil
}

// // BestJustification  Fetch the justification for the latest block finalized by GRANDPA, if any.
// func BestJustification[
// 	Hash runtime.Hash,
// 	N runtime.Number,
// 	Hasher runtime.Hasher[Hash],
// ](store api.AuxStore) (*GrandpaJustification[Hash, N], error) {
// 	justification := decodeGrandpaJustification[Hash, N, Hasher]{}
// 	err := loadDecoded(store, bestJustification, &justification)
// 	if err != nil {
// 		return nil, err
// 	}

// 	return justification.GrandpaJustification(), nil
// }

// writeVoterSetState will write voter set state.
func writeVoterSetState[H runtime.Hash, N runtime.Number](
	backend api.AuxStore,
	state voterSetState[H, N],
) error {
	vdt := newVoterSetStateVDT[H, N]()
	vdt.inner = state
	encoded, err := scale.Marshal(*vdt)
	if err != nil {
		return err
	}
	return backend.InsertAux([]api.KeyValue{
		{Key: setStateKey, Value: encoded},
	}, nil)
}

// writeConcludedRound will write the concluded round.
func writeConcludedRound[H runtime.Hash, N runtime.Number](
	backend api.AuxStore,
	roundData completedRound[H, N],
) error {
	key := append(concludedRounds, scale.MustMarshal(roundData.Number)...)

	encRoundData, err := scale.Marshal(roundData)
	if err != nil {
		return err
	}

	return backend.InsertAux([]api.KeyValue{
		{Key: key, Value: encRoundData},
	}, nil)
}

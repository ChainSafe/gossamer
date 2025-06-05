// Copyright 2023 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package grandpa

import (
	"errors"
	"fmt"

	"github.com/ChainSafe/gossamer/internal/client/api"
	shareddata "github.com/ChainSafe/gossamer/internal/client/consensus/common/shared-data"
	primitives "github.com/ChainSafe/gossamer/internal/primitives/consensus/grandpa"
	"github.com/ChainSafe/gossamer/internal/primitives/runtime"
	grandpa "github.com/ChainSafe/gossamer/pkg/finality-grandpa"
	"github.com/ChainSafe/gossamer/pkg/scale"
)

var (
	setStateKey       = []byte("grandpa_completed_round")
	concludedRounds   = []byte("grandpa_concluded_rounds")
	authoritySetKey   = []byte("grandpa_voters")
	bestJustification = []byte("grandpa_best_justification")

	errValueNotFound = errors.New("value not found")
)

type writeAux func(insertions []api.KeyValue) error

type getGenesisAuthorities func() (primitives.AuthorityList, error)

// / Persistent data kept between runs.
type persistentData[H runtime.Hash, N runtime.Number] struct {
	authoritySet *SharedAuthoritySet[H, N]
	setState     *SharedVoterSetState[H, N]
}

func loadDecoded[T any](store api.AuxStore, key []byte) (*T, error) {
	encodedValue, err := store.GetAux(key)
	if err != nil {
		return nil, err
	}
	if encodedValue == nil {
		return nil, nil
	}

	var dst T
	err = scale.Unmarshal(encodedValue, &dst)
	if err != nil {
		return nil, err
	}
	return &dst, nil
}

func loadPersistent[H runtime.Hash, N runtime.Number](
	store api.AuxStore,
	genesisHash H,
	genesisNumber N,
	genesisAuths getGenesisAuthorities,
) (*persistentData[H, N], error) {
	genesis := grandpa.HashNumber[H, N]{Hash: genesisHash, Number: genesisNumber}
	makeGenesisRound := grandpa.NewRoundState[H, N]

	authSet, err := loadDecoded[AuthoritySet[H, N]](store, authoritySetKey)
	if authSet != nil {
		var setState voterSetState[H, N]
		state, err := loadDecoded[voterSetStateVDT[H, N]](store, setStateKey)
		if err != nil {
			return nil, err
		}

		if state != nil && state.inner != nil {
			setState = state.inner
		} else {
			state := makeGenesisRound(genesis)
			if state.PrevoteGHOST == nil {
				panic("state is for completed round; completed rounds must have a prevote ghost; qed.")
			}
			base := state.PrevoteGHOST
			setState = newVoterSetStateLive(primitives.SetID(authSet.SetID), *authSet, *base)
		}

		return &persistentData[H, N]{
			authoritySet: &SharedAuthoritySet[H, N]{inner: *shareddata.NewSharedData(*authSet)},
			setState:     &SharedVoterSetState[H, N]{inner: setState},
		}, nil
	}

	logger.Info("👴 Loading GRANDPA authority set from genesis on what appears to be first startup")
	genesisAuthorities, err := genesisAuths()
	if err != nil {
		return nil, err
	}
	genesisSet, err := NewGenesisAuthoritySet[H, N](genesisAuthorities)
	if err != nil {
		panic("genesis authorities is non-empty; all weights are non-zero; qed.")
	}

	state := makeGenesisRound(genesis)
	base := state.PrevoteGHOST
	if base == nil {
		panic("state is for completed round; completed rounds must have a prevote ghost; qed.")
	}

	genesisState := newVoterSetStateLive(0, *genesisSet, *base)
	genesisStateVDT := newVoterSetStateVDT[H, N]()
	genesisStateVDT.inner = genesisState
	insert := []api.KeyValue{
		{Key: authoritySetKey, Value: scale.MustMarshal(*genesisSet)},
		{Key: setStateKey, Value: scale.MustMarshal(genesisStateVDT)},
	}
	err = store.InsertAux(insert, nil)
	if err != nil {
		return nil, err
	}

	return &persistentData[H, N]{
		authoritySet: &SharedAuthoritySet[H, N]{inner: *shareddata.NewSharedData(*genesisSet)},
		setState:     &SharedVoterSetState[H, N]{inner: genesisState},
	}, nil
}

// updateAuthoritySet Update the authority set on disk after a change.
//
// If there has just been a handoff, pass a newSet parameter that describes the handoff. set in all cases should
// reflect the current authority set, with all changes and handoffs applied.
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
		// we also overwrite the "last completed round" entry with a blank slate because from the perspective of the
		// finality gadget, the chain has reset.
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

// UpdateBestJustification updates the justification for the latest finalized block on-disk.
//
// We always keep around the justification for the best finalized block and overwrite it as we finalize new blocks,
// this makes sure that we don't store useless justifications but can always prove finality of the latest block.
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

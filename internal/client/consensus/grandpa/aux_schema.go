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
	setStateKey       = []byte("grandpa_completed_round")
	concludedRounds   = []byte("grandpa_concluded_rounds")
	authoritySetKey   = []byte("grandpa_voters")
	bestJustification = []byte("grandpa_best_justification")
)

// / Persistent data kept between runs.
type persistentData[H runtime.Hash, N runtime.Number] struct {
	authoritySet *SharedAuthoritySet[H, N]
	setState     *SharedVoterSetState[H, N]
}

type writeAux func(insertions []api.KeyValue) error

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

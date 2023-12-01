// Copyright 2023 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package grandpa

import (
	"errors"
	"fmt"
	"io"

	finalityGrandpa "github.com/ChainSafe/gossamer/pkg/finality-grandpa"
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

type writeAux func(insertions []KeyValue) error

type getGenesisAuthorities[ID AuthorityID] func() ([]Authority[ID], error)

type persistentData[H comparable, N constraints.Unsigned, ID AuthorityID, Sig AuthoritySignature] struct {
	authoritySet SharedAuthoritySet[H, N, ID]
	setState     SharedVoterSetState[H, N, ID, Sig]
}

func loadDecoded(store AuxStore, key []byte, destination any) error {
	encodedValue, err := store.Get(key)
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

func loadPersistent[H comparable, N constraints.Unsigned, ID AuthorityID, Sig AuthoritySignature](
	store AuxStore,
	genesisHash H,
	genesisNumber N,
	genesisAuths getGenesisAuthorities[ID]) (*persistentData[H, N, ID, Sig], error) {
	genesis := finalityGrandpa.HashNumber[H, N]{Hash: genesisHash, Number: genesisNumber}
	makeGenesisRound := finalityGrandpa.NewRoundState[H, N]

	authSet := &AuthoritySet[H, N, ID]{}
	err := loadDecoded(store, authoritySetKey, authSet)
	if err != nil && !errors.Is(err, errValueNotFound) {
		return nil, err
	}

	if !errors.Is(err, errValueNotFound) {
		setStateOld := voterSetState[H, N, ID, Sig]{}
		setState := setStateOld.New()
		err = loadDecoded(store, setStateKey, &setState)
		if err != nil && !errors.Is(err, errValueNotFound) {
			return nil, err
		}

		if errors.Is(err, errValueNotFound) {
			state := makeGenesisRound(genesis)
			base := state.PrevoteGHOST
			if base != nil {
				state, err := NewLiveVoterSetState[H, N, ID, Sig](authSet.SetID, *authSet, *base)
				if err != nil {
					return nil, err
				}
				setState = state
			} else {
				panic("state is for completed round; completed rounds must have a prevote ghost; qed")
			}
		}

		newSharedVoterSetState := sharedVoterSetState[H, N, ID, Sig]{
			Inner: setState,
		}

		return &persistentData[H, N, ID, Sig]{
			authoritySet: SharedAuthoritySet[H, N, ID]{inner: *authSet},
			setState:     SharedVoterSetState[H, N, ID, Sig]{Inner: newSharedVoterSetState}, //nolint
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

	state := finalityGrandpa.NewRoundState(finalityGrandpa.HashNumber[H, N]{Hash: genesisHash, Number: genesisNumber})
	base := state.PrevoteGHOST
	if base == nil {
		panic("state is for completed round; completed rounds must have a prevote ghost; qed.")
	}

	genesisState, err := NewLiveVoterSetState[H, N, ID, Sig](0, *genesisSet, *base)
	if err != nil {
		return nil, err
	}

	insert := []KeyValue{
		{authoritySetKey, scale.MustMarshal(*genesisSet)},
		{setStateKey, scale.MustMarshal(genesisState)},
	}

	err = store.Insert(insert, nil)
	if err != nil {
		return nil, err
	}

	newSharedVoterSetState := sharedVoterSetState[H, N, ID, Sig]{
		Inner: genesisState,
	}

	return &persistentData[H, N, ID, Sig]{
		authoritySet: SharedAuthoritySet[H, N, ID]{inner: *genesisSet},
		setState:     SharedVoterSetState[H, N, ID, Sig]{Inner: newSharedVoterSetState}, //nolint
	}, nil
}

// UpdateAuthoritySet Update the authority set on disk after a change.
//
// If there has just been a handoff, pass a `new_set` parameter that describes the
// handoff. `set` in all cases should reflect the current authority set, with all
// changes and handoffs applied.
func UpdateAuthoritySet[H comparable, N constraints.Unsigned, ID AuthorityID, Sig AuthoritySignature](
	set AuthoritySet[H, N, ID],
	newSet *NewAuthoritySetStruct[H, N, ID],
	write writeAux) error {
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
		setState, err := NewLiveVoterSetState[H, N, ID, Sig](uint64(newSet.SetId), set, genesisState)
		if err != nil {
			return err
		}

		encodedVoterSet, err := scale.Marshal(setState)
		if err != nil {
			return err
		}

		insert := []KeyValue{
			{authoritySetKey, encodedAuthSet},
			{setStateKey, encodedVoterSet},
		}
		err = write(insert)
		if err != nil {
			return err
		}

	} else {
		insert := []KeyValue{
			{authoritySetKey, encodedAuthSet},
		}

		err = write(insert)
		if err != nil {
			return err
		}
	}
	return nil
}

// UpdateBestJustification Update the justification for the authoritySetChangeIDLatest finalized block on-disk.
//
// We always keep around the justification for the best finalized block and overwrite it
// as we finalize new blocks, this makes sure that we don't store useless justifications
// but can always prove finality of the authoritySetChangeIDLatest block.
func updateBestJustification[
	Hash constraints.Ordered,
	N constraints.Unsigned,
	S comparable,
	ID AuthorityID,
](
	justification GrandpaJustification[Hash, N, S, ID],
	write writeAux) error {
	encodedJustificaiton, err := scale.Marshal(justification)
	if err != nil {
		return fmt.Errorf("marshalling: %w", err)
	}

	insert := []KeyValue{
		{bestJustification, encodedJustificaiton},
	}
	err = write(insert)
	if err != nil {
		return fmt.Errorf("inserting justification: %w", err)
	}
	return nil
}

type decodeGrandpaJustification[
	Hash constraints.Ordered,
	N constraints.Unsigned,
	S comparable,
	ID AuthorityID,
	H Header[Hash, N],
] GrandpaJustification[Hash, N, S, ID]

func (dgj *decodeGrandpaJustification[Hash, N, S, ID, H]) UnmarshalSCALE(reader io.Reader) (err error) {
	type roundCommit struct {
		Round  uint64
		Commit finalityGrandpa.Commit[Hash, N, S, ID]
	}
	rc := roundCommit{}
	decoder := scale.NewDecoder(reader)
	err = decoder.Decode(&rc)
	if err != nil {
		return
	}

	dgj.Round = rc.Round
	dgj.Commit = rc.Commit

	headers := []H{}
	err = decoder.Decode(&headers)
	dgj.VotesAncestries = make([]Header[Hash, N], len(headers))
	for i, header := range headers {
		dgj.VotesAncestries[i] = header
	}
	return
}

func (dgj decodeGrandpaJustification[Hash, N, S, ID, H]) GrandpaJustification() *GrandpaJustification[Hash, N, S, ID] {
	return &GrandpaJustification[Hash, N, S, ID]{
		dgj.Round, dgj.Commit, dgj.VotesAncestries,
	}
}

// BestJustification  Fetch the justification for the authoritySetChangeIDLatest block finalized by GRANDPA, if any.
func BestJustification[
	Hash constraints.Ordered,
	N constraints.Unsigned,
	S comparable,
	ID AuthorityID,
	H Header[Hash, N],
](store AuxStore) (*GrandpaJustification[Hash, N, S, ID], error) {
	justification := decodeGrandpaJustification[Hash, N, S, ID, H]{}
	err := loadDecoded(store, bestJustification, &justification)
	if err != nil {
		return nil, err
	}

	return justification.GrandpaJustification(), nil
}

// WriteVoterSetState Write voter set state.
func WriteVoterSetState[H comparable, N constraints.Unsigned, ID AuthorityID, Sig AuthoritySignature](
	setState voterSetState[H, N, ID, Sig],
	write writeAux) error {
	encodedVoterSet, err := scale.Marshal(setState)
	if err != nil {
		return err
	}
	insert := []KeyValue{
		{setStateKey, encodedVoterSet},
	}
	err = write(insert)
	if err != nil {
		return err
	}
	return nil
}

// WriteConcludedRound Write concluded round.
func WriteConcludedRound[H comparable, N constraints.Unsigned, ID AuthorityID, Sig AuthoritySignature](
	roundData completedRound[H, N, ID, Sig],
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

	insert := []KeyValue{
		{key, encRoundData},
	}
	err = write(insert)
	if err != nil {
		return err
	}
	return nil
}

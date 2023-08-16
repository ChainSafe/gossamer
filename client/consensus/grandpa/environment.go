// Copyright 2023 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package grandpa

import (
	"fmt"

	"github.com/ChainSafe/gossamer/lib/crypto/ed25519"
	grandpa "github.com/ChainSafe/gossamer/pkg/finality-grandpa"
	"github.com/ChainSafe/gossamer/pkg/scale"
	"golang.org/x/exp/constraints"
	"golang.org/x/exp/slices"
)

// CompletedRound Data about a completed round. The set of votes that is stored must be
// minimal, i.e. at most one equivocation is stored per voter.
type CompletedRound[H comparable, N constraints.Unsigned] struct {
	// The round number
	Number uint64
	// The round state (prevote ghost, estimate, finalized, etc.)
	State grandpa.RoundState[H, N]
	// The target block base used for voting in the round
	Base grandpa.HashNumber[H, N]
	// All the votes observed in the round
	// I think this is signature type, double check
	Votes []grandpa.SignedMessage[H, N, ed25519.SignatureBytes, ed25519.PublicKey]
}

// NumLastCompletedRounds NOTE: the current strategy for persisting completed rounds is very naive
// (update everything) and we also rely on cloning to do atomic updates,
// therefore this value should be kept small for now.
const NumLastCompletedRounds = 2

// CompletedRounds Data about last completed rounds within a single voter set. Stores
// NumLastCompletedRounds and always contains data about at least one round
// (genesis).
type CompletedRounds[H comparable, N constraints.Unsigned] struct {
	Rounds []CompletedRound[H, N]
	SetId  uint64
	Voters []ed25519.PublicKey
}

// NewCompletedRounds Create a new completed rounds tracker with NUM_LAST_COMPLETED_ROUNDS capacity.
func NewCompletedRounds[H comparable, N constraints.Unsigned](genesis *CompletedRound[H, N], setId uint64, voters AuthoritySet[H, N]) CompletedRounds[H, N] {
	rounds := make([]CompletedRound[H, N], 0, NumLastCompletedRounds)
	if genesis != nil {
		rounds = append(rounds, *genesis)
	}

	var voterIds []ed25519.PublicKey
	currentAuthorities := voters.CurrentAuthorities
	for _, auth := range currentAuthorities {
		voterIds = append(voterIds, auth.Key)
	}

	return CompletedRounds[H, N]{
		rounds,
		setId,
		voterIds,
	}
}

// Last Returns the last (latest) completed round
func (compRounds *CompletedRounds[H, N]) Last() CompletedRound[H, N] {
	if len(compRounds.Rounds) == 0 {
		panic("inner is never empty; always contains at least genesis; qed")
	}
	return compRounds.Rounds[0]
}

// Push a new completed round, oldest round is evicted if number of rounds
// is higher than `NUM_LAST_COMPLETED_ROUNDS`.
func (compRounds *CompletedRounds[H, N]) Push(completedRound CompletedRound[H, N]) {
	// TODO they use reverse, double check this
	idx, found := slices.BinarySearchFunc(
		compRounds.Rounds,
		N(completedRound.Number),
		func(a CompletedRound[H, N], b N) int {
			switch {
			case N(a.Number) == b:
				return 0
			case N(a.Number) > b:
				return 1
			case N(a.Number) < b:
				return -1
			default:
				panic("huh?")
			}
		},
	)

	if found {
		compRounds.Rounds[idx] = completedRound
	} else {
		if len(compRounds.Rounds) <= idx {
			compRounds.Rounds = append(compRounds.Rounds, completedRound)
		} else {
			compRounds.Rounds = append(compRounds.Rounds[:idx+1], compRounds.Rounds[idx:]...)
			compRounds.Rounds[idx] = completedRound
		}
	}

	if len(compRounds.Rounds) > NumLastCompletedRounds {
		compRounds.Rounds = compRounds.Rounds[:len(compRounds.Rounds)-1]
	}
}

// CurrentRounds A map with voter status information for currently live rounds,
// which votes have we cast and what are they.
type CurrentRounds[H comparable, N constraints.Unsigned] map[uint64]HasVoted[H, N]

// VoterSetState The state of the current voter set, whether it is currently active or not
// and information related to the previously completed rounds. Current round
// voting status is used when restarting the voter, i.e. it will re-use the
// previous votes for a given round if appropriate (same round and same local
// key).
type VoterSetState[H comparable, N constraints.Unsigned] scale.VaryingDataType

// Set will set a VaryingDataTypeValue using the underlying VaryingDataType
func (tve *VoterSetState[H, N]) Set(val scale.VaryingDataTypeValue) (err error) { //skipcq: GO-W1029
	vdt := scale.VaryingDataType(*tve)
	err = vdt.Set(val)
	if err != nil {
		return err
	}
	*tve = VoterSetState[H, N](vdt)
	return nil
}

// Value will return the value from the underlying VaryingDataType
func (tve *VoterSetState[H, N]) Value() (val scale.VaryingDataTypeValue, err error) { //skipcq: GO-W1029
	vdt := scale.VaryingDataType(*tve)
	return vdt.Value()
}

// New ...
func (tve VoterSetState[H, N]) New() *VoterSetState[H, N] {
	vdt, err := scale.NewVaryingDataType(Live[H, N]{
		CompletedRounds: CompletedRounds[H, N]{
			Rounds: make([]CompletedRound[H, N], 0, NumLastCompletedRounds),
		},
		CurrentRounds: make(map[uint64]HasVoted[H, N]), // init the map
	}, Paused[H, N]{})
	if err != nil {
		panic(err)
	}
	vs := VoterSetState[H, N](vdt)
	return &vs
}

// NewVoterSetState is constructor for VoterSetState
func NewVoterSetState[H comparable, N constraints.Unsigned]() *VoterSetState[H, N] {
	vdt, err := scale.NewVaryingDataType(Live[H, N]{
		CompletedRounds: CompletedRounds[H, N]{
			Rounds: make([]CompletedRound[H, N], 0, NumLastCompletedRounds),
		},
		CurrentRounds: make(map[uint64]HasVoted[H, N]), // init the map
	}, Paused[H, N]{})
	if err != nil {
		panic(err)
	}
	tve := VoterSetState[H, N](vdt)
	return &tve
}

// Live Create a new live VoterSetState with round 0 as a completed round using
// the given genesis state and the given authorities. Round 1 is added as a
// current round (with state `HasVoted::No`).
func (tve *VoterSetState[H, N]) Live(setId uint64, authSet AuthoritySet[H, N], genesisState grandpa.HashNumber[H, N]) (VoterSetState[H, N], error) {
	state := grandpa.NewRoundState[H, N](genesisState)
	completedRounds := NewCompletedRounds[H, N](
		&CompletedRound[H, N]{
			State: state,
			Base:  genesisState,
		},
		setId,
		authSet,
	)
	currentRounds := CurrentRounds[H, N](
		make(map[uint64]HasVoted[H, N]),
	)
	hasVoted := &HasVoted[H, N]{}
	hasVoted = hasVoted.New()
	err := hasVoted.Set(No{})
	if err != nil {
		return VoterSetState[H, N]{}, err
	}
	currentRounds[1] = *hasVoted

	voterSetState := *NewVoterSetState[H, N]()
	err = voterSetState.Set(Live[H, N]{
		CompletedRounds: completedRounds,
		CurrentRounds:   currentRounds,
	})

	if err != nil {
		return VoterSetState[H, N]{}, err
	}
	return voterSetState, nil
}

// CompletedRounds Returns the last completed rounds
func (tve *VoterSetState[H, N]) CompletedRounds() (CompletedRounds[H, N], error) {
	value, err := tve.Value()
	if err != nil {
		return CompletedRounds[H, N]{}, err
	}
	switch v := value.(type) {
	case Live[H, N]:
		return v.CompletedRounds, nil
	case Paused[H, N]:
		return v.CompletedRounds, nil
	default:
		panic("CompletedRounds: invalid voter set state")
	}
}

// LastCompletedRound Returns the last completed round
func (tve *VoterSetState[H, N]) LastCompletedRound() (CompletedRound[H, N], error) {
	value, err := tve.Value()
	if err != nil {
		return CompletedRound[H, N]{}, err
	}
	switch v := value.(type) {
	case Live[H, N]:
		return v.CompletedRounds.Last(), nil
	case Paused[H, N]:
		return v.CompletedRounds.Last(), nil
	default:
		panic("CompletedRounds: invalid voter set state")
	}
}

// WithCurrentRound Returns the voter set state validating that it includes the given round
// in current rounds and that the voter isn't paused
func (tve *VoterSetState[H, N]) WithCurrentRound(round uint64) (CompletedRounds[H, N], CurrentRounds[H, N], error) {
	value, err := tve.Value()
	if err != nil {
		return CompletedRounds[H, N]{}, CurrentRounds[H, N]{}, err
	}
	switch v := value.(type) {
	case Live[H, N]:
		_, contains := v.CurrentRounds[round]
		if contains {
			return v.CompletedRounds, v.CurrentRounds, nil
		}
		return CompletedRounds[H, N]{}, CurrentRounds[H, N]{}, fmt.Errorf("voter acting on a live round we are not tracking")
	case Paused[H, N]:
		return CompletedRounds[H, N]{}, CurrentRounds[H, N]{}, fmt.Errorf("voter acting while in paused state")
	default:
		panic("CompletedRounds: invalid voter set state")
	}
}

// Live The voter is live, i.e. participating in rounds.
type Live[H comparable, N constraints.Unsigned] struct {
	// The previously completed rounds
	CompletedRounds CompletedRounds[H, N]
	// Voter status for the currently live rounds.
	CurrentRounds CurrentRounds[H, N]
}

// Index returns VDT index
func (Live[H, N]) Index() uint { return 0 }

// Paused The voter is paused, i.e. not casting or importing any votes.
type Paused[H comparable, N constraints.Unsigned] struct {
	// The previously completed rounds
	CompletedRounds CompletedRounds[H, N]
}

// Index returns VDT index
func (Paused[H, N]) Index() uint { return 1 }

// HasVoted Whether we've voted already during a prior run of the program
type HasVoted[H comparable, N constraints.Unsigned] scale.VaryingDataType

// Set will set a VaryingDataTypeValue using the underlying VaryingDataType
func (hasVoted *HasVoted[H, N]) Set(val scale.VaryingDataTypeValue) (err error) { //skipcq: GO-W1029
	vdt := scale.VaryingDataType(*hasVoted)
	err = vdt.Set(val)
	if err != nil {
		return err
	}
	*hasVoted = HasVoted[H, N](vdt)
	return nil
}

// Value will return the value from the underlying VaryingDataType
func (hasVoted *HasVoted[H, N]) Value() (val scale.VaryingDataTypeValue, err error) { //skipcq: GO-W1029
	vdt := scale.VaryingDataType(*hasVoted)
	return vdt.Value()
}

// New is constructor for HasVoted
func (hasVoted HasVoted[H, N]) New() *HasVoted[H, N] {
	vdt, _ := scale.NewVaryingDataType(No{}, Yes[H, N]{})

	hv := HasVoted[H, N](vdt)
	return &hv
}

// No Has not voted already in this round
type No struct{}

// Index returns VDT index
func (No) Index() uint { return 0 }

// Yes Has voted in this round
type Yes[H comparable, N constraints.Unsigned] struct {
	AuthId ed25519.PublicKey
	Vote   Vote[H, N]
}

// Index returns VDT index
func (Yes[H, N]) Index() uint { return 1 }

func (Yes[H, N]) New() *Yes[H, N] {
	vote := &Vote[H, N]{}
	vote = vote.New()
	return &Yes[H, N]{
		Vote: *vote,
	}
}

// Propose Returns the proposal we should vote with (if any.)
func (hasVoted *HasVoted[H, N]) Propose() *grandpa.PrimaryPropose[H, N] {
	value, err := hasVoted.Value()
	if err != nil {
		return nil
	}
	switch v := value.(type) {
	case Yes[H, N]:
		value, err = v.Vote.Value()
		if err != nil {
			return nil
		}
		switch vote := value.(type) {
		case Propose[H, N]:
			return &vote.PrimaryPropose
		case Prevote[H, N]:
			return vote.PrimaryPropose
		case Precommit[H, N]:
			return vote.PrimaryPropose
		}
	}

	return nil
}

// Prevote Returns the prevote we should vote with (if any.)
func (hasVoted *HasVoted[H, N]) Prevote() *grandpa.Prevote[H, N] {
	value, err := hasVoted.Value()
	if err != nil {
		return nil
	}
	switch v := value.(type) {
	case Yes[H, N]:
		value, err = v.Vote.Value()
		if err != nil {
			return nil
		}
		switch vote := value.(type) {
		case Prevote[H, N]:
			return &vote.Vote
		case Precommit[H, N]:
			return &vote.Vote
		}
	}

	return nil
}

// Precommit Returns the precommit we should vote with (if any.)
func (hasVoted *HasVoted[H, N]) Precommit() *grandpa.Precommit[H, N] {
	value, err := hasVoted.Value()
	if err != nil {
		return nil
	}
	switch v := value.(type) {
	case Yes[H, N]:
		value, err = v.Vote.Value()
		if err != nil {
			return nil
		}
		switch vote := value.(type) {
		case Precommit[H, N]:
			return &vote.Commit
		}
	}

	return nil
}

// CanPropose Returns true if the voter can still propose, false otherwise
func (hasVoted *HasVoted[H, N]) CanPropose() bool {
	return hasVoted.Propose() == nil
}

// CanPrevote Returns true if the voter can still prevote, false otherwise
func (hasVoted *HasVoted[H, N]) CanPrevote() bool {
	return hasVoted.Prevote() == nil
}

// CanPrecommit Returns true if the voter can still precommit, false otherwise
func (hasVoted *HasVoted[H, N]) CanPrecommit() bool {
	return hasVoted.Precommit() == nil
}

// Vote Whether we've voted already during a prior run of the program
type Vote[H comparable, N constraints.Unsigned] scale.VaryingDataType

// Set will set a VaryingDataTypeValue using the underlying VaryingDataType
func (vote *Vote[H, N]) Set(val scale.VaryingDataTypeValue) (err error) { //skipcq: GO-W1029
	vdt := scale.VaryingDataType(*vote)
	err = vdt.Set(val)
	if err != nil {
		return err
	}
	*vote = Vote[H, N](vdt)
	return nil
}

// Value will return the value from the underlying VaryingDataType
func (vote *Vote[H, N]) Value() (val scale.VaryingDataTypeValue, err error) { //skipcq: GO-W1029
	vdt := scale.VaryingDataType(*vote)
	return vdt.Value()
}

// New is constructor for Vote
func (vote Vote[H, N]) New() *Vote[H, N] {
	vdt, err := scale.NewVaryingDataType(Propose[H, N]{}, Prevote[H, N]{}, Precommit[H, N]{})
	if err != nil {
		panic(err)
	}
	hv := Vote[H, N](vdt)
	return &hv
}

// Propose Has cast a proposal
type Propose[H comparable, N constraints.Unsigned] struct {
	PrimaryPropose grandpa.PrimaryPropose[H, N]
}

// Index returns VDT index
func (Propose[H, N]) Index() uint { return 0 }

// Prevote Has cast a prevote
type Prevote[H comparable, N constraints.Unsigned] struct {
	PrimaryPropose *grandpa.PrimaryPropose[H, N]
	Vote           grandpa.Prevote[H, N]
}

// Index returns VDT index
func (Prevote[H, N]) Index() uint { return 1 }

// Precommit Has cast a precommit (implies prevote.)
type Precommit[H comparable, N constraints.Unsigned] struct {
	PrimaryPropose *grandpa.PrimaryPropose[H, N]
	Vote           grandpa.Prevote[H, N]
	Commit         grandpa.Precommit[H, N]
}

// Index returns VDT index
func (Precommit[H, N]) Index() uint { return 2 }

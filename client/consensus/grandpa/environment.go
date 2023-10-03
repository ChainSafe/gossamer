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
	"sync"
)

// completedRound Data about a completed round. The set of votes that is stored must be
// minimal, i.e. at most one equivocation is stored per voter.
type completedRound[H comparable, N constraints.Unsigned] struct {
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

// completedRounds Data about last completed rounds within a single voter set. Stores
// NumLastCompletedRounds and always contains data about at least one round
// (genesis).
type completedRounds[H comparable, N constraints.Unsigned] struct {
	Rounds []completedRound[H, N]
	SetId  uint64
	Voters []ed25519.PublicKey
}

// NewCompletedRounds Create a new completed rounds tracker with NUM_LAST_COMPLETED_ROUNDS capacity.
func NewCompletedRounds[H comparable, N constraints.Unsigned](genesis *completedRound[H, N], setId uint64, voters AuthoritySet[H, N]) completedRounds[H, N] {
	rounds := make([]completedRound[H, N], 0, NumLastCompletedRounds)
	if genesis != nil {
		rounds = append(rounds, *genesis)
	}

	var voterIDs []ed25519.PublicKey
	currentAuthorities := voters.CurrentAuthorities
	for _, auth := range currentAuthorities {
		voterIDs = append(voterIDs, auth.Key)
	}

	return completedRounds[H, N]{
		rounds,
		setId,
		voterIDs,
	}
}

// last Returns the last (latest) completed round
func (compRounds *completedRounds[H, N]) last() completedRound[H, N] {
	if len(compRounds.Rounds) == 0 {
		panic("inner is never empty; always contains at least genesis; qed")
	}
	return compRounds.Rounds[0]
}

// push a new completed round, oldest round is evicted if number of rounds
// is higher than `NUM_LAST_COMPLETED_ROUNDS`.
func (compRounds *completedRounds[H, N]) push(compRound completedRound[H, N]) {
	// TODO they use reverse, double check this
	idx, found := slices.BinarySearchFunc(
		compRounds.Rounds,
		N(compRound.Number),
		func(a completedRound[H, N], b N) int {
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
		compRounds.Rounds[idx] = compRound
	} else {
		if len(compRounds.Rounds) <= idx {
			compRounds.Rounds = append(compRounds.Rounds, compRound)
		} else {
			compRounds.Rounds = append(compRounds.Rounds[:idx+1], compRounds.Rounds[idx:]...)
			compRounds.Rounds[idx] = compRound
		}
	}

	if len(compRounds.Rounds) > NumLastCompletedRounds {
		compRounds.Rounds = compRounds.Rounds[:len(compRounds.Rounds)-1]
	}
}

// CurrentRounds A map with voter status information for currently live rounds,
// which votes have we cast and what are they.
// TODO convert to btree after #3480 is implemented
type CurrentRounds[H comparable, N constraints.Unsigned] map[uint64]hasVoted[H, N]

// A tracker for the rounds that we are actively participating on (i.e. voting)
// and the authority id under which we are doing it.
type votingTracker struct {
	sync.Mutex
	Inner map[uint64]ed25519.PublicKey
}

type sharedVoterSetState[H comparable, N constraints.Unsigned] struct {
	sync.Mutex
	Inner voterSetState[H, N]
}

// SharedVoterSetState A voter set state meant to be shared safely across multiple owners
type SharedVoterSetState[H comparable, N constraints.Unsigned] struct {
	Inner  sharedVoterSetState[H, N]
	Voting votingTracker
}

// NewSharedVoterSetState Create a new shared voter set tracker with the given state.
func NewSharedVoterSetState[H comparable, N constraints.Unsigned](state voterSetState[H, N]) SharedVoterSetState[H, N] {
	return SharedVoterSetState[H, N]{
		Inner: sharedVoterSetState[H, N]{
			Inner: state,
		},
	}
}

// Read the inner voter set state
func (svss *SharedVoterSetState[H, N]) read() voterSetState[H, N] {
	svss.Inner.Lock()
	defer svss.Inner.Unlock()
	return svss.Inner.Inner
}

// Get the authority id that we are using to vote on the given round, if any
func (svss *SharedVoterSetState[H, N]) votingOn(round uint64) *ed25519.PublicKey {
	svss.Voting.Lock()
	defer svss.Voting.Unlock()
	key, ok := svss.Voting.Inner[round]
	if !ok {
		return nil
	}
	return &key
}

// Note that we started voting on the give round with the given authority id
func (svss *SharedVoterSetState[H, N]) startedVotingOn(round uint64, localID ed25519.PublicKey) {
	svss.Voting.Lock()
	defer svss.Voting.Unlock()
	svss.Voting.Inner[round] = localID
}

// Note that we have finished voting on the given round. If we were voting on
// the given round, the authority id that we were using to do it will be
// cleared.
func (svss *SharedVoterSetState[H, N]) finishedVotingOn(round uint64) {
	svss.Voting.Lock()
	defer svss.Voting.Unlock()
	delete(svss.Voting.Inner, round)
}

// Return vote status information for the current round
func (svss *SharedVoterSetState[H, N]) hasVoted(round uint64) (hasVoted[H, N], error) {
	svss.Inner.Lock()
	defer svss.Inner.Unlock()

	hasNotVotedFunc := func(newHasVoted hasVoted[H, N]) (hasVoted[H, N], error) {
		err := newHasVoted.Set(no{})
		if err != nil {
			return newHasVoted, err
		}

		return newHasVoted, nil
	}

	newHasVoted := hasVoted[H, N]{}
	newHasVoted = newHasVoted.New()

	vss, err := svss.Inner.Inner.Value()
	if err != nil {
		// Believe this is return hasVoted::No, but TODO check in review
		return hasNotVotedFunc(newHasVoted)
	}
	switch val := vss.(type) {
	case voterSetStateLive[H, N]:
		hasVoted, ok := val.CurrentRounds[round]
		if !ok {
			return hasNotVotedFunc(newHasVoted)
		}

		hasVotedValue, err := hasVoted.Value()
		if err != nil {
			return newHasVoted, err
		}
		switch hasVotedValue.(type) {
		case yes[H, N]:
			return hasVoted, nil
		}
	}

	return hasNotVotedFunc(newHasVoted)
}

// voterSetState The state of the current voter set, whether it is currently active or not
// and information related to the previously completed rounds. Current round
// voting status is used when restarting the voter, i.e. it will re-use the
// previous votes for a given round if appropriate (same round and same local
// key).
type voterSetState[H comparable, N constraints.Unsigned] scale.VaryingDataType

// Set will set a VaryingDataTypeValue using the underlying VaryingDataType
func (tve *voterSetState[H, N]) Set(val scale.VaryingDataTypeValue) (err error) { //skipcq: GO-W1029
	vdt := scale.VaryingDataType(*tve)
	err = vdt.Set(val)
	if err != nil {
		return err
	}
	*tve = voterSetState[H, N](vdt)
	return nil
}

// Value will return the value from the underlying VaryingDataType
func (tve *voterSetState[H, N]) Value() (val scale.VaryingDataTypeValue, err error) { //skipcq: GO-W1029
	vdt := scale.VaryingDataType(*tve)
	return vdt.Value()
}

// New Creates a new voterSetState
func (tve voterSetState[H, N]) New() voterSetState[H, N] {
	vdt, err := scale.NewVaryingDataType(voterSetStateLive[H, N]{
		CompletedRounds: completedRounds[H, N]{
			Rounds: make([]completedRound[H, N], 0, NumLastCompletedRounds),
		},
		CurrentRounds: make(map[uint64]hasVoted[H, N]), // init the map
	}, voterSetStatePaused[H, N]{})
	if err != nil {
		panic(err)
	}
	vs := voterSetState[H, N](vdt)
	return vs
}

// NewVoterSetState is constructor for voterSetState
func NewVoterSetState[H comparable, N constraints.Unsigned]() *voterSetState[H, N] {
	vdt, err := scale.NewVaryingDataType(voterSetStateLive[H, N]{
		CompletedRounds: completedRounds[H, N]{
			Rounds: make([]completedRound[H, N], 0, NumLastCompletedRounds),
		},
		CurrentRounds: make(map[uint64]hasVoted[H, N]), // init the map
	}, voterSetStatePaused[H, N]{})
	if err != nil {
		panic(err)
	}
	tve := voterSetState[H, N](vdt)
	return &tve
}

// NewLiveVoterSetState Create a new live voterSetState with round 0 as a completed round using
// the given genesis state and the given authorities. Round 1 is added as a
// current round (with state `hasVoted::no`).
func NewLiveVoterSetState[H comparable, N constraints.Unsigned](setId uint64, authSet AuthoritySet[H, N], genesisState grandpa.HashNumber[H, N]) (voterSetState[H, N], error) {
	state := grandpa.NewRoundState[H, N](genesisState)
	completedRounds := NewCompletedRounds[H, N](
		&completedRound[H, N]{
			State: state,
			Base:  genesisState,
		},
		setId,
		authSet,
	)
	currentRounds := CurrentRounds[H, N](
		make(map[uint64]hasVoted[H, N]),
	)
	hasVoted := hasVoted[H, N]{}
	hasVoted = hasVoted.New()
	err := hasVoted.Set(no{})
	if err != nil {
		return voterSetState[H, N]{}, err
	}
	currentRounds[1] = hasVoted

	newVoterSetState := *NewVoterSetState[H, N]()
	err = newVoterSetState.Set(voterSetStateLive[H, N]{
		CompletedRounds: completedRounds,
		CurrentRounds:   currentRounds,
	})

	if err != nil {
		return voterSetState[H, N]{}, err
	}
	return newVoterSetState, nil
}

// completedRounds Returns the last completed rounds
func (tve *voterSetState[H, N]) completedRounds() (completedRounds[H, N], error) {
	value, err := tve.Value()
	if err != nil {
		return completedRounds[H, N]{}, err
	}
	switch v := value.(type) {
	case voterSetStateLive[H, N]:
		return v.CompletedRounds, nil
	case voterSetStatePaused[H, N]:
		return v.CompletedRounds, nil
	default:
		panic("completedRounds: invalid voter set state")
	}
}

// lastCompletedRound Returns the last completed round
func (tve *voterSetState[H, N]) lastCompletedRound() (completedRound[H, N], error) {
	value, err := tve.Value()
	if err != nil {
		return completedRound[H, N]{}, err
	}
	switch v := value.(type) {
	case voterSetStateLive[H, N]:
		return v.CompletedRounds.last(), nil
	case voterSetStatePaused[H, N]:
		return v.CompletedRounds.last(), nil
	default:
		panic("completedRounds: invalid voter set state")
	}
}

// withCurrentRound Returns the voter set state validating that it includes the given round
// in current rounds and that the voter isn't paused
func (tve *voterSetState[H, N]) withCurrentRound(round uint64) (completedRounds[H, N], CurrentRounds[H, N], error) {
	value, err := tve.Value()
	if err != nil {
		return completedRounds[H, N]{}, CurrentRounds[H, N]{}, err
	}
	switch v := value.(type) {
	case voterSetStateLive[H, N]:
		_, contains := v.CurrentRounds[round]
		if contains {
			return v.CompletedRounds, v.CurrentRounds, nil
		}
		return completedRounds[H, N]{}, CurrentRounds[H, N]{}, fmt.Errorf("voter acting on a live round we are not tracking")
	case voterSetStatePaused[H, N]:
		return completedRounds[H, N]{}, CurrentRounds[H, N]{}, fmt.Errorf("voter acting while in paused state")
	default:
		panic("completedRounds: invalid voter set state")
	}
}

// voterSetStateLive The voter is live, i.e. participating in rounds.
type voterSetStateLive[H comparable, N constraints.Unsigned] struct {
	// The previously completed rounds
	CompletedRounds completedRounds[H, N]
	// Voter status for the currently live rounds.
	CurrentRounds CurrentRounds[H, N]
}

// Index returns VDT index
func (voterSetStateLive[H, N]) Index() uint { return 0 }

// voterSetStatePaused The voter is paused, i.e. not casting or importing any votes.
type voterSetStatePaused[H comparable, N constraints.Unsigned] struct {
	// The previously completed rounds
	CompletedRounds completedRounds[H, N]
}

// Index returns VDT index
func (voterSetStatePaused[H, N]) Index() uint { return 1 }

// hasVoted Whether we've voted already during a prior run of the program
type hasVoted[H comparable, N constraints.Unsigned] scale.VaryingDataType

// Set will set a VaryingDataTypeValue using the underlying VaryingDataType
func (hv *hasVoted[H, N]) Set(val scale.VaryingDataTypeValue) (err error) { //skipcq: GO-W1029
	vdt := scale.VaryingDataType(*hv)
	err = vdt.Set(val)
	if err != nil {
		return err
	}
	*hv = hasVoted[H, N](vdt)
	return nil
}

// Value will return the value from the underlying VaryingDataType
func (hv *hasVoted[H, N]) Value() (val scale.VaryingDataTypeValue, err error) { //skipcq: GO-W1029
	vdt := scale.VaryingDataType(*hv)
	return vdt.Value()
}

// New is constructor for hasVoted
func (hv hasVoted[H, N]) New() hasVoted[H, N] {
	vdt, _ := scale.NewVaryingDataType(no{}, yes[H, N]{})

	newHv := hasVoted[H, N](vdt)
	return newHv
}

// no Has not voted already in this round
type no struct{}

// Index returns VDT index
func (no) Index() uint { return 0 }

// yes Has voted in this round
type yes[H comparable, N constraints.Unsigned] struct {
	AuthId ed25519.PublicKey
	Vote   vote[H, N]
}

// Index returns VDT index
func (yes[H, N]) Index() uint { return 1 }

func (yes[H, N]) New() yes[H, N] {
	vote := vote[H, N]{}
	vote = vote.New()
	return yes[H, N]{
		Vote: vote,
	}
}

// propose Returns the proposal we should vote with (if any.)
func (hasVoted *hasVoted[H, N]) Propose() *grandpa.PrimaryPropose[H, N] {
	value, err := hasVoted.Value()
	if err != nil {
		return nil
	}
	switch v := value.(type) {
	case yes[H, N]:
		value, err = v.Vote.Value()
		if err != nil {
			return nil
		}
		switch vote := value.(type) {
		case propose[H, N]:
			return &vote.PrimaryPropose
		case prevote[H, N]:
			return vote.PrimaryPropose
		case precommit[H, N]:
			return vote.PrimaryPropose
		}
	}

	return nil
}

// prevote Returns the prevote we should vote with (if any.)
func (hasVoted *hasVoted[H, N]) Prevote() *grandpa.Prevote[H, N] {
	value, err := hasVoted.Value()
	if err != nil {
		return nil
	}
	switch v := value.(type) {
	case yes[H, N]:
		value, err = v.Vote.Value()
		if err != nil {
			return nil
		}
		switch vote := value.(type) {
		case prevote[H, N]:
			return &vote.Vote
		case precommit[H, N]:
			return &vote.Vote
		}
	}

	return nil
}

// precommit Returns the precommit we should vote with (if any.)
func (hasVoted *hasVoted[H, N]) Precommit() *grandpa.Precommit[H, N] {
	value, err := hasVoted.Value()
	if err != nil {
		return nil
	}
	switch v := value.(type) {
	case yes[H, N]:
		value, err = v.Vote.Value()
		if err != nil {
			return nil
		}
		switch vote := value.(type) {
		case precommit[H, N]:
			return &vote.Commit
		}
	}

	return nil
}

// CanPropose Returns true if the voter can still propose, false otherwise
func (hasVoted *hasVoted[H, N]) CanPropose() bool {
	return hasVoted.Propose() == nil
}

// CanPrevote Returns true if the voter can still prevote, false otherwise
func (hasVoted *hasVoted[H, N]) CanPrevote() bool {
	return hasVoted.Prevote() == nil
}

// CanPrecommit Returns true if the voter can still precommit, false otherwise
func (hasVoted *hasVoted[H, N]) CanPrecommit() bool {
	return hasVoted.Precommit() == nil
}

// vote Whether we've voted already during a prior run of the program
type vote[H comparable, N constraints.Unsigned] scale.VaryingDataType

// Set will set a VaryingDataTypeValue using the underlying VaryingDataType
func (v *vote[H, N]) Set(val scale.VaryingDataTypeValue) (err error) { //skipcq: GO-W1029
	vdt := scale.VaryingDataType(*v)
	err = vdt.Set(val)
	if err != nil {
		return err
	}
	*v = vote[H, N](vdt)
	return nil
}

// Value will return the value from the underlying VaryingDataType
func (v *vote[H, N]) Value() (val scale.VaryingDataTypeValue, err error) { //skipcq: GO-W1029
	vdt := scale.VaryingDataType(*v)
	return vdt.Value()
}

// New is constructor for vote
func (v vote[H, N]) New() vote[H, N] {
	vdt, err := scale.NewVaryingDataType(propose[H, N]{}, prevote[H, N]{}, precommit[H, N]{})
	if err != nil {
		panic(err)
	}
	newV := vote[H, N](vdt)
	return newV
}

// propose Has cast a proposal
type propose[H comparable, N constraints.Unsigned] struct {
	PrimaryPropose grandpa.PrimaryPropose[H, N]
}

// Index returns VDT index
func (propose[H, N]) Index() uint { return 0 }

// prevote Has cast a prevote
type prevote[H comparable, N constraints.Unsigned] struct {
	PrimaryPropose *grandpa.PrimaryPropose[H, N]
	Vote           grandpa.Prevote[H, N]
}

// Index returns VDT index
func (prevote[H, N]) Index() uint { return 1 }

// precommit Has cast a precommit (implies prevote.)
type precommit[H comparable, N constraints.Unsigned] struct {
	PrimaryPropose *grandpa.PrimaryPropose[H, N]
	Vote           grandpa.Prevote[H, N]
	Commit         grandpa.Precommit[H, N]
}

// Index returns VDT index
func (precommit[H, N]) Index() uint { return 2 }

// Copyright 2023 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package grandpa

import (
	"fmt"
	"sync"

	grandpa "github.com/ChainSafe/gossamer/pkg/finality-grandpa"
	"github.com/ChainSafe/gossamer/pkg/scale"
	"golang.org/x/exp/constraints"
	"golang.org/x/exp/slices"
)

// completedRound Data about a completed round. The set of votes that is stored must be
// minimal, i.e. at most one equivocation is stored per voter.
type completedRound[H comparable, N constraints.Unsigned, ID AuthorityID, Sig AuthoritySignature] struct {
	// The round number
	Number uint64
	// The round state (prevote ghost, estimate, finalized, etc.)
	State grandpa.RoundState[H, N]
	// The target block base used for voting in the round
	Base grandpa.HashNumber[H, N]
	// All the votes observed in the round
	// I think this is signature type, double check
	Votes []grandpa.SignedMessage[H, N, ID, Sig]
}

// numLastCompletedRounds NOTE: the current strategy for persisting completed rounds is very naive
// (update everything) and we also rely on cloning to do atomic updates,
// therefore this value should be kept small for now.
const numLastCompletedRounds = 2

// completedRounds Data about last completed rounds within a single voter set. Stores
// numLastCompletedRounds and always contains data about at least one round
// (genesis).
type completedRounds[H comparable, N constraints.Unsigned, ID AuthorityID, Sig AuthoritySignature] struct {
	Rounds []completedRound[H, N, ID, Sig]
	SetId  uint64
	Voters []ID
}

// NewCompletedRounds Create a new completed rounds tracker with NUM_LAST_COMPLETED_ROUNDS capacity.
func NewCompletedRounds[H comparable, N constraints.Unsigned, ID AuthorityID, Sig AuthoritySignature](
	genesis completedRound[H, N, ID, Sig],
	setId uint64,
	voters AuthoritySet[H, N, ID]) completedRounds[H, N, ID, Sig] {
	rounds := make([]completedRound[H, N, ID, Sig], 0, numLastCompletedRounds)
	rounds = append(rounds, genesis)

	var voterIDs []ID
	currentAuthorities := voters.CurrentAuthorities
	for _, auth := range currentAuthorities {
		voterIDs = append(voterIDs, auth.Key)
	}

	return completedRounds[H, N, ID, Sig]{
		rounds,
		setId,
		voterIDs,
	}
}

func (cr *completedRounds[H, N, ID, Sig]) iter() []completedRound[H, N, ID, Sig] {
	var reversed []completedRound[H, N, ID, Sig]
	for i := len(cr.Rounds) - 1; i >= 0; i-- {
		reversed = append(reversed, cr.Rounds[i])
	}
	return reversed
}

// last Returns the last (latest) completed round
func (cr *completedRounds[H, N, ID, Sig]) last() completedRound[H, N, ID, Sig] {
	if len(cr.Rounds) == 0 {
		panic("inner is never empty; always contains at least genesis; qed")
	}
	return cr.Rounds[0]
}

// push a new completed round, oldest round is evicted if number of rounds
// is higher than `NUM_LAST_COMPLETED_ROUNDS`.
func (cr *completedRounds[H, N, ID, Sig]) push(compRound completedRound[H, N, ID, Sig]) {
	idx, found := slices.BinarySearchFunc(
		cr.Rounds,
		N(compRound.Number),
		func(a completedRound[H, N, ID, Sig], b N) int {
			switch {
			case N(a.Number) == b:
				return 0
			case N(a.Number) < b:
				return 1
			case N(a.Number) > b:
				return -1
			default:
				panic("invalid result in binary search")
			}
		},
	)

	if found {
		cr.Rounds[idx] = compRound
	} else {
		if len(cr.Rounds) <= idx {
			cr.Rounds = append(cr.Rounds, compRound)
		} else {
			cr.Rounds = append(cr.Rounds[:idx+1], cr.Rounds[idx:]...)
			cr.Rounds[idx] = compRound
		}
	}

	if len(cr.Rounds) > numLastCompletedRounds {
		cr.Rounds = cr.Rounds[:len(cr.Rounds)-1]
	}
}

// CurrentRounds A map with voter status information for currently live rounds,
// which votes have we cast and what are they.
// TODO convert to btree after #3480 is implemented
type CurrentRounds[H comparable, N constraints.Unsigned, ID AuthorityID] map[uint64]hasVoted[H, N, ID]

// A tracker for the rounds that we are actively participating on (i.e. voting)
// and the authority id under which we are doing it.
type votingTracker[ID AuthorityID] struct {
	sync.Mutex
	Inner map[uint64]ID
}

type sharedVoterSetState[H comparable, N constraints.Unsigned, ID AuthorityID, Sig AuthoritySignature] struct {
	sync.Mutex
	Inner voterSetState[H, N, ID, Sig]
}

// SharedVoterSetState A voter set state meant to be shared safely across multiple owners
type SharedVoterSetState[H comparable, N constraints.Unsigned, ID AuthorityID, Sig AuthoritySignature] struct {
	Inner  sharedVoterSetState[H, N, ID, Sig]
	Voting votingTracker[ID]
}

// NewSharedVoterSetState Create a new shared voter set tracker with the given state.
func NewSharedVoterSetState[H comparable, N constraints.Unsigned, ID AuthorityID, Sig AuthoritySignature](
	state voterSetState[H, N, ID, Sig]) SharedVoterSetState[H, N, ID, Sig] {
	return SharedVoterSetState[H, N, ID, Sig]{
		Inner: sharedVoterSetState[H, N, ID, Sig]{
			Inner: state,
		},
	}
}

// Read the inner voter set state
func (svss *SharedVoterSetState[H, N, ID, Sig]) read() voterSetState[H, N, ID, Sig] { //nolint
	svss.Inner.Lock()
	defer svss.Inner.Unlock()
	return svss.Inner.Inner
}

// Get the authority id that we are using to vote on the given round, if any
func (svss *SharedVoterSetState[H, N, ID, Sig]) votingOn(round uint64) *ID { //nolint
	svss.Voting.Lock()
	defer svss.Voting.Unlock()
	key, ok := svss.Voting.Inner[round]
	if !ok {
		return nil
	}
	return &key
}

// Note that we started voting on the give round with the given authority id
func (svss *SharedVoterSetState[H, N, ID, Sig]) startedVotingOn(round uint64, localID ID) { //nolint
	svss.Voting.Lock()
	defer svss.Voting.Unlock()
	svss.Voting.Inner[round] = localID
}

// Note that we have finished voting on the given round. If we were voting on
// the given round, the authority id that we were using to do it will be
// cleared.
func (svss *SharedVoterSetState[H, N, ID, Sig]) finishedVotingOn(round uint64) { //nolint
	svss.Voting.Lock()
	defer svss.Voting.Unlock()
	delete(svss.Voting.Inner, round)
}

// Return vote status information for the current round
func (svss *SharedVoterSetState[H, N, ID, Sig]) hasVoted(round uint64) (hasVoted[H, N, ID], error) {
	svss.Inner.Lock()
	defer svss.Inner.Unlock()

	hasNotVotedFunc := func(newHasVoted hasVoted[H, N, ID]) (hasVoted[H, N, ID], error) {
		err := newHasVoted.Set(no{})
		if err != nil {
			return newHasVoted, err
		}

		return newHasVoted, nil
	}

	newHasVoted := hasVoted[H, N, ID]{}
	newHasVoted = newHasVoted.New()

	vss, err := svss.Inner.Inner.Value()
	if err != nil {
		// Believe this is return hasVoted::No, but TODO check in review
		return hasNotVotedFunc(newHasVoted)
	}
	switch val := vss.(type) {
	case voterSetStateLive[H, N, ID, Sig]:
		hasVoted, ok := val.CurrentRounds[round]
		if !ok {
			return hasNotVotedFunc(newHasVoted)
		}

		hasVotedValue, err := hasVoted.Value()
		if err != nil {
			return newHasVoted, err
		}
		switch hasVotedValue.(type) {
		case yes[H, N, ID]:
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
type voterSetState[H comparable, N constraints.Unsigned, ID AuthorityID, Sig AuthoritySignature] scale.VaryingDataType

// Set will set a VaryingDataTypeValue using the underlying VaryingDataType
func (tve *voterSetState[H, N, ID, Sig]) Set(val scale.VaryingDataTypeValue) (err error) { //skipcq: GO-W1029
	vdt := scale.VaryingDataType(*tve)
	err = vdt.Set(val)
	if err != nil {
		return err
	}
	*tve = voterSetState[H, N, ID, Sig](vdt)
	return nil
}

// Value will return the value from the underlying VaryingDataType
func (tve *voterSetState[H, N, ID, Sig]) Value() (val scale.VaryingDataTypeValue, err error) { //skipcq: GO-W1029
	vdt := scale.VaryingDataType(*tve)
	return vdt.Value()
}

// New Creates a new voterSetState
func (tve voterSetState[H, N, ID, Sig]) New() voterSetState[H, N, ID, Sig] {
	vdt, err := scale.NewVaryingDataType(voterSetStateLive[H, N, ID, Sig]{
		CompletedRounds: completedRounds[H, N, ID, Sig]{
			Rounds: make([]completedRound[H, N, ID, Sig], 0, numLastCompletedRounds),
		},
		CurrentRounds: make(map[uint64]hasVoted[H, N, ID]), // init the map
	}, voterSetStatePaused[H, N, ID, Sig]{})
	if err != nil {
		panic(err)
	}
	vs := voterSetState[H, N, ID, Sig](vdt)
	return vs
}

// NewVoterSetState is constructor for voterSetState
func NewVoterSetState[
	H comparable,
	N constraints.Unsigned,
	ID AuthorityID,
	Sig AuthoritySignature]() *voterSetState[H, N, ID, Sig] {
	vdt, err := scale.NewVaryingDataType(voterSetStateLive[H, N, ID, Sig]{
		CompletedRounds: completedRounds[H, N, ID, Sig]{
			Rounds: make([]completedRound[H, N, ID, Sig], 0, numLastCompletedRounds),
		},
		CurrentRounds: make(map[uint64]hasVoted[H, N, ID]), // init the map
	}, voterSetStatePaused[H, N, ID, Sig]{})
	if err != nil {
		panic(err)
	}
	tve := voterSetState[H, N, ID, Sig](vdt)
	return &tve
}

// NewLiveVoterSetState Create a new live voterSetState with round 0 as a completed round using
// the given genesis state and the given authorities. Round 1 is added as a
// current round (with state `hasVoted::no`).
func NewLiveVoterSetState[H comparable, N constraints.Unsigned, ID AuthorityID, Sig AuthoritySignature](
	setId uint64,
	authSet AuthoritySet[H, N, ID],
	genesisState grandpa.HashNumber[H, N]) (voterSetState[H, N, ID, Sig], error) {
	state := grandpa.NewRoundState[H, N](genesisState)
	completedRounds := NewCompletedRounds[H, N, ID, Sig](
		completedRound[H, N, ID, Sig]{
			State: state,
			Base:  genesisState,
		},
		setId,
		authSet,
	)
	//currentRounds := make(map[uint64]hasVoted[string, uint])
	currentRounds := CurrentRounds[H, N, ID](
		make(map[uint64]hasVoted[H, N, ID]),
	)
	hasVoted := hasVoted[H, N, ID]{}
	hasVoted = hasVoted.New()
	err := hasVoted.Set(no{})
	if err != nil {
		return voterSetState[H, N, ID, Sig]{}, err
	}
	currentRounds[1] = hasVoted

	liveState := voterSetStateLive[H, N, ID, Sig]{
		CompletedRounds: completedRounds,
		CurrentRounds:   currentRounds,
	}

	newVoterSetState := *NewVoterSetState[H, N, ID, Sig]()
	err = newVoterSetState.Set(liveState)

	if err != nil {
		return voterSetState[H, N, ID, Sig]{}, err
	}
	return newVoterSetState, nil
}

// completedRounds Returns the last completed rounds
func (tve *voterSetState[H, N, ID, Sig]) completedRounds() (completedRounds[H, N, ID, Sig], error) {
	value, err := tve.Value()
	if err != nil {
		return completedRounds[H, N, ID, Sig]{}, err
	}
	switch v := value.(type) {
	case voterSetStateLive[H, N, ID, Sig]:
		return v.CompletedRounds, nil
	case voterSetStatePaused[H, N, ID, Sig]:
		return v.CompletedRounds, nil
	default:
		panic("completedRounds: invalid voter set state")
	}
}

// lastCompletedRound Returns the last completed round
func (tve *voterSetState[H, N, ID, Sig]) lastCompletedRound() (completedRound[H, N, ID, Sig], error) {
	value, err := tve.Value()
	if err != nil {
		return completedRound[H, N, ID, Sig]{}, err
	}
	switch v := value.(type) {
	case voterSetStateLive[H, N, ID, Sig]:
		return v.CompletedRounds.last(), nil
	case voterSetStatePaused[H, N, ID, Sig]:
		return v.CompletedRounds.last(), nil
	default:
		panic("completedRounds: invalid voter set state")
	}
}

// withCurrentRound Returns the voter set state validating that it includes the given round
// in current rounds and that the voter isn't paused
func (tve *voterSetState[H, N, ID, Sig]) withCurrentRound(
	round uint64) (completedRounds[H, N, ID, Sig], CurrentRounds[H, N, ID], error) {
	value, err := tve.Value()
	if err != nil {
		return completedRounds[H, N, ID, Sig]{}, CurrentRounds[H, N, ID]{}, err
	}
	switch v := value.(type) {
	case voterSetStateLive[H, N, ID, Sig]:
		_, contains := v.CurrentRounds[round]
		if contains {
			return v.CompletedRounds, v.CurrentRounds, nil
		}
		return completedRounds[H, N, ID, Sig]{},
			CurrentRounds[H, N, ID]{},
			fmt.Errorf("voter acting on a live round we are not tracking")
	case voterSetStatePaused[H, N, ID, Sig]:
		return completedRounds[H, N, ID, Sig]{},
			CurrentRounds[H, N, ID]{},
			fmt.Errorf("voter acting while in paused state")
	default:
		panic("completedRounds: invalid voter set state")
	}
}

// voterSetStateLive The voter is live, i.e. participating in rounds.
type voterSetStateLive[H comparable, N constraints.Unsigned, ID AuthorityID, Sig AuthoritySignature] struct {
	// The previously completed rounds
	CompletedRounds completedRounds[H, N, ID, Sig]
	// Voter status for the currently live rounds.
	CurrentRounds CurrentRounds[H, N, ID]
}

// Index returns VDT index
func (voterSetStateLive[H, N, ID, Sig]) Index() uint { return 0 }

// voterSetStatePaused The voter is paused, i.e. not casting or importing any votes.
type voterSetStatePaused[H comparable, N constraints.Unsigned, ID AuthorityID, Sig AuthoritySignature] struct {
	// The previously completed rounds
	CompletedRounds completedRounds[H, N, ID, Sig]
}

// Index returns VDT index
func (voterSetStatePaused[H, N, ID, Sig]) Index() uint { return 1 }

// hasVoted Whether we've voted already during a prior run of the program
type hasVoted[H comparable, N constraints.Unsigned, ID AuthorityID] scale.VaryingDataType

// Set will set a VaryingDataTypeValue using the underlying VaryingDataType
func (hv *hasVoted[H, N, ID]) Set(val scale.VaryingDataTypeValue) (err error) { //skipcq: GO-W1029
	vdt := scale.VaryingDataType(*hv)
	err = vdt.Set(val)
	if err != nil {
		return err
	}
	*hv = hasVoted[H, N, ID](vdt)
	return nil
}

// Value will return the value from the underlying VaryingDataType
func (hv *hasVoted[H, N, ID]) Value() (val scale.VaryingDataTypeValue, err error) { //skipcq: GO-W1029
	vdt := scale.VaryingDataType(*hv)
	return vdt.Value()
}

// New is constructor for hasVoted
func (hv hasVoted[H, N, ID]) New() hasVoted[H, N, ID] {
	vdt, _ := scale.NewVaryingDataType(no{}, yes[H, N, ID]{})

	newHv := hasVoted[H, N, ID](vdt)
	return newHv
}

// no Has not voted already in this round
type no struct{}

// Index returns VDT index
func (no) Index() uint { return 0 }

// yes Has voted in this round
type yes[H comparable, N constraints.Unsigned, ID AuthorityID] struct {
	AuthId ID
	Vote   vote[H, N]
}

// Index returns VDT index
func (yes[H, N, ID]) Index() uint { return 1 }

func (yes[H, N, ID]) New() yes[H, N, ID] {
	vote := vote[H, N]{}
	vote = vote.New()
	return yes[H, N, ID]{
		Vote: vote,
	}
}

// propose Returns the proposal we should vote with (if any.)
func (hv *hasVoted[H, N, ID]) Propose() *grandpa.PrimaryPropose[H, N] {
	value, err := hv.Value()
	if err != nil {
		return nil
	}
	switch v := value.(type) {
	case yes[H, N, ID]:
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
func (hv *hasVoted[H, N, ID]) Prevote() *grandpa.Prevote[H, N] {
	value, err := hv.Value()
	if err != nil {
		return nil
	}
	switch v := value.(type) {
	case yes[H, N, ID]:
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
func (hv *hasVoted[H, N, ID]) Precommit() *grandpa.Precommit[H, N] {
	value, err := hv.Value()
	if err != nil {
		return nil
	}
	switch v := value.(type) {
	case yes[H, N, ID]:
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
func (hv *hasVoted[H, N, ID]) CanPropose() bool {
	return hv.Propose() == nil
}

// CanPrevote Returns true if the voter can still prevote, false otherwise
func (hv *hasVoted[H, N, ID]) CanPrevote() bool {
	return hv.Prevote() == nil
}

// CanPrecommit Returns true if the voter can still precommit, false otherwise
func (hv *hasVoted[H, N, ID]) CanPrecommit() bool {
	return hv.Precommit() == nil
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

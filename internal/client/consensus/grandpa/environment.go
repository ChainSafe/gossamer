// Copyright 2023 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package grandpa

import (
	"fmt"
	"sync"

	"github.com/ChainSafe/gossamer/internal/client/consensus"
	"github.com/ChainSafe/gossamer/internal/client/consensus/grandpa/communication"
	"github.com/ChainSafe/gossamer/internal/client/telemetry"
	pgrandpa "github.com/ChainSafe/gossamer/internal/primitives/consensus/grandpa"
	"github.com/ChainSafe/gossamer/internal/primitives/runtime"
	statemachine "github.com/ChainSafe/gossamer/internal/primitives/state-machine"
	grandpa "github.com/ChainSafe/gossamer/pkg/finality-grandpa"
	"github.com/ChainSafe/gossamer/pkg/scale"
	"golang.org/x/exp/constraints"
	"golang.org/x/exp/slices"
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
	Votes []pgrandpa.SignedMessage[H, N]
}

// numLastCompletedRounds NOTE: the current strategy for persisting completed rounds is very naive
// (update everything) and we also rely on cloning to do atomic updates,
// therefore this value should be kept small for now.
const numLastCompletedRounds = 2

// completedRounds Data about last completed rounds within a single voter set. Stores
// numLastCompletedRounds and always contains data about at least one round
// (genesis).
type completedRounds[H comparable, N constraints.Unsigned] struct {
	Rounds []completedRound[H, N]
	SetId  uint64
	Voters []pgrandpa.AuthorityID
}

// NewCompletedRounds Create a new completed rounds tracker with NUM_LAST_COMPLETED_ROUNDS capacity.
func NewCompletedRounds[H comparable, N constraints.Unsigned](
	genesis completedRound[H, N],
	setId uint64,
	voters AuthoritySet[H, N]) completedRounds[H, N] {
	rounds := make([]completedRound[H, N], 0, numLastCompletedRounds)
	rounds = append(rounds, genesis)

	var voterIDs []pgrandpa.AuthorityID
	currentAuthorities := voters.CurrentAuthorities
	for _, auth := range currentAuthorities {
		voterIDs = append(voterIDs, auth.AuthorityID)
	}

	return completedRounds[H, N]{
		rounds,
		setId,
		voterIDs,
	}
}

func (cr *completedRounds[H, N]) iter() []completedRound[H, N] {
	var reversed []completedRound[H, N]
	for i := len(cr.Rounds) - 1; i >= 0; i-- {
		reversed = append(reversed, cr.Rounds[i])
	}
	return reversed
}

// last Returns the last (latest) completed round
func (cr *completedRounds[H, N]) last() completedRound[H, N] {
	if len(cr.Rounds) == 0 {
		panic("inner is never empty; always contains at least genesis; qed")
	}
	return cr.Rounds[0]
}

// push a new completed round, oldest round is evicted if number of rounds
// is higher than `NUM_LAST_COMPLETED_ROUNDS`.
func (cr *completedRounds[H, N]) push(compRound completedRound[H, N]) {
	idx, found := slices.BinarySearchFunc(
		cr.Rounds,
		N(compRound.Number),
		func(a completedRound[H, N], b N) int {
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
type CurrentRounds[H comparable, N constraints.Unsigned] map[uint64]hasVoted[H, N]

// A tracker for the rounds that we are actively participating on (i.e. voting)
// and the authority id under which we are doing it.
type votingTracker struct {
	sync.Mutex
	Inner map[uint64]pgrandpa.AuthorityID
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
func NewSharedVoterSetState[H comparable, N constraints.Unsigned](
	state voterSetState[H, N]) SharedVoterSetState[H, N] {
	return SharedVoterSetState[H, N]{
		Inner: sharedVoterSetState[H, N]{
			Inner: state,
		},
	}
}

// Read the inner voter set state
func (svss *SharedVoterSetState[H, N]) read() voterSetState[H, N] { //nolint
	svss.Inner.Lock()
	defer svss.Inner.Unlock()
	return svss.Inner.Inner
}

// Get the authority id that we are using to vote on the given round, if any
func (svss *SharedVoterSetState[H, N]) votingOn(round uint64) *pgrandpa.AuthorityID { //nolint
	svss.Voting.Lock()
	defer svss.Voting.Unlock()
	key, ok := svss.Voting.Inner[round]
	if !ok {
		return nil
	}
	return &key
}

// Note that we started voting on the give round with the given authority id
func (svss *SharedVoterSetState[H, N]) startedVotingOn(round uint64, localID pgrandpa.AuthorityID) { //nolint
	svss.Voting.Lock()
	defer svss.Voting.Unlock()
	svss.Voting.Inner[round] = localID
}

// Note that we have finished voting on the given round. If we were voting on
// the given round, the authority id that we were using to do it will be
// cleared.
func (svss *SharedVoterSetState[H, N]) finishedVotingOn(round uint64) { //nolint
	svss.Voting.Lock()
	defer svss.Voting.Unlock()
	delete(svss.Voting.Inner, round)
}

// Return vote status information for the current round
func (svss *SharedVoterSetState[H, N]) hasVoted(round uint64) (hasVoted[H, N], error) {
	svss.Inner.Lock()
	defer svss.Inner.Unlock()

	hasNotVotedFunc := func(newHasVoted hasVoted[H, N]) (hasVoted[H, N], error) {
		err := newHasVoted.SetValue(no{})
		if err != nil {
			return newHasVoted, err
		}

		return newHasVoted, nil
	}

	var newHasVoted hasVoted[H, N]

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
type voterSetState[H comparable, N constraints.Unsigned] struct {
	inner any
}

type voterSetStateValues[H comparable, N constraints.Unsigned] interface {
	voterSetStateLive[H, N] | voterSetStatePaused[H, N]
}

func setVoterSetState[H comparable, N constraints.Unsigned, Value voterSetStateValues[H, N]](mvdt *voterSetState[H, N], value Value) {
	mvdt.inner = value
}

func (mvdt *voterSetState[H, N]) SetValue(value any) (err error) {
	switch value := value.(type) {
	case voterSetStateLive[H, N]:
		setVoterSetState[H, N](mvdt, value)
		return
	case voterSetStatePaused[H, N]:
		setVoterSetState[H, N](mvdt, value)
		return
	default:
		return fmt.Errorf("unsupported type")
	}
}

func (mvdt voterSetState[H, N]) IndexValue() (index uint, value any, err error) {
	switch mvdt.inner.(type) {
	case voterSetStateLive[H, N]:
		return 0, mvdt.inner, nil
	case voterSetStatePaused[H, N]:
		return 1, mvdt.inner, nil
	}
	return 0, nil, scale.ErrUnsupportedVaryingDataTypeValue
}

func (mvdt voterSetState[H, N]) Value() (value any, err error) {
	_, value, err = mvdt.IndexValue()
	return
}

func (mvdt voterSetState[H, N]) ValueAt(index uint) (value any, err error) {
	switch index {
	case 0:
		return *new(voterSetStateLive[H, N]), nil
	case 1:
		return *new(voterSetStatePaused[H, N]), nil
	}
	return nil, scale.ErrUnknownVaryingDataTypeValue
}

// NewVoterSetState is constructor for voterSetState
func NewVoterSetState[
	H comparable,
	N constraints.Unsigned,
]() *voterSetState[H, N] {
	return &voterSetState[H, N]{}
}

// NewLiveVoterSetState Create a new live voterSetState with round 0 as a completed round using
// the given genesis state and the given authorities. Round 1 is added as a
// current round (with state `hasVoted::no`).
func NewLiveVoterSetState[H comparable, N constraints.Unsigned](
	setId uint64,
	authSet AuthoritySet[H, N],
	genesisState grandpa.HashNumber[H, N]) (voterSetState[H, N], error) {
	state := grandpa.NewRoundState[H, N](genesisState)
	completedRounds := NewCompletedRounds[H, N](
		completedRound[H, N]{
			State: state,
			Base:  genesisState,
		},
		setId,
		authSet,
	)
	//currentRounds := make(map[uint64]hasVoted[string, uint])
	currentRounds := CurrentRounds[H, N](
		make(map[uint64]hasVoted[H, N]),
	)
	var hasVoted hasVoted[H, N]
	err := hasVoted.SetValue(no{})
	if err != nil {
		return voterSetState[H, N]{}, err
	}
	currentRounds[1] = hasVoted

	liveState := voterSetStateLive[H, N]{
		CompletedRounds: completedRounds,
		CurrentRounds:   currentRounds,
	}

	newVoterSetState := *NewVoterSetState[H, N]()
	err = newVoterSetState.SetValue(liveState)

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
func (tve *voterSetState[H, N]) withCurrentRound(
	round uint64) (completedRounds[H, N], CurrentRounds[H, N], error) {
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
		return completedRounds[H, N]{},
			CurrentRounds[H, N]{},
			fmt.Errorf("voter acting on a live round we are not tracking")
	case voterSetStatePaused[H, N]:
		return completedRounds[H, N]{},
			CurrentRounds[H, N]{},
			fmt.Errorf("voter acting while in paused state")
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

// voterSetStatePaused The voter is paused, i.e. not casting or importing any votes.
type voterSetStatePaused[H comparable, N constraints.Unsigned] struct {
	// The previously completed rounds
	CompletedRounds completedRounds[H, N]
}

// hasVoted Whether we've voted already during a prior run of the program
type hasVoted[H comparable, N constraints.Unsigned] struct {
	inner any
}

type hasVotedValues[H comparable, N constraints.Unsigned] interface {
	no | yes[H, N]
}

func setHasVoted[H comparable, N constraints.Unsigned, Value hasVotedValues[H, N]](mvdt *hasVoted[H, N], value Value) {
	mvdt.inner = value
}

func (mvdt *hasVoted[H, N]) SetValue(value any) (err error) {
	switch value := value.(type) {
	case no:
		setHasVoted[H, N](mvdt, value)
		return
	case yes[H, N]:
		setHasVoted[H, N](mvdt, value)
		return
	default:
		return fmt.Errorf("unsupported type")
	}
}

func (mvdt hasVoted[H, N]) IndexValue() (index uint, value any, err error) {
	switch mvdt.inner.(type) {
	case no:
		return 0, mvdt.inner, nil
	case yes[H, N]:
		return 1, mvdt.inner, nil
	}
	return 0, nil, scale.ErrUnsupportedVaryingDataTypeValue
}

func (mvdt hasVoted[H, N]) Value() (value any, err error) {
	_, value, err = mvdt.IndexValue()
	return
}

func (mvdt hasVoted[H, N]) ValueAt(index uint) (value any, err error) {
	switch index {
	case 0:
		return *new(no), nil
	case 1:
		return *new(yes[H, N]), nil
	}
	return nil, scale.ErrUnknownVaryingDataTypeValue
}

// no Has not voted already in this round
type no struct{}

type yes[H comparable, N constraints.Unsigned] struct {
	AuthId pgrandpa.AuthorityID
	Vote   vote[H, N]
}

// propose Returns the proposal we should vote with (if any.)
func (hv *hasVoted[H, N]) Propose() *grandpa.PrimaryPropose[H, N] {
	value, err := hv.Value()
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
func (hv *hasVoted[H, N]) Prevote() *grandpa.Prevote[H, N] {
	value, err := hv.Value()
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
func (hv *hasVoted[H, N]) Precommit() *grandpa.Precommit[H, N] {
	value, err := hv.Value()
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
func (hv *hasVoted[H, N]) CanPropose() bool {
	return hv.Propose() == nil
}

// CanPrevote Returns true if the voter can still prevote, false otherwise
func (hv *hasVoted[H, N]) CanPrevote() bool {
	return hv.Prevote() == nil
}

// CanPrecommit Returns true if the voter can still precommit, false otherwise
func (hv *hasVoted[H, N]) CanPrecommit() bool {
	return hv.Precommit() == nil
}

// vote Whether we've voted already during a prior run of the program
// type vote[H comparable, N constraints.Unsigned] scale.VaryingDataType
type vote[H comparable, N constraints.Unsigned] struct {
	inner any
}

type voteValues[H comparable, N constraints.Unsigned] interface {
	propose[H, N] | prevote[H, N] | precommit[H, N]
}

func setVote[H comparable, N constraints.Unsigned, Value voteValues[H, N]](mvdt *vote[H, N], value Value) {
	mvdt.inner = value
}

func (mvdt *vote[H, N]) SetValue(value any) (err error) {
	switch value := value.(type) {
	case propose[H, N]:
		setVote[H, N](mvdt, value)
		return
	case prevote[H, N]:
		setVote[H, N](mvdt, value)
		return
	case precommit[H, N]:
		setVote[H, N](mvdt, value)
		return
	default:
		return fmt.Errorf("unsupported type")
	}
}

func (mvdt vote[H, N]) IndexValue() (index uint, value any, err error) {
	switch mvdt.inner.(type) {
	case propose[H, N]:
		return 0, mvdt.inner, nil
	case prevote[H, N]:
		return 1, mvdt.inner, nil
	case precommit[H, N]:
		return 2, mvdt.inner, nil
	}
	return 0, nil, scale.ErrUnsupportedVaryingDataTypeValue
}

func (mvdt vote[H, N]) Value() (value any, err error) {
	_, value, err = mvdt.IndexValue()
	return
}

func (mvdt vote[H, N]) ValueAt(index uint) (value any, err error) {
	switch index {
	case 0:
		return *new(propose[H, N]), nil
	case 1:
		return *new(prevote[H, N]), nil
	case 2:
		return *new(precommit[H, N]), nil
	}
	return nil, scale.ErrUnknownVaryingDataTypeValue
}

// propose Has cast a proposal
type propose[H comparable, N constraints.Unsigned] struct {
	PrimaryPropose grandpa.PrimaryPropose[H, N]
}

// prevote Has cast a prevote
type prevote[H comparable, N constraints.Unsigned] struct {
	PrimaryPropose *grandpa.PrimaryPropose[H, N]
	Vote           grandpa.Prevote[H, N]
}

// precommit Has cast a precommit (implies prevote.)
type precommit[H comparable, N constraints.Unsigned] struct {
	PrimaryPropose *grandpa.PrimaryPropose[H, N]
	Vote           grandpa.Prevote[H, N]
	Commit         grandpa.Precommit[H, N]
}

// / Prometheus metrics for GRANDPA.
// pub(crate) struct Metrics {
type metrics struct {
	// 	finality_grandpa_round: Gauge<U64>,
	// 	finality_grandpa_prevotes: Counter<U64>,
	// 	finality_grandpa_precommits: Counter<U64>,
}

// / The environment we run GRANDPA in.
type environment[N runtime.Number, H runtime.Hash, T statemachine.Transaction] struct {
	Client              ClientForGrandpa[N, H, T]
	SelectChain         consensus.SelectChain[H, N]
	Voters              grandpa.VoterSet[string]
	Config              Config
	AuthoritySet        SharedAuthoritySet[H, N]
	Network             communication.NetworkBridge[H, N]
	SetID               SetID
	VoterSetState       SharedVoterSetState[H, N]
	VotingRule          VotingRule[H, N]
	Metrics             *metrics
	JustificationSender *GrandpaJustificationSender[H, N]
	Telemetry           *telemetry.TelemetryHandle
}

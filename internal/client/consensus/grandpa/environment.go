// Copyright 2023 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package grandpa

import (
	"fmt"
	"io"
	"sync"

	primitives "github.com/ChainSafe/gossamer/internal/primitives/consensus/grandpa"
	"github.com/ChainSafe/gossamer/internal/primitives/runtime"
	grandpa "github.com/ChainSafe/gossamer/pkg/finality-grandpa"
	"github.com/ChainSafe/gossamer/pkg/scale"
	"golang.org/x/exp/slices"
)

// completedRound Data about a completed round. The set of votes that is stored must be minimal, i.e. at most one
// equivocation is stored per voter.
type completedRound[H runtime.Hash, N runtime.Number] struct {
	// The round number
	Number primitives.RoundNumber
	// The round state (prevote ghost, estimate, finalized, etc.)
	State grandpa.RoundState[H, N]
	// The target block base used for voting in the round
	Base grandpa.HashNumber[H, N]
	// All the votes observed in the round
	Votes []primitives.SignedMessage[H, N]
}

// NOTE: the current strategy for persisting completed rounds is very naive
// (update everything) and we also rely on cloning to do atomic updates,
// therefore this value should be kept small for now.
const numLastCompletedRounds = 2

// Data about last completed rounds within a single voter set. Stores numLastCompletedRounds and always contains data
// about at least one round (genesis).
type completedRounds[H runtime.Hash, N runtime.Number] struct {
	Rounds []completedRound[H, N]
	SetId  primitives.SetID
	Voters []primitives.AuthorityID
}

// creates a new completed rounds tracker with numLastCompletedRounds capacity.
func newCompletedRounds[H runtime.Hash, N runtime.Number](
	genesis completedRound[H, N],
	setId primitives.SetID,
	voters AuthoritySet[H, N],
) completedRounds[H, N] {
	rounds := make([]completedRound[H, N], 0, numLastCompletedRounds)
	rounds = append(rounds, genesis)

	var voterIDs []primitives.AuthorityID
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

// Get the set-id and voter set of the completed rounds.
func (cr *completedRounds[H, N]) setInfo() (primitives.SetID, []primitives.AuthorityID) {
	return cr.SetId, cr.Voters
}

// Iterate over all completed rounds.
func (cr *completedRounds[H, N]) iter() []completedRound[H, N] {
	var reversed []completedRound[H, N]
	for i := len(cr.Rounds) - 1; i >= 0; i-- {
		reversed = append(reversed, cr.Rounds[i])
	}
	return reversed
}

// Returns the last (latest) completed round
func (cr *completedRounds[H, N]) last() completedRound[H, N] {
	if len(cr.Rounds) == 0 {
		panic("inner is never empty; always contains at least genesis; qed")
	}
	return cr.Rounds[0]
}

// Push a new completed round, oldest round is evicted if number of rounds is higher than numLastCompletedRounds.
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

// CurrentRounds is a map with voter status information for currently live rounds, which votes have we cast and what
// type of votes they are.
// TODO: this is a BtreeMap in rust. Convert to btree after #3480 is implemented
type currentRounds[H runtime.Hash, N runtime.Number] map[uint64]hasVoted[H, N]

func (cr currentRounds[H, N]) MarshalSCALE() ([]byte, error) {
	type helper map[uint64]hasVotedVDT[H, N]
	mapped := make(helper)
	for key, val := range cr {
		if val == nil {
			panic("should not be nil")
		}
		hv := hasVotedVDT[H, N]{}
		err := hv.SetValue(val)
		if err != nil {
			panic(fmt.Errorf("SetValue should not fail: %s", err))
		}
		mapped[key] = hv
	}
	return scale.Marshal(mapped)
}

func (cr *currentRounds[H, N]) UnmarshalSCALE(reader io.Reader) error {
	type helper map[uint64]hasVotedVDT[H, N]
	mapped := make(helper)

	decoder := scale.NewDecoder(reader)
	err := decoder.Decode(&mapped)
	if err != nil {
		return err
	}

	for key, value := range mapped {
		if value.inner == nil {
			panic("inner should not be nil")
		}
		if (*cr) == nil {
			*cr = make(map[uint64]hasVoted[H, N])
		}
		(*cr)[key] = value.inner
	}
	return nil
}

// SharedVoterSetState is a voter set state meant to be shared safely across multiple threads.
type SharedVoterSetState[H runtime.Hash, N runtime.Number] struct {
	// The inner shared `voterSetState`.
	innerMtx sync.RWMutex
	inner    voterSetState[H, N]
	// A tracker for the rounds that we are actively participating on (i.e. voting) and the authority id under which we
	// are doing it.
	votingMtx sync.RWMutex
	voting    map[primitives.RoundNumber]primitives.AuthorityID
}

// NewSharedVoterSetState Create a new shared voter set tracker with the given state.
func NewSharedVoterSetState[H runtime.Hash, N runtime.Number](state voterSetState[H, N]) *SharedVoterSetState[H, N] {
	return &SharedVoterSetState[H, N]{
		inner:  state,
		voting: make(map[primitives.RoundNumber]primitives.AuthorityID),
	}
}

// Get the authority id that we are using to vote on the given round, if any.
func (svss *SharedVoterSetState[H, N]) votingOn(round primitives.RoundNumber) *primitives.AuthorityID { //nolint: unused
	svss.votingMtx.RLock()
	defer svss.votingMtx.RUnlock()
	key, ok := svss.voting[round]
	if !ok {
		return nil
	}
	return &key
}

// Note that we started voting on the give round with the given authority id.
func (svss *SharedVoterSetState[H, N]) startedVotingOn( //nolint: unused
	round primitives.RoundNumber, localID primitives.AuthorityID,
) {
	svss.votingMtx.Lock()
	defer svss.votingMtx.Unlock()
	svss.voting[round] = localID
}

// Note that we have finished voting on the given round. If we were voting on the given round, the authority id that
// we were using to do it will be cleared.
func (svss *SharedVoterSetState[H, N]) finishedVotingOn(round primitives.RoundNumber) { //nolint: unused
	svss.votingMtx.Lock()
	defer svss.votingMtx.Unlock()
	delete(svss.voting, round)
}

// Return vote status information for the current round
func (svss *SharedVoterSetState[H, N]) hasVoted(round primitives.RoundNumber) hasVoted[H, N] {
	svss.votingMtx.RLock()
	defer svss.votingMtx.RUnlock()

	switch vss := svss.inner.(type) {
	case voterSetStateLive[H, N]:
		hasVoted, ok := vss.CurrentRounds[uint64(round)]
		if ok {
			switch hasVoted.(type) {
			case hasVotedYes[H, N]:
				return hasVoted
			case hasVotedNo[H, N]:
				return hasVoted
			default:
				panic("unreachable")
			}
		}
		return hasVotedNo[H, N]{}
	case voterSetStatePaused[H, N]:
		return hasVotedNo[H, N]{}
	default:
		panic("unreachable")
	}
}

// voterSetState The state of the current voter set, whether it is currently active or not and information related to
// the previously completed rounds. Current round voting status is used when restarting the voter, i.e. it will re-use
// the previous votes for a given round if appropriate (same round and same local key).
type voterSetState[H runtime.Hash, N runtime.Number] interface {
	completedRounds() completedRounds[H, N]
	lastCompletedRound() completedRound[H, N]
	withCurrentRound(round uint64) (completedRounds[H, N], currentRounds[H, N], error)
}
type voterSetStateVDT[H runtime.Hash, N runtime.Number] struct {
	inner voterSetState[H, N]
}

type voterSetStateValues[H runtime.Hash, N runtime.Number] interface {
	voterSetStateLive[H, N] | voterSetStatePaused[H, N]
	voterSetState[H, N]
}

func setVoterSetState[
	H runtime.Hash, N runtime.Number, Value voterSetStateValues[H, N],
](mvdt *voterSetStateVDT[H, N], value Value) {
	mvdt.inner = value
}

func (mvdt *voterSetStateVDT[H, N]) SetValue(value any) (err error) {
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

func (mvdt voterSetStateVDT[H, N]) IndexValue() (index uint, value any, err error) {
	switch mvdt.inner.(type) {
	case voterSetStateLive[H, N]:
		return 0, mvdt.inner, nil
	case voterSetStatePaused[H, N]:
		return 1, mvdt.inner, nil
	}
	return 0, nil, scale.ErrUnsupportedVaryingDataTypeValue
}

func (mvdt voterSetStateVDT[H, N]) Value() (value any, err error) {
	_, value, err = mvdt.IndexValue()
	return
}

func (mvdt voterSetStateVDT[H, N]) ValueAt(index uint) (value any, err error) {
	switch index {
	case 0:
		return *new(voterSetStateLive[H, N]), nil
	case 1:
		return *new(voterSetStatePaused[H, N]), nil
	}
	return nil, scale.ErrUnsupportedVaryingDataTypeValue
}

// newVoterSetState is constructor for voterSetState
func newVoterSetStateVDT[H runtime.Hash, N runtime.Number]() *voterSetStateVDT[H, N] {
	return &voterSetStateVDT[H, N]{}
}

// newVoterSetStateLive Create a new live voterSetState with round 0 as a completed round using the given genesis state
// and the given authorities. Round 1 is added as a current round (with state `hasVotedNo`).
func newVoterSetStateLive[H runtime.Hash, N runtime.Number](
	setId primitives.SetID,
	authSet AuthoritySet[H, N],
	genesisState grandpa.HashNumber[H, N],
) voterSetStateLive[H, N] {
	state := grandpa.NewRoundState[H, N](genesisState)
	completedRounds := newCompletedRounds[H, N](
		completedRound[H, N]{
			State: state,
			Base:  genesisState,
		},
		setId,
		authSet,
	)
	currentRounds := make(currentRounds[H, N])
	currentRounds[1] = hasVotedNo[H, N]{}

	liveState := voterSetStateLive[H, N]{
		CompletedRounds: completedRounds,
		CurrentRounds:   currentRounds,
	}
	return liveState
}

// voterSetStateLive The voter is live, i.e. participating in rounds.
type voterSetStateLive[H runtime.Hash, N runtime.Number] struct {
	// The previously completed rounds
	CompletedRounds completedRounds[H, N]
	// Voter status for the currently live rounds.
	CurrentRounds currentRounds[H, N]
}

func (vssl voterSetStateLive[H, N]) completedRounds() completedRounds[H, N] {
	return vssl.CompletedRounds
}
func (vssl voterSetStateLive[H, N]) lastCompletedRound() completedRound[H, N] { //nolint: unused
	return vssl.CompletedRounds.last()
}
func (vssl voterSetStateLive[H, N]) withCurrentRound( //nolint: unused
	round uint64,
) (completedRounds[H, N], currentRounds[H, N], error) {
	_, contains := vssl.CurrentRounds[round]
	if contains {
		return vssl.CompletedRounds, vssl.CurrentRounds, nil
	}
	return completedRounds[H, N]{},
		currentRounds[H, N]{},
		fmt.Errorf("voter acting on a live round we are not tracking")
}

// voterSetStatePaused means the voter is paused, i.e. not casting or importing any votes.
type voterSetStatePaused[H runtime.Hash, N runtime.Number] struct {
	// The previously completed rounds
	CompletedRounds completedRounds[H, N]
}

func (vssp voterSetStatePaused[H, N]) completedRounds() completedRounds[H, N] { //nolint: unused
	return vssp.CompletedRounds
}
func (vssp voterSetStatePaused[H, N]) lastCompletedRound() completedRound[H, N] {
	return vssp.CompletedRounds.last()
}
func (vssl voterSetStatePaused[H, N]) withCurrentRound( //nolint: unused
	round uint64,
) (completedRounds[H, N], currentRounds[H, N], error) {
	return completedRounds[H, N]{},
		currentRounds[H, N]{},
		fmt.Errorf("voter acting while in paused state")
}

// hasVoted Whether we've voted already during a prior run of the program
type hasVoted[H runtime.Hash, N runtime.Number] interface {
	// Returns the proposal we should vote with (if any.)
	Propose() *grandpa.PrimaryPropose[H, N]
	// Returns the prevote we should vote with (if any.)
	Prevote() *grandpa.Prevote[H, N]
	// Returns the precommit we should vote with (if any.)
	Precommit() *grandpa.Precommit[H, N]
	// Returns true if the voter can still propose, false otherwise.
	CanPropose() bool
	// Returns true if the voter can still prevote, false otherwise.
	CanPrevote() bool
	// Returns true if the voter can still precommit, false otherwise.
	CanPrecommit() bool
}
type hasVotedVDT[H runtime.Hash, N runtime.Number] struct {
	inner hasVoted[H, N]
}

type hasVotedValues[H runtime.Hash, N runtime.Number] interface {
	hasVotedNo[H, N] | hasVotedYes[H, N]
	hasVoted[H, N]
}

func setHasVoted[H runtime.Hash, N runtime.Number, Value hasVotedValues[H, N]](mvdt *hasVotedVDT[H, N], value Value) {
	mvdt.inner = value
}

func (mvdt *hasVotedVDT[H, N]) SetValue(value any) (err error) {
	switch value := value.(type) {
	case hasVotedNo[H, N]:
		setHasVoted[H, N](mvdt, value)
		return
	case hasVotedYes[H, N]:
		setHasVoted[H, N](mvdt, value)
		return
	default:
		return fmt.Errorf("unsupported type")
	}
}

func (mvdt hasVotedVDT[H, N]) IndexValue() (index uint, value any, err error) {
	switch mvdt.inner.(type) {
	case hasVotedNo[H, N]:
		return 0, mvdt.inner, nil
	case hasVotedYes[H, N]:
		return 1, mvdt.inner, nil
	}
	return 0, nil, scale.ErrUnsupportedVaryingDataTypeValue
}

func (mvdt hasVotedVDT[H, N]) Value() (value any, err error) {
	_, value, err = mvdt.IndexValue()
	return
}

func (mvdt hasVotedVDT[H, N]) ValueAt(index uint) (value any, err error) {
	switch index {
	case 0:
		return *new(hasVotedNo[H, N]), nil
	case 1:
		return *new(hasVotedYes[H, N]), nil
	}
	return nil, scale.ErrUnsupportedVaryingDataTypeValue
}

// hasVotedNo has not voted already in this round
type hasVotedNo[H runtime.Hash, N runtime.Number] struct{}

func (hasVotedNo[H, N]) Propose() *grandpa.PrimaryPropose[H, N] {
	return nil
}
func (hasVotedNo[H, N]) Prevote() *grandpa.Prevote[H, N] {
	return nil
}
func (hasVotedNo[H, N]) Precommit() *grandpa.Precommit[H, N] {
	return nil
}
func (hasVotedNo[H, N]) CanPropose() bool {
	return true
}
func (hasVotedNo[H, N]) CanPrevote() bool {
	return true
}
func (hasVotedNo[H, N]) CanPrecommit() bool {
	return true
}

type hasVotedYes[H runtime.Hash, N runtime.Number] struct {
	AuthorityID primitives.AuthorityID
	Vote        vote[H, N]
}

func (hvy hasVotedYes[H, N]) Propose() *grandpa.PrimaryPropose[H, N] {
	switch vote := hvy.Vote.(type) {
	case votePropose[H, N]:
		return &vote.PrimaryPropose
	case votePrevote[H, N]:
		return vote.PrimaryPropose
	case votePrecommit[H, N]:
		return vote.PrimaryPropose
	default:
		return nil
	}
}
func (hvy hasVotedYes[H, N]) Prevote() *grandpa.Prevote[H, N] {
	switch vote := hvy.Vote.(type) {
	case votePrevote[H, N]:
		return &vote.Prevote
	case votePrecommit[H, N]:
		return &vote.Prevote
	default:
		return nil
	}
}
func (hvy hasVotedYes[H, N]) Precommit() *grandpa.Precommit[H, N] {
	switch vote := hvy.Vote.(type) {
	case votePrecommit[H, N]:
		return &vote.Precommit
	default:
		return nil
	}
}
func (hvy hasVotedYes[H, N]) CanPropose() bool {
	return hvy.Propose() == nil
}
func (hvy hasVotedYes[H, N]) CanPrevote() bool {
	return hvy.Prevote() == nil
}
func (hvy hasVotedYes[H, N]) CanPrecommit() bool {
	return hvy.Precommit() == nil
}
func (hvy hasVotedYes[H, N]) MarshalSCALE() ([]byte, error) {
	type helper struct {
		AuthorityID primitives.AuthorityID
		Vote        voteVDT[H, N] //use vote vdt type since this needs to be encodable/decodable
	}
	voteVDT := voteVDT[H, N]{}
	err := voteVDT.SetValue(hvy.Vote)
	if err != nil {
		return nil, err
	}
	h := helper{
		AuthorityID: hvy.AuthorityID,
		Vote:        voteVDT,
	}
	return scale.Marshal(h)
}
func (hvy *hasVotedYes[H, N]) UnmarshalSCALE(reader io.Reader) error {
	type helper struct {
		AuthorityID primitives.AuthorityID
		Vote        voteVDT[H, N] //use vote vdt type since this needs to be encodable/decodable
	}
	h := helper{}

	decoder := scale.NewDecoder(reader)
	err := decoder.Decode(&h)
	if err != nil {
		return err
	}

	hvy.AuthorityID = h.AuthorityID
	_, err = h.Vote.Value()
	if err != nil {
		return err
	}
	hvy.Vote = h.Vote.inner
	return nil
}

// vote is whether we've voted already during a prior run of the program
type vote[H runtime.Hash, N runtime.Number] interface {
	isVote()
}
type voteVDT[H runtime.Hash, N runtime.Number] struct {
	inner vote[H, N]
}
type voteValues[H runtime.Hash, N runtime.Number] interface {
	votePropose[H, N] | votePrevote[H, N] | votePrecommit[H, N]
	vote[H, N]
}

func setVote[H runtime.Hash, N runtime.Number, Value voteValues[H, N]](mvdt *voteVDT[H, N], value Value) {
	mvdt.inner = value
}
func (mvdt *voteVDT[H, N]) SetValue(value any) (err error) {
	switch value := value.(type) {
	case votePropose[H, N]:
		setVote[H, N](mvdt, value)
		return
	case votePrevote[H, N]:
		setVote[H, N](mvdt, value)
		return
	case votePrecommit[H, N]:
		setVote[H, N](mvdt, value)
		return
	default:
		return fmt.Errorf("unsupported type")
	}
}
func (mvdt voteVDT[H, N]) IndexValue() (index uint, value any, err error) {
	switch mvdt.inner.(type) {
	case votePropose[H, N]:
		return 0, mvdt.inner, nil
	case votePrevote[H, N]:
		return 1, mvdt.inner, nil
	case votePrecommit[H, N]:
		return 2, mvdt.inner, nil
	}
	return 0, nil, scale.ErrUnsupportedVaryingDataTypeValue
}
func (mvdt voteVDT[H, N]) Value() (value any, err error) {
	_, value, err = mvdt.IndexValue()
	return
}
func (mvdt voteVDT[H, N]) ValueAt(index uint) (value any, err error) {
	switch index {
	case 0:
		return *new(votePropose[H, N]), nil
	case 1:
		return *new(votePrevote[H, N]), nil
	case 2:
		return *new(votePrecommit[H, N]), nil
	}
	return nil, scale.ErrUnsupportedVaryingDataTypeValue
}

// propose Has cast a proposal
type votePropose[H runtime.Hash, N runtime.Number] struct {
	PrimaryPropose grandpa.PrimaryPropose[H, N]
}

func (votePropose[H, N]) isVote() {}

// prevote Has cast a prevote
type votePrevote[H runtime.Hash, N runtime.Number] struct {
	PrimaryPropose *grandpa.PrimaryPropose[H, N]
	Prevote        grandpa.Prevote[H, N]
}

func (votePrevote[H, N]) isVote() {}

// precommit Has cast a precommit (implies prevote.)
type votePrecommit[H runtime.Hash, N runtime.Number] struct {
	PrimaryPropose *grandpa.PrimaryPropose[H, N]
	Prevote        grandpa.Prevote[H, N]
	Precommit      grandpa.Precommit[H, N]
}

func (votePrecommit[H, N]) isVote() {}

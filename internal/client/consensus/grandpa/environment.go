// Copyright 2023 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package grandpa

import (
	"fmt"
	"io"
	"sync"
	"time"

	rand "math/rand/v2"

	"github.com/ChainSafe/gossamer/internal/client/api"
	"github.com/ChainSafe/gossamer/internal/client/api/utils"
	"github.com/ChainSafe/gossamer/internal/client/consensus"
	"github.com/ChainSafe/gossamer/internal/primitives/blockchain"
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
type currentRounds[H runtime.Hash, N runtime.Number] map[primitives.RoundNumber]hasVoted[H, N]

func (cr currentRounds[H, N]) MarshalSCALE() ([]byte, error) {
	type helper map[primitives.RoundNumber]hasVotedVDT[H, N]
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
	type helper map[primitives.RoundNumber]hasVotedVDT[H, N]
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
			*cr = make(map[primitives.RoundNumber]hasVoted[H, N])
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
		hasVoted, ok := vss.CurrentRounds[round]
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
	withCurrentRound(round primitives.RoundNumber) (completedRounds[H, N], currentRounds[H, N], error)
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
	round primitives.RoundNumber,
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
	round primitives.RoundNumber,
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

// / The environment we run GRANDPA in.
type environment[
	H runtime.Hash,
	N runtime.Number,
	Hasher runtime.Hasher[H],
	Header runtime.Header[N, H],
	E runtime.Extrinsic,
] struct {
	Client        ClientForGrandpa[H, N, Hasher, Header, E]
	SelectChain   consensus.SelectChain[H, N, Header]
	Voters        grandpa.VoterSet[primitives.AuthorityID]
	Config        Config
	AuthoritySet  SharedAuthoritySet[H, N]
	Network       networkBridge[H, N, Hasher]
	SetID         SetID
	VoterSetState SharedVoterSetState[H, N]
	VotingRule    VotingRule[H, N, Header]
	// TODO: metrics
	JustificationSender *GrandpaJustificationSender[H, N, Header]
	// TODO: telemetry
}

// / Updates the voter set state using the given closure. The write lock is
// / held during evaluation of the closure and the environment's voter set
// / state is set to its result if successful.
func (e *environment[H, N, Hasher, Header, E]) updateVoterSetState(
	f func(voterSetState voterSetState[H, N]) (voterSetState[H, N], error),
) error {
	e.VoterSetState.innerMtx.Lock()
	defer e.VoterSetState.innerMtx.Unlock()

	newState, err := f(e.VoterSetState.inner)
	if err != nil {
		return err
	}
	if newState != nil {
		e.VoterSetState.inner = newState
	}

	// TODO: metrics
	// if let Some(metrics) = self.metrics.as_ref() {
	// 	if let VoterSetState::Live { completed_rounds, .. } = voter_set_state {
	// 		let highest = completed_rounds
	// 			.rounds
	// 			.iter()
	// 			.map(|round| round.number)
	// 			.max()
	// 			.expect("There is always one completed round (genesis); qed");

	// 		metrics.finality_grandpa_round.set(highest);
	// 	}
	// }
	return nil
}

// / Report the given equivocation to the GRANDPA runtime module. This method
// / generates a session membership proof of the offender and then submits an
// / extrinsic to report the equivocation. In particular, the session membership
// / proof must be generated at the block at which the given set was active which
// / isn't necessarily the best block if there are pending authority set changes.
func (e *environment[H, N, Hasher, Header, E]) reportEquivocation(
	equivocation primitives.Equivocation,
) error {
	localID := e.VoterSetState.votingOn(equivocation.Round())
	if localID != nil {
		if equivocation.Offender() == *localID {
			return fmt.Errorf("refraining from sending equivocation report for our own equivocation")
		}
	}

	isDescendentOf := utils.IsDescendantOf[H, N, Header, E](e.Client, nil)

	// TODO [#9158]: Use SelectChain::best_chain() to get a potentially
	// more accurate best block
	info := e.Client.Info()
	bestBlockHash := info.BestHash
	bestBlockNumber := info.BestNumber

	// 	let authority_set = self.authority_set.inner();
	e.AuthoritySet.mtx.Lock()
	defer e.AuthoritySet.mtx.Unlock()
	authoritySet := e.AuthoritySet.inner

	// block hash and number of the next pending authority set change in the
	// given best chain.
	nextChange, err := authoritySet.nextChange(bestBlockHash, isDescendentOf)
	if err != nil {
		return fmt.Errorf("%w: %s", ErrSafety, err)
	}

	// find the hash of the latest block in the current set
	var currentSetLatestHash H
	if nextChange != nil {
		if nextChange.Number == 0 {
			return fmt.Errorf("%w: authority set change signalled at genesis", ErrSafety)
		}
		// the next set starts at `n` so the current one lasts until `n - 1`. if
		// `n` is later than the best block, then the current set is still live
		// at best block.
		if nextChange.Number > bestBlockNumber {
			currentSetLatestHash = bestBlockHash
		} else {
			// this is the header at which the new set will start
			header, err := e.Client.Header(nextChange.Hash)
			if err != nil {
				return err
			}
			if header == nil {
				panic("got block hash from registered pending change; pending changes are only registered on block import; qed.")
			}
			// its parent block is the last block in the current set
			currentSetLatestHash = (*header).ParentHash()
		}
	} else {
		// there is no pending change, the latest block for the current set is
		// the best block.
		currentSetLatestHash = bestBlockHash
	}

	// generate key ownership proof at that block
	keyOwnerProof := e.Client.RuntimeAPI().GenerateKeyOwnershipProof(
		currentSetLatestHash,
		primitives.SetID(authoritySet.SetID),
		equivocation.Offender(),
	)
	if keyOwnerProof == nil {
		logger.Debugf("equivocation offender is not part of the authority set")
		return nil
	}

	// submit equivocation report at **best** block
	equivocationProof := primitives.NewEquivocationProof[H, N](primitives.SetID(authoritySet.SetID), equivocation)

	runtimeAPI := e.Client.RuntimeAPI()

	// 	runtime_api.register_extension(
	// 		self.offchain_tx_pool_factory.offchain_transaction_pool(best_block_hash),
	// 	);

	err = runtimeAPI.SubmitReportEquivocationUnsignedExtrinsic(
		bestBlockHash, equivocationProof, *keyOwnerProof,
	)
	if err != nil {
		return ErrRuntimeApi
	}

	return nil
}

func (e *environment[H, N, Hasher, Header, E]) Ancestry(
	base H,
	block H,
) ([]H, error) {
	return ancestry(e.Client, base, block)
}

func ancestry[H runtime.Hash, N runtime.Number](
	client blockchain.HeaderMetadata[H, N],
	base H,
	block H,
) ([]H, error) {
	if base == block {
		return []H{}, nil
	}

	// check if `base` is descendent of `block`
	if base == block {
		return nil, grandpa.ErrNotDescendent
	}

	treeRoute, err := blockchain.NewTreeRoute(client, block, base)
	if err != nil {
		logger.Debugf("encountered error computing ancestry between block %s and base %s: %s", block, base, err)
		return nil, grandpa.ErrNotDescendent
	}

	if treeRoute.CommonBlock().Hash != base {
		return nil, grandpa.ErrNotDescendent
	}

	// skip one because our ancestry is meant to start from the parent of `block`,
	// and `tree_route` includes it.
	retracted := treeRoute.Retracted()
	route := make([]H, len(retracted)-1)
	for i := 1; i < len(retracted); i++ {
		route[i-1] = retracted[i].Hash
	}
	return route, nil
}

func (e *environment[H, N, Hasher, Header, E]) IsEqualOrDescendantOf(base H, block H) bool {
	if base == block {
		return true
	}
	// TODO: currently this function always succeeds since the only error
	// variant is `Error::NotDescendent`, this may change in the future as
	// other errors (e.g. IO) are not being exposed.
	_, err := ancestry(e.Client, base, block)
	if err != nil {
		return false
	}
	return true
}

// impl<B, Block, C, N, S, SC, VR> voter::Environment<Block::Hash, NumberFor<Block>>
// 	for Environment<B, Block, C, N, S, SC, VR>
// where
// 	Block: BlockT,
// 	B: BackendT<Block>,
// 	C: ClientForGrandpa<Block, B> + 'static,
// 	C::Api: GrandpaApi<Block>,
// 	N: NetworkT<Block>,
// 	S: SyncingT<Block>,
// 	SC: SelectChainT<Block> + 'static,
// 	VR: VotingRuleT<Block, C> + Clone + 'static,
// 	NumberFor<Block>: BlockNumberOps,
// {
// 	type Timer = Pin<Box<dyn Future<Output = Result<(), Self::Error>> + Send>>;
// 	type BestChain = Pin<
// 		Box<
// 			dyn Future<Output = Result<Option<(Block::Hash, NumberFor<Block>)>, Self::Error>>
// 				+ Send,
// 		>,
// 	>;

// 	type Id = AuthorityId;
// 	type Signature = AuthoritySignature;

// 	// regular round message streams
// 	type In = Pin<
// 		Box<
// 			dyn Stream<
// 					Item = Result<
// 						::finality_grandpa::SignedMessage<
// 							Block::Hash,
// 							NumberFor<Block>,
// 							Self::Signature,
// 							Self::Id,
// 						>,
// 						Self::Error,
// 					>,
// 				> + Send,
// 		>,
// 	>;
// 	type Out = Pin<
// 		Box<
// 			dyn Sink<
// 					::finality_grandpa::Message<Block::Hash, NumberFor<Block>>,
// 					Error = Self::Error,
// 				> + Send,
// 		>,
// 	>;

// 	type Error = CommandOrError<Block::Hash, NumberFor<Block>>;

// 	fn best_chain_containing(&self, block: Block::Hash) -> Self::BestChain {
// 		let client = self.client.clone();
// 		let authority_set = self.authority_set.clone();
// 		let select_chain = self.select_chain.clone();
// 		let voting_rule = self.voting_rule.clone();
// 		let set_id = self.set_id;

// 		Box::pin(async move {
// 			// NOTE: when we finalize an authority set change through the sync protocol the voter is
// 			//       signaled asynchronously. therefore the voter could still vote in the next round
// 			//       before activating the new set. the `authority_set` is updated immediately thus
// 			//       we restrict the voter based on that.
// 			if set_id != authority_set.set_id() {
// 				return Ok(None)
// 			}

//			best_chain_containing(block, client, authority_set, select_chain, voting_rule)
//				.await
//				.map_err(|e| e.into())
//		})
//	}
func (e *environment[H, N, Hasher, Header, E]) BestChainContaining(
	block H,
) grandpa.BestChain[H, N] {
	ch := make(grandpa.BestChain[H, N], 1)
	// NOTE: when we finalize an authority set change through the sync protocol the voter is
	// signaled asynchronously. therefore the voter could still vote in the next round
	// before activating the new set. the `authority_set` is updated immediately thus
	// we restrict the voter based on that.
	if e.SetID != SetID(e.AuthoritySet.inner.SetID) {
		ch <- grandpa.BestChainOutput[H, N]{
			Value: nil,
			Error: nil,
		}
		close(ch)
		return ch
	}

	// best_chain_containing(block, client, authority_set, select_chain, voting_rule)
	//
	//	.await
	//	.map_err(|e| e.into())
	go func() {
		value, err := bestChainContaining(block, e.Client, &e.AuthoritySet, e.SelectChain, e.VotingRule)
		ch <- grandpa.BestChainOutput[H, N]{
			Value: value,
			Error: err,
		}
		close(ch)
	}()
	return ch
}

// fn round_data(
//
//	&self,
//	round: RoundNumber,
//
// ) -> voter::RoundData<Self::Id, Self::Timer, Self::In, Self::Out> {
func (e *environment[H, N, Hasher, Header, E]) RoundData(
	round uint64,
) grandpa.RoundData[H, N, primitives.AuthoritySignature, primitives.AuthorityID, grandpa.Message[H, N]] {
	// 		let prevote_timer = Delay::new(self.config.gossip_duration * 2);
	// 		let precommit_timer = Delay::new(self.config.gossip_duration * 4);
	prevoteTimer := time.NewTimer(e.Config.GossipDuration * 2)
	precommitTimer := time.NewTimer(e.Config.GossipDuration * 4)

	// 		let local_id = local_authority_id(&self.voters, self.config.keystore.as_ref());
	localID := localAuthorityID(e.Voters, &e.Config.KeyStore)

	// 		let has_voted = match self.voter_set_state.has_voted(round) {
	// 			HasVoted::Yes(id, vote) =>
	// 				if local_id.as_ref().map(|k| k == &id).unwrap_or(false) {
	// 					HasVoted::Yes(id, vote)
	// 				} else {
	// 					HasVoted::No
	// 				},
	// 			HasVoted::No => HasVoted::No,
	// 		};
	var hasVoted hasVoted[H, N]
	hv := e.VoterSetState.hasVoted(primitives.RoundNumber(round))
	switch hv := hv.(type) {
	case hasVotedYes[H, N]:
		if localID != nil && hv.AuthorityID == *localID {
			hasVoted = hv
		} else {
			hasVoted = hasVotedNo[H, N]{}
		}
	case hasVotedNo[H, N]:
		hasVoted = hv
	default:
		panic("unreachable")
	}

	// NOTE: we cache the local authority id that we'll be using to vote on the
	// given round. this is done to make sure we only check for available keys
	// from the keystore in this method when beginning the round, otherwise if
	// the keystore state changed during the round (e.g. a key was removed) it
	// could lead to internal state inconsistencies in the voter environment
	// (e.g. we wouldn't update the voter set state after prevoting since there's
	// no local authority id).
	// 		if let Some(id) = local_id.as_ref() {
	// 			self.voter_set_state.started_voting_on(round, id.clone());
	// 		}
	if localID != nil {
		e.VoterSetState.startedVotingOn(primitives.RoundNumber(round), *localID)
	}

	// we can only sign when we have a local key in the authority set
	// and we have a reference to the keystore.
	// 		let keystore = match (local_id.as_ref(), self.config.keystore.as_ref()) {
	// 			(Some(id), Some(keystore)) => Some((id.clone(), keystore.clone()).into()),
	// 			_ => None,
	// 		};
	var keystore *localIDKeystore
	if localID != nil && e.Config.KeyStore != nil {
		keystore = &localIDKeystore{
			AuthorityID: *localID,
			KeyStore:    e.Config.KeyStore,
		}
	}

	// 		let (incoming, outgoing) = self.network.round_communication(
	// 			keystore,
	// 			crate::communication::Round(round),
	// 			crate::communication::SetId(self.set_id),
	// 			self.voters.clone(),
	// 			has_voted,
	// 		);
	in, out := e.Network.roundCommunication(keystore, Round(round), e.SetID, &e.Voters, hasVoted)

	convertedIn := make(chan signedMessage[H, N])
	go func() {
		for signed := range in {
			convertedIn <- signedMessage[H, N]{signed}
		}
	}()

	// schedule incoming messages from the network to be held until
	// corresponding blocks are imported.
	// 		let incoming = Box::pin(
	// 			UntilVoteTargetImported::new(
	// 				self.client.import_notification_stream(),
	// 				self.network.clone(),
	// 				self.client.clone(),
	// 				incoming,
	// 				"round",
	// 				None,
	// 			)
	// 			.map_err(Into::into),
	// 		);
	incoming := newUntilVoteTargetImported(
		e.Client.RegisterImportNotificationStream(),
		&e.Network,
		e.Client,
		convertedIn,
		"round",
	)

	// NOTE: what should i do with the outgoing channel? need to figure out the OutgoingMessages integration.
	// schedule network message cleanup when sink drops.
	// 		let outgoing = Box::pin(outgoing.sink_err_into());

	// 		voter::RoundData {
	// 			voter_id: local_id,
	// 			prevote_timer: Box::pin(prevote_timer.map(Ok)),
	// 			precommit_timer: Box::pin(precommit_timer.map(Ok)),
	// 			incoming,
	// 			outgoing,
	// 		}
	convertedOut := make(chan grandpa.SignedMessageError[H, N, primitives.AuthoritySignature, primitives.AuthorityID])
	go func() {
		for signed := range incoming.Chan() {
			convertedOut <- grandpa.SignedMessageError[H, N, primitives.AuthoritySignature, primitives.AuthorityID]{
				SignedMessage: signed.SignedMessage.SignedMessage,
			}
		}
	}()
	return grandpa.NewRoundData(localID, *prevoteTimer, *precommitTimer, convertedOut, out.preSend)
}

// fn proposed(
//
//	&self,
//	round: RoundNumber,
//	propose: PrimaryPropose<Block::Header>,
//
// ) -> Result<(), Self::Error> {
func (e *environment[H, N, Hasher, Header, E]) Proposed(round uint64, propose grandpa.PrimaryPropose[H, N]) error {
	// 		let local_id = match self.voter_set_state.voting_on(round) {
	// 			Some(id) => id,
	// 			None => return Ok(()),
	// 		};
	localID := e.VoterSetState.votingOn(primitives.RoundNumber(round))
	if localID == nil {
		return nil
	}

	// 		self.update_voter_set_state(|voter_set_state| {
	err := e.updateVoterSetState(func(vss voterSetState[H, N]) (voterSetState[H, N], error) {
		// 			let (completed_rounds, current_rounds) = voter_set_state.with_current_round(round)?;
		// 			let current_round = current_rounds
		// 				.get(&round)
		// 				.expect("checked in with_current_round that key exists; qed.");
		completedRounds, currentRounds, err := vss.withCurrentRound(primitives.RoundNumber(round))
		if err != nil {
			return nil, err
		}
		currentRound, ok := currentRounds[primitives.RoundNumber(round)]
		if !ok {
			panic(fmt.Errorf("checked in with_current_round that key exists; qed."))
		}

		// 			if !current_round.can_propose() {
		// 				// we've already proposed in this round (in a previous run),
		// 				// ignore the given vote and don't update the voter set
		// 				// state
		// 				return Ok(None)
		// 			}
		if !currentRound.CanPropose() {
			// we've already proposed in this round (in a previous run),
			// ignore the given vote and don't update the voter set
			// state
			return nil, nil
		}

		// 			let mut current_rounds = current_rounds.clone();
		// 			let current_round = current_rounds
		// 				.get_mut(&round)
		// 				.expect("checked previously that key exists; qed.");

		// 			*current_round = HasVoted::Yes(local_id, Vote::Propose(propose));
		currentRound = hasVotedYes[H, N]{
			AuthorityID: *localID,
			Vote: votePropose[H, N]{
				PrimaryPropose: propose,
			},
		}
		currentRounds[primitives.RoundNumber(round)] = currentRound

		// 			let set_state = VoterSetState::<Block>::Live {
		// 				completed_rounds: completed_rounds.clone(),
		// 				current_rounds,
		// 			};
		setState := voterSetStateLive[H, N]{
			CompletedRounds: completedRounds,
			CurrentRounds:   currentRounds,
		}

		// 			crate::aux_schema::write_voter_set_state(&*self.client, &set_state)?;
		err = writeVoterSetState(e.Client, setState)
		if err != nil {
			return nil, err
		}

		// 			Ok(Some(set_state))
		return setState, nil
	})
	if err != nil {
		return err
	}

	return nil
}

// fn prevoted(
//
//	&self,
//	round: RoundNumber,
//	prevote: Prevote<Block::Header>,
//
// ) -> Result<(), Self::Error> {
func (e *environment[H, N, Hasher, Header, E]) Prevoted(round uint64, prevote grandpa.Prevote[H, N]) error {
	// 		let local_id = match self.voter_set_state.voting_on(round) {
	// 			Some(id) => id,
	// 			None => return Ok(()),
	// 		};
	localID := e.VoterSetState.votingOn(primitives.RoundNumber(round))
	if localID == nil {
		return nil
	}

	// 		let report_prevote_metrics = |prevote: &Prevote<Block::Header>| {
	// 			telemetry!(
	// 				self.telemetry;
	// 				CONSENSUS_DEBUG;
	// 				"afg.prevote_issued";
	// 				"round" => round,
	// 				"target_number" => ?prevote.target_number,
	// 				"target_hash" => ?prevote.target_hash,
	// 			);

	// 			if let Some(metrics) = self.metrics.as_ref() {
	// 				metrics.finality_grandpa_prevotes.inc();
	// 			}
	// 		};

	// 		self.update_voter_set_state(|voter_set_state| {
	err := e.updateVoterSetState(func(vss voterSetState[H, N]) (voterSetState[H, N], error) {
		// 			let (completed_rounds, current_rounds) = voter_set_state.with_current_round(round)?;
		// 			let current_round = current_rounds
		// 				.get(&round)
		// 				.expect("checked in with_current_round that key exists; qed.");
		completedRounds, currentRounds, err := vss.withCurrentRound(primitives.RoundNumber(round))
		if err != nil {
			return nil, err
		}
		currentRound, ok := currentRounds[primitives.RoundNumber(round)]
		if !ok {
			panic(fmt.Errorf("checked in with_current_round that key exists; qed."))
		}

		// 			if !current_round.can_prevote() {
		// 				// we've already prevoted in this round (in a previous run),
		// 				// ignore the given vote and don't update the voter set
		// 				// state
		// 				return Ok(None)
		// 			}
		if !currentRound.CanPrevote() {
			// we've already prevoted in this round (in a previous run),
			// ignore the given vote and don't update the voter set
			// state
			return nil, nil
		}

		// 			// report to telemetry and prometheus
		// 			report_prevote_metrics(&prevote);

		// 			let propose = current_round.propose();
		propose := currentRound.Propose()

		// 			let mut current_rounds = current_rounds.clone();
		// 			let current_round = current_rounds
		// 				.get_mut(&round)
		// 				.expect("checked previously that key exists; qed.");

		// 			*current_round = HasVoted::Yes(local_id, Vote::Prevote(propose.cloned(), prevote));
		currentRound = hasVotedYes[H, N]{
			AuthorityID: *localID,
			Vote: votePrevote[H, N]{
				PrimaryPropose: propose,
				Prevote:        prevote,
			},
		}
		currentRounds[primitives.RoundNumber(round)] = currentRound

		// 			let set_state = VoterSetState::<Block>::Live {
		// 				completed_rounds: completed_rounds.clone(),
		// 				current_rounds,
		// 			};
		setState := voterSetStateLive[H, N]{
			CompletedRounds: completedRounds,
			CurrentRounds:   currentRounds,
		}

		// 			crate::aux_schema::write_voter_set_state(&*self.client, &set_state)?;
		err = writeVoterSetState(e.Client, setState)
		if err != nil {
			return nil, err
		}

		return setState, nil
	})
	if err != nil {
		return err
	}

	return nil
}

// fn precommitted(
//
//	&self,
//	round: RoundNumber,
//	precommit: Precommit<Block::Header>,
//
// ) -> Result<(), Self::Error> {
func (e *environment[H, N, Hasher, Header, E]) Precommitted(round uint64, precommit grandpa.Precommit[H, N]) error {
	// 		let local_id = match self.voter_set_state.voting_on(round) {
	// 			Some(id) => id,
	// 			None => return Ok(()),
	// 		};
	localID := e.VoterSetState.votingOn(primitives.RoundNumber(round))
	if localID == nil {
		return nil
	}

	// 		let report_precommit_metrics = |precommit: &Precommit<Block::Header>| {
	// 			telemetry!(
	// 				self.telemetry;
	// 				CONSENSUS_DEBUG;
	// 				"afg.precommit_issued";
	// 				"round" => round,
	// 				"target_number" => ?precommit.target_number,
	// 				"target_hash" => ?precommit.target_hash,
	// 			);

	// 			if let Some(metrics) = self.metrics.as_ref() {
	// 				metrics.finality_grandpa_precommits.inc();
	// 			}
	// 		};

	// 		self.update_voter_set_state(|voter_set_state| {
	err := e.updateVoterSetState(func(vss voterSetState[H, N]) (voterSetState[H, N], error) {
		// 			let (completed_rounds, current_rounds) = voter_set_state.with_current_round(round)?;
		// 			let current_round = current_rounds
		// 				.get(&round)
		// 				.expect("checked in with_current_round that key exists; qed.");
		completedRounds, currentRounds, err := vss.withCurrentRound(primitives.RoundNumber(round))
		if err != nil {
			return nil, err
		}
		currentRound, ok := currentRounds[primitives.RoundNumber(round)]
		if !ok {
			panic(fmt.Errorf("checked in with_current_round that key exists; qed."))
		}

		// 			if !current_round.can_precommit() {
		// 				// we've already precommitted in this round (in a previous run),
		// 				// ignore the given vote and don't update the voter set
		// 				// state
		// 				return Ok(None)
		// 			}
		if !currentRound.CanPrecommit() {
			// we've already precommitted in this round (in a previous run),
			// ignore the given vote and don't update the voter set
			// state
			return nil, nil
		}

		// 			// report to telemetry and prometheus
		// 			report_precommit_metrics(&precommit);

		// 			let propose = current_round.propose();
		// 			let prevote = match current_round {
		// 				HasVoted::Yes(_, Vote::Prevote(_, prevote)) => prevote,
		// 				_ => {
		// 					let msg = "Voter precommitting before prevoting.";
		// 					return Err(Error::Safety(msg.to_string()))
		// 				},
		// 			};
		propose := currentRound.Propose()
		var prevote grandpa.Prevote[H, N]
		switch currentRound := currentRound.(type) {
		case hasVotedYes[H, N]:
			vote := currentRound.Vote
			vp, ok := vote.(votePrevote[H, N])
			if !ok {
				return nil, fmt.Errorf("%w: voter precommitting before prevoting", ErrSafety)
			}
			prevote = vp.Prevote
		default:
			return nil, fmt.Errorf("%w: voter precommitting before prevoting", ErrSafety)
		}

		// 			let mut current_rounds = current_rounds.clone();
		// 			let current_round = current_rounds
		// 				.get_mut(&round)
		// 				.expect("checked previously that key exists; qed.");

		// 			*current_round = HasVoted::Yes(
		// 				local_id,
		// 				Vote::Precommit(propose.cloned(), prevote.clone(), precommit),
		// 			);
		currentRound = hasVotedYes[H, N]{
			AuthorityID: *localID,
			Vote: votePrecommit[H, N]{
				PrimaryPropose: propose,
				Prevote:        prevote,
				Precommit:      precommit,
			},
		}
		currentRounds[primitives.RoundNumber(round)] = currentRound

		// 			let set_state = VoterSetState::<Block>::Live {
		// 				completed_rounds: completed_rounds.clone(),
		// 				current_rounds,
		// 			};
		setState := voterSetStateLive[H, N]{
			CompletedRounds: completedRounds,
			CurrentRounds:   currentRounds,
		}

		// 			crate::aux_schema::write_voter_set_state(&*self.client, &set_state)?;
		err = writeVoterSetState(e.Client, setState)
		if err != nil {
			return nil, err
		}

		// 			Ok(Some(set_state))
		return setState, nil
	})
	if err != nil {
		return err
	}

	return nil
}

// fn completed(
//
//	&self,
//	round: RoundNumber,
//	state: RoundState<Block::Hash, NumberFor<Block>>,
//	base: (Block::Hash, NumberFor<Block>),
//	historical_votes: &HistoricalVotes<Block>,
//
// ) -> Result<(), Self::Error> {
func (e *environment[H, N, Hasher, Header, E]) Completed(
	round uint64,
	state grandpa.RoundState[H, N],
	base grandpa.HashNumber[H, N],
	historicalVotes grandpa.HistoricalVotes[H, N, primitives.AuthoritySignature, primitives.AuthorityID],
) error {

	// 		debug!(
	// 			target: LOG_TARGET,
	// 			"Voter {} completed round {} in set {}. Estimate = {:?}, Finalized in round = {:?}",
	// 			self.config.name(),
	// 			round,
	// 			self.set_id,
	// 			state.estimate.as_ref().map(|e| e.1),
	// 			state.finalized.as_ref().map(|e| e.1),
	// 		);
	logger.Debugf("Voter %s completed round %d in set %d. Estimate = %v, Finalized in round = %v",
		e.Config.Name,
		round,
		e.SetID,
		state.Estimate.Number,
		state.Finalized.Number,
	)

	// 		self.update_voter_set_state(|voter_set_state| {
	err := e.updateVoterSetState(func(vss voterSetState[H, N]) (voterSetState[H, N], error) {
		// NOTE: we don't use `with_current_round` here, it is possible that
		// we are not currently tracking this round if it is a round we
		// caught up to.
		// 			let (completed_rounds, current_rounds) =
		// 				if let VoterSetState::Live { completed_rounds, current_rounds } = voter_set_state {
		// 					(completed_rounds, current_rounds)
		// 				} else {
		// 					let msg = "Voter acting while in paused state.";
		// 					return Err(Error::Safety(msg.to_string()))
		// 				};
		live, ok := vss.(voterSetStateLive[H, N])
		if !ok {
			return nil, fmt.Errorf("%w: voter acting while in paused state", ErrSafety)
		}
		completedRounds := live.CompletedRounds
		currentRounds := live.CurrentRounds

		// 			let mut completed_rounds = completed_rounds.clone();

		// TODO: Future integration will store the prevote and precommit index. See #2611.
		// 			let votes = historical_votes.seen().to_vec();
		seen := historicalVotes.Seen()
		votes := make([]primitives.SignedMessage[H, N], len(seen))
		for i, v := range seen {
			votes[i] = primitives.SignedMessage[H, N]{
				SignedMessage: v,
			}
		}

		// 			completed_rounds.push(CompletedRound {
		// 				number: round,
		// 				state: state.clone(),
		// 				base,
		// 				votes,
		// 			});
		completedRounds.push(completedRound[H, N]{
			Number: primitives.RoundNumber(round),
			State:  state,
			Base:   base,
			Votes:  votes,
		})

		// remove the round from live rounds and start tracking the next round
		// 			let mut current_rounds = current_rounds.clone();
		// 			current_rounds.remove(&round);
		delete(currentRounds, primitives.RoundNumber(round))

		// NOTE: this entry should always exist as GRANDPA rounds are always
		// started in increasing order, still it's better to play it safe.
		// 			current_rounds.entry(round + 1).or_insert(HasVoted::No);
		_, ok = currentRounds[primitives.RoundNumber(round+1)]
		if !ok {
			currentRounds[primitives.RoundNumber(round+1)] = hasVotedNo[H, N]{}
		}

		// 			let set_state = VoterSetState::<Block>::Live { completed_rounds, current_rounds };
		setState := voterSetStateLive[H, N]{
			CompletedRounds: completedRounds,
			CurrentRounds:   currentRounds,
		}

		// 			crate::aux_schema::write_voter_set_state(&*self.client, &set_state)?;
		err := writeVoterSetState(e.Client, setState)
		if err != nil {
			return nil, err
		}

		// 			Ok(Some(set_state))
		return setState, nil
	})
	if err != nil {
		return err
	}

	// clear any cached local authority id associated with this round
	// 		self.voter_set_state.finished_voting_on(round);
	e.VoterSetState.finishedVotingOn(primitives.RoundNumber(round))

	// 		Ok(())
	return nil
}

// fn concluded(
//
//	&self,
//	round: RoundNumber,
//	state: RoundState<Block::Hash, NumberFor<Block>>,
//	_base: (Block::Hash, NumberFor<Block>),
//	historical_votes: &HistoricalVotes<Block>,
//
// ) -> Result<(), Self::Error> {
func (e *environment[H, N, Hasher, Header, E]) Concluded(
	round uint64,
	state grandpa.RoundState[H, N],
	base grandpa.HashNumber[H, N],
	historicalVotes grandpa.HistoricalVotes[H, N, primitives.AuthoritySignature, primitives.AuthorityID],
) error {
	// 		debug!(
	// 			target: LOG_TARGET,
	// 			"Voter {} concluded round {} in set {}. Estimate = {:?}, Finalized in round = {:?}",
	// 			self.config.name(),
	// 			round,
	// 			self.set_id,
	// 			state.estimate.as_ref().map(|e| e.1),
	// 			state.finalized.as_ref().map(|e| e.1),
	// 		);
	logger.Debugf("Voter %s concluded round %d in set %d. Estimate = %v, Finalized in round = %v",
		e.Config.Name,
		round,
		e.SetID,
		state.Estimate.Number,
		state.Finalized.Number,
	)

	// 		self.update_voter_set_state(|voter_set_state| {
	err := e.updateVoterSetState(func(vss voterSetState[H, N]) (voterSetState[H, N], error) {
		// NOTE: we don't use `with_current_round` here, because a concluded
		// round is completed and cannot be current.
		// 			let (completed_rounds, current_rounds) =
		// 				if let VoterSetState::Live { completed_rounds, current_rounds } = voter_set_state {
		// 					(completed_rounds, current_rounds)
		// 				} else {
		// 					let msg = "Voter acting while in paused state.";
		// 					return Err(Error::Safety(msg.to_string()))
		// 				};
		live, ok := vss.(voterSetStateLive[H, N])
		if !ok {
			return nil, fmt.Errorf("%w: voter acting while in paused state", ErrSafety)
		}
		completedRounds := live.CompletedRounds
		currentRounds := live.CurrentRounds

		// 			let mut completed_rounds = completed_rounds.clone();

		// 			if let Some(already_completed) =
		// 				completed_rounds.rounds.iter_mut().find(|r| r.number == round)
		// 			{
		alreadyCompletedIndex := slices.IndexFunc(completedRounds.Rounds, func(r completedRound[H, N]) bool {
			return r.Number == primitives.RoundNumber(round)
		})
		if alreadyCompletedIndex >= 0 {
			alreadyCompleted := completedRounds.Rounds[alreadyCompletedIndex]
			// 				let n_existing_votes = already_completed.votes.len();
			nExistingVotes := len(alreadyCompleted.Votes)

			// the interface of Environment guarantees that the previous `historical_votes`
			// from `completable` is a prefix of what is passed to `concluded`.
			// 				already_completed
			// 					.votes
			// 					.extend(historical_votes.seen().iter().skip(n_existing_votes).cloned());
			// 				already_completed.state = state;
			// 				crate::aux_schema::write_concluded_round(&*self.client, already_completed)?;
			toAppend := make([]primitives.SignedMessage[H, N], len(historicalVotes.Seen())-nExistingVotes)
			for i, v := range historicalVotes.Seen()[nExistingVotes:] {
				toAppend[i] = primitives.SignedMessage[H, N]{
					SignedMessage: v,
				}
			}
			alreadyCompleted.Votes = append(alreadyCompleted.Votes, toAppend...)
			alreadyCompleted.State = state
			err := writeConcludedRound(e.Client, alreadyCompleted)
			if err != nil {
				return nil, err
			}
		}

		// 			let set_state = VoterSetState::<Block>::Live {
		// 				completed_rounds,
		// 				current_rounds: current_rounds.clone(),
		// 			};
		setState := voterSetStateLive[H, N]{
			CompletedRounds: completedRounds,
			CurrentRounds:   currentRounds,
		}

		// 			crate::aux_schema::write_voter_set_state(&*self.client, &set_state)?;
		err := writeVoterSetState(e.Client, setState)
		if err != nil {
			return nil, err
		}

		return setState, nil
	})
	if err != nil {
		return err
	}

	return nil
}

// fn finalize_block(
//
//	&self,
//	hash: Block::Hash,
//	number: NumberFor<Block>,
//	round: RoundNumber,
//	commit: Commit<Block::Header>,
//
// ) -> Result<(), Self::Error> {
func (e *environment[H, N, Hasher, Header, E]) FinalizeBlock(
	hash H,
	number N,
	round uint64,
	commit grandpa.Commit[H, N, primitives.AuthoritySignature, primitives.AuthorityID],
) error {
	// 		finalize_block(
	// 			self.client.clone(),
	// 			&self.authority_set,
	// 			Some(self.config.justification_generation_period),
	// 			hash,
	// 			number,
	// 			(round, commit).into(),
	// 			false,
	// 			self.justification_sender.as_ref(),
	// 			self.telemetry.clone(),
	// 		)
	return finalizeBlock(
		e.Client,
		&e.AuthoritySet,
		&e.Config.JustificationGenerationPeriod,
		hash,
		number,
		justificationOrCommitCommit[H, N]{
			RoundNumber: primitives.RoundNumber(round),
			Commit:      commit,
		},
		false,
		e.JustificationSender,
	)
}

// fn round_commit_timer(&self) -> Self::Timer {
func (e *environment[H, N, Hasher, Header, E]) RoundCommitTimer() time.Timer {
	// 		use rand::{thread_rng, Rng};

	// random between `[0, 2 * gossip_duration]` seconds.
	// 		let delay: u64 =
	// 			thread_rng().gen_range(0..2 * self.config.gossip_duration.as_millis() as u64);
	delay := rand.Int64N(2 * e.Config.GossipDuration.Milliseconds())
	// 		Box::pin(Delay::new(Duration::from_millis(delay)).map(Ok))
	timer := time.NewTimer(time.Duration(delay) * time.Millisecond)
	return *timer
}

// fn prevote_equivocation(
//
//	&self,
//	_round: RoundNumber,
//	equivocation: finality_grandpa::Equivocation<
//		Self::Id,
//		Prevote<Block::Header>,
//		Self::Signature,
//	>,
//
// ) {
func (e *environment[H, N, Hasher, Header, E]) PrevoteEquivocation(
	_round uint64,
	equivocation grandpa.Equivocation[primitives.AuthorityID, grandpa.Prevote[H, N], primitives.AuthoritySignature],
) {
	// 		warn!(
	// 			target: LOG_TARGET,
	// 			"Detected prevote equivocation in the finality worker: {:?}", equivocation
	// 		);
	logger.Warnf("Detected prevote equivocation in the finality worker: %v", equivocation)
	// 		if let Err(err) = self.report_equivocation(equivocation.into()) {
	// 			warn!(target: LOG_TARGET, "Error reporting prevote equivocation: {}", err);
	// 		}
	err := e.reportEquivocation(primitives.EquivocationPrevote[H, N](equivocation))
	if err != nil {
		logger.Warnf("Error reporting prevote equivocation: %s", err)
	}
}

func (e *environment[H, N, Hasher, Header, E]) PrecommitEquivocation(
	_round uint64,
	equivocation grandpa.Equivocation[primitives.AuthorityID, grandpa.Precommit[H, N], primitives.AuthoritySignature],
) {
	// 		warn!(
	// 			target: LOG_TARGET,
	// 			"Detected precommit equivocation in the finality worker: {:?}", equivocation
	// 		);
	logger.Warnf("Detected precommit equivocation in the finality worker: %v", equivocation)
	// 		if let Err(err) = self.report_equivocation(equivocation.into()) {
	// 			warn!(target: LOG_TARGET, "Error reporting precommit equivocation: {}", err);
	// 		}
	err := e.reportEquivocation(primitives.EquivocationPrecommit[H, N](equivocation))
	if err != nil {
		logger.Warnf("Error reporting precommit equivocation: %s", err)
	}
}

// async fn best_chain_containing<Block, Backend, Client, SelectChain, VotingRule>(
//
//	block: Block::Hash,
//	client: Arc<Client>,
//	authority_set: SharedAuthoritySet<Block::Hash, NumberFor<Block>>,
//	select_chain: SelectChain,
//	voting_rule: VotingRule,
//
// ) -> Result<Option<(Block::Hash, NumberFor<Block>)>, Error>
// where
//
//	Backend: BackendT<Block>,
//	Block: BlockT,
//	Client: ClientForGrandpa<Block, Backend>,
//	SelectChain: SelectChainT<Block> + 'static,
//	VotingRule: VotingRuleT<Block, Client>,
//
// {
func bestChainContaining[
	H runtime.Hash,
	N runtime.Number,
	Hasher runtime.Hasher[H],
	Header runtime.Header[N, H],
	E runtime.Extrinsic,
](
	block H,
	client ClientForGrandpa[H, N, Hasher, Header, E],
	authoritySet *SharedAuthoritySet[H, N],
	selectChain consensus.SelectChain[H, N, Header],
	votingRule VotingRule[H, N, Header],
) (value *grandpa.HashNumber[H, N], err error) {

	// 	let base_header = match client.header(block)? {
	// 		Some(h) => h,
	// 		None => {
	// 			warn!(
	// 				target: LOG_TARGET,
	// 				"Encountered error finding best chain containing {:?}: couldn't find base block",
	// 				block,
	// 			);

	// 			return Ok(None)
	// 		},
	// 	};
	var baseHeader Header
	h, err := client.Header(block)
	if err != nil {
		return nil, err
	}
	if h == nil {
		logger.Warnf("Encountered error finding best chain containing %s: couldn't find base block: %s", block, err)
		return nil, nil
	}
	baseHeader = *h

	// we refuse to vote beyond the current limit number where transitions are scheduled to occur.
	// once blocks are finalized that make that transition irrelevant or activate it, we will
	// proceed onwards. most of the time there will be no pending transition.  the limit, if any, is
	// guaranteed to be higher than or equal to the given base number.
	limit := authoritySet.currentLimit(baseHeader.Number())
	logger.Debugf("Finding best chain containing block %s with number limit %s", block, limit)

	// 	let mut target_header = match select_chain.finality_target(block, None).await {
	// 		Ok(target_hash) => client
	// 			.header(target_hash)?
	// 			.expect("Header known to exist after `finality_target` call; qed"),
	// 		Err(err) => {
	// 			debug!(
	// 				target: LOG_TARGET,
	// 				"Encountered error finding best chain containing {:?}: couldn't find target block: {}",
	// 				block,
	// 				err,
	// 			);

	// 			// NOTE: in case the given `SelectChain` doesn't provide any block we fallback to using
	// 			// the given base block provided by the GRANDPA voter.
	// 			//
	// 			// For example, `LongestChain` will error if the given block to use as base isn't part
	// 			// of the best chain (as defined by `LongestChain`), which could happen if there was a
	// 			// re-org.
	// 			base_header.clone()
	// 		},
	// 	};
	var targetHeader Header
	th := <-selectChain.FinalityTarget(block, nil)
	if th.Error != nil {
		logger.Debugf("Encountered error finding best chain containing %s: couldn't find target block: %s", block, th.Error)
		// NOTE: in case the given `SelectChain` doesn't provide any block we fallback to using
		// the given base block provided by the GRANDPA voter.
		//
		// For example, `LongestChain` will error if the given block to use as base isn't part
		// of the best chain (as defined by `LongestChain`), which could happen if there was a
		// re-org.
		targetHeader = baseHeader
	} else {
		h, err := client.Header(th.Hash)
		if err != nil {
			return nil, err
		}
		if h == nil {
			panic("Header known to exist after `finality_target` call; qed")
		}
		targetHeader = *h
	}

	// NOTE: this is purposefully done after `finality_target` to prevent a case
	// where in-between these two requests there is a block import and
	// `finality_target` returns something higher than `best_chain`.
	// 	let mut best_header = match select_chain.best_chain().await {
	// 		Ok(best_header) => best_header,
	// 		Err(err) => {
	// 			warn!(
	// 				target: LOG_TARGET,
	// 				"Encountered error finding best chain containing {:?}: couldn't find best block: {}",
	// 				block,
	// 				err,
	// 			);

	// 			return Ok(None)
	// 		},
	// 	};
	var bestHeader Header
	bh := <-selectChain.BestChain()
	if bh.Error != nil {
		logger.Warnf("Encountered error finding best chain containing %s: couldn't find best block: %s", block, bh.Error)
		return nil, nil
	}
	bestHeader = bh.Header

	// 	let is_descendent_of = is_descendent_of(&*client, None);
	isDescendentOf := utils.IsDescendantOf[H, N, Header, E](client, nil)

	// 	if target_header.number() > best_header.number() ||
	// 		target_header.number() == best_header.number() &&
	// 			target_header.hash() != best_header.hash() ||
	// 		!is_descendent_of(&target_header.hash(), &best_header.hash())?
	// 	{
	// 		debug!(
	// 			target: LOG_TARGET,
	// 			"SelectChain returned a finality target inconsistent with its best block. Restricting best block to target block"
	// 		);

	// 		best_header = target_header.clone();
	// 	}
	if targetHeader.Number() > bestHeader.Number() ||
		targetHeader.Number() == bestHeader.Number() && targetHeader.Hash() != bestHeader.Hash() {
		logger.Debugf("SelectChain returned a finality target inconsistent with its best block. Restricting best block to target block")
		bestHeader = targetHeader
	} else {
		isDescendent, err := isDescendentOf(targetHeader.Hash(), bestHeader.Hash())
		if err != nil {
			return nil, err
		}
		if !isDescendent {
			logger.Debugf("SelectChain returned a finality target inconsistent with its best block. Restricting best block to target block")
			bestHeader = targetHeader
		}
	}

	// 	debug!(
	// 		target: LOG_TARGET,
	// 		"SelectChain: finality target: #{} ({}), best block: #{} ({})",
	// 		target_header.number(),
	// 		target_header.hash(),
	// 		best_header.number(),
	// 		best_header.hash(),
	// 	);
	logger.Debugf(
		"SelectChain: finality target: %d (%s), best block: %d (%s)",
		targetHeader.Number(), targetHeader.Hash(), bestHeader.Number(), bestHeader.Hash(),
	)

	// check if our vote is currently being limited due to a pending change,
	// in which case we will restrict our target header to the given limit
	// 	if let Some(target_number) = limit.filter(|limit| limit < target_header.number()) {
	// 		// walk backwards until we find the target block
	// 		loop {
	// 			if *target_header.number() < target_number {
	// 				unreachable!(
	// 					"we are traversing backwards from a known block; \
	// 					 blocks are stored contiguously; \
	// 					 qed"
	// 				);
	// 			}

	// 			if *target_header.number() == target_number {
	// 				break
	// 			}

	// 			target_header = client
	// 				.header(*target_header.parent_hash())?
	// 				.expect("Header known to exist after `finality_target` call; qed");
	// 		}

	// 		debug!(
	// 			target: LOG_TARGET,
	// 			"Finality target restricted to #{} ({}) due to pending authority set change",
	// 			target_header.number(),
	// 			target_header.hash()
	// 		)
	// 	}
	if limit != nil && *limit < targetHeader.Number() {
		// walk backwards until we find the target block
		for {
			if targetHeader.Number() < *limit {
				panic("we are traversing backwards from a known block; blocks are stored contiguously; qed")
			}

			if targetHeader.Number() == *limit {
				break
			}

			h, err := client.Header(targetHeader.ParentHash())
			if err != nil {
				return nil, err
			}
			if h == nil {
				panic("Header known to exist after `finality_target` call; qed")
			}
			targetHeader = *h
		}

		logger.Debugf(
			"Finality target restricted to %d (%s) due to pending authority set change",
			targetHeader.Number(), targetHeader.Hash(),
		)
	}

	// restrict vote according to the given voting rule, if the voting rule
	// doesn't restrict the vote then we keep the previous target.
	//
	// we also make sure that the restricted vote is higher than the round base
	// (i.e. last finalized), otherwise the value returned by the given voting
	// rule is ignored and the original target is used instead.

	// Ok(voting_rule
	//
	//	.restrict_vote(client.clone(), &base_header, &best_header, &target_header)
	//	.await
	//	.filter(|(_, restricted_number)| {
	//		// we can only restrict votes within the interval [base, target]
	//		restricted_number >= base_header.number() && restricted_number < target_header.number()
	//	})
	//	.or_else(|| Some((target_header.hash(), *target_header.number()))))
	vrr := <-votingRule.RestrictVote(client, baseHeader, bestHeader, targetHeader)
	if vrr != nil {
		restrictedNumber := vrr.Number
		// we can only restrict votes within the interval [base, target]
		if restrictedNumber >= baseHeader.Number() && restrictedNumber < targetHeader.Number() {
			value = &grandpa.HashNumber[H, N]{
				Hash:   vrr.Hash,
				Number: restrictedNumber,
			}
			return value, nil
		}
	}
	return &grandpa.HashNumber[H, N]{
		Hash:   targetHeader.Hash(),
		Number: targetHeader.Number(),
	}, nil
}

// / Whether we should process a justification for the given block.
// /
// / This can be used to decide whether to import a justification (when
// / importing a block), or whether to generate a justification from a
// / commit (when validating). Justifications for blocks that change the
// / authority set will always be processed, otherwise we'll only process
// / justifications if the last one was `justification_period` blocks ago.
// pub(crate) fn should_process_justification<BE, Block, Client>(
//
//	client: &Client,
//	justification_period: u32,
//	number: NumberFor<Block>,
//	enacts_change: bool,
//
// ) -> bool
// where
//
//	Block: BlockT,
//	BE: BackendT<Block>,
//	Client: ClientForGrandpa<Block, BE>,
//
// {
func shouldProcessJustification[
	H runtime.Hash,
	N runtime.Number,
	Hasher runtime.Hasher[H],
	Header runtime.Header[N, H],
	E runtime.Extrinsic,
](
	client ClientForGrandpa[H, N, Hasher, Header, E],
	justificationPeriod uint32,
	number N,
	enactsChange bool,
) bool {
	// 	if enacts_change {
	// 		return true
	// 	}
	if enactsChange {
		return true
	}

	// 	let last_finalized_number = client.info().finalized_number;
	lastFinalizedNumber := client.Info().FinalizedNumber

	// keep the first justification before reaching the justification period
	// 	if last_finalized_number.is_zero() {
	// 		return true
	// 	}
	if lastFinalizedNumber == 0 {
		return true
	}

	// last_finalized_number / justification_period.into() != number / justification_period.into()
	return lastFinalizedNumber/N(justificationPeriod) != number/N(justificationPeriod)
}

//	pub(crate) enum JustificationOrCommit<Block: BlockT> {
//		Justification(GrandpaJustification<Block>),
//		Commit((RoundNumber, Commit<Block::Header>)),
//	}
type justificationOrCommit interface {
	isJustificationOrCommit()
}

type justificationOrCommitJustification[H runtime.Hash, N runtime.Number, Header runtime.Header[N, H]] struct {
	GrandpaJustification[H, N, Header]
}

func (justificationOrCommitJustification[H, N, Header]) isJustificationOrCommit() {}

type justificationOrCommitCommit[H runtime.Hash, N runtime.Number] struct {
	primitives.RoundNumber
	grandpa.Commit[H, N, primitives.AuthoritySignature, primitives.AuthorityID]
}

func (justificationOrCommitCommit[H, N]) isJustificationOrCommit() {}

// impl<Block: BlockT> From<(RoundNumber, Commit<Block::Header>)> for JustificationOrCommit<Block> {
// 	fn from(commit: (RoundNumber, Commit<Block::Header>)) -> JustificationOrCommit<Block> {
// 		JustificationOrCommit::Commit(commit)
// 	}
// }

// impl<Block: BlockT> From<GrandpaJustification<Block>> for JustificationOrCommit<Block> {
// 	fn from(justification: GrandpaJustification<Block>) -> JustificationOrCommit<Block> {
// 		JustificationOrCommit::Justification(justification)
// 	}
// }

// / Finalize the given block and apply any authority set changes. If an
// / authority set change is enacted then a justification is created (if not
// / given) and stored with the block when finalizing it.
// / This method assumes that the block being finalized has already been imported.
// pub(crate) fn finalize_block<BE, Block, Client>(
//
//	client: Arc<Client>,
//	authority_set: &SharedAuthoritySet<Block::Hash, NumberFor<Block>>,
//	justification_generation_period: Option<u32>,
//	hash: Block::Hash,
//	number: NumberFor<Block>,
//	justification_or_commit: JustificationOrCommit<Block>,
//	initial_sync: bool,
//	justification_sender: Option<&GrandpaJustificationSender<Block>>,
//	telemetry: Option<TelemetryHandle>,
//
// ) -> Result<(), CommandOrError<Block::Hash, NumberFor<Block>>>
// where
//
//	Block: BlockT,
//	BE: BackendT<Block>,
//	Client: ClientForGrandpa<Block, BE>,
//
// {
func finalizeBlock[
	H runtime.Hash,
	N runtime.Number,
	Hasher runtime.Hasher[H],
	Header runtime.Header[N, H],
	E runtime.Extrinsic,
](
	client ClientForGrandpa[H, N, Hasher, Header, E],
	authoritySet *SharedAuthoritySet[H, N],
	justificationGenerationPeriod *uint32,
	hash H,
	number N,
	justificationOrCommit justificationOrCommit,
	initialSync bool,
	justificationSender *GrandpaJustificationSender[H, N, Header], // can be nil
) error {

	// NOTE: lock must be held through writing to DB to avoid race. this lock
	//       also implicitly synchronizes the check for last finalized number
	//       below.
	// 	let mut authority_set = authority_set.inner();

	// 	let status = client.info();
	status := client.Info()

	// 	if number <= status.finalized_number && client.hash(number)? == Some(hash) {
	if number <= status.FinalizedNumber {
		hash, err := client.Hash(number)
		if err != nil {
			return err
		}
		if hash != nil {
			// This can happen after a forced change (triggered manually from the runtime when
			// finality is stalled), since the voter will be restarted at the median last finalized
			// block, which can be lower than the local best finalized block.
			// 		warn!(target: LOG_TARGET, "Re-finalized block #{:?} ({:?}) in the canonical chain, current best finalized is #{:?}",
			// 				hash,
			// 				number,
			// 				status.finalized_number,
			// 		);
			logger.Warnf("Re-finalized block %s (%d) in the canonical chain, current best finalized is %d",
				hash,
				number,
				status.FinalizedNumber,
			)

			// 		return Ok(())
			return nil
		}
	}

	// FIXME #1483: clone only when changed
	// 	let old_authority_set = authority_set.clone();
	// TODO: do I need to clone this?
	oldAuthoritySet := authoritySet.inner

	// 	let update_res: Result<_, Error> = client.lock_import_and_run(|import_op| {
	var vc voterCommand
	err := client.LockImportRun(func(importOp *api.ClientImportOperation[H, Hasher, N, Header, E]) error {
		// 		let status = authority_set
		// 			.apply_standard_changes(
		// 				hash,
		// 				number,
		// 				&is_descendent_of::<Block, _>(&*client, None),
		// 				initial_sync,
		// 				None,
		// 			)
		// 			.map_err(|e| Error::Safety(e.to_string()))?;
		status, err := authoritySet.applyStandardChanges(
			hash,
			number,
			utils.IsDescendantOf[H, N, Header, E](client, nil),
			initialSync,
		)
		if err != nil {
			return fmt.Errorf("%w: %s", ErrSafety, err)
		}

		// send a justification notification if a sender exists and in case of error log it.
		// 		fn notify_justification<Block: BlockT>(
		// 			justification_sender: Option<&GrandpaJustificationSender<Block>>,
		// 			justification: impl FnOnce() -> Result<GrandpaJustification<Block>, Error>,
		// 		) {
		var notifiyJustificaiton = func(
			justificationSender *GrandpaJustificationSender[H, N, Header],
			justification func() (GrandpaJustification[H, N, Header], error),
		) {

			// 			if let Some(sender) = justification_sender {
			// 				if let Err(err) = sender.notify(justification) {
			// 					warn!(
			// 						target: LOG_TARGET,
			// 						"Error creating justification for subscriber: {}", err
			// 					);
			// 				}
			// 			}
			if justificationSender != nil {
				err := justificationSender.Notify(justification)
				if err != nil {
					logger.Warnf("Error creating justification for subscriber: %s", err)
				}
			}
		}

		// NOTE: this code assumes that honest voters will never vote past a
		// transition block, thus we don't have to worry about the case where
		// we have a transition with `effective_block = N`, but we finalize
		// `N+1`. this assumption is required to make sure we store
		// justifications for transition blocks which will be requested by
		// syncing clients.
		// 		let (justification_required, justification) = match justification_or_commit {
		// 			JustificationOrCommit::Justification(justification) => (true, justification),
		// 			JustificationOrCommit::Commit((round_number, commit)) => {
		// 				let enacts_change = status.new_set_block.is_some();

		// 				let justification_required = justification_generation_period
		// 					.map(|period| {
		// 						should_process_justification(&*client, period, number, enacts_change)
		// 					})
		// 					.unwrap_or(enacts_change);

		// 				let justification =
		// 					GrandpaJustification::from_commit(&client, round_number, commit)?;

		// 				(justification_required, justification)
		// 			},
		// 		};
		var justificationRequired bool
		var justification GrandpaJustification[H, N, Header]
		switch joc := justificationOrCommit.(type) {
		case justificationOrCommitJustification[H, N, Header]:
			justification = joc.GrandpaJustification
			justificationRequired = true
		case justificationOrCommitCommit[H, N]:
			c := justificationOrCommit.(justificationOrCommitCommit[H, N])
			roundNumber := c.RoundNumber
			commit := primitives.Commit[H, N](c.Commit)
			enactsChange := status.NewSetBlock != nil

			justificationRequired = enactsChange
			if justificationGenerationPeriod != nil {
				justificationRequired = shouldProcessJustification(client, *justificationGenerationPeriod, number, enactsChange)
			}

			var err error
			justification, err = NewGrandpaJustificationFromCommit[H, N, Header](client, uint64(roundNumber), commit)
			if err != nil {
				return err
			}
		}

		// 		notify_justification(justification_sender, || Ok(justification.clone()));
		notifiyJustificaiton(justificationSender, func() (GrandpaJustification[H, N, Header], error) {
			return justification, nil
		})

		// 		let persisted_justification = if justification_required {
		// 			Some((GRANDPA_ENGINE_ID, justification.encode()))
		// 		} else {
		// 			None
		// 		};
		var persistedJustificationEngineID *runtime.Justification
		if justificationRequired {
			persistedJustificationEngineID = &runtime.Justification{
				ConsensusEngineID:    primitives.GrandpaEngineID,
				EncodedJustification: scale.MustMarshal(justification),
			}
		}

		// ideally some handle to a synchronization oracle would be used
		// to avoid unconditionally notifying.
		// 		client
		// 			.apply_finality(import_op, hash, persisted_justification, true)
		// 			.map_err(|e| {
		// 				warn!(
		// 					target: LOG_TARGET,
		// 					"Error applying finality to block {:?}: {}",
		// 					(hash, number),
		// 					e
		// 				);
		// 				e
		// 			})?;
		err = client.ApplyFinality(importOp, hash, persistedJustificationEngineID, true)
		if err != nil {
			logger.Warnf("Error applying finality to block {%s, %s}: %s", hash, number, err)
			return err
		}

		// 		debug!(target: LOG_TARGET, "Finalizing blocks up to ({:?}, {})", number, hash);
		logger.Debugf("Finalizing blocks up to (%s, %d)", hash, number)

		// 		telemetry!(
		// 			telemetry;
		// 			CONSENSUS_INFO;
		// 			"afg.finalized_blocks_up_to";
		// 			"number" => ?number, "hash" => ?hash,
		// 		);
		// TODO: telemetry

		// 		crate::aux_schema::update_best_justification(&justification, |insert| {
		// 			apply_aux(import_op, insert, &[])
		// 		})?;
		updateBestJustification[H, N](justification, func(insert []api.KeyValue) error {
			return api.ApplyAux(importOp, insert, nil)
		})

		// 		let new_authorities = if let Some((canon_hash, canon_number)) = status.new_set_block {
		var newAuthorities *newAuthoritySet[H, N]
		if status.NewSetBlock != nil {
			canonHash := status.NewSetBlock.Hash
			canonNumber := status.NewSetBlock.Number
			// the authority set has changed.
			// 			let (new_id, set_ref) = authority_set.current();
			newID, setRef := authoritySet.Current()

			// 			if set_ref.len() > 16 {
			var level func(format string, args ...interface{}) = logger.Debugf
			if initialSync {
				level = logger.Infof
			}
			if len(setRef) > 16 {
				// 				grandpa_log!(
				// 					initial_sync,
				// 					"👴 Applying GRANDPA set change to new set with {} authorities",
				// 					set_ref.len(),
				// 				);
				level("👴 Applying GRANDPA set change to new set with %d authorities", len(setRef))
			} else {
				// 				grandpa_log!(
				// 					initial_sync,
				// 					"👴 Applying GRANDPA set change to new set {:?}",
				// 					set_ref
				// 				);
				level("👴 Applying GRANDPA set change to new set %v", setRef)
			}

			// 			telemetry!(
			// 				telemetry;
			// 				CONSENSUS_INFO;
			// 				"afg.generating_new_authority_set";
			// 				"number" => ?canon_number, "hash" => ?canon_hash,
			// 				"authorities" => ?set_ref.to_vec(),
			// 				"set_id" => ?new_id,
			// 			);
			// TODO: telemetry

			// 			Some(NewAuthoritySet {
			// 				canon_hash,
			// 				canon_number,
			// 				set_id: new_id,
			// 				authorities: set_ref.to_vec(),
			// 			})
			newAuthorities = &newAuthoritySet[H, N]{
				CanonHash:   canonHash,
				CanonNumber: canonNumber,
				SetID:       primitives.SetID(newID),
				Authorities: setRef,
			}
		}

		// 		if status.changed {
		// 			let write_result = crate::aux_schema::update_authority_set::<Block, _, _>(
		// 				&authority_set,
		// 				new_authorities.as_ref(),
		// 				|insert| apply_aux(import_op, insert, &[]),
		// 			);

		// 			if let Err(e) = write_result {
		// 				warn!(
		// 					target: LOG_TARGET,
		// 					"Failed to write updated authority set to disk. Bailing."
		// 				);
		// 				warn!(target: LOG_TARGET, "Node is in a potentially inconsistent state.");

		// 				return Err(e.into())
		// 			}
		// 		}
		if status.Changed {
			err := updateAuthoritySet(authoritySet.inner, newAuthorities, func(insert []api.KeyValue) error {
				return api.ApplyAux(importOp, insert, nil)
			})
			if err != nil {
				logger.Warnf("Failed to write updated authority set to disk. Bailing.")
				logger.Warnf("Node is in a potentially inconsistent state.")
				return err
			}
		}

		// 		Ok(new_authorities.map(VoterCommand::ChangeAuthorities))
		if newAuthorities != nil {
			vc = voterCommandChangeAuthorities[H, N](*newAuthorities)
			return nil
		}
		vc = nil
		return nil
	})

	// 	match update_res {
	// 		Ok(Some(command)) => Err(CommandOrError::VoterCommand(command)),
	// 		Ok(None) => Ok(()),
	// 		Err(e) => {
	// 			*authority_set = old_authority_set;

	//			Err(CommandOrError::Error(e))
	//		},
	//	}
	if vc != nil {
		return vc
	}
	if err != nil {
		authoritySet.inner = oldAuthoritySet
		return err
	}
	return nil
}

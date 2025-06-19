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
	"github.com/ChainSafe/gossamer/internal/primitives/blockchain"
	"github.com/ChainSafe/gossamer/internal/primitives/consensus/common"
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
func (cr completedRounds[H, N]) setInfo() (primitives.SetID, []primitives.AuthorityID) {
	return cr.SetId, cr.Voters
}

// Iterate over all completed rounds.
func (cr completedRounds[H, N]) iter() []completedRound[H, N] {
	var reversed []completedRound[H, N]
	for i := len(cr.Rounds) - 1; i >= 0; i-- {
		reversed = append(reversed, cr.Rounds[i])
	}
	return reversed
}

// Returns the last (latest) completed round
func (cr completedRounds[H, N]) last() completedRound[H, N] {
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
func (svss *SharedVoterSetState[H, N]) votingOn(round primitives.RoundNumber) *primitives.AuthorityID {
	svss.votingMtx.RLock()
	defer svss.votingMtx.RUnlock()
	key, ok := svss.voting[round]
	if !ok {
		return nil
	}
	return &key
}

// Note that we started voting on the give round with the given authority id.
func (svss *SharedVoterSetState[H, N]) startedVotingOn(round primitives.RoundNumber, localID primitives.AuthorityID) {
	svss.votingMtx.Lock()
	defer svss.votingMtx.Unlock()
	svss.voting[round] = localID
}

// Note that we have finished voting on the given round. If we were voting on the given round, the authority id that
// we were using to do it will be cleared.
func (svss *SharedVoterSetState[H, N]) finishedVotingOn(round primitives.RoundNumber) {
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

// The environment we run GRANDPA in.
type environment[
	H runtime.Hash,
	N runtime.Number,
	Hasher runtime.Hasher[H],
	Header runtime.Header[N, H],
	E runtime.Extrinsic,
] struct {
	Client        ClientForGrandpa[H, N, Hasher, Header, E]
	SelectChain   common.SelectChain[H, N, Header]
	Voters        grandpa.VoterSet[primitives.AuthorityID]
	Config        Config
	AuthoritySet  *SharedAuthoritySet[H, N]
	Network       *networkBridge[H, N, Hasher]
	SetID         SetID
	VoterSetState *SharedVoterSetState[H, N]
	VotingRule    VotingRule[H, N, Header]
	// TODO: metrics
	JustificationSender *GrandpaJustificationSender[H, N, Header] // meant to be optional
	// TODO: telemetry
}

// Updates the voter set state using the given closure. The write lock is held during evaluation of the closure and
// the environment's voter set state is set to its result if successful.
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
	return nil
}

// Report the given equivocation to the GRANDPA runtime module. This method generates a session membership proof of
// the offender and then submits an extrinsic to report the equivocation. In particular, the session membership proof
// must be generated at the block at which the given set was active which isn't necessarily the best block if there
// are pending authority set changes.
func (e *environment[H, N, Hasher, Header, E]) reportEquivocation(
	equivocation primitives.Equivocation,
) error {
	localID := e.VoterSetState.votingOn(equivocation.Round())
	if localID != nil {
		if equivocation.Offender() == *localID {
			return fmt.Errorf("refraining from sending equivocation report for our own equivocation")
		}
	}

	isDescendentOf := utils.IsDescendantOf[H, N, Header](e.Client, nil)

	// TODO [Substrate #9158]: Use SelectChain::best_chain() to get a potentially more accurate best block
	info := e.Client.Info()
	bestBlockHash := info.BestHash
	bestBlockNumber := info.BestNumber

	authoritySet, unlock := e.AuthoritySet.inner.DataMut()
	defer unlock()

	// block hash and number of the next pending authority set change in the given best chain.
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
		// the next set starts at n so the current one lasts until n-1. if n is later than the best block, then the
		// current set is still live at best block.
		if nextChange.Number > bestBlockNumber {
			currentSetLatestHash = bestBlockHash
		} else {
			// this is the header at which the new set will start
			header, err := e.Client.Header(nextChange.Hash)
			if err != nil {
				return err
			}
			if header == nil {
				panic("got block hash from registered pending change; " +
					"pending changes are only registered on block import.")
			}
			// its parent block is the last block in the current set
			currentSetLatestHash = (*header).ParentHash()
		}
	} else {
		// there is no pending change, the latest block for the current set is the best block.
		currentSetLatestHash = bestBlockHash
	}

	runtimeAPI := e.Client.RuntimeAPI()

	// generate key ownership proof at that block
	keyOwnerProof := runtimeAPI.GenerateKeyOwnershipProof(
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

	// NOTE: in substrate, this registers the current transaction pool associated with best_block_hash.
	// Given we don't support offchain workers at the moment, I'm keeping this comment here to remember
	// the integration point for this.
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

	// check if base is descendent of block
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

	// skip one because our ancestry is meant to start from the parent of block, and treeRoute includes it.
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
	// variant is `ErrNotDescendent`, this may change in the future as
	// other errors (e.g. IO) are not being exposed.
	_, err := ancestry(e.Client, base, block)
	return err == nil
}

func (e *environment[H, N, Hasher, Header, E]) BestChainContaining(
	block H,
) grandpa.BestChain[H, N] {
	ch := make(grandpa.BestChain[H, N], 1)
	// NOTE: when we finalize an authority set change through the sync protocol the voter is signalled asynchronously
	// therefore the voter could still vote in the next round before activating the new set. the [AuthoritySet] is
	// updated immediately thus we restrict the voter based on that.
	if e.SetID != SetID(e.AuthoritySet.SetID()) {
		ch <- grandpa.BestChainOutput[H, N]{
			Value: nil,
			Error: nil,
		}
		close(ch)
		return ch
	}

	go func() {
		value, err := bestChainContaining(block, e.Client, e.AuthoritySet, e.SelectChain, e.VotingRule)
		ch <- grandpa.BestChainOutput[H, N]{
			Value: value,
			Error: err,
		}
		close(ch)
	}()
	return ch
}

func (e *environment[H, N, Hasher, Header, E]) RoundData(
	round uint64,
) grandpa.RoundData[H, N, primitives.AuthoritySignature, primitives.AuthorityID, grandpa.Message[H, N]] {
	prevoteTimer := time.NewTimer(e.Config.GossipDuration * 2)
	precommitTimer := time.NewTimer(e.Config.GossipDuration * 4)

	localID := localAuthorityID(e.Voters, e.Config.KeyStore)

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

	// NOTE: we cache the local authority id that we'll be using to vote on the given round. this is done to make sure
	// we only check for available keys from the keystore in this method when beginning the round, otherwise if the
	// keystore state changed during the round (e.g. a key was removed) it could lead to internal state inconsistencies
	// in the voter environment (e.g. we wouldn't update the voter set state after prevoting since there's no local
	// authority id).
	if localID != nil {
		e.VoterSetState.startedVotingOn(primitives.RoundNumber(round), *localID)
	}

	// we can only sign when we have a local key in the authority set and we have a reference to the keystore.
	var keystore *localIDKeystore
	if localID != nil && e.Config.KeyStore != nil {
		keystore = &localIDKeystore{
			AuthorityID: *localID,
			KeyStore:    e.Config.KeyStore,
		}
	}

	in, out := e.Network.roundCommunication(keystore, Round(round), e.SetID, e.Voters, hasVoted)

	convertedIn := make(chan signedMessage[H, N])
	go func() {
		for signed := range in {
			convertedIn <- signedMessage[H, N]{signed}
		}
	}()

	// schedule incoming messages from the network to be held until corresponding blocks are imported.
	incoming := newUntilVoteTargetImported(
		e.Client.RegisterImportNotificationStream(),
		e.Network,
		e.Client,
		convertedIn,
		"round",
	)

	convertedOut := make(chan grandpa.SignedMessageError[H, N, primitives.AuthoritySignature, primitives.AuthorityID])
	go func() {
		for be := range incoming.Chan() {
			signed := be.Blocked
			convertedOut <- grandpa.SignedMessageError[H, N, primitives.AuthoritySignature, primitives.AuthorityID]{
				SignedMessage: signed.SignedMessage.SignedMessage,
			}
		}
	}()
	return grandpa.NewRoundData(localID, *prevoteTimer, *precommitTimer, convertedOut, out.preSend)
}

func (e *environment[H, N, Hasher, Header, E]) Proposed(round uint64, propose grandpa.PrimaryPropose[H, N]) error {
	localID := e.VoterSetState.votingOn(primitives.RoundNumber(round))
	if localID == nil {
		return nil
	}

	err := e.updateVoterSetState(func(vss voterSetState[H, N]) (voterSetState[H, N], error) {
		completedRounds, currentRounds, err := vss.withCurrentRound(primitives.RoundNumber(round))
		if err != nil {
			return nil, err
		}
		currentRound, ok := currentRounds[primitives.RoundNumber(round)]
		if !ok {
			panic(fmt.Errorf("checked in withCurrentRound that key exists."))
		}

		if !currentRound.CanPropose() {
			// we've already proposed in this round (in a previous run), ignore the given vote and don't update the
			// voter set state
			return nil, nil
		}

		currentRound = hasVotedYes[H, N]{
			AuthorityID: *localID,
			Vote: votePropose[H, N]{
				PrimaryPropose: propose,
			},
		}
		currentRounds[primitives.RoundNumber(round)] = currentRound

		setState := voterSetStateLive[H, N]{
			CompletedRounds: completedRounds,
			CurrentRounds:   currentRounds,
		}

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

func (e *environment[H, N, Hasher, Header, E]) Prevoted(round uint64, prevote grandpa.Prevote[H, N]) error {
	localID := e.VoterSetState.votingOn(primitives.RoundNumber(round))
	if localID == nil {
		return nil
	}

	// TODO: telemetry and metrics

	return e.updateVoterSetState(func(vss voterSetState[H, N]) (voterSetState[H, N], error) {
		completedRounds, currentRounds, err := vss.withCurrentRound(primitives.RoundNumber(round))
		if err != nil {
			return nil, err
		}
		currentRound, ok := currentRounds[primitives.RoundNumber(round)]
		if !ok {
			panic(fmt.Errorf("checked in with_current_round that key exists; qed."))
		}

		if !currentRound.CanPrevote() {
			// we've already prevoted in this round (in a previous run), ignore the given vote and don't update the
			// voter set state
			return nil, nil
		}

		propose := currentRound.Propose()
		currentRound = hasVotedYes[H, N]{
			AuthorityID: *localID,
			Vote: votePrevote[H, N]{
				PrimaryPropose: propose,
				Prevote:        prevote,
			},
		}
		currentRounds[primitives.RoundNumber(round)] = currentRound

		setState := voterSetStateLive[H, N]{
			CompletedRounds: completedRounds,
			CurrentRounds:   currentRounds,
		}

		err = writeVoterSetState(e.Client, setState)
		if err != nil {
			return nil, err
		}

		return setState, nil
	})
}

func (e *environment[H, N, Hasher, Header, E]) Precommitted(round uint64, precommit grandpa.Precommit[H, N]) error {
	localID := e.VoterSetState.votingOn(primitives.RoundNumber(round))
	if localID == nil {
		return nil
	}

	// TODO: telemetry and metrics

	return e.updateVoterSetState(func(vss voterSetState[H, N]) (voterSetState[H, N], error) {
		completedRounds, currentRounds, err := vss.withCurrentRound(primitives.RoundNumber(round))
		if err != nil {
			return nil, err
		}
		currentRound, ok := currentRounds[primitives.RoundNumber(round)]
		if !ok {
			panic(fmt.Errorf("checked in with_current_round that key exists; qed."))
		}

		if !currentRound.CanPrecommit() {
			// we've already precommitted in this round (in a previous run), ignore the given vote and don't update
			// the voter set state
			return nil, nil
		}

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

		currentRound = hasVotedYes[H, N]{
			AuthorityID: *localID,
			Vote: votePrecommit[H, N]{
				PrimaryPropose: propose,
				Prevote:        prevote,
				Precommit:      precommit,
			},
		}
		currentRounds[primitives.RoundNumber(round)] = currentRound

		setState := voterSetStateLive[H, N]{
			CompletedRounds: completedRounds,
			CurrentRounds:   currentRounds,
		}

		err = writeVoterSetState(e.Client, setState)
		if err != nil {
			return nil, err
		}

		return setState, nil
	})
}

func (e *environment[H, N, Hasher, Header, E]) Completed(
	round uint64,
	state grandpa.RoundState[H, N],
	base grandpa.HashNumber[H, N],
	historicalVotes grandpa.HistoricalVotes[H, N, primitives.AuthoritySignature, primitives.AuthorityID],
) error {
	logger.Debugf("Voter %s completed round %d in set %d. Estimate = %v, Finalized in round = %v",
		e.Config.Name,
		round,
		e.SetID,
		state.Estimate.Number,
		state.Finalized.Number,
	)

	err := e.updateVoterSetState(func(vss voterSetState[H, N]) (voterSetState[H, N], error) {
		// NOTE: we don't use withCurrentRound() here, it is possible that we are not currently tracking this round if
		// it is a round we caught up to.
		live, ok := vss.(voterSetStateLive[H, N])
		if !ok {
			return nil, fmt.Errorf("%w: voter acting while in paused state", ErrSafety)
		}
		completedRounds := live.CompletedRounds
		currentRounds := live.CurrentRounds

		seen := historicalVotes.Seen()
		votes := make([]primitives.SignedMessage[H, N], len(seen))
		for i, v := range seen {
			votes[i] = primitives.SignedMessage[H, N]{
				SignedMessage: v,
			}
		}

		completedRounds.push(completedRound[H, N]{
			Number: primitives.RoundNumber(round),
			State:  state,
			Base:   base,
			Votes:  votes,
		})

		// remove the round from live rounds and start tracking the next round
		delete(currentRounds, primitives.RoundNumber(round))

		// NOTE: this entry should always exist as GRANDPA rounds are always
		// started in increasing order, still it's better to play it safe.
		_, ok = currentRounds[primitives.RoundNumber(round+1)]
		if !ok {
			currentRounds[primitives.RoundNumber(round+1)] = hasVotedNo[H, N]{}
		}

		setState := voterSetStateLive[H, N]{
			CompletedRounds: completedRounds,
			CurrentRounds:   currentRounds,
		}

		err := writeVoterSetState(e.Client, setState)
		if err != nil {
			return nil, err
		}

		return setState, nil
	})
	if err != nil {
		return err
	}

	// clear any cached local authority id associated with this round
	e.VoterSetState.finishedVotingOn(primitives.RoundNumber(round))

	return nil
}

func (e *environment[H, N, Hasher, Header, E]) Concluded(
	round uint64,
	state grandpa.RoundState[H, N],
	base grandpa.HashNumber[H, N],
	historicalVotes grandpa.HistoricalVotes[H, N, primitives.AuthoritySignature, primitives.AuthorityID],
) error {
	logger.Debugf("Voter %s concluded round %d in set %d. Estimate = %v, Finalized in round = %v",
		e.Config.Name,
		round,
		e.SetID,
		state.Estimate.Number,
		state.Finalized.Number,
	)

	return e.updateVoterSetState(func(vss voterSetState[H, N]) (voterSetState[H, N], error) {
		// NOTE: we don't use withCurrentRound() here, because a concluded round is completed and cannot be current.
		live, ok := vss.(voterSetStateLive[H, N])
		if !ok {
			return nil, fmt.Errorf("%w: voter acting while in paused state", ErrSafety)
		}
		completedRounds := live.CompletedRounds
		currentRounds := live.CurrentRounds

		alreadyCompletedIndex := slices.IndexFunc(completedRounds.Rounds, func(r completedRound[H, N]) bool {
			return r.Number == primitives.RoundNumber(round)
		})
		if alreadyCompletedIndex >= 0 {
			alreadyCompleted := completedRounds.Rounds[alreadyCompletedIndex]
			nExistingVotes := len(alreadyCompleted.Votes)

			// the interface of Environment guarantees that the previous historicalVotes
			// from completable is a prefix of what is passed to concluded.
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

		setState := voterSetStateLive[H, N]{
			CompletedRounds: completedRounds,
			CurrentRounds:   currentRounds,
		}

		err := writeVoterSetState(e.Client, setState)
		if err != nil {
			return nil, err
		}

		return setState, nil
	})
}

func (e *environment[H, N, Hasher, Header, E]) FinalizeBlock(
	hash H,
	number N,
	round uint64,
	commit grandpa.Commit[H, N, primitives.AuthoritySignature, primitives.AuthorityID],
) error {
	return finalizeBlock(
		e.Client,
		e.AuthoritySet,
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

func (e *environment[H, N, Hasher, Header, E]) RoundCommitTimer() time.Timer {
	// random duration between [0, 2 * GossipDuration] seconds.
	delay := rand.Int64N(2 * e.Config.GossipDuration.Milliseconds()) //nolint:gosec
	timer := time.NewTimer(time.Duration(delay) * time.Millisecond)
	return *timer
}

func (e *environment[H, N, Hasher, Header, E]) PrevoteEquivocation(
	_round uint64,
	equivocation grandpa.Equivocation[primitives.AuthorityID, grandpa.Prevote[H, N], primitives.AuthoritySignature],
) {
	logger.Warnf("Detected prevote equivocation in the finality worker: %v", equivocation)
	err := e.reportEquivocation(primitives.EquivocationPrevote[H, N](equivocation))
	if err != nil {
		logger.Warnf("Error reporting prevote equivocation: %s", err)
	}
}

func (e *environment[H, N, Hasher, Header, E]) PrecommitEquivocation(
	_round uint64,
	equivocation grandpa.Equivocation[primitives.AuthorityID, grandpa.Precommit[H, N], primitives.AuthoritySignature],
) {
	logger.Warnf("Detected precommit equivocation in the finality worker: %v", equivocation)
	err := e.reportEquivocation(primitives.EquivocationPrecommit[H, N](equivocation))
	if err != nil {
		logger.Warnf("Error reporting precommit equivocation: %s", err)
	}
}

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
	selectChain common.SelectChain[H, N, Header],
	votingRule VotingRule[H, N, Header],
) (value *grandpa.HashNumber[H, N], err error) {

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

	// We refuse to vote beyond the current limit number where transitions are scheduled to occur.
	// Once blocks are finalized that make that transition irrelevant or activate it, we will
	// proceed onwards. most of the time there will be no pending transition.  The limit, if any, is
	// guaranteed to be higher than or equal to the given base number.
	limit := authoritySet.currentLimit(baseHeader.Number())
	logger.Debugf("Finding best chain containing block %s with number limit %s", block, limit)

	var targetHeader Header
	th := <-selectChain.FinalityTarget(block, nil)
	if th.Error != nil {
		logger.Debugf(
			"Encountered error finding best chain containing %s: couldn't find target block: %s", block, th.Error)
		// NOTE: in case the given SelectChain doesn't provide any block we fallback to using the given base block
		// provided by the GRANDPA voter.
		//
		// For example, LongestChain will error if the given block to use as base isn't part of the best chain
		// (as defined by LongestChain), which could happen if there was a re-org.
		targetHeader = baseHeader
	} else {
		h, err := client.Header(th.Hash)
		if err != nil {
			return nil, err
		}
		if h == nil {
			panic("Header known to exist after FinalityTarget call.")
		}
		targetHeader = *h
	}

	// NOTE: this is purposefully done after FinalityTarget to prevent a case where in-between these two requests
	// there is a block import and FinalityTarget returns something higher than BestChain.
	var bestHeader Header
	bh := <-selectChain.BestChain()
	if bh.Error != nil {
		logger.Warnf(
			"Encountered error finding best chain containing %s: couldn't find best block: %s", block, bh.Error)
		return nil, nil //nolint: nilerr
	}
	bestHeader = bh.Header

	isDescendentOf := utils.IsDescendantOf[H, N, Header](client, nil)

	if targetHeader.Number() >= bestHeader.Number() {
		isDescendent, err := isDescendentOf(targetHeader.Hash(), bestHeader.Hash())
		if err != nil {
			return nil, err
		}
		if targetHeader.Hash() != bestHeader.Hash() && !isDescendent {
			logger.Debugf("SelectChain returned a finality target inconsistent with its best block. " +
				"Restricting best block to target block")
			bestHeader = targetHeader
		}
	}

	logger.Debugf(
		"SelectChain: finality target: %d (%s), best block: %d (%s)",
		targetHeader.Number(), targetHeader.Hash(), bestHeader.Number(), bestHeader.Hash(),
	)

	// check if our vote is currently being limited due to a pending change, in which case we will restrict our
	// target header to the given limit
	if limit != nil && *limit < targetHeader.Number() {
		// walk backwards until we find the target block
		for {
			if targetHeader.Number() < *limit {
				panic("we are traversing backwards from a known block; blocks are stored contiguously.")
			}

			if targetHeader.Number() == *limit {
				break
			}

			h, err := client.Header(targetHeader.ParentHash())
			if err != nil {
				return nil, err
			}
			if h == nil {
				panic("Header known to exist after FinalityTarget call.")
			}
			targetHeader = *h
		}

		logger.Debugf(
			"Finality target restricted to %d (%s) due to pending authority set change",
			targetHeader.Number(), targetHeader.Hash(),
		)
	}

	// restrict vote according to the given voting rule, if the voting rule doesn't restrict the vote then we keep
	// the previous target.
	//
	// we also make sure that the restricted vote is higher than the round base (i.e. last finalized), otherwise the
	// value returned by the given voting rule is ignored and the original target is used instead.
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

// Whether we should process a justification for the given block.
//
// This can be used to decide whether to import a justification (when importing a block), or whether to generate a
// justification from a commit (when validating). Justifications for blocks that change the authority set will always
// be processed, otherwise we'll only process justifications if the last one was justificationPeriod blocks ago.
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
	if enactsChange {
		return true
	}

	lastFinalizedNumber := client.Info().FinalizedNumber

	// keep the first justification before reaching the justification period
	if lastFinalizedNumber == 0 {
		return true
	}

	return lastFinalizedNumber/N(justificationPeriod) != number/N(justificationPeriod)
}

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

// Finalize the given block and apply any authority set changes. If an authority set change is enacted then a
// justification is created (if not given) and stored with the block when finalizing it. This method assumes that
// the block being finalized has already been imported.
func finalizeBlock[
	H runtime.Hash,
	N runtime.Number,
	Hasher runtime.Hasher[H],
	Header runtime.Header[N, H],
	E runtime.Extrinsic,
](
	client ClientForGrandpa[H, N, Hasher, Header, E],
	sharedAuthoritySet *SharedAuthoritySet[H, N],
	justificationGenerationPeriod *uint32,
	hash H,
	number N,
	justificationOrCommit justificationOrCommit,
	initialSync bool,
	justificationSender *GrandpaJustificationSender[H, N, Header], // can be nil
) error {

	// NOTE: lock must be held through writing to DB to avoid race. this lock also implicitly synchronises the check
	// for last finalized number below.
	authoritySet, unlock := sharedAuthoritySet.inner.DataMut()
	defer unlock()

	status := client.Info()

	if number <= status.FinalizedNumber {
		hash, err := client.Hash(number)
		if err != nil {
			return err
		}
		if hash != nil {
			// This can happen after a forced change (triggered manually from the runtime when finality is stalled),
			// since the voter will be restarted at the median last finalized block, which can be lower than the local
			// best finalized block.
			logger.Warnf("Re-finalized block %s (%d) in the canonical chain, current best finalized is %d",
				hash,
				number,
				status.FinalizedNumber,
			)
			return nil
		}
	}

	oldAuthoritySet := authoritySet.Clone()

	var vc voterCommand // closure specific variable checked after LockImportRun
	_, err := client.LockImportRun(func(importOp *api.ClientImportOperation[H, Hasher, N, Header, E]) (any, error) {
		status, err := authoritySet.applyStandardChanges(
			hash,
			number,
			utils.IsDescendantOf[H, N, Header](client, nil),
			initialSync,
		)
		if err != nil {
			return nil, fmt.Errorf("%w: %s", ErrSafety, err)
		}

		// send a justification notification if a sender exists and in case of error log it.
		var notifyJustification = func(
			justificationSender *GrandpaJustificationSender[H, N, Header],
			justification func() (GrandpaJustification[H, N, Header], error),
		) {
			if justificationSender != nil {
				err := justificationSender.Notify(justification)
				if err != nil {
					logger.Warnf("Error creating justification for subscriber: %s", err)
				}
			}
		}

		// NOTE: this code assumes that honest voters will never vote past a transition block, thus we don't have to
		// worry about the case where we have a transition with effective_block = N, but we finalize N+1. this
		// assumption is required to make sure we store justifications for transition blocks which will be requested
		// by syncing clients.
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
				justificationRequired = shouldProcessJustification(
					client, *justificationGenerationPeriod, number, enactsChange)
			}

			var err error
			justification, err = NewGrandpaJustificationFromCommit[H, N, Header](client, uint64(roundNumber), commit)
			if err != nil {
				return nil, err
			}
		}

		notifyJustification(justificationSender, func() (GrandpaJustification[H, N, Header], error) {
			return justification, nil
		})

		var persistedJustificationEngineID *runtime.Justification
		if justificationRequired {
			persistedJustificationEngineID = &runtime.Justification{
				ConsensusEngineID:    primitives.GrandpaEngineID,
				EncodedJustification: scale.MustMarshal(justification),
			}
		}

		// ideally some handle to a synchronisation oracle would be used
		// to avoid unconditionally notifying.
		err = client.ApplyFinality(importOp, hash, persistedJustificationEngineID, true)
		if err != nil {
			logger.Warnf("Error applying finality to block {%s, %s}: %s", hash, number, err)
			return nil, err
		}

		logger.Debugf("Finalizing blocks up to (%s, %d)", hash, number)

		// TODO: telemetry

		err = updateBestJustification[H, N](justification, func(insert []api.KeyValue) error {
			return api.ApplyAux(importOp, insert, nil)
		})
		if err != nil {
			return nil, err
		}

		var newAuthorities *newAuthoritySet[H, N]
		if status.NewSetBlock != nil {
			canonHash := status.NewSetBlock.Hash
			canonNumber := status.NewSetBlock.Number
			// the authority set has changed.
			newID, setRef := authoritySet.current()

			var level func(format string, args ...interface{}) = logger.Debugf
			if initialSync {
				level = logger.Infof
			}
			if len(setRef) > 16 {
				level("👴 Applying GRANDPA set change to new set with %d authorities", len(setRef))
			} else {
				level("👴 Applying GRANDPA set change to new set %v", setRef)
			}

			// TODO: telemetry

			newAuthorities = &newAuthoritySet[H, N]{
				CanonHash:   canonHash,
				CanonNumber: canonNumber,
				SetID:       primitives.SetID(newID),
				Authorities: setRef,
			}
		}

		if status.Changed {
			err := updateAuthoritySet(*authoritySet, newAuthorities, func(insert []api.KeyValue) error {
				return api.ApplyAux(importOp, insert, nil)
			})
			if err != nil {
				logger.Warnf("Failed to write updated authority set to disk. Bailing.")
				logger.Warnf("Node is in a potentially inconsistent state.")
				return nil, err
			}
		}

		if newAuthorities != nil {
			vc = voterCommandChangeAuthorities[H, N](*newAuthorities)
			return vc, nil
		}
		vc = nil
		return vc, nil
	})

	if vc != nil {
		return vc
	}
	if err != nil {
		*authoritySet = oldAuthoritySet
		return err
	}
	return nil
}

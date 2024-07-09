// Copyright 2023 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package grandpa

import (
	"fmt"
	"sync"

	"github.com/tidwall/btree"
	"golang.org/x/exp/constraints"
)

// Phase is the (voting) phases of a round, each corresponding to the type of
// votes cast in that phase.
type Phase uint

const (
	// The prevote phase in which `Prevote`s are cast.
	PrevotePhase Phase = iota
	// The precommit phase in which `Precommit`s are cast.
	PrecommitPhase
)

type voteSignature[Vote, Signature comparable] struct {
	Vote      Vote
	Signature Signature
}

type single[Vote, Signature comparable] voteSignature[Vote, Signature]

type equivocated[Vote, Signature comparable] [2]voteSignature[Vote, Signature]

// The observed vote from a single voter.
type voteMultiplicity[Vote, Signature comparable] struct {
	value interface{}
}

// can only use type constraint interfaces as function parameters
type voteMultiplicityValue[Vote, Signature comparable] interface {
	single[Vote, Signature] | equivocated[Vote, Signature]
}

func setvoteMultiplicity[
	Vote, Signature comparable,
	T voteMultiplicityValue[Vote, Signature],
](vm *voteMultiplicity[Vote, Signature], val T) {
	vm.value = val
}

func newVoteMultiplicity[
	Vote, Signature comparable,
	T voteMultiplicityValue[Vote, Signature],
](val T) (vm voteMultiplicity[Vote, Signature]) {
	return voteMultiplicity[Vote, Signature]{
		value: val,
	}
}

func (vm voteMultiplicity[Vote, Signature]) Value() interface{} {
	return vm.value
}

func (vm voteMultiplicity[Vote, Signature]) Contains(vote Vote, sig Signature) bool {
	vs := voteSignature[Vote, Signature]{vote, sig}
	switch in := vm.Value().(type) {
	case single[Vote, Signature]:
		return voteSignature[Vote, Signature](in) == vs
	case equivocated[Vote, Signature]:
		return in[0] == vs || in[1] == vs
	default:
		panic("invalid voteMultiplicityValue")
	}
}

type voteTracker[ID constraints.Ordered, Vote, Signature comparable] struct {
	votes         *btree.Map[ID, voteMultiplicity[Vote, Signature]]
	currentWeight VoteWeight
	mtx           sync.RWMutex
}

func newVoteTracker[ID constraints.Ordered, Vote, Signature comparable]() voteTracker[ID, Vote, Signature] {
	return voteTracker[ID, Vote, Signature]{
		votes: btree.NewMap[ID, voteMultiplicity[Vote, Signature]](2),
	}
}

// track a vote, returning a value containing the multiplicity of all votes from this ID
// and a bool indicating if the vote is duplicated.
// if the vote is the first equivocation, returns a value indicating
// it as such (the new vote is always the last in the multiplicity).
//
// if the vote is a further equivocation, it is ignored and there is nothing
// to do.
//
// since this struct doesn't track the round-number of votes, that must be set
// by the caller.
func (vt *voteTracker[ID, Vote, Signature]) addVote(
	id ID,
	vote Vote,
	signature Signature,
	weight VoterWeight,
) (*voteMultiplicity[Vote, Signature], bool) {
	vt.mtx.Lock()
	defer vt.mtx.Unlock()

	var ok bool
	vm, ok := vt.votes.Get(id)
	if !ok {
		// TODO: figure out saturating_add stuff
		// https://github.com/ChainSafe/gossamer/issues/3511
		vt.currentWeight = vt.currentWeight + VoteWeight(weight)
		multiplicity := newVoteMultiplicity[Vote, Signature](
			single[Vote, Signature]{vote, signature},
		)
		_, exists := vt.votes.Set(id, multiplicity)
		if exists {
			panic(fmt.Errorf("id %v should not exist in votes", id))
		}
		return &multiplicity, false
	}

	duplicated := vm.Contains(vote, signature)
	if duplicated {
		return nil, true
	}

	switch in := vm.Value().(type) {
	case single[Vote, Signature]:
		var eq = equivocated[Vote, Signature]{
			voteSignature[Vote, Signature](in),
			{
				Vote:      vote,
				Signature: signature,
			},
		}
		setvoteMultiplicity(&vm, eq)
		vt.votes.Set(id, vm)
		return &vm, false
	case equivocated[Vote, Signature]:
		// ignore further equivocations
		return nil, duplicated
	default:
		panic("invalid voteMultiplicity value")
		return nil, false
	}
}

type idVoteSignature[ID, Vote, Signature comparable] struct {
	ID ID
	voteSignature[Vote, Signature]
}

func (vt *voteTracker[ID, Vote, Signature]) Votes() (votes []idVoteSignature[ID, Vote, Signature]) {
	vt.mtx.RLock()
	defer vt.mtx.RUnlock()

	vt.votes.Scan(func(id ID, vm voteMultiplicity[Vote, Signature]) bool {
		switch in := vm.Value().(type) {
		case single[Vote, Signature]:
			votes = append(votes, idVoteSignature[ID, Vote, Signature]{
				ID:            id,
				voteSignature: voteSignature[Vote, Signature](in),
			})
		case equivocated[Vote, Signature]:
			for _, vs := range in {
				votes = append(votes, idVoteSignature[ID, Vote, Signature]{
					ID:            id,
					voteSignature: vs,
				})
			}
		default:
			panic("invalid voteMultiplicity value")
		}
		return true
	})
	return
}

func (vt *voteTracker[ID, Vote, Signature]) participation() (weight VoteWeight, numParticipants int) {
	return vt.currentWeight, vt.votes.Len()
}

// RoundState is the state of the round.
type RoundState[Hash, Number any] struct {
	// The prevote-GHOST block.
	PrevoteGHOST *HashNumber[Hash, Number]
	// The finalized block.
	Finalized *HashNumber[Hash, Number]
	// The new round-estimate.
	Estimate *HashNumber[Hash, Number]
	// Whether the round is completable.
	Completable bool
}

// NewRoundState is constructor of `RoundState` from a given genesis state.
func NewRoundState[Hash, Number any](genesis HashNumber[Hash, Number]) RoundState[Hash, Number] {
	return RoundState[Hash, Number]{
		PrevoteGHOST: &genesis,
		Finalized:    &genesis,
		Estimate:     &genesis,
		Completable:  true,
	}
}

// RoundParams are the parameters for starting a round.
type RoundParams[ID constraints.Ordered, Hash comparable, Number constraints.Unsigned] struct {
	// The round number for votes.
	RoundNumber uint64
	// Actors and weights in the round.
	Voters VoterSet[ID]
	// The base block to build on.
	Base HashNumber[Hash, Number]
}

// Round stores data for a round.
type Round[ID constraints.Ordered, Hash constraints.Ordered, Number constraints.Unsigned, Signature comparable] struct {
	number          uint64
	context         context[ID]
	graph           VoteGraph[Hash, Number, *voteNode[ID], vote[ID]]    // DAG of blocks which have been voted on.
	prevotes        voteTracker[ID, Prevote[Hash, Number], Signature]   // tracks prevotes that have been counted
	precommits      voteTracker[ID, Precommit[Hash, Number], Signature] // tracks precommits
	historicalVotes HistoricalVotes[Hash, Number, Signature, ID]        // historical votes
	prevoteGhost    *HashNumber[Hash, Number]                           // current memoized prevote-GHOST block
	precommitGhost  *HashNumber[Hash, Number]                           // current memoized precommit-GHOST block
	finalized       *HashNumber[Hash, Number]                           // best finalized block in this round.
	estimate        *HashNumber[Hash, Number]                           // current memoized round-estimate
	completable     bool                                                // whether the round is completable
}

// Result of importing a Prevote or Precommit.
type importResult[ID constraints.Ordered, P, Signature comparable] struct {
	ValidVoter   bool
	Duplicated   bool
	Equivocation *Equivocation[ID, P, Signature]
}

// NewRound creates a new round accumulator for given round number and with given weight.
func NewRound[ID constraints.Ordered, Hash constraints.Ordered, Number constraints.Unsigned, Signature comparable](
	roundParams RoundParams[ID, Hash, Number],
) *Round[ID, Hash, Number, Signature] {

	var newVoteNode = func() *voteNode[ID] {
		return &voteNode[ID]{newBitfield()}
	}
	return &Round[ID, Hash, Number, Signature]{
		number:  roundParams.RoundNumber,
		context: newContext(roundParams.Voters),
		graph: NewVoteGraph[Hash, Number, *voteNode[ID], vote[ID]](
			roundParams.Base.Hash,
			roundParams.Base.Number,
			newVoteNode(),
			newVoteNode,
		),
		prevotes:        newVoteTracker[ID, Prevote[Hash, Number], Signature](),
		precommits:      newVoteTracker[ID, Precommit[Hash, Number], Signature](),
		historicalVotes: NewHistoricalVotes[Hash, Number, Signature, ID](),
	}
}

// Number returns the round number.
func (r *Round[ID, H, N, S]) Number() uint64 {
	return r.number
}

// Import a prevote. Returns an equivocation proof, if the vote is an equivocation,
// and a bool indicating if the vote is duplicated (see `ImportResult`).
//
// Ignores duplicate prevotes (not equivocations).
func (r *Round[ID, H, N, S]) importPrevote(
	chain Chain[H, N], prevote Prevote[H, N], signer ID, signature S,
) (*importResult[ID, Prevote[H, N], S], error) {
	ir := importResult[ID, Prevote[H, N], S]{}

	info := r.context.Voters().Get(signer)
	if info == nil {
		return &ir, nil
	}

	ir.ValidVoter = true
	weight := info.weight

	var equivocation *Equivocation[ID, Prevote[H, N], S]
	var multiplicity *voteMultiplicity[Prevote[H, N], S]
	m, duplicated := r.prevotes.addVote(signer, prevote, signature, weight)
	if m != nil {
		multiplicity = m
	} else {
		ir.Duplicated = duplicated
		return &ir, nil
	}

	switch val := multiplicity.Value().(type) {
	case single[Prevote[H, N], S]:
		singleVote := val
		vote := newVote[ID](*info, PrevotePhase)
		err := r.graph.Insert(singleVote.Vote.TargetHash, singleVote.Vote.TargetNumber, vote, chain)
		if err != nil {
			return nil, err
		}

		// Push the vote into HistoricalVotes.
		message := Message[H, N]{}
		setMessage(&message, prevote)
		signedMessage := SignedMessage[H, N, S, ID]{
			Message:   message,
			Signature: signature,
			ID:        signer,
		}
		r.historicalVotes.PushVote(signedMessage)

	case equivocated[Prevote[H, N], S]:
		first := val[0]
		second := val[1]

		// mark the equivocator as such. no need to "undo" the first vote.
		r.context.Equivocated(*info, PrevotePhase)

		// Push the vote into HistoricalVotes.
		message := Message[H, N]{}
		setMessage(&message, prevote)
		signedMessage := SignedMessage[H, N, S, ID]{
			Message:   message,
			Signature: signature,
			ID:        signer,
		}
		r.historicalVotes.PushVote(signedMessage)
		equivocation = &Equivocation[ID, Prevote[H, N], S]{
			RoundNumber: r.number,
			Identity:    signer,
			First:       first,
			Second:      second,
		}
	default:
		panic("invalid voteMultiplicity value")
	}

	// update prevote-GHOST
	threshold := r.context.voters.threshold
	if r.prevotes.currentWeight >= VoteWeight(threshold) {
		r.prevoteGhost = r.graph.FindGHOST(r.prevoteGhost, func(v *voteNode[ID]) bool {
			return r.context.Weight(*v, PrevotePhase) >= VoteWeight(threshold)
		})
	}

	r.update()
	ir.Equivocation = equivocation
	return &ir, nil
}

// Import a precommit. Returns an equivocation proof, if the vote is an
// equivocation, and a bool indicating if the vote is duplicated (see `ImportResult`).
//
// Ignores duplicate precommits (not equivocations).
func (r *Round[ID, H, N, S]) importPrecommit(
	chain Chain[H, N], precommit Precommit[H, N], signer ID, signature S,
) (*importResult[ID, Precommit[H, N], S], error) {
	ir := importResult[ID, Precommit[H, N], S]{}

	info := r.context.Voters().Get(signer)
	if info == nil {
		return &ir, nil
	}

	ir.ValidVoter = true
	weight := info.weight

	var equivocation *Equivocation[ID, Precommit[H, N], S]
	var multiplicity *voteMultiplicity[Precommit[H, N], S]
	m, duplicated := r.precommits.addVote(signer, precommit, signature, weight)
	if m != nil {
		multiplicity = m
	} else {
		ir.Duplicated = duplicated
		return &ir, nil
	}

	switch val := multiplicity.Value().(type) {
	case single[Precommit[H, N], S]:
		singleVote := val
		vote := newVote[ID](*info, PrecommitPhase)
		err := r.graph.Insert(singleVote.Vote.TargetHash, singleVote.Vote.TargetNumber, vote, chain)
		if err != nil {
			return nil, err
		}

		// Push the vote into HistoricalVotes.
		message := Message[H, N]{}
		setMessage(&message, precommit)
		signedMessage := SignedMessage[H, N, S, ID]{
			Message:   message,
			Signature: signature,
			ID:        signer,
		}
		r.historicalVotes.PushVote(signedMessage)

	case equivocated[Precommit[H, N], S]:
		first := val[0]
		second := val[1]

		// mark the equivocator as such. no need to "undo" the first vote.
		r.context.Equivocated(*info, PrecommitPhase)

		// Push the vote into HistoricalVotes.
		message := Message[H, N]{}
		setMessage(&message, precommit)
		signedMessage := SignedMessage[H, N, S, ID]{
			Message:   message,
			Signature: signature,
			ID:        signer,
		}
		r.historicalVotes.PushVote(signedMessage)
		equivocation = &Equivocation[ID, Precommit[H, N], S]{
			RoundNumber: r.number,
			Identity:    signer,
			First:       first,
			Second:      second,
		}
	default:
		panic("invalid voteMultiplicity value")
	}

	r.update()
	ir.Equivocation = equivocation
	return &ir, nil
}

// update the round-estimate and whether the round is completable.
func (r *Round[ID, H, N, S]) update() {
	threshold := r.context.voters.threshold

	if r.prevotes.currentWeight < VoteWeight(threshold) {
		return
	}

	if r.prevoteGhost == nil {
		return
	}

	// anything new finalized? finalized blocks are those which have both
	// 2/3+ prevote and precommit weight.
	currentPrecommits := r.precommits.currentWeight
	if currentPrecommits >= VoteWeight(threshold) {
		r.finalized = r.graph.FindAncestor(r.prevoteGhost.Hash, r.prevoteGhost.Number, func(v *voteNode[ID]) bool {
			return r.context.Weight(*v, PrecommitPhase) >= VoteWeight(threshold)
		})
	}

	// figuring out whether a block can still be committed for is
	// not straightforward because we have to account for all possible future
	// equivocations and thus cannot discount weight from validators who
	// have already voted.
	var possibleToPrecommit = func(node *voteNode[ID]) bool {
		// find how many more equivocations we could still get.
		//
		// it is only important to consider the voters whose votes
		// we have already seen, because we are assuming any votes we
		// haven't seen will target this block.
		toleratedEquivocations := VoteWeight(r.context.voters.totalWeight - threshold)
		currentEquivocations := r.context.EquivocationWeight(PrecommitPhase)
		additionalEquiv := toleratedEquivocations - currentEquivocations
		remainingCommitVotes := VoteWeight(r.context.voters.totalWeight) - r.precommits.currentWeight

		// total precommits for this block, including equivocations.
		precommitedFor := r.context.Weight(*node, PrecommitPhase)

		// equivocations we could still get are out of those who
		// have already voted, but not on this block.
		var possibleEquivocations VoteWeight
		if currentPrecommits-precommitedFor <= additionalEquiv {
			possibleEquivocations = currentPrecommits - precommitedFor
		} else {
			possibleEquivocations = additionalEquiv
		}

		// all the votes already applied on this block,
		// assuming all remaining actors commit to this block,
		// and that we get further equivocations
		fullPossibleWeight := precommitedFor + remainingCommitVotes + possibleEquivocations
		return fullPossibleWeight >= VoteWeight(threshold)
	}

	// until we have threshold precommits, any new block could get supermajority
	// precommits because there are at least f + 1 precommits remaining and then
	// f equivocations.
	//
	// once it's at least that level, we only need to consider blocks
	// already referenced in the graph, because no new leaf nodes
	// could ever have enough precommits.
	//
	// the round-estimate is the highest block in the chain with head
	// `prevote_ghost` that could have supermajority-commits.
	if r.precommits.currentWeight >= VoteWeight(threshold) {
		r.estimate = r.graph.FindAncestor(r.prevoteGhost.Hash, r.prevoteGhost.Number, possibleToPrecommit)
	} else {
		r.estimate = &HashNumber[H, N]{r.prevoteGhost.Hash, r.prevoteGhost.Number}
		return
	}

	if r.estimate != nil {
		var ls bool = r.estimate.Hash != r.prevoteGhost.Hash
		var rs bool
		x := r.graph.FindGHOST(r.estimate, possibleToPrecommit)
		if x == nil {
			rs = true
		} else {
			rs = *x == *r.prevoteGhost
		}
		r.completable = ls || rs
	} else {
		r.completable = false
	}
}

// State returns the current state.
func (r *Round[ID, H, N, S]) State() RoundState[H, N] {
	return RoundState[H, N]{
		PrevoteGHOST: r.prevoteGhost,
		Finalized:    r.finalized,
		Estimate:     r.estimate,
		Completable:  r.completable,
	}
}

// PrecommitGHOST will compute and cache the precommit-GHOST.
func (r *Round[ID, H, N, S]) PrecommitGHOST() *HashNumber[H, N] {
	// update precommit-GHOST
	var threshold = r.Threshold()

	if r.precommits.currentWeight >= VoteWeight(threshold) {
		r.precommitGhost = r.graph.FindGHOST(r.precommitGhost, func(v *voteNode[ID]) bool {
			return r.context.Weight(*v, PrecommitPhase) >= VoteWeight(threshold)
		})
	}
	return r.precommitGhost
}

type yieldVotes[H constraints.Ordered, N constraints.Unsigned, S comparable] struct {
	yielded      uint
	multiplicity voteMultiplicity[Precommit[H, N], S]
}

func (yv *yieldVotes[H, N, S]) voteSignature() *voteSignature[Precommit[H, N], S] {
	switch vm := yv.multiplicity.Value().(type) {
	case single[Precommit[H, N], S]:
		if yv.yielded == 0 {
			yv.yielded++
			return &voteSignature[Precommit[H, N], S]{vm.Vote, vm.Signature}
		}
		return nil
	case equivocated[Precommit[H, N], S]:
		a := vm[0]
		b := vm[1]
		switch yv.yielded {
		case 0:
			return &a
		case 1:
			return &b
		default:
			return nil
		}
	default:
		panic("invalid voteMultiplicity value")
	}
}

// FinalizingPrecommits returns all precommits targeting the finalized hash.
//
// Only returns `nil` if no block has been finalized in this round.
func (r *Round[ID, H, N, S]) FinalizingPrecommits(chain Chain[H, N]) *[]SignedPrecommit[H, N, S, ID] {
	type idvoteMultiplicity struct {
		ID               ID
		voteMultiplicity voteMultiplicity[Precommit[H, N], S]
	}

	if r.finalized == nil {
		return nil
	}
	fHash := r.finalized.Hash
	var filtered []idvoteMultiplicity
	var findValidPrecommits []SignedPrecommit[H, N, S, ID]
	r.precommits.votes.Scan(func(id ID, multiplicity voteMultiplicity[Precommit[H, N], S]) bool {
		switch multiplicityValue := multiplicity.Value().(type) {
		case single[Precommit[H, N], S]:
			// if there is a single vote from this voter, we only include it
			// if it branches off of the target.
			if chain.IsEqualOrDescendantOf(fHash, multiplicityValue.Vote.TargetHash) {
				filtered = append(filtered, idvoteMultiplicity{id, multiplicity})
			}
		default:
			// equivocations count for everything, so we always include them.
			filtered = append(filtered, idvoteMultiplicity{id, multiplicity})
		}
		return true
	})
	for _, ivm := range filtered {
		yieldVotes := yieldVotes[H, N, S]{
			yielded:      0,
			multiplicity: ivm.voteMultiplicity,
		}
		if vs := yieldVotes.voteSignature(); vs != nil {
			findValidPrecommits = append(findValidPrecommits, SignedPrecommit[H, N, S, ID]{
				Precommit: vs.Vote,
				Signature: vs.Signature,
				ID:        ivm.ID,
			})
		}
	}
	return &findValidPrecommits
}

// Estimate will fetch the "round-estimate": the best block which might have been finalized
// in this round.
//
// Returns `nil` when new new blocks could have been finalized in this round,
// according to our estimate.
func (r *Round[ID, H, N, S]) Estimate() *HashNumber[H, N] {
	return r.estimate
}

// Finalized fetches the most recently finalized block.
func (r *Round[ID, H, N, S]) Finalized() *HashNumber[H, N] {
	return r.finalized
}

// Completable returns `true` when the round is completable.
//
// This is the case when the round-estimate is an ancestor of the prevote-ghost head,
// or when they are the same block _and_ none of its children could possibly have
// enough precommits.
func (r *Round[ID, H, N, S]) Completable() bool {
	return r.completable
}

// Threshold weight for supermajority.
func (r *Round[ID, H, N, S]) Threshold() VoterWeight {
	return r.context.voters.threshold
}

// Base returns the round base.
func (r *Round[ID, H, N, S]) Base() HashNumber[H, N] {
	return r.graph.Base()
}

// Voters returns the round voters and weights.
func (r *Round[ID, H, N, S]) Voters() VoterSet[ID] {
	return r.context.voters
}

// PrimaryVoter returns the primary voter of the round.
func (r *Round[ID, H, N, S]) PrimaryVoter() (ID, VoterInfo) {
	IDVoterInfo := r.context.Voters().NthMod(uint(r.number))
	return IDVoterInfo.ID, IDVoterInfo.VoterInfo
}

// PrevoteParticipation returns the current weight and number of voters who have participated in prevoting.
func (r *Round[ID, H, N, S]) PrevoteParticipation() (weight VoteWeight, numParticipants int) {
	return r.prevotes.participation()
}

// PrecommitParticipation returns the current weight and number of voters who have participated in precommitting.
func (r *Round[ID, H, N, S]) PrecommitParticipation() (weight VoteWeight, numParticipants int) {
	return r.precommits.participation()
}

// Prevotes returns all imported prevotes.
func (r *Round[ID, H, N, S]) Prevotes() []idVoteSignature[ID, Prevote[H, N], S] {
	return r.prevotes.Votes()
}

// Precommits returns all imported precommits.
func (r *Round[ID, H, N, S]) Precommits() []idVoteSignature[ID, Precommit[H, N], S] {
	return r.precommits.Votes()
}

// HistoricalVotes returns all votes for the round (prevotes and precommits), sorted by
// imported order and indicating the indices where we voted. At most two
// prevotes and two precommits per voter are present, further equivocations
// are not stored (as they are redundant).
func (r *Round[ID, H, N, S]) HistoricalVotes() HistoricalVotes[H, N, S, ID] {
	return r.historicalVotes
}

// SetPrevotedIdx will set the number of prevotes and precommits received at the moment of prevoting.
// It should be called inmediatly after prevoting.
func (r *Round[ID, H, N, S]) SetPrevotedIdx() {
	r.historicalVotes.SetPrevotedIdx()
}

// SetPrecommittedIdx will set the number of prevotes and precommits received at the moment of precommiting.
// It should be called inmediatly after precommiting.
func (r *Round[ID, H, N, S]) SetPrecommittedIdx() {
	r.historicalVotes.SetPrecommittedIdx()
}

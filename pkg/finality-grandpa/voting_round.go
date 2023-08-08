// Copyright 2023 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package grandpa

import (
	"fmt"
	"time"

	"golang.org/x/exp/constraints"
)

type Start[T any] [2]T

type Proposed[T any] [2]T

type Prevoting[T, W any] struct {
	T T
	W W
}

type Prevoted[T any] [1]T

type Precommitted struct{}

type States[T, W any] interface {
	Start[T] | Proposed[T] | Prevoting[T, W] | Prevoted[T] | Precommitted
}

type State any

type hashBestChain[Hash comparable, Number constraints.Unsigned] struct {
	Hash      Hash
	BestChain BestChain[Hash, Number]
}

// Whether we should vote in the current round (i.e. push votes to the sink.)
type voting uint

const (
	// Voting is disabled for the current round.
	votingNo voting = iota
	// Voting is enabled for the current round (prevotes and precommits.)
	votingYes
	// Voting is enabled for the current round and we are the primary proposer
	// (we can also push primary propose messages).
	votingPrimary
)

// Whether the voter should cast round votes (prevotes and precommits.)
func (v voting) isActive() bool {
	return v == votingYes || v == votingPrimary
}

// Whether the voter is the primary proposer.
func (v voting) isPrimary() bool {
	return v == votingPrimary
}

// Logic for a voter on a specific round.
type VotingRound[
	Hash constraints.Ordered,
	Number constraints.Unsigned,
	Signature comparable,
	ID constraints.Ordered,
	E Environment[Hash, Number, Signature, ID],
] struct {
	env    E
	voting voting
	// this is not an Option in the rust code. Using a pointer for copylocks
	votes             *Round[ID, Hash, Number, Signature]
	incoming          *wakerChan[SignedMessageError[Hash, Number, Signature, ID]]
	outgoing          *Buffered[Message[Hash, Number]]
	state             State
	bridgedRoundState *priorView[Hash, Number]
	lastRoundState    *latterView[Hash, Number]
	primaryBlock      *HashNumber[Hash, Number]
	finalizedSender   chan finalizedNotification[Hash, Number, Signature, ID]
	bestFinalized     *Commit[Hash, Number, Signature, ID]
}

// Create a new voting round.
func NewVotingRound[
	Hash constraints.Ordered, Number constraints.Unsigned, Signature comparable, ID constraints.Ordered,
	E Environment[Hash, Number, Signature, ID],
](
	roundNumber uint64, voters VoterSet[ID], base HashNumber[Hash, Number],
	lastRoundState *latterView[Hash, Number],
	finalizedSender chan finalizedNotification[Hash, Number, Signature, ID], env E,
) VotingRound[Hash, Number, Signature, ID, E] {
	outgoing := make(chan Message[Hash, Number])
	roundData := env.RoundData(roundNumber, outgoing)
	roundParams := RoundParams[ID, Hash, Number]{
		RoundNumber: roundNumber,
		Voters:      voters,
		Base:        base,
	}

	votes := NewRound[ID, Hash, Number, Signature](roundParams)

	primaryVoterID, _ := votes.PrimaryVoter()
	var voting voting //nolint:govet
	if roundData.VoterID != nil && *roundData.VoterID == primaryVoterID {
		voting = votingPrimary
	} else if roundData.VoterID != nil && votes.Voters().Contains(*roundData.VoterID) {
		voting = votingYes
	} else {
		voting = votingNo
	}

	return VotingRound[Hash, Number, Signature, ID, E]{
		votes:             votes,
		voting:            voting,
		incoming:          newWakerChan(roundData.Incoming),
		outgoing:          newBuffered(outgoing),
		state:             Start[Timer]{roundData.PrevoteTimer, roundData.PrecommitTimer},
		bridgedRoundState: nil,
		primaryBlock:      nil,
		bestFinalized:     nil,
		env:               env,
		lastRoundState:    lastRoundState,
		finalizedSender:   finalizedSender,
	}
}

// Create a voting round from a completed `Round`. We will not vote further
// in this round.
func NewVotingRoundCompleted[
	Hash constraints.Ordered, Number constraints.Unsigned, Signature comparable, ID constraints.Ordered,
	E Environment[Hash, Number, Signature, ID],
](
	votes *Round[ID, Hash, Number, Signature],
	finalizedSender chan finalizedNotification[Hash, Number, Signature, ID],
	lastRoundState *latterView[Hash, Number],
	env E,
) VotingRound[Hash, Number, Signature, ID, E] {
	outgoing := make(chan Message[Hash, Number])
	roundData := env.RoundData(votes.Number(), outgoing)
	return VotingRound[Hash, Number, Signature, ID, E]{
		votes:             votes,
		voting:            votingNo,
		incoming:          newWakerChan(roundData.Incoming),
		outgoing:          newBuffered(outgoing),
		state:             nil,
		bridgedRoundState: nil,
		primaryBlock:      nil,
		bestFinalized:     nil,
		env:               env,
		lastRoundState:    lastRoundState,
		finalizedSender:   finalizedSender,
	}
}

// Poll the round. When the round is completable and messages have been flushed, it will return `Poll::Ready` but
// can continue to be polled.
func (vr *VotingRound[Hash, Number, Signature, ID, E]) poll(waker *waker) (bool, error) {
	fmt.Printf(
		"Polling round %d, state = %+v, step = %T\n",
		vr.votes.Number(),
		vr.votes.State(),
		vr.state,
	)

	preState := vr.votes.State()
	err := vr.processIncoming(waker)
	if err != nil {
		return true, err
	}

	// we only cast votes when we have access to the previous round state.
	// we might have started this round as a prospect "future" round to
	// check whether the voter is lagging behind the current round.
	var lastRoundState *RoundState[Hash, Number]
	if vr.lastRoundState != nil {
		lrr := vr.lastRoundState.get(waker)
		lastRoundState = &lrr
	}
	if lastRoundState != nil {
		err := vr.primaryPropose(lastRoundState)
		if err != nil {
			return true, err
		}
		err = vr.prevote(waker, lastRoundState)
		if err != nil {
			return true, err
		}
		err = vr.precommit(waker, lastRoundState)
		if err != nil {
			return true, err
		}
	}

	ready, err := vr.outgoing.Poll(waker)
	if !ready {
		return false, nil
	}
	if err != nil {
		return true, err
	}
	err = vr.processIncoming(waker) // in case we got a new message signed locally.
	if err != nil {
		return true, err
	}

	// broadcast finality notifications after attempting to cast votes
	postState := vr.votes.State()
	vr.notify(preState, postState)

	completable := vr.votes.Completable()
	// early exit if the current round is not completable
	if !completable {
		return false, nil
	}

	// make sure that the previous round estimate has been finalized
	var lastRoundEstimateFinalized bool
	switch {
	case lastRoundState != nil && lastRoundState.Estimate != nil && lastRoundState.Finalized != nil:
		// either it was already finalized in the previous round
		finalizedInLastRound := lastRoundState.Estimate.Number <= lastRoundState.Finalized.Number

		// or it must be finalized in the current round
		var finalizedInCurrentRound bool
		if vr.Finalized() != nil {
			finalizedInCurrentRound = lastRoundState.Estimate.Number <= vr.Finalized().Number
		}

		lastRoundEstimateFinalized = finalizedInLastRound || finalizedInCurrentRound
	case lastRoundState == nil:
		// NOTE: when we catch up to a round we complete the round
		// without any last round state. in this case we already started
		// a new round after we caught up so this guard is unneeded.
		lastRoundEstimateFinalized = true
	default:
		lastRoundEstimateFinalized = false
	}

	// the previous round estimate must be finalized
	if !lastRoundEstimateFinalized {
		// TODO: trace!(target: "afg", "Round {} completable but estimate not finalized.", self.round_number());
		vr.logParticipation("TRACE")
		return false, nil
	}

	fmt.Printf(
		"Completed round %d, state = %+v, step = %T\n",
		vr.votes.Number(),
		vr.votes.State(),
		vr.state,
	)

	// TODO: self.log_participation(log::Level::Trace);
	vr.logParticipation("TRACE")
	return true, nil
}

// Inspect the state of this round.
func (vr *VotingRound[Hash, Number, Signature, ID, E]) State() any {
	return vr.state
}

// Get access to the underlying environment.
func (vr *VotingRound[Hash, Number, Signature, ID, E]) Env() E {
	return vr.env
}

// Get the round number.
func (vr *VotingRound[Hash, Number, Signature, ID, E]) RoundNumber() uint64 {
	return vr.votes.Number()
}

// Get the round state.
func (vr *VotingRound[Hash, Number, Signature, ID, E]) RoundState() RoundState[Hash, Number] {
	return vr.votes.State()
}

// Get the base block in the dag.
func (vr *VotingRound[Hash, Number, Signature, ID, E]) DagBase() HashNumber[Hash, Number] {
	return vr.votes.Base()
}

// Get the base block in the dag.
func (vr *VotingRound[Hash, Number, Signature, ID, E]) Voters() VoterSet[ID] {
	return vr.votes.Voters()
}

// Get the best block finalized in this round.
func (vr *VotingRound[Hash, Number, Signature, ID, E]) Finalized() *HashNumber[Hash, Number] {
	return vr.votes.State().Finalized
}

// Get the current total weight of prevotes.
func (vr *VotingRound[Hash, Number, Signature, ID, E]) PrevoteWeight() VoteWeight {
	weight, _ := vr.votes.PrevoteParticipation()
	return weight
}

// Get the current total weight of precommits.
func (vr *VotingRound[Hash, Number, Signature, ID, E]) PrecommitWeight() VoteWeight {
	weight, _ := vr.votes.PrecommitParticipation()
	return weight
}

// Get the current total weight of prevotes.
func (vr *VotingRound[Hash, Number, Signature, ID, E]) PrevoteIDs() []ID {
	var ids []ID
	for _, pv := range vr.votes.Prevotes() {
		ids = append(ids, pv.ID)
	}
	return ids
}

// Get the current total weight of prevotes.
func (vr *VotingRound[Hash, Number, Signature, ID, E]) PrecommitIDs() []ID {
	var ids []ID
	for _, pv := range vr.votes.Precommits() {
		ids = append(ids, pv.ID)
	}
	return ids
}

// Check a commit. If it's valid, import all the votes into the round as well.
// Returns the finalized base if it checks out.
func (vr *VotingRound[Hash, Number, Signature, ID, E]) CheckAndImportFromCommit(
	commit Commit[Hash, Number, Signature, ID],
) (*HashNumber[Hash, Number], error) {
	cvr, err := ValidateCommit[Hash, Number](commit, vr.Voters(), vr.env)
	if err != nil {
		return nil, err
	}
	if !cvr.Valid() {
		return nil, nil
	}

	for _, signedPrecommit := range commit.Precommits {
		precommit := signedPrecommit.Precommit
		signature := signedPrecommit.Signature
		id := signedPrecommit.ID

		importResult, err := vr.votes.importPrecommit(vr.env, precommit, id, signature)
		if err != nil {
			return nil, err
		}
		if importResult.Equivocation != nil {
			vr.env.PrecommitEquivocation(vr.RoundNumber(), *importResult.Equivocation)
		}
	}

	return &HashNumber[Hash, Number]{commit.TargetHash, commit.TargetNumber}, nil
}

// Get a clone of the finalized sender.
func (vr *VotingRound[Hash, Number, Signature, ID, E]) FinalizedSender() chan finalizedNotification[Hash, Number, Signature, ID] { //nolint:lll
	return vr.finalizedSender
}

// call this when we build on top of a given round in order to get a handle
// to updates to the latest round-state.
func (vr *VotingRound[Hash, Number, Signature, ID, E]) BridgeState() *latterView[Hash, Number] {
	priorView, latterView := bridgeState(vr.votes.State())
	if vr.bridgedRoundState != nil {
		// TODO:
		// warn!(target: "afg", "Bridged state from round {} more than once.",
		// 		self.votes.number());
		fmt.Printf("Bridged state from round %d more than once.\n", vr.votes.Number())
	}

	vr.bridgedRoundState = &priorView
	return &latterView
}

// Get a commit justifying the best finalized block.
func (vr *VotingRound[Hash, Number, Signature, ID, E]) FinalizingCommit() *Commit[Hash, Number, Signature, ID] {
	return vr.bestFinalized
}

// Return all votes for the round (prevotes and precommits), sorted by
// imported order and indicating the indices where we voted. At most two
// prevotes and two precommits per voter are present, further equivocations
// are not stored (as they are redundant).
func (vr *VotingRound[Hash, Number, Signature, ID, E]) HistoricalVotes() HistoricalVotes[Hash, Number, Signature, ID] {
	return vr.votes.HistoricalVotes()
}

// Handle a vote manually.
func (vr *VotingRound[Hash, Number, Signature, ID, E]) HandleVote(vote SignedMessage[Hash, Number, Signature, ID]) error { //nolint:lll
	message := vote.Message
	if !vr.env.IsEqualOrDescendantOf(vr.votes.Base().Hash, message.Target().Hash) {
		return nil
	}

	switch message := message.Value().(type) {
	case Prevote[Hash, Number]:
		prevote := message
		importResult, err := vr.votes.importPrevote(vr.env, prevote, vote.ID, vote.Signature)
		if err != nil {
			return err
		}
		if importResult.Equivocation != nil {
			vr.env.PrevoteEquivocation(vr.votes.Number(), *importResult.Equivocation)
		}
	case Precommit[Hash, Number]:
		precommit := message
		importResult, err := vr.votes.importPrecommit(vr.env, precommit, vote.ID, vote.Signature)
		if err != nil {
			return err
		}
		if importResult.Equivocation != nil {
			vr.env.PrecommitEquivocation(vr.votes.Number(), *importResult.Equivocation)
		}
	case PrimaryPropose[Hash, Number]:
		primary := message
		primaryID, _ := vr.votes.PrimaryVoter()
		// note that id here refers to the party which has cast the vote
		// and not the id of the party which has received the vote message.
		if vote.ID == primaryID {
			vr.primaryBlock = &HashNumber[Hash, Number]{primary.TargetHash, primary.TargetNumber}
		}
	}

	return nil
}

func (vr *VotingRound[Hash, Number, Signature, ID, E]) logParticipation(level any) {
	totalWeight := vr.Voters().TotalWeight()
	threshold := vr.Voters().Threshold()
	nVoters := vr.Voters().Len()
	number := vr.RoundNumber()

	prevoteWeight, nPrevotes := vr.votes.PrevoteParticipation()
	precommitWeight, nPrecommits := vr.votes.PrecommitParticipation()

	// TODO: log::log!(target: "afg", log_level, "Round {}: prevotes: {}/{}/{} weight, {}/{} actual",
	// number, prevote_weight, threshold, total_weight, n_prevotes, n_voters);
	fmt.Printf("%s: Round %d: prevotes: %d/%d/%d weight, %d/%d actual\n",
		level, number, prevoteWeight, threshold, totalWeight, nPrevotes, nVoters)

	// TODO: log::log!(target: "afg", log_level, "Round {}: precommits: {}/{}/{} weight, {}/{} actual",
	// number, precommit_weight, threshold, total_weight, n_precommits, n_voters);
	fmt.Printf("%s: Round %d: precommits: %d/%d/%d weight, %d/%d actual\n",
		level, number, precommitWeight, threshold, totalWeight, nPrecommits, nVoters)
}

func (vr *VotingRound[Hash, Number, Signature, ID, E]) processIncoming(waker *waker) error {
	vr.incoming.setWaker(waker)
	var (
		msgCount  = 0
		timer     *time.Timer
		timerChan <-chan time.Time
	)
while:
	for {
		select {
		case incoming := <-vr.incoming.channel():
			fmt.Printf("Round %d: Got incoming message\n", vr.RoundNumber())
			if timer != nil {
				timer.Stop()
				timer = nil
			}
			if incoming.Error != nil {
				return incoming.Error
			}
			err := vr.HandleVote(incoming.SignedMessage)
			if err != nil {
				return err
			}
			msgCount++
		case <-timerChan:
			if msgCount > 0 {
				fmt.Println("processed", msgCount, "messages")
			}
			break while
		default:
			if timer == nil {
				// delay 1ms before exiting this loop
				timer = time.NewTimer(1 * time.Millisecond)
				timerChan = timer.C
			}
		}
	}
	return nil
}

func (vr *VotingRound[Hash, Number, Signature, ID, E]) primaryPropose(lastRoundState *RoundState[Hash, Number]) error {
	// self.state.take()
	state := vr.state
	vr.state = nil

	if state == nil {
		return nil
	}
	switch state := state.(type) {
	case Start[Timer]:
		prevoteTimer := state[0]
		precommitTimer := state[1]

		maybeEstimate := lastRoundState.Estimate
		switch {
		case maybeEstimate != nil && vr.voting.isPrimary():
			lastRoundEstimate := maybeEstimate
			maybeFinalized := lastRoundState.Finalized

			var shouldSendPrimary = true
			if maybeFinalized != nil {
				shouldSendPrimary = lastRoundEstimate.Number > maybeFinalized.Number
			}
			if shouldSendPrimary {
				fmt.Printf("Sending primary block hint for round %d\n", vr.votes.Number())
				primary := PrimaryPropose[Hash, Number]{
					TargetHash:   lastRoundEstimate.Hash,
					TargetNumber: lastRoundEstimate.Number,
				}
				err := vr.env.Proposed(vr.RoundNumber(), primary)
				if err != nil {
					return err
				}
				message := newMessage(primary)
				vr.outgoing.Push(message)
				vr.state = Proposed[Timer]{prevoteTimer, precommitTimer}

				return nil
			}
			fmt.Printf(
				"Last round estimate has been finalized, not sending primary block hint for round %d\n",
				vr.votes.Number(),
			)

		case maybeEstimate == nil && vr.voting.isPrimary():
			fmt.Printf("Last round estimate does not exist, not sending primary block hint for round %d\n", vr.votes.Number())
		default:
		}

		vr.state = Start[Timer]{prevoteTimer, precommitTimer}
	default:
		vr.state = state
	}
	return nil
}

func (vr *VotingRound[Hash, Number, Signature, ID, E]) prevote(w *waker, lastRoundState *RoundState[Hash, Number]) error { //nolint:lll
	state := vr.state
	vr.state = nil

	var startPrevoting = func(prevoteTimer Timer, precommitTimer Timer, proposed bool, waker *waker) error {
		prevoteTimer.SetWaker(waker)
		var shouldPrevote bool
		elapsed, err := prevoteTimer.Elapsed()
		if elapsed {
			if err != nil {
				return err
			}
			shouldPrevote = true
		} else {
			shouldPrevote = vr.votes.Completable()
		}

		if shouldPrevote {
			if vr.voting.isActive() {
				fmt.Println("Constructing prevote for round", vr.votes.Number())

				base, bestChain := vr.constructPrevote(lastRoundState)

				// since we haven't polled the future above yet we need to
				// manually schedule the current task to be awoken so the
				// `best_chain` future is then polled below after we switch the
				// state to `Prevoting`.
				waker.wake()

				vr.state = Prevoting[Timer, hashBestChain[Hash, Number]]{
					precommitTimer, hashBestChain[Hash, Number]{base, bestChain},
				}
			} else {
				vr.state = Prevoted[Timer]{precommitTimer}
			}
		} else if proposed {
			vr.state = Proposed[Timer]{prevoteTimer, precommitTimer}
		} else {
			vr.state = Start[Timer]{prevoteTimer, precommitTimer}
		}

		return nil
	}

	var finishPrevoting = func(precommitTimer Timer, base Hash, bestChain BestChain[Hash, Number], waker *waker) error {
		wakerChan := newWakerChan(bestChain)
		wakerChan.setWaker(waker)
		var best *HashNumber[Hash, Number]
		res := <-wakerChan.channel()
		switch {
		case res.Error != nil:
			return res.Error
		case res.Value != nil:
			best = res.Value
		default:
			vr.state = Prevoting[Timer, hashBestChain[Hash, Number]]{
				precommitTimer, hashBestChain[Hash, Number]{base, bestChain},
			}
			return nil
		}

		if best != nil {
			prevote := Prevote[Hash, Number]{best.Hash, best.Number}

			// TODO: debug!(target: "afg", "Casting prevote for round {}", this.votes.number());
			fmt.Println("Casting prevote for round {}", vr.votes.Number())
			err := vr.env.Prevoted(vr.RoundNumber(), prevote)
			if err != nil {
				return err
			}
			vr.votes.SetPrevotedIdx()
			message := newMessage(prevote)
			vr.outgoing.Push(message)
			vr.state = Prevoted[Timer]{precommitTimer}
		} else {
			// if this block is considered unknown, something has gone wrong.
			// log and handle, but skip casting a vote.
			// TODO: warn!(target: "afg",
			// 	"Could not cast prevote: previously known block {:?} has disappeared",
			// 	base,
			// );
			fmt.Printf("Could not cast prevote: previously known block %v has disappeared\n", base)

			// when we can't construct a prevote, we shouldn't precommit.
			vr.state = nil
			vr.voting = votingNo
		}

		return nil
	}

	if state == nil {
		return nil
	}
	switch state := state.(type) {
	case Start[Timer]:
		return startPrevoting(state[0], state[1], false, w)
	case Proposed[Timer]:
		return startPrevoting(state[0], state[1], true, w)
	case Prevoting[Timer, hashBestChain[Hash, Number]]:
		return finishPrevoting(state.T, state.W.Hash, state.W.BestChain, w)
	default:
		vr.state = state
	}

	return nil
}

func (vr *VotingRound[Hash, Number, Signature, ID, E]) precommit(waker *waker, lastRoundState *RoundState[Hash, Number]) error { //nolint:lll
	state := vr.state
	vr.state = nil
	if state == nil {
		return nil
	}
	switch state := state.(type) {
	case Prevoted[Timer]:
		precommitTimer := state[0]
		precommitTimer.SetWaker(waker)
		lastRoundEstimate := lastRoundState.Estimate
		if lastRoundEstimate == nil {
			panic("Rounds only started when prior round completable; qed")
		}

		var shouldPrecommit bool
		var ls bool
		st := vr.votes.State()
		pg := st.PrevoteGHOST
		if pg != nil {
			ls = *pg == *lastRoundEstimate || vr.env.IsEqualOrDescendantOf(lastRoundEstimate.Hash, pg.Hash)
		}
		var rs bool
		elapsed, err := precommitTimer.Elapsed()
		if elapsed {
			if err != nil {
				return err
			} else {
				rs = true
			}
		} else {
			rs = vr.votes.Completable()
		}
		shouldPrecommit = ls && rs

		if shouldPrecommit {
			if vr.voting.isActive() {
				// TODO: debug!(target: "afg", "Casting precommit for round {}", self.votes.number());
				fmt.Println("Casting precommit for round {}", vr.votes.Number())
				precommit := vr.constructPrecommit()
				err := vr.env.Precommitted(vr.RoundNumber(), precommit)
				if err != nil {
					return err
				}
				vr.votes.SetPrecommittedIdx()
				message := newMessage(precommit)
				vr.outgoing.Push(message)
			}
			vr.state = Precommitted{}
		} else {
			vr.state = Prevoted[Timer]{precommitTimer}
		}
	default:
		vr.state = state
	}

	return nil
}

// construct a prevote message based on local state.
func (vr *VotingRound[Hash, Number, Signature, ID, E]) constructPrevote(lastRoundState *RoundState[Hash, Number]) (h Hash, bc BestChain[Hash, Number]) { //nolint:lll
	lastRoundEstimate := lastRoundState.Estimate
	if lastRoundEstimate == nil {
		panic("Rounds only started when prior round completable; qed")
	}

	var findDescendentOf Hash
	switch primaryBlock := vr.primaryBlock; primaryBlock {
	case nil:
		// vote for best chain containing prior round-estimate.
		findDescendentOf = lastRoundEstimate.Hash
	default:
		// we will vote for the best chain containing `p_hash` iff
		// the last round's prevote-GHOST included that block and
		// that block is a strict descendent of the last round-estimate that we are
		// aware of.
		lastPrevoteG := lastRoundState.PrevoteGHOST
		if lastPrevoteG == nil {
			panic("Rounds only started when prior round completable; qed")
		}

		// if the blocks are equal, we don't check ancestry.
		if *primaryBlock == *lastPrevoteG {
			findDescendentOf = primaryBlock.Hash
		} else if primaryBlock.Hash >= lastPrevoteG.Hash {
			findDescendentOf = lastRoundEstimate.Hash
		} else {
			// from this point onwards, the number of the primary-broadcasted
			// block is less than the last prevote-GHOST's number.
			// if the primary block is in the ancestry of p-G we vote for the
			// best chain containing it.
			pHash := primaryBlock.Hash
			pNum := primaryBlock.Number
			ancestry, err := vr.env.Ancestry(lastRoundEstimate.Hash, lastPrevoteG.Hash)
			if err != nil {
				// This is only possible in case of massive equivocation
				// TODO: check for error type Err(crate::Error::NotDescendent)
				// TODO: warn!(target: "afg",
				// 	"Possible case of massive equivocation:
				// 	last round prevote GHOST: {:?} is not a descendant of last round estimate: {:?}",
				// 	last_prevote_g,
				// 	last_round_estimate,
				// );
				fmt.Printf(
					"Possible case of massive equivocation: last round prevote GHOST: %v"+
						"is not a descendant of last round estimate: %v\n",
					lastPrevoteG,
					lastRoundEstimate,
				)

				findDescendentOf = lastRoundEstimate.Hash
			} else {
				toSub := pNum + 1

				var offset uint
				if lastPrevoteG.Number < toSub {
					offset = 0
				} else {
					offset = uint(lastPrevoteG.Number - toSub)
				}

				if offset >= uint(len(ancestry)) {
					findDescendentOf = lastRoundEstimate.Hash
				} else {
					if ancestry[offset] == pHash {
						findDescendentOf = pHash
					} else {
						findDescendentOf = lastRoundEstimate.Hash
					}
				}
			}
		}
	}

	return findDescendentOf, vr.env.BestChainContaining(findDescendentOf)
}

// construct a precommit message based on local state.
func (vr *VotingRound[Hash, Number, Signature, ID, E]) constructPrecommit() Precommit[Hash, Number] {
	var t HashNumber[Hash, Number]
	switch target := vr.votes.State().PrevoteGHOST; target {
	case nil:
		t = vr.votes.Base()
	default:
		t = *target
	}
	return Precommit[Hash, Number]{t.Hash, t.Number}
}

// notify when new blocks are finalized or when the round-estimate is updated
func (vr *VotingRound[Hash, Number, Signature, ID, E]) notify(
	lastState RoundState[Hash, Number],
	newState RoundState[Hash, Number],
) {
	// `RoundState` attributes have pointers to values so comparison here is on pointer address.
	// It's assumed that the `Round` attributes will use a new address for new values.
	// Given the caller of this function, we know that new values will use new addresses
	// so no need for deep value comparison.
	if lastState != newState {
		if vr.bridgedRoundState != nil {
			vr.bridgedRoundState.update(newState)
		}
	}

	// send notification only when the round is completable and we've cast votes.
	// this is a workaround that ensures when we re-instantiate the voter after
	// a shutdown, we never re-create the same round with a base that was finalized
	// in this round or after.
	// we try to notify if either the round state changed or if we haven't
	// sent any notification yet (this is to guard against seeing enough
	// votes to finalize before having precommited)
	stateChanged := lastState.Finalized != newState.Finalized
	sentFinalityNotifications := vr.bestFinalized != nil

	if newState.Completable && (stateChanged || !sentFinalityNotifications) {
		_, precommited := vr.state.(Precommitted)
		// we only cast votes when we have access to the previous round state,
		// which won't be the case whenever we catch up to a later round.
		cantVote := vr.lastRoundState == nil

		if precommited || cantVote {
			if newState.Finalized != nil {
				precommits := vr.votes.FinalizingPrecommits(vr.env)
				if precommits == nil {
					panic("always returns none if something was finalized; this is checked above; qed")
				}
				commit := Commit[Hash, Number, Signature, ID]{
					TargetHash:   newState.Finalized.Hash,
					TargetNumber: newState.Finalized.Number,
					Precommits:   *precommits,
				}
				vr.finalizedSender <- finalizedNotification[Hash, Number, Signature, ID]{
					Hash:   newState.Finalized.Hash,
					Number: newState.Finalized.Number,
					Round:  vr.votes.Number(),
					Commit: commit,
				}
				vr.bestFinalized = &commit
			}
		}
	}

}

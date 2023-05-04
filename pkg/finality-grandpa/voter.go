// Copyright 2023 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package grandpa

import (
	"fmt"
	"sync"

	"github.com/tidwall/btree"
	"golang.org/x/exp/constraints"
)

type wakerChan[Item any] struct {
	in    chan Item
	out   chan Item
	waker *Waker
}

func newWakerChan[Item any](in chan Item) *wakerChan[Item] {
	wc := &wakerChan[Item]{
		in:    in,
		out:   make(chan Item),
		waker: nil,
	}
	go wc.start()
	return wc
}

func (wc *wakerChan[Item]) start() {
	for item := range wc.in {
		if wc.waker != nil {
			wc.waker.Wake()
		}
		wc.out <- item
	}
}

func (wc *wakerChan[Item]) SetWaker(waker *Waker) {
	wc.waker = waker
}

// Chan returns a channel to consume `Item`.  Not thread safe, only supports one consumer
func (wc *wakerChan[Item]) Chan() chan Item {
	return wc.out
}

type Timer interface {
	SetWaker(waker *Waker)
	Elapsed() (bool, error)
}

type Output[Hash comparable, Number constraints.Unsigned] chan Message[Hash, Number]

// The input stream used to communicate with the outside world.
type Input[Hash comparable, Number constraints.Unsigned, Signature comparable, ID constraints.Ordered] chan SignedMessageError[Hash, Number, Signature, ID]

// func newInput[Hash comparable, Number constraints.Unsigned, Signature comparable, ID constraints.Ordered](
// 	in chan SignedMessageError[Hash, Number, Signature, ID],
// ) *Input[Hash, Number, Signature, ID] {
// 	wc := newWakerChan(in)
// 	input := Input[Hash, Number, Signature, ID]{wc}
// 	return &input
// }

// func (input *Input[Hash, Number, Signature, ID]) SetWaker(waker *Waker) {
// 	input.wakerChan.SetWaker(waker)
// }

// // Chan returns a channel to consume SignedMessageError.  Not thread safe, only supports one consumer
// func (input *Input[Hash, Number, Signature, ID]) Chan() chan SignedMessageError[Hash, Number, Signature, ID] {
// 	return input.wakerChan.Chan()
// }

type SignedMessageError[Hash comparable, Number constraints.Unsigned, Signature comparable, ID constraints.Ordered] struct {
	SignedMessage SignedMessage[Hash, Number, Signature, ID]
	Error         error
}
type BestChainOutput[Hash comparable, Number constraints.Unsigned] struct {
	Value *HashNumber[Hash, Number]
	Error error
}

// Associated future type for the environment used when asynchronously computing the
// best chain to vote on. See also [`Self::best_chain_containing`].
type BestChain[Hash comparable, Number constraints.Unsigned] chan BestChainOutput[Hash, Number]

// func NewBestChain[Hash comparable, Number constraints.Unsigned](in chan BestChainOutput[Hash, Number]) *BestChain[Hash, Number] {
// 	wc := newWakerChan(in)
// 	bestChain := BestChain[Hash, Number]{wc}
// 	return &bestChain
// }

// func (bc *BestChain[Hash, Number]) SetWaker(waker *Waker) {
// 	bc.wakerChan.SetWaker(waker)
// }

// // Chan returns a channel to consume SignedMessageError.  Not thread safe, only supports one consumer
// func (bc *BestChain[Hash, Number]) Chan() chan BestChainOutput[Hash, Number] {
// 	return bc.wakerChan.Chan()
// }

// Necessary environment for a voter.
//
// This encapsulates the database and networking layers of the chain.
type Environment[Hash comparable, Number constraints.Unsigned, Signature comparable, ID constraints.Ordered] interface {
	Chain[Hash, Number]
	// Return a future that will resolve to the hash of the best block whose chain
	// contains the given block hash, even if that block is `base` itself.
	//
	// If `base` is unknown the future outputs `None`.
	BestChainContaining(base Hash) BestChain[Hash, Number]

	// Produce data necessary to start a round of voting. This may also be called
	// with the round number of the most recently completed round, in which case
	// it should yield a valid input stream.
	//
	// The input stream should provide messages which correspond to known blocks
	// only.
	//
	// The voting logic will push unsigned messages over-eagerly into the
	// output stream. It is the job of this stream to determine if those messages
	// should be sent (for example, if the process actually controls a permissioned key)
	// and then to sign the message, multicast it to peers, and schedule it to be
	// returned by the `In` stream.
	//
	// This allows the voting logic to maintain the invariant that only incoming messages
	// may alter the state, and the logic remains the same regardless of whether a node
	// is a regular voter, the proposer, or simply an observer.
	//
	// Furthermore, this means that actual logic of creating and verifying
	// signatures is flexible and can be maintained outside this crate.
	RoundData(round uint64, outgoing chan Message[Hash, Number]) RoundData[ID, Timer, SignedMessageError[Hash, Number, Signature, ID]]

	// Return a timer that will be used to delay the broadcast of a commit
	// message. This delay should not be static to minimize the amount of
	// commit messages that are sent (e.g. random value in [0, 1] seconds).
	RoundCommitTimer() Timer

	// Note that we've done a primary proposal in the given round.
	Proposed(round uint64, propose PrimaryPropose[Hash, Number]) error

	// Note that we have prevoted in the given round.
	Prevoted(round uint64, prevote Prevote[Hash, Number]) error

	// Note that we have precommitted in the given round.
	Precommitted(round uint64, precommit Precommit[Hash, Number]) error

	// Note that a round is completed. This is called when a round has been
	// voted in and the next round can start. The round may continue to be run
	// in the background until _concluded_.
	// Should return an error when something fatal occurs.
	Completed(
		round uint64,
		state RoundState[Hash, Number],
		base HashNumber[Hash, Number],
		votes HistoricalVotes[Hash, Number, Signature, ID],
	) error

	// Note that a round has concluded. This is called when a round has been
	// `completed` and additionally, the round's estimate has been finalized.
	//
	// There may be more votes than when `completed`, and it is the responsibility
	// of the `Environment` implementation to deduplicate. However, the caller guarantees
	// that the votes passed to `completed` for this round are a prefix of the votes passed here.
	Concluded(
		round uint64,
		state RoundState[Hash, Number],
		base HashNumber[Hash, Number],
		votes HistoricalVotes[Hash, Number, Signature, ID],
	) error

	// Called when a block should be finalized.
	// TODO: make this a future that resolves when it's e.g. written to disk?
	FinalizeBlock(
		hash Hash,
		number Number,
		round uint64,
		commit Commit[Hash, Number, Signature, ID],
	) error

	// Note that an equivocation in prevotes has occurred.
	PrevoteEquivocation(
		round uint64,
		equivocation Equivocation[ID, Prevote[Hash, Number], Signature],
	)

	// Note that an equivocation in prevotes has occurred.
	PrecommitEquivocation(
		round uint64,
		equivocation Equivocation[ID, Precommit[Hash, Number], Signature],
	)
}

type finalizedNotification[Hash, Number, Signature, ID any] struct {
	Hash   Hash
	Number Number
	Round  uint64
	Commit Commit[Hash, Number, Signature, ID]
}

// Data necessary to participate in a round.
type RoundData[ID, Timer, SignedMessageError any] struct {
	// Local voter id (if any.)
	VoterID *ID
	// Timer before prevotes can be cast. This should be Start + 2T
	// where T is the gossip time estimate.
	PrevoteTimer Timer
	// Timer before precommits can be cast. This should be Start + 4T
	PrecommitTimer Timer
	// Incoming messages.
	Incoming chan SignedMessageError
}

type Buffered[I any] struct {
	inner   chan I
	buffer  []I
	mtx     sync.Mutex
	readyCh chan any
}

func newBuffered[I any](inner chan I) *Buffered[I] {
	b := &Buffered[I]{
		inner:   inner,
		readyCh: make(chan any, 1),
	}
	// prime the channel
	b.readyCh <- nil
	return b
}

func (b *Buffered[I]) Push(item I) {
	b.mtx.Lock()
	defer b.mtx.Unlock()
	b.buffer = append(b.buffer, item)
}

func (b *Buffered[I]) Poll(waker *Waker) (bool, error) {
	return b.flush(waker)
}

func (b *Buffered[I]) flush(waker *Waker) (bool, error) {
	if b.inner == nil {
		return false, fmt.Errorf("inner channel has been closed")
	}

	b.mtx.Lock()
	defer b.mtx.Unlock()
	if len(b.buffer) == 0 {
		return true, nil
	}
	select {
	case <-b.readyCh:
		// b.mtx.Lock()
		defer func() {
			b.readyCh <- nil
			// if waker != nil {
			// 	waker.Wake()
			// }
			// b.mtx.Unlock()
		}()

		for len(b.buffer) > 0 {
			b.inner <- b.buffer[0]
			b.buffer = b.buffer[1:]
			if waker != nil {
				waker.Wake()
			}
		}

	default:
	}
	return false, nil
}

func (b *Buffered[I]) Close() {
	b.mtx.Lock()
	defer b.mtx.Unlock()
	close(b.inner)
	b.inner = nil
}

// Instantiates the given last round, to be backgrounded until its estimate is finalized.
//
// This round must be completable based on the passed votes (and if not, `None` will be returned),
// but it may be the case that there are some more votes to propagate in order to push
// the estimate backwards and conclude the round (i.e. finalize its estimate).
//
// may only be called with non-zero last round.
func instantiateLastRound[Hash constraints.Ordered, Number constraints.Unsigned, Signature comparable, ID constraints.Ordered, E Environment[Hash, Number, Signature, ID]](
	voters VoterSet[ID],
	lastRoundVotes []SignedMessage[Hash, Number, Signature, ID],
	lastRoundNumber uint64,
	lastRoundBase HashNumber[Hash, Number],
	finalizedSender chan finalizedNotification[Hash, Number, Signature, ID],
	env E,
) *VotingRound[Hash, Number, Signature, ID, E] {
	lastRoundTracker := NewRound[ID, Hash, Number, Signature](RoundParams[ID, Hash, Number]{
		Voters:      voters,
		Base:        HashNumber[Hash, Number](lastRoundBase),
		RoundNumber: lastRoundNumber,
	})

	// start as completed so we don't cast votes.
	lastRound := NewVotingRoundCompleted(lastRoundTracker, finalizedSender, nil, env)

	for _, vote := range lastRoundVotes {
		// bail if any votes are bad.
		err := lastRound.HandleVote(vote)
		if err != nil {
			fmt.Printf("lastRound.Handlevote error: %v\n", err)
			return nil
		}
	}

	if lastRound.RoundState().Completable {
		return &lastRound
	} else {
		return nil
	}
}

// The inner state of a voter aggregating the currently running round state
// (i.e. best and background rounds). This state exists separately since it's
// useful to wrap in a `Arc<Mutex<_>>` for sharing.
type innerVoterState[Hash constraints.Ordered, Number constraints.Unsigned, Signature comparable, ID constraints.Ordered, E Environment[Hash, Number, Signature, ID]] struct {
	bestRound  VotingRound[Hash, Number, Signature, ID, E]
	pastRounds PastRounds[Hash, Number, Signature, ID, E]
	sync.Mutex
}

// Communication between nodes that is not round-localized.
type CommunicationOut struct {
	variant any
}

type CommuincationOutVariants[Hash constraints.Ordered, Number constraints.Unsigned, Signature comparable, ID constraints.Ordered] interface {
	CommunicationOutCommit[Hash, Number, Signature, ID]
}

func setCommunicationOut[Hash constraints.Ordered, Number constraints.Unsigned, Signature comparable, ID constraints.Ordered, T CommuincationOutVariants[Hash, Number, Signature, ID]](co *CommunicationOut, variant T) {
	co.variant = variant
}

type CommunicationOutCommit[Hash constraints.Ordered, Number constraints.Unsigned, Signature comparable, ID constraints.Ordered] numberCommit[Hash, Number, Signature, ID]

//  {
// 	Number uint64
// 	Commit Commit[Hash, Number, Signature, ID]
// }

type CommitProcessingOutcome struct {
	variant any
}

// It was beneficial to process this commit.
type CommitProcessingOutcomeGood GoodCommit

// It wasn't beneficial to process this commit. We wasted resources.
type CommitProcessingOutcomeBad BadCommit

// The result of processing for a good commit.
type GoodCommit struct{}

// The result of processing for a bad commit
type BadCommit struct {
	numPrecommits           uint
	numDuplicatedPrecommits uint
	numEquivocations        uint
	numInvalidVoters        uint
}

func NewBadCommit(cvr CommitValidationResult) BadCommit {
	return BadCommit{
		numPrecommits:           cvr.NumPrecommits(),
		numDuplicatedPrecommits: cvr.NumDuplicatedPrecommits(),
		numEquivocations:        cvr.NumEquiovcations(),
		numInvalidVoters:        cvr.NumInvalidVoters(),
	}
}

// The outcome of processing a catch up.
type CatchUpProcessingOutcome struct {
	variant any
}

func setCatchUpProcessingOutcome[T CatchUpProcessingOutcomes](cpo *CatchUpProcessingOutcome, variant T) {
	cpo.variant = variant
}

type CatchUpProcessingOutcomes interface {
	CatchUpProcessingOutcomeGood | CatchUpProcessingOutcomeBad | CatchUpProcessingOutcomeUseless
}

// It was beneficial to process this catch up.
type CatchUpProcessingOutcomeGood GoodCatchUp

// It wasn't beneficial to process this catch up, it is invalid and we
// wasted resources.
type CatchUpProcessingOutcomeBad BadCatchUp

// The catch up wasn't processed because it is useless, e.g. it is for a
// round lower than we're currently in.
type CatchUpProcessingOutcomeUseless struct{}

// The result of processing for a good catch up.
type GoodCatchUp struct{}

// The result of processing for a bad catch up.
type BadCatchUp struct{}

type CommunicationIn struct {
	variant any
}

func setCommunicationIn[
	Hash constraints.Ordered, Number constraints.Unsigned, Signature comparable, ID constraints.Ordered,
	T CommunicationInVariants[Hash, Number, Signature, ID],
](ci *CommunicationIn, variant T) {
	ci.variant = variant
}

type CommunicationInVariants[Hash constraints.Ordered, Number constraints.Unsigned, Signature comparable, ID constraints.Ordered] interface {
	CommunicationInCommit[Hash, Number, Signature, ID] | CommunicationInCatchUp[Hash, Number, Signature, ID]
}
type CommunicationInCommit[Hash constraints.Ordered, Number constraints.Unsigned, Signature comparable, ID constraints.Ordered] struct {
	Number        uint64
	CompactCommit CompactCommit[Hash, Number, Signature, ID]
	Callback      func(CommitProcessingOutcome)
}

type CommunicationInCatchUp[Hash constraints.Ordered, Number constraints.Unsigned, Signature comparable, ID constraints.Ordered] struct {
	CatchUp  CatchUp[Hash, Number, Signature, ID]
	Callback func(CatchUpProcessingOutcome)
}

type globalInItem struct {
	CommunicationIn
	Error error
}

// A future that maintains and multiplexes between different rounds,
// and caches votes.
//
// This voter also implements the commit protocol.
// The commit protocol allows a node to broadcast a message that finalizes a
// given block and includes a set of precommits as proof.
//
// - When a round is completable and we precommitted we start a commit timer
// and start accepting commit messages;
// - When we receive a commit message if it targets a block higher than what
// we've finalized we validate it and import its precommits if valid;
// - When our commit timer triggers we check if we've received any commit
// message for a block equal to what we've finalized, if we haven't then we
// broadcast a commit.
//
// Additionally, we also listen to commit messages from rounds that aren't
// currently running, we validate the commit and dispatch a finalization
// notification (if any) to the environment.
type Voter[Hash constraints.Ordered, Number constraints.Unsigned, Signature comparable, ID constraints.Ordered] struct {
	env                    Environment[Hash, Number, Signature, ID]
	voters                 VoterSet[ID]
	inner                  *innerVoterState[Hash, Number, Signature, ID, Environment[Hash, Number, Signature, ID]]
	finalizedNotifications chan finalizedNotification[Hash, Number, Signature, ID]
	lastFinalizedNumber    Number
	globalIn               chan globalInItem
	globalOut              *Buffered[CommunicationOut]
	// the commit protocol might finalize further than the current round (if we're
	// behind), we keep track of last finalized in round so we don't violate any
	// assumptions from round-to-round.
	lastFinalizedInRounds HashNumber[Hash, Number]

	stopChan chan any
}

// Create new `Voter` tracker with given round number and base block.
//
// Provide data about the last completed round. If there is no
// known last completed round, the genesis state (round number 0, no votes, genesis base),
// should be provided. When available, all messages required to complete
// the last round should be provided.
//
// The input stream for commit messages should provide commits which
// correspond to known blocks only (including all its precommits). It
// is also responsible for validating the signature data in commit
// messages.
func NewVoter[Hash constraints.Ordered, Number constraints.Unsigned, Signature comparable, ID constraints.Ordered](
	env Environment[Hash, Number, Signature, ID],
	voters VoterSet[ID],
	globalIn chan globalInItem,
	lastRoundNumber uint64,
	lastRoundVotes []SignedMessage[Hash, Number, Signature, ID],
	lastRoundBase HashNumber[Hash, Number],
	lastFinalized HashNumber[Hash, Number],
) (*Voter[Hash, Number, Signature, ID], chan CommunicationOut) {
	finalizedSender := make(chan finalizedNotification[Hash, Number, Signature, ID], 1)
	finalizedNotifications := finalizedSender
	lastFinalizedNumber := lastFinalized.Number

	pastRounds := NewPastRounds[Hash, Number, Signature, ID, Environment[Hash, Number, Signature, ID]]()
	_, lastRoundState := BridgeState(NewRoundState(lastRoundBase))

	if lastRoundNumber > 0 {
		maybeCompletedLastRound := instantiateLastRound(voters, lastRoundVotes, lastRoundNumber, lastRoundBase, finalizedSender, env)

		if maybeCompletedLastRound != nil {
			lastRound := *maybeCompletedLastRound
			lastRoundState = *lastRound.BridgeState()
			pastRounds.Push(env, lastRound)
		}

		// when there is no information about the last completed round,
		// the best we can do is assume that the estimate == the given base
		// and that it is finalized. This is always the case for the genesis
		// round of a set.
	}

	bestRound := NewVotingRound(
		lastRoundNumber+1,
		voters,
		lastFinalized,
		&lastRoundState,
		finalizedSender,
		env,
	)

	inner := &innerVoterState[Hash, Number, Signature, ID, Environment[Hash, Number, Signature, ID]]{
		bestRound:  bestRound,
		pastRounds: *pastRounds,
	}
	globalOut := make(chan CommunicationOut)
	return &Voter[Hash, Number, Signature, ID]{
		env:                    env,
		voters:                 voters,
		inner:                  inner,
		finalizedNotifications: finalizedNotifications,
		lastFinalizedNumber:    lastFinalizedNumber,
		lastFinalizedInRounds:  lastFinalized,
		globalIn:               globalIn,
		globalOut:              newBuffered(globalOut),
		stopChan:               make(chan any),
	}, globalOut
}

func (v *Voter[Hash, Number, Signature, ID]) pruneBackgroundRounds(waker *Waker) error {
	v.inner.Lock()
	defer v.inner.Unlock()

pastRounds:
	for {
		// Do work on all background rounds, broadcasting any commits generated.
		ready, nc, err := v.inner.pastRounds.pollNext(waker)
		switch ready {
		case true:
			if err != nil {
				return err
			}
			if nc != nil {
				co := CommunicationOut{}
				setCommunicationOut(&co, CommunicationOutCommit[Hash, Number, Signature, ID]{nc.Number, nc.Commit})
				v.globalOut.Push(co)
			} else {
				break pastRounds
			}
		case false:
			break pastRounds
		}
	}

finalizedNotifications:
	for {
		select {
		case notif := <-v.finalizedNotifications:
			fHash := notif.Hash
			fNum := notif.Number
			round := notif.Round
			commit := notif.Commit

			v.inner.pastRounds.UpdateFinalized(fNum)
			if v.setLastFinalizedNumber(fNum) {
				err := v.env.FinalizeBlock(fHash, fNum, round, commit)
				if err != nil {
					return err
				}
			}

			if fNum > v.lastFinalizedInRounds.Number {
				v.lastFinalizedInRounds = HashNumber[Hash, Number]{fHash, fNum}
			}
		default:
			// TODO: wake when messages come back
			break finalizedNotifications
		}
	}

	return nil
}

// Process all incoming messages from other nodes.
//
// Commit messages are handled with extra care. If a commit message references
// a currently backgrounded round, we send it to that round so that when we commit
// on that round, our commit message will be informed by those that we've seen.
//
// Otherwise, we will simply handle the commit and issue a finalization command
// to the environment.
func (v *Voter[Hash, Number, Signature, ID]) processIncoming(waker *Waker) error {
	// TODO: implement waker chans
loop:
	for {
		select {
		case item := <-v.globalIn:
			if item.Error != nil {
				return item.Error
			}
			switch variant := item.CommunicationIn.variant.(type) {
			case CommunicationInCommit[Hash, Number, Signature, ID]:
				roundNumber := variant.Number
				compactCommit := variant.CompactCommit
				processCommitOutcome := variant.Callback

				fmt.Printf("Got commit for round_number %+v: target_number: %+v, target_hash: %+v\n",
					roundNumber,
					compactCommit.TargetNumber,
					compactCommit.TargetHash,
				)

				commit := compactCommit.Commit()
				v.inner.Lock()

				// if the commit is for a background round dispatch to round committer.
				// that returns Some if there wasn't one.
				if imported := v.inner.pastRounds.ImportCommit(roundNumber, commit); imported != nil {
					// otherwise validate the commit and signal the finalized block from the
					// commit to the environment (if valid and higher than current finalized)
					validationResult, err := ValidateCommit(commit, v.voters, v.env.(Chain[Hash, Number]))
					if err != nil {
						return err
					}
					if validationResult.Valid() {
						lastFinalizedNumber := v.lastFinalizedNumber

						// clean up any background rounds
						v.inner.pastRounds.UpdateFinalized(imported.TargetNumber)

						if imported.TargetNumber > lastFinalizedNumber {
							v.lastFinalizedNumber = imported.TargetNumber
							err := v.env.FinalizeBlock(imported.TargetHash, imported.TargetNumber, roundNumber, *imported)
							if err != nil {
								v.inner.Unlock()
								return err
							}
						}

						outcome := CommitProcessingOutcome{CommitProcessingOutcomeGood(GoodCommit{})}
						if processCommitOutcome != nil {
							processCommitOutcome(outcome)
						}
					} else {
						// Failing validation of a commit is bad.
						outcome := CommitProcessingOutcome{CommitProcessingOutcomeBad(NewBadCommit(validationResult))}
						if processCommitOutcome != nil {
							processCommitOutcome(outcome)
						}
					}
				} else {
					// Import to backgrounded round is good.
					outcome := CommitProcessingOutcome{CommitProcessingOutcomeGood(GoodCommit{})}
					if processCommitOutcome != nil {
						processCommitOutcome(outcome)
					}
				}
				v.inner.Unlock()
			case CommunicationInCatchUp[Hash, Number, Signature, ID]:
				catchUp := variant.CatchUp
				processCatchUpOutcome := variant.Callback

				fmt.Printf("Got catch-up message for round %v\n", catchUp.RoundNumber)

				v.inner.Lock()

				round := validateCatchUp(catchUp, v.env, v.voters, v.inner.bestRound.RoundNumber())
				if round == nil {
					if processCatchUpOutcome != nil {
						processCatchUpOutcome(CatchUpProcessingOutcome{CatchUpProcessingOutcomeBad(BadCatchUp{})})
					}
					return nil
				}

				state := round.State()

				// beyond this point, we set this round to the past and
				// start voting in the next round.
				justCompleted := NewVotingRoundCompleted(round, v.inner.bestRound.FinalizedSender(), nil, v.env)

				newBest := NewVotingRound(
					justCompleted.RoundNumber()+1,
					v.voters,
					v.lastFinalizedInRounds,
					justCompleted.BridgeState(),
					v.inner.bestRound.FinalizedSender(),
					v.env,
				)

				// update last-finalized in rounds _after_ starting new round.
				// otherwise the base could be too eagerly set forward.
				if state.Finalized != nil {
					fNum := state.Finalized.Number
					if fNum > v.lastFinalizedInRounds.Number {
						v.lastFinalizedInRounds = HashNumber[Hash, Number](*state.Finalized)
					}
				}

				err := v.env.Completed(
					justCompleted.RoundNumber(),
					RoundState[Hash, Number](justCompleted.RoundState()),
					HashNumber[Hash, Number](justCompleted.DagBase()),
					justCompleted.HistoricalVotes(),
				)
				if err != nil {
					v.inner.Unlock()
					return err
				}

				v.inner.pastRounds.Push(v.env, justCompleted)

				oldBest := v.inner.bestRound
				v.inner.bestRound = newBest
				v.inner.pastRounds.Push(v.env, oldBest)

				if processCatchUpOutcome != nil {
					processCatchUpOutcome(CatchUpProcessingOutcome{CatchUpProcessingOutcomeGood(GoodCatchUp{})})
				}
				v.inner.Unlock()
			}
		default:
			break loop
		}
	}
	return nil
}

// process the logic of the best round.
func (v *Voter[Hash, Number, Signature, ID]) processBestRound(waker *Waker) (bool, error) {
	// If the current `best_round` is completable and we've already precommitted,
	// we start a new round at `best_round + 1`.
	{
		v.inner.Lock()

		var shouldStartNext bool
		completable, err := v.inner.bestRound.poll(waker)
		if err != nil {
			return true, err
		}

		var precomitted bool
		state := v.inner.bestRound.State()
		if state != nil {
			switch v.inner.bestRound.State().(type) {
			case Precommitted:
				precomitted = true
			}
		}

		shouldStartNext = completable && precomitted

		if !shouldStartNext {
			v.inner.Unlock()
			return false, nil
		}

		fmt.Printf("Best round at %v has become completable. Starting new best round at %v\n",
			v.inner.bestRound.RoundNumber(),
			v.inner.bestRound.RoundNumber()+1,
		)
		v.inner.Unlock()
	}

	err := v.completedBestRound()
	if err != nil {
		return true, err
	}

	// round has been updated. so we need to re-poll.
	return v.poll(waker)
}

func (v *Voter[Hash, Number, Signature, ID]) completedBestRound() error {
	v.inner.Lock()
	defer v.inner.Unlock()

	v.env.Completed(
		v.inner.bestRound.RoundNumber(),
		RoundState[Hash, Number](v.inner.bestRound.RoundState()),
		HashNumber[Hash, Number](v.inner.bestRound.DagBase()),
		v.inner.bestRound.HistoricalVotes(),
	)

	oldRoundNumber := v.inner.bestRound.RoundNumber()

	nextRound := NewVotingRound(
		oldRoundNumber+1,
		v.voters,
		v.lastFinalizedInRounds,
		v.inner.bestRound.BridgeState(),
		v.inner.bestRound.FinalizedSender(),
		v.env,
	)

	oldBest := v.inner.bestRound
	v.inner.bestRound = nextRound
	v.inner.pastRounds.Push(v.env, oldBest)
	return nil
}

func (v *Voter[Hash, Number, Signature, ID]) setLastFinalizedNumber(finalizedNumber Number) bool {
	if finalizedNumber > v.lastFinalizedNumber {
		v.lastFinalizedNumber = finalizedNumber
		return true
	}
	return false
}

func (v *Voter[Hash, Number, Signature, ID]) Start() error {
	waker := NewWaker()
	for {
		ready, err := v.poll(waker)
		if err != nil {
			return err
		}
		if ready {
			return nil
		}
		select {
		case <-waker.Chan():
		case <-v.stopChan:
			return fmt.Errorf("early voter stop")
		}
	}
}

func (v *Voter[Hash, Number, Signature, ID]) Stop() error {
	close(v.stopChan)
	// TODO: waitgroup to ensure Start() and other goroutines all exit
	v.globalOut.Close()
	return nil
}

func (v *Voter[Hash, Number, Signature, ID]) poll(waker *Waker) (bool, error) {
	err := v.processIncoming(waker)
	if err != nil {
		return true, err
	}
	err = v.pruneBackgroundRounds(waker)
	if err != nil {
		return true, err
	}
	ready, err := v.globalOut.Poll(waker)
	if !ready {
		return false, nil
	}
	if err != nil {
		return true, err
	}

	return v.processBestRound(waker)
}

type sharedVoteState[Hash constraints.Ordered, Number constraints.Unsigned, Signature comparable, ID constraints.Ordered, E Environment[Hash, Number, Signature, ID]] struct {
	inner *innerVoterState[Hash, Number, Signature, ID, E]
	mtx   sync.Mutex
}

func (svs *sharedVoteState[Hash, Number, Signature, ID, E]) Get() VoterStateReport[ID] {
	toRoundState := func(votingRound VotingRound[Hash, Number, Signature, ID, E]) (uint64, RoundStateReport[ID]) {
		return votingRound.RoundNumber(), RoundStateReport[ID]{
			TotalWeight:            votingRound.Voters().TotalWeight(),
			ThresholdWeight:        votingRound.Voters().Threshold(),
			PrevoteCurrentWeight:   votingRound.PrevoteWeight(),
			PrevoteIDs:             votingRound.PrevoteIDs(),
			PrecommitCurrentWeight: votingRound.PrecommitWeight(),
			PrecommitIDs:           votingRound.PrecommitIDs(),
		}
	}

	svs.mtx.Lock()
	defer svs.mtx.Unlock()

	bestRoundNum, bestRound := toRoundState(svs.inner.bestRound)
	backgroundRounds := svs.inner.pastRounds.VotingRounds()
	mappedBackgroundRounds := make(map[uint64]RoundStateReport[ID])
	for _, backgroundRound := range backgroundRounds {
		num, round := toRoundState(backgroundRound)
		mappedBackgroundRounds[num] = round
	}
	return VoterStateReport[ID]{
		BackgroundRounds: mappedBackgroundRounds,
		BestRound: struct {
			Number     uint64
			RoundState RoundStateReport[ID]
		}{
			Number:     bestRoundNum,
			RoundState: bestRound,
		},
	}
}

// Returns an object allowing to query the voter state.
func (v *Voter[Hash, Number, Signature, ID]) VoterState() VoterState[ID] {
	return &sharedVoteState[Hash, Number, Signature, ID, Environment[Hash, Number, Signature, ID]]{
		inner: v.inner,
	}
}

// Trait for querying the state of the voter. Used by `Voter` to return a queryable object
// without exposing too many data types.
type VoterState[ID comparable] interface {
	// Returns a plain data type, `report::VoterState`, describing the current state
	// of the voter relevant to the voting process.
	Get() VoterStateReport[ID]
}

// Validate the given catch up and return a completed round with all prevotes
// and precommits from the catch up imported. If the catch up is invalid `None`
// is returned instead.
func validateCatchUp[Hash constraints.Ordered, Number constraints.Unsigned, Signature comparable, ID constraints.Ordered, E Environment[Hash, Number, Signature, ID]](
	catchUp CatchUp[Hash, Number, Signature, ID],
	env E,
	voters VoterSet[ID],
	bestRoundNumber uint64,
) *Round[ID, Hash, Number, Signature] {
	if catchUp.RoundNumber <= bestRoundNumber {
		fmt.Println("Ignoring because best round number is", bestRoundNumber)
		return nil
	}

	type prevotedPrecommitted struct {
		prevoted     bool
		precommitted bool
	}
	// check threshold support in prevotes and precommits.
	{
		mapped := btree.NewMap[ID, prevotedPrecommitted](2)

		for _, prevote := range catchUp.Prevotes {
			if !voters.Contains(prevote.ID) {
				fmt.Println("Ignoring invalid catch up, invalid voter:", prevote.ID)
				return nil
			}

			entry, found := mapped.Get(prevote.ID)
			if !found {
				mapped.Set(prevote.ID, prevotedPrecommitted{true, false})
			} else {
				entry.prevoted = true
				mapped.Set(prevote.ID, entry)
			}
		}

		for _, precommit := range catchUp.Precommits {
			if !voters.Contains(precommit.ID) {
				fmt.Println("Ignoring invalid catch up, invalid voter:", precommit.ID)
				return nil
			}

			entry, found := mapped.Get(precommit.ID)
			if !found {
				mapped.Set(precommit.ID, prevotedPrecommitted{false, true})
			} else {
				entry.precommitted = true
				mapped.Set(precommit.ID, entry)
			}
		}

		var (
			pv VoteWeight
			pc VoteWeight
		)
		mapped.Scan(func(id ID, pp prevotedPrecommitted) bool {
			prevoted := pp.prevoted
			precommitted := pp.precommitted

			if vi := voters.Get(id); vi != nil {
				if prevoted {
					pv = pv + VoteWeight(vi.Weight())
				}

				if precommitted {
					pc = pc + VoteWeight(vi.Weight())
				}
			}
			return true
		})

		threshold := voters.Threshold()
		if pv < VoteWeight(threshold) || pc < VoteWeight(threshold) {
			fmt.Println("Ignoring invalid catch up, missing voter threshold")
			return nil
		}
	}

	round := NewRound[ID, Hash, Number, Signature](RoundParams[ID, Hash, Number]{
		catchUp.RoundNumber, voters, HashNumber[Hash, Number]{catchUp.BaseHash, catchUp.BaseNumber},
	})

	// import prevotes first
	for _, sp := range catchUp.Prevotes {
		_, err := round.importPrevote(env, sp.Prevote, sp.ID, sp.Signature)
		if err != nil {
			fmt.Println("Ignoring invalid catch up, error importing prevote:", err)
			return nil
		}
	}

	// then precommits.
	for _, sp := range catchUp.Precommits {
		_, err := round.importPrecommit(env, Precommit[Hash, Number](sp.Precommit), sp.ID, sp.Signature)
		if err != nil {
			fmt.Println("Ignoring invalid catch up, error importing precommit:", err)
			return nil
		}
	}

	state := round.State()
	if !state.Completable {
		return nil
	}

	return round
}

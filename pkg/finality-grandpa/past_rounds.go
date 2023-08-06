// Copyright 2023 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package grandpa

import (
	"fmt"

	"golang.org/x/exp/constraints"
)

// wraps a voting round with a new future that resolves when the round can
// be discarded from the working set.
//
// that point is when the round-estimate is finalized.
type backgroundRound[
	Hash constraints.Ordered, Number constraints.Unsigned, Signature comparable,
	ID constraints.Ordered, E Environment[Hash, Number, Signature, ID],
] struct {
	inner           VotingRound[Hash, Number, Signature, ID, E]
	finalizedNumber Number
	roundCommitter  *roundCommitter[Hash, Number, Signature, ID, E]

	waker *waker
}

func (br *backgroundRound[Hash, Number, Signature, ID, E]) roundNumber() uint64 {
	return br.inner.RoundNumber()
}

func (br *backgroundRound[Hash, Number, Signature, ID, E]) votingRound() VotingRound[Hash, Number, Signature, ID, E] {
	return br.inner
}

func (br *backgroundRound[Hash, Number, Signature, ID, E]) isDone() bool {
	// no need to listen on a round anymore once the estimate is finalized.
	//
	// we map `br.roundCommitter == nil` to true because
	//   - rounds are not backgrounded when incomplete unless we've skipped forward
	//   - if we skipped forward we may never complete this round and we don't need
	//     to keep it forever.
	var ls = br.roundCommitter == nil
	if !ls {
		return false
	}
	var rs = true
	estimate := br.inner.RoundState().Estimate
	if estimate != nil {
		rs = estimate.Number <= br.finalizedNumber
	}
	return ls && rs
}

func (br *backgroundRound[Hash, Number, Signature, ID, E]) updateFinalized(newFinalized Number) {
	switch {
	case br.finalizedNumber >= newFinalized:
	default:
		br.finalizedNumber = newFinalized
	}

	// wake up the future to be polled if done.
	if br.isDone() {
		br.waker.Wake()
	}
}

type concluded uint64
type committed[Hash, Number, Signature, ID any] Commit[Hash, Number, Signature, ID]

type backgroundRoundChange[Hash, Number, Signature, ID any] struct {
	variant any
}

func (brc backgroundRoundChange[Hash, Number, Signature, ID]) Variant() any {
	switch brc.variant.(type) {
	case concluded, committed[Hash, Number, Signature, ID]:
	default:
		panic("unsupported type")
	}
	return brc.variant
}

func (brc *backgroundRoundChange[Hash, Number, Signature, ID]) SetVariant(variant any) {
	switch variant := variant.(type) {
	case concluded:
		setBackgroundRoundChangeVariant(brc, variant)
	case committed[Hash, Number, Signature, ID]:
		setBackgroundRoundChangeVariant(brc, variant)
	default:
		panic("unsupported type")
	}
}

func setBackgroundRoundChangeVariant[
	Hash,
	Number,
	Signature,
	ID any,
	V backgroundRoundChanges[Hash, Number, Signature, ID],
](
	change *backgroundRoundChange[Hash, Number, Signature, ID], variant V,
) {
	change.variant = variant
}

type backgroundRoundChanges[Hash, Number, Signature, ID any] interface {
	concluded | committed[Hash, Number, Signature, ID]
}

func (br *backgroundRound[Hash, Number, Signature, ID, E]) poll(waker *waker) (
	bool,
	backgroundRoundChange[Hash, Number, Signature, ID],
	error,
) {
	br.waker = waker

	_, err := br.inner.poll(waker)
	if err != nil {
		return true, backgroundRoundChange[Hash, Number, Signature, ID]{}, err
	}

	committer := br.roundCommitter
	br.roundCommitter = nil
	switch committer {
	case nil:
	default:
		ready, commit, err := committer.commit(waker, br.inner)
		switch {
		case ready && commit == nil && err == nil:
			br.roundCommitter = nil
		case ready && commit != nil && err == nil:
			change := backgroundRoundChange[Hash, Number, Signature, ID]{}
			change.SetVariant(committed[Hash, Number, Signature, ID](*commit))
			return true, change, nil
		case !ready:
			br.roundCommitter = committer
		default:
			panic("wtf")
		}
	}

	if br.isDone() {
		// if this is fully concluded (has committed _and_ estimate finalized)
		// we bail for real.
		change := backgroundRoundChange[Hash, Number, Signature, ID]{}
		change.SetVariant(concluded(br.roundNumber()))
		return true, change, nil
	}
	return false, backgroundRoundChange[Hash, Number, Signature, ID]{}, nil
}

type roundCommitter[
	Hash constraints.Ordered, Number constraints.Unsigned, Signature comparable,
	ID constraints.Ordered, E Environment[Hash, Number, Signature, ID],
] struct {
	commitTimer   Timer
	importCommits *wakerChan[Commit[Hash, Number, Signature, ID]]
	lastCommit    *Commit[Hash, Number, Signature, ID]

	// pollChan is used to signal back to backgroundRound to poll again
	pollChan chan any
}

func newRoundCommitter[
	Hash constraints.Ordered, Number constraints.Unsigned, Signature comparable,
	ID constraints.Ordered, E Environment[Hash, Number, Signature, ID],
](
	commitTimer Timer,
	commitReceiver *wakerChan[Commit[Hash, Number, Signature, ID]],
) *roundCommitter[Hash, Number, Signature, ID, E] {
	return &roundCommitter[Hash, Number, Signature, ID, E]{
		commitTimer, commitReceiver, nil, make(chan any),
	}
}

func (rc *roundCommitter[Hash, Number, Signature, ID, E]) importCommit(
	votingRound VotingRound[Hash, Number, Signature, ID, E], commit Commit[Hash, Number, Signature, ID],
) (bool, error) {
	// ignore commits for a block lower than we already finalized
	if votingRound.Finalized() != nil && commit.TargetNumber < votingRound.Finalized().Number {
		return true, nil
	}

	base, err := votingRound.CheckAndImportFromCommit(commit)
	if err != nil {
		return false, err
	}
	if base == nil {
		return true, nil
	}

	rc.lastCommit = &commit

	return true, nil
}

func (rc *roundCommitter[Hash, Number, Signature, ID, E]) commit(
	waker *waker,
	votingRound VotingRound[Hash, Number, Signature, ID, E],
) (bool, *Commit[Hash, Number, Signature, ID], error) {
	rc.importCommits.SetWaker(waker)
loop:
	for {
		select {
		case commit, ok := <-rc.importCommits.Chan():
			if !ok {
				panic("TODO: handle channel closure")
			}
			imported, err := rc.importCommit(votingRound, commit)
			if err != nil {
				return true, nil, err
			}
			if !imported {
				// TODO: trace!(target: "afg", "Ignoring invalid commit");
				fmt.Println("Ignoring invalid commit")
			}
		default:
			// TODO: delay a little bit?
			break loop
		}
	}

	rc.commitTimer.SetWaker(waker)
	elapsed, err := rc.commitTimer.Elapsed()
	if elapsed {
		if err != nil {
			return true, nil, err
		}
	} else {
		return false, nil, nil
	}

	lastCommit := rc.lastCommit
	rc.lastCommit = nil
	finalized := votingRound.Finalized()

	switch {
	case lastCommit == nil && finalized != nil:
		return true, votingRound.FinalizingCommit(), nil
	case lastCommit != nil && finalized != nil && lastCommit.TargetNumber < finalized.Number:
		return true, votingRound.FinalizingCommit(), nil
	default:
		return true, nil, nil
	}
}

// A stream for past rounds, which produces any commit messages from those
// rounds and drives them to completion.
type pastRounds[Hash constraints.Ordered, Number constraints.Unsigned, Signature comparable,
	ID constraints.Ordered, E Environment[Hash, Number, Signature, ID],
] struct {
	pastRounds    []backgroundRound[Hash, Number, Signature, ID, E]
	commitSenders map[uint64]chan Commit[Hash, Number, Signature, ID]
}

func newPastRounds[Hash constraints.Ordered, Number constraints.Unsigned, Signature comparable,
	ID constraints.Ordered, E Environment[Hash, Number, Signature, ID]]() *pastRounds[Hash, Number, Signature, ID, E] {
	return &pastRounds[Hash, Number, Signature, ID, E]{
		commitSenders: make(map[uint64]chan Commit[Hash, Number, Signature, ID]),
	}
}

// push an old voting round onto this stream.
func (p *pastRounds[Hash, Number, Signature, ID, E]) Push(env E, round VotingRound[Hash, Number, Signature, ID, E]) {
	roundNumber := round.RoundNumber()
	ch := make(chan Commit[Hash, Number, Signature, ID])
	background := backgroundRound[Hash, Number, Signature, ID, E]{
		inner: round,
		// this will get updated in a call to pastRounds.UpdateFinalized() on next poll
		finalizedNumber: 0,
		roundCommitter:  newRoundCommitter[Hash, Number, Signature, ID, E](env.RoundCommitTimer(), newWakerChan(ch)),
	}
	p.pastRounds = append(p.pastRounds, background)
	p.commitSenders[roundNumber] = ch
}

// update the last finalized block. this will lead to
// any irrelevant background rounds being pruned.
func (p *pastRounds[Hash, Number, Signature, ID, E]) UpdateFinalized(fNum Number) { //skipcq: RVV-B0001
	// have the task check if it should be pruned.
	// if so, this future will be re-polled
	for i := range p.pastRounds {
		p.pastRounds[i].updateFinalized(fNum)
	}
}

// Get the underlying `VotingRound` items that are being run in the background.
func (p *pastRounds[Hash, Number, Signature, ID, E]) VotingRounds() []VotingRound[Hash, Number, Signature, ID, E] {
	var votingRounds []VotingRound[Hash, Number, Signature, ID, E]
	for _, bg := range p.pastRounds {
		votingRounds = append(votingRounds, bg.votingRound())
	}
	return votingRounds
}

// import the commit into the given backgrounded round. If not possible,
// just return and process the commit.
func (p pastRounds[Hash, Number, Signature, ID, E]) ImportCommit(
	roundNumber uint64,
	commit Commit[Hash, Number, Signature, ID],
) *Commit[Hash, Number, Signature, ID] {
	sender, ok := p.commitSenders[roundNumber]
	if !ok {
		return &commit
	}
	select {
	case sender <- commit:
		return nil
	default:
		return &commit
	}
}

type numberCommit[Hash, Number, Signature, ID any] struct {
	Number uint64
	Commit Commit[Hash, Number, Signature, ID]
}

func (p *pastRounds[Hash, Number, Signature, ID, E]) pollNext(waker *waker) (
	ready bool,
	nc *numberCommit[Hash, Number, Signature, ID],
	err error,
) {
	for {
		if len(p.pastRounds) == 0 {
			return true, nc, nil
		}
		br := p.pastRounds[0]
		ready, backgroundRoundChange, err := br.poll(waker)
		switch {
		case ready && err == nil:
			v := backgroundRoundChange.Variant()
			// empty stream
			if v == nil {
				return true, nil, nil
			}
			switch v := v.(type) {
			case concluded:
				number := v
				round := br.inner
				err := round.Env().Concluded(
					round.RoundNumber(),
					round.RoundState(),
					round.DagBase(),
					round.HistoricalVotes(),
				)
				if err != nil {
					return true, nil, err
				}

				delete(p.commitSenders, uint64(number))
				p.pastRounds = p.pastRounds[1:]
			case committed[Hash, Number, Signature, ID]:
				number := br.roundNumber()
				commit := Commit[Hash, Number, Signature, ID](v)

				// reschedule until irrelevant
				p.pastRounds = append(p.pastRounds[1:], br)

				fmt.Printf(
					"Committing: round_number = %v, target_number = %v, target_hash = %v\n",
					number,
					commit.TargetNumber,
					commit.TargetHash,
				)

				return true, &numberCommit[Hash, Number, Signature, ID]{number, commit}, nil
			}
		case ready && err != nil:
			return true, nc, err
		case !ready:
			// reschedule until irrelevant
			p.pastRounds = append(p.pastRounds[1:], br)
			return false, nc, nil
		}
	}

}

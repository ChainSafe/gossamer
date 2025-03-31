// Copyright 2023 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package grandpa

import (
	"testing"

	primitives "github.com/ChainSafe/gossamer/internal/primitives/consensus/grandpa"
	grandpa "github.com/ChainSafe/gossamer/pkg/finality-grandpa"
	"github.com/ChainSafe/gossamer/pkg/scale"
	"github.com/stretchr/testify/require"
)

type hashString string

func (hs hashString) Bytes() []byte {
	return []byte(hs)
}
func (hs hashString) Length() int {
	return len(hs)
}
func (hs hashString) String() string {
	return string(hs)
}

func TestSharedVoterSetState_hasVoted(t *testing.T) {
	// Has Not Voted
	hasNotVoted := hasVotedNo[hashString, uint]{}
	sharedVoterSetState := NewSharedVoterSetState(voterSetStateLive[hashString, uint]{
		CurrentRounds: make(currentRounds[hashString, uint]),
	})
	voted := sharedVoterSetState.hasVoted(0)
	require.Equal(t, hasNotVoted, voted)

	// Has Voted
	vote := votePropose[hashString, uint]{}
	yes := hasVotedYes[hashString, uint]{
		AuthorityID: newTestPublic(t, 1),
		Vote:        vote,
	}
	newCurrentRounds := currentRounds[hashString, uint]{
		1: yes,
	}
	liveState := voterSetStateLive[hashString, uint]{
		CurrentRounds: newCurrentRounds,
	}

	sharedVoterSetState = NewSharedVoterSetState[hashString, uint](liveState)
	voted = sharedVoterSetState.hasVoted(1)
	require.Equal(t, yes, voted)
}

func TestCompleteRoundEncoding(t *testing.T) {
	dummyHashNumber := grandpa.HashNumber[hashString, uint]{
		Hash:   "a",
		Number: 1,
	}

	compRound := completedRound[hashString, uint]{
		Number: 1,
		State: grandpa.RoundState[hashString, uint]{
			PrevoteGHOST: &dummyHashNumber,
			Finalized:    &dummyHashNumber,
			Estimate:     &dummyHashNumber,
			Completable:  true,
		},
		Base: dummyHashNumber,
	}

	enc, err := scale.Marshal(compRound)
	require.NoError(t, err)

	newCompletedRound := completedRound[hashString, uint]{}
	err = scale.Unmarshal(enc, &newCompletedRound)
	require.NoError(t, err)
	require.Equal(t, compRound, newCompletedRound)
}

func TestCompletedRoundsEncoding(t *testing.T) {
	authorities := AuthoritySet[hashString, uint]{
		SetID:                  1,
		PendingStandardChanges: NewChangeTree[hashString, uint](),
		PendingForcedChanges:   []PendingChange[hashString, uint]{},
		AuthoritySetChanges:    AuthoritySetChanges[uint]{},
	}

	dummyHashNumber := grandpa.HashNumber[hashString, uint]{
		Hash:   "a",
		Number: 1,
	}

	completedRound := completedRound[hashString, uint]{
		Number: 1,
		State: grandpa.RoundState[hashString, uint]{
			PrevoteGHOST: &dummyHashNumber,
			Finalized:    &dummyHashNumber,
			Estimate:     &dummyHashNumber,
			Completable:  true,
		},
		Base: dummyHashNumber,
	}

	compRounds := newCompletedRounds[hashString, uint](completedRound, 1, authorities)
	enc, err := scale.Marshal(compRounds)
	require.NoError(t, err)

	var newCompletedRounds completedRounds[hashString, uint]
	err = scale.Unmarshal(enc, &newCompletedRounds)
	require.NoError(t, err)
	require.Equal(t, compRounds, newCompletedRounds)
}

func TestCompletedRounds_Iter(t *testing.T) {
	dummyHashNumber := grandpa.HashNumber[hashString, uint]{
		Hash:   "a",
		Number: 1,
	}

	completedRound0 := completedRound[hashString, uint]{
		Number: 0,
		State: grandpa.RoundState[hashString, uint]{
			PrevoteGHOST: &dummyHashNumber,
			Finalized:    &dummyHashNumber,
			Estimate:     &dummyHashNumber,
			Completable:  true,
		},
		Base: dummyHashNumber,
	}

	completedRound1 := completedRound[hashString, uint]{
		Number: 1,
		State: grandpa.RoundState[hashString, uint]{
			PrevoteGHOST: &dummyHashNumber,
			Finalized:    &dummyHashNumber,
			Estimate:     &dummyHashNumber,
			Completable:  true,
		},
		Base: dummyHashNumber,
	}

	completedRound2 := completedRound[hashString, uint]{
		Number: 2,
		State: grandpa.RoundState[hashString, uint]{
			PrevoteGHOST: &dummyHashNumber,
			Finalized:    &dummyHashNumber,
			Estimate:     &dummyHashNumber,
			Completable:  true,
		},
		Base: dummyHashNumber,
	}
	rounds := make([]completedRound[hashString, uint], 0, 3)
	rounds = append(rounds, completedRound0)
	rounds = append(rounds, completedRound1)
	rounds = append(rounds, completedRound2)

	expRounds := make([]completedRound[hashString, uint], 0, 3)
	expRounds = append(expRounds, completedRound2)
	expRounds = append(expRounds, completedRound1)
	expRounds = append(expRounds, completedRound0)

	compRounds := completedRounds[hashString, uint]{
		Rounds: rounds,
	}

	revRounds := compRounds.iter()
	require.Equal(t, expRounds, revRounds)
}

func TestCompletedRounds_Last(t *testing.T) {
	authorities := AuthoritySet[hashString, uint]{
		SetID:                  1,
		PendingStandardChanges: NewChangeTree[hashString, uint](),
		PendingForcedChanges:   []PendingChange[hashString, uint]{},
		AuthoritySetChanges:    AuthoritySetChanges[uint]{},
	}

	dummyHashNumber := grandpa.HashNumber[hashString, uint]{
		Hash:   "a",
		Number: 1,
	}

	compRound := completedRound[hashString, uint]{
		Number: 1,
		State: grandpa.RoundState[hashString, uint]{
			PrevoteGHOST: &dummyHashNumber,
			Finalized:    &dummyHashNumber,
			Estimate:     &dummyHashNumber,
			Completable:  true,
		},
		Base: dummyHashNumber,
	}
	compRounds := newCompletedRounds[hashString, uint](compRound, 1, authorities)
	lastCompletedRound := compRounds.last()
	require.Equal(t, compRound, lastCompletedRound)

	emptyCompletedRounds := completedRounds[hashString, uint]{}
	require.Panics(t, func() { emptyCompletedRounds.last() }, "last did not panic")
}

func TestCompletedRounds_Push(t *testing.T) {
	authorities := AuthoritySet[hashString, uint]{
		SetID:                  1,
		PendingStandardChanges: NewChangeTree[hashString, uint](),
		PendingForcedChanges:   []PendingChange[hashString, uint]{},
		AuthoritySetChanges:    AuthoritySetChanges[uint]{},
	}

	dummyHashNumber := grandpa.HashNumber[hashString, uint]{
		Hash:   "a",
		Number: 1,
	}

	completedRound0 := completedRound[hashString, uint]{
		Number: 0,
		State: grandpa.RoundState[hashString, uint]{
			PrevoteGHOST: &dummyHashNumber,
			Finalized:    &dummyHashNumber,
			Estimate:     &dummyHashNumber,
			Completable:  true,
		},
		Base: dummyHashNumber,
	}

	completedRound1 := completedRound[hashString, uint]{
		Number: 1,
		State: grandpa.RoundState[hashString, uint]{
			PrevoteGHOST: &dummyHashNumber,
			Finalized:    &dummyHashNumber,
			Estimate:     &dummyHashNumber,
			Completable:  true,
		},
		Base: dummyHashNumber,
	}

	completedRound2 := completedRound[hashString, uint]{
		Number: 2,
		State: grandpa.RoundState[hashString, uint]{
			PrevoteGHOST: &dummyHashNumber,
			Finalized:    &dummyHashNumber,
			Estimate:     &dummyHashNumber,
			Completable:  true,
		},
		Base: dummyHashNumber,
	}
	completedRounds := newCompletedRounds[hashString, uint](completedRound1, 1, authorities)
	completedRounds.push(completedRound0)

	lastCompletedRound := completedRounds.last()
	require.Equal(t, completedRound1, lastCompletedRound)

	completedRounds.push(completedRound2)
	lastCompletedRound = completedRounds.last()
	require.Equal(t, completedRound2, lastCompletedRound)
}

func TestCurrentRoundsEncoding(t *testing.T) {
	currRounds := make(currentRounds[hashString, uint64])
	currRounds[1] = hasVotedNo[hashString, uint64]{}

	enc, err := scale.Marshal(currRounds)
	require.NoError(t, err)

	newCurrentRounds := make(currentRounds[hashString, uint64])
	err = scale.Unmarshal(enc, &newCurrentRounds)
	require.NoError(t, err)
	require.Equal(t, currRounds, newCurrentRounds)
}

func TestVoterSetStateEncoding(t *testing.T) {
	authorities := AuthoritySet[hashString, uint]{}

	dummyHashNumber := grandpa.HashNumber[hashString, uint]{
		Hash:   "a",
		Number: 1,
	}

	compRound := completedRound[hashString, uint]{
		Number: 1,
		State: grandpa.RoundState[hashString, uint]{
			PrevoteGHOST: &dummyHashNumber,
			Finalized:    &dummyHashNumber,
			Estimate:     &dummyHashNumber,
			Completable:  true,
		},
		Base: dummyHashNumber,
	}

	completedRounds := newCompletedRounds[hashString, uint](compRound, 1, authorities)
	var currentRounds currentRounds[hashString, uint]

	liveState := voterSetStateLive[hashString, uint]{
		CompletedRounds: completedRounds,
		CurrentRounds:   currentRounds,
	}

	voterSetState := newVoterSetStateVDT[hashString, uint]()
	err := voterSetState.SetValue(liveState)
	require.NoError(t, err)

	enc, err := scale.Marshal(*voterSetState)
	require.NoError(t, err)

	newVoterSetState := newVoterSetStateVDT[hashString, uint]()
	err = scale.Unmarshal(enc, newVoterSetState)
	require.NoError(t, err)

	oldVal, err := voterSetState.Value()
	require.NoError(t, err)

	newVal, err := newVoterSetState.Value()
	require.NoError(t, err)
	require.Equal(t, oldVal.(voterSetStateLive[hashString, uint]), newVal.(voterSetStateLive[hashString, uint]))
}

func TestVoterSetState_Live(t *testing.T) {
	authorities := AuthoritySet[hashString, uint]{
		SetID:                  1,
		PendingStandardChanges: NewChangeTree[hashString, uint](),
		PendingForcedChanges:   []PendingChange[hashString, uint]{},
		AuthoritySetChanges:    AuthoritySetChanges[uint]{},
	}

	dummyHashNumber := grandpa.HashNumber[hashString, uint]{
		Hash:   "a",
		Number: 1,
	}

	live := newVoterSetStateLive(5, authorities, dummyHashNumber)
	require.Equal(t, primitives.SetID(5), live.CompletedRounds.SetId)
	require.Equal(t, primitives.RoundNumber(0), live.CompletedRounds.Rounds[0].Number)
}

func TestVoterSetState_CompletedRounds(t *testing.T) {
	authorities := AuthoritySet[hashString, uint]{
		SetID:                  1,
		PendingStandardChanges: NewChangeTree[hashString, uint](),
		PendingForcedChanges:   []PendingChange[hashString, uint]{},
		AuthoritySetChanges:    AuthoritySetChanges[uint]{},
	}
	dummyHashNumber := grandpa.HashNumber[hashString, uint]{
		Hash:   "a",
		Number: 1,
	}
	state := grandpa.NewRoundState(dummyHashNumber)
	completedRounds := newCompletedRounds(
		completedRound[hashString, uint]{
			10,
			state,
			dummyHashNumber,
			[]primitives.SignedMessage[hashString, uint]{},
		},
		5,
		authorities,
	)

	voterSetState := voterSetStateLive[hashString, uint]{
		CompletedRounds: completedRounds,
	}

	rounds := voterSetState.completedRounds()
	require.Equal(t, completedRounds, rounds)
}

func TestVoterSetState_LastCompletedRound(t *testing.T) {
	authorities := AuthoritySet[hashString, uint]{
		SetID:                  1,
		PendingStandardChanges: NewChangeTree[hashString, uint](),
		PendingForcedChanges:   []PendingChange[hashString, uint]{},
		AuthoritySetChanges:    AuthoritySetChanges[uint]{},
	}
	dummyHashNumber := grandpa.HashNumber[hashString, uint]{
		Hash:   "a",
		Number: 1,
	}
	state := grandpa.NewRoundState[hashString, uint](dummyHashNumber)

	originalCompletedRound := completedRound[hashString, uint]{
		8,
		state,
		dummyHashNumber,
		[]primitives.SignedMessage[hashString, uint]{},
	}
	completedRounds := newCompletedRounds[hashString, uint](
		originalCompletedRound,
		5,
		authorities,
	)

	addedCompletedRound := completedRound[hashString, uint]{
		8,
		state,
		dummyHashNumber,
		[]primitives.SignedMessage[hashString, uint]{},
	}

	completedRounds.push(addedCompletedRound)

	voterSetState := voterSetStatePaused[hashString, uint]{
		CompletedRounds: completedRounds,
	}

	lastCompletedRound := voterSetState.lastCompletedRound()
	require.Equal(t, originalCompletedRound, lastCompletedRound)
}

func TestVoterSetState_WithCurrentRound(t *testing.T) {
	authorities := AuthoritySet[hashString, uint]{
		SetID:                  1,
		PendingStandardChanges: NewChangeTree[hashString, uint](),
		PendingForcedChanges:   []PendingChange[hashString, uint]{},
		AuthoritySetChanges:    AuthoritySetChanges[uint]{},
	}
	dummyHashNumber := grandpa.HashNumber[hashString, uint]{
		Hash:   "a",
		Number: 1,
	}
	state := grandpa.NewRoundState[hashString, uint](dummyHashNumber)
	completedRounds := newCompletedRounds[hashString, uint](
		completedRound[hashString, uint]{
			10,
			state,
			dummyHashNumber,
			[]primitives.SignedMessage[hashString, uint]{},
		},
		5,
		authorities,
	)

	var voterSetState voterSetState[hashString, uint]
	voterSetState = voterSetStatePaused[hashString, uint]{
		CompletedRounds: completedRounds,
	}
	_, _, err := voterSetState.withCurrentRound(1)
	require.NotNil(t, err)
	require.Equal(t, "voter acting while in paused state", err.Error())

	// voterSetStateLive: invalid round
	voterSetState = voterSetStateLive[hashString, uint]{
		CompletedRounds: completedRounds,
	}
	_, _, err = voterSetState.withCurrentRound(1)
	require.NotNil(t, err)
	require.Equal(t, "voter acting on a live round we are not tracking", err.Error())

	// Valid
	currentRounds := make(currentRounds[hashString, uint])
	currentRounds[1] = hasVotedNo[hashString, uint]{}
	voterSetState = voterSetStateLive[hashString, uint]{
		CompletedRounds: completedRounds,
		CurrentRounds:   currentRounds,
	}
	completed, current, err := voterSetState.withCurrentRound(1)
	require.NoError(t, err)
	require.Equal(t, completedRounds, completed)
	require.Equal(t, currentRounds, current)
}

func TestHasVotedEncoding(t *testing.T) {
	vote := voteVDT[hashString, uint]{}
	err := vote.SetValue(votePropose[hashString, uint]{})
	require.NoError(t, err)

	yes := hasVotedYes[hashString, uint]{
		AuthorityID: newTestPublic(t, 1),
		Vote:        vote.inner,
	}
	hv := hasVotedVDT[hashString, uint]{}
	err = hv.SetValue(yes)
	require.NoError(t, err)

	res, err := scale.Marshal(hv)
	require.NoError(t, err)

	newHasVoted := hasVotedVDT[hashString, uint]{}
	err = scale.Unmarshal(res, &newHasVoted)
	require.NoError(t, err)
	require.Equal(t, hv, newHasVoted)
}

func TestHasVoted_Propose(t *testing.T) {
	primaryPropose := grandpa.PrimaryPropose[hashString, uint]{
		TargetHash:   "a",
		TargetNumber: 2,
	}
	vote := votePropose[hashString, uint]{primaryPropose}
	yes := hasVotedYes[hashString, uint]{
		Vote: vote,
	}

	newPrimaryPropose := yes.Propose()
	require.NotNil(t, newPrimaryPropose)
	require.Equal(t, primaryPropose, *newPrimaryPropose)
}

func TestHasVoted_Prevote(t *testing.T) {
	prevoteVal := &grandpa.Prevote[hashString, uint]{
		TargetHash:   "a",
		TargetNumber: 2,
	}
	prevote := votePrevote[hashString, uint]{&grandpa.PrimaryPropose[hashString, uint]{}, *prevoteVal}

	yes := hasVotedYes[hashString, uint]{
		Vote: prevote,
	}
	newPrevote := yes.Prevote()
	require.Equal(t, prevoteVal, newPrevote)

	primaryPropose := &grandpa.PrimaryPropose[hashString, uint]{
		TargetHash:   "a",
		TargetNumber: 2,
	}
	proposeVote := votePropose[hashString, uint]{PrimaryPropose: *primaryPropose}
	yes = hasVotedYes[hashString, uint]{
		Vote: proposeVote,
	}
	newPrevote = yes.Prevote()
	require.Nil(t, newPrevote)
}

func TestHasVoted_Precommit(t *testing.T) {
	precommitVal := &grandpa.Precommit[hashString, uint]{
		TargetHash:   "a",
		TargetNumber: 2,
	}
	voteVal := votePrecommit[hashString, uint]{
		&grandpa.PrimaryPropose[hashString, uint]{},
		grandpa.Prevote[hashString, uint]{},
		*precommitVal,
	}
	y := hasVotedYes[hashString, uint]{
		Vote: voteVal,
	}

	newCommit := y.Precommit()
	require.Equal(t, precommitVal, newCommit)

	primaryPropose := &grandpa.PrimaryPropose[hashString, uint]{
		TargetHash:   "a",
		TargetNumber: 2,
	}
	proposeVote := votePropose[hashString, uint]{PrimaryPropose: *primaryPropose}
	y = hasVotedYes[hashString, uint]{
		Vote: proposeVote,
	}

	newCommit = y.Precommit()
	require.Nil(t, newCommit)
}

func TestHasVoted_CanPropose(t *testing.T) {
	primaryPropose := &grandpa.PrimaryPropose[hashString, uint]{
		TargetHash:   "a",
		TargetNumber: 2,
	}
	voteVal := votePropose[hashString, uint]{*primaryPropose}
	yes := hasVotedYes[hashString, uint]{
		Vote: voteVal,
	}
	require.False(t, yes.CanPropose())

	no := hasVotedNo[hashString, uint]{}
	require.True(t, no.CanPropose())
}

func TestHasVoted_CanPrevote(t *testing.T) {
	prevoteVal := &grandpa.Prevote[hashString, uint]{
		TargetHash:   "a",
		TargetNumber: 2,
	}
	voteVal := votePrevote[hashString, uint]{&grandpa.PrimaryPropose[hashString, uint]{}, *prevoteVal}
	yes := hasVotedYes[hashString, uint]{
		Vote: voteVal,
	}
	require.False(t, yes.CanPrevote())

	no := hasVotedNo[hashString, uint]{}
	require.True(t, no.CanPrevote())
}

func TestHasVoted_CanPrecommit(t *testing.T) {
	precommitVal := &grandpa.Precommit[hashString, uint]{
		TargetHash:   "a",
		TargetNumber: 2,
	}
	vote := votePrecommit[hashString, uint]{
		&grandpa.PrimaryPropose[hashString, uint]{},
		grandpa.Prevote[hashString, uint]{},
		*precommitVal,
	}
	yes := hasVotedYes[hashString, uint]{
		Vote: vote,
	}
	require.False(t, yes.CanPrecommit())

	no := hasVotedNo[hashString, uint]{}
	require.True(t, no.CanPrecommit())
}

func TestVoteEncoding(t *testing.T) {
	voteVal := voteVDT[hashString, uint]{}
	err := voteVal.SetValue(votePropose[hashString, uint]{
		PrimaryPropose: grandpa.PrimaryPropose[hashString, uint]{
			TargetHash:   "a",
			TargetNumber: 1,
		},
	})
	require.NoError(t, err)

	enc, err := scale.Marshal(voteVal)
	require.NoError(t, err)

	newVote := voteVDT[hashString, uint]{}
	err = scale.Unmarshal(enc, &newVote)
	require.NoError(t, err)
	require.Equal(t, voteVal, newVote)
}

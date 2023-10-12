package grandpa

import (
	"github.com/ChainSafe/gossamer/lib/crypto/ed25519"
	grandpa "github.com/ChainSafe/gossamer/pkg/finality-grandpa"
	"github.com/ChainSafe/gossamer/pkg/scale"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestSharedVoterSetState_hasVoted(t *testing.T) {
	// Has Not Voted
	hasNotVoted := hasVoted[string, uint]{}
	hasNotVoted = hasNotVoted.New()
	err := hasNotVoted.Set(no{})
	require.NoError(t, err)
	voterSetState := *NewVoterSetState[string, uint, uint, uint]()
	sharedVoterSetState := NewSharedVoterSetState[string, uint](voterSetState)
	voted, err := sharedVoterSetState.hasVoted(0)
	require.NoError(t, err)
	require.Equal(t, hasNotVoted, voted)

	// Has Voted
	pub, err := ed25519.NewPublicKey([]byte{1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1})
	require.NoError(t, err)

	vote := vote[string, uint]{}
	vote = vote.New()
	err = vote.Set(propose[string, uint]{})
	require.NoError(t, err)

	yes := yes[string, uint]{
		AuthId: *pub,
		Vote:   vote,
	}

	hasIndeedVoted := hasVoted[string, uint]{}
	hasIndeedVoted = hasIndeedVoted.New()
	err = hasIndeedVoted.Set(yes)
	require.NoError(t, err)

	example := make(map[uint64]hasVoted[string, uint])
	example[1] = hasIndeedVoted
	newCurrentRounds := CurrentRounds[string, uint](
		example,
	)
	liveState := voterSetStateLive[string, uint, uint, uint]{
		CurrentRounds: newCurrentRounds,
	}

	newVoterSetState := *NewVoterSetState[string, uint, uint, uint]()
	err = newVoterSetState.Set(liveState)
	require.NoError(t, err)

	sharedVoterSetState = NewSharedVoterSetState[string, uint](newVoterSetState)
	voted, err = sharedVoterSetState.hasVoted(1)
	require.NoError(t, err)
	require.Equal(t, hasIndeedVoted, voted)
}

func TestCompleteRoundEncoding(t *testing.T) {
	dummyHashNumber := grandpa.HashNumber[string, uint]{
		Hash:   "a",
		Number: 1,
	}

	compRound := completedRound[string, uint, uint, uint]{
		Number: 1,
		State: grandpa.RoundState[string, uint]{
			PrevoteGHOST: &dummyHashNumber,
			Finalized:    &dummyHashNumber,
			Estimate:     &dummyHashNumber,
			Completable:  true,
		},
		Base: dummyHashNumber,
	}

	enc, err := scale.Marshal(compRound)
	require.NoError(t, err)

	newCompletedRound := completedRound[string, uint, uint, uint]{}
	err = scale.Unmarshal(enc, &newCompletedRound)
	require.NoError(t, err)
	require.Equal(t, compRound, newCompletedRound)
}

func TestCompletedRoundsEncoding(t *testing.T) {
	authorities := AuthoritySet[string, uint, uint]{
		CurrentAuthorities:     []Authority[uint]{},
		SetID:                  1,
		PendingStandardChanges: NewChangeTree[string, uint, uint](),
		PendingForcedChanges:   []PendingChange[string, uint, uint]{},
		AuthoritySetChanges:    AuthoritySetChanges[uint]{},
	}

	dummyHashNumber := grandpa.HashNumber[string, uint]{
		Hash:   "a",
		Number: 1,
	}

	completedRound := completedRound[string, uint, uint, uint]{
		Number: 1,
		State: grandpa.RoundState[string, uint]{
			PrevoteGHOST: &dummyHashNumber,
			Finalized:    &dummyHashNumber,
			Estimate:     &dummyHashNumber,
			Completable:  true,
		},
		Base: dummyHashNumber,
	}

	compRounds := NewCompletedRounds[string, uint](completedRound, 1, authorities)
	enc, err := scale.Marshal(compRounds)
	require.NoError(t, err)

	var newCompletedRounds completedRounds[string, uint, uint, uint]
	err = scale.Unmarshal(enc, &newCompletedRounds)
	require.NoError(t, err)
	require.Equal(t, compRounds, newCompletedRounds)
}

func TestCompletedRounds_Last(t *testing.T) {
	authorities := AuthoritySet[string, uint, uint]{
		CurrentAuthorities:     []Authority[uint]{},
		SetID:                  1,
		PendingStandardChanges: NewChangeTree[string, uint, uint](),
		PendingForcedChanges:   []PendingChange[string, uint, uint]{},
		AuthoritySetChanges:    AuthoritySetChanges[uint]{},
	}

	dummyHashNumber := grandpa.HashNumber[string, uint]{
		Hash:   "a",
		Number: 1,
	}

	compRound := completedRound[string, uint, uint, uint]{
		Number: 1,
		State: grandpa.RoundState[string, uint]{
			PrevoteGHOST: &dummyHashNumber,
			Finalized:    &dummyHashNumber,
			Estimate:     &dummyHashNumber,
			Completable:  true,
		},
		Base: dummyHashNumber,
	}
	compRounds := NewCompletedRounds[string, uint](compRound, 1, authorities)
	lastCompletedRound := compRounds.last()
	require.Equal(t, compRound, lastCompletedRound)

	emptyCompletedRounds := completedRounds[string, uint, uint, uint]{}
	require.Panics(t, func() { emptyCompletedRounds.last() }, "last did not panic")
}

func TestCompletedRounds_Push(t *testing.T) {
	authorities := AuthoritySet[string, uint, uint]{
		CurrentAuthorities:     []Authority[uint]{},
		SetID:                  1,
		PendingStandardChanges: NewChangeTree[string, uint, uint](),
		PendingForcedChanges:   []PendingChange[string, uint, uint]{},
		AuthoritySetChanges:    AuthoritySetChanges[uint]{},
	}

	dummyHashNumber := grandpa.HashNumber[string, uint]{
		Hash:   "a",
		Number: 1,
	}

	completedRound0 := completedRound[string, uint, uint, uint]{
		Number: 1,
		State: grandpa.RoundState[string, uint]{
			PrevoteGHOST: &dummyHashNumber,
			Finalized:    &dummyHashNumber,
			Estimate:     &dummyHashNumber,
			Completable:  true,
		},
		Base: dummyHashNumber,
	}

	completedRound1 := completedRound[string, uint, uint, uint]{
		Number: 1,
		State: grandpa.RoundState[string, uint]{
			PrevoteGHOST: &dummyHashNumber,
			Finalized:    &dummyHashNumber,
			Estimate:     &dummyHashNumber,
			Completable:  true,
		},
		Base: dummyHashNumber,
	}

	completedRound2 := completedRound[string, uint, uint, uint]{
		Number: 1,
		State: grandpa.RoundState[string, uint]{
			PrevoteGHOST: &dummyHashNumber,
			Finalized:    &dummyHashNumber,
			Estimate:     &dummyHashNumber,
			Completable:  true,
		},
		Base: dummyHashNumber,
	}
	completedRounds := NewCompletedRounds[string, uint](completedRound1, 1, authorities)
	completedRounds.push(completedRound0)

	lastCompletedRound := completedRounds.last()
	require.Equal(t, completedRound0, lastCompletedRound)

	completedRounds.push(completedRound2)
	lastCompletedRound = completedRounds.last()
	require.Equal(t, completedRound0, lastCompletedRound)
}

func TestCurrentRoundsEncoding(t *testing.T) {
	// TODO this fails
	currentRounds := CurrentRounds[string, uint](
		make(map[uint64]hasVoted[string, uint]),
	)

	hv := hasVoted[string, uint]{}
	hv = hv.New()
	err := hv.Set(no{})
	require.NoError(t, err)
	currentRounds[1] = hv

	enc, err := scale.Marshal(currentRounds)
	require.NoError(t, err)

	hasVotedNew := hasVoted[string, uint]{}
	hasVotedNew = hv.New()
	example := make(map[uint64]hasVoted[string, uint])
	example[1] = hasVotedNew
	newCurrentRounds := CurrentRounds[string, uint](
		example,
	)
	err = scale.Unmarshal(enc, &newCurrentRounds)
	require.NoError(t, err)
	require.Equal(t, currentRounds, newCurrentRounds)
}

func TestVoterSetStateEncoding(t *testing.T) {
	authorities := AuthoritySet[string, uint, uint]{}

	dummyHashNumber := grandpa.HashNumber[string, uint]{
		Hash:   "a",
		Number: 1,
	}

	compRound := completedRound[string, uint, uint, uint]{
		Number: 1,
		State: grandpa.RoundState[string, uint]{
			PrevoteGHOST: &dummyHashNumber,
			Finalized:    &dummyHashNumber,
			Estimate:     &dummyHashNumber,
			Completable:  true,
		},
		Base: dummyHashNumber,
	}

	completedRounds := NewCompletedRounds[string, uint, uint, uint](compRound, 1, authorities)
	currentRounds := CurrentRounds[string, uint](
		make(map[uint64]hasVoted[string, uint]),
	)

	liveState := voterSetStateLive[string, uint, uint, uint]{
		CompletedRounds: completedRounds,
		CurrentRounds:   currentRounds,
	}

	voterSetState := *NewVoterSetState[string, uint, uint, uint]()
	err := voterSetState.Set(liveState)
	require.NoError(t, err)

	enc, err := scale.Marshal(voterSetState)
	require.NoError(t, err)

	newVoterSetState := *NewVoterSetState[string, uint, uint, uint]()
	err = scale.Unmarshal(enc, &newVoterSetState)
	require.NoError(t, err)

	oldVal, err := voterSetState.Value()
	require.NoError(t, err)

	newVal, err := newVoterSetState.Value()
	require.NoError(t, err)
	require.Equal(t, oldVal.(voterSetStateLive[string, uint, uint, uint]), newVal.(voterSetStateLive[string, uint, uint, uint]))
}

func TestVoterSetState_Live(t *testing.T) {
	authorities := AuthoritySet[string, uint, uint]{
		CurrentAuthorities:     []Authority[uint]{},
		SetID:                  1,
		PendingStandardChanges: NewChangeTree[string, uint, uint](),
		PendingForcedChanges:   []PendingChange[string, uint, uint]{},
		AuthoritySetChanges:    AuthoritySetChanges[uint]{},
	}

	dummyHashNumber := grandpa.HashNumber[string, uint]{
		Hash:   "a",
		Number: 1,
	}

	liveSetState, err := NewLiveVoterSetState[string, uint, uint, uint](5, authorities, dummyHashNumber)
	require.NoError(t, err)

	live, err := liveSetState.Value()
	require.NoError(t, err)

	val, ok := live.(voterSetStateLive[string, uint, uint, uint])
	require.True(t, ok)
	require.Equal(t, uint64(5), val.CompletedRounds.SetId)
	require.Equal(t, uint64(0), val.CompletedRounds.Rounds[0].Number)
}

func TestVoterSetState_CompletedRounds(t *testing.T) {
	authorities := AuthoritySet[string, uint, uint]{
		CurrentAuthorities:     []Authority[uint]{},
		SetID:                  1,
		PendingStandardChanges: NewChangeTree[string, uint, uint](),
		PendingForcedChanges:   []PendingChange[string, uint, uint]{},
		AuthoritySetChanges:    AuthoritySetChanges[uint]{},
	}
	dummyHashNumber := grandpa.HashNumber[string, uint]{
		Hash:   "a",
		Number: 1,
	}
	state := grandpa.NewRoundState[string, uint](dummyHashNumber)
	completedRounds := NewCompletedRounds[string, uint, uint, uint](
		completedRound[string, uint, uint, uint]{
			10,
			state,
			dummyHashNumber,
			[]grandpa.SignedMessage[string, uint, uint, uint]{},
		},
		5,
		authorities,
	)

	voterSetState := NewVoterSetState[string, uint, uint, uint]()

	err := voterSetState.Set(voterSetStateLive[string, uint, uint, uint]{
		CompletedRounds: completedRounds,
	})
	require.NoError(t, err)

	rounds, err := voterSetState.completedRounds()
	require.NoError(t, err)
	require.Equal(t, completedRounds, rounds)
}

func TestVoterSetState_LastCompletedRound(t *testing.T) {
	authorities := AuthoritySet[string, uint, uint]{
		CurrentAuthorities:     []Authority[uint]{},
		SetID:                  1,
		PendingStandardChanges: NewChangeTree[string, uint, uint](),
		PendingForcedChanges:   []PendingChange[string, uint, uint]{},
		AuthoritySetChanges:    AuthoritySetChanges[uint]{},
	}
	dummyHashNumber := grandpa.HashNumber[string, uint]{
		Hash:   "a",
		Number: 1,
	}
	state := grandpa.NewRoundState[string, uint](dummyHashNumber)
	completedRounds := NewCompletedRounds[string, uint, uint, uint](
		completedRound[string, uint, uint, uint]{
			10,
			state,
			dummyHashNumber,
			[]grandpa.SignedMessage[string, uint, uint, uint]{},
		},
		5,
		authorities,
	)

	addedCompletedRound := completedRound[string, uint, uint, uint]{
		8,
		state,
		dummyHashNumber,
		[]grandpa.SignedMessage[string, uint, uint, uint]{},
	}

	completedRounds.push(addedCompletedRound)

	voterSetState := NewVoterSetState[string, uint, uint, uint]()
	err := voterSetState.Set(voterSetStatePaused[string, uint, uint, uint]{
		CompletedRounds: completedRounds,
	})
	require.NoError(t, err)

	lastCompletedRound, err := voterSetState.lastCompletedRound()
	require.NoError(t, err)
	require.Equal(t, addedCompletedRound, lastCompletedRound)
}

func TestVoterSetState_WithCurrentRound(t *testing.T) {
	authorities := AuthoritySet[string, uint, uint]{
		CurrentAuthorities:     []Authority[uint]{},
		SetID:                  1,
		PendingStandardChanges: NewChangeTree[string, uint, uint](),
		PendingForcedChanges:   []PendingChange[string, uint, uint]{},
		AuthoritySetChanges:    AuthoritySetChanges[uint]{},
	}
	dummyHashNumber := grandpa.HashNumber[string, uint]{
		Hash:   "a",
		Number: 1,
	}
	state := grandpa.NewRoundState[string, uint](dummyHashNumber)
	completedRounds := NewCompletedRounds[string, uint, uint, uint](
		completedRound[string, uint, uint, uint]{
			10,
			state,
			dummyHashNumber,
			[]grandpa.SignedMessage[string, uint, uint, uint]{},
		},
		5,
		authorities,
	)

	voterSetState := NewVoterSetState[string, uint, uint, uint]()

	// voterSetStatePaused
	err := voterSetState.Set(voterSetStatePaused[string, uint, uint, uint]{
		CompletedRounds: completedRounds,
	})
	require.NoError(t, err)
	_, _, err = voterSetState.withCurrentRound(1)
	require.NotNil(t, err)
	require.Equal(t, "voter acting while in paused state", err.Error())

	// voterSetStateLive: invalid round
	err = voterSetState.Set(voterSetStateLive[string, uint, uint, uint]{
		CompletedRounds: completedRounds,
	})
	require.NoError(t, err)
	_, _, err = voterSetState.withCurrentRound(1)
	require.NotNil(t, err)
	require.Equal(t, "voter acting on a live round we are not tracking", err.Error())

	// Valid
	currentRounds := CurrentRounds[string, uint](
		make(map[uint64]hasVoted[string, uint]),
	)

	hasVoted := hasVoted[string, uint]{}
	hasVoted = hasVoted.New()
	err = hasVoted.Set(no{})
	require.NoError(t, err)
	currentRounds[1] = hasVoted
	err = voterSetState.Set(voterSetStateLive[string, uint, uint, uint]{
		CompletedRounds: completedRounds,
		CurrentRounds:   currentRounds,
	})
	require.NoError(t, err)
	completed, current, err := voterSetState.withCurrentRound(1)
	require.NoError(t, err)
	require.Equal(t, completedRounds, completed)
	require.Equal(t, currentRounds, current)
}

func TestHasVotedEncoding(t *testing.T) {
	pub, err := ed25519.NewPublicKey([]byte{1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1})
	require.NoError(t, err)

	vote := vote[string, uint]{}
	vote = vote.New()
	err = vote.Set(propose[string, uint]{})
	require.NoError(t, err)

	yes := yes[string, uint]{
		AuthId: *pub,
		Vote:   vote,
	}
	hv := hasVoted[string, uint]{}
	hv = hv.New()
	err = hv.Set(yes)
	require.NoError(t, err)

	res, err := scale.Marshal(hv)
	require.NoError(t, err)

	newHasVoted := hasVoted[string, uint]{}
	newHasVoted = hv.New()
	err = scale.Unmarshal(res, &newHasVoted)
	require.NoError(t, err)
	require.Equal(t, hv, newHasVoted)
}

func TestHasVoted_Propose(t *testing.T) {
	primaryPropose := &grandpa.PrimaryPropose[string, uint]{
		TargetHash:   "a",
		TargetNumber: 2,
	}
	vote := vote[string, uint]{}
	vote = vote.New()
	err := vote.Set(propose[string, uint]{*primaryPropose})
	require.NoError(t, err)

	yes := yes[string, uint]{
		Vote: vote,
	}
	hasVoted := hasVoted[string, uint]{}
	hasVoted = hasVoted.New()
	err = hasVoted.Set(yes)
	require.NoError(t, err)

	newPrimaryPropose := hasVoted.Propose()
	require.Equal(t, primaryPropose, newPrimaryPropose)
}

func TestHasVoted_Prevote(t *testing.T) {
	prevoteVal := &grandpa.Prevote[string, uint]{
		TargetHash:   "a",
		TargetNumber: 2,
	}
	voteVal := vote[string, uint]{}
	voteVal = voteVal.New()
	err := voteVal.Set(prevote[string, uint]{&grandpa.PrimaryPropose[string, uint]{}, *prevoteVal})
	require.NoError(t, err)

	y := yes[string, uint]{
		Vote: voteVal,
	}
	hasVoted := hasVoted[string, uint]{}
	hasVoted = hasVoted.New()
	err = hasVoted.Set(y)
	require.NoError(t, err)

	newPrevote := hasVoted.Prevote()
	require.Equal(t, prevoteVal, newPrevote)

	primaryPropose := &grandpa.PrimaryPropose[string, uint]{
		TargetHash:   "a",
		TargetNumber: 2,
	}
	proposeVote := vote[string, uint]{}
	proposeVote = voteVal.New()
	err = proposeVote.Set(propose[string, uint]{PrimaryPropose: *primaryPropose})
	require.NoError(t, err)

	y = yes[string, uint]{
		Vote: proposeVote,
	}
	hasVoted = hasVoted.New()
	err = hasVoted.Set(y)
	require.NoError(t, err)

	newPrevote = hasVoted.Prevote()
	require.Nil(t, newPrevote)
}

func TestHasVoted_Precommit(t *testing.T) {
	precommitVal := &grandpa.Precommit[string, uint]{
		TargetHash:   "a",
		TargetNumber: 2,
	}
	voteVal := vote[string, uint]{}
	voteVal = voteVal.New()
	err := voteVal.Set(precommit[string, uint]{&grandpa.PrimaryPropose[string, uint]{}, grandpa.Prevote[string, uint]{}, *precommitVal})
	require.NoError(t, err)

	y := yes[string, uint]{
		Vote: voteVal,
	}
	hasVoted := hasVoted[string, uint]{}
	hasVoted = hasVoted.New()
	err = hasVoted.Set(y)
	require.NoError(t, err)

	newCommit := hasVoted.Precommit()
	require.Equal(t, precommitVal, newCommit)

	primaryPropose := &grandpa.PrimaryPropose[string, uint]{
		TargetHash:   "a",
		TargetNumber: 2,
	}
	proposeVote := vote[string, uint]{}
	proposeVote = proposeVote.New()
	err = proposeVote.Set(propose[string, uint]{PrimaryPropose: *primaryPropose})
	require.NoError(t, err)

	y = yes[string, uint]{
		Vote: proposeVote,
	}
	hasVoted = hasVoted.New()
	err = hasVoted.Set(y)
	require.NoError(t, err)

	newCommit = hasVoted.Precommit()
	require.Nil(t, newCommit)
}

func TestHasVoted_CanPropose(t *testing.T) {
	primaryPropose := &grandpa.PrimaryPropose[string, uint]{
		TargetHash:   "a",
		TargetNumber: 2,
	}
	voteVal := vote[string, uint]{}
	voteVal = voteVal.New()
	err := voteVal.Set(propose[string, uint]{*primaryPropose})
	require.NoError(t, err)

	yes := yes[string, uint]{
		Vote: voteVal,
	}
	hasVoted := hasVoted[string, uint]{}
	hasVoted = hasVoted.New()
	err = hasVoted.Set(yes)
	require.NoError(t, err)
	require.False(t, hasVoted.CanPropose())

	hasVoted = hasVoted.New()
	err = hasVoted.Set(no{})
	require.NoError(t, err)
	require.True(t, hasVoted.CanPropose())
}

func TestHasVoted_CanPrevote(t *testing.T) {
	prevoteVal := &grandpa.Prevote[string, uint]{
		TargetHash:   "a",
		TargetNumber: 2,
	}
	voteVal := vote[string, uint]{}
	voteVal = voteVal.New()
	err := voteVal.Set(prevote[string, uint]{&grandpa.PrimaryPropose[string, uint]{}, *prevoteVal})
	require.NoError(t, err)

	yes := yes[string, uint]{
		Vote: voteVal,
	}
	hasVoted := hasVoted[string, uint]{}
	hasVoted = hasVoted.New()
	err = hasVoted.Set(yes)
	require.NoError(t, err)
	require.False(t, hasVoted.CanPrevote())

	hasVoted = hasVoted.New()
	err = hasVoted.Set(no{})
	require.NoError(t, err)
	require.True(t, hasVoted.CanPrevote())
}

func TestHasVoted_CanPrecommit(t *testing.T) {
	precommitVal := &grandpa.Precommit[string, uint]{
		TargetHash:   "a",
		TargetNumber: 2,
	}
	vote := vote[string, uint]{}
	vote = vote.New()
	err := vote.Set(precommit[string, uint]{&grandpa.PrimaryPropose[string, uint]{}, grandpa.Prevote[string, uint]{}, *precommitVal})
	require.NoError(t, err)

	yes := yes[string, uint]{
		Vote: vote,
	}
	hasVoted := hasVoted[string, uint]{}
	hasVoted = hasVoted.New()
	err = hasVoted.Set(yes)
	require.NoError(t, err)
	require.False(t, hasVoted.CanPrecommit())

	hasVoted = hasVoted.New()
	err = hasVoted.Set(no{})
	require.NoError(t, err)
	require.True(t, hasVoted.CanPrecommit())
}

func TestVoteEncoding(t *testing.T) {
	voteVal := vote[string, uint]{}
	voteVal = voteVal.New()
	err := voteVal.Set(propose[string, uint]{
		PrimaryPropose: grandpa.PrimaryPropose[string, uint]{
			TargetHash:   "a",
			TargetNumber: 1,
		},
	})
	require.NoError(t, err)

	enc, err := scale.Marshal(voteVal)
	require.NoError(t, err)

	newVote := vote[string, uint]{}
	newVote = newVote.New()
	err = scale.Unmarshal(enc, &newVote)
	require.NoError(t, err)
	require.Equal(t, voteVal, newVote)
}

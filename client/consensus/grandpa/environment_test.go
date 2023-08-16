package grandpa

import (
	"github.com/ChainSafe/gossamer/lib/crypto/ed25519"
	grandpa "github.com/ChainSafe/gossamer/pkg/finality-grandpa"
	"github.com/ChainSafe/gossamer/pkg/scale"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestCompleteRoundEncoding(t *testing.T) {
	dummyHashNumber := grandpa.HashNumber[string, uint]{
		Hash:   "a",
		Number: 1,
	}

	completedRound := CompletedRound[string, uint]{
		Number: 1,
		State: grandpa.RoundState[string, uint]{
			PrevoteGHOST: &dummyHashNumber,
			Finalized:    &dummyHashNumber,
			Estimate:     &dummyHashNumber,
			Completable:  true,
		},
		Base: dummyHashNumber,
	}

	enc, err := scale.Marshal(completedRound)
	require.NoError(t, err)

	newCompletedRound := CompletedRound[string, uint]{}
	err = scale.Unmarshal(enc, &newCompletedRound)
	require.NoError(t, err)
	require.Equal(t, completedRound, newCompletedRound)
}

func TestCompletedRoundsEncoding(t *testing.T) {
	authorities := AuthoritySet[string, uint]{
		CurrentAuthorities:     []Authority{},
		SetID:                  1,
		PendingStandardChanges: NewChangeTree[string, uint](),
		PendingForcedChanges:   []PendingChange[string, uint]{},
		AuthoritySetChanges:    AuthoritySetChanges[uint]{},
	}

	dummyHashNumber := grandpa.HashNumber[string, uint]{
		Hash:   "a",
		Number: 1,
	}

	completedRound := &CompletedRound[string, uint]{
		Number: 1,
		State: grandpa.RoundState[string, uint]{
			PrevoteGHOST: &dummyHashNumber,
			Finalized:    &dummyHashNumber,
			Estimate:     &dummyHashNumber,
			Completable:  true,
		},
		Base: dummyHashNumber,
	}

	completedRounds := NewCompletedRounds[string, uint](completedRound, 1, authorities)
	enc, err := scale.Marshal(completedRounds)
	require.NoError(t, err)

	var newCompletedRounds CompletedRounds[string, uint]
	err = scale.Unmarshal(enc, &newCompletedRounds)
	require.NoError(t, err)
	require.Equal(t, completedRounds, newCompletedRounds)
}

func TestCompletedRounds_Last(t *testing.T) {
	authorities := AuthoritySet[string, uint]{
		CurrentAuthorities:     []Authority{},
		SetID:                  1,
		PendingStandardChanges: NewChangeTree[string, uint](),
		PendingForcedChanges:   []PendingChange[string, uint]{},
		AuthoritySetChanges:    AuthoritySetChanges[uint]{},
	}

	dummyHashNumber := grandpa.HashNumber[string, uint]{
		Hash:   "a",
		Number: 1,
	}

	completedRound := CompletedRound[string, uint]{
		Number: 1,
		State: grandpa.RoundState[string, uint]{
			PrevoteGHOST: &dummyHashNumber,
			Finalized:    &dummyHashNumber,
			Estimate:     &dummyHashNumber,
			Completable:  true,
		},
		Base: dummyHashNumber,
	}
	completedRounds := NewCompletedRounds[string, uint](&completedRound, 1, authorities)
	lastCompletedRound := completedRounds.Last()
	require.Equal(t, completedRound, lastCompletedRound)

	completedRounds = NewCompletedRounds[string, uint](nil, 1, authorities)
	require.Panics(t, func() { completedRounds.Last() }, "Last did not panic")
}

func TestCompletedRounds_Push(t *testing.T) {
	authorities := AuthoritySet[string, uint]{
		CurrentAuthorities:     []Authority{},
		SetID:                  1,
		PendingStandardChanges: NewChangeTree[string, uint](),
		PendingForcedChanges:   []PendingChange[string, uint]{},
		AuthoritySetChanges:    AuthoritySetChanges[uint]{},
	}

	dummyHashNumber := grandpa.HashNumber[string, uint]{
		Hash:   "a",
		Number: 1,
	}

	completedRound0 := CompletedRound[string, uint]{
		Number: 1,
		State: grandpa.RoundState[string, uint]{
			PrevoteGHOST: &dummyHashNumber,
			Finalized:    &dummyHashNumber,
			Estimate:     &dummyHashNumber,
			Completable:  true,
		},
		Base: dummyHashNumber,
	}

	completedRound1 := CompletedRound[string, uint]{
		Number: 1,
		State: grandpa.RoundState[string, uint]{
			PrevoteGHOST: &dummyHashNumber,
			Finalized:    &dummyHashNumber,
			Estimate:     &dummyHashNumber,
			Completable:  true,
		},
		Base: dummyHashNumber,
	}

	completedRound2 := CompletedRound[string, uint]{
		Number: 1,
		State: grandpa.RoundState[string, uint]{
			PrevoteGHOST: &dummyHashNumber,
			Finalized:    &dummyHashNumber,
			Estimate:     &dummyHashNumber,
			Completable:  true,
		},
		Base: dummyHashNumber,
	}
	completedRounds := NewCompletedRounds[string, uint](&completedRound1, 1, authorities)
	completedRounds.Push(completedRound0)

	lastCompletedRound := completedRounds.Last()
	require.Equal(t, completedRound0, lastCompletedRound)

	completedRounds.Push(completedRound2)
	lastCompletedRound = completedRounds.Last()
	require.Equal(t, completedRound0, lastCompletedRound)
}

func TestCurrentRoundsEncoding(t *testing.T) {
	// TODO this fails
	currentRounds := CurrentRounds[string, uint](
		make(map[uint64]HasVoted[string, uint]),
	)

	hasVoted := &HasVoted[string, uint]{}
	hasVoted = hasVoted.New()
	err := hasVoted.Set(No{})
	require.NoError(t, err)
	currentRounds[1] = *hasVoted

	enc, err := scale.Marshal(currentRounds)
	require.NoError(t, err)

	hasVotedNew := &HasVoted[string, uint]{}
	hasVotedNew = hasVoted.New()
	example := make(map[uint64]HasVoted[string, uint])
	example[1] = *hasVotedNew
	newCurrentRounds := CurrentRounds[string, uint](
		example,
	)
	err = scale.Unmarshal(enc, &newCurrentRounds)
	require.NoError(t, err)
	require.Equal(t, currentRounds, newCurrentRounds)
}

func TestVoterSetStateEncoding(t *testing.T) {
	authorities := AuthoritySet[string, uint]{
		CurrentAuthorities:     []Authority{},
		SetID:                  1,
		PendingStandardChanges: NewChangeTree[string, uint](),
		PendingForcedChanges:   []PendingChange[string, uint]{},
		AuthoritySetChanges:    AuthoritySetChanges[uint]{},
	}

	dummyHashNumber := grandpa.HashNumber[string, uint]{
		Hash:   "a",
		Number: 1,
	}

	completedRound := &CompletedRound[string, uint]{
		Number: 1,
		State: grandpa.RoundState[string, uint]{
			PrevoteGHOST: &dummyHashNumber,
			Finalized:    &dummyHashNumber,
			Estimate:     &dummyHashNumber,
			Completable:  true,
		},
		Base: dummyHashNumber,
	}

	completedRounds := NewCompletedRounds[string, uint](completedRound, 1, authorities)
	currentRounds := make(map[uint64]HasVoted[string, uint])

	liveState := Live[string, uint]{
		CompletedRounds: completedRounds,
		CurrentRounds:   currentRounds,
	}

	voterSetState := *NewVoterSetState[string, uint]()
	err := voterSetState.Set(liveState)
	require.NoError(t, err)

	enc, err := scale.Marshal(voterSetState)
	require.NoError(t, err)

	newVoterSetState := *NewVoterSetState[string, uint]()
	err = scale.Unmarshal(enc, &newVoterSetState)
	require.NoError(t, err)

	oldVal, err := voterSetState.Value()
	require.NoError(t, err)

	newVal, err := newVoterSetState.Value()
	require.NoError(t, err)
	require.Equal(t, oldVal.(Live[string, uint]), newVal.(Live[string, uint]))
}

func TestVoterSetState_Live(t *testing.T) {
	authorities := AuthoritySet[string, uint]{
		CurrentAuthorities:     []Authority{},
		SetID:                  1,
		PendingStandardChanges: NewChangeTree[string, uint](),
		PendingForcedChanges:   []PendingChange[string, uint]{},
		AuthoritySetChanges:    AuthoritySetChanges[uint]{},
	}

	dummyHashNumber := grandpa.HashNumber[string, uint]{
		Hash:   "a",
		Number: 1,
	}

	voterSetState := NewVoterSetState[string, uint]()

	liveSetState, err := voterSetState.Live(5, authorities, dummyHashNumber)
	require.NoError(t, err)

	live, err := liveSetState.Value()
	require.NoError(t, err)

	val, ok := live.(Live[string, uint])
	require.True(t, ok)
	require.Equal(t, uint64(5), val.CompletedRounds.SetId)
	require.Equal(t, uint64(0), val.CompletedRounds.Rounds[0].Number)
}

func TestVoterSetState_CompletedRounds(t *testing.T) {
	authorities := AuthoritySet[string, uint]{
		CurrentAuthorities:     []Authority{},
		SetID:                  1,
		PendingStandardChanges: NewChangeTree[string, uint](),
		PendingForcedChanges:   []PendingChange[string, uint]{},
		AuthoritySetChanges:    AuthoritySetChanges[uint]{},
	}
	dummyHashNumber := grandpa.HashNumber[string, uint]{
		Hash:   "a",
		Number: 1,
	}
	state := grandpa.NewRoundState[string, uint](dummyHashNumber)
	completedRounds := NewCompletedRounds[string, uint](
		&CompletedRound[string, uint]{
			10,
			state,
			dummyHashNumber,
			[]grandpa.SignedMessage[string, uint, ed25519.SignatureBytes, ed25519.PublicKey]{},
		},
		5,
		authorities,
	)

	voterSetState := NewVoterSetState[string, uint]()

	err := voterSetState.Set(Live[string, uint]{
		CompletedRounds: completedRounds,
	})
	require.NoError(t, err)

	rounds, err := voterSetState.CompletedRounds()
	require.NoError(t, err)
	require.Equal(t, completedRounds, rounds)
}

func TestVoterSetState_LastCompletedRound(t *testing.T) {
	authorities := AuthoritySet[string, uint]{
		CurrentAuthorities:     []Authority{},
		SetID:                  1,
		PendingStandardChanges: NewChangeTree[string, uint](),
		PendingForcedChanges:   []PendingChange[string, uint]{},
		AuthoritySetChanges:    AuthoritySetChanges[uint]{},
	}
	dummyHashNumber := grandpa.HashNumber[string, uint]{
		Hash:   "a",
		Number: 1,
	}
	state := grandpa.NewRoundState[string, uint](dummyHashNumber)
	completedRounds := NewCompletedRounds[string, uint](
		&CompletedRound[string, uint]{
			10,
			state,
			dummyHashNumber,
			[]grandpa.SignedMessage[string, uint, ed25519.SignatureBytes, ed25519.PublicKey]{},
		},
		5,
		authorities,
	)

	addedCompletedRound := CompletedRound[string, uint]{
		8,
		state,
		dummyHashNumber,
		[]grandpa.SignedMessage[string, uint, ed25519.SignatureBytes, ed25519.PublicKey]{},
	}

	completedRounds.Push(addedCompletedRound)

	voterSetState := NewVoterSetState[string, uint]()
	err := voterSetState.Set(Paused[string, uint]{
		CompletedRounds: completedRounds,
	})
	require.NoError(t, err)

	lastCompletedRound, err := voterSetState.LastCompletedRound()
	require.NoError(t, err)
	require.Equal(t, addedCompletedRound, lastCompletedRound)
}

func TestVoterSetState_WithCurrentRound(t *testing.T) {
	authorities := AuthoritySet[string, uint]{
		CurrentAuthorities:     []Authority{},
		SetID:                  1,
		PendingStandardChanges: NewChangeTree[string, uint](),
		PendingForcedChanges:   []PendingChange[string, uint]{},
		AuthoritySetChanges:    AuthoritySetChanges[uint]{},
	}
	dummyHashNumber := grandpa.HashNumber[string, uint]{
		Hash:   "a",
		Number: 1,
	}
	state := grandpa.NewRoundState[string, uint](dummyHashNumber)
	completedRounds := NewCompletedRounds[string, uint](
		&CompletedRound[string, uint]{
			10,
			state,
			dummyHashNumber,
			[]grandpa.SignedMessage[string, uint, ed25519.SignatureBytes, ed25519.PublicKey]{},
		},
		5,
		authorities,
	)

	voterSetState := NewVoterSetState[string, uint]()

	// Paused
	err := voterSetState.Set(Paused[string, uint]{
		CompletedRounds: completedRounds,
	})
	require.NoError(t, err)
	_, _, err = voterSetState.WithCurrentRound(1)
	require.NotNil(t, err)
	require.Equal(t, "voter acting while in paused state", err.Error())

	// Live: invalid round
	err = voterSetState.Set(Live[string, uint]{
		CompletedRounds: completedRounds,
	})
	require.NoError(t, err)
	_, _, err = voterSetState.WithCurrentRound(1)
	require.NotNil(t, err)
	require.Equal(t, "voter acting on a live round we are not tracking", err.Error())

	// Valid
	currentRounds := CurrentRounds[string, uint](
		make(map[uint64]HasVoted[string, uint]),
	)

	hasVoted := &HasVoted[string, uint]{}
	hasVoted = hasVoted.New()
	err = hasVoted.Set(No{})
	require.NoError(t, err)
	currentRounds[1] = *hasVoted
	err = voterSetState.Set(Live[string, uint]{
		CompletedRounds: completedRounds,
		CurrentRounds:   currentRounds,
	})
	require.NoError(t, err)
	completed, current, err := voterSetState.WithCurrentRound(1)
	require.NoError(t, err)
	require.Equal(t, completedRounds, completed)
	require.Equal(t, currentRounds, current)
}

func TestHasVotedEncoding(t *testing.T) {
	pub, err := ed25519.NewPublicKey([]byte{1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1})
	require.NoError(t, err)

	vote := &Vote[string, uint]{}
	vote = vote.New()
	err = vote.Set(Propose[string, uint]{})
	require.NoError(t, err)

	yes := Yes[string, uint]{
		AuthId: *pub,
		Vote:   *vote,
	}
	hasVoted := &HasVoted[string, uint]{}
	hasVoted = hasVoted.New()
	err = hasVoted.Set(yes)
	require.NoError(t, err)

	res, err := scale.Marshal(hasVoted)
	require.NoError(t, err)

	newHasVoted := &HasVoted[string, uint]{}
	newHasVoted = hasVoted.New()
	err = scale.Unmarshal(res, &newHasVoted)
	require.NoError(t, err)
	require.Equal(t, hasVoted, newHasVoted)
}

func TestHasVoted_Propose(t *testing.T) {
	primaryPropose := &grandpa.PrimaryPropose[string, uint]{
		TargetHash:   "a",
		TargetNumber: 2,
	}
	vote := &Vote[string, uint]{}
	vote = vote.New()
	err := vote.Set(Propose[string, uint]{*primaryPropose})
	require.NoError(t, err)

	yes := Yes[string, uint]{
		Vote: *vote,
	}
	hasVoted := &HasVoted[string, uint]{}
	hasVoted = hasVoted.New()
	err = hasVoted.Set(yes)
	require.NoError(t, err)

	newPrimaryPropose := hasVoted.Propose()
	require.Equal(t, primaryPropose, newPrimaryPropose)
}

func TestHasVoted_Prevote(t *testing.T) {
	prevote := &grandpa.Prevote[string, uint]{
		TargetHash:   "a",
		TargetNumber: 2,
	}
	vote := &Vote[string, uint]{}
	vote = vote.New()
	err := vote.Set(Prevote[string, uint]{&grandpa.PrimaryPropose[string, uint]{}, *prevote})
	require.NoError(t, err)

	yes := Yes[string, uint]{
		Vote: *vote,
	}
	hasVoted := &HasVoted[string, uint]{}
	hasVoted = hasVoted.New()
	err = hasVoted.Set(yes)
	require.NoError(t, err)

	newPrevote := hasVoted.Prevote()
	require.Equal(t, prevote, newPrevote)

	primaryPropose := &grandpa.PrimaryPropose[string, uint]{
		TargetHash:   "a",
		TargetNumber: 2,
	}
	proposeVote := &Vote[string, uint]{}
	proposeVote = vote.New()
	err = proposeVote.Set(Propose[string, uint]{PrimaryPropose: *primaryPropose})
	require.NoError(t, err)

	yes = Yes[string, uint]{
		Vote: *proposeVote,
	}
	hasVoted = hasVoted.New()
	err = hasVoted.Set(yes)
	require.NoError(t, err)

	newPrevote = hasVoted.Prevote()
	require.Nil(t, newPrevote)
}

func TestHasVoted_Precommit(t *testing.T) {
	precommit := &grandpa.Precommit[string, uint]{
		TargetHash:   "a",
		TargetNumber: 2,
	}
	vote := &Vote[string, uint]{}
	vote = vote.New()
	err := vote.Set(Precommit[string, uint]{&grandpa.PrimaryPropose[string, uint]{}, grandpa.Prevote[string, uint]{}, *precommit})
	require.NoError(t, err)

	yes := Yes[string, uint]{
		Vote: *vote,
	}
	hasVoted := &HasVoted[string, uint]{}
	hasVoted = hasVoted.New()
	err = hasVoted.Set(yes)
	require.NoError(t, err)

	newCommit := hasVoted.Precommit()
	require.Equal(t, precommit, newCommit)

	primaryPropose := &grandpa.PrimaryPropose[string, uint]{
		TargetHash:   "a",
		TargetNumber: 2,
	}
	proposeVote := &Vote[string, uint]{}
	proposeVote = proposeVote.New()
	err = proposeVote.Set(Propose[string, uint]{PrimaryPropose: *primaryPropose})
	require.NoError(t, err)

	yes = Yes[string, uint]{
		Vote: *proposeVote,
	}
	hasVoted = hasVoted.New()
	err = hasVoted.Set(yes)
	require.NoError(t, err)

	newCommit = hasVoted.Precommit()
	require.Nil(t, newCommit)
}

func TestHasVoted_CanPropose(t *testing.T) {
	primaryPropose := &grandpa.PrimaryPropose[string, uint]{
		TargetHash:   "a",
		TargetNumber: 2,
	}
	vote := &Vote[string, uint]{}
	vote = vote.New()
	err := vote.Set(Propose[string, uint]{*primaryPropose})
	require.NoError(t, err)

	yes := Yes[string, uint]{
		Vote: *vote,
	}
	hasVoted := &HasVoted[string, uint]{}
	hasVoted = hasVoted.New()
	err = hasVoted.Set(yes)
	require.NoError(t, err)
	require.False(t, hasVoted.CanPropose())

	hasVoted = hasVoted.New()
	err = hasVoted.Set(No{})
	require.NoError(t, err)
	require.True(t, hasVoted.CanPropose())
}

func TestHasVoted_CanPrevote(t *testing.T) {
	prevote := &grandpa.Prevote[string, uint]{
		TargetHash:   "a",
		TargetNumber: 2,
	}
	vote := &Vote[string, uint]{}
	vote = vote.New()
	err := vote.Set(Prevote[string, uint]{&grandpa.PrimaryPropose[string, uint]{}, *prevote})
	require.NoError(t, err)

	yes := Yes[string, uint]{
		Vote: *vote,
	}
	hasVoted := &HasVoted[string, uint]{}
	hasVoted = hasVoted.New()
	err = hasVoted.Set(yes)
	require.NoError(t, err)
	require.False(t, hasVoted.CanPrevote())

	hasVoted = hasVoted.New()
	err = hasVoted.Set(No{})
	require.NoError(t, err)
	require.True(t, hasVoted.CanPrevote())
}

func TestHasVoted_CanPrecommit(t *testing.T) {
	precommit := &grandpa.Precommit[string, uint]{
		TargetHash:   "a",
		TargetNumber: 2,
	}
	vote := &Vote[string, uint]{}
	vote = vote.New()
	err := vote.Set(Precommit[string, uint]{&grandpa.PrimaryPropose[string, uint]{}, grandpa.Prevote[string, uint]{}, *precommit})
	require.NoError(t, err)

	yes := Yes[string, uint]{
		Vote: *vote,
	}
	hasVoted := &HasVoted[string, uint]{}
	hasVoted = hasVoted.New()
	err = hasVoted.Set(yes)
	require.NoError(t, err)
	require.False(t, hasVoted.CanPrecommit())

	hasVoted = hasVoted.New()
	err = hasVoted.Set(No{})
	require.NoError(t, err)
	require.True(t, hasVoted.CanPrecommit())
}

func TestVoteEncoding(t *testing.T) {
	vote := &Vote[string, uint]{}
	vote = vote.New()
	err := vote.Set(Propose[string, uint]{
		PrimaryPropose: grandpa.PrimaryPropose[string, uint]{
			TargetHash:   "a",
			TargetNumber: 1,
		},
	})
	require.NoError(t, err)

	enc, err := scale.Marshal(vote)
	require.NoError(t, err)

	newVote := &Vote[string, uint]{}
	newVote = newVote.New()
	err = scale.Unmarshal(enc, &newVote)
	require.NoError(t, err)
	require.Equal(t, vote, newVote)
}

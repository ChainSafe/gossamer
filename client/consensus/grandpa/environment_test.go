package grandpa

import (
	"github.com/ChainSafe/gossamer/lib/crypto/ed25519"
	grandpa "github.com/ChainSafe/gossamer/pkg/finality-grandpa"
	"github.com/ChainSafe/gossamer/pkg/scale"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestCompleteRoundEncoding(t *testing.T) {
	dummyHashNumber := grandpa.HashNumber[Hash, uint]{
		Hash:   bytesToHash([]byte{1}),
		Number: 1,
	}

	completedRound := CompletedRound[Hash, uint]{
		Number: 1,
		State: grandpa.RoundState[Hash, uint]{
			PrevoteGHOST: &dummyHashNumber,
			Finalized:    &dummyHashNumber,
			Estimate:     &dummyHashNumber,
			Completable:  true,
		},
		Base: dummyHashNumber,
	}

	enc, err := scale.Marshal(completedRound)
	require.NoError(t, err)

	newCompletedRound := CompletedRound[Hash, uint]{}
	err = scale.Unmarshal(enc, &newCompletedRound)
	require.NoError(t, err)
	require.Equal(t, completedRound, newCompletedRound)
}

func TestCompletedRoundsEncoding(t *testing.T) {
	authorities := AuthoritySet[Hash, uint]{
		CurrentAuthorities:     AuthorityList{},
		SetID:                  1,
		PendingStandardChanges: NewChangeTree[Hash, uint](),
		PendingForcedChanges:   []PendingChange[Hash, uint]{},
		AuthoritySetChanges:    AuthoritySetChanges[uint]{},
	}

	dummyHashNumber := grandpa.HashNumber[Hash, uint]{
		Hash:   bytesToHash([]byte{1}),
		Number: 1,
	}

	completedRound := &CompletedRound[Hash, uint]{
		Number: 1,
		State: grandpa.RoundState[Hash, uint]{
			PrevoteGHOST: &dummyHashNumber,
			Finalized:    &dummyHashNumber,
			Estimate:     &dummyHashNumber,
			Completable:  true,
		},
		Base: dummyHashNumber,
	}

	completedRounds := NewCompletedRounds[Hash, uint](completedRound, 1, authorities)
	enc, err := scale.Marshal(completedRounds)
	require.NoError(t, err)

	var newCompletedRounds CompletedRounds[Hash, uint]
	err = scale.Unmarshal(enc, &newCompletedRounds)
	require.NoError(t, err)
	require.Equal(t, completedRounds, newCompletedRounds)
}

func TestCompletedRounds_Last(t *testing.T) {
	authorities := AuthoritySet[Hash, uint]{
		CurrentAuthorities:     AuthorityList{},
		SetID:                  1,
		PendingStandardChanges: NewChangeTree[Hash, uint](),
		PendingForcedChanges:   []PendingChange[Hash, uint]{},
		AuthoritySetChanges:    AuthoritySetChanges[uint]{},
	}

	dummyHashNumber := grandpa.HashNumber[Hash, uint]{
		Hash:   bytesToHash([]byte{1}),
		Number: 1,
	}

	completedRound := CompletedRound[Hash, uint]{
		Number: 1,
		State: grandpa.RoundState[Hash, uint]{
			PrevoteGHOST: &dummyHashNumber,
			Finalized:    &dummyHashNumber,
			Estimate:     &dummyHashNumber,
			Completable:  true,
		},
		Base: dummyHashNumber,
	}
	completedRounds := NewCompletedRounds[Hash, uint](&completedRound, 1, authorities)
	lastCompletedRound := completedRounds.Last()
	require.Equal(t, completedRound, lastCompletedRound)

	completedRounds = NewCompletedRounds[Hash, uint](nil, 1, authorities)
	require.Panics(t, func() { completedRounds.Last() }, "Last did not panic")
}

func TestCompletedRounds_Push(t *testing.T) {
	authorities := AuthoritySet[Hash, uint]{
		CurrentAuthorities:     AuthorityList{},
		SetID:                  1,
		PendingStandardChanges: NewChangeTree[Hash, uint](),
		PendingForcedChanges:   []PendingChange[Hash, uint]{},
		AuthoritySetChanges:    AuthoritySetChanges[uint]{},
	}

	dummyHashNumber := grandpa.HashNumber[Hash, uint]{
		Hash:   bytesToHash([]byte{1}),
		Number: 1,
	}

	completedRound0 := CompletedRound[Hash, uint]{
		Number: 1,
		State: grandpa.RoundState[Hash, uint]{
			PrevoteGHOST: &dummyHashNumber,
			Finalized:    &dummyHashNumber,
			Estimate:     &dummyHashNumber,
			Completable:  true,
		},
		Base: dummyHashNumber,
	}

	completedRound1 := CompletedRound[Hash, uint]{
		Number: 1,
		State: grandpa.RoundState[Hash, uint]{
			PrevoteGHOST: &dummyHashNumber,
			Finalized:    &dummyHashNumber,
			Estimate:     &dummyHashNumber,
			Completable:  true,
		},
		Base: dummyHashNumber,
	}

	completedRound2 := CompletedRound[Hash, uint]{
		Number: 1,
		State: grandpa.RoundState[Hash, uint]{
			PrevoteGHOST: &dummyHashNumber,
			Finalized:    &dummyHashNumber,
			Estimate:     &dummyHashNumber,
			Completable:  true,
		},
		Base: dummyHashNumber,
	}
	completedRounds := NewCompletedRounds[Hash, uint](&completedRound1, 1, authorities)
	completedRounds.Push(completedRound0)

	lastCompletedRound := completedRounds.Last()
	require.Equal(t, completedRound0, lastCompletedRound)

	completedRounds.Push(completedRound2)
	lastCompletedRound = completedRounds.Last()
	require.Equal(t, completedRound0, lastCompletedRound)
}

func TestCurrentRoundsEncoding(t *testing.T) {
	// TODO this fails
	currentRounds := CurrentRounds[Hash, uint](
		make(map[uint64]HasVoted[Hash, uint]),
	)

	hasVoted := &HasVoted[Hash, uint]{}
	hasVoted = hasVoted.New()
	err := hasVoted.Set(No{})
	require.NoError(t, err)
	currentRounds[1] = *hasVoted

	enc, err := scale.Marshal(currentRounds)
	require.NoError(t, err)

	hasVotedNew := &HasVoted[Hash, uint]{}
	hasVotedNew = hasVoted.New()
	example := make(map[uint64]HasVoted[Hash, uint])
	example[1] = *hasVotedNew
	newCurrentRounds := CurrentRounds[Hash, uint](
		example,
	)
	err = scale.Unmarshal(enc, &newCurrentRounds)
	require.NoError(t, err)
	require.Equal(t, currentRounds, newCurrentRounds)
}

func TestVoterSetStateEncoding(t *testing.T) {
	authorities := AuthoritySet[Hash, uint]{
		CurrentAuthorities:     AuthorityList{},
		SetID:                  1,
		PendingStandardChanges: NewChangeTree[Hash, uint](),
		PendingForcedChanges:   []PendingChange[Hash, uint]{},
		AuthoritySetChanges:    AuthoritySetChanges[uint]{},
	}

	dummyHashNumber := grandpa.HashNumber[Hash, uint]{
		Hash:   bytesToHash([]byte{1}),
		Number: 1,
	}

	completedRound := &CompletedRound[Hash, uint]{
		Number: 1,
		State: grandpa.RoundState[Hash, uint]{
			PrevoteGHOST: &dummyHashNumber,
			Finalized:    &dummyHashNumber,
			Estimate:     &dummyHashNumber,
			Completable:  true,
		},
		Base: dummyHashNumber,
	}

	completedRounds := NewCompletedRounds[Hash, uint](completedRound, 1, authorities)
	currentRounds := make(map[uint64]HasVoted[Hash, uint])

	liveState := Live[Hash, uint]{
		CompletedRounds: completedRounds,
		CurrentRounds:   currentRounds,
	}

	voterSetState := *NewVoterSetState[Hash, uint]()
	err := voterSetState.Set(liveState)
	require.NoError(t, err)

	enc, err := scale.Marshal(voterSetState)
	require.NoError(t, err)

	newVoterSetState := *NewVoterSetState[Hash, uint]()
	err = scale.Unmarshal(enc, &newVoterSetState)
	require.NoError(t, err)

	oldVal, err := voterSetState.Value()
	require.NoError(t, err)

	newVal, err := newVoterSetState.Value()
	require.NoError(t, err)
	require.Equal(t, oldVal.(Live[Hash, uint]), newVal.(Live[Hash, uint]))
}

func TestVoterSetState_Live(t *testing.T) {
	authorities := AuthoritySet[Hash, uint]{
		CurrentAuthorities:     AuthorityList{},
		SetID:                  1,
		PendingStandardChanges: NewChangeTree[Hash, uint](),
		PendingForcedChanges:   []PendingChange[Hash, uint]{},
		AuthoritySetChanges:    AuthoritySetChanges[uint]{},
	}

	dummyHashNumber := grandpa.HashNumber[Hash, uint]{
		Hash:   bytesToHash([]byte{1}),
		Number: 1,
	}

	voterSetState := NewVoterSetState[Hash, uint]()

	liveSetState, err := voterSetState.Live(5, authorities, dummyHashNumber)
	require.NoError(t, err)

	live, err := liveSetState.Value()
	require.NoError(t, err)

	val, ok := live.(Live[Hash, uint])
	require.True(t, ok)
	require.Equal(t, uint64(5), val.CompletedRounds.SetId)
	require.Equal(t, uint64(0), val.CompletedRounds.Rounds[0].Number)
}

func TestVoterSetState_CompletedRounds(t *testing.T) {
	authorities := AuthoritySet[Hash, uint]{
		CurrentAuthorities:     AuthorityList{},
		SetID:                  1,
		PendingStandardChanges: NewChangeTree[Hash, uint](),
		PendingForcedChanges:   []PendingChange[Hash, uint]{},
		AuthoritySetChanges:    AuthoritySetChanges[uint]{},
	}
	dummyHashNumber := grandpa.HashNumber[Hash, uint]{
		Hash:   bytesToHash([]byte{1}),
		Number: 1,
	}
	state := grandpa.NewRoundState[Hash, uint](dummyHashNumber)
	completedRounds := NewCompletedRounds[Hash, uint](
		&CompletedRound[Hash, uint]{
			10,
			state,
			dummyHashNumber,
			[]grandpa.SignedMessage[Hash, uint, ed25519.SignatureBytes, ed25519.PublicKey]{},
		},
		5,
		authorities,
	)

	voterSetState := NewVoterSetState[Hash, uint]()

	err := voterSetState.Set(Live[Hash, uint]{
		CompletedRounds: completedRounds,
	})
	require.NoError(t, err)

	rounds, err := voterSetState.CompletedRounds()
	require.NoError(t, err)
	require.Equal(t, completedRounds, rounds)
}

func TestVoterSetState_LastCompletedRound(t *testing.T) {
	authorities := AuthoritySet[Hash, uint]{
		CurrentAuthorities:     AuthorityList{},
		SetID:                  1,
		PendingStandardChanges: NewChangeTree[Hash, uint](),
		PendingForcedChanges:   []PendingChange[Hash, uint]{},
		AuthoritySetChanges:    AuthoritySetChanges[uint]{},
	}
	dummyHashNumber := grandpa.HashNumber[Hash, uint]{
		Hash:   bytesToHash([]byte{1}),
		Number: 1,
	}
	state := grandpa.NewRoundState[Hash, uint](dummyHashNumber)
	completedRounds := NewCompletedRounds[Hash, uint](
		&CompletedRound[Hash, uint]{
			10,
			state,
			dummyHashNumber,
			[]grandpa.SignedMessage[Hash, uint, ed25519.SignatureBytes, ed25519.PublicKey]{},
		},
		5,
		authorities,
	)

	addedCompletedRound := CompletedRound[Hash, uint]{
		8,
		state,
		dummyHashNumber,
		[]grandpa.SignedMessage[Hash, uint, ed25519.SignatureBytes, ed25519.PublicKey]{},
	}

	completedRounds.Push(addedCompletedRound)

	voterSetState := NewVoterSetState[Hash, uint]()
	err := voterSetState.Set(Paused[Hash, uint]{
		CompletedRounds: completedRounds,
	})
	require.NoError(t, err)

	lastCompletedRound, err := voterSetState.LastCompletedRound()
	require.NoError(t, err)
	require.Equal(t, addedCompletedRound, lastCompletedRound)
}

func TestVoterSetState_WithCurrentRound(t *testing.T) {
	authorities := AuthoritySet[Hash, uint]{
		CurrentAuthorities:     AuthorityList{},
		SetID:                  1,
		PendingStandardChanges: NewChangeTree[Hash, uint](),
		PendingForcedChanges:   []PendingChange[Hash, uint]{},
		AuthoritySetChanges:    AuthoritySetChanges[uint]{},
	}
	dummyHashNumber := grandpa.HashNumber[Hash, uint]{
		Hash:   bytesToHash([]byte{1}),
		Number: 1,
	}
	state := grandpa.NewRoundState[Hash, uint](dummyHashNumber)
	completedRounds := NewCompletedRounds[Hash, uint](
		&CompletedRound[Hash, uint]{
			10,
			state,
			dummyHashNumber,
			[]grandpa.SignedMessage[Hash, uint, ed25519.SignatureBytes, ed25519.PublicKey]{},
		},
		5,
		authorities,
	)

	voterSetState := NewVoterSetState[Hash, uint]()

	// Paused
	err := voterSetState.Set(Paused[Hash, uint]{
		CompletedRounds: completedRounds,
	})
	require.NoError(t, err)
	_, _, err = voterSetState.WithCurrentRound(1)
	require.NotNil(t, err)
	require.Equal(t, "voter acting while in paused state", err.Error())

	// Live: invalid round
	err = voterSetState.Set(Live[Hash, uint]{
		CompletedRounds: completedRounds,
	})
	require.NoError(t, err)
	_, _, err = voterSetState.WithCurrentRound(1)
	require.NotNil(t, err)
	require.Equal(t, "voter acting on a live round we are not tracking", err.Error())

	// Valid
	currentRounds := CurrentRounds[Hash, uint](
		make(map[uint64]HasVoted[Hash, uint]),
	)

	hasVoted := &HasVoted[Hash, uint]{}
	hasVoted = hasVoted.New()
	err = hasVoted.Set(No{})
	require.NoError(t, err)
	currentRounds[1] = *hasVoted
	err = voterSetState.Set(Live[Hash, uint]{
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

	vote := &Vote[Hash, uint]{}
	vote = vote.New()
	err = vote.Set(Propose[Hash, uint]{})
	require.NoError(t, err)

	yes := Yes[Hash, uint]{
		AuthId: *pub,
		Vote:   *vote,
	}
	hasVoted := &HasVoted[Hash, uint]{}
	hasVoted = hasVoted.New()
	err = hasVoted.Set(yes)
	require.NoError(t, err)

	res, err := scale.Marshal(hasVoted)
	require.NoError(t, err)

	newHasVoted := &HasVoted[Hash, uint]{}
	newHasVoted = hasVoted.New()
	err = scale.Unmarshal(res, &newHasVoted)
	require.NoError(t, err)
	require.Equal(t, hasVoted, newHasVoted)
}

func TestHasVoted_Propose(t *testing.T) {
	primaryPropose := &grandpa.PrimaryPropose[Hash, uint]{
		TargetHash:   bytesToHash([]byte{1}),
		TargetNumber: 2,
	}
	vote := &Vote[Hash, uint]{}
	vote = vote.New()
	err := vote.Set(Propose[Hash, uint]{*primaryPropose})
	require.NoError(t, err)

	yes := Yes[Hash, uint]{
		Vote: *vote,
	}
	hasVoted := &HasVoted[Hash, uint]{}
	hasVoted = hasVoted.New()
	err = hasVoted.Set(yes)
	require.NoError(t, err)

	newPrimaryPropose := hasVoted.Propose()
	require.Equal(t, primaryPropose, newPrimaryPropose)
}

func TestHasVoted_Prevote(t *testing.T) {
	prevote := &grandpa.Prevote[Hash, uint]{
		TargetHash:   bytesToHash([]byte{1}),
		TargetNumber: 2,
	}
	vote := &Vote[Hash, uint]{}
	vote = vote.New()
	err := vote.Set(Prevote[Hash, uint]{&grandpa.PrimaryPropose[Hash, uint]{}, *prevote})
	require.NoError(t, err)

	yes := Yes[Hash, uint]{
		Vote: *vote,
	}
	hasVoted := &HasVoted[Hash, uint]{}
	hasVoted = hasVoted.New()
	err = hasVoted.Set(yes)
	require.NoError(t, err)

	newPrevote := hasVoted.Prevote()
	require.Equal(t, prevote, newPrevote)

	primaryPropose := &grandpa.PrimaryPropose[Hash, uint]{
		TargetHash:   bytesToHash([]byte{1}),
		TargetNumber: 2,
	}
	proposeVote := &Vote[Hash, uint]{}
	proposeVote = vote.New()
	err = proposeVote.Set(Propose[Hash, uint]{PrimaryPropose: *primaryPropose})
	require.NoError(t, err)

	yes = Yes[Hash, uint]{
		Vote: *proposeVote,
	}
	hasVoted = hasVoted.New()
	err = hasVoted.Set(yes)
	require.NoError(t, err)

	newPrevote = hasVoted.Prevote()
	require.Nil(t, newPrevote)
}

func TestHasVoted_Precommit(t *testing.T) {
	precommit := &grandpa.Precommit[Hash, uint]{
		TargetHash:   bytesToHash([]byte{1}),
		TargetNumber: 2,
	}
	vote := &Vote[Hash, uint]{}
	vote = vote.New()
	err := vote.Set(Precommit[Hash, uint]{&grandpa.PrimaryPropose[Hash, uint]{}, grandpa.Prevote[Hash, uint]{}, *precommit})
	require.NoError(t, err)

	yes := Yes[Hash, uint]{
		Vote: *vote,
	}
	hasVoted := &HasVoted[Hash, uint]{}
	hasVoted = hasVoted.New()
	err = hasVoted.Set(yes)
	require.NoError(t, err)

	newCommit := hasVoted.Precommit()
	require.Equal(t, precommit, newCommit)

	primaryPropose := &grandpa.PrimaryPropose[Hash, uint]{
		TargetHash:   bytesToHash([]byte{1}),
		TargetNumber: 2,
	}
	proposeVote := &Vote[Hash, uint]{}
	proposeVote = proposeVote.New()
	err = proposeVote.Set(Propose[Hash, uint]{PrimaryPropose: *primaryPropose})
	require.NoError(t, err)

	yes = Yes[Hash, uint]{
		Vote: *proposeVote,
	}
	hasVoted = hasVoted.New()
	err = hasVoted.Set(yes)
	require.NoError(t, err)

	newCommit = hasVoted.Precommit()
	require.Nil(t, newCommit)
}

func TestHasVoted_CanPropose(t *testing.T) {
	primaryPropose := &grandpa.PrimaryPropose[Hash, uint]{
		TargetHash:   bytesToHash([]byte{1}),
		TargetNumber: 2,
	}
	vote := &Vote[Hash, uint]{}
	vote = vote.New()
	err := vote.Set(Propose[Hash, uint]{*primaryPropose})
	require.NoError(t, err)

	yes := Yes[Hash, uint]{
		Vote: *vote,
	}
	hasVoted := &HasVoted[Hash, uint]{}
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
	prevote := &grandpa.Prevote[Hash, uint]{
		TargetHash:   bytesToHash([]byte{1}),
		TargetNumber: 2,
	}
	vote := &Vote[Hash, uint]{}
	vote = vote.New()
	err := vote.Set(Prevote[Hash, uint]{&grandpa.PrimaryPropose[Hash, uint]{}, *prevote})
	require.NoError(t, err)

	yes := Yes[Hash, uint]{
		Vote: *vote,
	}
	hasVoted := &HasVoted[Hash, uint]{}
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
	precommit := &grandpa.Precommit[Hash, uint]{
		TargetHash:   bytesToHash([]byte{1}),
		TargetNumber: 2,
	}
	vote := &Vote[Hash, uint]{}
	vote = vote.New()
	err := vote.Set(Precommit[Hash, uint]{&grandpa.PrimaryPropose[Hash, uint]{}, grandpa.Prevote[Hash, uint]{}, *precommit})
	require.NoError(t, err)

	yes := Yes[Hash, uint]{
		Vote: *vote,
	}
	hasVoted := &HasVoted[Hash, uint]{}
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
	vote := &Vote[Hash, uint]{}
	vote = vote.New()
	err := vote.Set(Propose[Hash, uint]{
		PrimaryPropose: grandpa.PrimaryPropose[Hash, uint]{
			TargetHash:   bytesToHash([]byte{1}),
			TargetNumber: 1,
		},
	})
	require.NoError(t, err)

	enc, err := scale.Marshal(vote)
	require.NoError(t, err)

	newVote := &Vote[Hash, uint]{}
	newVote = newVote.New()
	err = scale.Unmarshal(enc, &newVote)
	require.NoError(t, err)
	require.Equal(t, vote, newVote)
}

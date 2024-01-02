// Copyright 2023 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package grandpa

// func TestSharedVoterSetState_hasVoted(t *testing.T) {
// 	// Has Not Voted
// 	hasNotVoted := hasVoted[string, uint, dummyAuthID]{}
// 	hasNotVoted = hasNotVoted.New()
// 	err := hasNotVoted.Set(no{})
// 	require.NoError(t, err)
// 	voterSetState := *NewVoterSetState[string, uint, dummyAuthID, uint]()
// 	sharedVoterSetState := NewSharedVoterSetState[string, uint](voterSetState)
// 	voted, err := sharedVoterSetState.hasVoted(0)
// 	require.NoError(t, err)
// 	require.Equal(t, hasNotVoted, voted)

// 	// Has Voted
// 	vote := vote[string, uint]{}
// 	vote = vote.New()
// 	err = vote.Set(propose[string, uint]{})
// 	require.NoError(t, err)

// 	yes := yes[string, uint, dummyAuthID]{
// 		AuthId: key,
// 		Vote:   vote,
// 	}

// 	hasIndeedVoted := hasVoted[string, uint, dummyAuthID]{}
// 	hasIndeedVoted = hasIndeedVoted.New()
// 	err = hasIndeedVoted.Set(yes)
// 	require.NoError(t, err)

// 	example := make(map[uint64]hasVoted[string, uint, dummyAuthID])
// 	example[1] = hasIndeedVoted
// 	newCurrentRounds := CurrentRounds[string, uint, dummyAuthID](
// 		example,
// 	)
// 	liveState := voterSetStateLive[string, uint, dummyAuthID, uint]{
// 		CurrentRounds: newCurrentRounds,
// 	}

// 	newVoterSetState := *NewVoterSetState[string, uint, dummyAuthID, uint]()
// 	err = newVoterSetState.Set(liveState)
// 	require.NoError(t, err)

// 	sharedVoterSetState = NewSharedVoterSetState[string, uint](newVoterSetState)
// 	voted, err = sharedVoterSetState.hasVoted(1)
// 	require.NoError(t, err)
// 	require.Equal(t, hasIndeedVoted, voted)
// }

// func TestCompleteRoundEncoding(t *testing.T) {
// 	dummyHashNumber := grandpa.HashNumber[string, uint]{
// 		Hash:   "a",
// 		Number: 1,
// 	}

// 	compRound := completedRound[string, uint, dummyAuthID, uint]{
// 		Number: 1,
// 		State: grandpa.RoundState[string, uint]{
// 			PrevoteGHOST: &dummyHashNumber,
// 			Finalized:    &dummyHashNumber,
// 			Estimate:     &dummyHashNumber,
// 			Completable:  true,
// 		},
// 		Base: dummyHashNumber,
// 	}

// 	enc, err := scale.Marshal(compRound)
// 	require.NoError(t, err)

// 	newCompletedRound := completedRound[string, uint, dummyAuthID, uint]{}
// 	err = scale.Unmarshal(enc, &newCompletedRound)
// 	require.NoError(t, err)
// 	require.Equal(t, compRound, newCompletedRound)
// }

// func TestCompletedRoundsEncoding(t *testing.T) {
// 	authorities := AuthoritySet[string, uint, dummyAuthID]{
// 		CurrentAuthorities:     []Authority[dummyAuthID]{},
// 		SetID:                  1,
// 		PendingStandardChanges: NewChangeTree[string, uint, dummyAuthID](),
// 		PendingForcedChanges:   []PendingChange[string, uint, dummyAuthID]{},
// 		AuthoritySetChanges:    AuthoritySetChanges[uint]{},
// 	}

// 	dummyHashNumber := grandpa.HashNumber[string, uint]{
// 		Hash:   "a",
// 		Number: 1,
// 	}

// 	completedRound := completedRound[string, uint, dummyAuthID, uint]{
// 		Number: 1,
// 		State: grandpa.RoundState[string, uint]{
// 			PrevoteGHOST: &dummyHashNumber,
// 			Finalized:    &dummyHashNumber,
// 			Estimate:     &dummyHashNumber,
// 			Completable:  true,
// 		},
// 		Base: dummyHashNumber,
// 	}

// 	compRounds := NewCompletedRounds[string, uint](completedRound, 1, authorities)
// 	enc, err := scale.Marshal(compRounds)
// 	require.NoError(t, err)

// 	var newCompletedRounds completedRounds[string, uint, dummyAuthID, uint]
// 	err = scale.Unmarshal(enc, &newCompletedRounds)
// 	require.NoError(t, err)
// 	require.Equal(t, compRounds, newCompletedRounds)
// }

// func TestCompletedRounds_Iter(t *testing.T) {
// 	dummyHashNumber := grandpa.HashNumber[string, uint]{
// 		Hash:   "a",
// 		Number: 1,
// 	}

// 	completedRound0 := completedRound[string, uint, dummyAuthID, uint]{
// 		Number: 0,
// 		State: grandpa.RoundState[string, uint]{
// 			PrevoteGHOST: &dummyHashNumber,
// 			Finalized:    &dummyHashNumber,
// 			Estimate:     &dummyHashNumber,
// 			Completable:  true,
// 		},
// 		Base: dummyHashNumber,
// 	}

// 	completedRound1 := completedRound[string, uint, dummyAuthID, uint]{
// 		Number: 1,
// 		State: grandpa.RoundState[string, uint]{
// 			PrevoteGHOST: &dummyHashNumber,
// 			Finalized:    &dummyHashNumber,
// 			Estimate:     &dummyHashNumber,
// 			Completable:  true,
// 		},
// 		Base: dummyHashNumber,
// 	}

// 	completedRound2 := completedRound[string, uint, dummyAuthID, uint]{
// 		Number: 2,
// 		State: grandpa.RoundState[string, uint]{
// 			PrevoteGHOST: &dummyHashNumber,
// 			Finalized:    &dummyHashNumber,
// 			Estimate:     &dummyHashNumber,
// 			Completable:  true,
// 		},
// 		Base: dummyHashNumber,
// 	}
// 	rounds := make([]completedRound[string, uint, dummyAuthID, uint], 0, 3)
// 	rounds = append(rounds, completedRound0)
// 	rounds = append(rounds, completedRound1)
// 	rounds = append(rounds, completedRound2)

// 	expRounds := make([]completedRound[string, uint, dummyAuthID, uint], 0, 3)
// 	expRounds = append(expRounds, completedRound2)
// 	expRounds = append(expRounds, completedRound1)
// 	expRounds = append(expRounds, completedRound0)

// 	compRounds := completedRounds[string, uint, dummyAuthID, uint]{
// 		Rounds: rounds,
// 	}

// 	revRounds := compRounds.iter()
// 	require.Equal(t, expRounds, revRounds)
// }

// func TestCompletedRounds_Last(t *testing.T) {
// 	authorities := AuthoritySet[string, uint, dummyAuthID]{
// 		CurrentAuthorities:     []Authority[dummyAuthID]{},
// 		SetID:                  1,
// 		PendingStandardChanges: NewChangeTree[string, uint, dummyAuthID](),
// 		PendingForcedChanges:   []PendingChange[string, uint, dummyAuthID]{},
// 		AuthoritySetChanges:    AuthoritySetChanges[uint]{},
// 	}

// 	dummyHashNumber := grandpa.HashNumber[string, uint]{
// 		Hash:   "a",
// 		Number: 1,
// 	}

// 	compRound := completedRound[string, uint, dummyAuthID, uint]{
// 		Number: 1,
// 		State: grandpa.RoundState[string, uint]{
// 			PrevoteGHOST: &dummyHashNumber,
// 			Finalized:    &dummyHashNumber,
// 			Estimate:     &dummyHashNumber,
// 			Completable:  true,
// 		},
// 		Base: dummyHashNumber,
// 	}
// 	compRounds := NewCompletedRounds[string, uint](compRound, 1, authorities)
// 	lastCompletedRound := compRounds.last()
// 	require.Equal(t, compRound, lastCompletedRound)

// 	emptyCompletedRounds := completedRounds[string, uint, dummyAuthID, uint]{}
// 	require.Panics(t, func() { emptyCompletedRounds.last() }, "last did not panic")
// }

// func TestCompletedRounds_Push(t *testing.T) {
// 	authorities := AuthoritySet[string, uint, dummyAuthID]{
// 		CurrentAuthorities:     []Authority[dummyAuthID]{},
// 		SetID:                  1,
// 		PendingStandardChanges: NewChangeTree[string, uint, dummyAuthID](),
// 		PendingForcedChanges:   []PendingChange[string, uint, dummyAuthID]{},
// 		AuthoritySetChanges:    AuthoritySetChanges[uint]{},
// 	}

// 	dummyHashNumber := grandpa.HashNumber[string, uint]{
// 		Hash:   "a",
// 		Number: 1,
// 	}

// 	completedRound0 := completedRound[string, uint, dummyAuthID, uint]{
// 		Number: 0,
// 		State: grandpa.RoundState[string, uint]{
// 			PrevoteGHOST: &dummyHashNumber,
// 			Finalized:    &dummyHashNumber,
// 			Estimate:     &dummyHashNumber,
// 			Completable:  true,
// 		},
// 		Base: dummyHashNumber,
// 	}

// 	completedRound1 := completedRound[string, uint, dummyAuthID, uint]{
// 		Number: 1,
// 		State: grandpa.RoundState[string, uint]{
// 			PrevoteGHOST: &dummyHashNumber,
// 			Finalized:    &dummyHashNumber,
// 			Estimate:     &dummyHashNumber,
// 			Completable:  true,
// 		},
// 		Base: dummyHashNumber,
// 	}

// 	completedRound2 := completedRound[string, uint, dummyAuthID, uint]{
// 		Number: 2,
// 		State: grandpa.RoundState[string, uint]{
// 			PrevoteGHOST: &dummyHashNumber,
// 			Finalized:    &dummyHashNumber,
// 			Estimate:     &dummyHashNumber,
// 			Completable:  true,
// 		},
// 		Base: dummyHashNumber,
// 	}
// 	completedRounds := NewCompletedRounds[string, uint](completedRound1, 1, authorities)
// 	completedRounds.push(completedRound0)

// 	lastCompletedRound := completedRounds.last()
// 	require.Equal(t, completedRound1, lastCompletedRound)

// 	completedRounds.push(completedRound2)
// 	lastCompletedRound = completedRounds.last()
// 	require.Equal(t, completedRound2, lastCompletedRound)
// }

// func TestCurrentRoundsEncoding(t *testing.T) {
// 	currentRounds := CurrentRounds[string, uint, dummyAuthID](
// 		make(map[uint64]hasVoted[string, uint, dummyAuthID]),
// 	)

// 	hv := hasVoted[string, uint, dummyAuthID]{}
// 	hv = hv.New()
// 	err := hv.Set(no{})
// 	require.NoError(t, err)
// 	currentRounds[1] = hv

// 	enc, err := scale.Marshal(currentRounds)
// 	require.NoError(t, err)

// 	hasVotedNew := hasVoted[string, uint, dummyAuthID]{}
// 	hasVotedNew = hv.New()
// 	example := make(map[uint64]hasVoted[string, uint, dummyAuthID])
// 	example[1] = hasVotedNew
// 	newCurrentRounds := CurrentRounds[string, uint, dummyAuthID](
// 		example,
// 	)
// 	err = scale.Unmarshal(enc, &newCurrentRounds)
// 	require.NoError(t, err)
// 	require.Equal(t, currentRounds, newCurrentRounds)
// }

// func TestVoterSetStateEncoding(t *testing.T) {
// 	authorities := AuthoritySet[string, uint, dummyAuthID]{}

// 	dummyHashNumber := grandpa.HashNumber[string, uint]{
// 		Hash:   "a",
// 		Number: 1,
// 	}

// 	compRound := completedRound[string, uint, dummyAuthID, uint]{
// 		Number: 1,
// 		State: grandpa.RoundState[string, uint]{
// 			PrevoteGHOST: &dummyHashNumber,
// 			Finalized:    &dummyHashNumber,
// 			Estimate:     &dummyHashNumber,
// 			Completable:  true,
// 		},
// 		Base: dummyHashNumber,
// 	}

// 	completedRounds := NewCompletedRounds[string, uint, dummyAuthID, uint](compRound, 1, authorities)
// 	currentRounds := CurrentRounds[string, uint, dummyAuthID](
// 		make(map[uint64]hasVoted[string, uint, dummyAuthID]),
// 	)

// 	liveState := voterSetStateLive[string, uint, dummyAuthID, uint]{
// 		CompletedRounds: completedRounds,
// 		CurrentRounds:   currentRounds,
// 	}

// 	voterSetState := *NewVoterSetState[string, uint, dummyAuthID, uint]()
// 	err := voterSetState.Set(liveState)
// 	require.NoError(t, err)

// 	enc, err := scale.Marshal(voterSetState)
// 	require.NoError(t, err)

// 	newVoterSetState := *NewVoterSetState[string, uint, dummyAuthID, uint]()
// 	err = scale.Unmarshal(enc, &newVoterSetState)
// 	require.NoError(t, err)

// 	oldVal, err := voterSetState.Value()
// 	require.NoError(t, err)

// 	newVal, err := newVoterSetState.Value()
// 	require.NoError(t, err)
// 	require.Equal(t, oldVal.(voterSetStateLive[string, uint, dummyAuthID, uint]),
// 		newVal.(voterSetStateLive[string, uint, dummyAuthID, uint]))
// }

// func TestVoterSetState_Live(t *testing.T) {
// 	authorities := AuthoritySet[string, uint, dummyAuthID]{
// 		CurrentAuthorities:     []Authority[dummyAuthID]{},
// 		SetID:                  1,
// 		PendingStandardChanges: NewChangeTree[string, uint, dummyAuthID](),
// 		PendingForcedChanges:   []PendingChange[string, uint, dummyAuthID]{},
// 		AuthoritySetChanges:    AuthoritySetChanges[uint]{},
// 	}

// 	dummyHashNumber := grandpa.HashNumber[string, uint]{
// 		Hash:   "a",
// 		Number: 1,
// 	}

// 	liveSetState, err := NewLiveVoterSetState[string, uint, dummyAuthID, uint](5, authorities, dummyHashNumber)
// 	require.NoError(t, err)

// 	live, err := liveSetState.Value()
// 	require.NoError(t, err)

// 	val, ok := live.(voterSetStateLive[string, uint, dummyAuthID, uint])
// 	require.True(t, ok)
// 	require.Equal(t, uint64(5), val.CompletedRounds.SetId)
// 	require.Equal(t, uint64(0), val.CompletedRounds.Rounds[0].Number)
// }

// func TestVoterSetState_CompletedRounds(t *testing.T) {
// 	authorities := AuthoritySet[string, uint, dummyAuthID]{
// 		CurrentAuthorities:     []Authority[dummyAuthID]{},
// 		SetID:                  1,
// 		PendingStandardChanges: NewChangeTree[string, uint, dummyAuthID](),
// 		PendingForcedChanges:   []PendingChange[string, uint, dummyAuthID]{},
// 		AuthoritySetChanges:    AuthoritySetChanges[uint]{},
// 	}
// 	dummyHashNumber := grandpa.HashNumber[string, uint]{
// 		Hash:   "a",
// 		Number: 1,
// 	}
// 	state := grandpa.NewRoundState[string, uint](dummyHashNumber)
// 	completedRounds := NewCompletedRounds[string, uint, dummyAuthID, uint](
// 		completedRound[string, uint, dummyAuthID, uint]{
// 			10,
// 			state,
// 			dummyHashNumber,
// 			[]grandpa.SignedMessage[string, uint, dummyAuthID, uint]{},
// 		},
// 		5,
// 		authorities,
// 	)

// 	voterSetState := NewVoterSetState[string, uint, dummyAuthID, uint]()

// 	err := voterSetState.Set(voterSetStateLive[string, uint, dummyAuthID, uint]{
// 		CompletedRounds: completedRounds,
// 	})
// 	require.NoError(t, err)

// 	rounds, err := voterSetState.completedRounds()
// 	require.NoError(t, err)
// 	require.Equal(t, completedRounds, rounds)
// }

// func TestVoterSetState_LastCompletedRound(t *testing.T) {
// 	authorities := AuthoritySet[string, uint, dummyAuthID]{
// 		CurrentAuthorities:     []Authority[dummyAuthID]{},
// 		SetID:                  1,
// 		PendingStandardChanges: NewChangeTree[string, uint, dummyAuthID](),
// 		PendingForcedChanges:   []PendingChange[string, uint, dummyAuthID]{},
// 		AuthoritySetChanges:    AuthoritySetChanges[uint]{},
// 	}
// 	dummyHashNumber := grandpa.HashNumber[string, uint]{
// 		Hash:   "a",
// 		Number: 1,
// 	}
// 	state := grandpa.NewRoundState[string, uint](dummyHashNumber)

// 	originalCompletedRound := completedRound[string, uint, dummyAuthID, uint]{
// 		8,
// 		state,
// 		dummyHashNumber,
// 		[]grandpa.SignedMessage[string, uint, dummyAuthID, uint]{},
// 	}
// 	completedRounds := NewCompletedRounds[string, uint, dummyAuthID, uint](
// 		originalCompletedRound,
// 		5,
// 		authorities,
// 	)

// 	addedCompletedRound := completedRound[string, uint, dummyAuthID, uint]{
// 		8,
// 		state,
// 		dummyHashNumber,
// 		[]grandpa.SignedMessage[string, uint, dummyAuthID, uint]{},
// 	}

// 	completedRounds.push(addedCompletedRound)

// 	voterSetState := NewVoterSetState[string, uint, dummyAuthID, uint]()
// 	err := voterSetState.Set(voterSetStatePaused[string, uint, dummyAuthID, uint]{
// 		CompletedRounds: completedRounds,
// 	})
// 	require.NoError(t, err)

// 	lastCompletedRound, err := voterSetState.lastCompletedRound()
// 	require.NoError(t, err)
// 	require.Equal(t, originalCompletedRound, lastCompletedRound)
// }

// func TestVoterSetState_WithCurrentRound(t *testing.T) {
// 	authorities := AuthoritySet[string, uint, dummyAuthID]{
// 		CurrentAuthorities:     []Authority[dummyAuthID]{},
// 		SetID:                  1,
// 		PendingStandardChanges: NewChangeTree[string, uint, dummyAuthID](),
// 		PendingForcedChanges:   []PendingChange[string, uint, dummyAuthID]{},
// 		AuthoritySetChanges:    AuthoritySetChanges[uint]{},
// 	}
// 	dummyHashNumber := grandpa.HashNumber[string, uint]{
// 		Hash:   "a",
// 		Number: 1,
// 	}
// 	state := grandpa.NewRoundState[string, uint](dummyHashNumber)
// 	completedRounds := NewCompletedRounds[string, uint, dummyAuthID, uint](
// 		completedRound[string, uint, dummyAuthID, uint]{
// 			10,
// 			state,
// 			dummyHashNumber,
// 			[]grandpa.SignedMessage[string, uint, dummyAuthID, uint]{},
// 		},
// 		5,
// 		authorities,
// 	)

// 	voterSetState := NewVoterSetState[string, uint, dummyAuthID, uint]()

// 	// voterSetStatePaused
// 	err := voterSetState.Set(voterSetStatePaused[string, uint, dummyAuthID, uint]{
// 		CompletedRounds: completedRounds,
// 	})
// 	require.NoError(t, err)
// 	_, _, err = voterSetState.withCurrentRound(1)
// 	require.NotNil(t, err)
// 	require.Equal(t, "voter acting while in paused state", err.Error())

// 	// voterSetStateLive: invalid round
// 	err = voterSetState.Set(voterSetStateLive[string, uint, dummyAuthID, uint]{
// 		CompletedRounds: completedRounds,
// 	})
// 	require.NoError(t, err)
// 	_, _, err = voterSetState.withCurrentRound(1)
// 	require.NotNil(t, err)
// 	require.Equal(t, "voter acting on a live round we are not tracking", err.Error())

// 	// Valid
// 	currentRounds := CurrentRounds[string, uint, dummyAuthID](
// 		make(map[uint64]hasVoted[string, uint, dummyAuthID]),
// 	)

// 	hasVoted := hasVoted[string, uint, dummyAuthID]{}
// 	hasVoted = hasVoted.New()
// 	err = hasVoted.Set(no{})
// 	require.NoError(t, err)
// 	currentRounds[1] = hasVoted
// 	err = voterSetState.Set(voterSetStateLive[string, uint, dummyAuthID, uint]{
// 		CompletedRounds: completedRounds,
// 		CurrentRounds:   currentRounds,
// 	})
// 	require.NoError(t, err)
// 	completed, current, err := voterSetState.withCurrentRound(1)
// 	require.NoError(t, err)
// 	require.Equal(t, completedRounds, completed)
// 	require.Equal(t, currentRounds, current)
// }

// func TestHasVotedEncoding(t *testing.T) {
// 	vote := vote[string, uint]{}
// 	vote = vote.New()
// 	err := vote.Set(propose[string, uint]{})
// 	require.NoError(t, err)

// 	yes := yes[string, uint, dummyAuthID]{
// 		AuthId: key,
// 		Vote:   vote,
// 	}
// 	hv := hasVoted[string, uint, dummyAuthID]{}
// 	hv = hv.New()
// 	err = hv.Set(yes)
// 	require.NoError(t, err)

// 	res, err := scale.Marshal(hv)
// 	require.NoError(t, err)

// 	newHasVoted := hasVoted[string, uint, dummyAuthID]{}
// 	newHasVoted = hv.New()
// 	err = scale.Unmarshal(res, &newHasVoted)
// 	require.NoError(t, err)
// 	require.Equal(t, hv, newHasVoted)
// }

// func TestHasVoted_Propose(t *testing.T) {
// 	primaryPropose := &grandpa.PrimaryPropose[string, uint]{
// 		TargetHash:   "a",
// 		TargetNumber: 2,
// 	}
// 	vote := vote[string, uint]{}
// 	vote = vote.New()
// 	err := vote.Set(propose[string, uint]{*primaryPropose})
// 	require.NoError(t, err)

// 	yes := yes[string, uint, dummyAuthID]{
// 		Vote: vote,
// 	}
// 	hasVoted := hasVoted[string, uint, dummyAuthID]{}
// 	hasVoted = hasVoted.New()
// 	err = hasVoted.Set(yes)
// 	require.NoError(t, err)

// 	newPrimaryPropose := hasVoted.Propose()
// 	require.Equal(t, primaryPropose, newPrimaryPropose)
// }

// func TestHasVoted_Prevote(t *testing.T) {
// 	prevoteVal := &grandpa.Prevote[string, uint]{
// 		TargetHash:   "a",
// 		TargetNumber: 2,
// 	}
// 	voteVal := vote[string, uint]{}
// 	voteVal = voteVal.New()
// 	err := voteVal.Set(prevote[string, uint]{&grandpa.PrimaryPropose[string, uint]{}, *prevoteVal})
// 	require.NoError(t, err)

// 	y := yes[string, uint, dummyAuthID]{
// 		Vote: voteVal,
// 	}
// 	hasVoted := hasVoted[string, uint, dummyAuthID]{}
// 	hasVoted = hasVoted.New()
// 	err = hasVoted.Set(y)
// 	require.NoError(t, err)

// 	newPrevote := hasVoted.Prevote()
// 	require.Equal(t, prevoteVal, newPrevote)

// 	primaryPropose := &grandpa.PrimaryPropose[string, uint]{
// 		TargetHash:   "a",
// 		TargetNumber: 2,
// 	}
// 	proposeVote := vote[string, uint]{}
// 	proposeVote = voteVal.New()
// 	err = proposeVote.Set(propose[string, uint]{PrimaryPropose: *primaryPropose})
// 	require.NoError(t, err)

// 	y = yes[string, uint, dummyAuthID]{
// 		Vote: proposeVote,
// 	}
// 	hasVoted = hasVoted.New()
// 	err = hasVoted.Set(y)
// 	require.NoError(t, err)

// 	newPrevote = hasVoted.Prevote()
// 	require.Nil(t, newPrevote)
// }

// func TestHasVoted_Precommit(t *testing.T) {
// 	precommitVal := &grandpa.Precommit[string, uint]{
// 		TargetHash:   "a",
// 		TargetNumber: 2,
// 	}
// 	voteVal := vote[string, uint]{}
// 	voteVal = voteVal.New()
// 	err := voteVal.Set(precommit[string, uint]{
// 		&grandpa.PrimaryPropose[string, uint]{},
// 		grandpa.Prevote[string, uint]{},
// 		*precommitVal})
// 	require.NoError(t, err)

// 	y := yes[string, uint, dummyAuthID]{
// 		Vote: voteVal,
// 	}
// 	hasVoted := hasVoted[string, uint, dummyAuthID]{}
// 	hasVoted = hasVoted.New()
// 	err = hasVoted.Set(y)
// 	require.NoError(t, err)

// 	newCommit := hasVoted.Precommit()
// 	require.Equal(t, precommitVal, newCommit)

// 	primaryPropose := &grandpa.PrimaryPropose[string, uint]{
// 		TargetHash:   "a",
// 		TargetNumber: 2,
// 	}
// 	proposeVote := vote[string, uint]{}
// 	proposeVote = proposeVote.New()
// 	err = proposeVote.Set(propose[string, uint]{PrimaryPropose: *primaryPropose})
// 	require.NoError(t, err)

// 	y = yes[string, uint, dummyAuthID]{
// 		Vote: proposeVote,
// 	}
// 	hasVoted = hasVoted.New()
// 	err = hasVoted.Set(y)
// 	require.NoError(t, err)

// 	newCommit = hasVoted.Precommit()
// 	require.Nil(t, newCommit)
// }

// func TestHasVoted_CanPropose(t *testing.T) {
// 	primaryPropose := &grandpa.PrimaryPropose[string, uint]{
// 		TargetHash:   "a",
// 		TargetNumber: 2,
// 	}
// 	voteVal := vote[string, uint]{}
// 	voteVal = voteVal.New()
// 	err := voteVal.Set(propose[string, uint]{*primaryPropose})
// 	require.NoError(t, err)

// 	yes := yes[string, uint, dummyAuthID]{
// 		Vote: voteVal,
// 	}
// 	hasVoted := hasVoted[string, uint, dummyAuthID]{}
// 	hasVoted = hasVoted.New()
// 	err = hasVoted.Set(yes)
// 	require.NoError(t, err)
// 	require.False(t, hasVoted.CanPropose())

// 	hasVoted = hasVoted.New()
// 	err = hasVoted.Set(no{})
// 	require.NoError(t, err)
// 	require.True(t, hasVoted.CanPropose())
// }

// func TestHasVoted_CanPrevote(t *testing.T) {
// 	prevoteVal := &grandpa.Prevote[string, uint]{
// 		TargetHash:   "a",
// 		TargetNumber: 2,
// 	}
// 	voteVal := vote[string, uint]{}
// 	voteVal = voteVal.New()
// 	err := voteVal.Set(prevote[string, uint]{&grandpa.PrimaryPropose[string, uint]{}, *prevoteVal})
// 	require.NoError(t, err)

// 	yes := yes[string, uint, dummyAuthID]{
// 		Vote: voteVal,
// 	}
// 	hasVoted := hasVoted[string, uint, dummyAuthID]{}
// 	hasVoted = hasVoted.New()
// 	err = hasVoted.Set(yes)
// 	require.NoError(t, err)
// 	require.False(t, hasVoted.CanPrevote())

// 	hasVoted = hasVoted.New()
// 	err = hasVoted.Set(no{})
// 	require.NoError(t, err)
// 	require.True(t, hasVoted.CanPrevote())
// }

// func TestHasVoted_CanPrecommit(t *testing.T) {
// 	precommitVal := &grandpa.Precommit[string, uint]{
// 		TargetHash:   "a",
// 		TargetNumber: 2,
// 	}
// 	vote := vote[string, uint]{}
// 	vote = vote.New()
// 	err := vote.Set(precommit[string, uint]{
// 		&grandpa.PrimaryPropose[string, uint]{},
// 		grandpa.Prevote[string, uint]{},
// 		*precommitVal})
// 	require.NoError(t, err)

// 	yes := yes[string, uint, dummyAuthID]{
// 		Vote: vote,
// 	}
// 	hasVoted := hasVoted[string, uint, dummyAuthID]{}
// 	hasVoted = hasVoted.New()
// 	err = hasVoted.Set(yes)
// 	require.NoError(t, err)
// 	require.False(t, hasVoted.CanPrecommit())

// 	hasVoted = hasVoted.New()
// 	err = hasVoted.Set(no{})
// 	require.NoError(t, err)
// 	require.True(t, hasVoted.CanPrecommit())
// }

// func TestVoteEncoding(t *testing.T) {
// 	voteVal := vote[string, uint]{}
// 	voteVal = voteVal.New()
// 	err := voteVal.Set(propose[string, uint]{
// 		PrimaryPropose: grandpa.PrimaryPropose[string, uint]{
// 			TargetHash:   "a",
// 			TargetNumber: 1,
// 		},
// 	})
// 	require.NoError(t, err)

// 	enc, err := scale.Marshal(voteVal)
// 	require.NoError(t, err)

// 	newVote := vote[string, uint]{}
// 	newVote = newVote.New()
// 	err = scale.Unmarshal(enc, &newVote)
// 	require.NoError(t, err)
// 	require.Equal(t, voteVal, newVote)
// }

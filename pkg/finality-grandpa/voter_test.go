// Copyright 2023 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package grandpa

import (
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestVoter_TalkingToMyself(t *testing.T) {
	var localID ID = 5
	voters := NewVoterSet([]IDWeight[ID]{
		{localID, 100},
	})

	network := NewNetwork()

	env := newEnvironment(network, localID)

	// initialize chain
	var lastFinalized HashNumber[string, uint32]
	env.WithChain(func(chain *dummyChain) {
		chain.PushBlocks(GenesisHash, []string{"A", "B", "C", "D", "E"})
		lastFinalized.Hash, lastFinalized.Number = chain.LastFinalized()
	})

	finalized := env.FinalizedStream()
	voter, globalOut := NewVoter[string, uint32, Signature, ID](
		&env,
		*voters,
		make(chan globalInItem),
		0,
		nil,
		lastFinalized,
		lastFinalized,
	)

	globalIn := network.MakeGlobalComms(globalOut)
	voter.globalIn = newWakerChan(globalIn)

	done := make(chan any)
	go func() {
		defer close(done)
		err := voter.Start()
		// stops early, so this should return an error
		assert.Error(t, err)
	}()

	<-finalized
	err := voter.Stop()
	assert.NoError(t, err)
	<-done
}

func TestVoter_FinalizingAtFaultThreshold(t *testing.T) {
	weights := make([]IDWeight[ID], 10)
	for i := range weights {
		weights[i] = IDWeight[ID]{ID(i), 1}
	}
	voters := NewVoterSet(weights)

	network := NewNetwork()

	var wg sync.WaitGroup
	// 3 voters offline.
	for i := 0; i < 7; i++ {
		localID := ID(i)
		// initialize chain
		env := newEnvironment(network, localID)
		var lastFinalized HashNumber[string, uint32]
		env.WithChain(func(chain *dummyChain) {
			chain.PushBlocks(GenesisHash, []string{"A", "B", "C", "D", "E"})
			lastFinalized.Hash, lastFinalized.Number = chain.LastFinalized()
		})

		// run voter in background. scheduling it to shut down at the end.
		finalized := env.FinalizedStream()
		voter, globalOut := NewVoter[string, uint32, Signature, ID](
			&env,
			*voters,
			make(chan globalInItem),
			0,
			nil,
			lastFinalized,
			lastFinalized,
		)

		globalIn := network.MakeGlobalComms(globalOut)
		voter.globalIn = newWakerChan(globalIn)

		wg.Add(1)
		go voter.Start()
		go func() {
			defer wg.Done()
			<-finalized
			err := voter.Stop()
			assert.NoError(t, err)
		}()
	}
	wg.Wait()
}

func TestVoter_ExposingVoterState(t *testing.T) {
	numVoters := 10
	votersOnline := 7

	weights := make([]IDWeight[ID], numVoters)
	for i := range weights {
		weights[i] = IDWeight[ID]{ID(i), 1}
	}
	voterSet := NewVoterSet(weights)

	network := NewNetwork()

	var wg sync.WaitGroup
	voters := make([]*Voter[string, uint32, Signature, ID], votersOnline)
	voterStates := make([]VoterState[ID], votersOnline)
	// some voters offline
	for i := 0; i < votersOnline; i++ {
		localID := ID(i)
		// initialize chain
		env := newEnvironment(network, localID)
		var lastFinalized HashNumber[string, uint32]
		env.WithChain(func(chain *dummyChain) {
			chain.PushBlocks(GenesisHash, []string{"A", "B", "C", "D", "E"})
			lastFinalized.Hash, lastFinalized.Number = chain.LastFinalized()
		})

		// run voter in background. scheduling it to shut down at the end.
		finalized := env.FinalizedStream()
		voter, globalOut := NewVoter[string, uint32, Signature, ID](
			&env,
			*voterSet,
			make(chan globalInItem),
			0,
			nil,
			lastFinalized,
			lastFinalized,
		)

		globalIn := network.MakeGlobalComms(globalOut)
		voter.globalIn = newWakerChan(globalIn)

		voters[i] = voter
		voterStates[i] = voter.VoterState()

		wg.Add(1)
		go func() {
			defer wg.Done()
			<-finalized
		}()
	}

	voterState := voterStates[0]
	for _, vs := range voterStates {
		assert.Equal(t, vs.Get(), voterState.Get())
	}

	expectedRoundState := RoundStateReport[ID]{
		TotalWeight:            VoterWeight(numVoters),
		ThresholdWeight:        VoterWeight(votersOnline),
		PrevoteCurrentWeight:   0,
		PrevoteIDs:             nil,
		PrecommitCurrentWeight: 0,
		PrecommitIDs:           nil,
	}

	assert.Equal(t,
		VoterStateReport[ID]{
			BackgroundRounds: make(map[uint64]RoundStateReport[ID]),
			BestRound: struct {
				Number     uint64
				RoundState RoundStateReport[ID]
			}{1, expectedRoundState},
		},
		voterState.Get(),
	)

	for _, v := range voters {
		go v.Start()
	}
	wg.Wait()

	assert.Equal(t,
		voterState.Get().BestRound,
		struct {
			Number     uint64
			RoundState RoundStateReport[ID]
		}{2, expectedRoundState},
	)

	for _, v := range voters {
		err := v.Stop()
		assert.NoError(t, err)
	}
}

func TestVoter_BroadcastCommit(t *testing.T) {
	localID := ID(5)
	voterSet := NewVoterSet([]IDWeight[ID]{{localID, 100}})

	network := NewNetwork()

	env := newEnvironment(network, localID)

	// initialize chain
	var lastFinalized HashNumber[string, uint32]
	env.WithChain(func(chain *dummyChain) {
		chain.PushBlocks(GenesisHash, []string{"A", "B", "C", "D", "E"})
		lastFinalized.Hash, lastFinalized.Number = chain.LastFinalized()
	})

	// run voter in background. scheduling it to shut down at the end.
	voter, globalOut := NewVoter[string, uint32, Signature, ID](
		&env,
		*voterSet,
		make(chan globalInItem),
		0,
		nil,
		lastFinalized,
		lastFinalized,
	)

	commitsIn := network.MakeGlobalComms(globalOut)

	globalIn := network.MakeGlobalComms(globalOut)
	voter.globalIn = newWakerChan(globalIn)

	go voter.Start()
	<-commitsIn

	err := voter.Stop()
	assert.NoError(t, err)
}

func TestVoter_BroadcastCommitOnlyIfNewer(t *testing.T) {
	localID := ID(5)
	testID := ID(42)
	voterSet := NewVoterSet([]IDWeight[ID]{{localID, 100}, {testID, 201}})

	network := NewNetwork()

	commitsOut := make(chan CommunicationOut)
	commitsIn := network.MakeGlobalComms(commitsOut)

	roundOut := make(chan Message[string, uint32])
	roundIn := network.MakeRoundComms(1, testID, roundOut)

	prevote := Prevote[string, uint32]{"E", 6}
	precommit := Precommit[string, uint32]{"E", 6}

	commit := numberCommit[string, uint32, Signature, ID]{
		1, Commit[string, uint32, Signature, ID]{
			TargetHash:   "E",
			TargetNumber: 6,
			Precommits: []SignedPrecommit[string, uint32, Signature, ID]{
				{
					Precommit: Precommit[string, uint32]{"E", 6},
					Signature: Signature(testID),
					ID:        testID,
				},
			},
		},
	}

	env := newEnvironment(network, localID)

	// initialize chain
	var lastFinalized HashNumber[string, uint32]
	env.WithChain(func(chain *dummyChain) {
		chain.PushBlocks(GenesisHash, []string{"A", "B", "C", "D", "E"})
		lastFinalized.Hash, lastFinalized.Number = chain.LastFinalized()
	})

	// run voter in background. scheduling it to shut down at the end.
	voter, globalOut := NewVoter[string, uint32, Signature, ID](
		&env,
		*voterSet,
		nil,
		0,
		nil,
		lastFinalized,
		lastFinalized,
	)
	globalIn := network.MakeGlobalComms(globalOut)
	voter.globalIn = newWakerChan(globalIn)

	go func() {
		voter.Start()
	}()

	item := <-roundIn
	// wait for a prevote
	assert.NoError(t, item.Error)
	assert.IsType(t, Prevote[string, uint32]{}, item.SignedMessage.Message.Value)
	assert.Equal(t, localID, item.SignedMessage.ID)

	// send our prevote and precommit
	votes := []Message[string, uint32]{newMessage(prevote), newMessage(precommit)}
	for _, v := range votes {
		roundOut <- v
	}

waitForPrecommit:
	for {
		item = <-roundIn
		// wait for a precommit
		assert.NoError(t, item.Error)
		switch item.SignedMessage.Message.Value.(type) {
		case Precommit[string, uint32]:
			if item.SignedMessage.ID == localID {
				break waitForPrecommit
			}
		}
	}

	// send our commit
	co := newCommunicationOut(CommunicationOutCommit[string, uint32, Signature, ID](commit))
	commitsOut <- co

	timer := time.NewTimer(500 * time.Millisecond)
	var commitCount int
waitForCommits:
	for {
		select {
		case <-commitsIn:
			commitCount++
		case <-timer.C:
			break waitForCommits
		}
	}
	assert.Equal(t, 1, commitCount)

	err := voter.Stop()
	assert.NoError(t, err)
}

func TestVoter_ImportCommitForAnyRound(t *testing.T) {
	localID := ID(5)
	testID := ID(42)
	voterSet := NewVoterSet([]IDWeight[ID]{{localID, 100}, {testID, 201}})

	network := NewNetwork()
	commitsOut := make(chan CommunicationOut)
	_ = network.MakeGlobalComms(commitsOut)

	commit := Commit[string, uint32, Signature, ID]{
		TargetHash:   "E",
		TargetNumber: 6,
		Precommits: []SignedPrecommit[string, uint32, Signature, ID]{
			{
				Precommit: Precommit[string, uint32]{"E", 6},
				Signature: Signature(testID),
				ID:        testID,
			},
		},
	}

	env := newEnvironment(network, localID)

	// initialize chain
	var lastFinalized HashNumber[string, uint32]
	env.WithChain(func(chain *dummyChain) {
		chain.PushBlocks(GenesisHash, []string{"A", "B", "C", "D", "E"})
		lastFinalized.Hash, lastFinalized.Number = chain.LastFinalized()
	})

	// run voter in background. scheduling it to shut down at the end.
	voter, globalOut := NewVoter[string, uint32, Signature, ID](
		&env,
		*voterSet,
		nil,
		0,
		nil,
		lastFinalized,
		lastFinalized,
	)

	globalIn := network.MakeGlobalComms(globalOut)
	voter.globalIn = newWakerChan(globalIn)

	go func() {
		voter.Start()
	}()

	// Send the commit message
	co := newCommunicationOut(CommunicationOutCommit[string, uint32, Signature, ID]{
		Number: 0,
		Commit: commit,
	})
	commitsOut <- co

	finalized := <-env.FinalizedStream()
	assert.Equal(t, finalized.Commit, commit)

	err := voter.Stop()
	assert.NoError(t, err)
}

func TestVoter_SkipsToLatestRoundAfterCatchUp(t *testing.T) {
	voterIDs := make([]ID, 3)
	// 3 voters
	weights := make([]IDWeight[ID], 3)
	for i := range weights {
		weights[i] = IDWeight[ID]{ID(i), 1}
		voterIDs[i] = ID(i)
	}
	voterSet := NewVoterSet(weights)
	totalWeight := voterSet.TotalWeight()
	thresholdWeight := voterSet.Threshold()

	network := NewNetwork()

	// initialize unsynced voter at round 0
	localID := ID(4)

	env := newEnvironment(network, localID)
	var lastFinalized HashNumber[string, uint32]
	env.WithChain(func(chain *dummyChain) {
		chain.PushBlocks(GenesisHash, []string{"A", "B", "C", "D", "E"})
		lastFinalized.Hash, lastFinalized.Number = chain.LastFinalized()
	})

	unsyncedVoter, globalOut := NewVoter[string, uint32, Signature, ID](
		&env,
		*voterSet,
		nil,
		0,
		nil,
		lastFinalized,
		lastFinalized,
	)
	globalIn := network.MakeGlobalComms(globalOut)
	unsyncedVoter.globalIn = newWakerChan(globalIn)

	prevote := func(id uint32) SignedPrevote[string, uint32, Signature, ID] {
		return SignedPrevote[string, uint32, Signature, ID]{
			Prevote:   Prevote[string, uint32]{"C", 4},
			ID:        ID(id),
			Signature: Signature(99),
		}
	}

	precommit := func(id uint32) SignedPrecommit[string, uint32, Signature, ID] {
		return SignedPrecommit[string, uint32, Signature, ID]{
			Precommit: Precommit[string, uint32]{"C", 4},
			ID:        ID(id),
			Signature: Signature(99),
		}
	}

	// send in a catch-up message for round 5.
	ci := newCommunicationIn[string, uint32, Signature, ID](CommunicationInCatchUp[string, uint32, Signature, ID]{
		CatchUp: CatchUp[string, uint32, Signature, ID]{
			BaseNumber:  1,
			BaseHash:    GenesisHash,
			RoundNumber: 5,
			Prevotes:    []SignedPrevote[string, uint32, Signature, ID]{prevote(0), prevote(1), prevote(2)},
			Precommits:  []SignedPrecommit[string, uint32, Signature, ID]{precommit(0), precommit(1), precommit(2)},
		},
	})
	network.SendMessage(ci)

	voterState := unsyncedVoter.VoterState()
	_, ok := voterState.Get().BackgroundRounds[5]
	assert.False(t, ok)

	// spawn the voter in the background
	go unsyncedVoter.Start()

	finalized := env.FinalizedStream()

	// wait until it's caught up, it should skip to round 6 and send a
	// finality notification for the block that was finalized by catching
	// up.
	caughtUp := make(chan any)
	go func() {
		for {
			report := voterState.Get()
			if report.BestRound.Number == 6 {
				close(caughtUp)
				return
			}
			<-time.NewTimer(10 * time.Millisecond).C
		}
	}()

	<-caughtUp
	<-finalized
	assert.Equal(t,
		struct {
			Number     uint64
			RoundState RoundStateReport[ID]
		}{
			Number: 6,
			RoundState: RoundStateReport[ID]{
				TotalWeight:            totalWeight,
				ThresholdWeight:        thresholdWeight,
				PrevoteCurrentWeight:   0,
				PrevoteIDs:             nil,
				PrecommitCurrentWeight: 0,
				PrecommitIDs:           nil,
			},
		},
		voterState.Get().BestRound)

	assert.Equal(t,
		RoundStateReport[ID]{
			TotalWeight:            totalWeight,
			ThresholdWeight:        thresholdWeight,
			PrevoteCurrentWeight:   3,
			PrevoteIDs:             voterIDs,
			PrecommitCurrentWeight: 3,
			PrecommitIDs:           voterIDs,
		},
		voterState.Get().BackgroundRounds[5])

	err := unsyncedVoter.Stop()
	assert.NoError(t, err)
}

func TestVoter_PickUpFromPriorWithoutGrandparentState(t *testing.T) {
	localID := ID(5)
	voterSet := NewVoterSet([]IDWeight[ID]{{localID, 100}})

	network := NewNetwork()

	env := newEnvironment(network, localID)

	// initialize chain
	var lastFinalized HashNumber[string, uint32]
	env.WithChain(func(chain *dummyChain) {
		chain.PushBlocks(GenesisHash, []string{"A", "B", "C", "D", "E"})
		lastFinalized.Hash, lastFinalized.Number = chain.LastFinalized()
	})

	// run voter in background. scheduling it to shut down at the end.
	voter, globalOut := NewVoter[string, uint32, Signature, ID](
		&env,
		*voterSet,
		nil,
		10,
		nil,
		lastFinalized,
		lastFinalized,
	)
	globalIn := network.MakeGlobalComms(globalOut)
	voter.globalIn = newWakerChan(globalIn)

	go voter.Start()
	for finalized := range env.FinalizedStream() {
		if finalized.Number >= 6 {
			break
		}
	}

	err := voter.Stop()
	assert.NoError(t, err)
}

func TestVoter_PickUpFromPriorWithGrandparentStatus(t *testing.T) {
	localID := ID(99)
	weights := make([]IDWeight[ID], 100)
	for i := range weights {
		weights[i] = IDWeight[ID]{ID(i), 1}
	}
	voterSet := NewVoterSet(weights)

	network := NewNetwork()

	env := newEnvironment(network, localID)

	// initialize chain
	var lastFinalized HashNumber[string, uint32]
	env.WithChain(func(chain *dummyChain) {
		chain.PushBlocks(GenesisHash, []string{"A", "B", "C", "D", "E"})
		lastFinalized.Hash, lastFinalized.Number = chain.LastFinalized()
	})

	lastRoundVotes := make([]SignedMessage[string, uint32, Signature, ID], 0)

	// round 1 state on disk: 67 prevotes for "E". 66 precommits for "D". 1 precommit "E".
	// the round is completable, but the estimate ("E") is not finalized.
	for id := 0; id < 67; id++ {
		prevote := Prevote[string, uint32]{"E", 6}
		var precommit Precommit[string, uint32]
		if id < 66 {
			precommit = Precommit[string, uint32]{"D", 5}
		} else {
			precommit = Precommit[string, uint32]{"E", 6}
		}

		lastRoundVotes = append(lastRoundVotes, SignedMessage[string, uint32, Signature, ID]{
			Message:   newMessage(prevote),
			Signature: Signature(id),
			ID:        ID(id),
		})

		lastRoundVotes = append(lastRoundVotes, SignedMessage[string, uint32, Signature, ID]{
			Message:   newMessage(precommit),
			Signature: Signature(id),
			ID:        ID(id),
		})

		// round 2 has the same votes.
		//
		// this means we wouldn't be able to start round 3 until
		// the estimate of round-1 moves backwards.
		roundOut := make(chan Message[string, uint32])
		_ = network.MakeRoundComms(2, ID(id), roundOut)
		msgs := []Message[string, uint32]{newMessage(prevote), newMessage(precommit)}
		for _, msg := range msgs {
			roundOut <- msg
		}
	}

	// round 1 fresh communication. we send one more precommit for "D" so the estimate
	// moves backwards.
	sender := ID(67)
	roundOut := make(chan Message[string, uint32])
	_ = network.MakeRoundComms(1, sender, roundOut)
	lastPrecommit := Precommit[string, uint32]{"D", 3}
	roundOut <- newMessage(lastPrecommit)

	// run voter in background. scheduling it to shut down at the end.
	voter, globalOut := NewVoter[string, uint32, Signature, ID](
		&env,
		*voterSet,
		nil,
		1,
		lastRoundVotes,
		lastFinalized,
		lastFinalized,
	)
	globalIn := network.MakeGlobalComms(globalOut)
	voter.globalIn = newWakerChan(globalIn)
	go voter.Start()

	// wait until we see a prevote on round 3 from our local ID,
	// indicating that the round 3 has started.
	roundIn := network.MakeRoundComms(3, ID(1000), nil)
waitForPrevote:
	for sme := range roundIn {
		if sme.Error != nil {
			t.Errorf("wtf?")
		}

		msg := sme.SignedMessage.Message.Value
		switch msg.(type) {
		case Prevote[string, uint32]:
			if sme.SignedMessage.ID == localID {
				break waitForPrevote
			}
		}
	}

	assert.Equal(t, [2]uint64{2, 1}, env.LastCompletedAndConcluded())

	err := voter.Stop()
	assert.NoError(t, err)
}

func TestBuffered(_ *testing.T) {
	in := make(chan int32)
	buffered := newBuffered(in)

	run := true
	wg := sync.WaitGroup{}
	wg.Add(1)
	go func() {
		defer wg.Done()
		for run {
			buffered.Push(999)
			time.Sleep(1 * time.Millisecond)
		}
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		for run {
			buffered.flush(newWaker())
			time.Sleep(1 * time.Millisecond)
		}
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		for range in {
		}
	}()

	time.Sleep(100 * time.Millisecond)
	buffered.Close()

	run = false
	wg.Wait()
}

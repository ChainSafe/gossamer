// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package grandpa

import (
	"context"
	"fmt"
	"math/rand"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/ChainSafe/gossamer/dot/network"
	"github.com/ChainSafe/gossamer/dot/state"
	"github.com/ChainSafe/gossamer/dot/telemetry"
	"github.com/ChainSafe/gossamer/dot/types"
	"github.com/ChainSafe/gossamer/internal/log"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/lib/crypto/ed25519"
	"github.com/ChainSafe/gossamer/lib/keystore"
	"github.com/ChainSafe/gossamer/pkg/scale"
	"github.com/golang/mock/gomock"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/libp2p/go-libp2p-core/protocol"
	"github.com/stretchr/testify/require"
)

var testTimeout = 20 * time.Second

type testJustificationRequest struct {
	to  peer.ID
	num uint32
}

type testNetwork struct {
	t                    *testing.T
	out                  chan GrandpaMessage
	finalised            chan GrandpaMessage
	justificationRequest *testJustificationRequest
}

func newTestNetwork(t *testing.T) *testNetwork {
	return &testNetwork{
		t:         t,
		out:       make(chan GrandpaMessage, 128),
		finalised: make(chan GrandpaMessage, 128),
	}
}

func (n *testNetwork) GossipMessage(msg NotificationsMessage) {
	cm, ok := msg.(*ConsensusMessage)
	require.True(n.t, ok)

	gmsg, err := decodeMessage(cm)
	require.NoError(n.t, err)

	switch gmsg.(type) {
	case *CommitMessage:
		n.finalised <- gmsg
	default:
		n.out <- gmsg
	}
}

func (n *testNetwork) SendMessage(_ peer.ID, _ NotificationsMessage) error {
	return nil
}

func (n *testNetwork) SendJustificationRequest(to peer.ID, num uint32) {
	n.justificationRequest = &testJustificationRequest{
		to:  to,
		num: num,
	}
}

func (*testNetwork) RegisterNotificationsProtocol(
	_ protocol.ID,
	_ byte,
	_ network.HandshakeGetter,
	_ network.HandshakeDecoder,
	_ network.HandshakeValidator,
	_ network.MessageDecoder,
	_ network.NotificationsMessageHandler,
	_ network.NotificationsMessageBatchHandler,
	_ uint64,
) error {
	return nil
}

func (n *testNetwork) SendBlockReqestByHash(_ common.Hash) {}

func setupGrandpa(t *testing.T, kp *ed25519.Keypair) (
	*Service, chan *networkVoteMessage, chan GrandpaMessage, chan GrandpaMessage) {
	st := newTestState(t)
	net := newTestNetwork(t)

	ctrl := gomock.NewController(t)
	telemetryMock := NewMockClient(ctrl)
	telemetryMock.EXPECT().SendMessage(gomock.Any()).AnyTimes()

	telemetryMock.
		EXPECT().
		SendMessage(gomock.Any()).AnyTimes()

	cfg := &Config{
		BlockState:   st.Block,
		GrandpaState: st.Grandpa,
		Voters:       voters,
		Keypair:      kp,
		LogLvl:       log.Info,
		Authority:    true,
		Network:      net,
		Interval:     time.Second,
		Telemetry:    telemetryMock,
	}

	gs, err := NewService(cfg)
	require.NoError(t, err)
	return gs, gs.in, net.out, net.finalised
}

func TestGrandpa_BaseCase(t *testing.T) {
	// this is a base test case that asserts that all validators finalise the same block if they all see the
	// same pre-votes and pre-commits, even if their chains are different
	kr, err := keystore.NewEd25519Keyring()
	require.NoError(t, err)

	gss := make([]*Service, len(kr.Keys))
	prevotes := new(sync.Map)
	precommits := new(sync.Map)

	for i, gs := range gss {
		gs, _, _, _ = setupGrandpa(t, kr.Keys[i])
		gss[i] = gs
		state.AddBlocksToState(t, gs.blockState.(*state.BlockState), 15, false)
		pv, err := gs.determinePreVote()
		require.NoError(t, err)
		prevotes.Store(gs.publicKeyBytes(), &SignedVote{
			Vote: *pv,
		})
	}

	for _, gs := range gss {
		gs.prevotes = prevotes
		gs.precommits = precommits
	}

	for _, gs := range gss {
		pc, err := gs.determinePreCommit()
		require.NoError(t, err)
		precommits.Store(gs.publicKeyBytes(), &SignedVote{
			Vote: *pc,
		})
		err = gs.finalise()
		require.NoError(t, err)
		has, err := gs.blockState.HasJustification(gs.head.Hash())
		require.NoError(t, err)
		require.True(t, has)
	}

	finalised := gss[0].head.Hash()
	for _, gs := range gss {
		require.Equal(t, finalised, gs.head.Hash())
	}
}

func TestGrandpa_DifferentChains(t *testing.T) {
	// this asserts that all validators finalise the same block if they all see the
	// same pre-votes and pre-commits, even if their chains are different lengths (+/-1 block)
	kr, err := keystore.NewEd25519Keyring()
	require.NoError(t, err)

	gss := make([]*Service, len(kr.Keys))
	prevotes := new(sync.Map)
	precommits := new(sync.Map)

	for i, gs := range gss {
		gs, _, _, _ = setupGrandpa(t, kr.Keys[i])
		gss[i] = gs

		r := uint(rand.Intn(2)) // 0 or 1
		state.AddBlocksToState(t, gs.blockState.(*state.BlockState), 4+r, false)
		pv, err := gs.determinePreVote()
		require.NoError(t, err)
		prevotes.Store(gs.publicKeyBytes(), &SignedVote{
			Vote: *pv,
		})
	}

	// only want to add prevotes for a node that has a block that exists on its chain
	for _, gs := range gss {
		prevotes.Range(func(key, prevote interface{}) bool {
			k := key.(ed25519.PublicKeyBytes)
			pv := prevote.(*SignedVote)
			err = gs.validateVote(&pv.Vote)
			if err == nil {
				gs.prevotes.Store(k, pv)
			}
			return true
		})
	}

	for _, gs := range gss {
		pc, err := gs.determinePreCommit()
		require.NoError(t, err)
		precommits.Store(gs.publicKeyBytes(), &SignedVote{
			Vote: *pc,
		})
		err = gs.finalise()
		require.NoError(t, err)
	}

	t.Log(gss[0].blockState.BlocktreeAsString())
	finalised := gss[0].head.Hash()

	for _, gs := range gss[:1] {
		require.Equal(t, finalised, gs.head.Hash())
	}
}

func broadcastVotes(from <-chan GrandpaMessage, to []chan *networkVoteMessage, done *bool) {
	for v := range from {
		for _, tc := range to {
			if *done {
				return
			}

			tc <- &networkVoteMessage{
				msg: v.(*VoteMessage),
			}
		}
	}
}

func cleanup(gs *Service, in chan *networkVoteMessage, done *bool) {
	*done = true
	close(in)
	gs.cancel()
}

func TestPlayGrandpaRound_BaseCase(t *testing.T) {
	// this asserts that all validators finalise the same block if they all see the
	// same pre-votes and pre-commits, even if their chains are different lengths
	kr, err := keystore.NewEd25519Keyring()
	require.NoError(t, err)

	gss := make([]*Service, len(kr.Keys))
	ins := make([]chan *networkVoteMessage, len(kr.Keys))
	outs := make([]chan GrandpaMessage, len(kr.Keys))
	fins := make([]chan GrandpaMessage, len(kr.Keys))
	done := false

	for i := range gss {
		gs, in, out, fin := setupGrandpa(t, kr.Keys[i])
		defer cleanup(gs, in, &done)

		gss[i] = gs
		ins[i] = in
		outs[i] = out
		fins[i] = fin

		state.AddBlocksToState(t, gs.blockState.(*state.BlockState), 4, false)
	}

	for _, out := range outs {
		go broadcastVotes(out, ins, &done)
	}

	for _, gs := range gss {
		time.Sleep(time.Millisecond * 100)
		go gs.initiate()
	}

	wg := sync.WaitGroup{}
	wg.Add(len(kr.Keys))

	finalised := make([]*CommitMessage, len(kr.Keys))

	for i, fin := range fins {
		go func(i int, fin <-chan GrandpaMessage) {
			select {
			case f := <-fin:

				// receive first message, which is finalised block from previous round
				if f.(*CommitMessage).Round == 0 {
					select {
					case f = <-fin:
					case <-time.After(testTimeout):
						t.Errorf("did not receive finalised block from %d", i)
					}
				}

				finalised[i] = f.(*CommitMessage)

			case <-time.After(testTimeout):
				t.Errorf("did not receive finalised block from %d", i)
			}
			wg.Done()
		}(i, fin)

	}

	wg.Wait()

	for _, fb := range finalised {
		require.NotNil(t, fb)
		require.GreaterOrEqual(t, len(fb.Precommits), len(kr.Keys)/2)
		finalised[0].Precommits = []Vote{}
		finalised[0].AuthData = []AuthData{}
		fb.Precommits = []Vote{}
		fb.AuthData = []AuthData{}
		require.Equal(t, finalised[0], fb)
	}
}

func TestPlayGrandpaRound_VaryingChain(t *testing.T) {
	// this asserts that all validators finalise the same block if they all see the
	// same pre-votes and pre-commits, even if their chains are different lengths (+/-1 block)
	kr, err := keystore.NewEd25519Keyring()
	require.NoError(t, err)

	gss := make([]*Service, len(kr.Keys))
	ins := make([]chan *networkVoteMessage, len(kr.Keys))
	outs := make([]chan GrandpaMessage, len(kr.Keys))
	fins := make([]chan GrandpaMessage, len(kr.Keys))
	done := false

	// this represents the chains that will be slightly ahead of the others
	headers := []*types.Header{}
	const diff uint = 1

	for i := range gss {
		gs, in, out, fin := setupGrandpa(t, kr.Keys[i])
		defer cleanup(gs, in, &done)

		gss[i] = gs
		ins[i] = in
		outs[i] = out
		fins[i] = fin

		r := uint(rand.Intn(int(diff)))
		chain, _ := state.AddBlocksToState(t, gs.blockState.(*state.BlockState), 4+r, false)
		if r == diff-1 {
			headers = chain
		}
	}

	for _, out := range outs {
		go broadcastVotes(out, ins, &done)
	}

	for _, gs := range gss {
		time.Sleep(time.Millisecond * 100)
		go gs.initiate()
	}

	// mimic the chains syncing and catching up
	for _, gs := range gss {
		for _, h := range headers {
			time.Sleep(time.Millisecond * 10)
			block := &types.Block{
				Header: *h,
				Body:   types.Body{},
			}
			gs.blockState.(*state.BlockState).AddBlock(block)
		}
	}

	wg := sync.WaitGroup{}
	wg.Add(len(kr.Keys))

	finalised := make([]*CommitMessage, len(kr.Keys))

	for i, fin := range fins {

		go func(i int, fin <-chan GrandpaMessage) {
			select {
			case f := <-fin:

				// receive first message, which is finalised block from previous round
				if f.(*CommitMessage).Round == 0 {
					select {
					case f = <-fin:
					case <-time.After(testTimeout):
						t.Errorf("did not receive finalised block from %d", i)
					}
				}

				finalised[i] = f.(*CommitMessage)

			case <-time.After(testTimeout):
				t.Errorf("did not receive finalised block from %d", i)
			}
			wg.Done()
		}(i, fin)

	}

	wg.Wait()

	for _, fb := range finalised {
		require.NotNil(t, fb)
		require.GreaterOrEqual(t, len(fb.Precommits), len(kr.Keys)/2)
		require.GreaterOrEqual(t, len(fb.AuthData), len(kr.Keys)/2)
		finalised[0].Precommits = []Vote{}
		finalised[0].AuthData = []AuthData{}
		fb.Precommits = []Vote{}
		fb.AuthData = []AuthData{}
		require.Equal(t, finalised[0], fb)
	}
}

func TestPlayGrandpaRound_WithEquivocation(t *testing.T) {
	// this asserts that all validators finalise the same block even if 2/9 of voters equivocate
	kr, err := keystore.NewEd25519Keyring()
	require.NoError(t, err)

	gss := make([]*Service, len(kr.Keys))
	ins := make([]chan *networkVoteMessage, len(kr.Keys))
	outs := make([]chan GrandpaMessage, len(kr.Keys))
	fins := make([]chan GrandpaMessage, len(kr.Keys))

	done := false

	for i := range gss {
		gs, in, out, fin := setupGrandpa(t, kr.Keys[i])
		defer cleanup(gs, in, &done)

		gss[i] = gs
		ins[i] = in
		outs[i] = out
		fins[i] = fin

		// this creates a tree with 2 branches starting at depth 2
		branches := map[uint]int{2: 1}
		state.AddBlocksToStateWithFixedBranches(t, gs.blockState.(*state.BlockState), 4, branches)
	}

	// should have blocktree for all nodes
	leaves := gss[0].blockState.Leaves()

	for _, out := range outs {
		go broadcastVotes(out, ins, &done)
	}

	for _, gs := range gss {
		time.Sleep(time.Millisecond * 100)
		go gs.initiate()
	}

	// nodes 7 and 8 will equivocate
	for _, gs := range gss[7:] {
		vote, err := NewVoteFromHash(leaves[1], gs.blockState)
		require.NoError(t, err)

		_, vmsg, err := gs.createSignedVoteAndVoteMessage(vote, prevote)
		require.NoError(t, err)

		for _, in := range ins {
			in <- &networkVoteMessage{
				msg: vmsg,
			}
		}
	}

	wg := sync.WaitGroup{}
	wg.Add(len(kr.Keys))

	finalised := make([]*CommitMessage, len(kr.Keys))

	for i, fin := range fins {

		go func(i int, fin <-chan GrandpaMessage) {
			select {
			case f := <-fin:

				// receive first message, which is finalised block from previous round
				if f.(*CommitMessage).Round == 0 {

					select {
					case f = <-fin:
					case <-time.After(testTimeout):
						t.Errorf("did not receive finalised block from %d", i)
					}
				}

				finalised[i] = f.(*CommitMessage)
			case <-time.After(testTimeout):
				t.Errorf("did not receive finalised block from %d", i)
			}
			wg.Done()
		}(i, fin)

	}

	wg.Wait()

	for _, fb := range finalised {
		require.NotNil(t, fb)
		require.GreaterOrEqual(t, len(fb.Precommits), len(kr.Keys)/2)
		require.GreaterOrEqual(t, len(fb.AuthData), len(kr.Keys)/2)
		finalised[0].Precommits = []Vote{}
		finalised[0].AuthData = []AuthData{}
		fb.Precommits = []Vote{}
		fb.AuthData = []AuthData{}
		require.Equal(t, finalised[0], fb)
	}
}

func TestPlayGrandpaRound_MultipleRounds(t *testing.T) {
	// this asserts that all validators finalise the same block in successive rounds
	kr, err := keystore.NewEd25519Keyring()
	require.NoError(t, err)

	gss := make([]*Service, len(kr.Keys))
	ins := make([]chan *networkVoteMessage, len(kr.Keys))
	outs := make([]chan GrandpaMessage, len(kr.Keys))
	fins := make([]chan GrandpaMessage, len(kr.Keys))
	done := false

	for i := range gss {
		gs, in, out, fin := setupGrandpa(t, kr.Keys[i])
		defer cleanup(gs, in, &done)

		gss[i] = gs
		ins[i] = in
		outs[i] = out
		fins[i] = fin

		state.AddBlocksToState(t, gs.blockState.(*state.BlockState), 4, false)
	}

	for _, out := range outs {
		go broadcastVotes(out, ins, &done)
	}

	for _, gs := range gss {
		// start rounds at slightly different times to account for real-time node differences
		time.Sleep(time.Millisecond * 100)
		go gs.initiate()
	}

	rounds := 10

	for j := 0; j < rounds; j++ {

		wg := sync.WaitGroup{}
		wg.Add(len(kr.Keys))

		finalised := make([]*CommitMessage, len(kr.Keys))

		for i, fin := range fins {

			go func(i int, fin <-chan GrandpaMessage) {
				select {
				case f := <-fin:

					// receive first message, which is finalised block from previous round
					if f.(*CommitMessage).Round == uint64(j) {
						select {
						case f = <-fin:
						case <-time.After(testTimeout):
							t.Errorf("did not receive finalised block from %d", i)
						}
					}

					finalised[i] = f.(*CommitMessage)
				case <-time.After(testTimeout):
					t.Errorf("did not receive finalised block from %d", i)
				}
				wg.Done()
			}(i, fin)

		}

		wg.Wait()

		for _, fb := range finalised {
			require.NotNil(t, fb)
			require.Greater(t, len(fb.Precommits), len(kr.Keys)/2)
			require.Greater(t, len(fb.AuthData), len(kr.Keys)/2)
			finalised[0].Precommits = []Vote{}
			finalised[0].AuthData = []AuthData{}
			fb.Precommits = []Vote{}
			fb.AuthData = []AuthData{}
			require.Equal(t, finalised[0], fb)

			if j == rounds-1 {
				require.Greater(t, int(fb.Vote.Number), 4)
			}
		}

		chain, _ := state.AddBlocksToState(t, gss[0].blockState.(*state.BlockState), 1, false)
		block := &types.Block{
			Header: *(chain[0]),
			Body:   types.Body{},
		}

		for _, gs := range gss[1:] {
			err := gs.blockState.(*state.BlockState).AddBlock(block)
			require.NoError(t, err)
		}
	}
}

func TestSendingVotesInRightStage(t *testing.T) {
	ed25519Keyring, err := keystore.NewEd25519Keyring()
	require.NoError(t, err)

	bobAuthority := ed25519Keyring.Bob().(*ed25519.Keypair)
	votersPublicKeys := []*ed25519.Keypair{
		ed25519Keyring.Alice().(*ed25519.Keypair),
		bobAuthority,
		ed25519Keyring.Charlie().(*ed25519.Keypair),
		ed25519Keyring.Dave().(*ed25519.Keypair),
	}

	grandpaVoters := make([]types.GrandpaVoter, len(votersPublicKeys))
	for idx, pk := range votersPublicKeys {
		grandpaVoters[idx] = types.GrandpaVoter{
			Key: *pk.Public().(*ed25519.PublicKey),
		}
	}

	ctrl := gomock.NewController(t)
	mockedGrandpaState := NewMockGrandpaState(ctrl)
	mockedGrandpaState.EXPECT().
		NextGrandpaAuthorityChange(testGenesisHeader.Hash(), testGenesisHeader.Number).
		Return(uint(0), state.ErrNoNextAuthorityChange).
		AnyTimes()
	mockedGrandpaState.EXPECT().
		SetPrevotes(uint64(1), uint64(0), gomock.AssignableToTypeOf([]types.GrandpaSignedVote{})).
		Return(nil).
		Times(1)
	mockedGrandpaState.EXPECT().
		SetPrecommits(uint64(1), uint64(0), gomock.AssignableToTypeOf([]types.GrandpaSignedVote{})).
		Return(nil).
		Times(1)
	mockedGrandpaState.EXPECT().
		SetLatestRound(uint64(1)).
		Return(nil).
		Times(1)
	mockedGrandpaState.EXPECT().
		GetPrecommits(uint64(1), uint64(0)).
		Return([]types.GrandpaSignedVote{}, nil).
		Times(1)

	mockedState := NewMockBlockState(ctrl)
	mockedState.EXPECT().
		GenesisHash().
		Return(testGenesisHeader.Hash()).
		Times(2)
	// since the next 3 function has been called based on the amount of time we wait until we get enough
	// prevotes is hard to define a corret amount of times this function shoud be called
	mockedState.EXPECT().
		HasFinalisedBlock(uint64(1), uint64(0)).
		Return(false, nil).
		AnyTimes()
	mockedState.EXPECT().
		HasHeader(testGenesisHeader.Hash()).
		Return(true, nil).
		Times(4)
	mockedState.EXPECT().
		GetHighestRoundAndSetID().
		Return(uint64(0), uint64(0), nil).
		AnyTimes()
	mockedState.EXPECT().
		IsDescendantOf(testGenesisHeader.Hash(), testGenesisHeader.Hash()).
		Return(true, nil).
		AnyTimes()
	mockedState.EXPECT().
		BestBlockHeader().
		Return(testGenesisHeader, nil).
		Times(2)
		// we cannot assert the bytes since some votes is defined while playing grandpa round
	mockedState.EXPECT().
		SetJustification(testGenesisHeader.Hash(), gomock.AssignableToTypeOf([]byte{})).
		Return(nil).
		Times(1)
	mockedState.EXPECT().
		GetHeader(testGenesisHeader.Hash()).
		Return(testGenesisHeader, nil).
		Times(1)
	mockedState.EXPECT().
		SetFinalisedHash(testGenesisHeader.Hash(), uint64(1), uint64(0)).
		Return(nil).
		Times(1)

	expectedFinalizedTelemetryMessage := telemetry.NewAfgFinalizedBlocksUpTo(
		testGenesisHeader.Hash(),
		fmt.Sprint(testGenesisHeader.Number),
	)
	expectedAlicePrevoteTelemetryMessage := telemetry.NewAfgReceivedPrevote(
		testGenesisHeader.Hash(),
		fmt.Sprint(testGenesisHeader.Number),
		grandpaVoters[0].PublicKeyBytes().String(),
	)
	expectedCharliePrevoteTelemetryMessage := telemetry.NewAfgReceivedPrevote(
		testGenesisHeader.Hash(),
		fmt.Sprint(testGenesisHeader.Number),
		grandpaVoters[2].PublicKeyBytes().String(),
	)
	expectedAlicePrecommitTelemetryMessage := telemetry.NewAfgReceivedPrecommit(
		testGenesisHeader.Hash(),
		fmt.Sprint(testGenesisHeader.Number),
		grandpaVoters[0].PublicKeyBytes().String(),
	)
	expectedCharliePrecommitTelemetryMessage := telemetry.NewAfgReceivedPrecommit(
		testGenesisHeader.Hash(),
		fmt.Sprint(testGenesisHeader.Number),
		grandpaVoters[2].PublicKeyBytes().String(),
	)

	mockedTelemetry := NewMockClient(ctrl)
	mockedTelemetry.EXPECT().
		SendMessage(expectedAlicePrevoteTelemetryMessage).
		Times(1)
	mockedTelemetry.EXPECT().
		SendMessage(expectedCharliePrevoteTelemetryMessage).
		Times(1)
	mockedTelemetry.EXPECT().
		SendMessage(expectedAlicePrecommitTelemetryMessage).
		Times(1)
	mockedTelemetry.EXPECT().
		SendMessage(expectedCharliePrecommitTelemetryMessage).
		Times(1)
	mockedTelemetry.EXPECT().
		SendMessage(expectedFinalizedTelemetryMessage).
		Times(1)

	mockedNet := NewMockNetwork(ctrl)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// gossamer gossip a prevote/precommit message and then waits `subroundInterval` * 4
	// to issue another prevote/precommit message
	const subroundInterval = 100 * time.Millisecond
	grandpa := &Service{
		ctx:          ctx,
		cancel:       cancel,
		paused:       atomic.Value{},
		network:      mockedNet,
		blockState:   mockedState,
		grandpaState: mockedGrandpaState,
		in:           make(chan *networkVoteMessage),
		interval:     subroundInterval,
		state: &State{
			round:  1,
			setID:  0,
			voters: grandpaVoters,
		},
		head:               testGenesisHeader,
		authority:          true,
		keypair:            bobAuthority,
		prevotes:           new(sync.Map),
		precommits:         new(sync.Map),
		preVotedBlock:      make(map[uint64]*Vote),
		bestFinalCandidate: make(map[uint64]*Vote),
		telemetry:          mockedTelemetry,
	}
	grandpa.paused.Store(false)

	expectedVote := NewVote(testGenesisHeader.Hash(), uint32(testGenesisHeader.Number))
	_, expectedPrimaryProposal, err := grandpa.createSignedVoteAndVoteMessage(expectedVote, primaryProposal)
	require.NoError(t, err)

	primaryProposal, err := expectedPrimaryProposal.ToConsensusMessage()
	require.NoError(t, err)
	mockedNet.EXPECT().
		GossipMessage(primaryProposal).
		Times(1)

	// first of all we should determine our precommit based on our chain view
	_, expectedPrevoteMessage, err := grandpa.createSignedVoteAndVoteMessage(expectedVote, prevote)
	require.NoError(t, err)

	pv, err := expectedPrevoteMessage.ToConsensusMessage()
	require.NoError(t, err)
	mockedNet.EXPECT().
		GossipMessage(pv).
		AnyTimes()

	// after receive enough prevotes our node should define a precommit message and send it
	_, expectedPrecommitMessage, err := grandpa.createSignedVoteAndVoteMessage(expectedVote, precommit)
	require.NoError(t, err)

	pc, err := expectedPrecommitMessage.ToConsensusMessage()
	require.NoError(t, err)
	mockedNet.EXPECT().
		GossipMessage(pc).
		AnyTimes()

	doneCh := make(chan struct{})
	go func() {
		defer close(doneCh)
		err = grandpa.playGrandpaRound()
		require.NoError(t, err)
	}()

	time.Sleep(grandpa.interval * 5)

	// given that we are BOB and we already had predetermined our prevote in a set
	// of 4 authorities (ALICE, BOB, CHARLIE and DAVE) then we only need 2 more prevotes
	_, aliceVoteMessage, err := createSignedVoteUsing(t, votersPublicKeys[0], expectedVote, prevote, 1, 0)
	require.NoError(t, err)

	grandpa.in <- &networkVoteMessage{
		from: peer.ID("alice"),
		msg:  aliceVoteMessage,
	}

	_, charlieVoteMessage, err := createSignedVoteUsing(t, votersPublicKeys[2], expectedVote, prevote, 1, 0)
	require.NoError(t, err)

	grandpa.in <- &networkVoteMessage{
		from: peer.ID("charlie"),
		msg:  charlieVoteMessage,
	}

	// given that we are BOB and we already had predetermined the precommit given the prevotes
	// we only need 2 more precommit messages
	_, alicePrecommitMessage, err := createSignedVoteUsing(t, votersPublicKeys[0], expectedVote, precommit, 1, 0)
	require.NoError(t, err)

	grandpa.in <- &networkVoteMessage{
		from: peer.ID("alice"),
		msg:  alicePrecommitMessage,
	}

	_, charliePrecommitMessage, err := createSignedVoteUsing(t, votersPublicKeys[2], expectedVote, precommit, 1, 0)
	require.NoError(t, err)

	grandpa.in <- &networkVoteMessage{
		from: peer.ID("charlie"),
		msg:  charliePrecommitMessage,
	}

	commitMessage := &CommitMessage{
		Round:      1,
		Vote:       *NewVoteFromHeader(testGenesisHeader),
		Precommits: []types.GrandpaVote{},
		AuthData:   []AuthData{},
	}
	expectedGossipCommitMessage, err := commitMessage.ToConsensusMessage()
	require.NoError(t, err)
	mockedNet.EXPECT().
		GossipMessage(expectedGossipCommitMessage).
		Times(1)

	<-doneCh
}

func createSignedVoteUsing(t *testing.T, kp *ed25519.Keypair, vote *Vote, stage Subround, round, setID uint64) (*SignedVote, *VoteMessage, error) {
	t.Helper()

	msg, err := scale.Marshal(FullVote{
		Stage: stage,
		Vote:  *vote,
		Round: round,
		SetID: setID,
	})
	if err != nil {
		return nil, nil, err
	}

	sig, err := kp.Sign(msg)
	if err != nil {
		return nil, nil, err
	}

	pc := &SignedVote{
		Vote:        *vote,
		Signature:   ed25519.NewSignatureBytes(sig),
		AuthorityID: kp.Public().(*ed25519.PublicKey).AsBytes(),
	}

	sm := &SignedMessage{
		Stage:       stage,
		BlockHash:   pc.Vote.Hash,
		Number:      pc.Vote.Number,
		Signature:   ed25519.NewSignatureBytes(sig),
		AuthorityID: kp.Public().(*ed25519.PublicKey).AsBytes(),
	}

	vm := &VoteMessage{
		Round:   round,
		SetID:   setID,
		Message: *sm,
	}

	return pc, vm, nil
}

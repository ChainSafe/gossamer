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
	*Service, chan *networkVoteMessage) {
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
	return gs, gs.in
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
		gs, _ = setupGrandpa(t, kr.Keys[i])
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
		gs, _ = setupGrandpa(t, kr.Keys[i])
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

func TestPlayGrandpaRound(t *testing.T) {
	ed25519Keyring, err := keystore.NewEd25519Keyring()
	require.NoError(t, err)

	tests := map[string]struct {
		voters          []*ed25519.Keypair
		whoEquivocates  map[int]struct{}
		defineBlockTree func(t *testing.T, blockState BlockState, neighbourServices []*Service)
	}{
		// this asserts that all validators finalise the same block if they all see the
		// same pre-votes and pre-commits, even if their chains are different lengths
		"base_case": {
			voters: []*ed25519.Keypair{
				ed25519Keyring.Alice().(*ed25519.Keypair),
				ed25519Keyring.Bob().(*ed25519.Keypair),
				ed25519Keyring.Charlie().(*ed25519.Keypair),
				ed25519Keyring.Ian().(*ed25519.Keypair),
				ed25519Keyring.George().(*ed25519.Keypair),
			},
			defineBlockTree: func(t *testing.T, blockState BlockState, _ []*Service) {
				const withBranches = false
				const baseLenght = 4
				state.AddBlocksToState(t, blockState.(*state.BlockState), baseLenght, withBranches)
			},
		},

		"varying_chain": {
			voters: []*ed25519.Keypair{
				ed25519Keyring.Alice().(*ed25519.Keypair),
				ed25519Keyring.Bob().(*ed25519.Keypair),
				ed25519Keyring.Charlie().(*ed25519.Keypair),
				ed25519Keyring.Ian().(*ed25519.Keypair),
			},
			defineBlockTree: func(t *testing.T, blockState BlockState, neighbourServices []*Service) {
				const diff uint = 5
				rand := uint(rand.Intn(int(diff)))

				const withBranches = false
				const baseLenght = 4
				headers, _ := state.AddBlocksToState(t, blockState.(*state.BlockState),
					baseLenght+rand, withBranches)

				// sync the created blocks with the neighbour services
				// letting them know about those blocks
				for _, ns := range neighbourServices {
					for _, header := range headers {
						block := &types.Block{
							Header: *header,
							Body:   types.Body{},
						}
						ns.blockState.(*state.BlockState).AddBlock(block)
					}
				}
			},
		},

		"with_equivocations": {
			voters: []*ed25519.Keypair{
				ed25519Keyring.Alice().(*ed25519.Keypair),
				ed25519Keyring.Bob().(*ed25519.Keypair),
				ed25519Keyring.Charlie().(*ed25519.Keypair),
				ed25519Keyring.Dave().(*ed25519.Keypair),
				ed25519Keyring.Ian().(*ed25519.Keypair),
			},
			// alice and charlie equivocates
			// it is a map as it is easy to check
			whoEquivocates: map[int]struct{}{
				3: {},
				4: {},
			},
			defineBlockTree: func(t *testing.T, blockState BlockState, _ []*Service) {
				// this creates a tree with 2 branches starting at depth 2
				branches := map[uint]int{2: 1}
				const baseLenght = 4
				state.AddBlocksToStateWithFixedBranches(t, blockState.(*state.BlockState), baseLenght, branches)
			},
		},
	}

	for tname, tt := range tests {
		tt := tt
		t.Run(tname, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			grandpaServices := make([]*Service, len(tt.voters))
			grandpaVoters := make([]types.GrandpaVoter, 0, len(tt.voters))

			for _, kp := range tt.voters {
				grandpaVoters = append(grandpaVoters, types.GrandpaVoter{
					Key: *kp.Public().(*ed25519.PublicKey),
				})
			}

			for idx := range tt.voters {
				// gossamer gossip a prevote/precommit message and then waits `subroundInterval` * 4
				// to issue another prevote/precommit message
				const subroundInterval = 100 * time.Millisecond
				ctx, cancel := context.WithCancel(context.Background())
				defer cancel()

				st := newTestState(t)
				grandpaServices[idx] = &Service{
					ctx:          ctx,
					cancel:       cancel,
					paused:       atomic.Value{},
					blockState:   st.Block,
					grandpaState: st.Grandpa,
					//defined as in the grandpa service
					in:             make(chan *networkVoteMessage, 1024),
					receivedCommit: make(chan *CommitMessage),
					interval:       subroundInterval,
					state: &State{
						round:  1,
						setID:  0,
						voters: grandpaVoters,
					},
					head:               testGenesisHeader,
					authority:          true,
					keypair:            tt.voters[idx],
					prevotes:           new(sync.Map),
					precommits:         new(sync.Map),
					preVotedBlock:      make(map[uint64]*Vote),
					bestFinalCandidate: make(map[uint64]*Vote),
					pvEquivocations:    make(map[ed25519.PublicKeyBytes][]*SignedVote),
					pcEquivocations:    make(map[ed25519.PublicKeyBytes][]*SignedVote),
				}
				grandpaServices[idx].paused.Store(false)
			}

			neighbourServices := make([][]*Service, len(grandpaServices))
			for idx := range grandpaServices {
				neighbours := make([]*Service, len(grandpaServices)-1)
				copy(neighbours, grandpaServices[:idx])
				copy(neighbours[idx:], grandpaServices[idx+1:])
				neighbourServices[idx] = neighbours
			}

			producedCommitMessages := make([]*CommitMessage, len(grandpaServices))
			for idx, grandpaService := range grandpaServices {
				idx := idx
				neighbours := neighbourServices[idx]
				tt.defineBlockTree(t, grandpaServices[idx].blockState, neighbours)

				// if the service is an equivocator it should send a different vote
				// into the same round to all its neighbour peers
				serviceNetworkMock := func(serviceIdx int, neighbours []*Service,
					equivocateVote *networkVoteMessage) func(any) {
					return func(arg0 any) {
						consensusMessage, ok := arg0.(*network.ConsensusMessage)
						require.True(t, ok, "expecting *network.ConsensusMessage, got %T", arg0)

						message, err := decodeMessage(consensusMessage)
						require.NoError(t, err)

						switch msg := message.(type) {
						case *VoteMessage:
							for _, neighbour := range neighbours {
								voteMessage := &networkVoteMessage{
									from: peer.ID(fmt.Sprint(serviceIdx)),
									msg:  msg,
								}

								select {
								case neighbour.in <- voteMessage:
								default:
								}

								if equivocateVote != nil {
									select {
									case neighbour.in <- equivocateVote:
									default:
									}
								}
							}
						case *CommitMessage:
							producedCommitMessages[serviceIdx] = msg
						}
					}
				}

				// In this test it is not important to assert the arguments
				// to the telemetry SendMessage mocked func
				// the TestSendingVotesInRightStage does it properly
				telemetryMock := NewMockClient(ctrl)
				grandpaService.telemetry = telemetryMock
				telemetryMock.EXPECT().SendMessage(gomock.Any()).AnyTimes()

				// if the voter is an equivocator then we issue a vote
				// to another block into the same round and set id
				var equivocatedVoteMessage *networkVoteMessage
				_, isEquivocator := tt.whoEquivocates[idx]
				if isEquivocator {
					leaves := grandpaService.blockState.Leaves()

					vote, err := NewVoteFromHash(leaves[1], grandpaService.blockState)
					require.NoError(t, err)

					_, vmsg, err := grandpaService.createSignedVoteAndVoteMessage(
						vote, prevote)
					require.NoError(t, err)

					equivocatedVoteMessage = &networkVoteMessage{
						from: peer.ID(fmt.Sprint(idx)),
						msg:  vmsg,
					}
				}

				// The network mock works like a wire between the running
				// services, and it is not important to assert the arguments
				// the TestSendingVotesInRightStage does it properly
				mockNet := NewMockNetwork(ctrl)
				grandpaService.network = mockNet
				mockNet.EXPECT().
					GossipMessage(gomock.Any()).
					DoAndReturn(serviceNetworkMock(idx, neighbours, equivocatedVoteMessage)).
					AnyTimes()
			}

			wg := new(sync.WaitGroup)
			for _, grandpaService := range grandpaServices {
				wg.Add(1)

				// executes playGrandpaRound in a goroutine and
				// assert the results after the services reach a finalisation
				go func(wg *sync.WaitGroup, gs *Service) {
					defer wg.Done()
					err := gs.playGrandpaRound()
					require.NoError(t, err)
				}(wg, grandpaService)
			}

			wg.Wait()

			for _, grandpaService := range grandpaServices {
				close(grandpaService.in)
			}

			var latestHash common.Hash = grandpaServices[0].head.Hash()
			for _, grandpaService := range grandpaServices[1:] {
				serviceFinalizedHash := grandpaService.head.Hash()
				eql := serviceFinalizedHash.Equal(latestHash)
				if !eql {
					t.Errorf("miss match service finalized hash\n\texpecting %s\n\tgot%s\n",
						latestHash, serviceFinalizedHash)
				}
				latestHash = serviceFinalizedHash
			}

			var latestCommit *CommitMessage = producedCommitMessages[0]
			for _, commitMessage := range producedCommitMessages[1:] {
				require.NotNil(t, commitMessage)
				require.GreaterOrEqual(t, len(commitMessage.Precommits), len(tt.voters)/2)
				require.GreaterOrEqual(t, len(commitMessage.AuthData), len(tt.voters)/2)

				require.Equal(t, latestCommit.Round, commitMessage.Round)
				require.Equal(t, latestCommit.SetID, commitMessage.SetID)
				require.Equal(t, latestCommit.Vote, commitMessage.Vote)
				latestCommit = commitMessage
			}

			// assert that the services who got an equivocator vote
			// stored that information in the map properly
			if len(tt.whoEquivocates) > 0 {
				for idx, grandpaService := range grandpaServices {
					// who equivocates does not take itself in to account
					_, isEquivocator := tt.whoEquivocates[idx]
					if isEquivocator {
						require.LessOrEqual(t, len(grandpaService.pvEquivocations), len(tt.whoEquivocates)-1,
							"%s does not have enough equivocations", grandpaService.publicKeyBytes())
					} else {
						require.LessOrEqual(t, len(grandpaService.pvEquivocations), len(tt.whoEquivocates),
							"%s does not have enough equivocations", grandpaService.publicKeyBytes())
					}
				}
			}
		})
	}
}

func TestPlayGrandpaRoundMultipleRounds(t *testing.T) {
	ctrl := gomock.NewController(t)

	ed25519Keyring, err := keystore.NewEd25519Keyring()
	require.NoError(t, err)

	voters := []*ed25519.Keypair{
		ed25519Keyring.Alice().(*ed25519.Keypair),
		ed25519Keyring.Bob().(*ed25519.Keypair),
		ed25519Keyring.Charlie().(*ed25519.Keypair),
		ed25519Keyring.Dave().(*ed25519.Keypair),
		ed25519Keyring.Ian().(*ed25519.Keypair),
	}

	grandpaVoters := make([]types.GrandpaVoter, 0, len(voters))
	for _, kp := range voters {
		grandpaVoters = append(grandpaVoters, types.GrandpaVoter{
			Key: *kp.Public().(*ed25519.PublicKey),
		})
	}

	grandpaServices := make([]*Service, len(voters))
	for idx := range voters {
		const subroundInterval = 100 * time.Millisecond
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		st := newTestState(t)
		grandpaServices[idx] = &Service{
			ctx:            ctx,
			cancel:         cancel,
			paused:         atomic.Value{},
			blockState:     st.Block,
			grandpaState:   st.Grandpa,
			receivedCommit: make(chan *CommitMessage),
			interval:       subroundInterval,
			in:             make(chan *networkVoteMessage, 1024),
			state: &State{
				round:  1,
				setID:  0,
				voters: grandpaVoters,
			},
			head:               testGenesisHeader,
			authority:          true,
			keypair:            voters[idx],
			preVotedBlock:      make(map[uint64]*Vote),
			bestFinalCandidate: make(map[uint64]*Vote),
		}
		grandpaServices[idx].paused.Store(false)

		const withBranches = false
		const baseLenght = 4
		state.AddBlocksToState(t,
			grandpaServices[idx].blockState.(*state.BlockState),
			baseLenght, withBranches)
	}

	neighbourServices := make([][]*Service, len(grandpaServices))
	for idx := range grandpaServices {
		neighbours := make([]*Service, len(grandpaServices)-1)
		copy(neighbours, grandpaServices[:idx])
		copy(neighbours[idx:], grandpaServices[idx+1:])
		neighbourServices[idx] = neighbours
	}

	const totalRounds = 10
	for currentRound := 1; currentRound <= totalRounds; currentRound++ {
		fmt.Printf("=== starting round %d === \n", currentRound)
		for _, grandpaService := range grandpaServices {
			grandpaService.in = make(chan *networkVoteMessage, 1024)
			grandpaService.state.round = uint64(currentRound)
			grandpaService.prevotes = new(sync.Map)
			grandpaService.precommits = new(sync.Map)
			grandpaService.pvEquivocations = make(map[ed25519.PublicKeyBytes][]*SignedVote)
			grandpaService.pcEquivocations = make(map[ed25519.PublicKeyBytes][]*SignedVote)
		}

		// every grandpa service should produce a commit message
		// indicating that it achieved a finalisation in the round
		producedCommitMessages := make([]*CommitMessage, len(grandpaServices))
		for idx, grandpaService := range grandpaServices {
			idx := idx
			neighbours := neighbourServices[idx]

			serviceNetworkMock := func(serviceIdx int, neighbours []*Service) func(any) {
				return func(arg0 any) {
					consensusMessage, ok := arg0.(*network.ConsensusMessage)
					require.True(t, ok, "expecting *network.ConsensusMessage, got %T", arg0)

					message, err := decodeMessage(consensusMessage)
					require.NoError(t, err)

					switch msg := message.(type) {
					case *VoteMessage:
						for _, neighbour := range neighbours {
							voteMessage := &networkVoteMessage{
								from: peer.ID(fmt.Sprint(serviceIdx)),
								msg:  msg,
							}

							select {
							case neighbour.in <- voteMessage:
							default:
							}
						}
					case *CommitMessage:
						producedCommitMessages[serviceIdx] = msg
					}
				}
			}

			// In this test it is not important to assert the arguments
			// to the telemetry SendMessage mocked func
			// the TestSendingVotesInRightStage does it properly
			telemetryMock := NewMockClient(ctrl)
			grandpaService.telemetry = telemetryMock
			telemetryMock.EXPECT().SendMessage(gomock.Any()).AnyTimes()

			// The network mock works like a wire between the running
			// services, and it is not important to assert the arguments
			// the TestSendingVotesInRightStage does it properly
			mockNet := NewMockNetwork(ctrl)
			grandpaService.network = mockNet
			mockNet.EXPECT().
				GossipMessage(gomock.Any()).
				Do(serviceNetworkMock(idx, neighbours)).
				AnyTimes()
		}

		wg := new(sync.WaitGroup)
		for _, grandpaService := range grandpaServices {
			wg.Add(1)
			// executes playGrandpaRound in a goroutine and
			// assert the results after the services reach a finalisation
			go func(wg *sync.WaitGroup, gs *Service) {
				defer wg.Done()
				err := gs.playGrandpaRound()
				require.NoError(t, err)
			}(wg, grandpaService)
		}
		wg.Wait()

		for _, grandpaService := range grandpaServices {
			close(grandpaService.in)
		}

		const setID uint64 = 0
		assertSameFinalizationAndChainGrowth(t, grandpaServices,
			uint64(currentRound), setID)

		var latestCommit *CommitMessage = producedCommitMessages[0]
		for _, commitMessage := range producedCommitMessages[1:] {
			require.NotNil(t, commitMessage)
			require.GreaterOrEqual(t, len(commitMessage.Precommits), len(voters)/2)
			require.GreaterOrEqual(t, len(commitMessage.AuthData), len(voters)/2)

			require.Equal(t, commitMessage.Round, uint64(currentRound))
			require.Equal(t, latestCommit.Round, commitMessage.Round)
			require.Equal(t, latestCommit.SetID, commitMessage.SetID)
			require.Equal(t, latestCommit.Vote, commitMessage.Vote)
			latestCommit = commitMessage
		}
	}
}

// assertChainGrowth ensure that each service reach the same finalisation result
// and that the result belongs to the same chain as the previously finalized block
func assertSameFinalizationAndChainGrowth(t *testing.T, services []*Service, currentRount, setID uint64) {
	finalizedHeaderCurrentRound := make([]*types.Header, len(services))
	for idx, grandpaService := range services {
		finalizedHeader, err := grandpaService.blockState.GetFinalisedHeader(
			currentRount, setID)
		require.NoError(t, err)
		require.NotNil(t, finalizedHeader, "round %d does not contain an header", currentRount)
		finalizedHeaderCurrentRound[idx] = finalizedHeader
	}

	var latestFinalized common.Hash = finalizedHeaderCurrentRound[0].Hash()
	for _, finalizedHead := range finalizedHeaderCurrentRound[1:] {
		eq := finalizedHead.Hash().Equal(latestFinalized)
		if !eq {
			t.Errorf("miss match finalized hash\n\texpected %s\n\tgot%s\n",
				latestFinalized, finalizedHead)
		}
		latestFinalized = finalizedHead.Hash()
	}

	previousRound := currentRount - 1
	// considering that we start from round 1
	// there is nothing to compare before
	if previousRound == 0 {
		return
	}

	for _, grandpaService := range services {
		previouslyFinalized, err := grandpaService.blockState.
			GetFinalisedHeader(previousRound, setID)
		require.NoError(t, err)

		descendant, err := grandpaService.blockState.IsDescendantOf(
			previouslyFinalized.Hash(), latestFinalized)
		require.NoError(t, err)
		require.True(t, descendant)
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

func createSignedVoteUsing(t *testing.T, kp *ed25519.Keypair, vote *Vote,
	stage Subround, round, setID uint64) (*SignedVote, *VoteMessage, error) { //nolint:unparam
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

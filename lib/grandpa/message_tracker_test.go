// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package grandpa

import (
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/ChainSafe/gossamer/dot/state"
	"github.com/ChainSafe/gossamer/dot/types"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/lib/crypto/ed25519"
	"github.com/ChainSafe/gossamer/lib/keystore"
	"github.com/ChainSafe/gossamer/pkg/scale"
	"github.com/golang/mock/gomock"
	"github.com/libp2p/go-libp2p-core/peer"

	"github.com/stretchr/testify/require"
)

// getMessageFromVotesTracker returns the vote message
// from the votes tracker for the given block hash and authority ID.
func getMessageFromVotesTracker(votes votesTracker,
	blockHash common.Hash, authorityID ed25519.PublicKeyBytes) (
	message *VoteMessage) {
	authorityIDToElement, has := votes.mapping[blockHash]
	if !has {
		return nil
	}

	element, ok := authorityIDToElement[authorityID]
	if !ok {
		return nil
	}

	return element.Value.(networkVoteMessage).msg
}

func TestMessageTracker_ValidateMessage(t *testing.T) {
	kr, err := keystore.NewEd25519Keyring()
	require.NoError(t, err)

	gs := setupGrandpa(t, kr.Bob().(*ed25519.Keypair))
	state.AddBlocksToState(t, gs.blockState.(*state.BlockState), 3, false)
	gs.tracker = newTracker(gs.blockState, gs.messageHandler)

	fake := &types.Header{
		Number: 77,
	}

	gs.keypair = kr.Alice().(*ed25519.Keypair)
	_, msg, err := gs.createSignedVoteAndVoteMessage(NewVoteFromHeader(fake), prevote)
	require.NoError(t, err)
	gs.keypair = kr.Bob().(*ed25519.Keypair)

	expectedErr := fmt.Errorf("validating vote: %w", ErrBlockDoesNotExist)
	_, err = gs.validateVoteMessage("", msg)
	require.ErrorIs(t, err, ErrBlockDoesNotExist)
	require.Equal(t, err, expectedErr)

	authorityID := kr.Alice().Public().(*ed25519.PublicKey).AsBytes()
	voteMessage := getMessageFromVotesTracker(gs.tracker.votes, fake.Hash(), authorityID)
	require.Equal(t, msg, voteMessage)
}

func TestMessageTracker_SendMessage(t *testing.T) {
	kr, err := keystore.NewEd25519Keyring()
	require.NoError(t, err)

	gs := setupGrandpa(t, kr.Bob().(*ed25519.Keypair))

	state.AddBlocksToState(t, gs.blockState.(*state.BlockState), 3, false)
	gs.tracker = newTracker(gs.blockState, gs.messageHandler)
	gs.tracker.start()
	defer gs.tracker.stop()

	parent, err := gs.blockState.BestBlockHeader()
	require.NoError(t, err)

	digest := types.NewDigest()
	prd, err := types.NewBabeSecondaryPlainPreDigest(0, 1).ToPreRuntimeDigest()
	require.NoError(t, err)
	err = digest.Add(*prd)
	require.NoError(t, err)

	next := &types.Header{
		ParentHash: parent.Hash(),
		Number:     4,
		Digest:     digest,
	}

	aliceAuthority := kr.Alice().(*ed25519.Keypair)
	aliceSignedVote, aliceVoteMessage := createAndSignVoteMessage(t, aliceAuthority, gs.state.round,
		gs.state.setID, NewVoteFromHeader(next), prevote)

	expectedErr := fmt.Errorf("validating vote: %w", ErrBlockDoesNotExist)
	_, err = gs.validateVoteMessage("", aliceVoteMessage)
	require.ErrorIs(t, err, ErrBlockDoesNotExist)
	require.Equal(t, err, expectedErr)

	authorityID := kr.Alice().Public().(*ed25519.PublicKey).AsBytes()
	voteMessage := getMessageFromVotesTracker(gs.tracker.votes, next.Hash(), authorityID)
	require.Equal(t, aliceVoteMessage, voteMessage)

	err = gs.blockState.(*state.BlockState).AddBlock(&types.Block{
		Header: *next,
		Body:   types.Body{},
	})
	require.NoError(t, err)

	// grandpa tracker check every second if the block
	// was included in the block tree
	waitTracker := time.NewTimer(2 * time.Second)
	<-waitTracker.C

	aliceAuthorityPublicBytes := aliceAuthority.Public().(*ed25519.PublicKey).AsBytes()
	gotSignedVote, ok := gs.loadVote(aliceAuthorityPublicBytes, prevote)
	require.True(t, ok)
	require.Equal(t, aliceSignedVote, gotSignedVote)
}

func TestMessageTracker_ProcessMessage(t *testing.T) {
	kr, err := keystore.NewEd25519Keyring()
	require.NoError(t, err)

	gs := setupGrandpa(t, kr.Bob().(*ed25519.Keypair))
	defer gs.cancel()

	state.AddBlocksToState(t, gs.blockState.(*state.BlockState), 3, false)

	err = gs.Start()
	require.NoError(t, err)

	time.Sleep(time.Second) // wait for round to initiate

	parent, err := gs.blockState.BestBlockHeader()
	require.NoError(t, err)

	digest := types.NewDigest()
	prd, err := types.NewBabeSecondaryPlainPreDigest(0, 1).ToPreRuntimeDigest()
	require.NoError(t, err)
	err = digest.Add(*prd)
	require.NoError(t, err)

	next := &types.Header{
		ParentHash: parent.Hash(),
		Number:     4,
		Digest:     digest,
	}

	gs.keypair = kr.Alice().(*ed25519.Keypair)
	_, msg, err := gs.createSignedVoteAndVoteMessage(NewVoteFromHeader(next), prevote)
	require.NoError(t, err)
	gs.keypair = kr.Bob().(*ed25519.Keypair)

	expectedErr := fmt.Errorf("validating vote: %w", ErrBlockDoesNotExist)
	_, err = gs.validateVoteMessage("", msg)
	require.ErrorIs(t, err, ErrBlockDoesNotExist)
	require.Equal(t, err, expectedErr)

	authorityID := kr.Alice().Public().(*ed25519.PublicKey).AsBytes()
	voteMessage := getMessageFromVotesTracker(gs.tracker.votes, next.Hash(), authorityID)
	require.Equal(t, msg, voteMessage)

	err = gs.blockState.(*state.BlockState).AddBlock(&types.Block{
		Header: *next,
		Body:   types.Body{},
	})
	require.NoError(t, err)

	time.Sleep(time.Second)
	expectedVote := &Vote{
		Hash:   msg.Message.BlockHash,
		Number: msg.Message.Number,
	}
	pv, has := gs.prevotes.Load(kr.Alice().Public().(*ed25519.PublicKey).AsBytes())
	require.True(t, has)
	require.Equal(t, expectedVote, &pv.(*SignedVote).Vote, gs.tracker.votes)
}

func TestMessageTracker_MapInsideMap(t *testing.T) {
	kr, err := keystore.NewEd25519Keyring()
	require.NoError(t, err)

	gs := setupGrandpa(t, kr.Bob().(*ed25519.Keypair))
	state.AddBlocksToState(t, gs.blockState.(*state.BlockState), 3, false)
	gs.tracker = newTracker(gs.blockState, gs.messageHandler)

	header := &types.Header{
		Number: 77,
	}

	hash := header.Hash()
	messages := gs.tracker.votes.messages(hash)
	require.Empty(t, messages)

	gs.keypair = kr.Alice().(*ed25519.Keypair)
	authorityID := kr.Alice().Public().(*ed25519.PublicKey).AsBytes()
	_, msg, err := gs.createSignedVoteAndVoteMessage(NewVoteFromHeader(header), prevote)
	require.NoError(t, err)
	gs.keypair = kr.Bob().(*ed25519.Keypair)

	gs.tracker.addVote("", msg)

	voteMessage := getMessageFromVotesTracker(gs.tracker.votes, hash, authorityID)
	require.NotEmpty(t, voteMessage)
}

func TestMessageTracker_handleTick(t *testing.T) {
	// TODO: remove this skip once the PR https://github.com/ChainSafe/gossamer/pull/292 merges
	t.Skip()
	t.Parallel()

	tests := map[string]struct {
		serviceRound       uint64
		voteRound          uint64
		trackerKeepMessage bool
	}{
		"vote_round_greater_than_service_round": {
			serviceRound:       1,
			voteRound:          2,
			trackerKeepMessage: true,
		},
		"vote_round_less_than_service_round": {
			serviceRound:       2,
			voteRound:          1,
			trackerKeepMessage: false,
		},
	}

	for tname, tt := range tests {
		tt := tt

		t.Run(tname, func(t *testing.T) {
			t.Parallel()

			kr, err := keystore.NewEd25519Keyring()
			require.NoError(t, err)

			ctrl := gomock.NewController(t)

			telemetryMock := NewMockClient(ctrl)
			telemetryMock.EXPECT().SendMessage(gomock.Any()).Times(1)

			const setID uint64 = 0
			grandpaStateMock := NewMockGrandpaState(ctrl)

			blockStateMock := NewMockBlockState(ctrl)
			blockStateMock.EXPECT().
				GetImportedBlockNotifierChannel().
				Return(make(chan *types.Block)).
				Times(1)

			fakePeerID := peer.ID("charlie-fake-peer-id")
			networkMock := NewMockNetwork(ctrl)

			if tt.voteRound < tt.serviceRound {
				blockStateMock.EXPECT().
					GetFinalisedHeader(tt.voteRound, setID).
					Return(testGenesisHeader, nil).
					Times(1)

				grandpaStateMock.EXPECT().
					GetPrecommits(tt.voteRound, setID).
					Return([]types.GrandpaSignedVote{}, nil).
					Times(1)

				var notificationMessage NotificationsMessage = &ConsensusMessage{}
				networkMock.EXPECT().
					SendMessage(fakePeerID, gomock.AssignableToTypeOf(notificationMessage)).
					Times(1)
			}

			grandpaService := &Service{
				telemetry: telemetryMock,
				keypair:   kr.Bob().(*ed25519.Keypair),
				state: &State{
					voters: newTestVoters(),
					setID:  0,
					round:  tt.serviceRound,
				},
				grandpaState: grandpaStateMock,
				blockState:   blockStateMock,
				network:      networkMock,
				prevotes:     new(sync.Map),
			}

			messageHandler := NewMessageHandler(grandpaService, blockStateMock, telemetryMock)
			grandpaService.messageHandler = messageHandler
			grandpaService.tracker = newTracker(blockStateMock, messageHandler)

			vote := &Vote{
				Hash:   testGenesisHeader.Hash(),
				Number: uint32(testGenesisHeader.Number),
			}

			authority := kr.Charlie().(*ed25519.Keypair)
			_, voteMessage := createAndSignVoteMessage(t, authority,
				tt.voteRound, setID, vote, prevote)

			grandpaService.tracker.addVote(fakePeerID, voteMessage)
			grandpaService.tracker.handleTick()

			var expectedLen int = 1
			if !tt.trackerKeepMessage {
				expectedLen = 0
			}

			require.Len(t, grandpaService.tracker.votes.messages(vote.Hash), expectedLen)
		})
	}
}

func createAndSignVoteMessage(t *testing.T, kp *ed25519.Keypair, round, setID uint64,
	vote *Vote, stage Subround) (*SignedVote, *VoteMessage) {
	t.Helper()

	msg, err := scale.Marshal(FullVote{
		Stage: stage,
		Vote:  *vote,
		Round: round,
		SetID: setID,
	})
	require.NoError(t, err)

	sig, err := kp.Sign(msg)
	require.NoError(t, err)

	publicBytes := kp.Public().(*ed25519.PublicKey).AsBytes()
	pc := &SignedVote{
		Vote:        *vote,
		Signature:   ed25519.NewSignatureBytes(sig),
		AuthorityID: publicBytes,
	}

	sm := &SignedMessage{
		Stage:       stage,
		BlockHash:   pc.Vote.Hash,
		Number:      pc.Vote.Number,
		Signature:   ed25519.NewSignatureBytes(sig),
		AuthorityID: publicBytes,
	}

	vm := &VoteMessage{
		Round:   round,
		SetID:   setID,
		Message: *sm,
	}

	return pc, vm
}

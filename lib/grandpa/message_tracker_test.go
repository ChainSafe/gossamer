//go:build integration

// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package grandpa

import (
	"container/list"
	"fmt"
	"sync"
	"testing"

	"github.com/ChainSafe/gossamer/dot/telemetry"
	"github.com/ChainSafe/gossamer/dot/types"
	"github.com/ChainSafe/gossamer/internal/database"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/lib/crypto/ed25519"
	"github.com/ChainSafe/gossamer/lib/keystore"
	"github.com/ChainSafe/gossamer/pkg/scale"
	"github.com/libp2p/go-libp2p/core/peer"
	"go.uber.org/mock/gomock"

	"github.com/stretchr/testify/require"
)

// getMessageFromVotesMapping returns the vote message
// from the votes tracker for the given block hash and authority ID.
func getMessageFromVotesMapping(votesMapping map[common.Hash]map[ed25519.PublicKeyBytes]*list.Element,
	blockHash common.Hash, authorityID ed25519.PublicKeyBytes) (
	message *VoteMessage) {
	authorityIDToElement, has := votesMapping[blockHash]
	if !has {
		return nil
	}

	element, ok := authorityIDToElement[authorityID]
	if !ok {
		return nil
	}

	return element.Value.(networkVoteMessage).msg
}

func TestMessageTracker_handleTick_commitMessage(t *testing.T) {
	t.Parallel()

	kr, err := keystore.NewEd25519Keyring()
	require.NoError(t, err)

	testcases := map[string]struct {
		expectedCommitMessage bool
		newGrandpaService     func(ctrl *gomock.Controller) *Service
	}{
		"get_header_failed_should_keep_commit": {
			expectedCommitMessage: true,
			newGrandpaService: func(ctrl *gomock.Controller) *Service {
				networkMock := NewMockNetwork(ctrl)
				grandpaStateMock := NewMockGrandpaState(ctrl)

				blockStateMock := NewMockBlockState(ctrl)
				blockStateMock.EXPECT().
					GetImportedBlockNotifierChannel().
					Return(make(chan *types.Block))

				blockStateMock.EXPECT().
					GetHeader(testHash).
					Return(nil, database.ErrNotFound)

				grandpaService := &Service{
					telemetry: nil,
					keypair:   kr.Bob().(*ed25519.Keypair),
					state: &State{
						voters: newTestVoters(t),
						setID:  0,
						round:  1,
					},
					grandpaState: grandpaStateMock,
					blockState:   blockStateMock,
					network:      networkMock,
					prevotes:     new(sync.Map),
				}
				messageHandler := NewMessageHandler(grandpaService, blockStateMock, nil)
				grandpaService.messageHandler = messageHandler
				grandpaService.tracker = newTracker(blockStateMock, messageHandler)

				return grandpaService
			},
		},
		"handel_commit_successfully": {
			newGrandpaService: func(ctrl *gomock.Controller) *Service {
				networkMock := NewMockNetwork(ctrl)

				blockStateMock := NewMockBlockState(ctrl)
				blockStateMock.EXPECT().
					GetImportedBlockNotifierChannel().
					Return(make(chan *types.Block))
				blockStateMock.EXPECT().
					GetHeader(testHash).
					Return(&types.Header{
						Number: 1,
					}, nil)

				highestFinalizedHeader := &types.Header{}
				blockStateMock.EXPECT().
					GetHighestFinalisedHeader().
					Return(highestFinalizedHeader, nil)

				blockStateMock.EXPECT().
					IsDescendantOf(highestFinalizedHeader.Hash(), testHash).
					Return(true, nil)

				const commitMessageRound = uint64(100)
				const serviceStateSetID = uint64(0)

				blockStateMock.EXPECT().
					HasFinalisedBlock(commitMessageRound, serviceStateSetID).
					Return(false, nil)

				blockStateMock.EXPECT().
					SetFinalisedHash(testHash, commitMessageRound, serviceStateSetID).
					Return(nil)

				grandpaStateMock := NewMockGrandpaState(ctrl)
				grandpaStateMock.EXPECT().
					SetPrecommits(commitMessageRound, uint64(0), []types.GrandpaSignedVote{})

				telemetryMock := NewMockTelemetry(ctrl)

				commitMessageTelemetry := telemetry.NewAfgReceivedCommit(
					testHash, "1", []string{})
				telemetryMock.EXPECT().SendMessage(commitMessageTelemetry)

				grandpaService := &Service{
					telemetry: telemetryMock,
					keypair:   kr.Bob().(*ed25519.Keypair),
					state: &State{
						voters: []types.GrandpaVoter{},
						setID:  0,
						round:  1,
					},
					grandpaState: grandpaStateMock,
					blockState:   blockStateMock,
					network:      networkMock,
					prevotes:     new(sync.Map),
				}
				messageHandler := NewMessageHandler(grandpaService, blockStateMock, nil)
				grandpaService.messageHandler = messageHandler
				grandpaService.tracker = newTracker(blockStateMock, messageHandler)

				return grandpaService
			},
		},
	}

	for tname, tt := range testcases {
		tt := tt

		t.Run(tname, func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			grandpaService := tt.newGrandpaService(ctrl)

			commitMessage := &CommitMessage{
				Round: 100,
				SetID: 0,
				Vote: types.GrandpaVote{
					Hash:   testHash,
					Number: 1,
				},
			}

			grandpaService.tracker.addCommit(commitMessage)
			grandpaService.tracker.handleTick()

			trackedCommitMessage := grandpaService.tracker.commits.message(testHash)
			require.Equal(t, tt.expectedCommitMessage, trackedCommitMessage != nil)
		})
	}

}

func TestMessageTracker_handleTick_voteMessage(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		serviceRound uint64
		voteRound    uint64
		keepVoting   bool
	}{
		"vote_round_greater_than_service_round": {
			serviceRound: 1,
			voteRound:    2,
			keepVoting:   true,
		},
		"vote_round_less_than_service_round": {
			serviceRound: 2,
			voteRound:    1,
			keepVoting:   false,
		},
	}

	for tname, tt := range tests {
		tt := tt

		t.Run(tname, func(t *testing.T) {
			t.Parallel()

			kr, err := keystore.NewEd25519Keyring()
			require.NoError(t, err)

			ctrl := gomock.NewController(t)

			telemetryMock := NewMockTelemetry(ctrl)
			authority := kr.Charlie().(*ed25519.Keypair)
			publicBytes := authority.Public().(*ed25519.PublicKey).AsBytes()

			prevoteTelemetryMessage := telemetry.NewAfgReceivedPrevote(
				testGenesisHeader.Hash(),
				fmt.Sprint(testGenesisHeader.Number),
				publicBytes.String(),
			)

			telemetryMock.EXPECT().SendMessage(prevoteTelemetryMessage)

			const setID uint64 = 0
			grandpaStateMock := NewMockGrandpaState(ctrl)

			blockStateMock := NewMockBlockState(ctrl)
			blockStateMock.EXPECT().
				GetImportedBlockNotifierChannel().
				Return(make(chan *types.Block))

			fakePeerID := peer.ID("charlie-fake-peer-id")
			networkMock := NewMockNetwork(ctrl)

			if tt.voteRound < tt.serviceRound {
				blockStateMock.EXPECT().
					GetFinalisedHeader(tt.voteRound, setID).
					Return(testGenesisHeader, nil)

				grandpaStateMock.EXPECT().
					GetPrecommits(tt.voteRound, setID).
					Return([]types.GrandpaSignedVote{}, nil)

				var notificationMessage NotificationsMessage = &ConsensusMessage{}
				networkMock.EXPECT().
					SendMessage(fakePeerID, gomock.AssignableToTypeOf(notificationMessage))
			}

			grandpaService := &Service{
				telemetry: telemetryMock,
				keypair:   kr.Bob().(*ed25519.Keypair),
				state: &State{
					voters: newTestVoters(t),
					setID:  0,
					round:  tt.serviceRound,
				},
				grandpaState: grandpaStateMock,
				blockState:   blockStateMock,
				network:      networkMock,
				prevotes:     new(sync.Map),
			}

			messageHandler := NewMessageHandler(grandpaService, blockStateMock, telemetryMock)
			grandpaService.tracker = newTracker(blockStateMock, messageHandler)
			grandpaService.messageHandler = messageHandler

			vote := &Vote{
				Hash:   testGenesisHeader.Hash(),
				Number: uint32(testGenesisHeader.Number),
			}

			_, voteMessage := createAndSignVoteMessage(t, authority,
				tt.voteRound, setID, vote, prevote)

			grandpaService.tracker.addVote(fakePeerID, voteMessage)
			grandpaService.tracker.handleTick()

			expectedLen := 1
			if !tt.keepVoting {
				expectedLen = 0
			}

			require.Len(t, grandpaService.tracker.votes.messages(vote.Hash), expectedLen)
		})
	}
}

func createAndSignVoteMessage(t *testing.T, kp *ed25519.Keypair, round, setID uint64,
	vote *Vote, stage Subround) (*SignedVote, *VoteMessage) {
	t.Helper()

	fullVoteEncoded, err := scale.Marshal(FullVote{
		Stage: stage,
		Vote:  *vote,
		Round: round,
		SetID: setID,
	})
	require.NoError(t, err)

	signature, err := kp.Sign(fullVoteEncoded)
	require.NoError(t, err)

	publicKeyBytes := kp.Public().(*ed25519.PublicKey).AsBytes()
	singedVote := &SignedVote{
		Vote:        *vote,
		Signature:   ed25519.NewSignatureBytes(signature),
		AuthorityID: publicKeyBytes,
	}

	signedMessage := &SignedMessage{
		Stage:       stage,
		BlockHash:   singedVote.Vote.Hash,
		Number:      singedVote.Vote.Number,
		Signature:   ed25519.NewSignatureBytes(signature),
		AuthorityID: publicKeyBytes,
	}

	voteMessage := &VoteMessage{
		Round:   round,
		SetID:   setID,
		Message: *signedMessage,
	}

	return singedVote, voteMessage
}

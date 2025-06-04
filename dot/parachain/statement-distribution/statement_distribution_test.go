// Copyright 2025 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package statementdistribution

import (
	"bytes"
	"testing"

	"github.com/ChainSafe/gossamer/dot/parachain/backing"
	networkbridgemessages "github.com/ChainSafe/gossamer/dot/parachain/network-bridge/messages"
	parachaintypes "github.com/ChainSafe/gossamer/dot/parachain/types"
	validationprotocol "github.com/ChainSafe/gossamer/dot/parachain/validation-protocol"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestSendBackingFreshStatements(t *testing.T) {
	t.Run("should_send_3_statements_to_backing", func(t *testing.T) {
		ctrl := gomock.NewController(t)

		relayParent := common.Hash{0x01}
		groupIndex := parachaintypes.GroupIndex(0)

		confirmedCandidate := &confirmedCandidate{}
		h, err := confirmedCandidate.receipt.Hash()
		require.NoError(t, err)

		candidateHash := parachaintypes.CandidateHash{Value: h}

		validators := []parachaintypes.ValidatorIndex{0, 1, 2}
		initGroups := [][]parachaintypes.ValidatorIndex{
			validators,
			{3, 4, 5},
		}
		sessionState := &perSessionState{
			groups: newGroups(initGroups, 1),
		}

		stmtStore := NewMockstatementStore(ctrl)
		stmtStore.EXPECT().
			freshStatementsForBacking(validators, candidateHash).
			Return([]parachaintypes.SignedStatement{
				parachaintypes.SignedStatement(
					parachaintypes.UncheckedSignedCompactStatement{
						ValidatorIndex: parachaintypes.ValidatorIndex(0),
						Payload:        *parachaintypes.NewCompactSeconded(candidateHash).ToEncodable(),
						Signature:      parachaintypes.ValidatorSignature{0x01, 0x02, 0x03},
					},
				),
				parachaintypes.SignedStatement(
					parachaintypes.UncheckedSignedCompactStatement{
						ValidatorIndex: parachaintypes.ValidatorIndex(1),
						Payload:        *parachaintypes.NewCompactValid(candidateHash).ToEncodable(),
						Signature:      parachaintypes.ValidatorSignature{0x04, 0x04, 0x04},
					},
				),
				parachaintypes.SignedStatement(
					parachaintypes.UncheckedSignedCompactStatement{
						ValidatorIndex: parachaintypes.ValidatorIndex(2),
						Payload:        *parachaintypes.NewCompactValid(candidateHash).ToEncodable(),
						Signature:      parachaintypes.ValidatorSignature{0x05, 0x05, 0x05},
					},
				),
			})

		stmtStore.EXPECT().
			noteKnownByBacking(parachaintypes.ValidatorIndex(0),
				parachaintypes.NewCompactSeconded(candidateHash))
		stmtStore.EXPECT().
			noteKnownByBacking(parachaintypes.ValidatorIndex(1),
				parachaintypes.NewCompactValid(candidateHash))
		stmtStore.EXPECT().
			noteKnownByBacking(parachaintypes.ValidatorIndex(2),
				parachaintypes.NewCompactValid(candidateHash))

		relayParentState := &perRelayParentState{
			statementStore: stmtStore,
		}

		// the channel cap should be the same as the amount of messages
		// exepected to receive from the subsystem otherwise the test
		// will block
		overseerCh := make(chan any, 3)
		sd := &StatementDistribution{
			SubSystemToOverseer: overseerCh,
		}
		err = sd.sendBackingFreshStatements(
			candidateHash,
			groupIndex,
			relayParent,
			relayParentState,
			confirmedCandidate,
			sessionState,
		)
		require.NoError(t, err)

		// create the expected messages that backing should receive
		stmtSeconded := parachaintypes.NewStatementVDT()
		err = stmtSeconded.SetValue(parachaintypes.Seconded(confirmedCandidate.receipt))
		require.NoError(t, err)

		// create the expected messages that backing should receive
		stmtValid := parachaintypes.NewStatementVDT()
		err = stmtValid.SetValue(parachaintypes.Valid(candidateHash))
		require.NoError(t, err)

		expectedMessages := []backing.StatementMessage{
			{
				RelayParent: relayParent,
				SignedFullStatement: parachaintypes.SignedFullStatementWithPVD{
					SignedFullStatement: parachaintypes.SignedFullStatement{
						Payload:        stmtSeconded,
						ValidatorIndex: parachaintypes.ValidatorIndex(0),
						Signature:      parachaintypes.ValidatorSignature{0x01, 0x02, 0x03},
					},
					PersistedValidationData: confirmedCandidate.pvd,
				},
			},

			{
				RelayParent: relayParent,
				SignedFullStatement: parachaintypes.SignedFullStatementWithPVD{
					SignedFullStatement: parachaintypes.SignedFullStatement{
						Payload:        stmtValid,
						ValidatorIndex: parachaintypes.ValidatorIndex(1),
						Signature:      parachaintypes.ValidatorSignature{0x04, 0x04, 0x04},
					},
					PersistedValidationData: confirmedCandidate.pvd,
				},
			},

			{
				RelayParent: relayParent,
				SignedFullStatement: parachaintypes.SignedFullStatementWithPVD{
					SignedFullStatement: parachaintypes.SignedFullStatement{
						Payload:        stmtValid,
						ValidatorIndex: parachaintypes.ValidatorIndex(2),
						Signature:      parachaintypes.ValidatorSignature{0x05, 0x05, 0x05},
					},
					PersistedValidationData: confirmedCandidate.pvd,
				},
			},
		}

		for i := 0; i < len(expectedMessages); i++ {
			msg := <-overseerCh
			require.Equal(t, msg, expectedMessages[i])
		}
	})

	t.Run("should_fail_when_confirmed_candidate_does_not_match", func(t *testing.T) {
		ctrl := gomock.NewController(t)

		relayParent := common.Hash{0x01}
		groupIndex := parachaintypes.GroupIndex(0)

		confirmedCandidate := &confirmedCandidate{}
		h, err := confirmedCandidate.receipt.Hash()
		require.NoError(t, err)

		candidateHash := parachaintypes.CandidateHash{Value: h}
		sessionState := &perSessionState{
			groups: newGroups([][]parachaintypes.ValidatorIndex{{0, 1, 2}}, 1),
		}

		stmtStore := NewMockstatementStore(ctrl)
		// returning a compact statement with a candidate hash
		// that when encoded generates a different encoding from the
		// confirmed candidate we passed to sendBackingFreshStatements
		stmtStore.EXPECT().
			freshStatementsForBacking([]parachaintypes.ValidatorIndex{0, 1, 2}, candidateHash).
			Return([]parachaintypes.SignedStatement{
				parachaintypes.SignedStatement(
					parachaintypes.UncheckedSignedCompactStatement{
						ValidatorIndex: parachaintypes.ValidatorIndex(0),
						Payload: *parachaintypes.NewCompactSeconded(
							parachaintypes.CandidateHash{Value: common.Hash{0xab, 0xab}}).ToEncodable(),
					},
				),
			})

		relayParentState := &perRelayParentState{
			statementStore: stmtStore,
		}

		sd := &StatementDistribution{}
		err = sd.sendBackingFreshStatements(
			candidateHash,
			groupIndex,
			relayParent,
			relayParentState,
			confirmedCandidate,
			sessionState,
		)

		require.ErrorIs(t, err, errEncodedStatementsDoesNotMatch)
	})
}

func TestSendPendingGridMessages(t *testing.T) {
	t.Run("nil_local_validator", func(t *testing.T) {
		rpHash := common.Hash{0xab}
		peerID := peer.ID("peer-ex")
		validationVersion := validationprotocol.ValidationVersionV3
		peerValidatorID := parachaintypes.ValidatorIndex(0)
		rpState := &perRelayParentState{
			localValidator: nil,
		}

		sd := StatementDistribution{}
		err := sd.sendPendingGridMessages(rpHash, peerID,
			validationVersion, peerValidatorID, nil,
			rpState, nil,
		)

		require.ErrorIs(t, err, errUnkownLocalValidator)
	})

	t.Run("empty_pending_manifests_for_validator_id", func(t *testing.T) {
		gt := newGridTracker()

		rpHash := common.Hash{0xab}
		peerID := peer.ID("peer-ex")
		validationVersion := validationprotocol.ValidationVersionV3
		peerValidatorID := parachaintypes.ValidatorIndex(0)
		rpState := &perRelayParentState{
			localValidator: &localValidatorStore{
				gridTracker: gt,
			},
		}

		sd := StatementDistribution{}

		err := sd.sendPendingGridMessages(rpHash, peerID,
			validationVersion, peerValidatorID, nil,
			rpState, nil,
		)
		require.Nil(t, err)
	})

	t.Run("pending_stmts_but_none_confirmed", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		peerValidatorID := parachaintypes.ValidatorIndex(0)

		gt := newGridTracker()
		gt.pendingManifests[peerValidatorID] = manifestKindByCandidateHash(
			map[parachaintypes.CandidateHash]manifestKind{
				{Value: common.Hash{0x12}}: full,
				{Value: common.Hash{0xab}}: full,
			},
		)

		rpHash := common.Hash{0xab}
		peerID := peer.ID("peer-ex")
		validationVersion := validationprotocol.ValidationVersionV3
		rpState := &perRelayParentState{
			localValidator: &localValidatorStore{
				gridTracker: gt,
			},
		}

		candidatesMock := NewMockcandidatesStore(ctrl)
		candidatesMock.EXPECT().
			getConfirmed(parachaintypes.CandidateHash{Value: common.Hash{0x12}}).
			Return(nil, false)
		candidatesMock.EXPECT().
			getConfirmed(parachaintypes.CandidateHash{Value: common.Hash{0xab}}).
			Return(nil, false)

		sd := StatementDistribution{}

		err := sd.sendPendingGridMessages(rpHash, peerID,
			validationVersion, peerValidatorID, nil,
			rpState, candidatesMock,
		)
		require.Nil(t, err)
	})

	t.Run("pending_full_manifest_confirmed", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		peerValidatorID := parachaintypes.ValidatorIndex(4)

		gt := newGridTracker()
		gt.pendingManifests[peerValidatorID] = manifestKindByCandidateHash(
			map[parachaintypes.CandidateHash]manifestKind{
				{Value: common.Hash{0x12}}: full,
				{Value: common.Hash{0xab}}: full,
			},
		)

		candidatesMock := NewMockcandidatesStore(ctrl)
		candidatesMock.EXPECT().
			getConfirmed(parachaintypes.CandidateHash{Value: common.Hash{0x12}}).
			Return(&confirmedCandidate{
				assignedGroup: parachaintypes.GroupIndex(1),
				receipt: parachaintypes.CommittedCandidateReceiptV2{
					Descriptor: parachaintypes.CandidateDescriptorV2{
						ParaID: parachaintypes.ParaID(10),
					},
				},
				parentHash: common.Hash(bytes.Repeat([]byte{0xbc}, 32)),
			}, true)
		candidatesMock.EXPECT().
			getConfirmed(parachaintypes.CandidateHash{Value: common.Hash{0xab}}).
			Return(nil, false)

		gps := newGroups([][]parachaintypes.ValidatorIndex{
			{0, 1, 2},
			{3, 4, 5},
		}, 1)

		rpHash := common.Hash{0xab}
		peerID := peer.ID("peer-ex")
		v3 := validationprotocol.ValidationVersionV3

		var f *parachaintypes.StatementFilter
		stmtStoreMock := NewMockstatementStore(ctrl)
		stmtStoreMock.EXPECT().
			fillStatementFilter(
				parachaintypes.GroupIndex(1),
				parachaintypes.CandidateHash{Value: common.Hash{0x12}},
				gomock.AssignableToTypeOf((*parachaintypes.StatementFilter)(nil)),
			).
			Do(func(_ parachaintypes.GroupIndex, _ parachaintypes.CandidateHash, filter *parachaintypes.StatementFilter) {
				require.NoError(t, filter.SecondedInGroup.Set(1, true))
				f = filter
			})

		rpState := &perRelayParentState{
			localValidator: &localValidatorStore{
				gridTracker: gt,
			},
			statementStore: stmtStoreMock,
		}

		overseer := make(chan any, 1)
		sd := StatementDistribution{
			SubSystemToOverseer: overseer,
		}

		err := sd.sendPendingGridMessages(rpHash, peerID,
			v3, peerValidatorID, gps,
			rpState, candidatesMock,
		)
		require.Nil(t, err)

		require.Equal(t, 1, len(overseer))
		outgoingMessage := <-overseer

		// building the validation protocol exepected message
		manifest := validationprotocol.BackedCandidateManifest{
			RelayParent:        rpHash,
			CandidateHash:      parachaintypes.CandidateHash{Value: common.Hash{0x12}},
			GroupIndex:         parachaintypes.GroupIndex(1),
			ParaID:             parachaintypes.ParaID(10),
			ParentHeadDataHash: common.Hash(bytes.Repeat([]byte{0xbc}, 32)),
			StatementKnwoledge: *f,
		}

		sdm := validationprotocol.NewStatementDistributionMessage()
		require.NoError(t, sdm.SetValue(manifest))

		expectedMessage := validationprotocol.NewValidationProtocolVDT()
		require.NoError(t, expectedMessage.SetValue(
			validationprotocol.StatementDistribution{StatementDistributionMessage: sdm}))

		require.Equal(t, networkbridgemessages.SendValidationMessages{
			Messages: []*networkbridgemessages.SendValidationMessage{
				{
					To:                        []peer.ID{peerID},
					ValidationProtocolMessage: expectedMessage,
				},
			},
		}, outgoingMessage)
	})

	t.Run("pending_full_and_ack_manifest_confirmed", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		peerValidatorID := parachaintypes.ValidatorIndex(4)

		gt := newGridTracker()
		gt.pendingManifests[peerValidatorID] = manifestKindByCandidateHash(
			map[parachaintypes.CandidateHash]manifestKind{
				{Value: common.Hash{0x12}}: full,
				{Value: common.Hash{0xab}}: acknowledgement,
			},
		)

		candidatesMock := NewMockcandidatesStore(ctrl)
		candidatesMock.EXPECT().
			getConfirmed(parachaintypes.CandidateHash{Value: common.Hash{0x12}}).
			Return(&confirmedCandidate{
				assignedGroup: parachaintypes.GroupIndex(1),
				receipt: parachaintypes.CommittedCandidateReceiptV2{
					Descriptor: parachaintypes.CandidateDescriptorV2{
						ParaID: parachaintypes.ParaID(10),
					},
				},
				parentHash: common.Hash(bytes.Repeat([]byte{0xbc}, 32)),
			}, true)
		candidatesMock.EXPECT().
			getConfirmed(parachaintypes.CandidateHash{Value: common.Hash{0xab}}).
			Return(&confirmedCandidate{
				assignedGroup: parachaintypes.GroupIndex(0),
				receipt: parachaintypes.CommittedCandidateReceiptV2{
					Descriptor: parachaintypes.CandidateDescriptorV2{
						ParaID: parachaintypes.ParaID(11),
					},
				},
				parentHash: common.Hash(bytes.Repeat([]byte{0xee}, 32)),
			}, true)

		gps := newGroups([][]parachaintypes.ValidatorIndex{
			{0, 1, 2},
			{3, 4, 5},
		}, 1)

		rpHash := common.Hash{0xab}
		peerID := peer.ID("peer-ex")
		v3 := validationprotocol.ValidationVersionV3

		var fstKnowledge *parachaintypes.StatementFilter
		var sndKnowledge *parachaintypes.StatementFilter

		stmtStoreMock := NewMockstatementStore(ctrl)
		stmtStoreMock.EXPECT().
			fillStatementFilter(
				parachaintypes.GroupIndex(1),
				parachaintypes.CandidateHash{Value: common.Hash{0x12}},
				gomock.AssignableToTypeOf((*parachaintypes.StatementFilter)(nil)),
			).
			Do(func(_ parachaintypes.GroupIndex, _ parachaintypes.CandidateHash, filter *parachaintypes.StatementFilter) {
				require.NoError(t, filter.SecondedInGroup.Set(1, true))
				fstKnowledge = filter
			})

		stmtStoreMock.EXPECT().
			fillStatementFilter(
				parachaintypes.GroupIndex(0),
				parachaintypes.CandidateHash{Value: common.Hash{0xab}},
				gomock.AssignableToTypeOf((*parachaintypes.StatementFilter)(nil)),
			).
			Do(func(_ parachaintypes.GroupIndex, _ parachaintypes.CandidateHash, filter *parachaintypes.StatementFilter) {
				require.NoError(t, filter.SecondedInGroup.Set(0, true))
				sndKnowledge = filter
			})

		rpState := &perRelayParentState{
			localValidator: &localValidatorStore{
				gridTracker: gt,
			},
			statementStore: stmtStoreMock,
		}

		overseer := make(chan any, 1)
		sd := StatementDistribution{
			SubSystemToOverseer: overseer,
		}

		err := sd.sendPendingGridMessages(rpHash, peerID,
			v3, peerValidatorID, gps,
			rpState, candidatesMock,
		)
		require.Nil(t, err)

		require.Equal(t, 1, len(overseer))
		outgoingMessage := <-overseer

		// building the validation protocol exepected message
		manifest := validationprotocol.BackedCandidateManifest{
			RelayParent:        rpHash,
			CandidateHash:      parachaintypes.CandidateHash{Value: common.Hash{0x12}},
			GroupIndex:         parachaintypes.GroupIndex(1),
			ParaID:             parachaintypes.ParaID(10),
			ParentHeadDataHash: common.Hash(bytes.Repeat([]byte{0xbc}, 32)),
			StatementKnwoledge: *fstKnowledge,
		}

		ack := validationprotocol.BackedCandidateKnown{
			CandidateHash:      parachaintypes.CandidateHash{Value: common.Hash{0xab}},
			StatementKnwoledge: *sndKnowledge,
		}

		manifestSDM := validationprotocol.NewStatementDistributionMessage()
		require.NoError(t, manifestSDM.SetValue(manifest))

		ackSDM := validationprotocol.NewStatementDistributionMessage()
		require.NoError(t, ackSDM.SetValue(ack))

		manifestVPMessage := validationprotocol.NewValidationProtocolVDT()
		require.NoError(t, manifestVPMessage.SetValue(
			validationprotocol.StatementDistribution{StatementDistributionMessage: manifestSDM}))

		ackVPMessage := validationprotocol.NewValidationProtocolVDT()
		require.NoError(t, ackVPMessage.SetValue(
			validationprotocol.StatementDistribution{StatementDistributionMessage: ackSDM}))

		require.Equal(t, networkbridgemessages.SendValidationMessages{
			Messages: []*networkbridgemessages.SendValidationMessage{
				{
					To:                        []peer.ID{peerID},
					ValidationProtocolMessage: manifestVPMessage,
				},
				{
					To:                        []peer.ID{peerID},
					ValidationProtocolMessage: ackVPMessage,
				},
			},
		}, outgoingMessage)
	})

	// TODO: include tests for postAcknowledgementStatementMessages that
	// integrates grid tracker pending statements and statement store
}

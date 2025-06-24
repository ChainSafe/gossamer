// Copyright 2025 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package statementdistribution

import (
	"bytes"
	"testing"

	"github.com/ChainSafe/gossamer/dot/parachain/backing"
	networkbridgemessages "github.com/ChainSafe/gossamer/dot/parachain/network-bridge/messages"
	parachaintypes "github.com/ChainSafe/gossamer/dot/parachain/types"
	"github.com/ChainSafe/gossamer/dot/parachain/util"
	validationprotocol "github.com/ChainSafe/gossamer/dot/parachain/validation-protocol"
	"github.com/ChainSafe/gossamer/dot/peerset"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestSendBackingFreshStatements(t *testing.T) {
	t.Run("should_send_3_statements_to_backing", func(t *testing.T) {
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

		groups := newGroups(initGroups, 1)

		sessionState := &perSessionState{
			groups: groups,
		}

		stmtStore := newStatementStore(groups)
		{
			stmts := []parachaintypes.SignedStatement{
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
			}

			for _, stmt := range stmts {
				newStmt, err := stmtStore.insert(groups, &stmt, statementOriginRemote)
				require.NoError(t, err)
				require.True(t, newStmt)
			}
		}

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
		relayParent := common.Hash{0x01}
		groupIndex := parachaintypes.GroupIndex(0)

		confirmedCandidate := &confirmedCandidate{}
		h, err := confirmedCandidate.receipt.Hash()
		require.NoError(t, err)

		candidateHash := parachaintypes.CandidateHash{Value: h}
		groups := newGroups([][]parachaintypes.ValidatorIndex{{0, 1, 2}}, 1)

		sessionState := &perSessionState{
			groups: groups,
		}

		stmtStore := newStatementStore(groups)
		// returning a compact statement with a candidate hash
		// that when encoded generates a different encoding from the
		// confirmed candidate we passed to sendBackingFreshStatements
		freshStmt := parachaintypes.SignedStatement(
			parachaintypes.UncheckedSignedCompactStatement{
				ValidatorIndex: parachaintypes.ValidatorIndex(0),
				Payload: *parachaintypes.NewCompactSeconded(
					parachaintypes.CandidateHash{Value: common.Hash{0xab, 0xab}}).ToEncodable(),
			},
		)

		fp := fingerprint{
			validator:     parachaintypes.ValidatorIndex(1),
			kind:          fingerprintKindCompactSeconded,
			candidateHash: candidateHash,
		}
		stmtStore.knownStmts[fp] = &storedStatement{
			stmt:           &freshStmt,
			knownByBacking: false,
		}

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

		require.ErrorIs(t, err, errEncodedStatementsDoNotMatch)
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

		stmtStore := newStatementStore(gps)

		seconded, err := parachaintypes.NewBitVec([]bool{false, true, false})
		require.NoError(t, err)

		valid, err := parachaintypes.NewBitVec([]bool{false, false, false})
		require.NoError(t, err)

		stmtStore.groupStmts[groupAndCandidateHash{
			groupIdx:      parachaintypes.GroupIndex(1),
			candidateHash: parachaintypes.CandidateHash{Value: common.Hash{0x12}},
		}] = &groupStatements{
			seconded,
			valid,
		}

		rpState := &perRelayParentState{
			localValidator: &localValidatorStore{
				gridTracker: gt,
			},
			statementStore: stmtStore,
		}

		overseer := make(chan any, 1)
		sd := StatementDistribution{
			SubSystemToOverseer: overseer,
		}

		err = sd.sendPendingGridMessages(rpHash, peerID,
			v3, peerValidatorID, gps,
			rpState, candidatesMock,
		)
		require.Nil(t, err)

		require.Len(t, overseer, 1)
		outgoingMessage := <-overseer

		svm, ok := outgoingMessage.(networkbridgemessages.SendValidationMessages)
		require.True(t, ok)
		require.Len(t, svm.Messages, 1)

		svmVal, err := svm.Messages[0].ValidationProtocolMessage.Value()
		require.NoError(t, err)

		sdm, ok := svmVal.(validationprotocol.StatementDistribution)
		require.True(t, ok)

		sdmVal, err := sdm.StatementDistributionMessage.Value()
		require.NoError(t, err)

		bcm, ok := sdmVal.(validationprotocol.BackedCandidateManifest)
		require.True(t, ok)

		require.Equal(t, rpHash, bcm.RelayParent)
		require.Equal(t, parachaintypes.CandidateHash{Value: common.Hash{0x12}}, bcm.CandidateHash)
		require.Equal(t, parachaintypes.GroupIndex(1), bcm.GroupIndex)
		require.Equal(t, parachaintypes.ParaID(10), bcm.ParaID)
		require.Equal(t, common.Hash(bytes.Repeat([]byte{0xbc}, 32)), bcm.ParentHeadDataHash)
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

		stmtStore := newStatementStore(gps)

		seconded, err := parachaintypes.NewBitVec([]bool{false, true, false})
		require.NoError(t, err)

		valid, err := parachaintypes.NewBitVec([]bool{false, false, false})
		require.NoError(t, err)

		stmtStore.groupStmts[groupAndCandidateHash{
			groupIdx:      parachaintypes.GroupIndex(1),
			candidateHash: parachaintypes.CandidateHash{Value: common.Hash{0x12}},
		}] = &groupStatements{
			seconded,
			valid,
		}

		seconded, err = parachaintypes.NewBitVec([]bool{true, false, false})
		require.NoError(t, err)

		stmtStore.groupStmts[groupAndCandidateHash{
			groupIdx:      parachaintypes.GroupIndex(0),
			candidateHash: parachaintypes.CandidateHash{Value: common.Hash{0xab}},
		}] = &groupStatements{
			seconded,
			valid,
		}

		rpState := &perRelayParentState{
			localValidator: &localValidatorStore{
				gridTracker: gt,
			},
			statementStore: stmtStore,
		}

		overseer := make(chan any, 1)
		sd := StatementDistribution{
			SubSystemToOverseer: overseer,
		}

		err = sd.sendPendingGridMessages(rpHash, peerID,
			v3, peerValidatorID, gps,
			rpState, candidatesMock,
		)
		require.Nil(t, err)

		require.Equal(t, 1, len(overseer))
		outgoingMessage := <-overseer

		svm, ok := outgoingMessage.(networkbridgemessages.SendValidationMessages)
		require.True(t, ok)
		require.Len(t, svm.Messages, 2)

		firstSvm, err := svm.Messages[0].ValidationProtocolMessage.Value()
		require.NoError(t, err)

		firstSdm, ok := firstSvm.(validationprotocol.StatementDistribution)
		require.True(t, ok)

		firstSdmVal, err := firstSdm.StatementDistributionMessage.Value()
		require.NoError(t, err)

		secondSvm, err := svm.Messages[1].ValidationProtocolMessage.Value()
		require.NoError(t, err)

		secondSdm, ok := secondSvm.(validationprotocol.StatementDistribution)
		require.True(t, ok)

		secondSdmVal, err := secondSdm.StatementDistributionMessage.Value()
		require.NoError(t, err)

		var bcm validationprotocol.BackedCandidateManifest
		var bck validationprotocol.BackedCandidateKnown

		// order in svm.Messages is random
		bcm, ok = firstSdmVal.(validationprotocol.BackedCandidateManifest)
		if ok {
			bck, ok = secondSdmVal.(validationprotocol.BackedCandidateKnown)
			require.True(t, ok)
		} else {
			bck, ok = firstSdmVal.(validationprotocol.BackedCandidateKnown)
			require.True(t, ok)

			bcm, ok = secondSdmVal.(validationprotocol.BackedCandidateManifest)
			require.True(t, ok)
		}

		require.Equal(t, rpHash, bcm.RelayParent)
		require.Equal(t, parachaintypes.CandidateHash{Value: common.Hash{0x12}}, bcm.CandidateHash)
		require.Equal(t, parachaintypes.GroupIndex(1), bcm.GroupIndex)
		require.Equal(t, parachaintypes.ParaID(10), bcm.ParaID)
		require.Equal(t, common.Hash(bytes.Repeat([]byte{0xbc}, 32)), bcm.ParentHeadDataHash)

		require.Equal(t, parachaintypes.CandidateHash{Value: common.Hash{0xab}}, bck.CandidateHash)
	})

	// TODO: include tests for postAcknowledgementStatementMessages that
	// integrates grid tracker pending statements and statement store
}

func TestHandleIncomingManifestCommon(t *testing.T) {
	pierre := peer.ID("pierre")
	candidateHash := parachaintypes.CandidateHash{Value: common.Hash{0x12}}
	relayParent := common.Hash{0xab}
	paraID := parachaintypes.ParaID(10)

	t.Run("peer_not_connected", func(t *testing.T) {
		sd := StatementDistribution{}

		importSuccess := sd.handleIncomingManifestCommon(
			pierre,
			make(map[peer.ID]peerState),
			make(map[common.Hash]perRelayParentState),
			make(map[parachaintypes.SessionIndex]perSessionState),
			candidates{},
			candidateHash,
			relayParent,
			paraID,
			manifestSummary{},
			full,
			nil,
		)

		require.Nil(t, importSuccess)
	})

	t.Run("not_in_relay_parent_state", func(t *testing.T) {
		peers := map[peer.ID]peerState{
			pierre: {},
		}

		repAgg := util.NewReputationAggregator(func(rep util.UnifiedReputationChange) bool { return false })

		overseer := make(chan any, 1)
		sd := StatementDistribution{
			SubSystemToOverseer: overseer,
		}

		importSuccess := sd.handleIncomingManifestCommon(
			pierre,
			peers,
			make(map[common.Hash]perRelayParentState),
			make(map[parachaintypes.SessionIndex]perSessionState),
			candidates{},
			candidateHash,
			relayParent,
			paraID,
			manifestSummary{},
			full,
			repAgg,
		)

		require.Nil(t, importSuccess)

		repAgg.Send(overseer)
		require.Len(t, overseer, 1)

		msg := <-overseer

		repMsg, ok := msg.(networkbridgemessages.ReportPeer)
		require.True(t, ok)
		require.Equal(t, pierre, repMsg.PeerID)

		require.Equal(
			t,
			peerset.Reputation(costUnexpectedManifestMissingKnowledge.CostOrBenefit()),
			repMsg.ReputationChange.Value,
		)

		require.Equal(t, costUnexpectedManifestMissingKnowledge.Reason, repMsg.ReputationChange.Reason)
	})

	t.Run("happy_path", func(t *testing.T) {
		groupIndex := parachaintypes.GroupIndex(0)

		manifestSummary := manifestSummary{
			claimedGroupIndex: groupIndex,
			claimedParentHash: common.Hash{0x03},
		}

		seconded, err := parachaintypes.NewBitVec([]bool{true, false, false})
		require.NoError(t, err)
		valid, err := parachaintypes.NewBitVec([]bool{false, true, false})
		require.NoError(t, err)

		manifestSummary.statementKnowledge = parachaintypes.StatementFilter{
			SecondedInGroup:  seconded,
			ValidatedInGroup: valid,
		}

		authKey := [32]byte{0x04}
		peerStateEntry := peerState{
			discoveryIds: &map[parachaintypes.AuthorityDiscoveryID]struct{}{authKey: {}},
		}
		peers := map[peer.ID]peerState{
			pierre: peerStateEntry,
		}

		gt := newGridTracker()
		localValidator := &localValidatorStore{
			gridTracker: gt,
		}

		initGroups := [][]parachaintypes.ValidatorIndex{
			{0, 1, 2},
			{3, 4, 5},
		}
		groups := newGroups(initGroups, 2)

		relayParentState := perRelayParentState{
			session:        1,
			localValidator: localValidator,
			groupsPerPara:  map[parachaintypes.ParaID][]parachaintypes.GroupIndex{paraID: {groupIndex}},
			assignmentsPerGroup: map[parachaintypes.GroupIndex][]parachaintypes.ParaID{
				groupIndex: {paraID},
			},
		}
		perRelayParent := map[common.Hash]perRelayParentState{
			relayParent: relayParentState,
		}

		validatorIndex := parachaintypes.ValidatorIndex(1)
		gridTopology := &sessionTopologyView{
			groupViews: map[parachaintypes.GroupIndex]groupSubView{
				groupIndex: {
					sending:   make(map[parachaintypes.ValidatorIndex]struct{}),
					receiving: map[parachaintypes.ValidatorIndex]struct{}{validatorIndex: {}},
				},
			},
		}

		sessionInfo := parachaintypes.SessionInfo{
			DiscoveryKeys: []parachaintypes.AuthorityDiscoveryID{
				{0x0a},  // Validator 0
				authKey, // Validator 1 - matches our peer's authority key
				{0x0c},  // Validator 2
			},
		}

		sessionState := perSessionState{
			gridView:       gridTopology,
			groups:         groups,
			sessionInfo:    sessionInfo,
			localValidator: &validatorIndex,
		}
		perSession := map[parachaintypes.SessionIndex]perSessionState{
			1: sessionState,
		}

		candidates := candidates{
			candidates: make(map[parachaintypes.CandidateHash]candidateState),
			byParent:   make(map[hashAndParaID]map[parachaintypes.CandidateHash]struct{}),
		}

		reputation := util.NewReputationAggregator(func(rep util.UnifiedReputationChange) bool { return false })

		overseerCh := make(chan any, 1)
		sd := &StatementDistribution{
			SubSystemToOverseer: overseerCh,
		}

		importSuccess := sd.handleIncomingManifestCommon(
			pierre,
			peers,
			perRelayParent,
			perSession,
			candidates,
			candidateHash,
			relayParent,
			paraID,
			manifestSummary,
			full,
			reputation,
		)

		require.NotNil(t, importSuccess)
		require.Equal(t, relayParentState, importSuccess.relayParentState)
		require.Equal(t, sessionState, importSuccess.perSession)
		require.Equal(t, validatorIndex, importSuccess.senderIndex)

		// Verify the candidate was added as unconfirmed in candidate store and grid tracker

		_, ok := candidates.candidates[candidateHash]
		require.True(t, ok)

		validatorAndGroups, ok := gt.unconfirmed[candidateHash]
		require.True(t, ok)
		require.Len(t, validatorAndGroups, 1)
		require.Equal(t, validatorIndex, validatorAndGroups[0].validator)
		require.Equal(t, groupIndex, validatorAndGroups[0].group)
	})
}

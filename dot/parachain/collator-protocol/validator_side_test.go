// Copyright 2023 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package collatorprotocol

import (
	"testing"

	collatorprotocolmessages "github.com/ChainSafe/gossamer/dot/parachain/collator-protocol/messages"
	parachaintypes "github.com/ChainSafe/gossamer/dot/parachain/types"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/stretchr/testify/require"
	gomock "go.uber.org/mock/gomock"
)

func TestProcessOverseerMessage(t *testing.T) {
	t.Parallel()

	var testCollatorID parachaintypes.CollatorID
	tempCollatID := common.MustHexToBytes("0x48215b9d322601e5b1a95164cea0dc4626f545f98343d07f1551eb9543c4b147")
	copy(testCollatorID[:], tempCollatID)
	peerID := peer.ID("testPeerID")
	testRelayParent := getDummyHash(5)

	// commitments := parachaintypes.CandidateCommitments{
	// 	UpwardMessages:    []parachaintypes.UpwardMessage{{1, 2, 3}},
	// 	NewValidationCode: &parachaintypes.ValidationCode{1, 2, 3},
	// 	HeadData: parachaintypes.HeadData{
	// 		Data: []byte{1, 2, 3},
	// 	},
	// 	ProcessedDownwardMessages: uint32(5),
	// 	HrmpWatermark:             uint32(0),
	// }

	// testCandidateReceipt := parachaintypes.CandidateReceipt{
	// 	Descriptor: parachaintypes.CandidateDescriptor{
	// 		ParaID:                      uint32(1000),
	// 		RelayParent:                 common.MustHexToHash("0xded542bacb3ca6c033a57676f94ae7c8f36834511deb44e3164256fd3b1c0de0"), //nolint:lll
	// 		Collator:                    testCollatorID,
	// 		PersistedValidationDataHash: common.MustHexToHash("0x690d8f252ef66ab0f969c3f518f90012b849aa5ac94e1752c5e5ae5a8996de37"), //nolint:lll
	// 		PovHash:                     common.MustHexToHash("0xe7df1126ac4b4f0fb1bc00367a12ec26ca7c51256735a5e11beecdc1e3eca274"), //nolint:lll
	// 		ErasureRoot:                 common.MustHexToHash("0xc07f658163e93c45a6f0288d229698f09c1252e41076f4caa71c8cbc12f118a1"), //nolint:lll
	// 		ParaHead:                    common.MustHexToHash("0x9a8a7107426ef873ab89fc8af390ec36bdb2f744a9ff71ad7f18a12d55a7f4f5"), //nolint:lll
	// 	},

	// 	CommitmentsHash: commitments.Hash(),
	// }

	testCases := []struct {
		description           string
		msg                   any
		peerData              map[peer.ID]PeerData
		net                   Network
		fetchedCandidates     map[string]CollationEvent
		deletesFetchCandidate bool
		blockedAdvertisements map[string][]BlockedAdvertisement
		errString             string
	}{
		// {
		// 	description: "CollateOn message fails with message not expected",
		// 	msg:         collatorprotocolmessages.CollateOn(2),
		// 	errString:   ErrNotExpectedOnValidatorSide.Error(),
		// },
		// {
		// 	description: "DistributeCollation message fails with message not expected",
		// 	msg:         collatorprotocolmessages.DistributeCollation{},
		// 	errString:   ErrNotExpectedOnValidatorSide.Error(),
		// },
		// {
		// 	description: "ReportCollator message fails with peer not found for collator",
		// 	msg:         collatorprotocolmessages.ReportCollator(testCollatorID),
		// 	errString:   ErrPeerIDNotFoundForCollator.Error(),
		// },
		// {
		// 	description: "ReportCollator message succeeds and reports a bad collator",
		// 	msg:         collatorprotocolmessages.ReportCollator(testCollatorID),
		// 	net: func() Network {
		// 		ctrl := gomock.NewController(t)
		// 		net := NewMockNetwork(ctrl)
		// 		net.EXPECT().ReportPeer(peerset.ReputationChange{
		// 			Value:  peerset.ReportBadCollatorValue,
		// 			Reason: peerset.ReportBadCollatorReason,
		// 		}, peerID)

		// 		return net
		// 	}(),
		// 	peerData: map[peer.ID]PeerData{
		// 		peerID: {
		// 			view: View{},
		// 			state: PeerStateInfo{
		// 				PeerState: Collating,
		// 				CollatingPeerState: CollatingPeerState{
		// 					CollatorID: testCollatorID,
		// 					ParaID:     parachaintypes.ParaID(6),
		// 				},
		// 			},
		// 		},
		// 	},
		// 	errString: "",
		// },
		// {
		// 	description: "InvalidOverseerMsg message fails with peer not found for collator",
		// 	msg: collatorprotocolmessages.Invalid{
		// 		Parent:           testRelayParent,
		// 		CandidateReceipt: testCandidateReceipt,
		// 	},
		// 	fetchedCandidates: func() map[string]CollationEvent {
		// 		fetchedCollation, err := newFetchedCollationInfo(testCandidateReceipt)
		// 		require.NoError(t, err)

		// 		return map[string]CollationEvent{
		// 			fetchedCollation.String(): {
		// 				CollatorId: testCandidateReceipt.Descriptor.Collator,
		// 				PendingCollation: PendingCollation{
		// 					CommitmentHash: &testCandidateReceipt.CommitmentsHash,
		// 				},
		// 			},
		// 		}
		// 	}(),
		// 	deletesFetchCandidate: true,
		// 	errString:             ErrPeerIDNotFoundForCollator.Error(),
		// },
		// {
		// 	description: "InvalidOverseerMsg message succeeds, reports a bad collator and removes fetchedCandidate",
		// 	msg: collatorprotocolmessages.Invalid{
		// 		Parent:           testRelayParent,
		// 		CandidateReceipt: testCandidateReceipt,
		// 	},
		// 	net: func() Network {
		// 		ctrl := gomock.NewController(t)
		// 		net := NewMockNetwork(ctrl)
		// 		net.EXPECT().ReportPeer(peerset.ReputationChange{
		// 			Value:  peerset.ReportBadCollatorValue,
		// 			Reason: peerset.ReportBadCollatorReason,
		// 		}, peerID)

		// 		return net
		// 	}(),
		// 	fetchedCandidates: func() map[string]CollationEvent {
		// 		fetchedCollation, err := newFetchedCollationInfo(testCandidateReceipt)
		// 		require.NoError(t, err)

		// 		return map[string]CollationEvent{
		// 			fetchedCollation.String(): {
		// 				CollatorId: testCandidateReceipt.Descriptor.Collator,
		// 				PendingCollation: PendingCollation{
		// 					CommitmentHash: &testCandidateReceipt.CommitmentsHash,
		// 				},
		// 			},
		// 		}
		// 	}(),
		// 	peerData: map[peer.ID]PeerData{
		// 		peerID: {
		// 			view: View{},
		// 			state: PeerStateInfo{
		// 				PeerState: Collating,
		// 				CollatingPeerState: CollatingPeerState{
		// 					CollatorID: testCollatorID,
		// 					ParaID:     parachaintypes.ParaID(6),
		// 				},
		// 			},
		// 		},
		// 	},
		// 	deletesFetchCandidate: true,
		// 	errString:             "",
		// },
		// {
		// 	description: "SecondedOverseerMsg message fails with peer not found for collator and removes fetchedCandidate",
		// 	msg: collatorprotocolmessages.Seconded{
		// 		Parent: testRelayParent,
		// 		Stmt: func() parachaintypes.UncheckedSignedFullStatement {
		// 			vdt := parachaintypes.NewStatementVDT()
		// 			vdt.Set(parachaintypes.Seconded(
		// 				parachaintypes.CommittedCandidateReceipt{
		// 					Descriptor:  testCandidateReceipt.Descriptor,
		// 					Commitments: commitments,
		// 				},
		// 			))
		// 			return parachaintypes.UncheckedSignedFullStatement{
		// 				Payload: vdt,
		// 			}
		// 		}(),
		// 	},
		// 	fetchedCandidates: func() map[string]CollationEvent {
		// 		fetchedCollation, err := newFetchedCollationInfo(testCandidateReceipt)
		// 		require.NoError(t, err)
		// 		return map[string]CollationEvent{
		// 			fetchedCollation.String(): {
		// 				CollatorId: testCandidateReceipt.Descriptor.Collator,
		// 				PendingCollation: PendingCollation{
		// 					CommitmentHash: &testCandidateReceipt.CommitmentsHash,
		// 				},
		// 			},
		// 		}
		// 	}(),
		// 	deletesFetchCandidate: true,
		// 	errString:             ErrPeerIDNotFoundForCollator.Error(),
		// },
		// {
		// 	description: "SecondedOverseerMsg message succceds, reports a good collator and removes fetchedCandidate",
		// 	msg: collatorprotocolmessages.Seconded{
		// 		Parent: testRelayParent,
		// 		Stmt: func() parachaintypes.UncheckedSignedFullStatement {
		// 			vdt := parachaintypes.NewStatementVDT()
		// 			vdt.Set(parachaintypes.Seconded(
		// 				parachaintypes.CommittedCandidateReceipt{
		// 					Descriptor:  testCandidateReceipt.Descriptor,
		// 					Commitments: commitments,
		// 				},
		// 			))
		// 			return parachaintypes.UncheckedSignedFullStatement{
		// 				Payload: vdt,
		// 			}
		// 		}(),
		// 	},
		// 	net: func() Network {
		// 		ctrl := gomock.NewController(t)
		// 		net := NewMockNetwork(ctrl)
		// 		net.EXPECT().ReportPeer(peerset.ReputationChange{
		// 			Value:  peerset.BenefitNotifyGoodValue,
		// 			Reason: peerset.BenefitNotifyGoodReason,
		// 		}, peerID)

		// 		net.EXPECT().SendMessage(peerID, gomock.AssignableToTypeOf(&CollationProtocol{}))

		// 		return net
		// 	}(),
		// 	fetchedCandidates: func() map[string]CollationEvent {
		// 		fetchedCollation, err := newFetchedCollationInfo(testCandidateReceipt)
		// 		require.NoError(t, err)
		// 		return map[string]CollationEvent{
		// 			fetchedCollation.String(): {
		// 				CollatorId: testCandidateReceipt.Descriptor.Collator,
		// 				PendingCollation: PendingCollation{
		// 					CommitmentHash: &testCandidateReceipt.CommitmentsHash,
		// 				},
		// 			},
		// 		}
		// 	}(),
		// 	peerData: map[peer.ID]PeerData{
		// 		peerID: {
		// 			view: View{},
		// 			state: PeerStateInfo{
		// 				PeerState: Collating,
		// 				CollatingPeerState: CollatingPeerState{
		// 					CollatorID: testCollatorID,
		// 					ParaID:     parachaintypes.ParaID(6),
		// 				},
		// 			},
		// 		},
		// 	},
		// 	deletesFetchCandidate: true,
		// 	errString:             "",
		// },
		{
			description: "Backed message fails with unknown relay parent",
			msg: collatorprotocolmessages.Backed{
				ParaID:   parachaintypes.ParaID(6),
				ParaHead: common.Hash{},
			},

			blockedAdvertisements: map[string][]BlockedAdvertisement{
				"para id: 6, para head: 0x0000000000000000000000000000000000000000000000000000000000000000": {
					{
						peerID:               peerID,
						collatorID:           testCollatorID,
						candidateRelayParent: testRelayParent,
						candidateHash:        parachaintypes.CandidateHash{},
					},
				},
				"para id: 7, para head: 0x0000000000000000000000000000000000000000000000000000000000000001": {
					{
						peerID:               peerID,
						collatorID:           testCollatorID,
						candidateRelayParent: testRelayParent,
						candidateHash:        parachaintypes.CandidateHash{},
					},
				},
			},
			errString: ErrRelayParentUnknown.Error(),
		},
	}
	for _, c := range testCases {
		c := c
		t.Run(c.description, func(t *testing.T) {
			t.Parallel()
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			o := NewMockOverseerI(ctrl)
			cpvs := CollatorProtocolValidatorSide{
				net: c.net,
				// SubSystemToOverseer: make(chan<- any),
				overseer: o,
				// perRelayParent: c.perRelayParent,
				fetchedCandidates:     c.fetchedCandidates,
				peerData:              c.peerData,
				BlockedAdvertisements: c.blockedAdvertisements,
				// activeLeaves:   c.activeLeaves,
			}

			lenFetchedCandidatesBefore := len(cpvs.fetchedCandidates)

			err := cpvs.processMessage(c.msg)
			if c.errString == "" {
				require.NoError(t, err)
			} else {
				require.ErrorContains(t, err, c.errString)
			}

			if c.deletesFetchCandidate {
				require.Equal(t, lenFetchedCandidatesBefore-1, len(cpvs.fetchedCandidates))
			} else {
				require.Equal(t, lenFetchedCandidatesBefore, len(cpvs.fetchedCandidates))
			}
		})
	}
}

// TODO: Just Write those black box tests.

// Copyright 2023 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package collatorprotocol

import (
	_ "embed"
	"fmt"
	"testing"

	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/lib/crypto"
	"github.com/ChainSafe/gossamer/lib/crypto/sr25519"
	"github.com/ChainSafe/gossamer/pkg/scale"
	gomock "github.com/golang/mock/gomock"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"

	"github.com/ChainSafe/gossamer/dot/network"
	parachaintypes "github.com/ChainSafe/gossamer/dot/parachain/types"
	"github.com/ChainSafe/gossamer/dot/peerset"
)

//go:embed testdata/collation_protocol.yaml
var testDataCollationProtocolRaw string

var testDataCollationProtocol map[string]string

func init() {
	err := yaml.Unmarshal([]byte(testDataCollationProtocolRaw), &testDataCollationProtocol)
	if err != nil {
		fmt.Println("Error unmarshaling test data:", err)
		return
	}
}

func getDummyHash(num byte) common.Hash {
	hash := common.Hash{}
	for i := 0; i < 32; i++ {
		hash[i] = num
	}
	return hash
}

func TestCollationProtocol(t *testing.T) {
	t.Parallel()

	var collatorID parachaintypes.CollatorID
	tempCollatID := common.MustHexToBytes("0x48215b9d322601e5b1a95164cea0dc4626f545f98343d07f1551eb9543c4b147")
	copy(collatorID[:], tempCollatID)

	var collatorSignature parachaintypes.CollatorSignature
	tempSignature := common.MustHexToBytes(testDataCollationProtocol["collatorSignature"])
	copy(collatorSignature[:], tempSignature)

	var validatorSignature parachaintypes.ValidatorSignature
	copy(validatorSignature[:], tempSignature)

	hash5 := getDummyHash(5)

	secondedEnumValue := parachaintypes.Seconded{
		Descriptor: parachaintypes.CandidateDescriptor{
			ParaID:                      uint32(1),
			RelayParent:                 hash5,
			Collator:                    collatorID,
			PersistedValidationDataHash: hash5,
			PovHash:                     hash5,
			ErasureRoot:                 hash5,
			Signature:                   collatorSignature,
			ParaHead:                    hash5,
			ValidationCodeHash:          parachaintypes.ValidationCodeHash(hash5),
		},
		Commitments: parachaintypes.CandidateCommitments{
			UpwardMessages:    []parachaintypes.UpwardMessage{{1, 2, 3}},
			NewValidationCode: &parachaintypes.ValidationCode{1, 2, 3},
			HeadData: parachaintypes.HeadData{
				Data: []byte{1, 2, 3},
			},
			ProcessedDownwardMessages: uint32(5),
			HrmpWatermark:             uint32(0),
		},
	}

	statementVDTWithSeconded := parachaintypes.NewStatementVDT()
	err := statementVDTWithSeconded.Set(secondedEnumValue)
	require.NoError(t, err)

	testCases := []struct {
		name          string
		enumValue     scale.VaryingDataTypeValue
		encodingValue []byte
	}{
		{
			name: "Declare",
			enumValue: Declare{
				CollatorId:        collatorID,
				ParaID:            uint32(5),
				CollatorSignature: collatorSignature,
			},
			encodingValue: common.MustHexToBytes(testDataCollationProtocol["declare"]),
		},
		{
			name:          "AdvertiseCollation",
			enumValue:     AdvertiseCollation(hash5),
			encodingValue: common.MustHexToBytes("0x00010505050505050505050505050505050505050505050505050505050505050505"),
		},
		{
			name: "CollationSeconded",
			enumValue: CollationSeconded{
				RelayParent: hash5,
				Statement: parachaintypes.UncheckedSignedFullStatement{
					Payload:        statementVDTWithSeconded,
					ValidatorIndex: parachaintypes.ValidatorIndex(5),
					Signature:      validatorSignature,
				},
			},
			encodingValue: common.MustHexToBytes(testDataCollationProtocol["collationSeconded"]),
		},
	}

	for _, c := range testCases {
		c := c
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			t.Run("marshal", func(t *testing.T) {
				t.Parallel()

				vdt_parent := NewCollationProtocol()
				vdt_child := NewCollatorProtocolMessage()

				err := vdt_child.Set(c.enumValue)
				require.NoError(t, err)

				err = vdt_parent.Set(vdt_child)
				require.NoError(t, err)

				bytes, err := scale.Marshal(vdt_parent)
				require.NoError(t, err)

				require.Equal(t, c.encodingValue, bytes)
			})

			t.Run("unmarshal", func(t *testing.T) {
				t.Parallel()

				vdt_parent := NewCollationProtocol()
				err := scale.Unmarshal(c.encodingValue, &vdt_parent)
				require.NoError(t, err)

				vdt_child_temp, err := vdt_parent.Value()
				require.NoError(t, err)
				require.Equal(t, uint(0), vdt_child_temp.Index())

				vdt_child := vdt_child_temp.(CollatorProtocolMessage)
				require.NoError(t, err)

				actualData, err := vdt_child.Value()
				require.NoError(t, err)

				require.Equal(t, c.enumValue.Index(), actualData.Index())
				require.EqualValues(t, c.enumValue, actualData)
			})
		})
	}
}

func TestDecodeCollationHandshake(t *testing.T) {
	t.Parallel()

	testHandshake := &collatorHandshake{}

	enc, err := testHandshake.Encode()
	require.NoError(t, err)
	require.Equal(t, []byte{}, enc)

	msg, err := decodeCollatorHandshake(enc)
	require.NoError(t, err)
	require.Equal(t, testHandshake, msg)
}

func TestHandleCollationMessageCommon(t *testing.T) {
	cpvs := CollatorProtocolValidatorSide{}

	peerID := peer.ID("testPeerID")

	// fail with wrong message type
	msg1 := &network.BlockAnnounceMessage{}
	propagate, err := cpvs.handleCollationMessage(peerID, msg1)
	require.False(t, propagate)
	require.ErrorIs(t, err, ErrUnexpectedMessageOnCollationProtocol)

	// fail if we can't cast the message to type `*CollationProtocol`
	msg2 := NewCollationProtocol()
	propagate, err = cpvs.handleCollationMessage(peerID, msg2)
	require.False(t, propagate)
	require.ErrorContains(t, err, "failed to cast into collator protocol message, expected: *CollationProtocol, got: collatorprotocol.CollationProtocol")

	// fail if no value set in the collator protocol message
	msg3 := NewCollationProtocol()
	propagate, err = cpvs.handleCollationMessage(peerID, &msg3)
	require.False(t, propagate)
	require.ErrorContains(t, err, "getting collator protocol value: varying data type not set")
}

func TestHandleCollationMessageDeclare(t *testing.T) {
	peerID := peer.ID("testPeerID")

	collatorKeypair, err := sr25519.GenerateKeypair()
	require.NoError(t, err)
	collatorID, err := sr25519.NewPublicKey(collatorKeypair.Public().Encode())
	require.NoError(t, err)

	payload := getDeclareSignaturePayload(peerID)
	signatureBytes, err := collatorKeypair.Sign(payload)
	require.NoError(t, err)
	collatorSignature := [sr25519.SignatureLength]byte{}
	copy(collatorSignature[:], signatureBytes)

	var invalidCollatorSignature parachaintypes.CollatorSignature
	tempSignature := common.MustHexToBytes(testDataCollationProtocol["collatorSignature"])
	copy(invalidCollatorSignature[:], tempSignature)

	testCases := []struct {
		description        string
		declareMsg         Declare
		peerData           map[peer.ID]PeerData
		currentAssignments map[parachaintypes.ParaID]uint
		net                Network
		errString          string
	}{
		{
			description: " fail with unknown peer and report the sender if sender is not stored in our peerdata",
			declareMsg:  Declare{},
			net: func() Network {
				ctrl := gomock.NewController(t)
				net := NewMockNetwork(ctrl)
				net.EXPECT().ReportPeer(peerset.ReputationChange{
					Value:  peerset.UnexpectedMessageValue,
					Reason: peerset.UnexpectedMessageReason,
				}, peerID)
				return net
			}(),
			errString: ErrUnknownPeer.Error(),
		},
		{
			description: "report the sender if the collatorId in the Declare message belongs to any peer stored in our peer data",
			declareMsg: Declare{
				CollatorId: collatorID.AsBytes(),
			},
			peerData: map[peer.ID]PeerData{
				peerID: {
					view: View{},
					state: PeerStateInfo{
						PeerState: Collating,
						CollatingPeerState: CollatingPeerState{
							CollatorID: collatorID.AsBytes(),
						},
					},
				},
			},
			net: func() Network {
				ctrl := gomock.NewController(t)
				net := NewMockNetwork(ctrl)
				net.EXPECT().ReportPeer(peerset.ReputationChange{
					Value:  peerset.UnexpectedMessageValue,
					Reason: peerset.UnexpectedMessageReason,
				}, peerID)
				return net
			}(),
		},
		{
			description: "fail if collator signature could not be verified",
			declareMsg: Declare{
				CollatorId:        collatorID.AsBytes(),
				ParaID:            uint32(5),
				CollatorSignature: parachaintypes.CollatorSignature{},
			},
			peerData: map[peer.ID]PeerData{
				peerID: {
					view: View{},
					state: PeerStateInfo{
						PeerState: Connected,
					},
				},
			},
			errString: "verifying signature",
		},
		{
			description: "fail if collator signature is invalid and report the sender",
			declareMsg: Declare{
				CollatorId:        collatorID.AsBytes(),
				ParaID:            uint32(5),
				CollatorSignature: invalidCollatorSignature,
			},
			peerData: map[peer.ID]PeerData{
				peerID: {
					view: View{},
					state: PeerStateInfo{
						PeerState: Connected,
					},
				},
			},
			net: func() Network {
				ctrl := gomock.NewController(t)
				net := NewMockNetwork(ctrl)
				net.EXPECT().ReportPeer(peerset.ReputationChange{
					Value:  peerset.InvalidSignatureValue,
					Reason: peerset.InvalidSignatureReason,
				}, peerID)
				return net
			}(),
			errString: crypto.ErrSignatureVerificationFailed.Error(),
		},
		{
			// TODO: test that we disconnect sender in this case, after we add that functionality
			description: "fail if paraID in Declare message is not assigned to our peer and report the sender",
			declareMsg: Declare{
				CollatorId:        collatorID.AsBytes(),
				ParaID:            uint32(5),
				CollatorSignature: collatorSignature,
			},
			peerData: map[peer.ID]PeerData{
				peerID: {
					view: View{},
					state: PeerStateInfo{
						PeerState: Connected,
					},
				},
			},
			currentAssignments: make(map[parachaintypes.ParaID]uint),
			net: func() Network {
				ctrl := gomock.NewController(t)
				net := NewMockNetwork(ctrl)
				net.EXPECT().ReportPeer(peerset.ReputationChange{
					Value:  peerset.UnneededCollatorValue,
					Reason: peerset.UnneededCollatorReason,
				}, peerID)
				return net
			}(),
		},
		{
			description: "success case: check if PeerState of the sender has changed to Collating from Connected",
			declareMsg: Declare{
				CollatorId:        collatorID.AsBytes(),
				ParaID:            uint32(5),
				CollatorSignature: collatorSignature,
			},
			peerData: map[peer.ID]PeerData{
				peerID: {
					view: View{},
					state: PeerStateInfo{
						PeerState: Connected,
					},
				},
			},
			currentAssignments: map[parachaintypes.ParaID]uint{
				parachaintypes.ParaID(5): 1,
			},
		},
	}

	for _, c := range testCases {
		c := c
		t.Run(c.description, func(t *testing.T) {
			t.Parallel()
			cpvs := CollatorProtocolValidatorSide{
				net:                c.net,
				peerData:           c.peerData,
				currentAssignments: c.currentAssignments,
			}
			msg := NewCollationProtocol()
			vdt_child := NewCollatorProtocolMessage()

			err = vdt_child.Set(c.declareMsg)
			require.NoError(t, err)

			err = msg.Set(vdt_child)
			require.NoError(t, err)

			propagate, err := cpvs.handleCollationMessage(peerID, &msg)
			require.False(t, propagate)
			if c.errString == "" {
				require.NoError(t, err)
			} else {
				require.ErrorContains(t, err, c.errString)
			}
		})
	}
}

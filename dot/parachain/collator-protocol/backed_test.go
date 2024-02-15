// Copyright 2023 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package collatorprotocol

import (
	"testing"

	"github.com/ChainSafe/gossamer/dot/network"
	"github.com/ChainSafe/gossamer/dot/parachain/backing"
	collatorprotocolmessages "github.com/ChainSafe/gossamer/dot/parachain/collator-protocol/messages"
	"github.com/ChainSafe/gossamer/dot/parachain/overseer"
	parachaintypes "github.com/ChainSafe/gossamer/dot/parachain/types"
	types "github.com/ChainSafe/gossamer/dot/types"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/libp2p/go-libp2p/core/peer"
	protocol "github.com/libp2p/go-libp2p/core/protocol"
	"github.com/stretchr/testify/require"
	gomock "go.uber.org/mock/gomock"
)

func TestProcessBackedOverseerMessage(t *testing.T) {
	t.Parallel()

	var testCollatorID parachaintypes.CollatorID
	tempCollatID := common.MustHexToBytes("0x48215b9d322601e5b1a95164cea0dc4626f545f98343d07f1551eb9543c4b147")
	copy(testCollatorID[:], tempCollatID)
	peerID := peer.ID("testPeerID")
	testRelayParent := getDummyHash(5)

	testCases := []struct {
		description                 string
		msg                         any
		deletesBlockedAdvertisement bool
		blockedAdvertisements       map[string][]BlockedAdvertisement
		canSecond                   bool
		errString                   string
	}{
		{
			description: "Backed message fails with unknown relay parent",
			msg: collatorprotocolmessages.Backed{
				ParaID:   parachaintypes.ParaID(6),
				ParaHead: common.Hash{},
			},
			canSecond:                   true,
			deletesBlockedAdvertisement: true,
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
		{
			description: "Backed message gets processed successfully when seconding is not allowed",
			msg: collatorprotocolmessages.Backed{
				ParaID:   parachaintypes.ParaID(6),
				ParaHead: common.Hash{},
			},
			canSecond: false,
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
			errString: "",
		},
	}
	for _, c := range testCases {
		c := c
		t.Run(c.description, func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			mockBlockState := NewMockBlockState(ctrl)
			finalizedNotifierChan := make(chan *types.FinalisationInfo)
			importedBlockNotiferChan := make(chan *types.Block)

			mockBlockState.EXPECT().GetFinalisedNotifierChannel().Return(finalizedNotifierChan)
			mockBlockState.EXPECT().GetImportedBlockNotifierChannel().Return(importedBlockNotiferChan)
			mockBlockState.EXPECT().FreeFinalisedNotifierChannel(finalizedNotifierChan)
			mockBlockState.EXPECT().FreeImportedBlockNotifierChannel(importedBlockNotiferChan)

			overseer := overseer.NewOverseer(mockBlockState)
			err := overseer.Start()
			require.NoError(t, err)

			defer overseer.Stop()

			collationProtocolID := "/6761727661676500000000000000000000000000000000000000000000000000/1/collations/1"

			net := NewMockNetwork(ctrl)
			net.EXPECT().RegisterNotificationsProtocol(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(nil)
			net.EXPECT().GetRequestResponseProtocol(gomock.Any(), collationFetchingRequestTimeout, uint64(collationFetchingMaxResponseSize)).Return(&network.RequestResponseProtocol{})
			cpvs, err := Register(net, protocol.ID(collationProtocolID), overseer.SubsystemsToOverseer)
			require.NoError(t, err)

			cpvs.BlockedAdvertisements = c.blockedAdvertisements

			mockBacking := NewMockSubsystem(ctrl)
			mockBacking.EXPECT().Name().Return(parachaintypes.CandidateBacking)
			overseerToBacking := overseer.RegisterSubsystem(mockBacking)

			go func() {
				msg, _ := (<-overseerToBacking).(backing.CanSecondMessage)
				msg.ResponseCh <- c.canSecond
			}()

			lenBlackedAdvertisementsBefore := len(cpvs.BlockedAdvertisements)

			err = cpvs.processMessage(c.msg)
			if c.errString == "" {
				require.NoError(t, err)
			} else {
				require.ErrorContains(t, err, c.errString)
			}

			if c.deletesBlockedAdvertisement {
				require.Equal(t, lenBlackedAdvertisementsBefore-1, len(cpvs.BlockedAdvertisements))
			} else {
				require.Equal(t, lenBlackedAdvertisementsBefore, len(cpvs.BlockedAdvertisements))
			}
		})
	}
}

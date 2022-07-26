// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package grandpa

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/ChainSafe/gossamer/dot/network"
	"github.com/ChainSafe/gossamer/dot/types"
	"github.com/golang/mock/gomock"

	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGrandpaHandshake_Encode(t *testing.T) {
	hs := &GrandpaHandshake{
		Roles: 4,
	}

	enc, err := hs.Encode()
	require.NoError(t, err)

	res := new(GrandpaHandshake)
	err = res.Decode(enc)
	require.NoError(t, err)
	require.Equal(t, hs, res)

	s := &Service{}
	res2, err := s.decodeHandshake(enc)
	require.NoError(t, err)
	require.Equal(t, hs, res2)
}

func TestHandleNetworkMessage(t *testing.T) {
	gs, st := newTestService(t)

	just := []SignedVote{
		{
			Vote:        *testVote,
			Signature:   testSignature,
			AuthorityID: gs.publicKeyBytes(),
		},
	}
	err := st.Grandpa.SetPrecommits(77, gs.state.setID, just)
	require.NoError(t, err)

	fm, err := gs.newCommitMessage(gs.head, 77)
	require.NoError(t, err)

	cm, err := fm.ToConsensusMessage()
	require.NoError(t, err)
	gs.state.voters = gs.state.voters[:1]

	ctrl := gomock.NewController(t)
	telemetryMock := NewMockClient(ctrl)
	telemetryMock.EXPECT().SendMessage(gomock.Any()).AnyTimes()

	telemetryMock.
		EXPECT().
		SendMessage(gomock.Any()).
		AnyTimes()

	h := NewMessageHandler(gs, st.Block, telemetryMock)
	gs.messageHandler = h

	propagate, err := gs.handleNetworkMessage(peer.ID(""), cm)
	require.NoError(t, err)
	require.True(t, propagate)

	neighbourMsg := &NeighbourMessage{}
	cm, err = neighbourMsg.ToConsensusMessage()
	require.NoError(t, err)

	propagate, err = gs.handleNetworkMessage(peer.ID(""), cm)
	require.NoError(t, err)
	require.False(t, propagate)
}

func TestNotifyNeighbor(t *testing.T) {
	const interval = 2 * time.Second

	tests := map[string]struct {
		notifyInterval     time.Duration
		finalizeBlock      bool
		finalizeBlockAfter time.Duration
		expectWithin       time.Duration
	}{
		"should_send_neighbor_message": {
			expectWithin:   2 * time.Second,
			notifyInterval: interval,
		},
		"should_reset_timer_and_then_send_neighbor_message": {
			finalizeBlock:      true,
			finalizeBlockAfter: 1 * time.Second,
			notifyInterval:     interval,
			expectWithin:       3 * time.Second,
		},
	}

	for tname, tt := range tests {
		tt := tt
		t.Run(tname, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			mockedNet := NewMockNetwork(ctrl)
			mockedCh := make(chan *types.FinalisationInfo)

			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			s := Service{
				ctx: ctx,
				state: &State{
					round: 1,
					setID: 0,
				},
				finalisedCh: mockedCh,
				head:        &types.Header{Number: 0},
				network:     mockedNet,
			}

			expectedNeighborMessage := &NeighbourMessage{
				Version: 1,
				Round:   s.state.round,
				SetID:   s.state.setID,
				Number:  uint32(s.head.Number),
			}
			cm, err := expectedNeighborMessage.ToConsensusMessage()
			require.NoError(t, err)

			timecheck := new(time.Time)

			wg := new(sync.WaitGroup)
			wg.Add(1)

			ensureGossipMessageCalledRightTime := func(_ network.NotificationsMessage) {
				defer wg.Done()
				const roundOverSec = 1 * time.Second

				calledWithin := time.Now().Sub(*timecheck)
				calledWithin = calledWithin.Round(roundOverSec) // avoid decimal points
				assert.Equal(t, tt.expectWithin, calledWithin)
			}

			mockedNet.EXPECT().GossipMessage(cm).Times(1).DoAndReturn(ensureGossipMessageCalledRightTime)

			*timecheck = time.Now()
			go s.notifyNeighbor(tt.notifyInterval)

			if tt.finalizeBlock {
				<-time.After(tt.finalizeBlockAfter)
				mockedCh <- &types.FinalisationInfo{}
			}

			wg.Wait()
		})
	}
}

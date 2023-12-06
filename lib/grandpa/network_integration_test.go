//go:build integration

// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package grandpa

import (
	"testing"

	"github.com/ChainSafe/gossamer/lib/crypto/ed25519"
	"github.com/ChainSafe/gossamer/lib/keystore"
	"go.uber.org/mock/gomock"

	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/stretchr/testify/require"
)

func TestGrandpaHandshake_Encode(t *testing.T) {
	hs := &GrandpaHandshake{
		Role: 4,
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
	kr, err := keystore.NewEd25519Keyring()
	require.NoError(t, err)
	aliceKeyPair := kr.Alice().(*ed25519.Keypair)

	gs, st := newTestService(t, aliceKeyPair)

	just := []SignedVote{
		{
			Vote:        *testVote,
			Signature:   testSignature,
			AuthorityID: gs.publicKeyBytes(),
		},
	}
	err = st.Grandpa.SetPrecommits(77, gs.state.setID, just)
	require.NoError(t, err)

	fm, err := gs.newCommitMessage(gs.head, 77, 0)
	require.NoError(t, err)

	cm, err := fm.ToConsensusMessage()
	require.NoError(t, err)
	gs.state.voters = gs.state.voters[:1]

	ctrl := gomock.NewController(t)
	telemetryMock := NewMockTelemetry(ctrl)

	h := NewMessageHandler(gs, st.Block, telemetryMock)
	gs.messageHandler = h

	propagate, err := gs.handleNetworkMessage(peer.ID(""), cm)
	require.NoError(t, err)
	require.True(t, propagate)

	neighbourMsg := &NeighbourPacketV1{}
	cm, err = neighbourMsg.ToConsensusMessage()
	require.NoError(t, err)

	propagate, err = gs.handleNetworkMessage(peer.ID(""), cm)
	require.NoError(t, err)
	require.False(t, propagate)
}

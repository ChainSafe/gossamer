// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package grandpa

import (
	"testing"

	"github.com/golang/mock/gomock"

	"github.com/libp2p/go-libp2p-core/peer"
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
	gs, st := newTestService(t, aliceKeyPair)

	just := []SignedVote{
		{
			Vote:        *testVote,
			Signature:   testSignature,
			AuthorityID: gs.publicKeyBytes(),
		},
	}
	err := st.Grandpa.SetPrecommits(77, gs.state.setID, just)
	require.NoError(t, err)

	fm, err := gs.newCommitMessage(gs.head, 77, 0)
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

	neighbourMsg := &NeighbourPacketV1{}
	cm, err = neighbourMsg.ToConsensusMessage()
	require.NoError(t, err)

	propagate, err = gs.handleNetworkMessage(peer.ID(""), cm)
	require.NoError(t, err)
	require.False(t, propagate)
}

// func TestSendNeighbourMessage(t *testing.T) {
// 	gs, st := newTestService(t, aliceKeyPair)
// 	go gs.sendNeighbourMessage(time.Second)

// 	digest := types.NewDigest()
// 	prd, err := types.NewBabeSecondaryPlainPreDigest(0, 1).ToPreRuntimeDigest()
// 	require.NoError(t, err)
// 	err = digest.Add(*prd)
// 	require.NoError(t, err)
// 	block := &types.Block{
// 		Header: types.Header{
// 			ParentHash: st.Block.GenesisHash(),
// 			Number:     1,
// 			Digest:     digest,
// 		},
// 		Body: types.Body{},
// 	}

// 	err = st.Block.AddBlock(block)
// 	require.NoError(t, err)

// 	hash := block.Header.Hash()
// 	round := uint64(7)
// 	setID := uint64(33)

// 	// waits 1.5 seconds and then finalize the block
// 	// we will first send a neighbour message with the initial values
// 	// and send another neighbour message with the finalized block values
// 	time.Sleep(1500 * time.Millisecond)
// 	err = st.Block.SetFinalisedHash(hash, round, setID)
// 	require.NoError(t, err)

// 	select {
// 	case <-time.After(time.Second):
// 		t.Fatal("did not send message")
// 	case msg := <-gs.network.(*testNetwork).out:
// 		expected := &NeighbourPacketV1{
// 			SetID:  0,
// 			Round:  0,
// 			Number: 0,
// 		}

// 		nm, ok := msg.(*NeighbourPacketV1)
// 		require.True(t, ok)
// 		require.Equal(t, expected, nm)
// 	}

// 	select {
// 	case <-time.After(time.Second):
// 		t.Fatal("did not send message")
// 	case msg := <-gs.network.(*testNetwork).out:
// 		expected := &NeighbourPacketV1{
// 			SetID:  setID,
// 			Round:  round,
// 			Number: 1,
// 		}

// 		nm, ok := msg.(*NeighbourPacketV1)
// 		require.True(t, ok)
// 		require.Equal(t, expected, nm)
// 	}
// }

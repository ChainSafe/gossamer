// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package grandpa

import (
	"math/big"
	"testing"
	"time"

	"github.com/ChainSafe/gossamer/dot/types"

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

	h := NewMessageHandler(gs, st.Block)
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

func TestSendNeighbourMessage(t *testing.T) {
	gs, st := newTestService(t)
	neighbourMessageInterval = time.Second
	defer func() {
		neighbourMessageInterval = time.Minute * 5
	}()
	go gs.sendNeighbourMessage()

	digest := types.NewDigest()
	prd, err := types.NewBabeSecondaryPlainPreDigest(0, 1).ToPreRuntimeDigest()
	require.NoError(t, err)
	err = digest.Add(*prd)
	require.NoError(t, err)
	block := &types.Block{
		Header: types.Header{
			ParentHash: st.Block.GenesisHash(),
			Number:     big.NewInt(1),
			Digest:     digest,
		},
		Body: types.Body{},
	}

	err = st.Block.AddBlock(block)
	require.NoError(t, err)

	hash := block.Header.Hash()
	round := uint64(7)
	setID := uint64(33)
	err = st.Block.SetFinalisedHash(hash, round, setID)
	require.NoError(t, err)

	expected := &NeighbourMessage{
		Version: 1,
		SetID:   setID,
		Round:   round,
		Number:  1,
	}

	select {
	case <-time.After(time.Second):
		t.Fatal("did not send message")
	case msg := <-gs.network.(*testNetwork).out:
		nm, ok := msg.(*NeighbourMessage)
		require.True(t, ok)
		require.Equal(t, expected, nm)
	}

	require.Equal(t, expected, gs.neighbourMessage)

	select {
	case <-time.After(time.Second * 2):
		t.Fatal("did not send message")
	case msg := <-gs.network.(*testNetwork).out:
		nm, ok := msg.(*NeighbourMessage)
		require.True(t, ok)
		require.Equal(t, expected, nm)
	}
}

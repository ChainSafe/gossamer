// Copyright 2020 ChainSafe Systems (ON) Corp.
// This file is part of gossamer.
//
// The gossamer library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The gossamer library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the gossamer library. If not, see <http://www.gnu.org/licenses/>.

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

	gs.justification[77] = []*SignedPrecommit{
		{
			Vote:        testVote,
			Signature:   testSignature,
			AuthorityID: gs.publicKeyBytes(),
		},
	}

	fm := gs.newCommitMessage(gs.head, 77)
	cm, err := fm.ToConsensusMessage()
	require.NoError(t, err)
	gs.state.voters = gs.state.voters[:1]

	h := NewMessageHandler(gs, st.Block)
	gs.messageHandler = h

	err = gs.handleNetworkMessage(peer.ID(""), cm)
	require.NoError(t, err)

	select {
	case <-gs.network.(*testNetwork).out:
	case <-time.After(testTimeout):
		t.Fatal("expected to send message")
	}
}

func TestSendNeighbourMessage(t *testing.T) {
	gs, st := newTestService(t)
	neighbourMessageInterval = time.Second
	defer func() {
		neighbourMessageInterval = time.Minute * 5
	}()
	go gs.sendNeighbourMessage()

	block := &types.Block{
		Header: &types.Header{
			ParentHash: testGenesisHeader.Hash(),
			Number:     big.NewInt(1),
		},
		Body: &types.Body{},
	}

	err := st.Block.AddBlock(block)
	require.NoError(t, err)

	hash := block.Header.Hash()
	round := uint64(7)
	setID := uint64(33)
	err = st.Block.SetFinalizedHash(hash, round, setID)
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

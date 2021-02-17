// Copyright 2019 ChainSafe Systems (ON) Corp.
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

package network

import (
	"math/big"
	"testing"

	"github.com/ChainSafe/gossamer/dot/types"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/lib/utils"

	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/stretchr/testify/require"
)

func TestBlockAnnounce_Encode(t *testing.T) {
	testBlockAnnounce := &BlockAnnounceMessage{
		ParentHash:     common.Hash{1},
		Number:         big.NewInt(77),
		StateRoot:      common.Hash{2},
		ExtrinsicsRoot: common.Hash{3},
		Digest:         types.Digest{},
	}

	enc, err := testBlockAnnounce.Encode()
	require.NoError(t, err)

	res := &BlockAnnounceMessage{
		Number: big.NewInt(0),
		Digest: types.Digest{},
	}
	err = res.Decode(enc)
	require.NoError(t, err)
	require.Equal(t, testBlockAnnounce, res)
}

func TestDecodeBlockAnnounceHandshake(t *testing.T) {
	testHandshake := &BlockAnnounceHandshake{
		Roles:           4,
		BestBlockNumber: 77,
		BestBlockHash:   common.Hash{1},
		GenesisHash:     common.Hash{2},
	}

	enc, err := testHandshake.Encode()
	require.NoError(t, err)

	msg, err := decodeBlockAnnounceHandshake(enc)
	require.NoError(t, err)
	require.Equal(t, testHandshake, msg)
}

func TestDecodeBlockAnnounceMessage(t *testing.T) {
	testBlockAnnounce := &BlockAnnounceMessage{
		ParentHash:     common.Hash{1},
		Number:         big.NewInt(77),
		StateRoot:      common.Hash{2},
		ExtrinsicsRoot: common.Hash{3},
		Digest:         types.Digest{},
	}

	enc, err := testBlockAnnounce.Encode()
	require.NoError(t, err)

	msg, err := decodeBlockAnnounceMessage(enc)
	require.NoError(t, err)
	require.Equal(t, testBlockAnnounce, msg)
}

func TestHandleBlockAnnounceMessage(t *testing.T) {
	basePath := utils.NewTestBasePath(t, "nodeA")

	// removes all data directories created within test directory
	defer utils.RemoveTestDir(t)

	config := &Config{
		BasePath:    basePath,
		Port:        7001,
		RandSeed:    1,
		NoBootstrap: true,
		NoMDNS:      true,
	}

	s := createTestService(t, config)

	peerID := peer.ID("noot")
	msg := &BlockAnnounceMessage{
		Number: big.NewInt(10),
	}

	s.handleBlockAnnounceMessage(peerID, msg)
	require.NotNil(t, s.syncQueue.isSyncing(peerID))
}

func TestValidateBlockAnnounceHandshake(t *testing.T) {
	configA := &Config{
		BasePath:    utils.NewTestBasePath(t, "nodeA"),
		Port:        7001,
		RandSeed:    1,
		NoBootstrap: true,
		NoMDNS:      true,
	}

	nodeA := createTestService(t, configA)
	nodeA.noGossip = true
	nodeA.notificationsProtocols[BlockAnnounceMsgType] = &notificationsProtocol{
		handshakeData: make(map[peer.ID]*handshakeData),
	}

	testPeerID := peer.ID("noot")
	err := nodeA.validateBlockAnnounceHandshake(testPeerID, &BlockAnnounceHandshake{
		BestBlockNumber: 100,
		GenesisHash:     nodeA.blockState.GenesisHash(),
	})
	require.NoError(t, err)
}

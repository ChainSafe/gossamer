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
	"bytes"
	"math/big"
	"testing"
	"time"

	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/lib/utils"

	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/stretchr/testify/require"
)

func TestDecodeBlockAnnounceHandshake(t *testing.T) {
	testHandshake := &BlockAnnounceHandshake{
		Roles:           4,
		BestBlockNumber: 77,
		BestBlockHash:   common.Hash{1},
		GenesisHash:     common.Hash{2},
	}

	enc, err := testHandshake.Encode()
	require.NoError(t, err)

	buf := &bytes.Buffer{}
	buf.Write(enc)

	msg, err := decodeBlockAnnounceHandshake(buf)
	require.NoError(t, err)
	require.Equal(t, testHandshake, msg)
}

func TestDecodeBlockAnnounceMessage(t *testing.T) {
	testBlockAnnounce := &BlockAnnounceMessage{
		ParentHash:     common.Hash{1},
		Number:         big.NewInt(77),
		StateRoot:      common.Hash{2},
		ExtrinsicsRoot: common.Hash{3},
		Digest:         [][]byte{},
	}

	enc, err := testBlockAnnounce.Encode()
	require.NoError(t, err)

	buf := &bytes.Buffer{}
	buf.Write(enc)

	msg, err := decodeBlockAnnounceMessage(buf)
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
	require.NotNil(t, s.syncing[peerID])
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

	configB := &Config{
		BasePath:    utils.NewTestBasePath(t, "nodeB"),
		Port:        7002,
		RandSeed:    2,
		NoBootstrap: true,
		NoMDNS:      true,
	}

	nodeB := createTestService(t, configB)
	nodeB.noGossip = true

	handler := newTestStreamHandler(testBlockRequestMessageDecoder)
	nodeB.host.registerStreamHandler(syncID, handler.handleStream)

	addrInfosB, err := nodeB.host.addrInfos()
	require.NoError(t, err)

	err = nodeA.host.connect(*addrInfosB[0])
	if failedToDial(err) {
		time.Sleep(TestBackoffTimeout)
		err = nodeA.host.connect(*addrInfosB[0])
	}
	require.NoError(t, err)

	err = nodeA.validateBlockAnnounceHandshake(nodeB.host.id(), &BlockAnnounceHandshake{
		BestBlockNumber: 100,
		GenesisHash:     nodeB.blockState.GenesisHash(),
	})
	require.NoError(t, err)

	time.Sleep(TestMessageTimeout)
	require.NotNil(t, handler.messages[nodeA.host.id()], "node B timeout waiting for message from node A")
}

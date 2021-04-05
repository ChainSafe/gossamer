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
	"sync"
	"testing"
	"time"

	"github.com/ChainSafe/gossamer/dot/types"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/lib/utils"

	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/stretchr/testify/require"
)

func TestCreateDecoder_BlockAnnounce(t *testing.T) {
	basePath := utils.NewTestBasePath(t, "nodeA")

	config := &Config{
		BasePath:    basePath,
		Port:        7001,
		RandSeed:    1,
		NoBootstrap: true,
		NoMDNS:      true,
	}

	s := createTestService(t, config)

	// create info and decoder
	info := &notificationsProtocol{
		protocolID:    s.host.protocolID + blockAnnounceID,
		getHandshake:  s.getBlockAnnounceHandshake,
		handshakeData: new(sync.Map),
	}
	decoder := createDecoder(info, decodeBlockAnnounceHandshake, decodeBlockAnnounceMessage)

	// haven't received handshake from peer
	testPeerID := peer.ID("QmaCpDMGvV2BGHeYERUEnRQAwe3N8SzbUtfsmvsqQLuvuJ")
	info.handshakeData.Store(testPeerID, &handshakeData{
		received: false,
	})

	testHandshake := &BlockAnnounceHandshake{
		Roles:           4,
		BestBlockNumber: 77,
		BestBlockHash:   common.Hash{1},
		GenesisHash:     common.Hash{2},
	}

	enc, err := testHandshake.Encode()
	require.NoError(t, err)

	msg, err := decoder(enc, testPeerID)
	require.NoError(t, err)
	require.Equal(t, testHandshake, msg)

	testBlockAnnounce := &BlockAnnounceMessage{
		ParentHash:     common.Hash{1},
		Number:         big.NewInt(77),
		StateRoot:      common.Hash{2},
		ExtrinsicsRoot: common.Hash{3},
		Digest:         types.Digest{},
	}

	enc, err = testBlockAnnounce.Encode()
	require.NoError(t, err)

	// set handshake data to received
	hsData, _ := info.getHandshakeData(testPeerID)
	hsData.received = true
	msg, err = decoder(enc, testPeerID)
	require.NoError(t, err)
	require.Equal(t, testBlockAnnounce, msg)
}

func TestCreateNotificationsMessageHandler_BlockAnnounce(t *testing.T) {
	basePath := utils.NewTestBasePath(t, "nodeA")

	config := &Config{
		BasePath:    basePath,
		Port:        7001,
		RandSeed:    1,
		NoBootstrap: true,
		NoMDNS:      true,
	}

	s := createTestService(t, config)

	configB := &Config{
		BasePath:    utils.NewTestBasePath(t, "nodeB"),
		Port:        7002,
		RandSeed:    2,
		NoBootstrap: true,
		NoMDNS:      true,
	}

	b := createTestService(t, configB)

	// don't set handshake data ie. this stream has just been opened
	testPeerID := b.host.id()

	// connect nodes
	addrInfosB, err := b.host.addrInfos()
	require.NoError(t, err)

	err = s.host.connect(*addrInfosB[0])
	if failedToDial(err) {
		time.Sleep(TestBackoffTimeout)
		err = s.host.connect(*addrInfosB[0])
	}
	require.NoError(t, err)

	stream, err := s.host.h.NewStream(s.ctx, b.host.id(), s.host.protocolID+blockAnnounceID)
	require.NoError(t, err)

	// create info and handler
	info := &notificationsProtocol{
		protocolID:    s.host.protocolID + blockAnnounceID,
		getHandshake:  s.getBlockAnnounceHandshake,
		handshakeData: new(sync.Map),
	}
	handler := s.createNotificationsMessageHandler(info, s.validateBlockAnnounceHandshake, s.handleBlockAnnounceMessage)

	// set handshake data to received
	info.handshakeData.Store(testPeerID, &handshakeData{
		received:  true,
		validated: true,
	})
	msg := &BlockAnnounceMessage{
		Number: big.NewInt(10),
	}

	err = handler(stream, msg)
	require.NoError(t, err)
}

func TestCreateNotificationsMessageHandler_BlockAnnounceHandshake(t *testing.T) {
	config := &Config{
		BasePath:    utils.NewTestBasePath(t, "nodeA"),
		Port:        7001,
		RandSeed:    1,
		NoBootstrap: true,
		NoMDNS:      true,
	}

	s := createTestService(t, config)

	// create info and handler
	info := &notificationsProtocol{
		protocolID:    s.host.protocolID + blockAnnounceID,
		getHandshake:  s.getBlockAnnounceHandshake,
		handshakeData: new(sync.Map),
	}
	handler := s.createNotificationsMessageHandler(info, s.validateBlockAnnounceHandshake, s.handleBlockAnnounceMessage)

	configB := &Config{
		BasePath:    utils.NewTestBasePath(t, "nodeB"),
		Port:        7002,
		RandSeed:    2,
		NoBootstrap: true,
		NoMDNS:      true,
	}

	b := createTestService(t, configB)

	// don't set handshake data ie. this stream has just been opened
	testPeerID := b.host.id()

	// connect nodes
	addrInfosB, err := b.host.addrInfos()
	require.NoError(t, err)

	err = s.host.connect(*addrInfosB[0])
	if failedToDial(err) {
		time.Sleep(TestBackoffTimeout)
		err = s.host.connect(*addrInfosB[0])
	}
	require.NoError(t, err)

	stream, err := s.host.h.NewStream(s.ctx, b.host.id(), s.host.protocolID+blockAnnounceID)
	require.NoError(t, err)

	// try invalid handshake
	testHandshake := &BlockAnnounceHandshake{
		Roles:           4,
		BestBlockNumber: 77,
		BestBlockHash:   common.Hash{1},
		GenesisHash:     common.Hash{2},
	}

	err = handler(stream, testHandshake)
	require.Equal(t, errCannotValidateHandshake, err)
	data, has := info.getHandshakeData(testPeerID)
	require.True(t, has)
	require.True(t, data.received)
	require.False(t, data.validated)

	// try valid handshake
	testHandshake = &BlockAnnounceHandshake{
		Roles:           4,
		BestBlockNumber: 77,
		BestBlockHash:   common.Hash{1},
		GenesisHash:     s.blockState.GenesisHash(),
	}

	err = handler(stream, testHandshake)
	require.NoError(t, err)
	data, has = info.getHandshakeData(testPeerID)
	require.True(t, has)
	require.True(t, data.received)
	require.True(t, data.validated)
}

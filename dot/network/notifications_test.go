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
	"context"
	"fmt"
	"math/big"
	"sync"
	"testing"
	"time"

	"github.com/ChainSafe/gossamer/dot/types"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/lib/utils"
	ma "github.com/multiformats/go-multiaddr"

	"github.com/libp2p/go-libp2p"
	libp2pnetwork "github.com/libp2p/go-libp2p-core/network"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/stretchr/testify/require"
)

func TestHandshake_SizeOf(t *testing.T) {
	require.Equal(t, uint32(maxHandshakeSize), uint32(72))
}

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
		protocolID:            s.host.protocolID + blockAnnounceID,
		getHandshake:          s.getBlockAnnounceHandshake,
		handshakeValidator:    s.validateBlockAnnounceHandshake,
		inboundHandshakeData:  new(sync.Map),
		outboundHandshakeData: new(sync.Map),
	}
	decoder := createDecoder(info, decodeBlockAnnounceHandshake, decodeBlockAnnounceMessage)

	// haven't received handshake from peer
	testPeerID := peer.ID("QmaCpDMGvV2BGHeYERUEnRQAwe3N8SzbUtfsmvsqQLuvuJ")
	info.inboundHandshakeData.Store(testPeerID, handshakeData{
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

	msg, err := decoder(enc, testPeerID, true)
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
	hsData, _ := info.getHandshakeData(testPeerID, true)
	hsData.received = true
	info.inboundHandshakeData.Store(testPeerID, hsData)
	msg, err = decoder(enc, testPeerID, true)
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
		protocolID:            s.host.protocolID + blockAnnounceID,
		getHandshake:          s.getBlockAnnounceHandshake,
		handshakeValidator:    s.validateBlockAnnounceHandshake,
		inboundHandshakeData:  new(sync.Map),
		outboundHandshakeData: new(sync.Map),
	}
	handler := s.createNotificationsMessageHandler(info, s.handleBlockAnnounceMessage)

	// set handshake data to received
	info.inboundHandshakeData.Store(testPeerID, handshakeData{
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
		protocolID:            s.host.protocolID + blockAnnounceID,
		getHandshake:          s.getBlockAnnounceHandshake,
		handshakeValidator:    s.validateBlockAnnounceHandshake,
		inboundHandshakeData:  new(sync.Map),
		outboundHandshakeData: new(sync.Map),
	}
	handler := s.createNotificationsMessageHandler(info, s.handleBlockAnnounceMessage)

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
	data, has := info.getHandshakeData(testPeerID, true)
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

	info.inboundHandshakeData.Delete(testPeerID)

	err = handler(stream, testHandshake)
	require.NoError(t, err)
	data, has = info.getHandshakeData(testPeerID, true)
	require.True(t, has)
	require.True(t, data.received)
	require.True(t, data.validated)
}

func Test_HandshakeTimeout(t *testing.T) {
	// create service A
	config := &Config{
		BasePath:    utils.NewTestBasePath(t, "nodeA"),
		Port:        7001,
		RandSeed:    1,
		NoBootstrap: true,
		NoMDNS:      true,
	}
	ha := createTestService(t, config)

	// create info and handler
	info := &notificationsProtocol{
		protocolID:            ha.host.protocolID + blockAnnounceID,
		getHandshake:          ha.getBlockAnnounceHandshake,
		handshakeValidator:    ha.validateBlockAnnounceHandshake,
		inboundHandshakeData:  new(sync.Map),
		outboundHandshakeData: new(sync.Map),
	}

	// creating host b with will never respond to a handshake
	addrB, err := ma.NewMultiaddr(fmt.Sprintf("/ip4/0.0.0.0/tcp/%d", 7002))
	require.NoError(t, err)

	hb, err := libp2p.New(
		context.Background(), libp2p.ListenAddrs(addrB),
	)
	require.NoError(t, err)

	testHandshakeMsg := &BlockAnnounceHandshake{
		Roles:           4,
		BestBlockNumber: 77,
		BestBlockHash:   common.Hash{1},
		GenesisHash:     common.Hash{2},
	}

	hb.SetStreamHandler(info.protocolID, func(stream libp2pnetwork.Stream) {
		fmt.Println("never respond a handshake message")
	})

	addrBInfo := peer.AddrInfo{
		ID:    hb.ID(),
		Addrs: hb.Addrs(),
	}

	err = ha.host.connect(addrBInfo)
	if failedToDial(err) {
		time.Sleep(TestBackoffTimeout)
		err = ha.host.connect(addrBInfo)
	}
	require.NoError(t, err)

	go ha.sendData(hb.ID(), testHandshakeMsg, info, nil)

	time.Sleep(handshakeTimeout / 2)
	// peer should be stored in handshake data until timeout
	_, ok := info.outboundHandshakeData.Load(hb.ID())
	require.True(t, ok)

	// a stream should be open until timeout
	connAToB := ha.host.h.Network().ConnsToPeer(hb.ID())
	require.Len(t, connAToB, 1)
	require.Len(t, connAToB[0].GetStreams(), 1)

	// after the timeout
	time.Sleep(handshakeTimeout)

	// handshake data should be removed
	_, ok = info.outboundHandshakeData.Load(hb.ID())
	require.False(t, ok)

	// stream should be closed
	connAToB = ha.host.h.Network().ConnsToPeer(hb.ID())
	require.Len(t, connAToB, 1)
	require.Len(t, connAToB[0].GetStreams(), 0)
}

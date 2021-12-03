// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package network

import (
	"errors"
	"math/big"
	"reflect"
	"sync"
	"testing"
	"time"
	"unsafe"

	libp2pnetwork "github.com/libp2p/go-libp2p-core/network"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/stretchr/testify/require"

	"github.com/ChainSafe/gossamer/dot/types"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/lib/utils"
)

func TestCreateDecoder_BlockAnnounce(t *testing.T) {
	basePath := utils.NewTestBasePath(t, "nodeA")

	config := &Config{
		BasePath:    basePath,
		Port:        7001,
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
	info.inboundHandshakeData.Store(testPeerID, &handshakeData{
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
		Digest:         types.NewDigest(),
	}

	enc, err = testBlockAnnounce.Encode()
	require.NoError(t, err)

	// set handshake data to received
	hsData, _ := info.getInboundHandshakeData(testPeerID)
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
		NoBootstrap: true,
		NoMDNS:      true,
	}

	s := createTestService(t, config)

	configB := &Config{
		BasePath:    utils.NewTestBasePath(t, "nodeB"),
		Port:        7002,
		NoBootstrap: true,
		NoMDNS:      true,
	}

	b := createTestService(t, configB)

	// don't set handshake data ie. this stream has just been opened
	testPeerID := b.host.id()

	// connect nodes
	addrInfoB := b.host.addrInfo()
	err := s.host.connect(addrInfoB)
	if failedToDial(err) {
		time.Sleep(TestBackoffTimeout)
		err = s.host.connect(addrInfoB)
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
	handler := s.createNotificationsMessageHandler(info, s.handleBlockAnnounceMessage, nil)

	// set handshake data to received
	info.inboundHandshakeData.Store(testPeerID, &handshakeData{
		received:  true,
		validated: true,
	})

	msg := &BlockAnnounceMessage{
		Number: big.NewInt(10),
		Digest: types.NewDigest(),
	}

	err = handler(stream, msg)
	require.NoError(t, err)
}

func TestCreateNotificationsMessageHandler_BlockAnnounceHandshake(t *testing.T) {
	config := &Config{
		BasePath:    utils.NewTestBasePath(t, "nodeA"),
		Port:        7001,
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
	handler := s.createNotificationsMessageHandler(info, s.handleBlockAnnounceMessage, nil)

	configB := &Config{
		BasePath:    utils.NewTestBasePath(t, "nodeB"),
		Port:        7002,
		NoBootstrap: true,
		NoMDNS:      true,
	}

	b := createTestService(t, configB)

	// don't set handshake data ie. this stream has just been opened
	testPeerID := b.host.id()

	// connect nodes
	addrInfoB := b.host.addrInfo()
	err := s.host.connect(addrInfoB)
	if failedToDial(err) {
		time.Sleep(TestBackoffTimeout)
		err = s.host.connect(addrInfoB)
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
	data, has := info.getInboundHandshakeData(testPeerID)
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
	data, has = info.getInboundHandshakeData(testPeerID)
	require.True(t, has)
	require.True(t, data.received)
	require.True(t, data.validated)
}

func Test_HandshakeTimeout(t *testing.T) {
	basePathA := utils.NewTestBasePath(t, "nodeA")
	configA := &Config{
		BasePath:    basePathA,
		Port:        7001,
		NoBootstrap: true,
		NoMDNS:      true,
	}

	nodeA := createTestService(t, configA)
	nodeA.noGossip = true

	basePathB := utils.NewTestBasePath(t, "nodeB")
	configB := &Config{
		BasePath:    basePathB,
		Port:        7002,
		RandSeed:    2,
		NoBootstrap: true,
		NoMDNS:      true,
	}

	nodeB := createTestService(t, configB)
	nodeB.noGossip = true

	// create info and handler
	testHandshakeDecoder := func([]byte) (Handshake, error) {
		return nil, errors.New("unimplemented")
	}
	info := newNotificationsProtocol(nodeA.host.protocolID+blockAnnounceID,
		nodeA.getBlockAnnounceHandshake, testHandshakeDecoder, nodeA.validateBlockAnnounceHandshake)

	nodeB.host.h.SetStreamHandler(info.protocolID, func(stream libp2pnetwork.Stream) {
		// should not respond to a handshake message
	})

	addrInfosB := nodeB.host.addrInfo()

	err := nodeA.host.connect(addrInfosB)
	// retry connect if "failed to dial" error
	if failedToDial(err) {
		time.Sleep(TestBackoffTimeout)
		err = nodeA.host.connect(addrInfosB)
	}
	require.NoError(t, err)

	testHandshakeMsg := &BlockAnnounceHandshake{
		Roles:           4,
		BestBlockNumber: 77,
		BestBlockHash:   common.Hash{1},
		GenesisHash:     common.Hash{2},
	}
	nodeA.GossipMessage(testHandshakeMsg)

	info.outboundHandshakeMutexes.Store(nodeB.host.id(), new(sync.Mutex))
	go nodeA.sendData(nodeB.host.id(), testHandshakeMsg, info, nil)

	time.Sleep(time.Second)

	// handshake data shouldn't exist, as nodeB hasn't responded yet
	_, ok := info.getOutboundHandshakeData(nodeB.host.id())
	require.False(t, ok)

	// a stream should be open until timeout
	connAToB := nodeA.host.h.Network().ConnsToPeer(nodeB.host.id())
	require.Len(t, connAToB, 1)
	require.Len(t, connAToB[0].GetStreams(), 1)

	// after the timeout
	time.Sleep(handshakeTimeout)

	// handshake data shouldn't exist still
	_, ok = info.getOutboundHandshakeData(nodeB.host.id())
	require.False(t, ok)

	// stream should be closed
	connAToB = nodeA.host.h.Network().ConnsToPeer(nodeB.host.id())
	require.Len(t, connAToB, 1)
	require.Len(t, connAToB[0].GetStreams(), 0)
}

func TestCreateNotificationsMessageHandler_HandleTransaction(t *testing.T) {
	const batchSize = 5
	basePath := utils.NewTestBasePath(t, "nodeA")
	config := &Config{
		BasePath:    basePath,
		Port:        7001,
		NoBootstrap: true,
		NoMDNS:      true,
		batchSize:   batchSize,
	}

	srvc1 := createTestService(t, config)

	configB := &Config{
		BasePath:    utils.NewTestBasePath(t, "nodeB"),
		Port:        7002,
		NoBootstrap: true,
		NoMDNS:      true,
	}

	srvc2 := createTestService(t, configB)

	txnBatch := make(chan *BatchMessage, batchSize)
	txnBatchHandler := srvc1.createBatchMessageHandler(txnBatch)

	// connect nodes
	addrInfoB := srvc2.host.addrInfo()
	err := srvc1.host.connect(addrInfoB)
	if failedToDial(err) {
		time.Sleep(TestBackoffTimeout)
		err = srvc1.host.connect(addrInfoB)
		require.NoError(t, err)
	}
	require.NoError(t, err)

	txnProtocolID := srvc1.host.protocolID + transactionsID
	stream, err := srvc1.host.h.NewStream(srvc1.ctx, srvc2.host.id(), txnProtocolID)
	require.NoError(t, err)

	// create info and handler
	info := &notificationsProtocol{
		protocolID:            txnProtocolID,
		getHandshake:          srvc1.getTransactionHandshake,
		handshakeValidator:    validateTransactionHandshake,
		inboundHandshakeData:  new(sync.Map),
		outboundHandshakeData: new(sync.Map),
	}
	handler := srvc1.createNotificationsMessageHandler(info, srvc1.handleTransactionMessage, txnBatchHandler)

	// set handshake data to received
	info.inboundHandshakeData.Store(srvc2.host.id(), handshakeData{
		received:  true,
		validated: true,
	})

	msg := &TransactionMessage{
		Extrinsics: []types.Extrinsic{{1, 1}, {2, 2}},
	}
	err = handler(stream, msg)
	require.NoError(t, err)
	require.Len(t, txnBatch, 1)

	msg = &TransactionMessage{
		Extrinsics: []types.Extrinsic{{1, 1}, {2, 2}},
	}
	err = handler(stream, msg)
	require.NoError(t, err)
	require.Len(t, txnBatch, 2)

	msg = &TransactionMessage{
		Extrinsics: []types.Extrinsic{{1, 1}, {2, 2}},
	}
	err = handler(stream, msg)
	require.NoError(t, err)
	require.Len(t, txnBatch, 3)

	msg = &TransactionMessage{
		Extrinsics: []types.Extrinsic{{1, 1}, {2, 2}},
	}
	err = handler(stream, msg)
	require.NoError(t, err)
	require.Len(t, txnBatch, 4)

	msg = &TransactionMessage{
		Extrinsics: []types.Extrinsic{{1, 1}, {2, 2}},
	}
	err = handler(stream, msg)
	require.NoError(t, err)
	require.Len(t, txnBatch, 5)

	// reached batch size limit, below transaction will not be included in batch.
	msg = &TransactionMessage{
		Extrinsics: []types.Extrinsic{{1, 1}, {2, 2}},
	}
	err = handler(stream, msg)
	require.NoError(t, err)
	require.Len(t, txnBatch, 5)

	msg = &TransactionMessage{
		Extrinsics: []types.Extrinsic{{1, 1}, {2, 2}},
	}
	// wait for transaction batch channel to process.
	time.Sleep(1300 * time.Millisecond)
	err = handler(stream, msg)
	require.NoError(t, err)
	require.Len(t, txnBatch, 1)
}

func TestBlockAnnounceHandshakeSize(t *testing.T) {
	require.Equal(t, unsafe.Sizeof(BlockAnnounceHandshake{}), reflect.TypeOf(BlockAnnounceHandshake{}).Size())
}

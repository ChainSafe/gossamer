// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package network

import (
	"errors"
	"fmt"
	"reflect"
	"testing"
	"time"
	"unsafe"

	libp2pnetwork "github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/stretchr/testify/require"

	"github.com/ChainSafe/gossamer/dot/types"
	"github.com/ChainSafe/gossamer/lib/common"
)

func TestCreateDecoder_BlockAnnounce(t *testing.T) {
	t.Parallel()

	config := &Config{
		BasePath:    t.TempDir(),
		Port:        availablePort(t),
		NoBootstrap: true,
		NoMDNS:      true,
	}

	s := createTestService(t, config)

	// create info and decoder
	info := &notificationsProtocol{
		protocolID:         s.host.protocolID + blockAnnounceID,
		getHandshake:       s.getBlockAnnounceHandshake,
		handshakeValidator: s.validateBlockAnnounceHandshake,
		peersData:          newPeersData(),
	}
	decoder := createDecoder(info, decodeBlockAnnounceHandshake, decodeBlockAnnounceMessage)

	// haven't received handshake from peer
	testPeerID := peer.ID("QmaCpDMGvV2BGHeYERUEnRQAwe3N8SzbUtfsmvsqQLuvuJ")
	info.peersData.setInboundHandshakeData(testPeerID, &handshakeData{
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
		Number:         77,
		StateRoot:      common.Hash{2},
		ExtrinsicsRoot: common.Hash{3},
		Digest:         nil,
	}

	enc, err = testBlockAnnounce.Encode()
	require.NoError(t, err)

	// set handshake data to received
	hsData := info.peersData.getInboundHandshakeData(testPeerID)
	hsData.received = true
	info.peersData.setInboundHandshakeData(testPeerID, hsData)
	msg, err = decoder(enc, testPeerID, true)
	require.NoError(t, err)
	require.Equal(t, testBlockAnnounce, msg)
}

func TestCreateNotificationsMessageHandler_BlockAnnounce(t *testing.T) {
	t.Parallel()

	config := &Config{
		BasePath:    t.TempDir(),
		Port:        availablePort(t),
		NoBootstrap: true,
		NoMDNS:      true,
	}

	s := createTestService(t, config)

	configB := &Config{
		BasePath:    t.TempDir(),
		Port:        availablePort(t),
		NoBootstrap: true,
		NoMDNS:      true,
	}

	b := createTestService(t, configB)

	// don't set handshake data ie. this stream has just been opened
	testPeerID := b.host.id()

	// connect nodes
	addrInfoB := addrInfo(b.host)
	err := s.host.connect(addrInfoB)
	if failedToDial(err) {
		time.Sleep(TestBackoffTimeout)
		err = s.host.connect(addrInfoB)
	}
	require.NoError(t, err)

	stream, err := s.host.p2pHost.NewStream(s.ctx, b.host.id(), s.host.protocolID+blockAnnounceID)
	require.NoError(t, err)

	// create info and handler
	info := &notificationsProtocol{
		protocolID:         s.host.protocolID + blockAnnounceID,
		getHandshake:       s.getBlockAnnounceHandshake,
		handshakeValidator: s.validateBlockAnnounceHandshake,
		peersData:          newPeersData(),
	}
	handler := s.createNotificationsMessageHandler(info, s.handleBlockAnnounceMessage, nil)

	// set handshake data to received
	info.peersData.setInboundHandshakeData(testPeerID, &handshakeData{
		received:  true,
		validated: true,
	})

	msg := &BlockAnnounceMessage{
		Number: 10,
		Digest: types.NewDigest(),
	}

	err = handler(stream, msg)
	require.NoError(t, err)
}

func TestCreateNotificationsMessageHandler_BlockAnnounceHandshake(t *testing.T) {
	t.Parallel()

	config := &Config{
		BasePath:    t.TempDir(),
		Port:        availablePort(t),
		NoBootstrap: true,
		NoMDNS:      true,
	}

	s := createTestService(t, config)

	// create info and handler
	info := &notificationsProtocol{
		protocolID:         s.host.protocolID + blockAnnounceID,
		getHandshake:       s.getBlockAnnounceHandshake,
		handshakeValidator: s.validateBlockAnnounceHandshake,
		peersData:          newPeersData(),
	}
	handler := s.createNotificationsMessageHandler(info, s.handleBlockAnnounceMessage, nil)

	configB := &Config{
		BasePath:    t.TempDir(),
		Port:        availablePort(t),
		NoBootstrap: true,
		NoMDNS:      true,
	}

	b := createTestService(t, configB)

	// don't set handshake data ie. this stream has just been opened
	testPeerID := b.host.id()

	// connect nodes
	addrInfoB := addrInfo(b.host)
	err := s.host.connect(addrInfoB)
	if failedToDial(err) {
		time.Sleep(TestBackoffTimeout)
		err = s.host.connect(addrInfoB)
	}
	require.NoError(t, err)

	stream, err := s.host.p2pHost.NewStream(s.ctx, b.host.id(), s.host.protocolID+blockAnnounceID)
	require.NoError(t, err)

	// try invalid handshake
	testHandshake := &BlockAnnounceHandshake{
		Roles:           common.AuthorityRole,
		BestBlockNumber: 77,
		BestBlockHash:   common.Hash{1},
		// we are using a different genesis here, thus this
		// handshake would be validated to be incorrect.
		GenesisHash: common.Hash{2},
	}

	err = handler(stream, testHandshake)
	require.ErrorIs(t, err, errCannotValidateHandshake)

	expectedErrorMessage := fmt.Sprintf("handling handshake: %s from peer %s using protocol %s: genesis hash mismatch",
		errCannotValidateHandshake, testPeerID, info.protocolID)
	require.EqualError(t, err, expectedErrorMessage)

	data := info.peersData.getInboundHandshakeData(testPeerID)
	require.NotNil(t, data)
	require.True(t, data.received)
	require.False(t, data.validated)

	// try valid handshake
	testHandshake = &BlockAnnounceHandshake{
		Roles:           4,
		BestBlockNumber: 77,
		BestBlockHash:   common.Hash{1},
		GenesisHash:     s.blockState.GenesisHash(),
	}

	info.peersData.deleteInboundHandshakeData(testPeerID)

	err = handler(stream, testHandshake)
	require.NoError(t, err)
	data = info.peersData.getInboundHandshakeData(testPeerID)
	require.NotNil(t, data)
	require.True(t, data.received)
	require.True(t, data.validated)
}

func Test_HandshakeTimeout(t *testing.T) {
	t.Parallel()

	configA := &Config{
		BasePath:    t.TempDir(),
		Port:        availablePort(t),
		NoBootstrap: true,
		NoMDNS:      true,
	}

	nodeA := createTestService(t, configA)
	nodeA.noGossip = true

	configB := &Config{
		BasePath:    t.TempDir(),
		Port:        availablePort(t),
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
	info := newNotificationsProtocol(nodeA.host.protocolID+blockAnnounceID, nodeA.getBlockAnnounceHandshake,
		testHandshakeDecoder, nodeA.validateBlockAnnounceHandshake, maxBlockAnnounceNotificationSize)

	nodeB.host.p2pHost.SetStreamHandler(info.protocolID, func(stream libp2pnetwork.Stream) {
		// should not respond to a handshake message
	})

	addrInfosB := addrInfo(nodeB.host)

	err := nodeA.host.connect(addrInfosB)
	// retry connect if "failed to dial" error
	if failedToDial(err) {
		time.Sleep(TestBackoffTimeout)
		err = nodeA.host.connect(addrInfosB)
	}
	require.NoError(t, err)

	// clear handshake data from connection handler
	time.Sleep(time.Millisecond * 100)
	info.peersData.deleteOutboundHandshakeData(nodeB.host.id())
	connAToB := nodeA.host.p2pHost.Network().ConnsToPeer(nodeB.host.id())
	for _, stream := range connAToB[0].GetStreams() {
		err := stream.Close()
		require.NoError(t, err)
	}

	testHandshakeMsg := &BlockAnnounceHandshake{
		Roles:           4,
		BestBlockNumber: 77,
		BestBlockHash:   common.Hash{1},
		GenesisHash:     common.Hash{2},
	}

	info.peersData.setMutex(nodeB.host.id())
	go nodeA.sendData(nodeB.host.id(), testHandshakeMsg, info, nil)

	time.Sleep(time.Second)

	// handshake data shouldn't exist, as nodeB hasn't responded yet
	data := info.peersData.getOutboundHandshakeData(nodeB.host.id())
	require.Nil(t, data)

	// a stream should be open until timeout
	connAToB = nodeA.host.p2pHost.Network().ConnsToPeer(nodeB.host.id())
	require.Len(t, connAToB, 1)
	require.Len(t, connAToB[0].GetStreams(), 1)

	// after the timeout
	time.Sleep(handshakeTimeout)

	// handshake data shouldn't exist still
	data = info.peersData.getOutboundHandshakeData(nodeB.host.id())
	require.Nil(t, data)

	// stream should be closed
	connAToB = nodeA.host.p2pHost.Network().ConnsToPeer(nodeB.host.id())
	require.Len(t, connAToB, 1)
	require.Len(t, connAToB[0].GetStreams(), 0)
}

func TestCreateNotificationsMessageHandler_HandleTransaction(t *testing.T) {
	t.Parallel()

	const batchSize = 5
	config := &Config{
		BasePath:    t.TempDir(),
		Port:        availablePort(t),
		NoBootstrap: true,
		NoMDNS:      true,
		batchSize:   batchSize,
	}

	srvc1 := createTestService(t, config)

	configB := &Config{
		BasePath:    t.TempDir(),
		Port:        availablePort(t),
		NoBootstrap: true,
		NoMDNS:      true,
	}

	srvc2 := createTestService(t, configB)

	txnBatch := make(chan *batchMessage, batchSize)
	txnBatchHandler := srvc1.createBatchMessageHandler(txnBatch)

	// connect nodes
	addrInfoB := addrInfo(srvc2.host)
	err := srvc1.host.connect(addrInfoB)
	if failedToDial(err) {
		time.Sleep(TestBackoffTimeout)
		err = srvc1.host.connect(addrInfoB)
		require.NoError(t, err)
	}
	require.NoError(t, err)

	txnProtocolID := srvc1.host.protocolID + transactionsID
	stream, err := srvc1.host.p2pHost.NewStream(srvc1.ctx, srvc2.host.id(), txnProtocolID)
	require.NoError(t, err)

	// create info and handler
	info := &notificationsProtocol{
		protocolID:         txnProtocolID,
		getHandshake:       srvc1.getTransactionHandshake,
		handshakeValidator: validateTransactionHandshake,
		peersData:          newPeersData(),
	}
	handler := srvc1.createNotificationsMessageHandler(info, srvc1.handleTransactionMessage, txnBatchHandler)

	// set handshake data to received
	info.peersData.setInboundHandshakeData(srvc2.host.id(), &handshakeData{
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
		Extrinsics: []types.Extrinsic{{1, 1}, {2, 2}, {3, 3}},
	}
	err = handler(stream, msg)
	require.NoError(t, err)
	require.Len(t, txnBatch, 2)

	msg = &TransactionMessage{
		Extrinsics: []types.Extrinsic{{1, 1}, {2, 2}, {3, 3}, {4, 4}},
	}
	err = handler(stream, msg)
	require.NoError(t, err)
	require.Len(t, txnBatch, 3)

	msg = &TransactionMessage{
		Extrinsics: []types.Extrinsic{{1, 1}, {2, 2}, {3, 3}, {4, 4}, {5, 5}},
	}
	err = handler(stream, msg)
	require.NoError(t, err)
	require.Len(t, txnBatch, 4)

	msg = &TransactionMessage{
		Extrinsics: []types.Extrinsic{{1, 1}, {2, 2}, {3, 3}, {4, 4}, {5, 5}, {6, 6}},
	}
	err = handler(stream, msg)
	require.NoError(t, err)
	require.Len(t, txnBatch, 5)

	// reached batch size limit, below transaction will not be included in batch.
	msg = &TransactionMessage{
		Extrinsics: []types.Extrinsic{{1, 1}, {2, 2}, {3, 3}, {4, 4}, {5, 5}, {6, 6}, {7, 7}},
	}
	err = handler(stream, msg)
	require.NoError(t, err)
	require.Len(t, txnBatch, 5)

	msg = &TransactionMessage{
		Extrinsics: []types.Extrinsic{{1, 1}, {2, 2}, {3, 3}, {4, 4}, {5, 5}, {6, 6}, {7, 7}, {8, 8}},
	}
	// wait for transaction batch channel to process.
	time.Sleep(1300 * time.Millisecond)
	err = handler(stream, msg)
	require.NoError(t, err)
	require.Len(t, txnBatch, 1)
}

func TestBlockAnnounceHandshakeSize(t *testing.T) {
	t.Parallel()

	require.Equal(t, unsafe.Sizeof(BlockAnnounceHandshake{}), reflect.TypeOf(BlockAnnounceHandshake{}).Size())
}

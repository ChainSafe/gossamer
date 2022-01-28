// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package network

import (
	"context"
	"math/big"
	"strings"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/stretchr/testify/require"

	"github.com/ChainSafe/gossamer/dot/types"
	"github.com/ChainSafe/gossamer/internal/log"
	"github.com/ChainSafe/gossamer/lib/common"
)

var TestProtocolID = "/gossamer/test/0"

// maximum wait time for non-status message to be handled
var TestMessageTimeout = time.Second

// time between connection retries (BackoffBase default 5 seconds)
var TestBackoffTimeout = 5 * time.Second

// failedToDial returns true if "failed to dial" error, otherwise false
func failedToDial(err error) bool {
	return err != nil && strings.Contains(err.Error(), "failed to dial")
}

//go:generate mockgen -destination=mock_telemetry_test.go -package $GOPACKAGE github.com/ChainSafe/gossamer/dot/telemetry Client

func createServiceHelper(t *testing.T, num int) []*Service {
	t.Helper()

	var srvcs []*Service
	for i := 0; i < num; i++ {
		config := &Config{
			BasePath:    t.TempDir(),
			Port:        availablePort(t),
			NoBootstrap: true,
			NoMDNS:      true,
		}

		srvc := createTestService(t, config)
		srvc.noGossip = true
		handler := newTestStreamHandler(testBlockAnnounceMessageDecoder)
		srvc.host.registerStreamHandler(srvc.host.protocolID, handler.handleStream)

		srvcs = append(srvcs, srvc)
	}
	return srvcs
}

func newTestBlockResponseMessage(t *testing.T) *BlockResponseMessage {
	t.Helper()

	const blockRequestSize = 128
	msg := &BlockResponseMessage{
		BlockData: make([]*types.BlockData, blockRequestSize),
	}

	for i := 0; i < blockRequestSize; i++ {
		testHeader := &types.Header{
			Number: big.NewInt(int64(77 + i)),
			Digest: types.NewDigest(),
		}

		body := types.NewBody([]types.Extrinsic{[]byte{4, 4, 2}})

		msg.BlockData[i] = &types.BlockData{
			Hash:   testHeader.Hash(),
			Header: testHeader,
			Body:   body,
		}
	}

	return msg
}

//go:generate mockgen -destination=mock_block_state_test.go -package $GOPACKAGE . BlockState
//go:generate mockgen -destination=mock_syncer_test.go -package $GOPACKAGE . Syncer

// helper method to create and start a new network service
func createTestService(t *testing.T, cfg *Config) (srvc *Service) {
	t.Helper()
	ctrl := gomock.NewController(t)

	if cfg == nil {
		cfg = &Config{
			BasePath:     t.TempDir(),
			Port:         availablePort(t),
			NoBootstrap:  true,
			NoMDNS:       true,
			LogLvl:       log.Warn,
			SlotDuration: time.Second,
		}
	}

	if cfg.BlockState == nil {
		header := &types.Header{
			ParentHash:     common.Hash{},
			Number:         big.NewInt(1),
			StateRoot:      common.Hash{},
			ExtrinsicsRoot: common.Hash{},
			Digest:         types.NewDigest(),
		}

		blockstate := NewMockBlockState(ctrl)

		blockstate.EXPECT().BestBlockHeader().Return(header, nil).AnyTimes()
		blockstate.EXPECT().GetHighestFinalisedHeader().Return(header, nil).AnyTimes()
		blockstate.EXPECT().GenesisHash().Return(common.NewHash([]byte{})).AnyTimes()
		blockstate.EXPECT().BestBlockNumber().Return(big.NewInt(1), nil).AnyTimes()

		blockstate.EXPECT().HasBlockBody(
			gomock.AssignableToTypeOf(common.Hash([32]byte{}))).Return(false, nil).AnyTimes()
		blockstate.EXPECT().GetHashByNumber(gomock.Any()).Return(common.Hash{}, nil).AnyTimes()

		cfg.BlockState = blockstate
	}

	if cfg.TransactionHandler == nil {
		th := NewMockTransactionHandler(ctrl)
		th.EXPECT().
			HandleTransactionMessage(
				gomock.AssignableToTypeOf(peer.ID("")),
				gomock.Any()).
			Return(true, nil).AnyTimes()

		th.EXPECT().TransactionsCount().Return(0).AnyTimes()
		cfg.TransactionHandler = th
	}

	cfg.SlotDuration = time.Second
	cfg.ProtocolID = TestProtocolID // default "/gossamer/gssmr/0"

	if cfg.LogLvl == 0 {
		cfg.LogLvl = 4
	}

	if cfg.Syncer == nil {
		syncer := NewMockSyncer(ctrl)
		syncer.EXPECT().
			HandleBlockAnnounceHandshake(
				gomock.AssignableToTypeOf(peer.ID("")), gomock.Any()).
			Return(nil).AnyTimes()

		syncer.EXPECT().
			HandleBlockAnnounce(
				gomock.AssignableToTypeOf(peer.ID("")), gomock.Any()).
			Return(nil).AnyTimes()

		syncer.EXPECT().
			CreateBlockResponse(gomock.Any()).
			Return(newTestBlockResponseMessage(t), nil).AnyTimes()

		syncer.EXPECT().IsSynced().Return(false).AnyTimes()
		cfg.Syncer = syncer
	}

	if cfg.Telemetry == nil {
		telemetryMock := NewMockClient(ctrl)
		telemetryMock.EXPECT().SendMessage(gomock.Any()).AnyTimes()
		cfg.Telemetry = telemetryMock
	}

	cfg.noPreAllocate = true

	srvc, err := NewService(cfg)
	require.NoError(t, err)

	srvc.noDiscover = true

	err = srvc.Start()
	require.NoError(t, err)

	t.Cleanup(func() {
		err := srvc.Stop()
		require.NoError(t, err)
	})
	return srvc
}

// test network service starts
func TestStartService(t *testing.T) {
	t.Parallel()

	node := createTestService(t, nil)
	require.NoError(t, node.Stop())
}

// test broacast messages from core service
func TestBroadcastMessages(t *testing.T) {
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
		NoBootstrap: true,
		NoMDNS:      true,
	}

	nodeB := createTestService(t, configB)
	nodeB.noGossip = true
	handler := newTestStreamHandler(testBlockAnnounceHandshakeDecoder)
	nodeB.host.registerStreamHandler(nodeB.host.protocolID+blockAnnounceID, handler.handleStream)

	addrInfoB := nodeB.host.addrInfo()
	err := nodeA.host.connect(addrInfoB)
	// retry connect if "failed to dial" error
	if failedToDial(err) {
		time.Sleep(TestBackoffTimeout)
		err = nodeA.host.connect(addrInfoB)
	}
	require.NoError(t, err)

	anounceMessage := &BlockAnnounceMessage{
		Number: big.NewInt(128 * 7),
		Digest: types.NewDigest(),
	}

	// simulate message sent from core service
	nodeA.GossipMessage(anounceMessage)
	time.Sleep(time.Second * 2)

	messages, _ := handler.messagesFrom(nodeA.host.id())
	require.NotNil(t, messages)
}

func Test_Broadcast_Duplicate_Messages_WithDisabled_MessageCache(t *testing.T) {
	t.Parallel()

	configA := &Config{
		BasePath:        t.TempDir(),
		Port:            availablePort(t),
		NoBootstrap:     true,
		NoMDNS:          true,
		MessageCacheTTL: 2 * time.Second,
	}

	nodeA := createTestService(t, configA)
	nodeA.noGossip = true

	configB := &Config{
		BasePath:        t.TempDir(),
		Port:            availablePort(t),
		NoBootstrap:     true,
		NoMDNS:          true,
		MessageCacheTTL: 2 * time.Second,
	}

	nodeB := createTestService(t, configB)
	nodeB.noGossip = true

	handler := newTestStreamHandler(testBlockAnnounceHandshakeDecoder)
	nodeB.host.registerStreamHandler(nodeB.host.protocolID+blockAnnounceID, handler.handleStream)

	addrInfoB := nodeB.host.addrInfo()
	err := nodeA.host.connect(addrInfoB)
	// retry connect if "failed to dial" error
	if failedToDial(err) {
		time.Sleep(TestBackoffTimeout)
		err = nodeA.host.connect(addrInfoB)
	}
	require.NoError(t, err)

	stream, err := nodeA.host.h.NewStream(context.Background(), nodeB.host.id(), nodeB.host.protocolID+blockAnnounceID)
	require.NoError(t, err)
	require.NotNil(t, stream)

	protocol := nodeA.notificationsProtocols[BlockAnnounceMsgType]
	protocol.outboundHandshakeData.Store(nodeB.host.id(), &handshakeData{
		received:  true,
		validated: true,
		stream:    stream,
	})

	announceMessage := &BlockAnnounceMessage{
		Number: big.NewInt(128 * 7),
		Digest: types.NewDigest(),
	}

	// disable message cache before sending the messages
	nodeA.host.messageCache = nil

	// All 5 message will be sent since cache is disabled.
	for i := 0; i < 5; i++ {
		nodeA.GossipMessage(announceMessage)
		time.Sleep(time.Millisecond * 10)
	}

	messages, _ := handler.messagesFrom(nodeA.host.id())
	require.Len(t, messages, 6)
}

func Test_Broadcast_Duplicate_Messages_With_MessageCache(t *testing.T) {
	t.Parallel()

	configA := &Config{
		BasePath:        t.TempDir(),
		Port:            availablePort(t),
		NoBootstrap:     true,
		NoMDNS:          true,
		MessageCacheTTL: 2 * time.Second,
	}

	nodeA := createTestService(t, configA)
	nodeA.noGossip = true

	configB := &Config{
		BasePath:        t.TempDir(),
		Port:            availablePort(t),
		NoBootstrap:     true,
		NoMDNS:          true,
		MessageCacheTTL: 2 * time.Second,
	}

	nodeB := createTestService(t, configB)
	nodeB.noGossip = true

	handler := newTestStreamHandler(testBlockAnnounceHandshakeDecoder)
	nodeB.host.registerStreamHandler(nodeB.host.protocolID+blockAnnounceID, handler.handleStream)

	addrInfoB := nodeB.host.addrInfo()
	err := nodeA.host.connect(addrInfoB)
	// retry connect if "failed to dial" error
	if failedToDial(err) {
		time.Sleep(TestBackoffTimeout)
		err = nodeA.host.connect(addrInfoB)
	}
	require.NoError(t, err)

	stream, err := nodeA.host.h.NewStream(context.Background(), nodeB.host.id(), nodeB.host.protocolID+blockAnnounceID)
	require.NoError(t, err)
	require.NotNil(t, stream)

	protocol := nodeA.notificationsProtocols[BlockAnnounceMsgType]
	protocol.outboundHandshakeData.Store(nodeB.host.id(), &handshakeData{
		received:  true,
		validated: true,
		stream:    stream,
	})

	announceMessage := &BlockAnnounceMessage{
		Number: big.NewInt(128 * 7),
		Digest: types.NewDigest(),
	}

	// Only one message will be sent.
	for i := 0; i < 5; i++ {
		nodeA.GossipMessage(announceMessage)
		time.Sleep(time.Millisecond * 10)
	}

	time.Sleep(time.Millisecond * 200)
	messages, _ := handler.messagesFrom(nodeA.host.id())
	require.Len(t, messages, 2)
}

func TestService_NodeRoles(t *testing.T) {
	t.Parallel()

	cfg := &Config{
		BasePath: t.TempDir(),
		Roles:    1,
		Port:     availablePort(t),
	}
	svc := createTestService(t, cfg)

	role := svc.NodeRoles()
	require.Equal(t, cfg.Roles, role)
}

func TestService_Health(t *testing.T) {
	t.Parallel()
	ctrl := gomock.NewController(t)

	config := &Config{
		BasePath:    t.TempDir(),
		Port:        availablePort(t),
		NoBootstrap: true,
		NoMDNS:      true,
	}

	syncer := NewMockSyncer(ctrl)

	s := createTestService(t, config)
	s.syncer = syncer

	syncer.EXPECT().IsSynced().Return(false)
	h := s.Health()
	require.Equal(t, true, h.IsSyncing)

	syncer.EXPECT().IsSynced().Return(true)
	h = s.Health()
	require.Equal(t, false, h.IsSyncing)
}

func TestPersistPeerStore(t *testing.T) {
	t.Parallel()

	nodes := createServiceHelper(t, 2)
	nodeA := nodes[0]
	nodeB := nodes[1]

	addrInfoB := nodeB.host.addrInfo()
	err := nodeA.host.connect(addrInfoB)
	if failedToDial(err) {
		time.Sleep(TestBackoffTimeout)
		err = nodeA.host.connect(addrInfoB)
	}
	require.NoError(t, err)

	require.NotEmpty(t, nodeA.host.h.Peerstore().PeerInfo(nodeB.host.id()).Addrs)

	// Stop a node and reinitialise a new node with same base path.
	err = nodeA.Stop()
	require.NoError(t, err)

	// Since nodeAA uses the persistent peerstore of nodeA, it should be have nodeB in it's peerstore.
	nodeAA := createTestService(t, nodeA.cfg)
	require.NotEmpty(t, nodeAA.host.h.Peerstore().PeerInfo(nodeB.host.id()).Addrs)
}

func TestHandleConn(t *testing.T) {
	t.Parallel()

	configA := &Config{
		BasePath:    t.TempDir(),
		Port:        availablePort(t),
		NoBootstrap: true,
		NoMDNS:      true,
	}

	nodeA := createTestService(t, configA)

	configB := &Config{
		BasePath:    t.TempDir(),
		Port:        availablePort(t),
		NoBootstrap: true,
		NoMDNS:      true,
	}

	nodeB := createTestService(t, configB)

	addrInfoB := nodeB.host.addrInfo()
	err := nodeA.host.connect(addrInfoB)
	if failedToDial(err) {
		time.Sleep(TestBackoffTimeout)
		err = nodeA.host.connect(addrInfoB)
	}
	require.NoError(t, err)
}

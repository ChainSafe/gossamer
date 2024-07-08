// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package network

import (
	"errors"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/ChainSafe/gossamer/dot/types"
	"github.com/ChainSafe/gossamer/internal/log"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/lib/common/variadic"
	libp2pnetwork "github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

const (
	// TestProtocolID default protocol to testing
	TestProtocolID = "/gossamer/test/0"

	// TestMessageTimeout maximum wait time for non-status message to be handled
	TestMessageTimeout = time.Second

	// TestBackoffTimeout time between connection retries (BackoffBase default 5 seconds)
	TestBackoffTimeout = 5 * time.Second
)

type testStreamHandler struct {
	messages map[peer.ID][]Message
	decoder  messageDecoder
	exit     bool
}

func newTestStreamHandler(decoder messageDecoder) *testStreamHandler {
	return &testStreamHandler{
		messages: make(map[peer.ID][]Message),
		decoder:  decoder,
	}
}

func (s *testStreamHandler) handleStream(stream libp2pnetwork.Stream) {
	conn := stream.Conn()
	if conn == nil {
		logger.Error("Failed to get connection from stream")
		return
	}

	peer := conn.RemotePeer()
	s.readStream(stream, peer, s.decoder, s.handleMessage)
}

func (s *testStreamHandler) handleMessage(stream libp2pnetwork.Stream, msg Message) error {
	msgs := s.messages[stream.Conn().RemotePeer()]
	s.messages[stream.Conn().RemotePeer()] = append(msgs, msg)

	announceHandshake := &BlockAnnounceHandshake{
		BestBlockNumber: 0,
	}
	return s.writeToStream(stream, announceHandshake)
}

func (s *testStreamHandler) writeToStream(stream libp2pnetwork.Stream, msg Message) error {
	encMsg, err := msg.Encode()
	if err != nil {
		return err
	}

	msgLen := uint64(len(encMsg))
	lenBytes := Uint64ToLEB128(msgLen)
	encMsg = append(lenBytes, encMsg...)

	_, err = stream.Write(encMsg)
	return err
}

func (s *testStreamHandler) readStream(stream libp2pnetwork.Stream,
	peer peer.ID, decoder messageDecoder, handler messageHandler) {
	msgBytes := make([]byte, MaxBlockResponseSize)

	defer func() {
		s.exit = true
	}()

	for {
		tot, err := readStream(stream, &msgBytes, MaxBlockResponseSize)
		if errors.Is(err, io.EOF) {
			return
		} else if err != nil {
			logger.Debugf("failed to read from stream using protocol %s: %s", stream.Protocol(), err)
			err := stream.Close()
			if err != nil {
				logger.Warnf("failed to close stream: %s", err)
			}
			return
		}

		// decode message based on message type
		msg, err := decoder(msgBytes[:tot], peer, isInbound(stream))
		if err != nil {
			logger.Errorf("failed to decode message from peer %s: %s", peer, err)
			continue
		}

		// handle message based on peer status and message type
		err = handler(stream, msg)
		if err != nil {
			logger.Errorf("failed to handle message %s from stream: %s", msg, err)
			err := stream.Close()
			if err != nil {
				logger.Warnf("failed to close stream: %s", err)
			}
			return
		}
	}
}

var starting, _ = variadic.NewUint32OrHash(uint32(1))

var one = uint32(1)

func newTestBlockRequestMessage(t *testing.T) *BlockRequestMessage {
	t.Helper()

	return &BlockRequestMessage{
		RequestedData: RequestedDataHeader + RequestedDataBody + RequestedDataJustification,
		StartingBlock: *starting,
		Direction:     1,
		Max:           &one,
	}
}

func testBlockRequestMessageDecoder(in []byte, _ peer.ID, _ bool) (Message, error) {
	msg := new(BlockRequestMessage)
	err := msg.Decode(in)
	return msg, err
}

func testBlockAnnounceMessageDecoder(in []byte, _ peer.ID, _ bool) (Message, error) {
	msg := BlockAnnounceMessage{
		Number: 0,
		Digest: types.NewDigest(),
	}
	err := msg.Decode(in)
	return &msg, err
}

func testBlockAnnounceHandshakeDecoder(in []byte, _ peer.ID, _ bool) (Message, error) {
	msg := new(BlockAnnounceHandshake)
	err := msg.Decode(in)
	return msg, err
}

// addrInfo returns the libp2p peer.AddrInfo of the host
func addrInfo(h *host) peer.AddrInfo {
	return peer.AddrInfo{
		ID:    h.p2pHost.ID(),
		Addrs: h.p2pHost.Addrs(),
	}
}

// returns a slice of peers that are unprotected and may be pruned.
func unprotectedPeers(cm *ConnManager, peers []peer.ID) []peer.ID {
	unprot := []peer.ID{}
	for _, id := range peers {
		if cm.IsProtected(id, "") {
			continue
		}

		_, isPersistent := cm.persistentPeers.Load(id)
		if !isPersistent {
			unprot = append(unprot, id)
		}
	}

	return unprot
}

// failedToDial returns true if "failed to dial" error, otherwise false
func failedToDial(err error) bool {
	return err != nil && strings.Contains(err.Error(), "failed to dial")
}

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
			Number:         1,
			StateRoot:      common.Hash{},
			ExtrinsicsRoot: common.Hash{},
			Digest:         types.NewDigest(),
		}

		blockstate := NewMockBlockState(ctrl)

		blockstate.EXPECT().BestBlockHeader().Return(header, nil).AnyTimes()
		blockstate.EXPECT().GetHighestFinalisedHeader().Return(header, nil).AnyTimes()
		blockstate.EXPECT().GenesisHash().Return(common.NewHash([]byte{})).AnyTimes()

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
			CreateBlockResponse(gomock.Any(), gomock.Any()).
			Return(newTestBlockResponseMessage(t), nil).AnyTimes()

		syncer.EXPECT().IsSynced().Return(false).AnyTimes()
		cfg.Syncer = syncer
	}

	if cfg.Telemetry == nil {
		telemetryMock := NewMockTelemetry(ctrl)
		telemetryMock.EXPECT().SendMessage(gomock.Any()).AnyTimes()
		cfg.Telemetry = telemetryMock
	}

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

func newTestBlockResponseMessage(t *testing.T) *BlockResponseMessage {
	t.Helper()

	const blockRequestSize = 128
	msg := &BlockResponseMessage{
		BlockData: make([]*types.BlockData, blockRequestSize),
	}

	for i := uint(0); i < blockRequestSize; i++ {
		testHeader := &types.Header{
			Number: 77 + i,
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

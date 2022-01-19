// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package network

import (
	"errors"
	"io"
	"math/big"
	"testing"
	"time"

	"github.com/ChainSafe/gossamer/dot/types"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/lib/common/variadic"

	libp2pnetwork "github.com/libp2p/go-libp2p-core/network"
	"github.com/libp2p/go-libp2p-core/peer"
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
	lenBytes := uint64ToLEB128(msgLen)
	encMsg = append(lenBytes, encMsg...)

	_, err = stream.Write(encMsg)
	return err
}

func (s *testStreamHandler) readStream(stream libp2pnetwork.Stream,
	peer peer.ID, decoder messageDecoder, handler messageHandler) {
	msgBytes := make([]byte, maxBlockResponseSize)

	defer func() {
		s.exit = true
	}()

	for {
		tot, err := readStream(stream, msgBytes)
		if errors.Is(err, io.EOF) {
			return
		} else if err != nil {
			logger.Debugf("failed to read from stream using protocol %s: %s", stream.Protocol(), err)
			_ = stream.Close()
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
			_ = stream.Close()
			return
		}
	}
}

var starting, _ = variadic.NewUint64OrHash(uint64(1))

var one = uint32(1)

func newTestBlockRequestMessage(t *testing.T) *BlockRequestMessage {
	t.Helper()

	return &BlockRequestMessage{
		RequestedData: RequestedDataHeader + RequestedDataBody + RequestedDataJustification,
		StartingBlock: *starting,
		EndBlockHash:  &common.Hash{},
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
		Number: big.NewInt(0),
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

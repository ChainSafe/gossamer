// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package network

import (
	"errors"
	"io"
	"math/big"

	"github.com/stretchr/testify/mock"

	"github.com/ChainSafe/gossamer/dot/types"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/lib/common/variadic"

	libp2pnetwork "github.com/libp2p/go-libp2p-core/network"
	"github.com/libp2p/go-libp2p-core/peer"
)

const blockRequestSize uint32 = 128

// NewMockBlockState create and return a network BlockState interface mock
func NewMockBlockState(n *big.Int) *MockBlockState {
	parentHash, _ := common.HexToHash("0x4545454545454545454545454545454545454545454545454545454545454545")
	stateRoot, _ := common.HexToHash("0xb3266de137d20a5d0ff3a6401eb57127525fd9b2693701f0bf5a8a853fa3ebe0")
	extrinsicsRoot, _ := common.HexToHash("0x03170a2e7597b7b7e3d84c05391d139a62b157e78786d8c082f29dcf4c111314")

	if n == nil {
		n = big.NewInt(1)
	}
	header := &types.Header{
		ParentHash:     parentHash,
		Number:         n,
		StateRoot:      stateRoot,
		ExtrinsicsRoot: extrinsicsRoot,
		Digest:         types.NewDigest(),
	}

	m := new(MockBlockState)
	m.On("BestBlockHeader").Return(header, nil)
	m.On("GetHighestFinalisedHeader").Return(header, nil)
	m.On("GenesisHash").Return(common.NewHash([]byte{}))
	m.On("BestBlockNumber").Return(big.NewInt(1), nil)
	m.On("HasBlockBody", mock.AnythingOfType("common.Hash")).Return(false, nil)
	m.On("GetHashByNumber", mock.AnythingOfType("*big.Int")).Return(common.Hash{}, nil)

	return m
}

// NewMockSyncer create and return a network Syncer interface mock
func NewMockSyncer() *MockSyncer {
	mocksyncer := new(MockSyncer)
	mocksyncer.
		On("HandleBlockAnnounceHandshake",
			mock.AnythingOfType("peer.ID"),
			mock.AnythingOfType("*network.BlockAnnounceHandshake")).
		Return(nil, nil)
	mocksyncer.
		On("HandleBlockAnnounce",
			mock.AnythingOfType("peer.ID"),
			mock.AnythingOfType("*network.BlockAnnounceMessage")).
		Return(nil, nil)
	mocksyncer.
		On("CreateBlockResponse",
			mock.AnythingOfType("*network.BlockRequestMessage")).
		Return(testBlockResponseMessage(), nil)
	mocksyncer.
		On("IsSynced").Return(false)
	return mocksyncer
}

// NewMockTransactionHandler create and return a network TransactionHandler interface
func NewMockTransactionHandler() *MockTransactionHandler {
	mocktxhandler := new(MockTransactionHandler)
	mocktxhandler.On("HandleTransactionMessage",
		mock.AnythingOfType("peer.ID"),
		mock.AnythingOfType("*network.TransactionMessage")).
		Return(true, nil)
	mocktxhandler.On("TransactionsCount").Return(0)
	return mocktxhandler
}

func testBlockResponseMessage() *BlockResponseMessage {
	msg := &BlockResponseMessage{
		BlockData: []*types.BlockData{},
	}

	for i := 0; i < int(blockRequestSize); i++ {
		testHeader := &types.Header{
			Number: big.NewInt(int64(77 + i)),
			Digest: types.NewDigest(),
		}

		body := types.NewBody([]types.Extrinsic{[]byte{4, 4, 2}})

		msg.BlockData = append(msg.BlockData, &types.BlockData{
			Hash:          testHeader.Hash(),
			Header:        testHeader,
			Body:          body,
			MessageQueue:  nil,
			Receipt:       nil,
			Justification: nil,
		})
	}

	return msg
}

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
	return s.writeToStream(stream, testBlockAnnounceHandshake)
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

var testBlockRequestMessage = &BlockRequestMessage{
	RequestedData: RequestedDataHeader + RequestedDataBody + RequestedDataJustification,
	StartingBlock: *starting,
	EndBlockHash:  &common.Hash{},
	Direction:     1,
	Max:           &one,
}

func testBlockRequestMessageDecoder(in []byte, _ peer.ID, _ bool) (Message, error) {
	msg := new(BlockRequestMessage)
	err := msg.Decode(in)
	return msg, err
}

var testBlockAnnounceMessage = &BlockAnnounceMessage{
	Number: big.NewInt(128 * 7),
	Digest: types.NewDigest(),
}

var testBlockAnnounceHandshake = &BlockAnnounceHandshake{
	BestBlockNumber: 0,
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

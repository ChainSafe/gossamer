package network

import (
	"io"
	"math/big"

	"github.com/ChainSafe/gossamer/dot/types"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/lib/common/optional"
	"github.com/ChainSafe/gossamer/lib/common/variadic"
	"github.com/stretchr/testify/mock"

	libp2pnetwork "github.com/libp2p/go-libp2p-core/network"
	"github.com/libp2p/go-libp2p-core/peer"
)

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
		Digest:         types.Digest{},
	}

	m := new(MockBlockState)
	m.On("BestBlockHeader").Return(header, nil)

	m.On("GenesisHash").Return(common.NewHash([]byte{}))
	m.On("BestBlockNumber").Return(big.NewInt(1), nil)
	m.On("HasBlockBody", mock.AnythingOfType("common.Hash")).Return(false, nil)
	m.On("GetFinalizedHeader", mock.AnythingOfType("uint64"), mock.AnythingOfType("uint64")).Return(header, nil)
	m.On("GetHashByNumber", mock.AnythingOfType("*big.Int")).Return(common.Hash{}, nil)

	return m
}

// NewMockSyncer create and return a network Syncer interface mock
func NewMockSyncer() *MockSyncer {
	mocksyncer := new(MockSyncer)
	mocksyncer.On("HandleBlockAnnounce", mock.AnythingOfType("*network.BlockAnnounceMessage")).Return(nil, nil)
	mocksyncer.On("CreateBlockResponse", mock.AnythingOfType("*network.BlockRequestMessage")).Return(testBlockResponseMessage(), nil)
	mocksyncer.On("ProcessJustification", mock.AnythingOfType("[]*types.BlockData")).Return(0, nil)
	mocksyncer.On("ProcessBlockData", mock.AnythingOfType("[]*types.BlockData")).Return(0, nil)
	mocksyncer.On("SetSyncing", mock.AnythingOfType("bool"))
	mocksyncer.On("IsSynced").Return(false)
	return mocksyncer
}

// NewMockTransactionHandler create and return a network TransactionHandler interface
func NewMockTransactionHandler() *MockTransactionHandler {
	mocktxhandler := new(MockTransactionHandler)
	mocktxhandler.On("HandleTransactionMessage", mock.AnythingOfType("*network.TransactionMessage")).Return(nil)
	mocktxhandler.On("TransactionsCount").Return(0)
	return mocktxhandler
}

func testBlockResponseMessage() *BlockResponseMessage {
	msg := &BlockResponseMessage{
		BlockData: []*types.BlockData{},
	}

	for i := 0; i < int(blockRequestSize); i++ {
		testHeader := types.Header{
			Number: big.NewInt(int64(77 + i)),
			Digest: types.Digest{},
		}

		msg.BlockData = append(msg.BlockData, &types.BlockData{
			Hash:          testHeader.Hash(),
			Header:        testHeader.AsOptional(),
			Body:          optional.NewBody(true, []byte{4, 4, 2}),
			MessageQueue:  optional.NewBytes(false, nil),
			Receipt:       optional.NewBytes(false, nil),
			Justification: optional.NewBytes(false, nil),
		})
	}

	return msg
}

type testStreamHandler struct {
	messages map[peer.ID][]Message
	decoder  messageDecoder
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

func (s *testStreamHandler) readStream(stream libp2pnetwork.Stream, peer peer.ID, decoder messageDecoder, handler messageHandler) {
	var (
		maxMessageSize uint64 = maxBlockResponseSize // TODO: determine actual max message size
		msgBytes              = make([]byte, maxMessageSize)
	)

	for {
		tot, err := readStream(stream, msgBytes)
		if err == io.EOF {
			continue
		} else if err != nil {
			logger.Debug("failed to read from stream", "protocol", stream.Protocol(), "error", err)
			_ = stream.Close()
			return
		}

		// decode message based on message type
		msg, err := decoder(msgBytes[:tot], peer, isInbound(stream))
		if err != nil {
			logger.Error("Failed to decode message from peer", "peer", peer, "err", err)
			continue
		}

		// handle message based on peer status and message type
		err = handler(stream, msg)
		if err != nil {
			logger.Error("Failed to handle message from stream", "message", msg, "error", err)
			_ = stream.Close()
			return
		}
	}
}

var start, _ = variadic.NewUint64OrHash(uint64(1))

var testBlockRequestMessage = &BlockRequestMessage{
	RequestedData: RequestedDataHeader + RequestedDataBody + RequestedDataJustification,
	StartingBlock: start,
	EndBlockHash:  optional.NewHash(true, common.Hash{}),
	Direction:     1,
	Max:           optional.NewUint32(true, 1),
}

func testBlockRequestMessageDecoder(in []byte, _ peer.ID, _ bool) (Message, error) {
	msg := new(BlockRequestMessage)
	err := msg.Decode(in)
	return msg, err
}

var testBlockAnnounceMessage = &BlockAnnounceMessage{
	Number: big.NewInt(128 * 7),
}

var testBlockAnnounceHandshake = &BlockAnnounceHandshake{
	BestBlockNumber: 0,
}

func testBlockAnnounceMessageDecoder(in []byte, _ peer.ID, _ bool) (Message, error) {
	msg := new(BlockAnnounceMessage)
	err := msg.Decode(in)
	return msg, err
}

func testBlockAnnounceHandshakeDecoder(in []byte, _ peer.ID, _ bool) (Message, error) {
	msg := new(BlockAnnounceHandshake)
	err := msg.Decode(in)
	return msg, err
}

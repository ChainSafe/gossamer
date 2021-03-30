package network

import (
	"io"
	"math/big"

	"github.com/ChainSafe/gossamer/dot/types"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/lib/common/optional"
	"github.com/ChainSafe/gossamer/lib/common/variadic"

	libp2pnetwork "github.com/libp2p/go-libp2p-core/network"
	"github.com/libp2p/go-libp2p-core/peer"
)

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

type mockSyncer struct {
	highestSeen *big.Int
	synced      bool
}

func newMockSyncer() *mockSyncer {
	return &mockSyncer{
		highestSeen: big.NewInt(0),
		synced:      false,
	}
}

func (s *mockSyncer) CreateBlockResponse(msg *BlockRequestMessage) (*BlockResponseMessage, error) {
	return testBlockResponseMessage(), nil
}

func (s *mockSyncer) HandleBlockAnnounce(msg *BlockAnnounceMessage) error {
	return nil
}

func (s *mockSyncer) ProcessBlockData(data []*types.BlockData) (int, error) {
	return 0, nil
}

func (s *mockSyncer) IsSynced() bool {
	return s.synced
}

func (s *mockSyncer) SetSyncing(syncing bool) {
	s.synced = !syncing
}

type testStreamHandler struct {
	messages map[peer.ID]Message
	decoder  messageDecoder
}

func newTestStreamHandler(decoder messageDecoder) *testStreamHandler {
	return &testStreamHandler{
		messages: make(map[peer.ID]Message),
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
	s.messages[stream.Conn().RemotePeer()] = msg
	return nil
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
		msg, err := decoder(msgBytes[:tot], peer)
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

func testBlockRequestMessageDecoder(in []byte, _ peer.ID) (Message, error) {
	msg := new(BlockRequestMessage)
	err := msg.Decode(in)
	return msg, err
}

var testBlockAnnounceMessage = &BlockAnnounceMessage{
	Number: big.NewInt(128 * 7),
}

func testBlockAnnounceMessageDecoder(in []byte, _ peer.ID) (Message, error) {
	msg := new(BlockAnnounceMessage)
	err := msg.Decode(in)
	return msg, err
}

func testBlockAnnounceHandshakeDecoder(in []byte, _ peer.ID) (Message, error) {
	msg := new(BlockAnnounceHandshake)
	err := msg.Decode(in)
	return msg, err
}
